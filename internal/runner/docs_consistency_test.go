package runner

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAIReferenceCoversRuntimeVocabulary(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	data, err := os.ReadFile(filepath.Join(root, "docs", "book", "src", "reference", "ai-reference.md"))
	if err != nil {
		t.Fatalf("read AI reference: %v", err)
	}
	doc := strings.ReplaceAll(string(data), " ", "")

	for _, variable := range GetVariableInfo() {
		name := strings.ReplaceAll(variable.Name, " ", "")
		if !strings.Contains(doc, name) {
			t.Errorf("AI reference is missing built-in variable or function %q", variable.Name)
		}
	}

	for _, matcher := range GetMatchersInfo() {
		if !strings.Contains(string(data), "`"+matcher.Name+"`") {
			t.Errorf("AI reference is missing matcher %q", matcher.Name)
		}
	}

	for _, strategy := range GetBackoffStrategies() {
		if !strings.Contains(string(data), strategy) {
			t.Errorf("AI reference is missing retry strategy %q", strategy)
		}
	}
}
