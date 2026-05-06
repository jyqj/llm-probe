package sweatlas

import (
	"embed"
	"fmt"
	"io"

	"detector-service/internal/intelligence"
)

const (
	DatasetName    = "SWE-Atlas-QnA"
	DatasetURL     = "https://huggingface.co/datasets/ScaleAI/SWE-Atlas-QnA"
	DatasetVersion = "hf-main-2026-05-06"
	csvPath        = "data/cae_qna_124_public.csv"
)

//go:embed data/cae_qna_124_public.csv
var datasetFS embed.FS

// Load returns the embedded SWE-Atlas-QnA dataset as a generic Dataset.
func Load() (intelligence.Dataset, error) {
	f, err := datasetFS.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("open embedded swe-atlas csv: %w", err)
	}
	defer f.Close()
	return LoadCSV(f)
}

// LoadCSV loads from a reader using the generic CSV loader.
func LoadCSV(r io.Reader) (intelligence.Dataset, error) {
	return intelligence.LoadCSV(r, DatasetName, DatasetVersion, DatasetURL)
}
