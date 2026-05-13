package channeltest

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

var probeBashTool = &Probe{
	ID: "bash_tool", Label: "Bash 工具探针",
	Tags:      []string{},
	EstTokens: 200,
	Checks:    []string{"bash_stop_reason", "bash_tool_name", "bash_tool_rejected"},
	Run:       (*Runner).runBashTool,
}

func (p *Runner) runBashTool(targetBase, targetKey, model string) ([]CheckResult, error) {
	validChecks, err := p.runBashToolValid(targetBase, targetKey, model)
	if err != nil {
		validChecks = []CheckResult{
			{Name: "bash_stop_reason", Pass: false, Expected: "tool_use", Actual: "请求失败", Detail: err.Error()},
			{Name: "bash_tool_name", Pass: false, Expected: `name="bash"`, Actual: "请求失败", Detail: err.Error()},
		}
	}

	invalidCheck, err := p.runBashToolInvalid(targetBase, targetKey, model)
	if err != nil {
		invalidCheck = CheckResult{Name: "bash_tool_rejected", Pass: false, Expected: "400", Actual: "请求失败", Detail: err.Error()}
	}

	return append(validChecks, invalidCheck), nil
}

func (p *Runner) runBashToolValid(targetBase, targetKey, model string) ([]CheckResult, error) {
	body := toJSON(map[string]any{
		"model":      model,
		"max_tokens": 1024,
		"stream":     false,
		"tools": []any{
			map[string]any{
				"type": "bash_20250124",
				"name": "bash",
			},
		},
		"tool_choice": map[string]any{"type": "tool", "name": "bash"},
		"messages":    []any{umsg("Run: echo hello")},
	})

	j, err := p.sendReadJSON(targetBase, targetKey, body)
	if err != nil {
		return nil, err
	}

	var checks []CheckResult

	sr, _ := j["stop_reason"].(string)
	if sr == "tool_use" {
		checks = append(checks, CheckResult{
			Name: "bash_stop_reason", Pass: true,
			Expected: "tool_use", Actual: sr,
			Detail: "stop_reason=tool_use OK",
		})
	} else {
		checks = append(checks, CheckResult{
			Name: "bash_stop_reason", Pass: false,
			Expected: "tool_use", Actual: sr,
			Detail: fmt.Sprintf("stop_reason=%q, 期望 tool_use", sr),
		})
	}

	content, _ := j["content"].([]any)
	foundBash := false
	for _, cb := range content {
		m, ok := cb.(map[string]any)
		if !ok {
			continue
		}
		if n, _ := m["name"].(string); n == "bash" {
			foundBash = true
			break
		}
	}
	if foundBash {
		checks = append(checks, CheckResult{
			Name: "bash_tool_name", Pass: true,
			Expected: `name="bash"`, Actual: `name="bash"`,
			Detail: `content 包含 name="bash" 的 tool_use 块`,
		})
	} else {
		checks = append(checks, CheckResult{
			Name: "bash_tool_name", Pass: false,
			Expected: `name="bash"`, Actual: "未找到",
			Detail: "content 无 name=bash 的 tool_use 块",
		})
	}

	return checks, nil
}

func (p *Runner) runBashToolInvalid(targetBase, targetKey, model string) (CheckResult, error) {
	body := toJSON(map[string]any{
		"model":      model,
		"max_tokens": 1024,
		"stream":     false,
		"tools": []any{
			map[string]any{
				"type": "bash_20250124",
				"name": "invalid_name",
			},
		},
		"messages": []any{umsg("hello")},
	})

	url := strings.TrimRight(targetBase, "/") + "/v1/messages"
	req, err := http.NewRequest("POST", url, strings.NewReader(string(body)))
	if err != nil {
		return CheckResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", targetKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "interleaved-thinking-2025-05-14")

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return CheckResult{}, err
	}
	defer func() { io.Copy(io.Discard, resp.Body); resp.Body.Close() }()

	status := resp.StatusCode
	actual := fmt.Sprintf("%d", status)
	if status == 400 {
		return CheckResult{
			Name: "bash_tool_rejected", Pass: true,
			Expected: "400", Actual: actual,
			Detail: fmt.Sprintf("状态码 %d, 非法 bash tool name 被正确拒绝", status),
		}, nil
	}
	return CheckResult{
		Name: "bash_tool_rejected", Pass: false,
		Expected: "400", Actual: actual,
		Detail: fmt.Sprintf("状态码 %d, 期望 400", status),
	}, nil
}
