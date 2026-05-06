package probe

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
)

func readSSE(r io.Reader) (string, map[string]any, map[string]any) {
	var raw strings.Builder
	var msgStart, msgDelta map[string]any

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 1<<20), 10<<20)
	for sc.Scan() {
		line := sc.Text()
		raw.WriteString(line)
		raw.WriteByte('\n')
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		d := line[6:]
		if d == "[DONE]" {
			continue
		}
		var ev map[string]any
		if json.Unmarshal([]byte(d), &ev) != nil {
			continue
		}
		switch ev["type"] {
		case "message_start":
			if m, ok := ev["message"].(map[string]any); ok {
				msgStart = m
			}
		case "message_delta":
			if dd, ok := ev["delta"].(map[string]any); ok {
				msgDelta = dd
			}
			if u, ok := ev["usage"].(map[string]any); ok {
				if msgDelta == nil {
					msgDelta = make(map[string]any)
				}
				msgDelta["usage"] = u
			}
		}
	}
	return raw.String(), msgStart, msgDelta
}

func merge(start, delta map[string]any, sse string) map[string]any {
	if start == nil {
		return nil
	}
	f := make(map[string]any)
	for k, v := range start {
		f[k] = v
	}
	if blocks := extractBlocks(sse); len(blocks) > 0 {
		f["content"] = blocks
	}
	if delta != nil {
		for _, k := range []string{"stop_reason", "stop_sequence", "stop_details"} {
			if v, ok := delta[k]; ok {
				f[k] = v
			}
		}
		if du, ok := delta["usage"].(map[string]any); ok {
			if bu, ok := f["usage"].(map[string]any); ok {
				for k, v := range du {
					bu[k] = v
				}
			} else {
				f["usage"] = du
			}
		}
	}
	return f
}

func extractBlocks(sse string) []any {
	var blocks []map[string]any
	acc := map[int]map[string]string{}

	for _, line := range strings.Split(sse, "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var ev map[string]any
		if json.Unmarshal([]byte(line[6:]), &ev) != nil {
			continue
		}
		switch ev["type"] {
		case "content_block_start":
			idx := iVal(ev, "index")
			if cb, ok := ev["content_block"].(map[string]any); ok {
				for len(blocks) <= idx {
					blocks = append(blocks, nil)
				}
				blocks[idx] = cb
				acc[idx] = map[string]string{}
			}
		case "content_block_delta":
			idx := iVal(ev, "index")
			d, _ := ev["delta"].(map[string]any)
			if d == nil {
				continue
			}
			if acc[idx] == nil {
				acc[idx] = map[string]string{}
			}
			switch d["type"] {
			case "thinking_delta":
				if v, ok := d["thinking"].(string); ok {
					acc[idx]["thinking"] += v
				}
			case "text_delta":
				if v, ok := d["text"].(string); ok {
					acc[idx]["text"] += v
				}
			case "signature_delta":
				if v, ok := d["signature"].(string); ok {
					acc[idx]["signature"] += v
				}
			case "input_json_delta":
				if v, ok := d["partial_json"].(string); ok {
					acc[idx]["input_json"] += v
				}
			}
		}
	}

	for i, b := range blocks {
		if b == nil || acc[i] == nil {
			continue
		}
		switch b["type"] {
		case "thinking":
			if v := acc[i]["thinking"]; v != "" {
				b["thinking"] = v
			}
			if v := acc[i]["signature"]; v != "" {
				b["signature"] = v
			}
		case "text":
			if v := acc[i]["text"]; v != "" {
				b["text"] = v
			}
		case "tool_use", "server_tool_use":
			if v := acc[i]["input_json"]; v != "" {
				var p any
				if json.Unmarshal([]byte(v), &p) == nil {
					b["input"] = p
				}
			}
		}
	}

	out := make([]any, 0, len(blocks))
	for _, b := range blocks {
		if b != nil {
			out = append(out, b)
		}
	}
	return out
}

func iVal(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	}
	return 0
}
