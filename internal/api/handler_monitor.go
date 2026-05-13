package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"detector-service/internal/intelligence"
	"detector-service/internal/monitor"
)

func (a *API) handleMonitorTargets(w http.ResponseWriter, r *http.Request) {
	if a.monitorStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "monitor not configured"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		targets := a.monitorStore.ListTargets()
		states := a.monitorStore.ListStates()
		writeJSON(w, http.StatusOK, map[string]any{"targets": targets, "states": states})

	case http.MethodPost:
		var req monitor.TargetCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		t, err := a.monitorStore.CreateTarget(req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, t)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *API) handleMonitorTargetDetail(w http.ResponseWriter, r *http.Request) {
	if a.monitorStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "monitor not configured"})
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/monitor/targets/")
	if strings.Contains(id, "/") {
		suffix := id[strings.Index(id, "/"):]
		id = id[:strings.Index(id, "/")]
		if suffix == "/run" && r.Method == http.MethodPost {
			a.handleMonitorManualRun(w, r, id)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		t := a.monitorStore.GetTarget(id)
		if t == nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "target not found"})
			return
		}
		writeJSON(w, http.StatusOK, t)

	case http.MethodPatch:
		var req monitor.TargetUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		t, err := a.monitorStore.UpdateTarget(id, req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, t)

	case http.MethodDelete:
		if a.monitorStore.DeleteTarget(id) {
			writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
		} else {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "target not found"})
		}

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *API) handleMonitorManualRun(w http.ResponseWriter, r *http.Request, targetID string) {
	if a.monitorRunner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "monitor runner not configured"})
		return
	}

	t := a.monitorStore.GetTarget(targetID)
	if t == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "target not found"})
		return
	}

	runs := a.monitorRunner.RunAll(t)

	if a.cfg.Alert.Enabled && a.alertEvaluator != nil {
		for _, run := range runs {
			state := a.monitorStore.GetState(run.TargetID, run.Model)
			events := a.alertEvaluator.Evaluate(run, state)
			if a.alertNotifier != nil {
				a.alertNotifier.NotifyAll(events)
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
}

func (a *API) handleMonitorRuns(w http.ResponseWriter, r *http.Request) {
	if a.monitorStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "monitor not configured"})
		return
	}

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	targetID := r.URL.Query().Get("target_id")
	runs := a.monitorStore.ListRuns(targetID, 50)
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
}

func (a *API) handleMonitorStatus(w http.ResponseWriter, r *http.Request) {
	if a.monitorStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "monitor not configured"})
		return
	}

	states := a.monitorStore.ListStates()
	targets := a.monitorStore.ListTargets()

	var activeAlerts []*any
	if a.alertStore != nil {
		for _, ev := range a.alertStore.ActiveEvents() {
			v := any(ev)
			activeAlerts = append(activeAlerts, &v)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"targets":       targets,
		"states":        states,
		"active_alerts": activeAlerts,
	})
}

// ═══ Baseline CRUD ═══

func (a *API) handleMonitorBaselines(w http.ResponseWriter, r *http.Request) {
	if a.baselineStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "baselines not configured"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"baselines": a.baselineStore.List()})
	case http.MethodPost:
		var req monitor.BaselineCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		baseline, err := a.recordBaseline(r.Context(), req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, baseline)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *API) handleMonitorBaselineDetail(w http.ResponseWriter, r *http.Request) {
	if a.baselineStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "baselines not configured"})
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/monitor/baselines/")
	switch r.Method {
	case http.MethodGet:
		b := a.baselineStore.Get(id)
		if b == nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "baseline not found"})
			return
		}
		writeJSON(w, http.StatusOK, b)
	case http.MethodDelete:
		if a.baselineStore.Delete(id) {
			writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
		} else {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "baseline not found"})
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *API) recordBaseline(ctx context.Context, req monitor.BaselineCreateRequest) (*monitor.Baseline, error) {
	if req.BaseURL == "" || req.APIKey == "" || req.Model == "" {
		return nil, fmt.Errorf("base_url, api_key, and model are required")
	}
	if req.ThinkingEffort == "" {
		req.ThinkingEffort = "off"
	}

	b := &monitor.Baseline{
		Name:           req.Name,
		Model:          req.Model,
		ThinkingEffort: req.ThinkingEffort,
		Effort:         req.Effort,
		ThinkingMode:   req.ThinkingMode,
		MaxTokens:      req.MaxTokens,
		CreatedAt:      time.Now(),
	}

	if a.channelRunner != nil {
		report, err := a.channelRunner.Run(req.BaseURL, req.APIKey, req.Model, 2)
		if err != nil {
			a.logger.Warn("baseline channel test failed", "error", err)
		} else {
			b.ChannelReport = report
		}
	}

	if req.Dataset != "" && a.intelligenceRegistry != nil && a.intelligenceRunner != nil {
		ds, ok := a.intelligenceRegistry.Get(req.Dataset)
		if ok {
			thinking := req.ThinkingEffort != "" && req.ThinkingEffort != "off"
			intReport, err := a.intelligenceRunner.Run(ctx, ds, intelligence.RunRequest{
				TargetBase:   req.BaseURL,
				TargetKey:    req.APIKey,
				Model:        req.Model,
				Thinking:     thinking,
				Effort:       req.Effort,
				ThinkingMode: req.ThinkingMode,
				MaxTokens:    req.MaxTokens,
			})
			if err != nil {
				a.logger.Warn("baseline intelligence test failed", "error", err)
			} else {
				b.IntelligenceReport = intReport
			}
		}
	}

	a.baselineStore.Add(b)
	return b, nil
}
