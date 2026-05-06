package api

import (
	"log/slog"

	"detector-service/internal/audit"
	"detector-service/internal/channeltest"
	"detector-service/internal/config"
	"detector-service/internal/intelligence"
)

type API struct {
	cfg                  *config.Config
	logger               *slog.Logger
	channelStore         *channeltest.Store
	intelligenceRegistry *intelligence.Registry
	intelligenceRunner   *intelligence.Runner
	intelligenceHistory  *intelligence.HistoryStore
	auditRunner          *audit.Runner
}

func New(cfg *config.Config, logger *slog.Logger, channelStore *channeltest.Store, intelligenceRegistry *intelligence.Registry, intelligenceRunner *intelligence.Runner) *API {
	return &API{
		cfg:                  cfg,
		logger:               logger,
		channelStore:         channelStore,
		intelligenceRegistry: intelligenceRegistry,
		intelligenceRunner:   intelligenceRunner,
		intelligenceHistory:  intelligence.NewHistoryStore(),
		auditRunner:          audit.NewRunner(channelStore, intelligenceRegistry, intelligenceRunner),
	}
}
