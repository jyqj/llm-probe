package channeltest

import "strings"

// ModelCaps describes the capabilities and pricing of a Claude model.
type ModelCaps struct {
	Family         string   // "haiku", "sonnet", "opus"
	Generation     string   // "4-5", "4-6", "4-7"
	Thinking       bool     // supports any thinking mode
	ThinkingMode   string   // "adaptive_only", "adaptive", "enabled", "" (none)
	Signatures     bool     // thinking blocks have signatures
	DisplayDefault string   // "summarized" or "omitted"
	MaxOutput      int      // max output tokens
	InputPrice     float64  // USD per million input tokens
	OutputPrice    float64  // USD per million output tokens
	EffortLevels   []string // supported effort levels: "low","medium","high","xhigh","max"
}

var knownModels = map[string]ModelCaps{
	"claude-haiku-4-5": {
		Family: "haiku", Generation: "4-5",
		Thinking: true, ThinkingMode: "enabled", Signatures: false,
		DisplayDefault: "summarized", MaxOutput: 64000,
		InputPrice: 1.0, OutputPrice: 5.0,
		EffortLevels: nil,
	},
	"claude-sonnet-4-6": {
		Family: "sonnet", Generation: "4-6",
		Thinking: true, ThinkingMode: "adaptive", Signatures: true,
		DisplayDefault: "summarized", MaxOutput: 64000,
		InputPrice: 3.0, OutputPrice: 15.0,
		EffortLevels: []string{"low", "medium", "high", "max"},
	},
	"claude-opus-4-5": {
		Family: "opus", Generation: "4-5",
		Thinking: true, ThinkingMode: "enabled", Signatures: true,
		DisplayDefault: "summarized", MaxOutput: 64000,
		InputPrice: 5.0, OutputPrice: 25.0,
		EffortLevels: []string{"low", "medium", "high"},
	},
	"claude-opus-4-6": {
		Family: "opus", Generation: "4-6",
		Thinking: true, ThinkingMode: "adaptive", Signatures: true,
		DisplayDefault: "summarized", MaxOutput: 128000,
		InputPrice: 5.0, OutputPrice: 25.0,
		EffortLevels: []string{"low", "medium", "high", "max"},
	},
	"claude-opus-4-7": {
		Family: "opus", Generation: "4-7",
		Thinking: true, ThinkingMode: "adaptive_only", Signatures: true,
		DisplayDefault: "omitted", MaxOutput: 128000,
		InputPrice: 5.0, OutputPrice: 25.0,
		EffortLevels: []string{"low", "medium", "high", "xhigh", "max"},
	},
}

// GetModelCaps returns capabilities for a model. Falls back to sonnet-4-6 defaults.
func GetModelCaps(model string) ModelCaps {
	if caps, ok := knownModels[model]; ok {
		return caps
	}
	for prefix, caps := range knownModels {
		if strings.HasPrefix(model, prefix) {
			return caps
		}
	}
	return knownModels["claude-sonnet-4-6"]
}

// ThinkingParam builds the correct thinking parameter for a model.
func ThinkingParam(model string) map[string]any {
	caps := GetModelCaps(model)
	switch caps.ThinkingMode {
	case "adaptive_only", "adaptive":
		return map[string]any{"type": "adaptive"}
	case "enabled":
		return map[string]any{"type": "enabled", "budget_tokens": 10000}
	default:
		return nil
	}
}

// ThinkingEnabledParam builds thinking with type=enabled for models that support it.
func ThinkingEnabledParam(model string, budget int) map[string]any {
	caps := GetModelCaps(model)
	if !caps.Thinking {
		return nil
	}
	if caps.ThinkingMode == "adaptive_only" {
		return map[string]any{"type": "adaptive"}
	}
	return map[string]any{"type": "enabled", "budget_tokens": budget}
}

// SupportsEffort checks if a model supports a given effort level.
func SupportsEffort(model, effort string) bool {
	caps := GetModelCaps(model)
	for _, e := range caps.EffortLevels {
		if e == effort {
			return true
		}
	}
	return false
}

// EffortParam builds the output_config for a given effort level.
func EffortParam(effort string) map[string]any {
	return map[string]any{"effort": effort}
}

// BillingEstimate holds cost estimation for a test run.
type BillingEstimate struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	InputCost    float64 `json:"input_cost"`
	OutputCost   float64 `json:"output_cost"`
	TotalCost    float64 `json:"total_cost"`
	PriceRatio   float64 `json:"price_ratio"`
	Source       string  `json:"source,omitempty"`
}

// EstimateBilling computes cost from token counts and model pricing.
func EstimateBilling(model string, inputTokens, outputTokens int) BillingEstimate {
	caps := GetModelCaps(model)
	inCost := float64(inputTokens) / 1_000_000 * caps.InputPrice
	outCost := float64(outputTokens) / 1_000_000 * caps.OutputPrice
	total := inCost + outCost
	ratio := 1.0
	if caps.InputPrice > 0 {
		ratio = caps.InputPrice / 3.0
	}
	return BillingEstimate{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		InputCost:    round4(inCost),
		OutputCost:   round4(outCost),
		TotalCost:    round4(total),
		PriceRatio:   round2(ratio),
	}
}

func round4(f float64) float64 {
	return float64(int(f*10000+0.5)) / 10000
}

// SupportedModels returns all model IDs we know about.
func SupportedModels() []string {
	return []string{
		"claude-haiku-4-5",
		"claude-sonnet-4-6",
		"claude-opus-4-5",
		"claude-opus-4-6",
		"claude-opus-4-7",
	}
}
