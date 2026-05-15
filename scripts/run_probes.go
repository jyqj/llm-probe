// +build ignore

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"detector-service/internal/channeltest"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <api-key> <model> [target-url]\n", os.Args[0])
		os.Exit(1)
	}
	apiKey := os.Args[1]
	model := os.Args[2]
	target := "https://api.anthropic.com"
	if len(os.Args) >= 4 {
		target = os.Args[3]
	}

	runner := channeltest.NewRunner()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	fmt.Fprintf(os.Stderr, "Running probes against %s with model %s ...\n", target, model)

	report, err := runner.RunCtx(ctx, target, apiKey, model, 0, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	out, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(out))
}
