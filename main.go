package main

import (
	"context"
	"flag"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"detector-service/internal/alert"
	"detector-service/internal/api"
	"detector-service/internal/channeltest"
	"detector-service/internal/config"
	"detector-service/internal/fingerprint"
	"detector-service/internal/intelligence"
	"detector-service/internal/intelligence/datasets/sweatlas"
	"detector-service/internal/monitor"
	"detector-service/internal/persist"
	"detector-service/web"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logLevel := slog.LevelInfo
	switch cfg.Log.Level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	if cfg.Channel.SigSecret != "" {
		fingerprint.SetSigSecret(cfg.Channel.SigSecret)
	}

	// ── Intelligence Dataset Registry ──
	registry := intelligence.NewRegistry()

	// Register built-in SWE-Atlas-QnA (embedded)
	sweAtlas, err := sweatlas.Load()
	if err != nil {
		logger.Error("failed to load embedded intelligence dataset", "error", err)
		os.Exit(1)
	}
	registry.Register(sweAtlas)

	// ── Persistence ──
	db, err := persist.Open(cfg.Storage.Path)
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	persistLog := func(op string, err error) {
		logger.Warn("persist error", "op", op, "error", err)
	}

	keywordStore := channeltest.NewKeywordStore(&channeltest.KeywordPersist{
		DB:      db.Conn(),
		LogErr:  persistLog,
		Save:    persist.SaveKeyword,
		Delete:  persist.DeleteKeyword,
		LoadAll: persist.LoadAllKeywords,
	})

	channelRunner := channeltest.NewRunner()
	channelRunner.KeywordStore = keywordStore
	channelStore := channeltest.NewStore(channelRunner, logger, false, nil, &channeltest.StorePersist{
		DB:         db.Conn(),
		LogErr:     persistLog,
		Save:       persist.SaveChannelHistory,
		Delete:     persist.DeleteChannelHistory,
		UpdateName: persist.UpdateChannelHistoryName,
		Load:       persist.LoadAllChannelHistory,
	}, cfg.Storage.MaxHistory)
	runner := intelligence.NewRunner(nil)

	// ── Monitor + Alert ──
	monitorStore := monitor.NewStore(&monitor.MonitorPersist{
		DB:                  db.Conn(),
		LogErr:              persistLog,
		SaveTarget:          persist.SaveTarget,
		DeleteTarget:        persist.DeleteTarget,
		LoadAllTargets:      persist.LoadAllTargets,
		SaveHealthState:     persist.SaveHealthState,
		DeleteHealthStates:  persist.DeleteHealthStates,
		DeleteHealthState:   persist.DeleteHealthState,
		LoadAllHealthStates: persist.LoadAllHealthStates,
		SaveRun:             persist.SaveRun,
		LoadAllRuns:         persist.LoadAllRuns,
		TrimRuns:            persist.TrimRuns,
	})
	baselineStore := monitor.NewBaselineStore(&monitor.BaselinePersist{
		DB:      db.Conn(),
		LogErr:  persistLog,
		Save:    persist.SaveBaseline,
		Delete:  persist.DeleteBaseline,
		LoadAll: persist.LoadAllBaselines,
	})
	alertStore := alert.NewStore(&alert.StorePersist{
		DB:     db.Conn(),
		LogErr: persistLog,
		Save:   persist.SaveAlertEvent,
		Update: persist.UpdateAlertEvent,
		Load:   persist.LoadAllAlertEvents,
		Trim:   persist.TrimAlertEvents,
	})
	alertRules := alert.DefaultRules()
	alertEvaluator := alert.NewEvaluator(alertRules, alertStore)
	var webhookDests []alert.WebhookDest
	for _, wh := range cfg.Alert.Webhooks {
		if wh.URL != "" {
			webhookDests = append(webhookDests, alert.WebhookDest{
				Name:    wh.Name,
				URL:     wh.URL,
				Headers: wh.Headers,
			})
		}
	}
	alertNotifier := alert.NewNotifier(webhookDests, logger)

	alertEnabled := cfg.Alert.Enabled
	monitorRunner := monitor.NewRunner(channelRunner, runner, registry, baselineStore, monitorStore, logger, func(run *monitor.MonitorRun) {
		if !alertEnabled {
			return
		}
		state := monitorStore.GetState(run.TargetID, run.Model, run.CheckType)
		events := alertEvaluator.Evaluate(run, state)
		alertNotifier.NotifyAll(events)
	})

	scheduler := monitor.NewScheduler(monitorStore, monitorRunner, logger)
	scheduler.Start()

	intelligenceHistory := intelligence.NewHistoryStore(&intelligence.HistoryPersist{
		DB:     db.Conn(),
		LogErr: persistLog,
		Save:   persist.SaveIntelligenceHistory,
		Delete: persist.DeleteIntelligenceHistory,
		Load:   persist.LoadAllIntelligenceHistory,
	}, cfg.Storage.MaxHistory)

	mux := http.NewServeMux()
	api := api.New(cfg, logger, channelStore, registry, runner, intelligenceHistory)
	api.SetMonitor(monitorStore, monitorRunner, baselineStore, channelRunner)
	api.SetKeywords(keywordStore)
	api.SetAlerts(alertStore, alertEvaluator, alertNotifier, alertRules)
	api.RegisterRoutes(mux)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","mode":"channel_intelligence"}`))
	})

	staticFS, err := fs.Sub(web.StaticFS, "static")
	if err != nil {
		logger.Error("failed to load static files", "error", err)
		os.Exit(1)
	}
	fileServer := http.FileServer(http.FS(staticFS))
	mux.Handle("/", fileServer)

	names := registry.List()
	logger.Info("starting channel/intelligence service",
		"listen", cfg.Server.Listen,
		"default_target", cfg.Upstream.BaseURL,
		"intelligence_datasets", names,
		"intelligence_dataset_count", len(names),
		"webui", "http://localhost"+cfg.Server.Listen,
	)

	srv := &http.Server{Addr: cfg.Server.Listen, Handler: mux}
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		logger.Info("shutdown signal received", "signal", sig.String())
		scheduler.Stop()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("shutdown error", "error", err)
		}
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
	logger.Info("server stopped")
}
