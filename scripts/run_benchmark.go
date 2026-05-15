// +build ignore

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"detector-service/internal/intelligence"
	"detector-service/internal/intelligence/datasets/sweatlas"
)

type combo struct {
	Model  string
	Effort string // "" means default (no effort param)
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <api-key> <output-dir>\n", os.Args[0])
		os.Exit(1)
	}
	apiKey := os.Args[1]
	outDir := os.Args[2]

	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	ds, err := sweatlas.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load dataset: %v\n", err)
		os.Exit(1)
	}

	combos := []combo{
		// haiku: no effort levels
		{"claude-haiku-4-5", ""},
		// sonnet-4-6
		{"claude-sonnet-4-6", ""},
		{"claude-sonnet-4-6", "low"},
		{"claude-sonnet-4-6", "medium"},
		{"claude-sonnet-4-6", "high"},
		{"claude-sonnet-4-6", "max"},
		// opus-4-5
		{"claude-opus-4-5", ""},
		{"claude-opus-4-5", "low"},
		{"claude-opus-4-5", "medium"},
		{"claude-opus-4-5", "high"},
		// opus-4-6
		{"claude-opus-4-6", ""},
		{"claude-opus-4-6", "low"},
		{"claude-opus-4-6", "medium"},
		{"claude-opus-4-6", "high"},
		{"claude-opus-4-6", "max"},
		// opus-4-7
		{"claude-opus-4-7", ""},
		{"claude-opus-4-7", "low"},
		{"claude-opus-4-7", "medium"},
		{"claude-opus-4-7", "high"},
		{"claude-opus-4-7", "xhigh"},
		{"claude-opus-4-7", "max"},
	}

	runner := intelligence.NewRunner(nil)
	totalCombos := len(combos)

	for i, c := range combos {
		effortLabel := c.Effort
		if effortLabel == "" {
			effortLabel = "default"
		}
		filename := fmt.Sprintf("%s_%s.json", c.Model, effortLabel)
		outPath := filepath.Join(outDir, filename)

		// Skip if already completed
		if info, err := os.Stat(outPath); err == nil && info.Size() > 100 {
			fmt.Fprintf(os.Stderr, "[%d/%d] SKIP %s %s (already exists)\n", i+1, totalCombos, c.Model, effortLabel)
			continue
		}

		fmt.Fprintf(os.Stderr, "[%d/%d] START %s effort=%s ...\n", i+1, totalCombos, c.Model, effortLabel)
		startTime := time.Now()

		req := intelligence.RunRequest{
			TargetBase:  "https://api.anthropic.com",
			TargetKey:   apiKey,
			Model:       c.Model,
			Thinking:    true,
			Effort:      c.Effort,
			MaxTokens:   16384,
			Concurrency: 1,
		}

		ctx := context.Background()
		completed := 0
		report, err := runner.RunStream(ctx, ds, req, func(ev intelligence.StreamEvent) {
			if ev.Type == "progress" {
				completed++
				if completed%10 == 0 || completed == ev.Total {
					elapsed := time.Since(startTime)
					fmt.Fprintf(os.Stderr, "  [%d/%d] %d/%d tasks (%.0fs)\n",
						i+1, totalCombos, completed, ev.Total, elapsed.Seconds())
				}
			}
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR: %v\n", err)
			continue
		}

		out, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(outPath, out, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "  write error: %v\n", err)
			continue
		}

		scoreStr := "N/A"
		if report.ScoreTotal != nil {
			scoreStr = fmt.Sprintf("%.1f", *report.ScoreTotal)
		}
		passStr := "N/A"
		if report.PassRate != nil {
			passStr = fmt.Sprintf("%.1f%%", *report.PassRate)
		}

		elapsed := time.Since(startTime)
		fmt.Fprintf(os.Stderr, "[%d/%d] DONE %s effort=%s | score=%s pass=%s | %d/%d tasks | %.0fs\n",
			i+1, totalCombos, c.Model, effortLabel,
			scoreStr, passStr,
			report.TaskCompleted, report.TaskTotal, elapsed.Seconds())
	}

	fmt.Fprintf(os.Stderr, "\nAll combinations completed. Results in %s\n", outDir)
}
