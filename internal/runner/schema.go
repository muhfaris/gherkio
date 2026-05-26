package runner

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/muhfaris/gherkio/internal/model"
	"gopkg.in/yaml.v3"
)

// LoadSchema loads a schema by name from the project's schemas directory.
// name example: "users/user-response" or "user-response"
// Resolution: <schemasDir>/<name>.yaml, then <schemasDir>/<name>.yml
// The schemas directory is resolved from config (schemas.path) if available,
// otherwise defaults to .gherkio/schemas relative to projectDir.
func LoadSchema(name string, projectDir string) (*model.Schema, error) {
	schemasDir := filepath.Join(projectDir, ".gherkio", "schemas")

	// Override from config if available
	cfg, err := LoadConfig(projectDir)
	if err == nil && cfg.Schemas.Path != "" {
		if filepath.IsAbs(cfg.Schemas.Path) {
			schemasDir = cfg.Schemas.Path
		} else {
			schemasDir = filepath.Join(projectDir, cfg.Schemas.Path)
		}
	}

	// Check .yaml first
	yamlPath := filepath.Join(schemasDir, name+".yaml")
	if _, err := os.Stat(yamlPath); err == nil {
		return parseSchemaFile(yamlPath)
	}

	// Check .yml fallback
	ymlPath := filepath.Join(schemasDir, name+".yml")
	if _, err := os.Stat(ymlPath); err == nil {
		return parseSchemaFile(ymlPath)
	}

	return nil, fmt.Errorf("schema file not found at %s", yamlPath)
}

func parseSchemaFile(path string) (*model.Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	var schema model.Schema
	if err := yaml.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema file: %w", err)
	}

	return &schema, nil
}
