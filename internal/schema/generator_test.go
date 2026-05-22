package schema

import (
	"testing"
)

func TestGenerateJSONSchema(t *testing.T) {
	b, err := GenerateJSONSchema()
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	if len(b) == 0 {
		t.Fatal("Expected generated schema, got empty bytes")
	}
}
