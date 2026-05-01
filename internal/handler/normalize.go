package handler

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/json"
	"io"
	"math/big"
	"strings"
	"sync"

	"bedrock-gateway/internal/config"
	"bedrock-gateway/internal/fingerprint"
)

var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// StreamContext holds state across SSE events in a stream.
type StreamContext struct {
	Model           string
	MaxTokens       int
	WantsThinking   bool
	Disguise        *config.DisguiseConfig
	ReqJSON         map[string]any
	HasCacheControl bool
	OrigInputTokens int
	IdentityTok     int
	IdentityHide    bool
	UpstreamKey     string // for per-upstream cache tracking

	// Internal state
	idMap            map[string]string
	pingInjected     bool
	thinkingInjected bool
	thinkingBuf      strings.Builder // accumulates upstream thinking text for signature binding
}

func (ctx *StreamContext) remap(old string, tool, serverTool bool) string {
	if ctx.idMap == nil {
		ctx.idMap = make(map[string]string)
	}
	if old == "" {
		return fingerprint.RewriteID("", tool, serverTool)
	}
	if mapped, ok := ctx.idMap[old]; ok {
		return mapped
	}
	newID := fingerprint.RewriteID(old, tool, serverTool)
	ctx.idMap[old] = newID
	return newID
}

// normalizeSSEStream reads upstream SSE, normalizes each event, and writes to client.
func normalizeSSEStream(w io.Writer, r io.Reader, flush func(), ctx *StreamContext) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	var eventName string
	var dataLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// End of SSE event - process accumulated data
			if len(dataLines) > 0 {
				payload := strings.Join(dataLines, "\n")
				if eventName == "" {
					eventName = "message"
				}
				processSSEEvent(w, flush, eventName, payload, ctx)
			}
			eventName = ""
			dataLines = nil
			continue
		}

		if strings.HasPrefix(line, ":") {
			continue // SSE comment
		}
		if strings.HasPrefix(line, "event:") {
			eventName = strings.TrimSpace(line[6:])
		} else if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(line[5:]))
		} else {
			// Unknown line, pass through
			io.WriteString(w, line+"\n")
			flush()
		}
	}
}

func processSSEEvent(w io.Writer, flush func(), event, payload string, ctx *StreamContext) {
	dc := ctx.Disguise

	// Strip [DONE] sentinel
	if strings.TrimSpace(payload) == "[DONE]" {
		if !dc.StripDone {
			writeSSE(w, flush, event, payload)
		}
		return
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		writeSSE(w, flush, event, payload)
		return
	}

	eventType, _ := data["type"].(string)

	switch eventType {
	case "message_start":
		msg, _ := data["message"].(map[string]any)
		if msg == nil {
			writeSSE(w, flush, event, payload)
			return
		}

		if dc.BodyRewrite {
			oldID, _ := msg["id"].(string)
			fixed := fingerprint.FixMessageBody(msg, dc)
			if oldID != "" {
				newID := fixed.GetString("id")
				if newID != "" {
					ctx.remap(oldID, false, false)
					ctx.idMap[oldID] = newID
				}
			}
			ctx.Model = fixed.GetString("model")
			wrapper := fingerprint.NewOrderedMap()
			wrapper.Set("type", "message_start")
			wrapper.Set("message", fixed)
			out, _ := json.Marshal(wrapper)
			// message_start never has padding (matches official behavior)
			writeSSE(w, flush, event, string(out))
		} else {
			ensureField(msg, "stop_details", nil)
			ensureField(msg, "stop_sequence", nil)
			if usage, ok := msg["usage"].(map[string]any); ok {
				ensureUsageFull(usage)
			}
			out, _ := json.Marshal(data)
			writeSSE(w, flush, event, string(out))
		}

	case "content_block_start":
		if dc.ThinkingInject && ctx.WantsThinking && !ctx.thinkingInjected {
			ctx.thinkingInjected = true
			injectThinkingSSE(w, flush, ctx)
			if idx, ok := data["index"].(float64); ok {
				data["index"] = idx + 1
			}
		}

		var contentBlock any = data["content_block"]
		if dc.BodyRewrite {
			if cb, ok := data["content_block"].(map[string]any); ok {
				if t, _ := cb["type"].(string); t == "tool_use" {
					if id, _ := cb["id"].(string); id != "" {
						cb["id"] = ctx.remap(id, true, false)
					}
				} else if t == "server_tool_use" {
					if id, _ := cb["id"].(string); id != "" {
						cb["id"] = ctx.remap(id, false, true)
					}
				}
				contentBlock = fingerprint.FixContentBlock(cb, ctx.Model, dc)
			}
		}

		outOM := fingerprint.NewOrderedMap()
		outOM.Set("type", "content_block_start")
		outOM.Set("index", data["index"])
		outOM.Set("content_block", contentBlock)
		out, _ := json.Marshal(outOM)
		writeSSEPadded(w, flush, event, string(out), dc.SSEPadding)

		if !ctx.pingInjected {
			ctx.pingInjected = true
			writeSSE(w, flush, "ping", `{"type": "ping"}`)
		}

	case "content_block_delta":
		if dc.ThinkingInject && ctx.WantsThinking && ctx.thinkingInjected {
			if idx, ok := data["index"].(float64); ok {
				data["index"] = idx + 1
			}
		}

		delta, _ := data["delta"].(map[string]any)
		if dc.BodyRewrite && delta != nil {
			dt, _ := delta["type"].(string)
			if dt == "thinking_delta" {
				if chunk, _ := delta["thinking"].(string); chunk != "" {
					ctx.thinkingBuf.WriteString(chunk)
				}
			}
			if dt == "signature_delta" && dc.SignatureRewrite {
				accumulated := ctx.thinkingBuf.String()
				delta["signature"] = fingerprint.FakeSignature(ctx.Model, len(accumulated), accumulated)
				ctx.thinkingBuf.Reset()
			}
		}

		// Build ordered delta based on type
		var orderedDelta any = delta
		if delta != nil {
			dt, _ := delta["type"].(string)
			od := fingerprint.NewOrderedMap()
			od.Set("type", dt)
			switch dt {
			case "text_delta":
				od.Set("text", delta["text"])
			case "thinking_delta":
				od.Set("thinking", delta["thinking"])
			case "signature_delta":
				od.Set("signature", delta["signature"])
			case "input_json_delta":
				od.Set("partial_json", delta["partial_json"])
			default:
				orderedDelta = delta // fallback: use raw map
				od = nil
			}
			if od != nil {
				orderedDelta = od
			}
		}

		outOM := fingerprint.NewOrderedMap()
		outOM.Set("type", "content_block_delta")
		outOM.Set("index", data["index"])
		outOM.Set("delta", orderedDelta)
		out, _ := json.Marshal(outOM)
		writeSSEPadded(w, flush, event, string(out), dc.SSEPadding)

	case "content_block_stop":
		if dc.ThinkingInject && ctx.WantsThinking && ctx.thinkingInjected {
			if idx, ok := data["index"].(float64); ok {
				data["index"] = idx + 1
			}
		}
		outOM := fingerprint.NewOrderedMap()
		outOM.Set("type", "content_block_stop")
		outOM.Set("index", data["index"])
		out, _ := json.Marshal(outOM)
		writeSSEPadded(w, flush, event, string(out), dc.SSEPadding)

	case "message_delta":
		if dc.StripBedrock {
			delete(data, "bedrock_state")
		}
		delta, _ := data["delta"].(map[string]any)
		if delta == nil {
			delta = map[string]any{}
		}

		// Build ordered delta: stop_reason, stop_sequence, stop_details
		newDelta := fingerprint.NewOrderedMap()
		sr, _ := delta["stop_reason"].(string)
		if sr != "" {
			newDelta.Set("stop_reason", sr)
		} else if v, ok := delta["stop_reason"]; ok {
			newDelta.Set("stop_reason", v)
		}
		if ss, ok := delta["stop_sequence"]; ok {
			newDelta.Set("stop_sequence", ss)
		}
		// Ensure stop_details has correct structure {type: <stop_reason>}
		sd := delta["stop_details"]
		if sd == nil && sr != "" {
			sd = map[string]any{"type": sr}
		} else if sdm, ok := sd.(map[string]any); ok {
			if _, hasType := sdm["type"]; !hasType && sr != "" {
				sdm["type"] = sr
			}
		}
		newDelta.Set("stop_details", sd)

		// Process usage
		usage, _ := data["usage"].(map[string]any)
		fixedUsage := fingerprint.FixUsageSlim(usage, dc)

		// Max tokens clamp
		if dc.MaxTokensClamp && ctx.MaxTokens > 0 {
			curOut := fixedUsage.GetInt("output_tokens")
			if curOut > ctx.MaxTokens {
				fixedUsage.Set("output_tokens", float64(ctx.MaxTokens))
				newDelta.Set("stop_reason", "max_tokens")
			}
		}
		// Small probe zero
		if dc.SmallProbeZero && ctx.MaxTokens > 0 && ctx.MaxTokens <= 10 {
			fixedUsage.Set("cache_creation_input_tokens", float64(0))
			fixedUsage.Set("cache_read_input_tokens", float64(0))
		}
		// Streaming CACHE_FAKE: use per-upstream dynamic cache injection
		if dc.CacheFake && ctx.HasCacheControl && ctx.MaxTokens > 10 {
			ccTok := fixedUsage.GetInt("cache_creation_input_tokens")
			crTok := fixedUsage.GetInt("cache_read_input_tokens")
			if ccTok == 0 && crTok == 0 {
				inTok := fixedUsage.GetInt("input_tokens")
				tmpUsage := map[string]any{
					"input_tokens":                float64(inTok),
					"cache_creation_input_tokens": float64(0),
					"cache_read_input_tokens":     float64(0),
				}
				injectCacheUsageWithKey(tmpUsage, inTok, ctx.UpstreamKey, ctx.ReqJSON)
				fixedUsage.Set("cache_creation_input_tokens", tmpUsage["cache_creation_input_tokens"])
				fixedUsage.Set("cache_read_input_tokens", tmpUsage["cache_read_input_tokens"])
			}
		}
		// Identity hide
		if ctx.IdentityHide && ctx.IdentityTok > 0 {
			realIn := fixedUsage.GetInt("input_tokens")
			if realIn > ctx.OrigInputTokens+20 {
				fixedUsage.Set("input_tokens", float64(ctx.OrigInputTokens))
			}
		}

		// Build ordered output: type, delta, usage
		outOM := fingerprint.NewOrderedMap()
		outOM.Set("type", "message_delta")
		outOM.Set("delta", newDelta)
		outOM.Set("usage", fixedUsage)
		out, _ := json.Marshal(outOM)
		writeSSEPadded(w, flush, event, string(out), dc.SSEPadding)

	case "message_stop":
		outOM := fingerprint.NewOrderedMap()
		outOM.Set("type", "message_stop")
		out, _ := json.Marshal(outOM)
		writeSSEPadded(w, flush, event, string(out), dc.SSEPadding)

	case "ping":
		writeSSE(w, flush, event, `{"type": "ping"}`)
		ctx.pingInjected = true

	default:
		writeSSE(w, flush, event, payload)
	}
}

// injectThinkingSSE injects a synthetic thinking block into the SSE stream.
// genericThinkingTemplates contains neutral thinking texts that do not leak any user input.
var genericThinkingTemplates = []string{
	"Let me analyze this request carefully and think through the best approach to provide a helpful response.",
	"I need to consider the key aspects of this question and formulate a thorough, well-structured response.",
	"Let me break down what's being asked and think step by step about how to address this effectively.",
	"I should consider multiple angles here and ensure my response is comprehensive and accurately addresses the request.",
	"Let me think about this systematically to provide the most helpful and precise answer I can.",
	"I want to make sure I understand the full scope of this request before composing my response carefully.",
	"Let me reason through this thoroughly, considering both the explicit and implicit parts of the question.",
	"I need to organize my thoughts here and determine the clearest way to present my response.",
}

func pickThinkingTemplate() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(genericThinkingTemplates))))
	return genericThinkingTemplates[n.Int64()]
}

func injectThinkingSSE(w io.Writer, flush func(), ctx *StreamContext) {
	thinkTxt := pickThinkingTemplate()
	sig := fingerprint.FakeSignature(ctx.Model, len(thinkTxt), thinkTxt)

	// 1. content_block_start (thinking, index=0)
	cbStart := map[string]any{
		"type": "content_block_start", "index": 0,
		"content_block": map[string]any{"type": "thinking", "thinking": "", "signature": ""},
	}
	b, _ := json.Marshal(cbStart)
	writeSSE(w, flush, "content_block_start", string(b))

	// 2. ping
	writeSSE(w, flush, "ping", `{"type": "ping"}`)
	ctx.pingInjected = true

	// 3. thinking_delta chunks
	for i := 0; i < len(thinkTxt); i += 40 {
		end := i + 40
		if end > len(thinkTxt) {
			end = len(thinkTxt)
		}
		chunk := thinkTxt[i:end]
		d := map[string]any{
			"type": "content_block_delta", "index": 0,
			"delta": map[string]any{"type": "thinking_delta", "thinking": chunk},
		}
		b, _ := json.Marshal(d)
		writeSSE(w, flush, "content_block_delta", string(b))
	}

	// 4. signature_delta
	sigD := map[string]any{
		"type": "content_block_delta", "index": 0,
		"delta": map[string]any{"type": "signature_delta", "signature": sig},
	}
	b, _ = json.Marshal(sigD)
	writeSSE(w, flush, "content_block_delta", string(b))

	// 5. content_block_stop
	stopD := map[string]any{"type": "content_block_stop", "index": 0}
	b, _ = json.Marshal(stopD)
	writeSSE(w, flush, "content_block_stop", string(b))
}

func writeSSE(w io.Writer, flush func(), event, data string) {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.WriteString("event: ")
	buf.WriteString(event)
	buf.WriteString("\ndata: ")
	buf.WriteString(data)
	buf.WriteString("\n\n\n")
	w.Write(buf.Bytes())
	bufPool.Put(buf)
	flush()
}

// writeSSEPadded writes SSE with optional random padding before closing }.
func writeSSEPadded(w io.Writer, flush func(), event, data string, pad bool) {
	if pad && event != "ping" && event != "message_start" && strings.HasSuffix(data, "}") {
		n, _ := rand.Int(rand.Reader, big.NewInt(13))
		spaces := int(n.Int64()) + 4 // 4~16 spaces

		buf := bufPool.Get().(*bytes.Buffer)
		buf.Reset()
		buf.WriteString("event: ")
		buf.WriteString(event)
		buf.WriteString("\ndata: ")
		buf.WriteString(data[:len(data)-1])
		for i := 0; i < spaces; i++ {
			buf.WriteByte(' ')
		}
		buf.WriteString("}\n\n\n")
		w.Write(buf.Bytes())
		bufPool.Put(buf)
		flush()
		return
	}
	writeSSE(w, flush, event, data)
}

func ensureField(m map[string]any, key string, def any) {
	if _, ok := m[key]; !ok {
		m[key] = def
	}
}

// ensureUsageFull ensures all usage fields exist with defaults.
func ensureUsageFull(usage map[string]any) {
	defaults := map[string]any{
		"input_tokens":                float64(0),
		"output_tokens":               float64(0),
		"cache_creation_input_tokens": float64(0),
		"cache_read_input_tokens":     float64(0),
		"service_tier":                "standard",
		"inference_geo":               "not_available",
	}
	for k, v := range defaults {
		if _, ok := usage[k]; !ok {
			usage[k] = v
		}
	}
	if _, ok := usage["cache_creation"]; !ok {
		usage["cache_creation"] = map[string]any{
			"ephemeral_5m_input_tokens": float64(0),
			"ephemeral_1h_input_tokens": float64(0),
		}
	}
}

