package api

import (
	"net/http"
	"strings"
)

func (a *API) RegisterRoutes(mux *http.ServeMux) {
	// 渠道测试：判断 API 是否像真实 Anthropic/Claude 渠道。
	mux.HandleFunc("/api/channel/run/stream", a.adminAuth(a.handleChannelRunStream))
	mux.HandleFunc("/api/channel/run", a.adminAuth(a.handleChannelRun))
	mux.HandleFunc("/api/channel/report", a.adminAuth(a.handleChannelReport))
	mux.HandleFunc("/api/channel/history", a.adminAuth(a.handleChannelHistory))
	mux.HandleFunc("/api/channel/history/", a.adminAuth(a.handleChannelHistoryDetail))

	// 智商测试：数据集、题库加载和模型能力运行。
	mux.HandleFunc("/api/intelligence/datasets", a.adminAuth(a.handleIntelligenceList))
	mux.HandleFunc("/api/intelligence/datasets/", a.adminAuth(a.handleIntelligenceDatasetRoute))
	mux.HandleFunc("/api/intelligence/fetch", a.adminAuth(a.handleIntelligenceFetch))
	mux.HandleFunc("/api/intelligence/history", a.adminAuth(a.handleIntelligenceHistory))
	mux.HandleFunc("/api/intelligence/history/", a.adminAuth(a.handleIntelligenceHistoryDetail))

	// 综合审计：只编排渠道测试 + 智商测试，不承载具体检测逻辑。
	mux.HandleFunc("/api/audit/run", a.adminAuth(a.handleAuditRun))

	// 持续监控：target 管理、调度、运行记录、健康状态。
	mux.HandleFunc("/api/monitor/targets", a.adminAuth(a.handleMonitorTargets))
	mux.HandleFunc("/api/monitor/targets/", a.adminAuth(a.handleMonitorTargetDetail))
	mux.HandleFunc("/api/monitor/runs", a.adminAuth(a.handleMonitorRuns))
	mux.HandleFunc("/api/monitor/status", a.adminAuth(a.handleMonitorStatus))

	// 基线：录制 / 列表 / 删除。
	mux.HandleFunc("/api/monitor/baselines", a.adminAuth(a.handleMonitorBaselines))
	mux.HandleFunc("/api/monitor/baselines/", a.adminAuth(a.handleMonitorBaselineDetail))

	// 告警：事件列表、规则查看。
	mux.HandleFunc("/api/alert/events", a.adminAuth(a.handleAlertEvents))
	mux.HandleFunc("/api/alert/rules", a.adminAuth(a.handleAlertRules))
}

func (a *API) adminAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(a.cfg.Admin.Token)
		if token != "" && r.Header.Get("X-Admin-Token") != token {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
			return
		}
		h(w, r)
	}
}
