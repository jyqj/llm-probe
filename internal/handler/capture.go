package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// captureRequest saves a request/response pair to disk for debugging.
func captureRequest(dir, label string, reqJSON map[string]any, respBody []byte, respStatus int, headers http.Header) {
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}
	ts := time.Now().Format("20060102_150405.000000")
	fn := filepath.Join(dir, ts+"_"+label+".json")

	// Keep only diagnostic headers
	kept := make(map[string]string)
	for _, k := range []string{
		"anthropic-beta", "anthropic-version", "user-agent",
		"x-stainless-lang", "x-stainless-package-version",
		"x-app", "content-type", "accept",
	} {
		if v := headers.Get(k); v != "" {
			kept[k] = v
		}
	}

	clientKey := headers.Get("x-api-key")
	if clientKey == "" {
		clientKey = headers.Get("Authorization")
	}
	if len(clientKey) > 20 {
		clientKey = clientKey[:20] + "…"
	}

	respStr := string(respBody)
	if len(respStr) > 8000 {
		respStr = respStr[:8000]
	}

	data := map[string]any{
		"time":        ts,
		"label":       label,
		"client_key":  clientKey,
		"headers":     kept,
		"req":         reqJSON,
		"resp_status":  respStatus,
		"resp_body":   respStr,
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}
	if err := os.WriteFile(fn, raw, 0644); err != nil {
		// Non-fatal: capture is a side-channel, don't block requests
		_ = err
	}
}

// stripHeadersForPassthrough removes hop-by-hop and fingerprint-leaking headers.
func stripHeadersForPassthrough(src http.Header) http.Header {
	skip := map[string]bool{
		"content-length":    true,
		"content-encoding":  true,
		"transfer-encoding": true,
		"connection":        true,
		"keep-alive":        true,
		"date":              true,
	}
	out := http.Header{}
	for k, vs := range src {
		kl := strings.ToLower(k)
		if skip[kl] {
			continue
		}
		// Strip upstream-specific headers that leak non-official fingerprint
		if strings.HasPrefix(kl, "x-new-api") || strings.HasPrefix(kl, "x-oneapi") || kl == "x-cache-status" {
			continue
		}
		for _, v := range vs {
			out.Add(k, v)
		}
	}
	// Ensure Server header is set to cloudflare
	out.Set("Server", "cloudflare")
	return out
}
