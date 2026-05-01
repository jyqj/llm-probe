package admin

import (
	"sync"
	"time"
)

// RequestLog represents a single API request log entry.
type RequestLog struct {
	ID            string        `json:"id"`
	Timestamp     time.Time     `json:"timestamp"`
	Method        string        `json:"method"`
	Path          string        `json:"path"`
	ClientIP      string        `json:"client_ip"`
	Model         string        `json:"model"`
	BedrockModel  string        `json:"bedrock_model"`
	Stream        bool          `json:"stream"`
	InputTokens   int           `json:"input_tokens"`
	OutputTokens  int           `json:"output_tokens"`
	StatusCode    int           `json:"status_code"`
	Latency       time.Duration `json:"latency_ms"`
	Error         string        `json:"error,omitempty"`
	Sanitized     bool          `json:"sanitized"`
	SanitizeWarns []string      `json:"sanitize_warns,omitempty"`
	Route         string        `json:"route"` // "invoke", "converse", "converse-messages"
}

// Stats holds aggregate statistics.
type Stats struct {
	TotalRequests   int64   `json:"total_requests"`
	SuccessRequests int64   `json:"success_requests"`
	ErrorRequests   int64   `json:"error_requests"`
	TotalInputToks  int64   `json:"total_input_tokens"`
	TotalOutputToks int64   `json:"total_output_tokens"`
	AvgLatencyMs    float64 `json:"avg_latency_ms"`
	RequestsPerMin  float64 `json:"requests_per_min"`
	ActiveStreams   int64   `json:"active_streams"`

	// Per-model stats
	ModelStats map[string]*ModelStats `json:"model_stats"`

	// Per-minute request counts for chart (last 60 minutes)
	RequestsTimeline []TimePoint `json:"requests_timeline"`
}

// ModelStats per-model statistics.
type ModelStats struct {
	Requests    int64   `json:"requests"`
	InputToks   int64   `json:"input_tokens"`
	OutputToks  int64   `json:"output_tokens"`
	AvgLatency  float64 `json:"avg_latency_ms"`
	Errors      int64   `json:"errors"`
	totalLatMs  int64
}

// TimePoint for timeline charts.
type TimePoint struct {
	Time  time.Time `json:"time"`
	Count int64     `json:"count"`
}

// RequestLogger stores and queries request logs with stats.
type RequestLogger struct {
	mu            sync.RWMutex
	logs          []RequestLog
	maxLogs       int
	stats         Stats
	activeStreams int64
	startTime     time.Time
	minuteBuckets map[int64]int64 // unix minute -> count
}

// NewRequestLogger creates a logger with a max log buffer size.
func NewRequestLogger(maxLogs int) *RequestLogger {
	return &RequestLogger{
		logs:          make([]RequestLog, 0, maxLogs),
		maxLogs:       maxLogs,
		startTime:     time.Now(),
		minuteBuckets: make(map[int64]int64),
		stats: Stats{
			ModelStats: make(map[string]*ModelStats),
		},
	}
}

// Log records a new request.
func (rl *RequestLogger) Log(entry RequestLog) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Append log, evict oldest if full
	if len(rl.logs) >= rl.maxLogs {
		rl.logs = rl.logs[1:]
	}
	rl.logs = append(rl.logs, entry)

	// Update stats
	rl.stats.TotalRequests++
	if entry.StatusCode >= 200 && entry.StatusCode < 400 {
		rl.stats.SuccessRequests++
	} else {
		rl.stats.ErrorRequests++
	}

	rl.stats.TotalInputToks += int64(entry.InputTokens)
	rl.stats.TotalOutputToks += int64(entry.OutputTokens)

	if rl.stats.TotalRequests > 0 {
		// Running average latency
		prevTotal := rl.stats.AvgLatencyMs * float64(rl.stats.TotalRequests-1)
		rl.stats.AvgLatencyMs = (prevTotal + float64(entry.Latency.Milliseconds())) / float64(rl.stats.TotalRequests)
	}

	// Per-model stats
	model := entry.BedrockModel
	if model == "" {
		model = entry.Model
	}
	if _, ok := rl.stats.ModelStats[model]; !ok {
		rl.stats.ModelStats[model] = &ModelStats{}
	}
	ms := rl.stats.ModelStats[model]
	ms.Requests++
	ms.InputToks += int64(entry.InputTokens)
	ms.OutputToks += int64(entry.OutputTokens)
	ms.totalLatMs += entry.Latency.Milliseconds()
	if ms.Requests > 0 {
		ms.AvgLatency = float64(ms.totalLatMs) / float64(ms.Requests)
	}
	if entry.StatusCode >= 400 {
		ms.Errors++
	}

	// Minute bucket
	minuteKey := entry.Timestamp.Unix() / 60
	rl.minuteBuckets[minuteKey]++

	// Requests per minute
	elapsed := time.Since(rl.startTime).Minutes()
	if elapsed > 0 {
		rl.stats.RequestsPerMin = float64(rl.stats.TotalRequests) / elapsed
	}
}

// AddActiveStream increments active stream count.
func (rl *RequestLogger) AddActiveStream() {
	rl.mu.Lock()
	rl.activeStreams++
	rl.mu.Unlock()
}

// RemoveActiveStream decrements active stream count.
func (rl *RequestLogger) RemoveActiveStream() {
	rl.mu.Lock()
	if rl.activeStreams > 0 {
		rl.activeStreams--
	}
	rl.mu.Unlock()
}

// GetStats returns current statistics.
func (rl *RequestLogger) GetStats() Stats {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	stats := rl.stats
	stats.ActiveStreams = rl.activeStreams

	// Build timeline (last 60 minutes)
	now := time.Now()
	stats.RequestsTimeline = make([]TimePoint, 60)
	for i := 59; i >= 0; i-- {
		t := now.Add(-time.Duration(i) * time.Minute)
		minuteKey := t.Unix() / 60
		stats.RequestsTimeline[59-i] = TimePoint{
			Time:  t.Truncate(time.Minute),
			Count: rl.minuteBuckets[minuteKey],
		}
	}

	// Deep copy model stats
	stats.ModelStats = make(map[string]*ModelStats, len(rl.stats.ModelStats))
	for k, v := range rl.stats.ModelStats {
		cp := *v
		stats.ModelStats[k] = &cp
	}

	return stats
}

// GetLogs returns recent logs, newest first. limit=0 means all.
func (rl *RequestLogger) GetLogs(limit int, offset int) ([]RequestLog, int) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	total := len(rl.logs)
	if total == 0 {
		return nil, 0
	}

	// Reverse order (newest first)
	reversed := make([]RequestLog, total)
	for i, log := range rl.logs {
		reversed[total-1-i] = log
	}

	if offset >= total {
		return nil, total
	}

	end := offset + limit
	if limit == 0 || end > total {
		end = total
	}

	return reversed[offset:end], total
}
