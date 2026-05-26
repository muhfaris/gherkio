package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRootAndGetMeta(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Create a dummy .gherkio project structure
	gDir := filepath.Join(tmpDir, ".gherkio")
	err := os.MkdirAll(filepath.Join(gDir, "tests"), 0755)
	if err != nil {
		t.Fatalf("failed to setup: %v", err)
	}

	configYAML := `
project:
  name: "test-mcp-project"
  version: "2.3.4"
`
	err = os.WriteFile(filepath.Join(gDir, "config.yaml"), []byte(configYAML), 0644)
	if err != nil {
		t.Fatalf("failed to setup config: %v", err)
	}

	// 2. Test FindRoot from a nested directory
	nestedDir := filepath.Join(tmpDir, "nested", "deeply")
	_ = os.MkdirAll(nestedDir, 0755)

	root, err := FindRoot(nestedDir)
	if err != nil {
		t.Fatalf("FindRoot failed: %v", err)
	}
	if root != tmpDir {
		t.Errorf("expected root %s, got %s", tmpDir, root)
	}

	// 3. Test GetMeta
	meta, err := GetMeta(tmpDir)
	if err != nil {
		t.Fatalf("GetMeta failed: %v", err)
	}

	if meta.Name != "test-mcp-project" {
		t.Errorf("expected name test-mcp-project, got %s", meta.Name)
	}
	if meta.Version != "2.3.4" {
		t.Errorf("expected version 2.3.4, got %s", meta.Version)
	}
	if meta.TestsDir != filepath.Join(tmpDir, ".gherkio/tests") {
		t.Errorf("unexpected tests dir: %s", meta.TestsDir)
	}
}
