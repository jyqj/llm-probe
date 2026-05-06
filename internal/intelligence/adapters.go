package intelligence

import "sort"

// HFSource describes the default HuggingFace source for a known intelligence.
type HFSource struct {
	Dataset string
	Config  string
	Split   string
}

var knownAdapters = map[string]ColumnAdapter{
	"SWE-bench-Verified": SWEBenchAdapter{},
	"GPQA-Diamond":       GPQAAdapter{},
	"HLE":                HLEAdapter{},
	"LiveCodeBench":      LiveCodeBenchAdapter{},
}

var knownHFSources = map[string]HFSource{
	"SWE-bench-Verified": {Dataset: "princeton-nlp/SWE-bench_Verified", Config: "default", Split: "test"},
	"GPQA-Diamond":       {Dataset: "Idavidrein/gpqa", Config: "default", Split: "train"},
	"HLE":                {Dataset: "cais/hle", Config: "default", Split: "test"},
	"LiveCodeBench":      {Dataset: "livecodebench/code_generation_lite", Config: "release_v5", Split: "test"},
}

// KnownAdapter returns the column adapter for a built-in public intelligence.
func KnownAdapter(name string) (ColumnAdapter, bool) {
	adapter, ok := knownAdapters[name]
	return adapter, ok
}

// KnownHFSource returns the default HuggingFace dataset metadata for a intelligence.
func KnownHFSource(name string) (HFSource, bool) {
	source, ok := knownHFSources[name]
	return source, ok
}

// KnownAdapterNames returns stable sorted names of built-in public intelligence adapters.
func KnownAdapterNames() []string {
	names := make([]string, 0, len(knownAdapters))
	for name := range knownAdapters {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
