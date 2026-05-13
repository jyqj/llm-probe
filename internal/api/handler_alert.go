package api

import (
	"net/http"
)

func (a *API) handleAlertEvents(w http.ResponseWriter, r *http.Request) {
	if a.alertStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "alerts not configured"})
		return
	}

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	events := a.alertStore.ListEvents(100)
	active := a.alertStore.ActiveEvents()
	writeJSON(w, http.StatusOK, map[string]any{
		"events": events,
		"active": active,
		"total":  len(events),
	})
}

func (a *API) handleAlertRules(w http.ResponseWriter, r *http.Request) {
	if a.alertEvaluator == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "alerts not configured"})
		return
	}

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"rules": a.alertRules})
}
