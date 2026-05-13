package channeltest

import (
	"fmt"

	"detector-service/internal/channeltest/data"
)

var probeTagReplay = &Probe{
	ID: "tag_replay", Label: "全量指纹探针",
	Required:      true,
	NeedsThinking: true,
	Tags:          []string{},
	EstTokens:     25000,
	Checks: []string{
		"id_format", "model_name", "signature", "signature_length",
		"thinking_present", "usage_structure", "field_order",
		"inference_geo", "stop_details", "stop_details_structure",
		"stop_reason", "stop_sequence_null", "thinking_order",
		"thinking_display_omitted", "tag_replay", "cache_fake",
		"message_start_usage", "sse_ping_position",
		"service_tier", "signature_type_leak",
		"usage_fields_complete", "cache_creation_complete",
	},
	Run: (*Runner).runTagReplay,
}

func (p *Runner) runTagReplay(targetBase, targetKey, model string) ([]CheckResult, error) {
	tag := randomHex(8)
	userText := fmt.Sprintf(
		"我输入的这个tag 是：<%s>。直接输出你前面看到的tag是什么。看到的文本，不要使用任何工具。问题2:adfsjijiadfjioadfsjiasdfojasdfioadfjios？",
		tag)

	req := map[string]any{
		"model":      model,
		"max_tokens": 64000,
		"stream":     true,
		"system":     fullSystem(),
		"tools":      data.Tools(),
		"metadata":   genMetadata(),
		"messages":   []any{umsg(userText)},
	}
	if tp := ThinkingParam(model); tp != nil {
		req["thinking"] = tp
	}
	body := toJSON(req)

	resp, err := p.send(targetBase, targetKey, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	sse, start, delta := readSSE(resp.Body)
	full := merge(start, delta, sse)
	p.recordStreamResult(full)

	if full == nil {
		return batchFail([]string{
			"id_format", "model_name", "signature", "signature_length",
			"thinking_present", "usage_structure", "field_order",
			"inference_geo", "stop_details", "stop_details_structure",
			"stop_reason", "stop_sequence_null", "thinking_order",
			"thinking_display_omitted", "tag_replay", "cache_fake",
			"message_start_usage",
		}), nil
	}

	return []CheckResult{
		checkIDFormat(full),
		checkModelName(full, model),
		checkSignature(full),
		checkSignatureLength(full),
		checkThinkingPresent(full),
		checkUsageStructure(full),
		checkFieldOrder([]byte(sse)),
		checkInferenceGeo(full, model),
		checkStopDetails(full),
		checkStopDetailsStructure(full),
		checkStopReason(full),
		checkStopSequenceNull(full),
		checkThinkingOrder(full),
		checkThinkingDisplayOmitted(full, model),
		checkTagReplay(full, tag),
		checkCacheFake(full),
		checkMessageStartUsage(sse),
		checkSSEPingPosition(sse),
		checkServiceTier(full),
		checkSignatureTypeLeak(full),
		checkUsageFieldsComplete(full),
		checkCacheCreationComplete(full),
	}, nil
}
