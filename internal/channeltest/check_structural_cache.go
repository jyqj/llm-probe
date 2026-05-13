package channeltest

import (
	"fmt"

	"detector-service/internal/fingerprint"
)

func checkCacheSmallProbe(body map[string]any) CheckResult {
	usage, _ := body["usage"].(map[string]any)
	if usage == nil {
		return CheckResult{Name: "cache_small_probe", Pass: true, Expected: "usage 对象存在", Actual: "无 usage", Detail: "no usage"}
	}
	ccCreate := fingerprint.IntVal(usage, "cache_creation_input_tokens")
	ccRead := fingerprint.IntVal(usage, "cache_read_input_tokens")
	actual := fmt.Sprintf("create=%d read=%d", ccCreate, ccRead)
	if ccCreate != 0 || ccRead != 0 {
		return CheckResult{Name: "cache_small_probe", Pass: false,
			Expected: "cache create=0 read=0", Actual: actual,
			Detail: fmt.Sprintf("small probe has non-zero cache: %s", actual), Fix: "small_probe_zero"}
	}
	return CheckResult{Name: "cache_small_probe", Pass: true,
		Expected: "cache create=0 read=0", Actual: actual,
		Detail: "cache values are zero for small probe"}
}

func checkCacheFake(body map[string]any) CheckResult {
	usage, _ := body["usage"].(map[string]any)
	if usage == nil {
		return CheckResult{Name: "cache_fake", Pass: true, Expected: "usage 对象存在", Actual: "无 usage", Detail: "no usage"}
	}
	ccCreate := fingerprint.IntVal(usage, "cache_creation_input_tokens")
	ccRead := fingerprint.IntVal(usage, "cache_read_input_tokens")
	actual := fmt.Sprintf("create=%d read=%d", ccCreate, ccRead)
	if ccCreate == 0 && ccRead == 0 {
		return CheckResult{Name: "cache_fake", Pass: false,
			Expected: "cache 非零 (已使用 cache_control)", Actual: "create=0 read=0",
			Detail: "cache_control used but cache all zero", Fix: "cache_fake"}
	}
	return CheckResult{Name: "cache_fake", Pass: true,
		Expected: "cache 非零 (已使用 cache_control)", Actual: actual,
		Detail: fmt.Sprintf("cache values non-zero: %s", actual)}
}

func checkSmallProbeExact(body map[string]any) []CheckResult {
	var checks []CheckResult
	usage, _ := body["usage"].(map[string]any)
	if usage == nil {
		checks = append(checks, CheckResult{Name: "small_output_tokens", Pass: false,
			Expected: "usage 对象存在", Actual: "无 usage", Detail: "no usage object", Fix: "body_rewrite"})
		return checks
	}

	outTok := fingerprint.IntVal(usage, "output_tokens")
	actual := fmt.Sprintf("%d", outTok)
	if outTok == 1 {
		checks = append(checks, CheckResult{Name: "small_output_tokens", Pass: true,
			Expected: "1", Actual: actual, Detail: "output_tokens=1 OK"})
	} else {
		checks = append(checks, CheckResult{Name: "small_output_tokens", Pass: false,
			Expected: "1", Actual: actual, Detail: fmt.Sprintf("output_tokens=%d expected 1", outTok), Fix: "body_rewrite"})
	}

	sr, _ := body["stop_reason"].(string)
	if sr == "max_tokens" {
		checks = append(checks, CheckResult{Name: "small_stop_reason", Pass: true,
			Expected: "max_tokens", Actual: sr, Detail: "stop_reason=max_tokens OK"})
	} else {
		checks = append(checks, CheckResult{Name: "small_stop_reason", Pass: false,
			Expected: "max_tokens", Actual: sr, Detail: "stop_reason=" + sr + " expected max_tokens", Fix: "body_rewrite"})
	}

	cc, hasCC := usage["cache_creation"].(map[string]any)
	if !hasCC {
		checks = append(checks, CheckResult{Name: "small_ephemeral_zero", Pass: false,
			Expected: "cache_creation 嵌套对象存在", Actual: "不存在",
			Detail: "no cache_creation nested object", Fix: "body_rewrite"})
	} else {
		e5m := fingerprint.IntVal(cc, "ephemeral_5m_input_tokens")
		e1h := fingerprint.IntVal(cc, "ephemeral_1h_input_tokens")
		actual := fmt.Sprintf("5m=%d 1h=%d", e5m, e1h)
		checks = append(checks, CheckResult{Name: "small_ephemeral_zero", Pass: true,
			Expected: "ephemeral 字段存在", Actual: actual,
			Detail: fmt.Sprintf("ephemeral_5m=%d ephemeral_1h=%d (cache_control used)", e5m, e1h)})
	}

	ccCreate := fingerprint.IntVal(usage, "cache_creation_input_tokens")
	ccRead := fingerprint.IntVal(usage, "cache_read_input_tokens")
	actual = fmt.Sprintf("create=%d read=%d", ccCreate, ccRead)
	checks = append(checks, CheckResult{Name: "small_cache_zero", Pass: true,
		Expected: "cache 字段存在", Actual: actual,
		Detail: fmt.Sprintf("cache create=%d read=%d (cache_control used)", ccCreate, ccRead)})

	return checks
}

func checkCacheCreationComplete(body map[string]any) CheckResult {
	usage, _ := body["usage"].(map[string]any)
	if usage == nil {
		return CheckResult{Name: "cache_creation_complete", Pass: true, Expected: "usage 存在", Actual: "无 usage (skip)", Detail: "no usage (skip)"}
	}
	cc, ok := usage["cache_creation"].(map[string]any)
	if !ok {
		return CheckResult{Name: "cache_creation_complete", Pass: false,
			Expected: "cache_creation 嵌套对象", Actual: "不存在",
			Detail: "no cache_creation nested object", Fix: "body_rewrite"}
	}
	_, has5m := cc["ephemeral_5m_input_tokens"]
	_, has1h := cc["ephemeral_1h_input_tokens"]
	if has5m && has1h {
		return CheckResult{Name: "cache_creation_complete", Pass: true,
			Expected: "ephemeral_5m + ephemeral_1h", Actual: "两者均存在",
			Detail: "both ephemeral fields present"}
	}
	if has5m && !has1h {
		return CheckResult{Name: "cache_creation_complete", Pass: false,
			Expected: "ephemeral_5m + ephemeral_1h", Actual: "仅 ephemeral_5m",
			Detail: "missing ephemeral_1h_input_tokens", Fix: "body_rewrite"}
	}
	return CheckResult{Name: "cache_creation_complete", Pass: false,
		Expected: "ephemeral_5m + ephemeral_1h", Actual: "均缺失",
		Detail: "missing ephemeral fields", Fix: "body_rewrite"}
}
