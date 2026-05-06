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

	"detector-service/internal/api"
	"detector-service/internal/channeltest"
	"detector-service/internal/config"
	"detector-service/internal/fingerprint"
	"detector-service/internal/intelligence"
	"detector-service/internal/intelligence/datasets/sweatlas"
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

	channelRunner := channeltest.NewRunner()
	channelStore := channeltest.NewStore(channelRunner, logger, false, nil)
	runner := intelligence.NewRunner(nil)

	mux := http.NewServeMux()
	api := api.New(cfg, logger, channelStore, registry, runner)
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
