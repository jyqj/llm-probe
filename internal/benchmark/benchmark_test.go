package benchmark

import (
	"context"
	"strings"
	"testing"
)

func TestLoadCSV(t *testing.T) {
	csv := `task_id,prompt,language,category
t1,"What is 2+2?",math,basic
t2,"Explain recursion",python,CS
`
	ds, err := LoadCSV(strings.NewReader(csv), "test", "v1", "test-source")
	if err != nil {
		t.Fatalf("LoadCSV error: %v", err)
	}
	if ds.Name() != "test" {
		t.Errorf("name = %q, want %q", ds.Name(), "test")
	}
	if len(ds.Tasks()) != 2 {
		t.Fatalf("tasks = %d, want 2", len(ds.Tasks()))
	}
	if ds.Tasks()[0].TaskID != "t1" || ds.Tasks()[0].Prompt != "What is 2+2?" {
		t.Errorf("task 0 = %+v", ds.Tasks()[0])
	}
	if ds.Tasks()[1].Language != "python" {
		t.Errorf("task 1 language = %q", ds.Tasks()[1].Language)
	}
}

func TestLoadCSV_MissingRequiredColumn(t *testing.T) {
	csv := `id,question
1,"hello"
`
	_, err := LoadCSV(strings.NewReader(csv), "test", "v1", "")
	if err == nil {
		t.Fatal("expected error for missing task_id column")
	}
	if !strings.Contains(err.Error(), "task_id") {
		t.Errorf("error = %q, want mention of task_id", err.Error())
	}
}

func TestLoadCSV_ExtraColumnsAsMetadata(t *testing.T) {
	csv := `task_id,prompt,docker_image,repo_url
t1,"do stuff",img:latest,github.com/foo
`
	ds, err := LoadCSV(strings.NewReader(csv), "test", "v1", "")
	if err != nil {
		t.Fatalf("LoadCSV error: %v", err)
	}
	task := ds.Tasks()[0]
	if task.Metadata == nil {
		t.Fatal("metadata is nil")
	}
	if task.Metadata["docker_image"] != "img:latest" {
		t.Errorf("metadata[docker_image] = %q", task.Metadata["docker_image"])
	}
}

func TestLoadJSON_FullFormat(t *testing.T) {
	j := `{"name":"jbench","version":"v2","tasks":[
		{"task_id":"j1","prompt":"hello"},
		{"task_id":"j2","prompt":"world","language":"go"}
	]}`
	ds, err := LoadJSON(strings.NewReader(j), "fallback", "v0", "")
	if err != nil {
		t.Fatalf("LoadJSON error: %v", err)
	}
	if ds.Name() != "jbench" {
		t.Errorf("name = %q, want jbench", ds.Name())
	}
	if len(ds.Tasks()) != 2 {
		t.Fatalf("tasks = %d, want 2", len(ds.Tasks()))
	}
}

func TestLoadJSON_ArrayFormat(t *testing.T) {
	j := `[{"task_id":"a1","prompt":"q1"},{"task_id":"a2","prompt":"q2"}]`
	ds, err := LoadJSON(strings.NewReader(j), "arr", "v1", "")
	if err != nil {
		t.Fatalf("LoadJSON error: %v", err)
	}
	if ds.Name() != "arr" {
		t.Errorf("name = %q, want arr", ds.Name())
	}
	if len(ds.Tasks()) != 2 {
		t.Fatalf("tasks = %d, want 2", len(ds.Tasks()))
	}
}

func TestLoadJSON_Invalid(t *testing.T) {
	_, err := LoadJSON(strings.NewReader("not json at all"), "x", "v1", "")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestRegistry(t *testing.T) {
	reg := NewRegistry()
	if len(reg.List()) != 0 {
		t.Fatalf("expected empty registry")
	}

	ds := &GenericDataset{DataName: "bench-a", DataVersion: "v1", TaskList: []Task{{TaskID: "1", Prompt: "p"}}}
	reg.Register(ds)

	names := reg.List()
	if len(names) != 1 || names[0] != "bench-a" {
		t.Errorf("list = %v", names)
	}

	got, ok := reg.Get("bench-a")
	if !ok || got.Name() != "bench-a" {
		t.Errorf("Get failed")
	}

	_, ok = reg.Get("nonexist")
	if ok {
		t.Error("Get(nonexist) should return false")
	}

	reg.Remove("bench-a")
	if len(reg.List()) != 0 {
		t.Error("expected empty after remove")
	}
}

func TestGenericDataset_Filter(t *testing.T) {
	ds := &GenericDataset{
		DataName: "test",
		TaskList: []Task{
			{TaskID: "1", Prompt: "p1", Language: "go", Category: "A"},
			{TaskID: "2", Prompt: "p2", Language: "python", Category: "B"},
			{TaskID: "3", Prompt: "p3", Language: "go", Category: "A"},
			{TaskID: "4", Prompt: "p4", Language: "go", Category: "B"},
		},
	}

	// Filter by language
	got := ds.Filter(Filter{Language: "go"})
	if len(got) != 3 {
		t.Errorf("go filter = %d, want 3", len(got))
	}

	// Filter by category
	got = ds.Filter(Filter{Category: "B"})
	if len(got) != 2 {
		t.Errorf("B filter = %d, want 2", len(got))
	}

	// Limit
	got = ds.Filter(Filter{Language: "go", Limit: 2})
	if len(got) != 2 {
		t.Errorf("go limit=2 = %d, want 2", len(got))
	}

	// By IDs
	got = ds.Filter(Filter{TaskIDs: []string{"2", "4"}})
	if len(got) != 2 || got[0].TaskID != "2" {
		t.Errorf("ids filter = %v", got)
	}

	// Find
	task, ok := ds.Find("3")
	if !ok || task.TaskID != "3" {
		t.Errorf("Find(3) failed")
	}
	_, ok = ds.Find("999")
	if ok {
		t.Error("Find(999) should return false")
	}
}

// stubDataset is a minimal Dataset for testing Runner error paths.
type stubDataset struct {
	tasks []Task
}

func (s *stubDataset) Name() string    { return "stub" }
func (s *stubDataset) Version() string { return "v1" }
func (s *stubDataset) Source() string  { return "" }
func (s *stubDataset) Stats() Stats    { return Stats{Name: "stub", TotalTasks: len(s.tasks)} }
func (s *stubDataset) Tasks() []Task   { return s.tasks }
func (s *stubDataset) Filter(f Filter) []Task {
	if f.Limit > 0 && f.Limit < len(s.tasks) {
		return s.tasks[:f.Limit]
	}
	return s.tasks
}
func (s *stubDataset) Find(id string) (Task, bool) {
	for _, t := range s.tasks {
		if t.TaskID == id {
			return t, true
		}
	}
	return Task{}, false
}

func TestRunner_MissingParams(t *testing.T) {
	runner := NewRunner(nil)
	ds := &stubDataset{tasks: []Task{{TaskID: "1", Prompt: "hi"}}}

	tests := []struct {
		name string
		req  RunRequest
	}{
		{"missing target_base", RunRequest{TargetKey: "k", Model: "m"}},
		{"missing target_key", RunRequest{TargetBase: "http://x", Model: "m"}},
		{"missing model", RunRequest{TargetBase: "http://x", TargetKey: "k"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runner.Run(context.Background(), ds, tt.req)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestRunStream_MissingParams(t *testing.T) {
	runner := NewRunner(nil)
	ds := &stubDataset{tasks: []Task{{TaskID: "1", Prompt: "hi"}}}

	var events []StreamEvent
	_, err := runner.RunStream(context.Background(), ds, RunRequest{}, func(ev StreamEvent) {
		events = append(events, ev)
	})
	if err == nil {
		t.Error("expected error for empty request")
	}
	if len(events) != 0 {
		t.Error("should not emit events on validation failure")
	}
}

func TestRunner_EmptyDataset(t *testing.T) {
	runner := NewRunner(nil)
	ds := &stubDataset{tasks: nil}

	report, err := runner.Run(context.Background(), ds, RunRequest{
		TargetBase: "http://x", TargetKey: "k", Model: "m",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.TaskTotal != 0 || len(report.Results) != 0 {
		t.Errorf("expected empty report, got %d tasks", report.TaskTotal)
	}
}
