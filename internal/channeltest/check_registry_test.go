package channeltest

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestCheckResultNamesAreRegistered(t *testing.T) {
	re := regexp.MustCompile(`Name:\s*"([^"]+)"`)
	files, err := filepath.Glob("check_*.go")
	if err != nil {
		t.Fatalf("glob check files: %v", err)
	}
	files = append(files, "runner.go")
	for _, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		for _, match := range re.FindAllStringSubmatch(string(b), -1) {
			name := match[1]
			if _, ok := checkRegistry[name]; !ok {
				t.Errorf("%s uses unregistered check %q", file, name)
			}
		}
	}
}

func TestDefaultFixFallback(t *testing.T) {
	checks := []CheckResult{{Name: "id_format", Pass: false, Detail: "missing id"}}
	rec := RecommendFixes(checks)
	if len(rec.Fixes) != 1 || rec.Fixes[0] != "id_rewrite" {
		t.Fatalf("RecommendFixes should use registry default fix when CheckResult.Fix is empty, got %#v", rec.Fixes)
	}
	if got := BuildSummary(checks); !strings.Contains(got, "id_rewrite") {
		t.Fatalf("BuildSummary should include registry default fix, got %q", got)
	}
}
