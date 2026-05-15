package channeltest

import "detector-service/internal/channeltest/data"

var probeLogic = &Probe{
	ID: "logic", Label: "逻辑推理探针",
	Tags:      []string{"heavy"},
	EstTokens: 25000,
	Checks:    []string{"logic_answer"},
	Run:       (*Runner).runLogicProbe,
}

func (p *Runner) runLogicProbe(targetBase, targetKey, model string) ([]CheckResult, error) {
	req := map[string]any{
		"model":      model,
		"max_tokens": 64000,
		"stream":     true,
		"system":     fullSystem(),
		"tools":      data.Tools(),
		"metadata":   genMetadata(),
		"messages":   []any{umsg("请逐步推理：一栋楼有3个开关控制3楼的3盏灯，你在1楼只能上去一次。如何确定每个开关对应哪盏灯？如果变成4个开关4盏灯，仍然只能上去一次，怎么办？")},
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
		return []CheckResult{{Name: "logic_answer", Pass: false, Detail: "parse failed"}}, nil
	}

	return []CheckResult{checkLogicAnswer(full)}, nil
}
