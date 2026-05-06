package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"detector-service/internal/target"
)

func (a *API) handleChannelRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		TargetBase string `json:"target_base"`
		TargetKey  string `json:"target_key"`
		Model      string `json:"model"`
		Quick      bool   `json:"quick"`
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
	report, err := a.channelStore.RunSync(t.BaseURL, t.APIKey, t.Model, body.Quick)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (a *API) handleChannelHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		// DELETE /api/channel/history?id=xxx — delete all history
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "use /api/channel/history/{id} to delete"})
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": a.channelStore.ListHistory()})
}

func (a *API) handleChannelHistoryDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/channel/history/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id is required"})
		return
	}
	if r.Method == http.MethodDelete {
		if a.channelStore.DeleteHistory(id) {
			writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
		} else {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		}
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	report := a.channelStore.GetHistory(id)
	if report == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, report)
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
