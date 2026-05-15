package channeltest

import "sort"

const (
	channelProbeDocBase = "docs/channel-probes/"
)

// ModelMetadata is the API-facing single source of truth for supported model capabilities.
type ModelMetadata struct {
	ID             string   `json:"id"`
	Family         string   `json:"family"`
	Generation     string   `json:"generation"`
	Thinking       bool     `json:"thinking"`
	ThinkingMode   string   `json:"thinking_mode"`
	Signatures     bool     `json:"signatures"`
	DisplayDefault string   `json:"display_default"`
	MaxOutput      int      `json:"max_output"`
	InputPrice     float64  `json:"input_price"`
	OutputPrice    float64  `json:"output_price"`
	PriceUnit      string   `json:"price_unit"`
	EffortLevels   []string `json:"effort_levels,omitempty"`
}

// ProbeMetadata is the API-facing descriptor for a channel probe.
type ProbeMetadata struct {
	ID             string   `json:"id"`
	Label          string   `json:"label"`
	Tags           []string `json:"tags,omitempty"`
	EstTokens      int      `json:"est_tokens"`
	Checks         []string `json:"checks,omitempty"`
	DocPath        string   `json:"doc_path"`
	Required       bool     `json:"required"`
	NeedsThinking  bool     `json:"needs_thinking"`
	NeedsSignature bool     `json:"needs_signature"`
}

// CheckMetadata is the API-facing descriptor for a channel check.
type CheckMetadata struct {
	Name       string   `json:"name"`
	Label      string   `json:"label"`
	Category   Category `json:"category"`
	DefaultFix Fix      `json:"default_fix,omitempty"`
	DocPath    string   `json:"doc_path"`
}

// ProbeDocPath returns the canonical documentation path for a probe.
func ProbeDocPath(id string) string {
	if id == "" {
		return ""
	}
	return channelProbeDocBase + id + ".md"
}

// CheckDocPath returns the canonical documentation path for a check.
func CheckDocPath(name string) string {
	if name == "" {
		return ""
	}
	for _, p := range AllProbes() {
		if p == nil {
			continue
		}
		for _, check := range p.Checks {
			if check == name {
				return ProbeDocPath(p.ID)
			}
		}
	}
	return "docs/detection-model.md"
}

// ListModelMetadata returns supported model capabilities in stable display order.
func ListModelMetadata() []ModelMetadata {
	ids := supportedModelIDsStable()
	out := make([]ModelMetadata, 0, len(ids))
	for _, id := range ids {
		caps := GetModelCaps(id)
		out = append(out, ModelMetadata{
			ID:             id,
			Family:         caps.Family,
			Generation:     caps.Generation,
			Thinking:       caps.Thinking,
			ThinkingMode:   caps.ThinkingMode,
			Signatures:     caps.Signatures,
			DisplayDefault: caps.DisplayDefault,
			MaxOutput:      caps.MaxOutput,
			InputPrice:     caps.InputPrice,
			OutputPrice:    caps.OutputPrice,
			PriceUnit:      "USD per 1M tokens",
			EffortLevels:   copyStrings(caps.EffortLevels),
		})
	}
	return out
}

// ListProbeMetadata returns every registered channel probe in execution order.
func ListProbeMetadata() []ProbeMetadata {
	probes := AllProbes()
	out := make([]ProbeMetadata, 0, len(probes))
	for _, p := range probes {
		if p == nil {
			continue
		}
		out = append(out, ProbeMetadata{
			ID:             p.ID,
			Label:          p.Label,
			Tags:           copyStrings(p.Tags),
			EstTokens:      p.EstTokens,
			Checks:         copyStrings(p.Checks),
			DocPath:        ProbeDocPath(p.ID),
			Required:       p.Required,
			NeedsThinking:  p.NeedsThinking,
			NeedsSignature: p.NeedsSignature,
		})
	}
	return out
}

// ListCheckMetadata returns every registered check in stable category/name order.
func ListCheckMetadata() []CheckMetadata {
	keys := make([]string, 0, len(checkRegistry))
	for name := range checkRegistry {
		keys = append(keys, name)
	}

	rank := categoryRank()
	sort.Slice(keys, func(i, j int) bool {
		left := checkRegistry[keys[i]]
		right := checkRegistry[keys[j]]
		if rank[left.Category] != rank[right.Category] {
			return rank[left.Category] < rank[right.Category]
		}
		return left.Name < right.Name
	})

	out := make([]CheckMetadata, 0, len(keys))
	for _, name := range keys {
		meta := checkRegistry[name]
		out = append(out, CheckMetadata{
			Name:       meta.Name,
			Label:      meta.Label,
			Category:   meta.Category,
			DefaultFix: meta.DefaultFix,
			DocPath:    CheckDocPath(meta.Name),
		})
	}
	return out
}

func supportedModelIDsStable() []string {
	ids := SupportedModels()
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		seen[id] = true
	}

	var missing []string
	for id := range knownModels {
		if !seen[id] {
			missing = append(missing, id)
		}
	}
	sort.Strings(missing)
	ids = append(copyStrings(ids), missing...)
	return ids
}

func categoryRank() map[Category]int {
	rank := make(map[Category]int, len(categoryOrder))
	for i, cat := range categoryOrder {
		rank[cat.Key] = i
	}
	return rank
}

func copyStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}
