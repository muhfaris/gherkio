package report

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnforceRetention(t *testing.T) {
	// Create a temp directory for reports base path
	tempDir, err := os.MkdirTemp("", "gherkio-retention-test-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Sub-folder names (simulated runs)
	// We have 5 timestamp directories and some non-timestamp folders
	timestampDirs := []string{
		"20260528_100000", // Oldest
		"20260528_100500",
		"20260528_101000",
		"20260528_101500",
		"20260528_102000", // Newest
	}

	nonTimestampDirs := []string{
		"latest",
		"archive",
		"failures",
	}

	// Create directories
	for _, dirName := range timestampDirs {
		dirPath := filepath.Join(tempDir, dirName)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dirName, err)
		}
	}
	for _, dirName := range nonTimestampDirs {
		dirPath := filepath.Join(tempDir, dirName)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dirName, err)
		}
	}

	// Test EnforceRetention with limit = 3
	// Expected to keep: the 3 newest (101000, 101500, 102000)
	// Expected to prune: 100000, 100500
	// Non-timestamp directories (latest, archive, failures) should remain untouched.
	limit := 3
	if err := EnforceRetention("", tempDir, limit); err != nil {
		t.Fatalf("EnforceRetention failed: %v", err)
	}

	// Verify pruning
	shouldBePruned := []string{
		"20260528_100000",
		"20260528_100500",
	}
	shouldBeKept := []string{
		"20260528_101000",
		"20260528_101500",
		"20260528_102000",
		"latest",
		"archive",
		"failures",
	}

	for _, dirName := range shouldBePruned {
		dirPath := filepath.Join(tempDir, dirName)
		if _, err := os.Stat(dirPath); !os.IsNotExist(err) {
			t.Errorf("expected directory %s to be pruned, but it still exists", dirName)
		}
	}

	for _, dirName := range shouldBeKept {
		dirPath := filepath.Join(tempDir, dirName)
		if _, err := os.Stat(dirPath); err != nil {
			t.Errorf("expected directory %s to be kept, but stat failed: %v", dirName, err)
		}
	}
}
