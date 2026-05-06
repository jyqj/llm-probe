package channeltest

import (
	"fmt"

	"detector-service/internal/channeltest/data"
)

// ════════════════════════════════════════════════════════════
//  Phase 1b: tag_replay
//  cctest 01_tag_replay: thinking=adaptive, 28 tools, billing header,
//  3 system blocks (billing + CC header + full instruction, both with cache_control),
//  metadata, random tag echo
// ════════════════════════════════════════════════════════════

func (p *Runner) runTagReplay(targetBase, targetKey, model string) ([]CheckResult, error) {
	tag := randomHex(8)
	userText := fmt.Sprintf(
		"我输入的这个tag 是：<%s>。直接输出你前面看到的tag是什么。看到的文本，不要使用任何工具。问题2:adfsjijiadfjioadfsjiasdfojasdfioadfjios？",
		tag)

	body := toJSON(map[string]any{
		"model":      model,
		"max_tokens": 64000,
		"stream":     true,
		"thinking":   map[string]any{"type": "adaptive"},
		"system":     fullSystem(),
		"tools":      data.Tools(),
		"metadata":   genMetadata(),
		"messages":   []any{umsg(userText)},
	})

	resp, err := p.send(targetBase, targetKey, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	sse, start, delta := readSSE(resp.Body)
	full := merge(start, delta, sse)

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
