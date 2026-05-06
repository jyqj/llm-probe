package sweatlas

import (
	"testing"

	"detector-service/internal/intelligence"
)

func TestLoadEmbeddedDataset(t *testing.T) {
	ds, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	tasks := ds.Tasks()
	if got, want := len(tasks), 124; got != want {
		t.Fatalf("len(tasks) = %d, want %d", got, want)
	}
	if tasks[0].TaskID == "" || tasks[0].Prompt == "" || len(tasks[0].Rubric) == 0 {
		t.Fatalf("first task not parsed correctly: %+v", tasks[0])
	}
	stats := ds.Stats()
	if stats.Languages["go"] == 0 || stats.Categories["Architecture & system design"] == 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestFilterAndFind(t *testing.T) {
	ds, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	goTasks := ds.Filter(intelligence.Filter{Language: "go", Limit: 2})
	if len(goTasks) != 2 {
		t.Fatalf("go tasks = %d, want 2", len(goTasks))
	}
	for _, task := range goTasks {
		if task.Language != "go" {
			t.Fatalf("got non-go task: %+v", task)
		}
	}
	tasks := ds.Tasks()
	id := tasks[0].TaskID
	found, ok := ds.Find(id)
	if !ok || found.TaskID != id {
		t.Fatalf("Find(%q) failed", id)
	}
}
