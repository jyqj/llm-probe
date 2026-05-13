package channeltest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// referenceFile represents a captured cctest response file.
type referenceFile struct {
	Headers map[string]string `json:"headers"`
	Body    map[string]any    `json:"body"`
}

func (r *referenceFile) httpHeaders() http.Header {
	h := http.Header{}
	for k, v := range r.Headers {
		if k == "_status" {
			continue
		}
		h.Set(k, v)
	}
	return h
}

// testSpec maps test file names to the checks that should be run against them.
type testSpec struct {
	name       string
	runChecks  func(body map[string]any, headers http.Header, model string) []CheckResult
	needsTag   bool   // tag_replay needs the tag from the prompt
	promptFile string // relative prompt file name (optional)
}

var testSpecs = []testSpec{
	{
		name: "08_mini_probe",
		runChecks: func(body map[string]any, headers http.Header, model string) []CheckResult {
			var checks []CheckResult
			checks = append(checks, checkBackendType(body))
			checks = append(checks, checkSmallProbeExact(body)...)
			checks = append(checks, checkTokenBudget(body, model))
			return checks
		},
	},
	{
		name: "09_hidden_prompt",
		runChecks: func(body map[string]any, headers http.Header, model string) []CheckResult {
			return []CheckResult{checkHiddenPrompt(body)}
		},
	},
	{
		name: "03_identity_probe",
		runChecks: func(body map[string]any, headers http.Header, model string) []CheckResult {
			var checks []CheckResult
			checks = append(checks, checkNonStreamBody(body)...)
			checks = append(checks, checkIDFormat(body))
			checks = append(checks, checkIdentityResponse(body))
			checks = append(checks, checkIdentityNoLeak(body))
			checks = append(checks, checkIdentityPlatform(body))
			checks = append(checks, checkPoisonAnswer(body))
			checks = append(checks, checkStopSequenceNull(body))
			checks = append(checks, checkServiceTier(body))
			checks = append(checks, checkSignatureTypeLeak(body))
			checks = append(checks, checkUsageFieldsComplete(body))
			return checks
		},
	},
	{
		name: "00_precheck",
		runChecks: func(body map[string]any, headers http.Header, model string) []CheckResult {
			var checks []CheckResult
			checks = append(checks, checkHeaders(headers))
			checks = append(checks, checkRequestID(headers))
			checks = append(checks, checkXNewApiVersion(headers))
			checks = append(checks, checkServerHeader(headers))
			checks = append(checks, checkContainer(body))
			checks = append(checks, checkBedrockState(body))
			return checks
		},
	},
	{
		name: "01_tag_replay",
		runChecks: func(body map[string]any, headers http.Header, model string) []CheckResult {
			var checks []CheckResult
			checks = append(checks, checkIDFormat(body))
			checks = append(checks, checkModelName(body, model))
			checks = append(checks, checkSignature(body))
			checks = append(checks, checkSignatureLength(body))
			checks = append(checks, checkThinkingPresent(body))
			checks = append(checks, checkUsageStructure(body))
			checks = append(checks, checkInferenceGeo(body, model))
			checks = append(checks, checkStopDetails(body))
			checks = append(checks, checkStopDetailsStructure(body))
			checks = append(checks, checkStopReason(body))
			checks = append(checks, checkStopSequenceNull(body))
			checks = append(checks, checkThinkingOrder(body))
			checks = append(checks, checkThinkingDisplayOmitted(body, model))
			checks = append(checks, checkCacheFake(body))
			checks = append(checks, checkServiceTier(body))
			checks = append(checks, checkSignatureTypeLeak(body))
			checks = append(checks, checkUsageFieldsComplete(body))
			checks = append(checks, checkCacheCreationComplete(body))
			return checks
		},
	},
	{
		name: "02_logic_reasoning",
		runChecks: func(body map[string]any, headers http.Header, model string) []CheckResult {
			return []CheckResult{checkLogicAnswer(body)}
		},
	},
	{
		name: "04_self_intro",
		runChecks: func(body map[string]any, headers http.Header, model string) []CheckResult {
			return []CheckResult{
				checkNoThinkingWhenDisabled(body),
				checkIdentityResponse(body),
				checkStructuredJSONValid(body),
				checkStructuredSchemaMatch(body),
				checkStructuredStopReason(body),
			}
		},
	},
	{
		name: "05_tool_use",
		runChecks: func(body map[string]any, headers http.Header, model string) []CheckResult {
			return []CheckResult{
				checkToolUseID(body),
				checkToolStopReason(body),
				checkToolForcedCompliance(body),
				checkWebSearchResult(body),
				checkStopReason(body),
				checkServerToolType(body),
				checkCitationsPresent(body),
				checkServerToolUsage(body),
			}
		},
	},
	{
		name: "10_magic_refusal",
		runChecks: func(body map[string]any, headers http.Header, model string) []CheckResult {
			return []CheckResult{checkMagicRefusal(body), checkStopReason(body)}
		},
	},
}

const refBaseDir = "../../cctest_reference"

func TestReferenceData(t *testing.T) {
	channels := []string{"azure", "claude-console", "kiro", "windsurf"}
	models := []string{"opus-4-6", "opus-4-7", "sonnet-4-6"}

	// Track summary for final report
	type result struct {
		channel string
		model   string
		test    string
		check   string
		pass    bool
		detail  string
	}
	var allResults []result

	for _, channel := range channels {
		for _, model := range models {
			for _, spec := range testSpecs {
				respPath := filepath.Join(refBaseDir, "responses", channel, model, spec.name+".json")
				raw, err := os.ReadFile(respPath)
				if err != nil {
					continue // file doesn't exist for this combo
				}

				var ref referenceFile
				if err := json.Unmarshal(raw, &ref); err != nil {
					t.Errorf("parse %s: %v", respPath, err)
					continue
				}

				// Skip error responses (e.g. 503 Service Unavailable)
				if _, hasErr := ref.Body["error"]; hasErr {
					continue
				}

				headers := ref.httpHeaders()
				fullModel := "claude-" + model
				checks := spec.runChecks(ref.Body, headers, fullModel)

				for _, c := range checks {
					allResults = append(allResults, result{
						channel: channel,
						model:   model,
						test:    spec.name,
						check:   c.Name,
						pass:    c.Pass,
						detail:  c.Detail,
					})
				}
			}
		}
	}

	// Report: official channels (azure, claude-console) should have zero failures
	officialChannels := map[string]bool{"azure": true, "claude-console": true}
	var officialFails []result
	var thirdPartyFails []result

	for _, r := range allResults {
		if !r.pass {
			if officialChannels[r.channel] {
				officialFails = append(officialFails, r)
			} else {
				thirdPartyFails = append(thirdPartyFails, r)
			}
		}
	}

	// Print summary
	t.Log("=== REFERENCE DATA TEST SUMMARY ===")

	// Count per channel
	channelStats := map[string][2]int{} // [pass, fail]
	for _, r := range allResults {
		s := channelStats[r.channel]
		if r.pass {
			s[0]++
		} else {
			s[1]++
		}
		channelStats[r.channel] = s
	}
	sortedChannels := make([]string, 0, len(channelStats))
	for ch := range channelStats {
		sortedChannels = append(sortedChannels, ch)
	}
	sort.Strings(sortedChannels)
	for _, ch := range sortedChannels {
		s := channelStats[ch]
		total := s[0] + s[1]
		tag := "official"
		if !officialChannels[ch] {
			tag = "3rd-party"
		}
		t.Logf("  %-16s [%s]  %d/%d passed (%.0f%%)", ch, tag, s[0], total, float64(s[0])/float64(total)*100)
	}

	// Official channel failures = test failures
	if len(officialFails) > 0 {
		t.Log("")
		t.Log("=== OFFICIAL CHANNEL FAILURES (should be 0) ===")
		for _, r := range officialFails {
			t.Errorf("  FAIL  %-16s %-10s %-20s %-28s %s", r.channel, r.model, r.test, r.check, r.detail)
		}
	} else {
		t.Log("")
		t.Log("  Official channels: ALL PASS")
	}

	// Third-party failures are informational
	if len(thirdPartyFails) > 0 {
		t.Log("")
		t.Log("=== THIRD-PARTY CHANNEL FAILURES (expected, informational) ===")
		// Group by channel
		byChannel := map[string][]result{}
		for _, r := range thirdPartyFails {
			byChannel[r.channel] = append(byChannel[r.channel], r)
		}
		for _, ch := range []string{"kiro", "windsurf"} {
			fails := byChannel[ch]
			if len(fails) == 0 {
				continue
			}
			t.Logf("  --- %s ---", ch)
			for _, r := range fails {
				t.Logf("  FAIL  %-10s %-20s %-28s %s", r.model, r.test, r.check, r.detail)
			}
		}
	}

	// Score simulation for each channel+model
	t.Log("")
	t.Log("=== SCORE SIMULATION ===")
	type cmKey struct{ channel, model string }
	grouped := map[cmKey][]CheckResult{}
	for _, r := range allResults {
		k := cmKey{r.channel, r.model}
		grouped[k] = append(grouped[k], CheckResult{Name: r.check, Pass: r.pass, Detail: r.detail})
	}
	keys := make([]cmKey, 0, len(grouped))
	for k := range grouped {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].channel != keys[j].channel {
			return keys[i].channel < keys[j].channel
		}
		return keys[i].model < keys[j].model
	})
	for _, k := range keys {
		report := CalculateScore(grouped[k], "full")
		tag := "official"
		if !officialChannels[k.channel] {
			tag = "3rd-party"
		}
		t.Logf("  %-16s %-10s [%s]  Score: %5.1f  Grade: %-3s  Verdict: %s (%s)",
			k.channel, k.model, tag, report.TotalScore, report.Grade, report.Verdict, report.VerdictLabel)

		// Print per-category breakdown
		for _, cat := range report.Categories {
			failNames := []string{}
			for _, c := range cat.Checks {
				if !c.Pass {
					failNames = append(failNames, c.Name)
				}
			}
			failStr := ""
			if len(failNames) > 0 {
				failStr = fmt.Sprintf("  fails: %s", strings.Join(failNames, ", "))
			}
			t.Logf("    %-20s %d/%d (%.0f%%)%s", cat.Label, cat.Passed, cat.Total, cat.Percentage, failStr)
		}
	}
}
