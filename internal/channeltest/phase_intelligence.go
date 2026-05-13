package channeltest

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var probeIntelligence = &Probe{
	ID: "intelligence", Label: "降智检测探针",
	NeedsThinking: true,
	OnlyModels:    []string{"claude-opus-4-6"},
	Tags:          []string{},
	EstTokens:     500,
	Checks:    []string{"intelligence_answer"},
	Run:       (*Runner).runIntelligence,
}

var numberRe = regexp.MustCompile(`\d+`)

func (p *Runner) runIntelligence(targetBase, targetKey, model string) ([]CheckResult, error) {
	body := toJSON(map[string]any{
		"model":      model,
		"max_tokens": 8000,
		"stream":     false,
		"thinking": map[string]any{
			"type":          "enabled",
			"budget_tokens": 5000,
		},
		"messages": []any{umsg("请逐步推理：在一个黑色的袋子里放有三种口味的糖果，每种糖果有两种不同的形状（圆形和五角星形）。苹果味圆形7个五角星7个，桃子味圆形9个五角星6个，西瓜味圆形8个五角星4个。最少取出多少个糖才能保证手中同时拥有不同形状的苹果味和桃子味的糖？请在回答最后一行只写一个数字作为最终答案。")},
	})

	j, err := p.sendReadJSON(targetBase, targetKey, body)
	if err != nil {
		return nil, err
	}

	text := strings.TrimSpace(collectResponseText(j))
	if text == "" {
		return []CheckResult{{Name: "intelligence_answer", Pass: false,
			Expected: "21", Actual: "无文本输出",
			Detail: "response 无 text 内容"}}, nil
	}

	lines := strings.Split(strings.TrimSpace(text), "\n")
	lastLine := strings.TrimSpace(lines[len(lines)-1])
	nums := numberRe.FindAllString(lastLine, -1)

	var answer int
	if len(nums) > 0 {
		answer, _ = strconv.Atoi(nums[len(nums)-1])
	}

	actual := fmt.Sprintf("%d", answer)
	if answer == 21 {
		return []CheckResult{{
			Name: "intelligence_answer", Pass: true,
			Expected: "21", Actual: actual,
			Detail: "糖果抽屉题答案正确: 21",
		}}, nil
	}

	return []CheckResult{{
		Name: "intelligence_answer", Pass: false,
		Expected: "21", Actual: actual,
		Detail: fmt.Sprintf("模型答案 %d, 正确答案 21", answer),
	}}, nil
}
