package channeltest

var probePrecheck = &Probe{
	ID: "precheck", Label: "基线流式探针",
	Required: true,
	Tags:     []string{"monitor"},
	EstTokens: 500,
	Checks: []string{
		"headers", "request_id", "x_new_api_version", "cf_headers",
		"server_timing", "cf_ray_format", "cookie_domain", "server_header",
		"sse_done", "sse_event_order", "sse_tailing", "delta_usage_slim",
		"sse_ping_position", "message_start_output_zero",
		"container", "bedrock_state", "cache_small_probe",
	},
	Run: (*Runner).runPrecheck,
}

func (p *Runner) runPrecheck(targetBase, targetKey, model string) ([]CheckResult, error) {
	body := toJSON(map[string]any{
		"model":      model,
		"messages":   []any{umsg("say ok")},
		"max_tokens": 20,
		"stream":     true,
	})

	resp, err := p.send(targetBase, targetKey, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var checks []CheckResult
	checks = append(checks, checkHeaders(resp.Header))
	checks = append(checks, checkRequestID(resp.Header))
	checks = append(checks, checkXNewApiVersion(resp.Header))
	checks = append(checks, checkCfHeaders(resp.Header))
	checks = append(checks, checkServerTiming(resp.Header))
	checks = append(checks, checkCfRayFormat(resp.Header))
	checks = append(checks, checkCookieDomain(resp.Header))
	checks = append(checks, checkServerHeader(resp.Header))

	sse, start, _ := readSSE(resp.Body)
	checks = append(checks, checkSSEDone(sse))
	checks = append(checks, checkSSEEventOrder(sse))
	checks = append(checks, checkSSETailing(sse))
	checks = append(checks, checkMessageDeltaUsage(sse))
	checks = append(checks, checkSSEPingPosition(sse))
	checks = append(checks, checkMessageStartOutputZero(sse))

	if start != nil {
		checks = append(checks, checkContainer(start))
		checks = append(checks, checkBedrockState(start))
		checks = append(checks, checkCacheSmallProbe(start))
	}
	return checks, nil
}
