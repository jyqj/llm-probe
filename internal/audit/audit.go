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
	OverallStatus     string                  `json:"overall_status"`
	OverallScore      float64                 `json:"overall_score"`
	Channel           *channeltest.Report     `json:"channel,omitempty"`
	ChannelStatus     string                  `json:"channel_status,omitempty"`
	ChannelError      string                  `json:"channel_error,omitempty"`
	Intelligence      *intelligence.RunReport `json:"intelligence,omitempty"`
	IntelligenceStatus string                 `json:"intelligence_status,omitempty"`
	IntelligenceError string                  `json:"intelligence_error,omitempty"`
	Recommendations   []string                `json:"recommendations,omitempty"`
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
		channelReport, err := r.ChannelStore.RunSync(req.Target.BaseURL, req.Target.APIKey, req.Target.Model, "", 2, "")
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

	// Derive overall status and score
	report.OverallStatus, report.OverallScore, report.ChannelStatus, report.IntelligenceStatus, report.Recommendations = deriveOverall(report)
	return report
}

func deriveOverall(r *Report) (status string, score float64, chStatus, intStatus string, recs []string) {
	chStatus = "skipped"
	intStatus = "skipped"
	var scores []float64

	if r.ChannelError != "" {
		chStatus = "error"
	} else if r.Channel != nil && r.Channel.Score != nil {
		s := r.Channel.Score.TotalScore
		scores = append(scores, s)
		switch {
		case s >= 80:
			chStatus = "ok"
		case s >= 50:
			chStatus = "warning"
			recs = append(recs, "渠道得分偏低,建议检查渠道配置")
		default:
			chStatus = "critical"
			recs = append(recs, "渠道得分极低,疑似非官方渠道")
		}
	}

	if r.IntelligenceError != "" {
		intStatus = "error"
	} else if r.Intelligence != nil {
		rate := 0.0
		if r.Intelligence.TaskTotal > 0 {
			rate = float64(r.Intelligence.TaskCompleted-r.Intelligence.TaskErrors) / float64(r.Intelligence.TaskTotal) * 100
		}
		scores = append(scores, rate)
		switch {
		case r.Intelligence.TaskErrors == 0:
			intStatus = "ok"
		case r.Intelligence.TaskErrors <= r.Intelligence.TaskTotal/10:
			intStatus = "warning"
			recs = append(recs, "Benchmark 有少量错误,建议检查具体失败项")
		default:
			intStatus = "critical"
			recs = append(recs, "Benchmark 大量失败,模型能力存疑")
		}
	}

	if len(scores) > 0 {
		for _, s := range scores {
			score += s
		}
		score /= float64(len(scores))
	}

	switch {
	case chStatus == "critical" || intStatus == "critical":
		status = "critical"
	case chStatus == "warning" || intStatus == "warning":
		status = "warning"
	case chStatus == "error" || intStatus == "error":
		status = "warning"
	default:
		status = "ok"
	}
	return
}
