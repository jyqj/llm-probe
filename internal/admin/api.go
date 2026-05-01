package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bedrock-gateway/internal/config"
	"bedrock-gateway/internal/keymap"
	"bedrock-gateway/internal/probe"
)

// AdminAPI provides the management API endpoints.
type AdminAPI struct {
	cfg        *config.Config
	logger     *RequestLogger
	keyMap     *keymap.KeyMap
	probeStore *probe.Store
}

// NewAdminAPI creates the admin API handler.
func NewAdminAPI(cfg *config.Config, logger *RequestLogger, km *keymap.KeyMap, ps *probe.Store) *AdminAPI {
	return &AdminAPI{
		cfg:        cfg,
		logger:     logger,
		keyMap:     km,
		probeStore: ps,
	}
}

// RegisterRoutes registers admin API routes on the given mux.
func (a *AdminAPI) RegisterRoutes(mux *http.ServeMux) {
	// Stats & logs - require admin auth for sensitive endpoints
	mux.HandleFunc("/api/stats", a.handleStats) // public stats OK
	mux.HandleFunc("/api/logs", a.adminAuth(a.handleLogs))
	mux.HandleFunc("/api/config", a.adminAuth(a.handleConfig))
	mux.HandleFunc("/api/config/models", a.adminAuth(a.handleModels))
	mux.HandleFunc("/api/config/keys", a.adminAuth(a.handleKeys))
	mux.HandleFunc("/api/config/sanitize", a.adminAuth(a.handleSanitize))
	mux.HandleFunc("/api/config/upstream", a.adminAuth(a.handleUpstream))

	// Key map management (new, from Python)
	mux.HandleFunc("/admin/register", a.adminAuth(a.handleRegister))
	mux.HandleFunc("/admin/keys", a.adminAuth(a.handleKeyMapList))
	mux.HandleFunc("/admin/keys/", a.adminAuth(a.handleKeyMapDelete))
	mux.HandleFunc("/admin/bulk_register", a.adminAuth(a.handleBulkRegister))
	mux.HandleFunc("/admin/test/", a.adminAuth(a.handleKeyMapTest))
	mux.HandleFunc("/admin/export", a.adminAuth(a.handleExport))
	mux.HandleFunc("/admin/import", a.adminAuth(a.handleImport))

	// Probe endpoints
	mux.HandleFunc("/admin/probe", a.adminAuth(a.handleProbe))
	mux.HandleFunc("/admin/probe/report", a.adminAuth(a.handleProbeReport))
	mux.HandleFunc("/admin/probe/apply", a.adminAuth(a.handleProbeApply))
	mux.HandleFunc("/admin/probe/quick", a.adminAuth(a.handleQuickProbe))

	// Disguise config endpoints
	mux.HandleFunc("/api/config/disguise", a.adminAuth(a.handleDisguise))
	mux.HandleFunc("/api/config/disguise/upstream", a.adminAuth(a.handleDisguiseUpstream))
}

func (a *AdminAPI) adminAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Admin-Token")
		if a.cfg.KeyMap.AdminToken == "" || token != a.cfg.KeyMap.AdminToken {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		h(w, r)
	}
}

// === Key Map Admin Endpoints ===

func (a *AdminAPI) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	if a.keyMap == nil {
		jsonResp(w, 400, map[string]any{"error": "keymap not enabled"})
		return
	}
	var body struct {
		UpstreamBase  string `json:"upstream_base"`
		UpstreamKey   string `json:"upstream_key"`
		Label         string `json:"label"`
		DownstreamKey string `json:"downstream_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonResp(w, 400, map[string]any{"error": "invalid json"})
		return
	}
	if body.UpstreamBase == "" || body.UpstreamKey == "" {
		jsonResp(w, 400, map[string]any{"error": "upstream_base and upstream_key required"})
		return
	}
	downKey := a.keyMap.Register(
		strings.TrimRight(body.UpstreamBase, "/"),
		body.UpstreamKey, body.Label, body.DownstreamKey,
	)
	jsonResp(w, 200, map[string]any{
		"downstream_key": downKey,
		"gateway_base":   a.cfg.KeyMap.PublicGatewayBase,
		"label":          body.Label,
	})
}

func (a *AdminAPI) handleKeyMapList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	if a.keyMap == nil {
		jsonResp(w, 200, map[string]any{"keys": []any{}, "gateway_base": a.cfg.KeyMap.PublicGatewayBase})
		return
	}
	entries := a.keyMap.List()
	out := make([]map[string]any, 0, len(entries))
	for k, v := range entries {
		preview := "****"
		if len(v.UpstreamKey) > 12 {
			preview = v.UpstreamKey[:8] + "..." + v.UpstreamKey[len(v.UpstreamKey)-4:]
		}
		out = append(out, map[string]any{
			"downstream_key":     k,
			"upstream_base":      v.UpstreamBase,
			"upstream_key_preview": preview,
			"label":              v.Label,
			"created_at":         v.CreatedAt,
		})
	}
	jsonResp(w, 200, map[string]any{"keys": out, "gateway_base": a.cfg.KeyMap.PublicGatewayBase})
}

func (a *AdminAPI) handleKeyMapDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", 405)
		return
	}
	if a.keyMap == nil {
		jsonResp(w, 400, map[string]any{"error": "keymap not enabled"})
		return
	}
	key := strings.TrimPrefix(r.URL.Path, "/admin/keys/")
	removed := a.keyMap.Delete(key)
	jsonResp(w, 200, map[string]any{"removed": removed})
}

func (a *AdminAPI) handleBulkRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	if a.keyMap == nil {
		jsonResp(w, 400, map[string]any{"error": "keymap not enabled"})
		return
	}
	var body struct {
		Entries []struct {
			UpstreamBase  string `json:"upstream_base"`
			UpstreamKey   string `json:"upstream_key"`
			Label         string `json:"label"`
			DownstreamKey string `json:"downstream_key"`
		} `json:"entries"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonResp(w, 400, map[string]any{"error": "invalid json"})
		return
	}
	results := make([]map[string]any, 0, len(body.Entries))
	for _, e := range body.Entries {
		if e.UpstreamBase == "" || e.UpstreamKey == "" {
			results = append(results, map[string]any{"error": "upstream_base/upstream_key required"})
			continue
		}
		downKey := a.keyMap.Register(
			strings.TrimRight(e.UpstreamBase, "/"),
			e.UpstreamKey, e.Label, e.DownstreamKey,
		)
		results = append(results, map[string]any{
			"downstream_key": downKey,
			"label":          e.Label,
		})
	}
	jsonResp(w, 200, map[string]any{"results": results, "gateway_base": a.cfg.KeyMap.PublicGatewayBase})
}

func (a *AdminAPI) handleKeyMapTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	if a.keyMap == nil {
		jsonResp(w, 400, map[string]any{"error": "keymap not enabled"})
		return
	}
	key := strings.TrimPrefix(r.URL.Path, "/admin/test/")
	entry := a.keyMap.Resolve(key)
	if entry == nil {
		jsonResp(w, 404, map[string]any{"ok": false, "error": "key not found"})
		return
	}
	// Simple connectivity test
	url := strings.TrimRight(entry.UpstreamBase, "/") + "/v1/messages"
	testModel := "claude-sonnet-4-20250514"
	if a.cfg.Models.DefaultModel != "" {
		testModel = a.cfg.Models.DefaultModel
	}
	testBody := `{"model":"` + testModel + `","max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`
	req, _ := http.NewRequest("POST", url, strings.NewReader(testBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", entry.UpstreamKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30e9}
	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		jsonResp(w, 200, map[string]any{"ok": false, "error": err.Error(), "elapsed_ms": elapsed})
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var rj map[string]any
	json.Unmarshal(respBody, &rj)
	ok := resp.StatusCode == 200
	if rj != nil {
		_, hasContent := rj["content"]
		ok = ok && hasContent
	}
	result := map[string]any{
		"ok":         ok,
		"status":     resp.StatusCode,
		"elapsed_ms": elapsed,
	}
	if rj != nil {
		result["model"] = rj["model"]
		if !ok {
			if errObj, _ := rj["error"].(map[string]any); errObj != nil {
				result["error"] = errObj["message"]
			}
		}
	}
	jsonResp(w, 200, result)
}

func (a *AdminAPI) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	if a.keyMap == nil {
		jsonResp(w, 200, map[string]any{})
		return
	}
	data := a.keyMap.Export()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="keys.json"`)
	json.NewEncoder(w).Encode(data)
}

func (a *AdminAPI) handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	if a.keyMap == nil {
		jsonResp(w, 400, map[string]any{"error": "keymap not enabled"})
		return
	}
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		jsonResp(w, 400, map[string]any{"error": "invalid json"})
		return
	}
	// Try {merge, data} format
	var wrapper struct {
		Merge bool                      `json:"merge"`
		Data  map[string]*keymap.Entry  `json:"data"`
	}
	merge := true
	var data map[string]*keymap.Entry
	if json.Unmarshal(raw, &wrapper) == nil && wrapper.Data != nil {
		merge = wrapper.Merge
		data = wrapper.Data
	} else {
		// Direct dict format
		json.Unmarshal(raw, &data)
	}
	if data == nil {
		jsonResp(w, 400, map[string]any{"error": "data must be dict"})
		return
	}
	count := a.keyMap.Import(data, merge)
	jsonResp(w, 200, map[string]any{
		"imported": count,
		"total":    a.keyMap.Count(),
		"merge":    merge,
	})
}

// === Probe Endpoints ===

func (a *AdminAPI) handleProbe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}

	var body struct {
		TargetBase string `json:"target_base"`
		TargetKey  string `json:"target_key"`
		Model      string `json:"model"`
		Quick      bool   `json:"quick"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonResp(w, 400, map[string]any{"error": "invalid json"})
		return
	}

	if body.TargetBase == "" {
		body.TargetBase = a.cfg.Upstream.BaseURL
	}
	if body.TargetKey == "" {
		body.TargetKey = a.cfg.Upstream.APIKey
	}
	if body.Model == "" {
		body.Model = "claude-opus-4-6"
	}

	report, err := a.probeStore.ProbeSync(body.TargetBase, body.TargetKey, body.Model, body.Quick)
	if err != nil {
		jsonResp(w, 500, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	jsonResp(w, 200, map[string]any{
		"ok":          true,
		"target":      report.Target,
		"elapsed_ms":  report.ElapsedMs,
		"model":       report.Model,
		"checks":      report.Checks,
		"summary":     report.Summary,
		"recommended": report.Recommended,
		"score":       report.Score,
	})
}

func (a *AdminAPI) handleProbeReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}

	// If target_base is specified, show that upstream's report; otherwise list all.
	targetBase := r.URL.Query().Get("target_base")
	if targetBase != "" {
		entry := a.probeStore.GetEntry(targetBase)
		if entry == nil || entry.Report == nil {
			jsonResp(w, 404, map[string]any{"error": "no probe report for " + targetBase})
			return
		}
		rpt := entry.Report
		jsonResp(w, 200, map[string]any{
			"ok":          true,
			"target":      rpt.Target,
			"model":       rpt.Model,
			"timestamp":   rpt.Timestamp,
			"elapsed_ms":  rpt.ElapsedMs,
			"checks":      rpt.Checks,
			"summary":     rpt.Summary,
			"recommended": rpt.Recommended,
			"probed_at":   entry.ProbedAt,
			"score":       rpt.Score,
		})
		return
	}

	// List all cached probe results
	entries := a.probeStore.ListEntries()
	results := make([]map[string]any, 0, len(entries))
	for base, entry := range entries {
		item := map[string]any{
			"target":    base,
			"probed_at": entry.ProbedAt,
			"probing":   entry.Probing,
		}
		if entry.Report != nil {
			item["model"] = entry.Report.Model
			item["summary"] = entry.Report.Summary
			item["checks_total"] = len(entry.Report.Checks)
			passed := 0
			for _, c := range entry.Report.Checks {
				if c.Pass {
					passed++
				}
			}
			item["checks_passed"] = passed
			if entry.Report.Score != nil {
				item["score"] = entry.Report.Score.TotalScore
				item["grade"] = entry.Report.Score.Grade
				item["grade_color"] = entry.Report.Score.GradeColor
				item["verdict"] = entry.Report.Score.Verdict
				item["verdict_label"] = entry.Report.Score.VerdictLabel
				item["verdict_color"] = entry.Report.Score.VerdictColor
			}
		}
		results = append(results, item)
	}
	jsonResp(w, 200, map[string]any{"ok": true, "upstreams": results})
}

func (a *AdminAPI) handleProbeApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var body struct {
		TargetBase  string                `json:"target_base"`
		Recommended config.DisguiseConfig `json:"recommended"`
		ApplyGlobal bool                  `json:"apply_global"` // also update global config
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonResp(w, 400, map[string]any{"error": "invalid json"})
		return
	}

	rec := body.Recommended
	rec.Enabled = true

	applied := 0
	if rec.BodyRewrite { applied++ }
	if rec.IDRewrite { applied++ }
	if rec.SignatureRewrite { applied++ }
	if rec.HeadersFake { applied++ }
	if rec.StripDone { applied++ }
	if rec.StripContainer { applied++ }
	if rec.StripBedrock { applied++ }
	if rec.ForceGeo { applied++ }
	if rec.ThinkingInject { applied++ }
	if rec.SmallProbeZero { applied++ }
	if rec.CacheFake { applied++ }

	// Store in ProbeStore for per-upstream use
	targetBase := body.TargetBase
	if targetBase == "" {
		targetBase = a.cfg.Upstream.BaseURL
	}
	a.probeStore.SetResult(targetBase, nil, rec)

	// Optionally also update global config
	if body.ApplyGlobal {
		a.cfg.Disguise.Enabled = true
		a.cfg.Disguise.BodyRewrite = rec.BodyRewrite
		a.cfg.Disguise.IDRewrite = rec.IDRewrite
		a.cfg.Disguise.SignatureRewrite = rec.SignatureRewrite
		a.cfg.Disguise.HeadersFake = rec.HeadersFake
		a.cfg.Disguise.StripDone = rec.StripDone
		a.cfg.Disguise.StripContainer = rec.StripContainer
		a.cfg.Disguise.StripBedrock = rec.StripBedrock
		a.cfg.Disguise.ForceGeo = rec.ForceGeo
		a.cfg.Disguise.ThinkingInject = rec.ThinkingInject
		a.cfg.Disguise.SmallProbeZero = rec.SmallProbeZero
		a.cfg.Disguise.CacheFake = rec.CacheFake
		// Update fallback in ProbeStore too
		a.probeStore.SetFallback(a.cfg.Disguise)
	}

	jsonResp(w, 200, map[string]any{"ok": true, "applied": applied, "target": targetBase, "apply_global": body.ApplyGlobal})
}

// === Existing admin endpoints ===

func (a *AdminAPI) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	stats := a.logger.GetStats()
	writeAdminJSON(w, stats)
}

func (a *AdminAPI) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 50
	}
	logs, total := a.logger.GetLogs(limit, offset)
	writeAdminJSON(w, map[string]any{
		"logs":  logs,
		"total": total,
	})
}

func (a *AdminAPI) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	maskedKey := "****"
	if len(a.cfg.Upstream.APIKey) > 8 {
		maskedKey = a.cfg.Upstream.APIKey[:4] + "****" + a.cfg.Upstream.APIKey[len(a.cfg.Upstream.APIKey)-4:]
	}
	safeConfig := map[string]any{
		"server":   map[string]any{"listen": a.cfg.Server.Listen},
		"upstream": map[string]any{"base_url": a.cfg.Upstream.BaseURL, "api_key": maskedKey, "timeout": a.cfg.Upstream.Timeout},
		"auth":     map[string]any{"key_count": len(a.cfg.Auth.APIKeys)},
		"sanitize": a.cfg.Sanitize,
		"models":   a.cfg.Models,
		"log":      a.cfg.Log,
		"disguise": a.cfg.Disguise,
		"keymap":   map[string]any{"enabled": a.cfg.KeyMap.Enabled, "strict": a.cfg.KeyMap.Strict, "key_count": func() int { if a.keyMap != nil { return a.keyMap.Count() }; return 0 }()},
	}
	writeAdminJSON(w, safeConfig)
}

func (a *AdminAPI) handleModels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeAdminJSON(w, a.cfg.Models)
	case http.MethodPut:
		var models config.ModelConfig
		if err := json.NewDecoder(r.Body).Decode(&models); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		a.cfg.Models = models
		writeAdminJSON(w, map[string]string{"status": "ok"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *AdminAPI) handleKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		masked := make([]string, len(a.cfg.Auth.APIKeys))
		for i, k := range a.cfg.Auth.APIKeys {
			if len(k) > 8 {
				masked[i] = k[:4] + "****" + k[len(k)-4:]
			} else {
				masked[i] = "****"
			}
		}
		writeAdminJSON(w, map[string]any{"keys": masked, "count": len(masked)})
	case http.MethodPost:
		var body struct {
			Key string `json:"key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Key == "" {
			http.Error(w, "invalid key", http.StatusBadRequest)
			return
		}
		a.cfg.Auth.APIKeys = append(a.cfg.Auth.APIKeys, body.Key)
		writeAdminJSON(w, map[string]string{"status": "ok"})
	case http.MethodDelete:
		var body struct {
			Index int `json:"index"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		if body.Index < 0 || body.Index >= len(a.cfg.Auth.APIKeys) {
			http.Error(w, "index out of range", http.StatusBadRequest)
			return
		}
		a.cfg.Auth.APIKeys = append(a.cfg.Auth.APIKeys[:body.Index], a.cfg.Auth.APIKeys[body.Index+1:]...)
		writeAdminJSON(w, map[string]string{"status": "ok"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *AdminAPI) handleSanitize(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeAdminJSON(w, a.cfg.Sanitize)
	case http.MethodPut:
		var sanitize config.SanitizeConfig
		if err := json.NewDecoder(r.Body).Decode(&sanitize); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		a.cfg.Sanitize = sanitize
		writeAdminJSON(w, map[string]string{"status": "ok"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *AdminAPI) handleUpstream(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		maskedKey := "****"
		if len(a.cfg.Upstream.APIKey) > 8 {
			maskedKey = a.cfg.Upstream.APIKey[:4] + "****" + a.cfg.Upstream.APIKey[len(a.cfg.Upstream.APIKey)-4:]
		}
		writeAdminJSON(w, map[string]any{
			"base_url": a.cfg.Upstream.BaseURL,
			"api_key":  maskedKey,
			"timeout":  a.cfg.Upstream.Timeout,
		})
	case http.MethodPut:
		var body struct {
			BaseURL string `json:"base_url"`
			APIKey  string `json:"api_key"`
			Timeout int    `json:"timeout"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body.BaseURL != "" {
			a.cfg.Upstream.BaseURL = body.BaseURL
		}
		if body.APIKey != "" {
			a.cfg.Upstream.APIKey = body.APIKey
		}
		if body.Timeout > 0 {
			a.cfg.Upstream.Timeout = body.Timeout
		}
		writeAdminJSON(w, map[string]string{"status": "ok"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func writeAdminJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonResp(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// === Disguise Config Endpoints ===

func (a *AdminAPI) handleDisguise(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Return global disguise config with switch metadata
		jsonResp(w, 200, map[string]any{
			"config":   a.cfg.Disguise,
			"switches": disguiseSwitchMeta(),
		})
	case http.MethodPut:
		var body config.DisguiseConfig
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonResp(w, 400, map[string]any{"error": "invalid json: " + err.Error()})
			return
		}
		// Update global config
		a.cfg.Disguise = body
		a.probeStore.SetFallback(body)
		jsonResp(w, 200, map[string]any{"ok": true, "config": a.cfg.Disguise})
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (a *AdminAPI) handleDisguiseUpstream(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// List all per-upstream configs
		entries := a.probeStore.ListEntries()
		results := make([]map[string]any, 0, len(entries))
		for base, entry := range entries {
			item := map[string]any{
				"upstream_base": base,
				"config":        entry.Config,
				"probed_at":     entry.ProbedAt,
			}
			if entry.Report != nil {
				item["has_report"] = true
				if entry.Report.Score != nil {
					item["score"] = entry.Report.Score.TotalScore
					item["grade"] = entry.Report.Score.Grade
				}
			}
			results = append(results, item)
		}
		jsonResp(w, 200, map[string]any{"upstreams": results})

	case http.MethodPut:
		var body struct {
			UpstreamBase string                `json:"upstream_base"`
			Config       config.DisguiseConfig `json:"config"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonResp(w, 400, map[string]any{"error": "invalid json"})
			return
		}
		if body.UpstreamBase == "" {
			jsonResp(w, 400, map[string]any{"error": "upstream_base required"})
			return
		}
		// Update per-upstream config (preserve existing report if any)
		entry := a.probeStore.GetEntry(body.UpstreamBase)
		var report *probe.ProbeReport
		if entry != nil {
			report = entry.Report
		}
		a.probeStore.SetResult(body.UpstreamBase, report, body.Config)
		jsonResp(w, 200, map[string]any{"ok": true, "upstream_base": body.UpstreamBase})

	case http.MethodDelete:
		upstreamBase := r.URL.Query().Get("upstream_base")
		if upstreamBase == "" {
			jsonResp(w, 400, map[string]any{"error": "upstream_base query param required"})
			return
		}
		a.probeStore.Delete(upstreamBase)
		jsonResp(w, 200, map[string]any{"ok": true, "deleted": upstreamBase})

	default:
		http.Error(w, "method not allowed", 405)
	}
}

// handleQuickProbe runs a minimal probe (precheck only) for fast testing.
func (a *AdminAPI) handleQuickProbe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}

	var body struct {
		TargetBase string `json:"target_base"`
		TargetKey  string `json:"target_key"`
		Model      string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonResp(w, 400, map[string]any{"error": "invalid json"})
		return
	}

	if body.TargetBase == "" {
		body.TargetBase = a.cfg.Upstream.BaseURL
	}
	if body.TargetKey == "" {
		body.TargetKey = a.cfg.Upstream.APIKey
	}
	if body.Model == "" {
		body.Model = "claude-opus-4-6"
	}

	// Run quick probe (precheck + tag_replay only)
	report, err := a.probeStore.ProbeSync(body.TargetBase, body.TargetKey, body.Model, true)
	if err != nil {
		jsonResp(w, 200, map[string]any{
			"ok":      false,
			"error":   err.Error(),
			"target":  body.TargetBase,
		})
		return
	}

	// Summarize what switches are needed
	neededSwitches := make([]string, 0)
	for _, c := range report.Checks {
		if !c.Pass && c.Fix != "" {
			found := false
			for _, s := range neededSwitches {
				if s == c.Fix {
					found = true
					break
				}
			}
			if !found {
				neededSwitches = append(neededSwitches, c.Fix)
			}
		}
	}

	passed := 0
	for _, c := range report.Checks {
		if c.Pass {
			passed++
		}
	}

	jsonResp(w, 200, map[string]any{
		"ok":              true,
		"target":          body.TargetBase,
		"model":           body.Model,
		"elapsed_ms":      report.ElapsedMs,
		"checks_passed":   passed,
		"checks_total":    len(report.Checks),
		"needed_switches": neededSwitches,
		"recommended":     report.Recommended,
		"score":           report.Score,
	})
}

// disguiseSwitchMeta returns metadata for all disguise switches (for UI grouping).
func disguiseSwitchMeta() []map[string]any {
	return []map[string]any{
		// Body Rewrite Group
		{"key": "body_rewrite", "label": "Body Rewrite", "group": "body", "desc": "Fix JSON field ordering, usage structure, stop_details"},
		{"key": "id_rewrite", "label": "ID Rewrite", "group": "body", "desc": "Convert msg_bdrk_/gen- to msg_01 format"},
		{"key": "signature_rewrite", "label": "Signature Rewrite", "group": "body", "desc": "Re-sign thinking.signature with HMAC"},
		{"key": "strip_bedrock", "label": "Strip Bedrock", "group": "body", "desc": "Remove bedrock_state from usage"},
		{"key": "strip_container", "label": "Strip Container", "group": "body", "desc": "Remove container field from response"},
		{"key": "force_geo", "label": "Force Geo", "group": "body", "desc": "Override inference_geo by model"},

		// Headers Group
		{"key": "headers_fake", "label": "Fake Headers", "group": "headers", "desc": "Add Anthropic ratelimit headers, Cf-Ray, Request-Id"},

		// SSE Group
		{"key": "strip_done", "label": "Strip [DONE]", "group": "sse", "desc": "Remove [DONE] sentinel from SSE stream"},
		{"key": "sse_padding", "label": "SSE Padding", "group": "sse", "desc": "Add random whitespace in SSE data (rarely needed)"},

		// Thinking Group
		{"key": "thinking_inject", "label": "Thinking Inject", "group": "thinking", "desc": "Inject thinking block when response has none"},
		{"key": "strip_thinking", "label": "Strip Thinking", "group": "thinking", "desc": "Strip thinking from assistant messages before forwarding"},

		// Cache Group
		{"key": "small_probe_zero", "label": "Small Probe Zero", "group": "cache", "desc": "Set cache=0 for max_tokens<=10 requests"},
		{"key": "cache_fake", "label": "Cache Fake", "group": "cache", "desc": "Fake cache usage when upstream returns all zeros"},

		// Token Group
		{"key": "max_tokens_clamp", "label": "Max Tokens Clamp", "group": "tokens", "desc": "Clamp output_tokens <= max_tokens"},

		// Advanced Group
		{"key": "refusal", "label": "Refusal Intercept", "group": "advanced", "desc": "Intercept system prompt leak attempts"},
		{"key": "sig_verify", "label": "Signature Verify", "group": "advanced", "desc": "Verify client-returned thinking.signature"},
		{"key": "identity", "label": "Identity Inject", "group": "advanced", "desc": "Inject model identity system prompt"},
		{"key": "identity_hide", "label": "Identity Hide", "group": "advanced", "desc": "Hide injected identity token cost"},
		{"key": "web_search", "label": "Web Search", "group": "advanced", "desc": "Synthesize web search responses locally"},

		// Passthrough Group
		{"key": "passthrough_body", "label": "Passthrough Body", "group": "passthrough", "desc": "Skip all body rewrite (raw proxy)"},
		{"key": "passthrough_headers", "label": "Passthrough Headers", "group": "passthrough", "desc": "Forward upstream headers as-is"},

		// Capture Group
		{"key": "capture_enabled", "label": "Capture Enabled", "group": "debug", "desc": "Save request/response captures to disk"},
	}
}
