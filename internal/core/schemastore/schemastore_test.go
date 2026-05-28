package schemastore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/muhfaris/gherkio/internal/model"
)

func TestSchemaStoreCRUD(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Create a dummy .gherkio setup
	gDir := filepath.Join(tmpDir, ".gherkio")
	schemasDir := filepath.Join(gDir, "schemas")
	_ = os.MkdirAll(schemasDir, 0755)
	_ = os.WriteFile(filepath.Join(gDir, "config.yaml"), []byte(""), 0644)

	// 2. Create Schema
	validSchema := &model.Schema{
		Type:     "object",
		Required: []string{"id"},
		Properties: map[string]*model.Schema{
			"id": {Type: "string", Format: "uuid"},
		},
	}

	err := Create(tmpDir, "auth/login-response", validSchema)
	if err != nil {
		t.Fatalf("Create schema failed: %v", err)
	}

	// 3. List Schemas
	schemas, err := List(tmpDir)
	if err != nil {
		t.Fatalf("List schemas failed: %v", err)
	}
	if len(schemas) != 1 {
		t.Errorf("expected 1 schema, got %d", len(schemas))
	}
	if schemas[0].Name != "auth/login-response" {
		t.Errorf("expected name auth/login-response, got %s", schemas[0].Name)
	}

	// 4. Update Schema
	validSchema.Type = "string"
	err = Update(tmpDir, "auth/login-response", validSchema)
	if err != nil {
		t.Fatalf("Update schema failed: %v", err)
	}

	// 5. Read Schema
	readSchema, err := Read(tmpDir, "auth/login-response")
	if err != nil {
		t.Fatalf("Read schema failed: %v", err)
	}
	if readSchema.Type != "string" {
		t.Errorf("expected string, got %s", readSchema.Type)
	}

	// 6. Delete Schema
	err = Delete(tmpDir, "auth/login-response")
	if err != nil {
		t.Fatalf("Delete schema failed: %v", err)
	}
}
