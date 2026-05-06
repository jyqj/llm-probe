package detectapi

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"detector-service/internal/benchmark"
	"detector-service/internal/config"
	"detector-service/internal/probe"
)

type API struct {
	cfg        *config.Config
	logger     *slog.Logger
	probeStore *probe.Store
	registry   *benchmark.Registry
	runner     *benchmark.Runner
}

func New(cfg *config.Config, logger *slog.Logger, probeStore *probe.Store, registry *benchmark.Registry, runner *benchmark.Runner) *API {
	return &API{cfg: cfg, logger: logger, probeStore: probeStore, registry: registry, runner: runner}
}

func (a *API) RegisterRoutes(mux *http.ServeMux) {
	// Detection
	mux.HandleFunc("/api/detect/structure", a.adminAuth(a.handleStructureDetect))
	mux.HandleFunc("/api/detect/report", a.adminAuth(a.handleStructureReport))

	// Generic benchmark routes: /api/benchmark → list all; /api/benchmark/{name}/*
	mux.HandleFunc("/api/benchmark", a.adminAuth(a.handleBenchmarkList))
	mux.HandleFunc("/api/benchmark/", a.adminAuth(a.handleBenchmarkRoute))

	// Fetch from HuggingFace
	mux.HandleFunc("/api/benchmark-fetch", a.adminAuth(a.handleBenchmarkFetch))

	// Combined audit
	mux.HandleFunc("/api/audit", a.adminAuth(a.handleAudit))
}

func (a *API) adminAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(a.cfg.Admin.Token)
		if token != "" && r.Header.Get("X-Admin-Token") != token {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "unauthorized"})
			return
		}
		h(w, r)
	}
}

// ═══════════════════════════════════════════
// Detection endpoints (unchanged)
// ═══════════════════════════════════════════

func (a *API) handleStructureDetect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		TargetBase string `json:"target_base"`
		TargetKey  string `json:"target_key"`
		Model      string `json:"model"`
		Quick      bool   `json:"quick"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json: " + err.Error()})
		return
	}
	fillDefaults(a.cfg, &body.TargetBase, &body.TargetKey, &body.Model)
	if body.TargetBase == "" || body.TargetKey == "" || body.Model == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "target_base, target_key and model are required"})
		return
	}
	report, err := a.probeStore.ProbeSync(body.TargetBase, body.TargetKey, body.Model, body.Quick)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (a *API) handleStructureReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	targetBase := strings.TrimSpace(r.URL.Query().Get("target_base"))
	if targetBase != "" {
		entry := a.probeStore.GetEntry(targetBase)
		if entry == nil || entry.Report == nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "no structure report for " + targetBase})
			return
		}
		writeJSON(w, http.StatusOK, entry.Report)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": a.probeStore.ListEntries()})
}

// ═══════════════════════════════════════════
// Generic benchmark endpoints
// ═══════════════════════════════════════════

// GET /api/benchmark → list all registered datasets
func (a *API) handleBenchmarkList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"datasets": a.registry.ListStats(),
	})
}

// /api/benchmark/{name}[/action] → route to appropriate handler
func (a *API) handleBenchmarkRoute(w http.ResponseWriter, r *http.Request) {
	// Parse: /api/benchmark/{name}[/{action}]
	path := strings.TrimPrefix(r.URL.Path, "/api/benchmark/")
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

	ds, ok := a.registry.Get(dsName)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error":     "dataset not found: " + dsName,
			"available": a.registry.List(),
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

func (a *API) handleDatasetInfo(w http.ResponseWriter, r *http.Request, ds benchmark.Dataset) {
	writeJSON(w, http.StatusOK, map[string]any{
		"stats":      ds.Stats(),
		"languages":  benchmark.UniqueSorted(ds.Stats().Languages),
		"categories": benchmark.UniqueSorted(ds.Stats().Categories),
	})
}

func (a *API) handleDatasetTasks(w http.ResponseWriter, r *http.Request, ds benchmark.Dataset) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	includeRubric := r.URL.Query().Get("rubric") == "1" || strings.EqualFold(r.URL.Query().Get("rubric"), "true")
	tasks := ds.Filter(benchmark.Filter{
		Language: r.URL.Query().Get("language"),
		Category: r.URL.Query().Get("category"),
		Limit:    limit,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"total": len(tasks),
		"tasks": benchmark.Summaries(tasks, includeRubric),
	})
}

func (a *API) handleDatasetTask(w http.ResponseWriter, r *http.Request, ds benchmark.Dataset, taskID string) {
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

func (a *API) handleDatasetRun(w http.ResponseWriter, r *http.Request, ds benchmark.Dataset) {
	var req benchmark.RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json: " + err.Error()})
		return
	}
	fillDefaults(a.cfg, &req.TargetBase, &req.TargetKey, &req.Model)
	report, err := a.runner.Run(r.Context(), ds, req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (a *API) handleDatasetStream(w http.ResponseWriter, r *http.Request, ds benchmark.Dataset) {
	var req benchmark.RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json: " + err.Error()})
		return
	}
	fillDefaults(a.cfg, &req.TargetBase, &req.TargetKey, &req.Model)

	// Validate before switching to SSE — after this point we can't send JSON errors.
	if strings.TrimSpace(req.TargetBase) == "" || strings.TrimSpace(req.TargetKey) == "" || strings.TrimSpace(req.Model) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "target_base, target_key and model are required"})
		return
	}

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
	writeSSE := func(event benchmark.StreamEvent) {
		data, _ := json.Marshal(event)
		mu.Lock()
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		mu.Unlock()
	}

	_, err := a.runner.RunStream(r.Context(), ds, req, func(ev benchmark.StreamEvent) {
		writeSSE(ev)
	})
	if err != nil {
		writeSSE(benchmark.StreamEvent{Type: "error", ErrorMsg: err.Error()})
	}
}

// handleDatasetUpload allows uploading a new dataset via CSV or JSON.
// POST /api/benchmark/{name}/upload  with multipart form or JSON body.
func (a *API) handleDatasetUpload(w http.ResponseWriter, r *http.Request, dsName string) {
	if dsName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "dataset name is required in URL"})
		return
	}

	contentType := r.Header.Get("Content-Type")
	var ds benchmark.Dataset
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

	a.registry.Register(ds)
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "dataset registered",
		"stats":   ds.Stats(),
	})
}

func (a *API) uploadJSON(r *http.Request, dsName string) (benchmark.Dataset, error) {
	return benchmark.LoadJSON(r.Body, dsName, "upload", "user-upload")
}

func (a *API) uploadMultipart(r *http.Request, dsName string) (benchmark.Dataset, error) {
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
		return benchmark.LoadJSON(file, name, version, "user-upload")
	}
	return benchmark.LoadCSV(file, name, version, "user-upload")
}

// ═══════════════════════════════════════════
// Audit endpoint
// ═══════════════════════════════════════════

func (a *API) handleAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		TargetBase     string `json:"target_base"`
		TargetKey      string `json:"target_key"`
		Model          string `json:"model"`
		Quick          bool   `json:"quick"`
		BenchmarkName  string `json:"benchmark_name"`
		BenchmarkLimit int    `json:"benchmark_limit"`
		BenchmarkLang  string `json:"benchmark_lang"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json: " + err.Error()})
		return
	}
	fillDefaults(a.cfg, &body.TargetBase, &body.TargetKey, &body.Model)
	if body.TargetBase == "" || body.TargetKey == "" || body.Model == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "target_base, target_key and model are required"})
		return
	}

	started := time.Now()
	result := map[string]any{"target": body.TargetBase, "model": body.Model, "started_at": started}

	// Phase 1: structure detection
	report, err := a.probeStore.ProbeSync(body.TargetBase, body.TargetKey, body.Model, body.Quick)
	if err != nil {
		result["detection"] = map[string]any{"error": err.Error()}
	} else {
		result["detection"] = report
	}

	// Phase 2: benchmark — use specified dataset or first available
	dsName := body.BenchmarkName
	if dsName == "" {
		names := a.registry.List()
		if len(names) > 0 {
			dsName = names[0]
		}
	}
	if dsName != "" {
		if ds, ok := a.registry.Get(dsName); ok {
			benchReq := benchmark.RunRequest{
				TargetBase: body.TargetBase,
				TargetKey:  body.TargetKey,
				Model:      body.Model,
				Language:   body.BenchmarkLang,
				Limit:      body.BenchmarkLimit,
			}
			benchReport, benchErr := a.runner.Run(r.Context(), ds, benchReq)
			if benchErr != nil {
				result["benchmark"] = map[string]any{"error": benchErr.Error()}
			} else {
				result["benchmark"] = benchReport
			}
		}
	}

	result["elapsed_ms"] = time.Since(started).Milliseconds()
	writeJSON(w, http.StatusOK, result)
}

// ═══════════════════════════════════════════
// HuggingFace fetch
// ═══════════════════════════════════════════

// Known adapters for public benchmarks
var knownAdapters = map[string]benchmark.ColumnAdapter{
	"SWE-bench-Verified": benchmark.SWEBenchAdapter{},
	"GPQA-Diamond":       benchmark.GPQAAdapter{},
	"HLE":                benchmark.HLEAdapter{},
	"LiveCodeBench":      benchmark.LiveCodeBenchAdapter{},
}

var knownHFDatasets = map[string]struct{ dataset, config, split string }{
	"SWE-bench-Verified": {"princeton-nlp/SWE-bench_Verified", "default", "test"},
	"GPQA-Diamond":       {"Idavidrein/gpqa", "default", "train"},
	"HLE":                {"cais/hle", "default", "test"},
	"LiveCodeBench":      {"livecodebench/code_generation_lite", "release_v5", "test"},
}

func (a *API) handleBenchmarkFetch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Name    string `json:"name"`              // Known adapter name or custom
		Dataset string `json:"dataset,omitempty"` // HF dataset path (e.g. "org/name")
		Config  string `json:"config,omitempty"`
		Split   string `json:"split,omitempty"`
		Limit   int    `json:"limit,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json: " + err.Error()})
		return
	}

	adapter, hasAdapter := knownAdapters[body.Name]
	if !hasAdapter {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "unknown benchmark adapter: " + body.Name,
			"available": func() []string {
				out := make([]string, 0)
				for k := range knownAdapters {
					out = append(out, k)
				}
				return out
			}(),
		})
		return
	}

	// Use known HF info or override
	hfInfo, ok := knownHFDatasets[body.Name]
	if ok {
		if body.Dataset == "" {
			body.Dataset = hfInfo.dataset
		}
		if body.Config == "" {
			body.Config = hfInfo.config
		}
		if body.Split == "" {
			body.Split = hfInfo.split
		}
	}
	if body.Dataset == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "dataset is required"})
		return
	}

	hf := benchmark.NewHFLoader()
	ds, err := hf.LoadDataset(body.Dataset, body.Config, body.Split, body.Limit, adapter)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}

	a.registry.Register(ds)
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "dataset fetched and registered",
		"stats":   ds.Stats(),
	})
}

// ═══════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════

func fillDefaults(cfg *config.Config, targetBase, targetKey, model *string) {
	if strings.TrimSpace(*targetBase) == "" {
		*targetBase = cfg.Upstream.BaseURL
	}
	if strings.TrimSpace(*targetKey) == "" {
		*targetKey = cfg.Upstream.APIKey
	}
	if strings.TrimSpace(*model) == "" {
		*model = cfg.Models.DefaultModel
	}
	*targetBase = strings.TrimRight(strings.TrimSpace(*targetBase), "/")
	*targetKey = strings.TrimSpace(*targetKey)
	*model = strings.TrimSpace(*model)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
