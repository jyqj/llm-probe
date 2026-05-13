package api

import (
	"log/slog"

	"detector-service/internal/alert"
	"detector-service/internal/audit"
	"detector-service/internal/channeltest"
	"detector-service/internal/config"
	"detector-service/internal/intelligence"
	"detector-service/internal/monitor"
)

type API struct {
	cfg                  *config.Config
	logger               *slog.Logger
	channelStore         *channeltest.Store
	intelligenceRegistry *intelligence.Registry
	intelligenceRunner   *intelligence.Runner
	intelligenceHistory  *intelligence.HistoryStore
	auditRunner          *audit.Runner

	monitorStore   *monitor.Store
	monitorRunner  *monitor.MonitorRunner
	baselineStore  *monitor.BaselineStore
	channelRunner  *channeltest.Runner
	alertStore     *alert.Store
	alertEvaluator *alert.Evaluator
	alertNotifier  *alert.Notifier
	alertRules     []alert.Rule
}

func New(cfg *config.Config, logger *slog.Logger, channelStore *channeltest.Store, intelligenceRegistry *intelligence.Registry, intelligenceRunner *intelligence.Runner, intelligenceHistory *intelligence.HistoryStore) *API {
	return &API{
		cfg:                  cfg,
		logger:               logger,
		channelStore:         channelStore,
		intelligenceRegistry: intelligenceRegistry,
		intelligenceRunner:   intelligenceRunner,
		intelligenceHistory:  intelligenceHistory,
		auditRunner:          audit.NewRunner(channelStore, intelligenceRegistry, intelligenceRunner),
	}
}

// SetMonitor injects the monitor subsystem.
func (a *API) SetMonitor(store *monitor.Store, runner *monitor.MonitorRunner, baselineStore *monitor.BaselineStore, channelRunner *channeltest.Runner) {
	a.monitorStore = store
	a.monitorRunner = runner
	a.baselineStore = baselineStore
	a.channelRunner = channelRunner
}

// SetAlerts injects the alert subsystem.
func (a *API) SetAlerts(store *alert.Store, evaluator *alert.Evaluator, notifier *alert.Notifier, rules []alert.Rule) {
	a.alertStore = store
	a.alertEvaluator = evaluator
	a.alertNotifier = notifier
	a.alertRules = rules
}
