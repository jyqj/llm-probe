package intelligence

import "sort"

// HFSource describes the default HuggingFace source for a known intelligence dataset.
type HFSource struct {
	Dataset string
	Config  string
	Split   string
}

// knownAdapters maps dataset names to column adapters for HuggingFace loading.
var knownAdapters = map[string]ColumnAdapter{
	"SWE-Atlas-QnA": sweAtlasAdapter{},
}

var knownHFSources = map[string]HFSource{
	"SWE-Atlas-QnA": {
		Dataset: "ScaleAI/SWE-Atlas-QnA",
		Config:  "default",
		Split:   "test",
	},
}

// sweAtlasAdapter converts SWE-Atlas-QnA HF rows to Tasks.
type sweAtlasAdapter struct{}

func (sweAtlasAdapter) DatasetName() string    { return "SWE-Atlas-QnA" }
func (sweAtlasAdapter) DatasetVersion() string { return "hf-live" }
func (sweAtlasAdapter) ToTask(row map[string]any) Task {
	return Task{
		TaskID:          strVal(row, "task_id"),
		Prompt:          strVal(row, "prompt"),
		ReferenceAnswer: strVal(row, "reference_answer"),
		Language:        strVal(row, "language"),
		Category:        strVal(row, "category"),
	}
}

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
