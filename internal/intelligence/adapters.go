package intelligence

import "sort"

// HFSource describes the default HuggingFace source for a known intelligence dataset.
type HFSource struct {
	Dataset string
	Config  string
	Split   string
}

// Currently only SWE-Atlas-QnA is supported as a built-in dataset (embedded CSV).
// Additional adapters can be added here when new benchmarks are needed.
var knownAdapters = map[string]ColumnAdapter{}

var knownHFSources = map[string]HFSource{}

// KnownAdapter returns the column adapter for a built-in public intelligence dataset.
func KnownAdapter(name string) (ColumnAdapter, bool) {
	adapter, ok := knownAdapters[name]
	return adapter, ok
}

// KnownHFSource returns the default HuggingFace dataset metadata.
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
