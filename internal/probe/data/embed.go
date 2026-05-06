package data

import (
	_ "embed"
	"encoding/json"
)

//go:embed system_prompt.txt
var SystemPrompt string

//go:embed tools.json
var toolsJSON []byte

// Tools returns the full 28 Claude Code tools array parsed from embedded JSON.
func Tools() []any {
	var tools []any
	json.Unmarshal(toolsJSON, &tools)
	return tools
}
