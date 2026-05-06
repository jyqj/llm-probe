package api

import (
	"encoding/json"
	"net/http"

	"detector-service/internal/intelligence"
)

func (a *API) handleIntelligenceFetch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Name    string `json:"name"`              // Known adapter name or custom
		Dataset string `json:"dataset,omitempty"` // HF dataset path (e.g. "org/name")
		Config  string `json:"config,omitempty"`
		Split   string `json:"split,omitempty"`
		Limit   int    `json:"limit,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json: " + err.Error()})
		return
	}

	adapter, hasAdapter := intelligence.KnownAdapter(body.Name)
	if !hasAdapter {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":     "unknown intelligence adapter: " + body.Name,
			"available": intelligence.KnownAdapterNames(),
		})
		return
	}

	// Use known HF info or override
	hfInfo, ok := intelligence.KnownHFSource(body.Name)
	if ok {
		if body.Dataset == "" {
			body.Dataset = hfInfo.Dataset
		}
		if body.Config == "" {
			body.Config = hfInfo.Config
		}
		if body.Split == "" {
			body.Split = hfInfo.Split
		}
	}
	if body.Dataset == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "dataset is required"})
		return
	}

	hf := intelligence.NewHFLoader()
	ds, err := hf.LoadDataset(body.Dataset, body.Config, body.Split, body.Limit, adapter)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}

	a.intelligenceRegistry.Register(ds)
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "dataset fetched and registered",
		"stats":   ds.Stats(),
	})
}
