package channeltest

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

var probeSignatureReject = &Probe{
	ID: "signature_reject", Label: "空签名拒绝探针",
	NeedsSignature: true,
	Tags:           []string{"monitor"},
	EstTokens:      100,
	Checks:    []string{"signature_empty_rejected"},
	Run:       (*Runner).runSignatureReject,
}

func (p *Runner) runSignatureReject(targetBase, targetKey, model string) ([]CheckResult, error) {
	body := toJSON(map[string]any{
		"model":      model,
		"max_tokens": 1024,
		"stream":     false,
		"thinking": map[string]any{
			"type":          "enabled",
			"budget_tokens": 5000,
		},
		"messages": []any{
			map[string]any{"role": "user", "content": "What is 2+2?"},
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{
						"type":      "thinking",
						"thinking":  "Let me think about this.",
						"signature": "",
					},
					map[string]any{
						"type": "text",
						"text": "4",
					},
				},
			},
			map[string]any{"role": "user", "content": "Are you sure?"},
		},
	})

	url := strings.TrimRight(targetBase, "/") + "/v1/messages"
	req, err := http.NewRequest("POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", targetKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "interleaved-thinking-2025-05-14")

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { io.Copy(io.Discard, resp.Body); resp.Body.Close() }()

	status := resp.StatusCode
	actual := fmt.Sprintf("%d", status)
	if status == 400 {
		return []CheckResult{{
			Name: "signature_empty_rejected", Pass: true,
			Expected: "400", Actual: actual,
			Detail: fmt.Sprintf("状态码 %d, 空 signature 被正确拒绝", status),
		}}, nil
	}
	return []CheckResult{{
		Name: "signature_empty_rejected", Pass: false,
		Expected: "400", Actual: actual,
		Detail: fmt.Sprintf("状态码 %d, 期望 400", status),
	}}, nil
}
