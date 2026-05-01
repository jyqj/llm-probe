package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"bedrock-gateway/internal/config"
	"bedrock-gateway/internal/keymap"
)

// Auth returns an API key authentication middleware.
// If keyMap is provided and enabled, it checks both static keys and key map.
func Auth(cfg *config.Config, km *keymap.KeyMap) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := extractAPIKey(r)
			if key == "" {
				writeError(w, http.StatusUnauthorized, "authentication_error", "missing api key")
				return
			}

			// Check static API keys first
			if cfg.HasAPIKey(key) {
				next.ServeHTTP(w, r)
				return
			}

			// Check key map
			if km != nil && cfg.KeyMap.Enabled {
				if entry := km.Resolve(key); entry != nil {
					next.ServeHTTP(w, r)
					return
				}
			}

			// If keymap is strict, handler will reject later with proper error
			// If not strict, allow through and handler will use default upstream
			if cfg.KeyMap.Enabled && !cfg.KeyMap.Strict {
				next.ServeHTTP(w, r)
				return
			}

			writeError(w, http.StatusUnauthorized, "authentication_error", "invalid x-api-key")
		})
	}
}

func extractAPIKey(r *http.Request) string {
	if key := r.Header.Get("x-api-key"); key != "" {
		return strings.TrimSpace(key)
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	return ""
}

func writeError(w http.ResponseWriter, status int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"type": "error",
		"error": map[string]string{
			"type":    errType,
			"message": message,
		},
	})
}
