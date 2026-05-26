package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSchema(t *testing.T) {
	// Setup test directory
	projectDir := t.TempDir()
	schemasDir := filepath.Join(projectDir, ".gherkio", "schemas")
	err := os.MkdirAll(schemasDir, 0755)
	if err != nil {
		t.Fatalf("failed to create schemas directory: %v", err)
	}

	// Create a valid schema file
	validSchemaYAML := `type: object
required:
  - id
properties:
  id:
    type: integer`
	validSchemaPath := filepath.Join(schemasDir, "user.yaml")
	err = os.WriteFile(validSchemaPath, []byte(validSchemaYAML), 0644)
	if err != nil {
		t.Fatalf("failed to write valid schema file: %v", err)
	}

	// Create an invalid schema file
	invalidSchemaYAML := `type: object
required:
  - id
properties:
  id:
    type: [invalid]` // Invalid YAML array syntax
	invalidSchemaPath := filepath.Join(schemasDir, "invalid.yaml")
	err = os.WriteFile(invalidSchemaPath, []byte(invalidSchemaYAML), 0644)
	if err != nil {
		t.Fatalf("failed to write invalid schema file: %v", err)
	}

	tests := []struct {
		name       string
		schemaName string
		wantErr    bool
	}{
		{
			name:       "valid schema",
			schemaName: "user",
			wantErr:    false,
		},
		{
			name:       "non-existent schema",
			schemaName: "missing",
			wantErr:    true,
		},
		{
			name:       "invalid yaml",
			schemaName: "invalid",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := LoadSchema(tt.schemaName, projectDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadSchema() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && schema == nil {
				t.Errorf("LoadSchema() returned nil schema")
			}
		})
	}
}
