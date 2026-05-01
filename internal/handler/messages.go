package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"bedrock-gateway/internal/config"
	"bedrock-gateway/internal/fingerprint"
	"bedrock-gateway/internal/keymap"
	"bedrock-gateway/internal/middleware"
	"bedrock-gateway/internal/probe"
	"bedrock-gateway/internal/proxy"
	"bedrock-gateway/internal/types"
)

// MessagesHandler handles /v1/messages requests.
type MessagesHandler struct {
	cfg        *config.Config
	proxy      *proxy.UpstreamProxy
	sanitizer  *middleware.Sanitizer
	logger     *slog.Logger
	keyMap     *keymap.KeyMap
	probeStore *probe.Store
	dumpMu     sync.Mutex
}

// NewMessagesHandler creates a new handler.
func NewMessagesHandler(cfg *config.Config, p *proxy.UpstreamProxy, logger *slog.Logger, km *keymap.KeyMap, ps *probe.Store) *MessagesHandler {
	return &MessagesHandler{
		cfg:        cfg,
		proxy:      p,
		sanitizer:  middleware.NewSanitizer(cfg.Sanitize),
		logger:     logger,
		keyMap:     km,
		probeStore: ps,
	}
}

// disguiseCfg returns the DisguiseConfig for the given upstream target.
// If ProbeStore has a cached per-upstream config, merge it with global settings.
// Per-upstream probe results only control the "fixable" switches; global-only
// switches (SSEPadding, Passthrough*, MaxTokensClamp) always come from the global config.
func (h *MessagesHandler) disguiseCfg(target *proxy.UpstreamTarget, model string) *config.DisguiseConfig {
	d := &h.cfg.Disguise

	// Try per-upstream config from ProbeStore
	if h.probeStore != nil && target != nil && target.BaseURL != "" {
		perUp := h.probeStore.GetConfig(target.BaseURL, target.APIKey, model)
		// Start from per-upstream probe result, override global-only switches.
		merged := perUp // value copy
		merged.MaxTokensClamp = d.MaxTokensClamp
		merged.SSEPadding = d.SSEPadding
		merged.PassthroughBody = d.PassthroughBody
		merged.PassthroughHeaders = d.PassthroughHeaders
		return &merged
	}

	// Fallback: return a copy of global config to avoid concurrent mutation.
	cp := *d
	return &cp
}

// ServeHTTP handles the /v1/messages endpoint.
func (h *MessagesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "invalid_request_error", "method not allowed")
		return
	}

	// Limit request body to 10MB
	const maxRequestSize = 10 * 1024 * 1024
	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestSize))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request_error", "failed to read request body")
		return
	}
	defer r.Body.Close()
	if len(body) >= maxRequestSize {
		writeJSONError(w, http.StatusRequestEntityTooLarge, "invalid_request_error", "request body too large (max 10MB)")
		return
	}

	// Parse into typed struct for sanitization
	var req types.MessagesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request_error", fmt.Sprintf("invalid JSON: %v", err))
		return
	}

	// Sanitize request
	result := h.sanitizer.SanitizeMessagesRequest(&req)
	if result.Blocked {
		h.logger.Warn("request blocked", "reason", result.BlockReason)
		writeJSONError(w, http.StatusBadRequest, "invalid_request_error", result.BlockReason)
		return
	}
	for _, warn := range result.Warnings {
		h.logger.Debug("sanitize warning", "warning", warn)
	}

	// Apply model mapping
	req.Model = h.resolveModel(req.Model)

	// Also parse into map for flexible manipulation (fields not covered by the
	// typed struct, e.g. unknown/future Anthropic fields, are preserved here).
	var reqJSON map[string]any
	if err := json.Unmarshal(body, &reqJSON); err != nil || reqJSON == nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request_error", fmt.Sprintf("invalid JSON (map): %v", err))
		return
	}
	// Apply all sanitized fields back to the map
	reqJSON["model"] = req.Model
	if result.Modified {
		reqJSON["max_tokens"] = float64(req.MaxTokens)
		if req.Temperature != nil {
			reqJSON["temperature"] = *req.Temperature
		}
		if req.TopP != nil {
			reqJSON["top_p"] = *req.TopP
		} else {
			delete(reqJSON, "top_p")
		}
		if req.OutputConfig != nil {
			oc, _ := json.Marshal(req.OutputConfig)
			var ocMap any
			json.Unmarshal(oc, &ocMap)
			reqJSON["output_config"] = ocMap
		} else {
			delete(reqJSON, "output_config")
		}
		if req.System != nil {
			reqJSON["system"] = req.System
		}
		if req.Thinking != nil {
			tk, _ := json.Marshal(req.Thinking)
			var tkMap any
			json.Unmarshal(tk, &tkMap)
			reqJSON["thinking"] = tkMap
		}
		// Re-normalize message content (string -> text block)
		if msgs, ok := reqJSON["messages"].([]any); ok {
			for i, msg := range msgs {
				if m, ok := msg.(map[string]any); ok {
					if i < len(req.Messages) {
						m["content"] = req.Messages[i].Content
					}
				}
			}
		}
	}

	// Extract client key
	clientKey := extractClientKey(r)

	// Resolve upstream via key map
	var target *proxy.UpstreamTarget
	if h.keyMap != nil && h.cfg.KeyMap.Enabled {
		entry := h.keyMap.Resolve(clientKey)
		if entry != nil {
			target = &proxy.UpstreamTarget{
				BaseURL: entry.UpstreamBase,
				APIKey:  entry.UpstreamKey,
			}
		} else if h.cfg.KeyMap.Strict {
			writeJSONError(w, http.StatusUnauthorized, "authentication_error", "invalid x-api-key")
			return
		}
	}

	model := req.Model
	stream := req.Stream

	// Dump request if enabled
	if h.cfg.Log.DumpRequests {
		h.dumpEntry(map[string]any{
			"type":      "request",
			"timestamp": time.Now().Format(time.RFC3339Nano),
			"client_ip": r.RemoteAddr,
			"body":      json.RawMessage(body),
		})
	}

	// === #6 Refusal: intercept system prompt leak attempts ===
	if h.cfg.Disguise.Enabled && h.cfg.Disguise.Refusal && checkRefusal(reqJSON) {
		h.logger.Warn("refusal triggered: system prompt leak attempt")
		h.serveRefusal(w, model, clientKey, stream, reqJSON)
		return
	}

	// === #7 Web Search synthesis (default off) ===
	if h.cfg.Disguise.Enabled && h.cfg.Disguise.WebSearch && hasWebSearchTool(reqJSON) {
		h.logger.Info("web search synthesis triggered", "model", model)
		h.serveWebSearch(w, model, clientKey, req.Stream, reqJSON)
		return
	}

	// === #9 Signature verification (default off) ===
	if h.cfg.Disguise.Enabled && h.cfg.Disguise.SigVerify {
		if errMsg := verifyRequestSignatures(reqJSON); errMsg != "" {
			writeJSONError(w, http.StatusBadRequest, "invalid_request_error", errMsg)
			return
		}
	}

	// === Strip thinking from assistant messages (upstream may not support multi-turn thinking) ===
	if h.cfg.Disguise.Enabled && h.cfg.Disguise.StripThinking {
		stripThinkingFromMessages(reqJSON)
	}

	// === Detect if client wants thinking ===
	wantsThinking := false
	if thinking, ok := reqJSON["thinking"].(map[string]any); ok {
		if tp, _ := thinking["type"].(string); tp == "enabled" || tp == "adaptive" {
			wantsThinking = true
		}
	}
	if beta := r.Header.Get("anthropic-beta"); strings.Contains(strings.ToLower(beta), "thinking") {
		wantsThinking = true
	}

	// Remove thinking config from request (upstream may not support it)
	if wantsThinking {
		delete(reqJSON, "thinking")
	}

	// === #8 Identity injection (default off) ===
	identityTok := 0
	if h.cfg.Disguise.Enabled && h.cfg.Disguise.Identity {
		identityTok = injectIdentity(reqJSON, model)
	}

	// Estimate original input tokens (before identity injection) for identity_hide
	origInputTokens := estimateInputTokens(reqJSON)

	// === Vertex AI compatibility: remove top_p when temperature is set ===
	if reqJSON["temperature"] != nil && reqJSON["top_p"] != nil {
		delete(reqJSON, "top_p")
	}

	// Re-marshal cleaned request
	cleanBody, err := json.Marshal(reqJSON)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "api_error", "marshal error")
		return
	}

	h.logger.Info("forwarding request",
		"model", model,
		"stream", stream,
		"has_target", target != nil,
	)

	maxTokens := intFromAny(reqJSON["max_tokens"])

	if stream {
		h.handleStream(w, r, cleanBody, model, clientKey, target, wantsThinking, maxTokens, reqJSON, origInputTokens, identityTok)
	} else {
		h.handleSync(w, r, cleanBody, model, clientKey, target, wantsThinking, maxTokens, reqJSON, origInputTokens, identityTok)
	}
}

// handleSync handles non-streaming requests.
func (h *MessagesHandler) handleSync(w http.ResponseWriter, r *http.Request, body []byte,
	model, clientKey string, target *proxy.UpstreamTarget,
	wantsThinking bool, maxTokens int, reqJSON map[string]any,
	origInputTokens, identityTok int) {

	resp, err := h.proxy.SendMessages(r.Context(), body, false, target, r.Header)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "api_error", fmt.Sprintf("upstream error: %v", err))
		return
	}

	respBody, err := proxy.ReadResponse(resp)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "api_error", fmt.Sprintf("read response: %v", err))
		return
	}

	dc := h.disguiseCfg(target, model)

	// Handle error responses
	if resp.StatusCode >= 400 {
		var errJSON map[string]any
		if json.Unmarshal(respBody, &errJSON) == nil {
			fixed := fingerprint.FixError(errJSON, resp.StatusCode)
			respBody, _ = json.Marshal(fixed)
		}
		if dc.HeadersFake && !dc.PassthroughHeaders {
			fingerprint.ApplyHeaders(w, fingerprint.BuildResponseHeaders(model, clientKey, false, nil))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(respBody)
		return
	}

	// Parse successful response
	var respJSON map[string]any
	if err := json.Unmarshal(respBody, &respJSON); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(respBody)
		return
	}

	var outBody []byte

	if h.cfg.Disguise.Enabled && dc.BodyRewrite && !dc.PassthroughBody {
		if dc.StripContainer {
			delete(respJSON, "container")
		}

		// --- Collect all respJSON mutations before building OrderedMap ---

		// Max tokens clamp
		if dc.MaxTokensClamp && maxTokens > 0 {
			if usage, ok := respJSON["usage"].(map[string]any); ok {
				curOut := fingerprint.IntVal(usage, "output_tokens")
				if curOut > maxTokens {
					usage["output_tokens"] = float64(maxTokens)
					respJSON["stop_reason"] = "max_tokens"
				}
			}
		}

		// Small probe zero: max_tokens<=10 时 cache 字段全 0
		if dc.SmallProbeZero && maxTokens > 0 && maxTokens <= 10 {
			if usage, ok := respJSON["usage"].(map[string]any); ok {
				usage["cache_creation_input_tokens"] = float64(0)
				usage["cache_read_input_tokens"] = float64(0)
				if cc, ok := usage["cache_creation"].(map[string]any); ok {
					cc["ephemeral_5m_input_tokens"] = float64(0)
					cc["ephemeral_1h_input_tokens"] = float64(0)
				}
			}
		}

		// Cache fake: 上游 cache 全 0 且请求有 cache_control 时伪造首次创建信号
		if dc.CacheFake && maxTokens > 10 && hasCacheControl(reqJSON) {
			if usage, ok := respJSON["usage"].(map[string]any); ok {
				if isCacheAllZero(usage) {
					upKey := ""
					if target != nil {
						upKey = target.BaseURL
					}
					injectCacheUsageWithKey(usage, fingerprint.IntVal(usage, "input_tokens"), upKey, reqJSON)
				}
			}
		}

		// #11 Identity hide: revert input_tokens to pre-injection estimate
		if h.cfg.Disguise.IdentityHide && identityTok > 0 {
			if usage, ok := respJSON["usage"].(map[string]any); ok {
				realIn := fingerprint.IntVal(usage, "input_tokens")
				if realIn > origInputTokens+20 {
					usage["input_tokens"] = float64(origInputTokens)
				}
			}
		}

		// --- Build OrderedMap once with all mutations applied ---
		om := fingerprint.FixMessageBody(respJSON, dc)

		// Thinking injection (operates on OrderedMap, must be after FixMessageBody)
		if dc.ThinkingInject && wantsThinking {
			injectThinkingIntoOrderedMap(om, model, reqJSON)
		}

		outBody, _ = json.Marshal(om)
	} else {
		outBody, _ = json.Marshal(respJSON)
	}

	// #12 Capture
	if h.cfg.Disguise.CaptureEnabled {
		captureRequest(h.cfg.Disguise.CaptureDir, "nonstream", reqJSON, outBody, 200, r.Header)
	}

	// Headers
	if dc.PassthroughHeaders {
		// Forward upstream headers as-is (skip hop-by-hop)
		for k, vs := range stripHeadersForPassthrough(resp.Header) {
			for _, v := range vs {
				w.Header().Set(k, v)
			}
		}
	} else if dc.HeadersFake {
		inTok, outTok := 0, 0
		if usage, ok := respJSON["usage"].(map[string]any); ok {
			inTok = fingerprint.IntVal(usage, "input_tokens")
			outTok = fingerprint.IntVal(usage, "output_tokens")
		}
		rl := fingerprint.RateLimitTick(model, inTok, outTok)
		fingerprint.ApplyHeaders(w, fingerprint.BuildResponseHeaders(model, clientKey, false, rl))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(outBody)
}

// handleStream handles streaming requests.
func (h *MessagesHandler) handleStream(w http.ResponseWriter, r *http.Request, body []byte,
	model, clientKey string, target *proxy.UpstreamTarget,
	wantsThinking bool, maxTokens int, reqJSON map[string]any,
	origInputTokens, identityTok int) {

	resp, err := h.proxy.SendMessages(r.Context(), body, true, target, r.Header)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "api_error", fmt.Sprintf("upstream stream error: %v", err))
		return
	}
	defer resp.Body.Close()

	dc := h.disguiseCfg(target, model)

	// If upstream returned an error, forward it
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		var errJSON map[string]any
		if json.Unmarshal(respBody, &errJSON) == nil {
			fixed := fingerprint.FixError(errJSON, resp.StatusCode)
			respBody, _ = json.Marshal(fixed)
		}
		if dc.HeadersFake && !dc.PassthroughHeaders {
			fingerprint.ApplyHeaders(w, fingerprint.BuildResponseHeaders(model, clientKey, false, nil))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(respBody)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.logger.Error("streaming not supported by response writer")
		return
	}

	// Set response headers
	if dc.PassthroughHeaders {
		for k, vs := range stripHeadersForPassthrough(resp.Header) {
			for _, v := range vs {
				w.Header().Set(k, v)
			}
		}
	} else if dc.HeadersFake {
		rl := fingerprint.RateLimitTick(model, 500, 500)
		fingerprint.ApplyHeaders(w, fingerprint.BuildResponseHeaders(model, clientKey, true, rl))
	} else {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
	}
	w.Header().Set("X-Accel-Buffering", "no")

	upstreamKey := ""
	if target != nil {
		upstreamKey = target.BaseURL
	}
	ctx := &StreamContext{
		Model:           model,
		MaxTokens:       maxTokens,
		WantsThinking:   wantsThinking,
		Disguise:        dc,
		ReqJSON:         reqJSON,
		HasCacheControl: hasCacheControl(reqJSON),
		OrigInputTokens: origInputTokens,
		IdentityTok:     identityTok,
		IdentityHide:    h.cfg.Disguise.IdentityHide,
		UpstreamKey:     upstreamKey,
	}

	if dc.PassthroughBody {
		// Raw passthrough: no body rewrite, capped at 100MB
		io.Copy(w, io.LimitReader(resp.Body, 100*1024*1024))
	} else {
		normalizeSSEStream(w, resp.Body, flusher.Flush, ctx)
	}
}

// stripThinkingFromMessages removes thinking/redacted_thinking blocks from assistant messages.
func stripThinkingFromMessages(reqJSON map[string]any) {
	messages, ok := reqJSON["messages"].([]any)
	if !ok {
		return
	}
	for _, msg := range messages {
		m, ok := msg.(map[string]any)
		if !ok || m["role"] != "assistant" {
			continue
		}
		content, ok := m["content"].([]any)
		if !ok {
			continue
		}
		filtered := make([]any, 0, len(content))
		for _, block := range content {
			b, ok := block.(map[string]any)
			if !ok {
				filtered = append(filtered, block)
				continue
			}
			t, _ := b["type"].(string)
			if t != "thinking" && t != "redacted_thinking" {
				filtered = append(filtered, block)
			}
		}
		if len(filtered) != len(content) {
			m["content"] = filtered
		}
	}
}

// injectThinkingIntoOrderedMap adds a thinking block to an OrderedMap response if it has none.
func injectThinkingIntoOrderedMap(om *fingerprint.OrderedMap, model string, reqJSON map[string]any) {
	contentVal := om.Get("content")
	content, ok := contentVal.([]any)
	if !ok {
		return
	}
	// Check if already has thinking
	for _, cb := range content {
		if m, ok := cb.(*fingerprint.OrderedMap); ok {
			if m.GetString("type") == "thinking" {
				return
			}
		}
		if m, ok := cb.(map[string]any); ok {
			if t, _ := m["type"].(string); t == "thinking" {
				return
			}
		}
	}
	thinkTxt := pickThinkingTemplate()

	tb := fingerprint.NewOrderedMap()
	tb.Set("type", "thinking")
	tb.Set("thinking", thinkTxt)
	tb.Set("signature", fingerprint.FakeSignature(model, len(thinkTxt), thinkTxt))

	om.Set("content", append([]any{tb}, content...))
}


func extractClientKey(r *http.Request) string {
	if key := r.Header.Get("x-api-key"); key != "" {
		return strings.TrimSpace(key)
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	return ""
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	}
	return 0
}

func (h *MessagesHandler) resolveModel(model string) string {
	if h.cfg.Models.ModelMap != nil {
		if mapped, ok := h.cfg.Models.ModelMap[model]; ok {
			return mapped
		}
	}
	if model == "" && h.cfg.Models.DefaultModel != "" {
		return h.cfg.Models.DefaultModel
	}
	return model
}

func (h *MessagesHandler) dumpEntry(entry map[string]any) {
	dumpFile := h.cfg.Log.DumpFile
	if dumpFile == "" {
		h.logger.Warn("dump_requests enabled but dump_file not configured, skipping")
		return
	}
	line, err := json.Marshal(entry)
	if err != nil {
		h.logger.Warn("dump marshal error", "error", err)
		return
	}
	h.dumpMu.Lock()
	defer h.dumpMu.Unlock()
	f, err := os.OpenFile(dumpFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		h.logger.Warn("dump file open error", "error", err, "file", dumpFile)
		return
	}
	defer f.Close()
	if _, err := f.Write(line); err != nil {
		h.logger.Warn("dump write error", "error", err)
	}
	f.Write([]byte("\n"))
}

// serveRefusal returns a synthetic refusal response for system prompt leak attempts.
func (h *MessagesHandler) serveRefusal(w http.ResponseWriter, model, clientKey string, stream bool, reqJSON map[string]any) {
	dc := h.disguiseCfg(nil, model)
	msgID := fingerprint.NewMsgID()
	origIn := estimateInputTokens(reqJSON)
	outTok := len(refusalText) / 4
	if outTok < 50 {
		outTok = 50
	}

	if stream {
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeJSONError(w, http.StatusInternalServerError, "api_error", "streaming not supported")
			return
		}
		if dc.HeadersFake && !dc.PassthroughHeaders {
			rl := fingerprint.RateLimitTick(model, origIn, outTok)
			fingerprint.ApplyHeaders(w, fingerprint.BuildResponseHeaders(model, clientKey, true, rl))
		} else {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
		}
		w.Header().Set("X-Accel-Buffering", "no")

		flush := flusher.Flush

		// message_start
		ms := fmt.Sprintf(`{"type":"message_start","message":{"model":%q,"id":%q,"type":"message","role":"assistant","content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":%d,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"cache_creation":{"ephemeral_5m_input_tokens":0,"ephemeral_1h_input_tokens":0},"output_tokens":1,"service_tier":"standard","inference_geo":%q}}}`,
			model, msgID, origIn, fingerprint.GeoForModel(model))
		writeSSE(w, flush, "message_start", ms)

		// content_block_start
		writeSSE(w, flush, "content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)
		writeSSE(w, flush, "ping", `{"type": "ping"}`)

		// text_delta chunks
		for i := 0; i < len(refusalText); i += 40 {
			end := i + 40
			if end > len(refusalText) {
				end = len(refusalText)
			}
			chunk := refusalText[i:end]
			d, _ := json.Marshal(map[string]any{
				"type": "content_block_delta", "index": 0,
				"delta": map[string]any{"type": "text_delta", "text": chunk},
			})
			writeSSE(w, flush, "content_block_delta", string(d))
		}

		writeSSE(w, flush, "content_block_stop", `{"type":"content_block_stop","index":0}`)

		md := fmt.Sprintf(`{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null,"stop_details":{"type":"end_turn"}},"usage":{"input_tokens":%d,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":%d}}`,
			origIn, outTok)
		writeSSE(w, flush, "message_delta", md)
		writeSSE(w, flush, "message_stop", `{"type":"message_stop"}`)
	} else {
		body := fmt.Sprintf(`{"model":%q,"id":%q,"type":"message","role":"assistant","content":[{"type":"text","text":%s}],"stop_reason":"end_turn","stop_sequence":null,"stop_details":{"type":"end_turn"},"usage":{"input_tokens":%d,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"cache_creation":{"ephemeral_5m_input_tokens":0,"ephemeral_1h_input_tokens":0},"output_tokens":%d,"service_tier":"standard","inference_geo":%q}}`,
			model, msgID, mustJSON(refusalText), origIn, outTok, fingerprint.GeoForModel(model))

		if dc.HeadersFake && !dc.PassthroughHeaders {
			rl := fingerprint.RateLimitTick(model, origIn, outTok)
			fingerprint.ApplyHeaders(w, fingerprint.BuildResponseHeaders(model, clientKey, false, rl))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}
}

func mustJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// verifyRequestSignatures checks thinking.signature in assistant messages.
func verifyRequestSignatures(reqJSON map[string]any) string {
	messages, _ := reqJSON["messages"].([]any)
	for _, msg := range messages {
		m, ok := msg.(map[string]any)
		if !ok {
			continue
		}
		role, _ := m["role"].(string)
		if role != "assistant" {
			continue
		}
		content, ok := m["content"].([]any)
		if !ok {
			continue
		}
		for _, block := range content {
			b, ok := block.(map[string]any)
			if !ok {
				continue
			}
			t, _ := b["type"].(string)
			if t == "thinking" {
				sig, _ := b["signature"].(string)
				ok, reason := fingerprint.VerifySignature(sig)
				if !ok {
					return fmt.Sprintf("`thinking.signature` is invalid: %s", reason)
				}
			}
		}
	}
	return ""
}

// hasCacheControl checks if the request has any cache_control markers.
func hasCacheControl(reqJSON map[string]any) bool {
	messages, _ := reqJSON["messages"].([]any)
	for _, msg := range messages {
		m, ok := msg.(map[string]any)
		if !ok {
			continue
		}
		content, ok := m["content"].([]any)
		if !ok {
			continue
		}
		for _, block := range content {
			if b, ok := block.(map[string]any); ok {
				if _, has := b["cache_control"]; has {
					return true
				}
			}
		}
	}
	if sysp, ok := reqJSON["system"].([]any); ok {
		for _, block := range sysp {
			if b, ok := block.(map[string]any); ok {
				if _, has := b["cache_control"]; has {
					return true
				}
			}
		}
	}
	if tools, ok := reqJSON["tools"].([]any); ok {
		for _, t := range tools {
			if b, ok := t.(map[string]any); ok {
				if _, has := b["cache_control"]; has {
					return true
				}
			}
		}
	}
	return false
}

// isCacheAllZero returns true if all cache-related usage fields are 0.
func isCacheAllZero(usage map[string]any) bool {
	if fingerprint.IntVal(usage, "cache_creation_input_tokens") != 0 {
		return false
	}
	if fingerprint.IntVal(usage, "cache_read_input_tokens") != 0 {
		return false
	}
	if cc, ok := usage["cache_creation"].(map[string]any); ok {
		if fingerprint.IntVal(cc, "ephemeral_5m_input_tokens") != 0 {
			return false
		}
		if fingerprint.IntVal(cc, "ephemeral_1h_input_tokens") != 0 {
			return false
		}
	}
	return true
}

// cacheSessionState tracks per-upstream cache hit patterns for realistic simulation.
var (
	cacheCounters  sync.Map // map[string]*cacheEntry
	cacheCleanOnce sync.Once
)

type cacheEntry struct {
	mu       sync.Mutex
	count    int
	lastSeen time.Time
}

const cacheEntryTTL = 30 * time.Minute

func initCacheCleaner() {
	cacheCleanOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				cutoff := time.Now().Add(-cacheEntryTTL)
				cacheCounters.Range(func(key, value any) bool {
					entry := value.(*cacheEntry)
					entry.mu.Lock()
					expired := entry.lastSeen.Before(cutoff)
					entry.mu.Unlock()
					if expired {
						cacheCounters.Delete(key)
					}
					return true
				})
			}
		}()
	})
}

// getCacheCallCount returns and increments the cache call count for an upstream.
func getCacheCallCount(upstreamKey string) int {
	initCacheCleaner()
	newEntry := &cacheEntry{}
	actual, _ := cacheCounters.LoadOrStore(upstreamKey, newEntry)
	entry := actual.(*cacheEntry)
	entry.mu.Lock()
	entry.count++
	entry.lastSeen = time.Now()
	count := entry.count
	entry.mu.Unlock()
	return count
}

// contentCacheKey computes a deterministic hash from the request's system prompt
// and tools definition, so that identical content always maps to the same cache
// counter — mimicking the content-addressable behaviour of the real API.
func contentCacheKey(reqJSON map[string]any) string {
	h := sha256.New()
	// Hash system prompt
	switch s := reqJSON["system"].(type) {
	case string:
		h.Write([]byte(s))
	case []any:
		for _, block := range s {
			if b, ok := block.(map[string]any); ok {
				if text, _ := b["text"].(string); text != "" {
					h.Write([]byte(text))
				}
			}
		}
	}
	// Hash tools
	if tools, ok := reqJSON["tools"].([]any); ok {
		tb, _ := json.Marshal(tools)
		h.Write(tb)
	}
	return fmt.Sprintf("%x", h.Sum(nil)[:8])
}


// injectCacheUsageWithKey is the keyed version for per-upstream tracking.
// When reqJSON is provided, the cache counter key is derived from content hash
// combined with upstreamKey, so identical content shares cache hit progression.
func injectCacheUsageWithKey(usage map[string]any, inputTokens int, upstreamKey string, reqJSON map[string]any) {
	counterKey := upstreamKey
	if reqJSON != nil {
		counterKey = contentCacheKey(reqJSON) + "|" + upstreamKey
	}
	callNum := getCacheCallCount(counterKey)

	est := inputTokens
	if est < 500 {
		est = 500
	}

	// Dynamic ratios: early calls = more creation, later = more reads
	var createRatio, readRatio float64
	switch {
	case callNum <= 2:
		createRatio = 0.45 + randJitter(0.10)
		readRatio = 0.05 + randJitter(0.03)
	case callNum <= 10:
		createRatio = 0.10 + randJitter(0.05)
		readRatio = 0.40 + randJitter(0.10)
	default:
		createRatio = 0.02 + randJitter(0.02)
		readRatio = 0.55 + randJitter(0.15)
	}

	createTok := int(float64(est) * createRatio)
	readTok := int(float64(est) * readRatio)

	usage["cache_creation_input_tokens"] = float64(createTok)
	usage["cache_read_input_tokens"] = float64(readTok)
	if cc, ok := usage["cache_creation"].(map[string]any); ok {
		cc["ephemeral_5m_input_tokens"] = float64(createTok)
		cc["ephemeral_1h_input_tokens"] = float64(0)
	}
}

func randJitter(amplitude float64) float64 {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000))
	return amplitude * (float64(n.Int64())/500.0 - 1.0) // range: [-amplitude, +amplitude]
}

func writeJSONError(w http.ResponseWriter, status int, errType, message string) {
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
