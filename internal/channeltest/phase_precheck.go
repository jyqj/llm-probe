package channeltest

// ════════════════════════════════════════════════════════════
//  Phase 1a: precheck
//  cctest 00_precheck: bare "say ok", max_tokens=20, stream=true
//  NO system, NO tools, NO metadata, NO thinking
// ════════════════════════════════════════════════════════════

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
