package api

import (
	"net/http"

	"detector-service/internal/channeltest"
)

func (a *API) handleMetaModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"models": channeltest.ListModelMetadata()})
}

func (a *API) handleChannelProbes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"probes": channeltest.ListProbeMetadata()})
}

func (a *API) handleChannelChecks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"checks": channeltest.ListCheckMetadata()})
}
