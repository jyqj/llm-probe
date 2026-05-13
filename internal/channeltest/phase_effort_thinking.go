package channeltest

import (
	"encoding/base64"
	"fmt"
)

var probeEffortThinking = &Probe{
	ID: "effort_thinking", Label: "Effort 级别思考探针",
	NeedsThinking: true,
	Tags:          []string{},
	EstTokens:     2000,
	Checks: []string{
		"effort_high_thinking", "effort_high_signature",
		"effort_medium_no_think", "effort_low_no_think",
		"effort_max_thinking", "effort_xhigh_thinking",
	},
	Run: (*Runner).runEffortThinking,
}

func (p *Runner) runEffortThinking(targetBase, targetKey, model string) ([]CheckResult, error) {
	caps := GetModelCaps(model)
	if len(caps.EffortLevels) == 0 {
		return nil, nil
	}

	highChecks, err := p.runEffortHigh(targetBase, targetKey, model)
	if err != nil {
		highChecks = []CheckResult{
			{Name: "effort_high_thinking", Pass: false, Expected: "thinking 块", Actual: "请求失败", Detail: err.Error()},
			{Name: "effort_high_signature", Pass: false, Expected: "有效 signature", Actual: "请求失败", Detail: err.Error()},
		}
	}

	medChecks, err := p.runEffortMedium(targetBase, targetKey, model)
	if err != nil {
		medChecks = []CheckResult{
			{Name: "effort_medium_no_think", Pass: false, Expected: "text 或短 thinking", Actual: "请求失败", Detail: err.Error()},
		}
	}

	lowChecks, err := p.runEffortLow(targetBase, targetKey, model)
	if err != nil {
		lowChecks = []CheckResult{
			{Name: "effort_low_no_think", Pass: false, Expected: "text (无 thinking)", Actual: "请求失败", Detail: err.Error()},
		}
	}

	var all []CheckResult
	all = append(all, highChecks...)
	all = append(all, medChecks...)
	all = append(all, lowChecks...)

	if SupportsEffort(model, "max") {
		maxChecks, err := p.runEffortMax(targetBase, targetKey, model)
		if err != nil {
			maxChecks = []CheckResult{
				{Name: "effort_max_thinking", Pass: false, Expected: "thinking 块", Actual: "请求失败", Detail: err.Error()},
			}
		}
		all = append(all, maxChecks...)
	}

	if SupportsEffort(model, "xhigh") {
		xhighChecks, err := p.runEffortXHigh(targetBase, targetKey, model)
		if err != nil {
			xhighChecks = []CheckResult{
				{Name: "effort_xhigh_thinking", Pass: false, Expected: "thinking 块", Actual: "请求失败", Detail: err.Error()},
			}
		}
		all = append(all, xhighChecks...)
	}

	return all, nil
}

func (p *Runner) runEffortHigh(targetBase, targetKey, model string) ([]CheckResult, error) {
	req := map[string]any{
		"model":         model,
		"max_tokens":    16000,
		"stream":        false,
		"messages":      []any{umsg("请逐步推理：在一个黑色的袋子里放有三种口味的糖果，每种糖果有两种不同的形状（圆形和五角星形）。苹果味圆形7个五角星7个，桃子味圆形9个五角星6个，西瓜味圆形8个五角星4个。最少取出多少个糖才能保证手中同时拥有不同形状的苹果味和桃子味的糖？")},
		"output_config": EffortParam("high"),
	}
	if tp := ThinkingParam(model); tp != nil {
		req["thinking"] = tp
	}
	body := toJSON(req)

	j, err := p.sendReadJSON(targetBase, targetKey, body)
	if err != nil {
		return nil, err
	}

	return checkEffortAcceptedAndSignature(j, "effort_high_thinking", "effort_high_signature"), nil
}

// checkEffortAcceptedAndSignature validates:
// - effort_*_thinking: request was accepted and produced valid content (pass).
//   Adaptive thinking at effort=high may or may not produce thinking blocks.
// - effort_*_signature: if a thinking block exists, validate its signature.
func checkEffortAcceptedAndSignature(j map[string]any, thinkCheckName, sigCheckName string) []CheckResult {
	var checks []CheckResult
	content, _ := j["content"].([]any)

	hasContent := false
	var thinkingBlock map[string]any
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		hasContent = true
		if t, _ := m["type"].(string); t == "thinking" {
			thinkingBlock = m
		}
	}

	if hasContent {
		detail := "effort 请求被接受，响应含有效 content"
		if thinkingBlock != nil {
			detail += " (含 thinking 块)"
		}
		checks = append(checks, CheckResult{
			Name: thinkCheckName, Pass: true,
			Expected: "effort 请求被接受", Actual: fmt.Sprintf("%d content blocks", len(content)),
			Detail: detail,
		})
	} else {
		checks = append(checks, CheckResult{
			Name: thinkCheckName, Pass: false,
			Expected: "effort 请求被接受", Actual: "无 content",
			Detail: "effort 请求返回了空 content",
		})
	}

	if thinkingBlock != nil {
		sig, _ := thinkingBlock["signature"].(string)
		if sig == "" {
			checks = append(checks, CheckResult{
				Name: sigCheckName, Pass: false,
				Expected: "有效 base64 signature", Actual: "空 signature",
				Detail: "thinking block 无 signature", Fix: "signature_rewrite",
			})
		} else {
			raw, err := base64.StdEncoding.DecodeString(sig)
			if err != nil || len(raw) < 4 {
				checks = append(checks, CheckResult{
					Name: sigCheckName, Pass: false,
					Expected: "有效 base64 (≥4 bytes)", Actual: "解码失败或过短",
					Detail: "signature 无效", Fix: "signature_rewrite",
				})
			} else {
				checks = append(checks, CheckResult{
					Name: sigCheckName, Pass: true,
					Expected: "有效 base64 signature", Actual: fmt.Sprintf("%d bytes", len(raw)),
					Detail: fmt.Sprintf("signature 有效 (%d bytes)", len(raw)),
				})
			}
		}
	} else {
		checks = append(checks, CheckResult{
			Name: sigCheckName, Pass: true,
			Expected: "signature (如有 thinking)", Actual: "无 thinking block, 跳过",
			Detail: "adaptive thinking 未产生 thinking 块，signature 检查跳过",
		})
	}

	return checks
}

func (p *Runner) runEffortMedium(targetBase, targetKey, model string) ([]CheckResult, error) {
	req := map[string]any{
		"model":         model,
		"max_tokens":    256,
		"stream":        false,
		"messages":      []any{umsg("What is 2+2?")},
		"output_config": EffortParam("medium"),
	}
	if tp := ThinkingParam(model); tp != nil {
		req["thinking"] = tp
	}
	body := toJSON(req)

	j, err := p.sendReadJSON(targetBase, targetKey, body)
	if err != nil {
		return nil, err
	}

	content, _ := j["content"].([]any)
	if len(content) == 0 {
		return []CheckResult{{Name: "effort_medium_no_think", Pass: false,
			Expected: "text 或短 thinking", Actual: "无 content",
			Detail: "response has no content blocks"}}, nil
	}

	first, ok := content[0].(map[string]any)
	if !ok {
		return []CheckResult{{Name: "effort_medium_no_think", Pass: false,
			Expected: "text 或短 thinking", Actual: "content[0] 非对象",
			Detail: "content[0] not a map"}}, nil
	}

	firstType, _ := first["type"].(string)
	if firstType == "thinking" {
		thinkText, _ := first["thinking"].(string)
		return []CheckResult{{
			Name: "effort_medium_no_think", Pass: true,
			Expected: "text 或短 thinking", Actual: fmt.Sprintf("thinking (%d 字符)", len(thinkText)),
			Detail: fmt.Sprintf("content[0].type = %q (%d 字符 thinking)", firstType, len(thinkText)),
		}}, nil
	}
	if firstType == "text" {
		return []CheckResult{{
			Name: "effort_medium_no_think", Pass: true,
			Expected: "text 或短 thinking", Actual: "text (无 thinking)",
			Detail: fmt.Sprintf("content[0].type = %q (thinking 已跳过)", firstType),
		}}, nil
	}

	return []CheckResult{{
		Name: "effort_medium_no_think", Pass: false,
		Expected: "text 或 thinking", Actual: firstType,
		Detail: fmt.Sprintf("content[0].type = %q (unexpected)", firstType),
	}}, nil
}

func (p *Runner) runEffortLow(targetBase, targetKey, model string) ([]CheckResult, error) {
	body := toJSON(map[string]any{
		"model":      model,
		"max_tokens": 256,
		"stream":     false,
		"messages":   []any{umsg("Say hello.")},
		"output_config": map[string]any{
			"effort": "low",
		},
	})

	j, err := p.sendReadJSON(targetBase, targetKey, body)
	if err != nil {
		return nil, err
	}

	content, _ := j["content"].([]any)
	if len(content) == 0 {
		return []CheckResult{{Name: "effort_low_no_think", Pass: false,
			Expected: "text (无 thinking)", Actual: "无 content",
			Detail: "response has no content blocks"}}, nil
	}

	first, ok := content[0].(map[string]any)
	if !ok {
		return []CheckResult{{Name: "effort_low_no_think", Pass: false,
			Expected: "text (无 thinking)", Actual: "content[0] 非对象",
			Detail: "content[0] not a map"}}, nil
	}

	firstType, _ := first["type"].(string)
	if firstType == "text" {
		return []CheckResult{{
			Name: "effort_low_no_think", Pass: true,
			Expected: "text (无 thinking)", Actual: "text",
			Detail: fmt.Sprintf("content[0].type = %q (thinking 已跳过)", firstType),
		}}, nil
	}

	if firstType == "thinking" {
		thinkText, _ := first["thinking"].(string)
		if len(thinkText) <= 20 {
			return []CheckResult{{
				Name: "effort_low_no_think", Pass: true,
				Expected: "text (无 thinking)", Actual: fmt.Sprintf("thinking 极短 (%d 字符)", len(thinkText)),
				Detail: fmt.Sprintf("content[0].type = %q, thinking 极短 (%d 字符)", firstType, len(thinkText)),
			}}, nil
		}
		return []CheckResult{{
			Name: "effort_low_no_think", Pass: false,
			Expected: "text (无 thinking)", Actual: fmt.Sprintf("thinking %d 字符", len(thinkText)),
			Detail: fmt.Sprintf("content[0].type = %q, effort=low 但产生了 %d 字符 thinking", firstType, len(thinkText)),
		}}, nil
	}

	return []CheckResult{{
		Name: "effort_low_no_think", Pass: false,
		Expected: "text (无 thinking)", Actual: firstType,
		Detail: fmt.Sprintf("content[0].type = %q (unexpected)", firstType),
	}}, nil
}

func (p *Runner) runEffortMax(targetBase, targetKey, model string) ([]CheckResult, error) {
	req := map[string]any{
		"model":         model,
		"max_tokens":    16000,
		"stream":        false,
		"messages":      []any{umsg("请逐步推理：一个标准的 8×8 棋盘，去掉左上角和右下角的两个格子。能否用 1×2 的骨牌完全覆盖剩余的 62 个格子？请解释原因。")},
		"output_config": EffortParam("max"),
	}
	if tp := ThinkingParam(model); tp != nil {
		req["thinking"] = tp
	}
	body := toJSON(req)

	j, err := p.sendReadJSON(targetBase, targetKey, body)
	if err != nil {
		return nil, err
	}

	return checkEffortAccepted(j, "effort_max_thinking", "max"), nil
}

func (p *Runner) runEffortXHigh(targetBase, targetKey, model string) ([]CheckResult, error) {
	req := map[string]any{
		"model":         model,
		"max_tokens":    16000,
		"stream":        false,
		"messages":      []any{umsg("Explain briefly why P != NP is believed to be true by most computer scientists.")},
		"output_config": EffortParam("xhigh"),
	}
	if tp := ThinkingParam(model); tp != nil {
		req["thinking"] = tp
	}
	body := toJSON(req)

	j, err := p.sendReadJSON(targetBase, targetKey, body)
	if err != nil {
		return nil, err
	}

	return checkEffortAccepted(j, "effort_xhigh_thinking", "xhigh"), nil
}

func checkEffortAccepted(j map[string]any, checkName, level string) []CheckResult {
	content, _ := j["content"].([]any)
	hasContent := false
	hasThinking := false
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		hasContent = true
		if t, _ := m["type"].(string); t == "thinking" {
			hasThinking = true
		}
	}

	if !hasContent {
		return []CheckResult{{
			Name: checkName, Pass: false,
			Expected: fmt.Sprintf("effort=%s 请求被接受", level), Actual: "无 content",
			Detail: fmt.Sprintf("effort=%s 请求返回了空 content", level),
		}}
	}

	detail := fmt.Sprintf("effort=%s 请求被接受", level)
	if hasThinking {
		detail += " (含 thinking 块)"
	}
	return []CheckResult{{
		Name: checkName, Pass: true,
		Expected: fmt.Sprintf("effort=%s 请求被接受", level), Actual: fmt.Sprintf("%d content blocks", len(content)),
		Detail: detail,
	}}
}

