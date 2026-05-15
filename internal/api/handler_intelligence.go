package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"detector-service/internal/intelligence"
	"detector-service/internal/target"
)

// GET /api/intelligence/datasets → list all registered datasets
func (a *API) handleIntelligenceList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"datasets": a.intelligenceRegistry.ListStats(),
	})
}

// /api/intelligence/datasets/{name}[/action] → route to appropriate handler
func (a *API) handleIntelligenceDatasetRoute(w http.ResponseWriter, r *http.Request) {
	// Parse: /api/intelligence/datasets/{name}[/{action}]
	path := strings.TrimPrefix(r.URL.Path, "/api/intelligence/datasets/")
	parts := strings.SplitN(path, "/", 2)
	dsName := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	// Upload doesn't require the dataset to exist yet
	if action == "upload" && r.Method == http.MethodPost {
		a.handleDatasetUpload(w, r, dsName)
		return
	}

	ds, ok := a.intelligenceRegistry.Get(dsName)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error":     "dataset not found: " + dsName,
			"available": a.intelligenceRegistry.List(),
		})
		return
	}

	switch {
	case action == "" && r.Method == http.MethodGet:
		a.handleDatasetInfo(w, r, ds)
	case action == "tasks" && r.Method == http.MethodGet:
		a.handleDatasetTasks(w, r, ds)
	case strings.HasPrefix(action, "tasks/") && r.Method == http.MethodGet:
		taskID := strings.TrimPrefix(action, "tasks/")
		a.handleDatasetTask(w, r, ds, taskID)
	case action == "run" && r.Method == http.MethodPost:
		a.handleDatasetRun(w, r, ds)
	case action == "stream" && r.Method == http.MethodPost:
		a.handleDatasetStream(w, r, ds)
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error":   "unknown action",
			"actions": []string{"GET info", "GET tasks", "GET tasks/{id}", "POST run", "POST stream"},
		})
	}
}

func (a *API) handleDatasetInfo(w http.ResponseWriter, r *http.Request, ds intelligence.Dataset) {
	writeJSON(w, http.StatusOK, map[string]any{
		"stats":      ds.Stats(),
		"languages":  intelligence.UniqueSorted(ds.Stats().Languages),
		"categories": intelligence.UniqueSorted(ds.Stats().Categories),
	})
}

func (a *API) handleDatasetTasks(w http.ResponseWriter, r *http.Request, ds intelligence.Dataset) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	includeRubric := r.URL.Query().Get("rubric") == "1" || strings.EqualFold(r.URL.Query().Get("rubric"), "true")
	tasks := ds.Filter(intelligence.Filter{
		Language: r.URL.Query().Get("language"),
		Category: r.URL.Query().Get("category"),
		Limit:    limit,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"total": len(tasks),
		"tasks": intelligence.Summaries(tasks, includeRubric),
	})
}

func (a *API) handleDatasetTask(w http.ResponseWriter, r *http.Request, ds intelligence.Dataset, taskID string) {
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "task id is required"})
		return
	}
	task, ok := ds.Find(taskID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "task not found"})
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (a *API) handleDatasetRun(w http.ResponseWriter, r *http.Request, ds intelligence.Dataset) {
	var req intelligence.RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json: " + err.Error()})
		return
	}
	t := target.Resolve(a.cfg, req.TargetBase, req.TargetKey, req.Model)
	if err := t.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	req.TargetBase, req.TargetKey, req.Model = t.BaseURL, t.APIKey, t.Model

	lockKey := req.TargetBase + "#" + req.Model
	mu := a.getBenchLock(lockKey)
	if !mu.TryLock() {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "benchmark already running for this target+model"})
		return
	}
	defer mu.Unlock()

	report, err := a.intelligenceRunner.Run(r.Context(), ds, req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	a.applyBaselineComparison(report, req.BaselineID)
	a.intelligenceHistory.Add(report)
	writeJSON(w, http.StatusOK, report)
}

func (a *API) handleDatasetStream(w http.ResponseWriter, r *http.Request, ds intelligence.Dataset) {
	var req intelligence.RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json: " + err.Error()})
		return
	}
	t := target.Resolve(a.cfg, req.TargetBase, req.TargetKey, req.Model)

	// Validate before switching to SSE — after this point we can't send JSON errors.
	if err := t.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	req.TargetBase, req.TargetKey, req.Model = t.BaseURL, t.APIKey, t.Model

	lockKey := req.TargetBase + "#" + req.Model
	benchMu := a.getBenchLock(lockKey)
	if !benchMu.TryLock() {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "benchmark already running for this target+model"})
		return
	}
	defer benchMu.Unlock()

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "streaming not supported"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	mu := &sync.Mutex{}
	done := make(chan struct{})
	writeSSE := func(event intelligence.StreamEvent) {
		data, _ := json.Marshal(event)
		mu.Lock()
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		mu.Unlock()
	}

	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				mu.Lock()
				fmt.Fprintf(w, ": ping\n\n")
				flusher.Flush()
				mu.Unlock()
			case <-done:
				return
			case <-r.Context().Done():
				return
			}
		}
	}()

	var finalEvent *intelligence.StreamEvent
	report, err := a.intelligenceRunner.RunStream(r.Context(), ds, req, func(ev intelligence.StreamEvent) {
		// Runner emits the complete event before the API layer can attach baseline
		// comparison and persist history. Hold only that terminal event here, then
		// send an enriched complete event below so SSE clients and history observe
		// the same final report. Progress/error events keep their original timing.
		if ev.Type == "complete" {
			event := ev
			finalEvent = &event
			return
		}
		writeSSE(ev)
	})
	close(done)
	if err != nil {
		writeSSE(intelligence.StreamEvent{Type: "error", ErrorMsg: err.Error()})
	}
	if report != nil {
		a.applyBaselineComparison(report, req.BaselineID)
		a.intelligenceHistory.Add(report)
		if finalEvent == nil {
			finalEvent = &intelligence.StreamEvent{Type: "complete"}
		}
		finalEvent.Total = report.TaskTotal
		finalEvent.Completed = report.TaskCompleted
		finalEvent.Errors = report.TaskErrors
		finalEvent.Report = report
		writeSSE(*finalEvent)
	}
}

// handleDatasetUpload allows uploading a new dataset via CSV or JSON.
// POST /api/intelligence/datasets/{name}/upload  with multipart form or JSON body.
func (a *API) handleDatasetUpload(w http.ResponseWriter, r *http.Request, dsName string) {
	if dsName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "dataset name is required in URL"})
		return
	}

	contentType := r.Header.Get("Content-Type")
	var ds intelligence.Dataset
	var err error

	switch {
	case strings.Contains(contentType, "application/json"):
		ds, err = a.uploadJSON(r, dsName)
	case strings.Contains(contentType, "multipart/form-data"):
		ds, err = a.uploadMultipart(r, dsName)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Content-Type must be application/json or multipart/form-data"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	a.intelligenceRegistry.Register(ds)
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "dataset registered",
		"stats":   ds.Stats(),
	})
}

func (a *API) uploadJSON(r *http.Request, dsName string) (intelligence.Dataset, error) {
	return intelligence.LoadJSON(r.Body, dsName, "upload", "user-upload")
}

func (a *API) uploadMultipart(r *http.Request, dsName string) (intelligence.Dataset, error) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		return nil, fmt.Errorf("parse multipart: %w", err)
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return nil, fmt.Errorf("missing 'file' in form: %w", err)
	}
	defer file.Close()

	name := r.FormValue("name")
	if name == "" {
		name = dsName
	}
	version := r.FormValue("version")
	if version == "" {
		version = "upload"
	}

	fname := strings.ToLower(header.Filename)
	if strings.HasSuffix(fname, ".json") {
		return intelligence.LoadJSON(file, name, version, "user-upload")
	}
	return intelligence.LoadCSV(file, name, version, "user-upload")
}

// applyBaselineComparison attaches baseline comparison data to the report if a baseline ID is provided.
func (a *API) applyBaselineComparison(report *intelligence.RunReport, baselineID string) {
	if baselineID == "" || a.baselineStore == nil || report == nil {
		return
	}
	baseline := a.baselineStore.Get(baselineID)
	if baseline == nil || baseline.IntelligenceReport == nil {
		return
	}
	report.CompareToBaseline(baseline.ID, baseline.Name, baseline.IntelligenceReport)
}

func (a *API) handleIntelligenceHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	all := a.intelligenceHistory.List()
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	total := len(all)
	if offset > total {
		offset = total
	}
	all = all[offset:]
	if limit > 0 && limit < len(all) {
		all = all[:limit]
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": all, "total": total})
}

func (a *API) handleIntelligenceHistoryDetail(w http.ResponseWriter, r *http.Request) {
	remainder := strings.TrimPrefix(r.URL.Path, "/api/intelligence/history/")
	if remainder == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "id is required"})
		return
	}

	// Sub-route: /api/intelligence/history/{id}/retry
	if strings.HasSuffix(remainder, "/retry") {
		id := strings.TrimSuffix(remainder, "/retry")
		a.handleIntelligenceRetry(w, r, id)
		return
	}

	id := remainder
	if r.Method == http.MethodDelete {
		if a.intelligenceHistory.Delete(id) {
			writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
		} else {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		}
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	report := a.intelligenceHistory.Get(id)
	if report == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// handleIntelligenceRetry re-runs selected tasks in a completed report.
// POST /api/intelligence/history/{id}/retry
// Body: {"task_indices": [0,3,5], "target_key": "sk-...", "failed": false}
// If "failed" is true, all errored/failed tasks are retried.
func (a *API) handleIntelligenceRetry(w http.ResponseWriter, r *http.Request, reportID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		TaskIndices []int  `json:"task_indices"`
		Failed      bool   `json:"failed"`
		TargetKey   string `json:"target_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json: " + err.Error()})
		return
	}
	if body.TargetKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "target_key is required"})
		return
	}

	report := a.intelligenceHistory.Get(reportID)
	if report == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "report not found"})
		return
	}

	ds, ok := a.intelligenceRegistry.Get(report.DatasetName)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "dataset not found: " + report.DatasetName})
		return
	}

	indices := body.TaskIndices
	if body.Failed {
		indices = nil
		for i, res := range report.Results {
			if res.Error != "" || (res.Pass != nil && !*res.Pass) {
				indices = append(indices, i)
			}
		}
	}
	if len(indices) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"retried": 0, "report": report})
		return
	}

	req := intelligence.RunRequest{
		TargetBase:   report.Target,
		TargetKey:    body.TargetKey,
		Model:        report.Model,
		Thinking:     report.Thinking,
		Effort:       report.Effort,
		ThinkingMode: report.ThinkingMode,
		MaxTokens:    report.MaxTokens,
	}

	var retried []intelligence.TaskRunResult
	for _, idx := range indices {
		if idx < 0 || idx >= len(report.Results) {
			continue
		}
		taskID := report.Results[idx].Task.TaskID
		task, found := ds.Find(taskID)
		if !found {
			continue
		}
		result := a.intelligenceRunner.RunSingleTask(r.Context(), req, task)
		result.Index = idx
		a.intelligenceHistory.UpdateResult(report, idx, result)
		retried = append(retried, result)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"retried": len(retried),
		"results": retried,
		"report":  report,
	})
}
