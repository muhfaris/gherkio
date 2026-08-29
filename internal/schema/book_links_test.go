package schema

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestBookSummaryLinksExist(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	srcDir := filepath.Join(root, "docs", "book", "src")
	data, err := os.ReadFile(filepath.Join(srcDir, "SUMMARY.md"))
	if err != nil {
		t.Fatalf("read SUMMARY.md: %v", err)
	}

	linkPattern := regexp.MustCompile(`\[[^]]+\]\(([^)]+)\)`)
	for _, match := range linkPattern.FindAllStringSubmatch(string(data), -1) {
		target := strings.SplitN(match[1], "#", 2)[0]
		if target == "" || strings.Contains(target, "://") {
			continue
		}
		if _, err := os.Stat(filepath.Join(srcDir, filepath.FromSlash(target))); err != nil {
			t.Errorf("SUMMARY.md target %q does not exist", match[1])
		}
	}
}
