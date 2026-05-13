package audit

import (
	"context"
	"time"

	"detector-service/internal/channeltest"
	"detector-service/internal/intelligence"
	"detector-service/internal/target"
)

// Request composes the two product lines: channel authenticity and model capability.
type Request struct {
	Target               target.Config `json:"target"`
	QuickChannel         bool          `json:"quick_channel"`
	IntelligenceDataset  string        `json:"intelligence_dataset,omitempty"`
	IntelligenceLimit    int           `json:"intelligence_limit,omitempty"`
	IntelligenceLanguage string        `json:"intelligence_language,omitempty"`
}

// Report is the combined audit result. It contains no testing logic; it only
// presents the outputs from channeltest and intelligence side by side.
type Report struct {
	Target            target.Config           `json:"target"`
	StartedAt         time.Time               `json:"started_at"`
	ElapsedMs         int64                   `json:"elapsed_ms"`
	Channel           *channeltest.Report     `json:"channel,omitempty"`
	ChannelError      string                  `json:"channel_error,omitempty"`
	Intelligence      *intelligence.RunReport `json:"intelligence,omitempty"`
	IntelligenceError string                  `json:"intelligence_error,omitempty"`
}

// Runner orchestrates channeltest + intelligence without owning either domain's logic.
type Runner struct {
	ChannelStore       *channeltest.Store
	DatasetRegistry    *intelligence.Registry
	IntelligenceRunner *intelligence.Runner
}

func NewRunner(channelStore *channeltest.Store, datasetRegistry *intelligence.Registry, intelligenceRunner *intelligence.Runner) *Runner {
	return &Runner{ChannelStore: channelStore, DatasetRegistry: datasetRegistry, IntelligenceRunner: intelligenceRunner}
}

func (r *Runner) Run(ctx context.Context, req Request) *Report {
	started := time.Now()
	report := &Report{Target: req.Target, StartedAt: started}

	if r.ChannelStore != nil {
		channelReport, err := r.ChannelStore.RunSync(req.Target.BaseURL, req.Target.APIKey, req.Target.Model, "", 2)
		if err != nil {
			report.ChannelError = err.Error()
		} else {
			report.Channel = channelReport
		}
	}

	if r.DatasetRegistry != nil && r.IntelligenceRunner != nil {
		datasetName := req.IntelligenceDataset
		if datasetName == "" {
			names := r.DatasetRegistry.List()
			if len(names) > 0 {
				datasetName = names[0]
			}
		}
		if datasetName != "" {
			if ds, ok := r.DatasetRegistry.Get(datasetName); ok {
				intelligenceReport, err := r.IntelligenceRunner.Run(ctx, ds, intelligence.RunRequest{
					TargetBase: req.Target.BaseURL,
					TargetKey:  req.Target.APIKey,
					Model:      req.Target.Model,
					Language:   req.IntelligenceLanguage,
					Limit:      req.IntelligenceLimit,
				})
				if err != nil {
					report.IntelligenceError = err.Error()
				} else {
					report.Intelligence = intelligenceReport
				}
			} else {
				report.IntelligenceError = "dataset not found: " + datasetName
			}
		}
	}

	report.ElapsedMs = time.Since(started).Milliseconds()
	return report
}
