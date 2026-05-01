package main

import (
	"context"
	"flag"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"bedrock-gateway/internal/admin"
	"bedrock-gateway/internal/config"
	"bedrock-gateway/internal/fingerprint"
	"bedrock-gateway/internal/handler"
	"bedrock-gateway/internal/keymap"
	"bedrock-gateway/internal/middleware"
	"bedrock-gateway/internal/probe"
	"bedrock-gateway/internal/proxy"
	"bedrock-gateway/web"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Logger
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

	// Signature secret
	if cfg.Disguise.SigSecret != "" {
		fingerprint.SetSigSecret(cfg.Disguise.SigSecret)
	}

	// Request logger for admin dashboard
	reqLogger := admin.NewRequestLogger(10000)

	// Key map (optional)
	var km *keymap.KeyMap
	if cfg.KeyMap.Enabled {
		km = keymap.New(cfg.KeyMap.KeysFile)
		logger.Info("key map loaded", "file", cfg.KeyMap.KeysFile, "count", km.Count())
	}

	// Upstream proxy
	upstreamProxy := proxy.NewUpstreamProxy(cfg.Upstream)

	// Probe store (per-upstream fingerprint config cache)
	prober := probe.NewProber()
	probeStore := probe.NewStore(prober, cfg.Disguise, logger, cfg.Probe.AutoProbe)

	// Handler
	messagesHandler := handler.NewMessagesHandler(cfg, upstreamProxy, logger, km, probeStore)

	// Middleware chain
	auth := middleware.Auth(cfg, km)
	logging := middleware.Logging(reqLogger)

	// Routes
	mux := http.NewServeMux()

	// Messages API — main endpoint
	mux.Handle("/v1/messages", logging(auth(messagesHandler)))

	// Admin API
	adminAPI := admin.NewAdminAPI(cfg, reqLogger, km, probeStore)
	adminAPI.RegisterRoutes(mux)

	// Health
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// WebUI (only for exact "/" path)
	staticFS, err := fs.Sub(web.StaticFS, "static")
	if err != nil {
		logger.Error("failed to load static files", "error", err)
		os.Exit(1)
	}
	fileServer := http.FileServer(http.FS(staticFS))

	// Passthrough proxy: forward non-matched /v1/* requests to upstream
	passthroughProxy := proxy.NewPassthroughHandler(cfg.Upstream, logger)

	// Catch-all: serve static files for known assets, passthrough for /v1/* API paths
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/v1/") && path != "/v1/messages" {
			passthroughProxy.ServeHTTP(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	logger.Info("starting bedrock gateway",
		"listen", cfg.Server.Listen,
		"upstream", cfg.Upstream.BaseURL,
		"disguise", cfg.Disguise.Enabled,
		"keymap", cfg.KeyMap.Enabled,
		"auto_probe", cfg.Probe.AutoProbe,
		"webui", "http://localhost"+cfg.Server.Listen,
	)

	// Graceful shutdown
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
