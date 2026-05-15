package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"detector-service/internal/channeltest"
	"detector-service/internal/target"
)

func (a *API) handleChannelProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"profiles": channeltest.ListProfiles()})
}

func (a *API) handleChannelRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		TargetBase  string   `json:"target_base"`
		TargetKey   string   `json:"target_key"`
		Model       string   `json:"model"`
		Models      []string `json:"models"`
		ChannelName string   `json:"channel_name"`
		Concurrency int      `json:"concurrency"`
		Profile     string   `json:"profile"`
		BaselineID  string   `json:"baseline_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json: " + err.Error()})
		return
	}
	t := target.Resolve(a.cfg, body.TargetBase, body.TargetKey, body.Model)
	if err := t.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	models := body.Models
	if len(models) == 0 {
		models = []string{t.Model}
	}

	ctx := r.Context()
	if len(models) == 1 {
		report, err := a.channelStore.RunSyncCtx(ctx, t.BaseURL, t.APIKey, models[0], body.ChannelName, body.Concurrency, body.Profile)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
			return
		}
		a.applyChannelBaselineComparison(report, body.BaselineID)
		a.channelStore.SaveReport(report)
		writeJSON(w, http.StatusOK, report)
		return
	}

	reports, err := a.channelStore.RunMultiSyncCtx(ctx, t.BaseURL, t.APIKey, body.ChannelName, models, body.Concurrency, body.Profile)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	for _, rpt := range reports {
		a.applyChannelBaselineComparison(rpt, body.BaselineID)
		a.channelStore.SaveReport(rpt)
	}
	writeJSON(w, http.StatusOK, map[string]any{"reports": reports})
}

func (a *API) handleChannelRunStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		TargetBase  string   `json:"target_base"`
		TargetKey   string   `json:"target_key"`
		Model       string   `json:"model"`
		Models      []string `json:"models"`
		ChannelName string   `json:"channel_name"`
		Concurrency int      `json:"concurrency"`
		Profile     string   `json:"profile"`
		BaselineID  string   `json:"baseline_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json: " + err.Error()})
		return
	}
	t := target.Resolve(a.cfg, body.TargetBase, body.TargetKey, body.Model)
	if err := t.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	models := body.Models
	if len(models) == 0 {
		models = []string{t.Model}
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "streaming not supported"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	mu := &sync.Mutex{}
	done := make(chan struct{})
	writeSSE := func(ev channeltest.StreamEvent) {
		data, _ := json.Marshal(ev)
		mu.Lock()
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		mu.Unlock()
	}

	// heartbeat keeps proxies/CDNs from closing idle connections
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				mu.Lock()
				fmt.Fprintf(w, ": ping\n\n")
				flusher.Flush()
				mu.Unlock()
			case <-done:
				return
			case <-r.Context().Done():
				return
			}
		}
	}()

	runID := fmt.Sprintf("%d", time.Now().UnixNano())

	totalProbes := 0
	modelProbes := make(map[string][]channeltest.ProbeInfo)
	for _, m := range models {
		probes := channeltest.ProbesForModel(m)
		totalProbes += len(probes)
		infos := make([]channeltest.ProbeInfo, len(probes))
		for i, p := range probes {
			infos[i] = channeltest.ProbeInfo{ID: p.ID, Label: p.Label}
		}
		modelProbes[m] = infos
	}

	writeSSE(channeltest.StreamEvent{
		Type:        "start",
		RunID:       runID,
		Models:      models,
		TotalProbes: totalProbes,
		ModelProbes: modelProbes,
	})

	ctx := r.Context()
	reports, err := a.channelStore.RunMultiStream(ctx, t.BaseURL, t.APIKey, body.ChannelName, models, body.Concurrency, body.Profile, writeSSE)
	close(done)
	if err != nil {
		writeSSE(channeltest.StreamEvent{Type: "error", Error: err.Error()})
		return
	}
	for _, rpt := range reports {
		a.applyChannelBaselineComparison(rpt, body.BaselineID)
		a.channelStore.SaveReport(rpt)
	}

	writeSSE(channeltest.StreamEvent{Type: "done", Reports: reports})
}

func (a *API) handleChannelHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "use /api/channel/history/{id} to delete"})
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	all := a.channelStore.ListHistorySummary()
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	total := len(all)
	if offset > total {
		offset = total
	}
	all = all[offset:]
	if limit > 0 && limit < len(all) {
		all = all[:limit]
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": all, "total": total})
}

func (a *API) handleChannelHistoryDetail(w http.ResponseWriter, r *http.Request) {
	remainder := strings.TrimPrefix(r.URL.Path, "/api/channel/history/")
	if remainder == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id is required"})
		return
	}

	// group sub-route: /api/channel/history/group/{groupID}
	if strings.HasPrefix(remainder, "group/") {
		groupID := strings.TrimPrefix(remainder, "group/")
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		reports := a.channelStore.GetHistoryGroupLite(groupID)
		if len(reports) == 0 {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "group not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"reports": reports})
		return
	}

	// probe sub-route: /api/channel/history/{id}/probe/{probeID}[/retry]
	if idx := strings.Index(remainder, "/probe/"); idx >= 0 {
		reportID := remainder[:idx]
		probeRest := remainder[idx+7:]

		if strings.HasSuffix(probeRest, "/retry") {
			probeID := strings.TrimSuffix(probeRest, "/retry")
			a.handleProbeRetry(w, r, reportID, probeID)
			return
		}
		a.handleProbeDetail(w, r, reportID, probeRest)
		return
	}

	id := remainder
	if r.Method == http.MethodDelete {
		if a.channelStore.DeleteHistory(id) {
			writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
		} else {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		}
		return
	}
	if r.Method == "PATCH" {
		var body struct {
			ChannelName string `json:"channel_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
			return
		}
		if a.channelStore.UpdateHistoryName(id, body.ChannelName) {
			writeJSON(w, http.StatusOK, map[string]any{"updated": true})
		} else {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		}
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Return lite report (probe_results without exchanges) by default.
	// Use ?full=1 for the complete payload including exchanges.
	if r.URL.Query().Get("full") == "1" {
		report := a.channelStore.GetHistory(id)
		if report == nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusOK, report)
		return
	}
	report := a.channelStore.GetHistoryLite(id)
	if report == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (a *API) handleProbeDetail(w http.ResponseWriter, r *http.Request, reportID, probeID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pr := a.channelStore.GetProbeResult(reportID, probeID)
	if pr == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "probe not found"})
		return
	}
	writeJSON(w, http.StatusOK, pr)
}

func (a *API) handleProbeRetry(w http.ResponseWriter, r *http.Request, reportID, probeID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		TargetKey string `json:"target_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json: " + err.Error()})
		return
	}
	if body.TargetKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "target_key is required"})
		return
	}

	report, err := a.channelStore.RetryProbe(r.Context(), reportID, probeID, body.TargetKey)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}

	// Return the updated probe result + score (lite, without exchanges from other probes)
	var retried *channeltest.ProbeResult
	for i := range report.ProbeResults {
		if report.ProbeResults[i].ProbeID == probeID {
			retried = &report.ProbeResults[i]
			break
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"probe":  retried,
		"score":  report.Score,
		"checks": report.Checks,
	})
}

// applyChannelBaselineComparison attaches baseline comparison data to the channel report.
func (a *API) applyChannelBaselineComparison(report *channeltest.Report, baselineID string) {
	if baselineID == "" || a.baselineStore == nil || report == nil {
		return
	}
	baseline := a.baselineStore.Get(baselineID)
	if baseline == nil || baseline.ChannelReport == nil {
		return
	}
	report.CompareToBaseline(baseline.ID, baseline.Name, baseline.ChannelReport)
}

func (a *API) handleChannelKeywords(w http.ResponseWriter, r *http.Request) {
	if a.keywordStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "keyword store not configured"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, a.keywordStore.ListAll())
	case http.MethodPost:
		var body struct {
			Pattern string   `json:"pattern"`
			Channel string   `json:"channel"`
			Scopes  []string `json:"scopes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json: " + err.Error()})
			return
		}
		if body.Pattern == "" || body.Channel == "" || len(body.Scopes) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "pattern, channel, and scopes are required"})
			return
		}
		kw := &channeltest.CustomKeyword{
			Pattern: body.Pattern,
			Channel: body.Channel,
			Scopes:  body.Scopes,
			Enabled: true,
		}
		a.keywordStore.Add(kw)
		writeJSON(w, http.StatusCreated, kw)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *API) handleChannelKeywordDetail(w http.ResponseWriter, r *http.Request) {
	if a.keywordStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "keyword store not configured"})
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/channel/keywords/")
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if a.keywordStore.Delete(id) {
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
	} else {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "keyword not found"})
	}
}

func (a *API) handleChannelReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	targetBase := strings.TrimSpace(r.URL.Query().Get("target_base"))
	if targetBase != "" {
		entry := a.channelStore.GetEntry(targetBase)
		if entry == nil || entry.Report == nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "no channel report for " + targetBase})
			return
		}
		writeJSON(w, http.StatusOK, entry.Report)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": a.channelStore.ListEntries()})
}
