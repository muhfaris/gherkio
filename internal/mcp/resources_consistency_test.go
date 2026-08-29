package mcp

import (
	"strings"
	"testing"
)

func TestSpecResourceHasCanonicalRequestFields(t *testing.T) {
	spec := (&Server{}).buildSpecResource()
	queryLine := "- **query**: (Map of string:string, Optional)"
	if count := strings.Count(spec, queryLine); count != 1 {
		t.Fatalf("request query field appears %d times, want exactly once", count)
	}
	if strings.Contains(spec, "auto-detected if omitted") {
		t.Fatal("multipart documentation must not claim automatic MIME detection")
	}
	if !strings.Contains(spec, "application/octet-stream") {
		t.Fatal("multipart documentation must state the default content type")
	}
}
