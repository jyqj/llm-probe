package channeltest

import (
	"crypto/rand"
	"math/big"

	"detector-service/internal/channeltest/data"
)

var structuredNames = []string{
	"Alice Carter", "Bob Fischer", "Clara Zheng", "David Petrov",
	"Elena Rossi", "Frank Nakamura", "Grace Okonkwo", "Henry Larsson",
}

func randomStructuredName() string {
	idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(structuredNames))))
	return structuredNames[idx.Int64()]
}

var probeSelfIntro = &Probe{
	ID: "self_intro", Label: "结构化自述探针",
	Tags:      []string{},
	EstTokens: 25000,
	Checks:    []string{"no_thinking_leak", "structured_json_valid", "structured_schema_match", "structured_name_correct", "structured_stop_reason"},
	Run:       (*Runner).runSelfIntroProbe,
}

func (p *Runner) runSelfIntroProbe(targetBase, targetKey, model string) ([]CheckResult, error) {
	expectedName := randomStructuredName()
	body := toJSON(map[string]any{
		"model":      model,
		"max_tokens": 1024,
		"stream":     true,
		"system":     fullSystem(),
		"tools":      data.Tools(),
		"metadata":   genMetadata(),
		"messages":   []any{umsg("Describe a person named " + expectedName + " in one sentence, including their name, title, and description.")},
		// NO thinking field — matches cctest 04
		"output_config": map[string]any{
			"format": map[string]any{
				"type": "json_schema",
				"schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name":  map[string]any{"type": "string"},
						"title": map[string]any{"type": "string"},
						"desc":  map[string]any{"type": "string"},
					},
					"required":             []string{"name", "title", "desc"},
					"additionalProperties": false,
				},
			},
		},
	})

	resp, err := p.send(targetBase, targetKey, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	sse, start, delta := readSSE(resp.Body)
	full := merge(start, delta, sse)
	p.recordStreamResult(full)
	if full == nil {
		return []CheckResult{
			{Name: "no_thinking_leak", Pass: false, Detail: "parse failed", Fix: "body_rewrite"},
			{Name: "structured_json_valid", Pass: false, Detail: "parse failed", Fix: "body_rewrite"},
			{Name: "structured_schema_match", Pass: false, Detail: "parse failed", Fix: "body_rewrite"},
			{Name: "structured_name_correct", Pass: false, Detail: "parse failed", Fix: "body_rewrite"},
			{Name: "structured_stop_reason", Pass: false, Detail: "parse failed", Fix: "body_rewrite"},
		}, nil
	}

	return []CheckResult{
		checkNoThinkingWhenDisabled(full),
		checkStructuredJSONValid(full),
		checkStructuredSchemaMatch(full),
		checkStructuredNameCorrect(full, expectedName),
		checkStructuredStopReason(full),
	}, nil
}
