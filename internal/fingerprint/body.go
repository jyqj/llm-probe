package fingerprint

import (
	"encoding/json"
	"strings"

	"bedrock-gateway/internal/config"
)

// FixMessageBody rewrites a non-streaming response body to match official field order and format.
// Returns *OrderedMap which implements json.Marshaler for ordered output.
func FixMessageBody(body map[string]any, cfg *config.DisguiseConfig) *OrderedMap {
	model, _ := body["model"].(string)

	content, _ := body["content"].([]any)
	if content == nil {
		content = []any{}
	}

	fixedContent := make([]any, 0, len(content))
	for _, cb := range content {
		if m, ok := cb.(map[string]any); ok {
			fixedContent = append(fixedContent, FixContentBlock(m, model, cfg))
		} else {
			fixedContent = append(fixedContent, cb)
		}
	}

	usage, _ := body["usage"].(map[string]any)

	id := strVal(body, "id")
	if cfg.IDRewrite {
		id = RewriteID(id, false, false)
	}

	// Preserve null vs string for stop_reason/stop_sequence
	var stopReason any = body["stop_reason"]
	stopReasonStr, _ := body["stop_reason"].(string)
	stopSeq := body["stop_sequence"]

	out := NewOrderedMap()
	out.Set("model", model)
	out.Set("id", id)
	out.Set("type", strOrDefault(body, "type", "message"))
	out.Set("role", strOrDefault(body, "role", "assistant"))
	out.Set("content", fixedContent)
	out.Set("stop_reason", stopReason)
	out.Set("stop_sequence", stopSeq)
	out.Set("stop_details", buildStopDetails(body["stop_details"], stopReasonStr))
	out.Set("usage", FixUsageFull(usage, model, cfg))
	return out
}

// FixContentBlock rewrites a content block to official format.
// Returns *OrderedMap for ordered JSON output.
func FixContentBlock(cb map[string]any, model string, cfg *config.DisguiseConfig) *OrderedMap {
	t, _ := cb["type"].(string)
	switch t {
	case "text":
		out := NewOrderedMap()
		out.Set("type", "text")
		out.Set("text", cb["text"])
		if cit, ok := cb["citations"]; ok && cit != nil {
			if cfg.SignatureRewrite {
				cit = fixCitations(cit)
			}
			out.Set("citations", cit)
		}
		return out

	case "thinking":
		thinkTxt, _ := cb["thinking"].(string)
		sig, _ := cb["signature"].(string)
		if cfg.SignatureRewrite {
			sig = FakeSignature(model, len(thinkTxt), thinkTxt)
		}
		out := NewOrderedMap()
		out.Set("type", "thinking")
		out.Set("thinking", thinkTxt)
		out.Set("signature", sig)
		return out

	case "redacted_thinking":
		out := NewOrderedMap()
		out.Set("type", "redacted_thinking")
		data, _ := cb["data"].(string)
		if data == "" {
			data = FakeSignature(model, 0, "")
		}
		out.Set("data", data)
		return out

	case "tool_use":
		id := strVal(cb, "id")
		if cfg.IDRewrite {
			id = RewriteID(id, true, false)
		}
		out := NewOrderedMap()
		out.Set("type", "tool_use")
		out.Set("id", id)
		out.Set("name", cb["name"])
		out.Set("input", cleanToolInput(cb["input"]))
		caller := cb["caller"]
		if caller == nil {
			caller = map[string]any{"type": "direct"}
		}
		out.Set("caller", caller)
		return out

	case "tool_result":
		tid := strVal(cb, "tool_use_id")
		if cfg.IDRewrite {
			tid = RewriteID(tid, true, false)
		}
		out := NewOrderedMap()
		out.Set("type", "tool_result")
		out.Set("tool_use_id", tid)
		if c, ok := cb["content"]; ok {
			out.Set("content", c)
		}
		if e, ok := cb["is_error"]; ok {
			out.Set("is_error", e)
		}
		return out

	case "server_tool_use":
		id := strVal(cb, "id")
		if cfg.IDRewrite {
			id = RewriteID(id, false, true)
		}
		out := NewOrderedMap()
		out.Set("type", "server_tool_use")
		out.Set("id", id)
		out.Set("name", cb["name"])
		out.Set("input", cleanToolInput(cb["input"]))
		return out

	case "web_search_tool_result":
		tid := strVal(cb, "tool_use_id")
		if cfg.IDRewrite {
			tid = RewriteID(tid, false, true)
		}
		out := NewOrderedMap()
		out.Set("type", "web_search_tool_result")
		out.Set("tool_use_id", tid)
		content := cb["content"]
		if cfg.SignatureRewrite {
			content = fixWebSearchResultItems(content)
		}
		out.Set("content", content)
		return out
	}
	// Unknown type: wrap as-is preserving order
	out := NewOrderedMap()
	for k, v := range cb {
		out.Set(k, v)
	}
	return out
}

// FixUsageFull normalizes usage for non-streaming (message_start) responses.
func FixUsageFull(usage map[string]any, model string, cfg *config.DisguiseConfig) *OrderedMap {
	if usage == nil {
		usage = map[string]any{}
	}
	u := copyMap(usage)
	if cfg.StripBedrock {
		delete(u, "bedrock_state")
	}
	setDefault(u, "input_tokens", float64(0))
	setDefault(u, "cache_creation_input_tokens", float64(0))
	setDefault(u, "cache_read_input_tokens", float64(0))

	// Ensure cache_creation nested object
	hasNestedCC := false
	var ccMap map[string]any
	if cc, ok := u["cache_creation"].(map[string]any); ok {
		hasNestedCC = true
		ccMap = copyMap(cc)
		setDefault(ccMap, "ephemeral_5m_input_tokens", float64(0))
		setDefault(ccMap, "ephemeral_1h_input_tokens", float64(0))
	}
	if !hasNestedCC {
		totalCC := IntVal(u, "cache_creation_input_tokens")
		ccMap = map[string]any{
			"ephemeral_5m_input_tokens": float64(0),
			"ephemeral_1h_input_tokens": float64(0),
		}
		if totalCC > 0 {
			ccMap["ephemeral_5m_input_tokens"] = float64(totalCC)
		}
	}

	setDefault(u, "output_tokens", float64(0))

	geo := GeoForModel(model)
	if !cfg.ForceGeo {
		if existing, ok := u["inference_geo"].(string); ok && existing != "" {
			geo = existing
		}
	}

	// Build ordered cache_creation
	ccOrdered := NewOrderedMap()
	ccOrdered.Set("ephemeral_5m_input_tokens", ccMap["ephemeral_5m_input_tokens"])
	ccOrdered.Set("ephemeral_1h_input_tokens", ccMap["ephemeral_1h_input_tokens"])

	out := NewOrderedMap()
	out.Set("input_tokens", u["input_tokens"])
	out.Set("cache_creation_input_tokens", u["cache_creation_input_tokens"])
	out.Set("cache_read_input_tokens", u["cache_read_input_tokens"])
	out.Set("cache_creation", ccOrdered)
	out.Set("output_tokens", u["output_tokens"])
	out.Set("service_tier", "standard")
	out.Set("inference_geo", geo)
	// Append remaining keys not in standard order
	for k, v := range u {
		if !out.Has(k) && k != "bedrock_state" && k != "cache_creation" {
			out.Set(k, v)
		}
	}
	return out
}

// FixUsageSlim normalizes usage for message_delta events.
func FixUsageSlim(usage map[string]any, cfg *config.DisguiseConfig) *OrderedMap {
	if usage == nil {
		usage = map[string]any{}
	}
	u := copyMap(usage)
	if cfg.StripBedrock {
		delete(u, "bedrock_state")
	}
	setDefault(u, "input_tokens", float64(0))
	setDefault(u, "cache_creation_input_tokens", float64(0))
	setDefault(u, "cache_read_input_tokens", float64(0))
	setDefault(u, "output_tokens", float64(0))

	out := NewOrderedMap()
	for _, k := range []string{"input_tokens", "cache_creation_input_tokens", "cache_read_input_tokens", "output_tokens"} {
		out.Set(k, u[k])
	}
	return out
}

// GeoForModel returns the expected inference_geo value.
func GeoForModel(model string) string {
	if containsCI(model, "haiku") {
		return "not_available"
	}
	return "global"
}

// FixError converts upstream error body to Anthropic standard format.
func FixError(body map[string]any, status int) map[string]any {
	var msg, etype string

	if code, ok := body["code"].(string); ok && code != "" {
		msg, _ = body["message"].(string)
		switch code {
		case "INVALID_API_KEY":
			etype = "authentication_error"
		default:
			etype = "invalid_request_error"
		}
	} else if errObj, ok := body["error"].(map[string]any); ok {
		msg, _ = errObj["message"].(string)
		t, _ := errObj["type"].(string)
		switch t {
		case "", "<nil>":
			etype = "api_error"
		case "api_error":
			etype = "overloaded_error"
		case "new_api_error":
			etype = "invalid_request_error"
		default:
			etype = t
		}
	} else {
		b, _ := json.Marshal(body)
		msg = string(b)
		etype = "api_error"
	}

	switch {
	case status == 401:
		etype = "authentication_error"
	case status == 403:
		etype = "permission_error"
	case status == 404:
		etype = "not_found_error"
	case status == 429:
		etype = "rate_limit_error"
	case status >= 500:
		etype = "api_error"
	}

	return map[string]any{
		"type": "error",
		"error": map[string]any{
			"type":    etype,
			"message": msg,
		},
	}
}

// buildStopDetails ensures stop_details is a properly structured object.
// Official format: {"type": "end_turn"|"stop_sequence"|"max_tokens"|"tool_use"|"refusal"}
// If stop_details is already correct, pass through; otherwise synthesize from stop_reason.
func buildStopDetails(existing any, stopReason string) any {
	if existing != nil {
		if sd, ok := existing.(map[string]any); ok {
			if t, _ := sd["type"].(string); t != "" {
				return sd
			}
		}
	}
	// Synthesize from stop_reason if non-empty
	if stopReason != "" {
		return map[string]any{"type": stopReason}
	}
	return nil
}

// === helpers ===

func strVal(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func strOrDefault(m map[string]any, key, def string) string {
	v, _ := m[key].(string)
	if v == "" {
		return def
	}
	return v
}

func IntVal(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	}
	return 0
}

func setDefault(m map[string]any, key string, def any) {
	if _, ok := m[key]; !ok {
		m[key] = def
	}
}

func copyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// fixWebSearchResultItems rewrites encrypted_content in web_search_result items.
func fixWebSearchResultItems(content any) any {
	items, ok := content.([]any)
	if !ok {
		return content
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			out = append(out, item)
			continue
		}
		t, _ := m["type"].(string)
		if t == "web_search_result" {
			fixed := NewOrderedMap()
			fixed.Set("type", "web_search_result")
			fixed.Set("title", m["title"])
			fixed.Set("url", m["url"])
			fixed.Set("encrypted_content", FakeEncryptedContent())
			if pa, ok := m["page_age"]; ok {
				fixed.Set("page_age", pa)
			}
			out = append(out, fixed)
		} else {
			out = append(out, item)
		}
	}
	return out
}

// fixCitations rewrites encrypted_index in citation objects.
func fixCitations(cits any) any {
	items, ok := cits.([]any)
	if !ok {
		return cits
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			out = append(out, item)
			continue
		}
		fixed := NewOrderedMap()
		fixed.Set("type", strOrDefault(m, "type", "web_search_result_location"))
		fixed.Set("cited_text", m["cited_text"])
		fixed.Set("url", m["url"])
		fixed.Set("title", m["title"])
		fixed.Set("encrypted_index", FakeEncryptedIndex())
		out = append(out, fixed)
	}
	return out
}

func cleanToolInput(inp any) any {
	m, ok := inp.(map[string]any)
	if !ok {
		return inp
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		if !strings.HasPrefix(k, "_") {
			out[k] = v
		}
	}
	return out
}

// === OrderedMap: preserves insertion order for JSON serialization ===

// OrderedMap preserves insertion order when marshaled to JSON.
type OrderedMap struct {
	keys   []string
	values map[string]any
}

func NewOrderedMap() *OrderedMap {
	return &OrderedMap{values: make(map[string]any)}
}

func (o *OrderedMap) Set(key string, val any) {
	if _, exists := o.values[key]; !exists {
		o.keys = append(o.keys, key)
	}
	o.values[key] = val
}

func (o *OrderedMap) Has(key string) bool {
	_, ok := o.values[key]
	return ok
}

func (o *OrderedMap) Get(key string) any {
	return o.values[key]
}

func (o *OrderedMap) GetInt(key string) int {
	return IntVal(o.values, key)
}

func (o *OrderedMap) GetString(key string) string {
	v, _ := o.values[key].(string)
	return v
}

func (o *OrderedMap) MarshalJSON() ([]byte, error) {
	var buf strings.Builder
	buf.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		kb, _ := json.Marshal(k)
		buf.Write(kb)
		buf.WriteByte(':')
		vb, err := json.Marshal(o.values[k])
		if err != nil {
			return nil, err
		}
		buf.Write(vb)
	}
	buf.WriteByte('}')
	return []byte(buf.String()), nil
}
