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
