package channeltest

import (
	"net/url"
	"strings"
)

// ProbeFunc is the signature for a probe's execution logic.
type ProbeFunc func(r *Runner, base, key, model string) ([]CheckResult, error)

// Probe is the self-contained definition of a single test probe.
// Each probe knows its identity, cost, checks, and how to run.
type Probe struct {
	ID             string    // unique identifier, matches phase name
	Label          string    // human-readable Chinese description
	Required       bool      // must run in every suite execution
	Tags           []string  // "monitor" = lightweight periodic, "heavy" = high token cost
	EstTokens      int       // estimated input tokens per invocation
	Checks         []string  // check names this probe produces
	NeedsThinking  bool      // requires model with thinking support
	NeedsSignature bool      // requires model with signature support
	OnlyModels     []string  // if set, only run on these model prefixes (e.g. "claude-opus-4-6")
	Run            ProbeFunc // execution logic
}

// HasTag returns true if the probe has the given tag.
func (p *Probe) HasTag(tag string) bool {
	for _, t := range p.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// AllProbes returns every registered probe in execution order.
func AllProbes() []*Probe {
	return allProbes
}

// FilterProbes returns probes matching the given tag.
func FilterProbes(tag string) []*Probe {
	var out []*Probe
	for _, p := range allProbes {
		if p.HasTag(tag) {
			out = append(out, p)
		}
	}
	return out
}

// RequiredProbes returns probes that must run.
func RequiredProbes() []*Probe {
	var out []*Probe
	for _, p := range allProbes {
		if p.Required {
			out = append(out, p)
		}
	}
	return out
}

// OptionalProbes returns probes that are not required.
func OptionalProbes() []*Probe {
	var out []*Probe
	for _, p := range allProbes {
		if !p.Required {
			out = append(out, p)
		}
	}
	return out
}

// EstimateCost returns the total estimated input tokens for a set of probes.
func EstimateCost(probes []*Probe) int {
	total := 0
	for _, p := range probes {
		total += p.EstTokens
	}
	return total
}

// ProbesForModel returns probes applicable to a given model, respecting capabilities.
func ProbesForModel(model string) []*Probe {
	caps := GetModelCaps(model)
	var out []*Probe
	for _, p := range allProbes {
		if p.NeedsThinking && !caps.Thinking {
			continue
		}
		if p.NeedsSignature && !caps.Signatures {
			continue
		}
		if len(p.OnlyModels) > 0 {
			matched := false
			for _, m := range p.OnlyModels {
				if strings.HasPrefix(model, m) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		out = append(out, p)
	}
	return out
}

// AutoChannelName generates a display name from target URL + model.
func AutoChannelName(targetBase, model string) string {
	u, err := url.Parse(targetBase)
	if err != nil || u.Host == "" {
		return targetBase + " / " + model
	}
	return u.Host + " / " + model
}

// allProbes is the ordered list of all probes.
// Each probe is defined in its own phase_*.go file.
var allProbes = []*Probe{
	probePrecheck,
	probeTagReplay,
	probeMiniProbe,
	probeIdentity,
	probeSelfIntro,
	probeToolUse,
	probeLogic,
	probeHiddenPrompt,
	probeImageOCR,
	probePDFExtract,
	probeMagicRefusal,
	probeEffortThinking,
	probeSignatureReject,
	probeBashTool,
	probeMinimalTokens,
}
