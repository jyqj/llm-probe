package api

import (
	"encoding/json"
	"net/http"

	"detector-service/internal/audit"
	"detector-service/internal/target"
)

func (a *API) handleAuditRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		TargetBase           string `json:"target_base"`
		TargetKey            string `json:"target_key"`
		Model                string `json:"model"`
		QuickChannel         bool   `json:"quick_channel"`
		IntelligenceDataset  string `json:"intelligence_dataset"`
		IntelligenceLimit    int    `json:"intelligence_limit"`
		IntelligenceLanguage string `json:"intelligence_language"`
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

	report := a.auditRunner.Run(r.Context(), audit.Request{
		Target:               t,
		QuickChannel:         body.QuickChannel,
		IntelligenceDataset:  body.IntelligenceDataset,
		IntelligenceLimit:    body.IntelligenceLimit,
		IntelligenceLanguage: body.IntelligenceLanguage,
	})
	writeJSON(w, http.StatusOK, report)
}
