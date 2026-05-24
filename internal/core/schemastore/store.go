package schemastore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/muhfaris/gherkio/internal/core/project"
	"github.com/muhfaris/gherkio/internal/model"
	"gopkg.in/yaml.v3"
)

// SchemaInfo details discovered custom schema metadata.
type SchemaInfo struct {
	Name     string `json:"name"`
	FilePath string `json:"filePath"`
}

// List scans all schema definition files recursively under .gherkio/schemas/.
func List(projectDir string) ([]SchemaInfo, error) {
	meta, err := project.GetMeta(projectDir)
	if err != nil {
		return nil, err
	}

	var schemas []SchemaInfo
	err = filepath.Walk(meta.SchemasDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible files
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".yaml") && !strings.HasSuffix(info.Name(), ".yml") {
			return nil
		}

		relPath, err := filepath.Rel(meta.SchemasDir, path)
		if err != nil {
			relPath = path
		}

		name := strings.TrimSuffix(strings.TrimSuffix(relPath, ".yaml"), ".yml")

		schemas = append(schemas, SchemaInfo{
			Name:     name,
			FilePath: path,
		})
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk schemas directory: %w", err)
	}

	return schemas, nil
}

// Read loads and parses a custom Gherkio schema definition.
func Read(projectDir, name string) (*model.Schema, error) {
	meta, err := project.GetMeta(projectDir)
	if err != nil {
		return nil, err
	}

	fullPath := filepath.Join(meta.SchemasDir, name+".yaml")
	data, err := os.ReadFile(fullPath)
	if os.IsNotExist(err) {
		fullPath = filepath.Join(meta.SchemasDir, name+".yml")
		data, err = os.ReadFile(fullPath)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to read schema '%s': %w", name, err)
	}

	var schema model.Schema
	if err := yaml.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema '%s': %w", name, err)
	}

	return &schema, nil
}

// Create writes a new schema definition file.
func Create(projectDir, name string, schema *model.Schema) error {
	meta, err := project.GetMeta(projectDir)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(meta.SchemasDir, name+".yaml")
	if _, err := os.Stat(fullPath); err == nil {
		return fmt.Errorf("schema '%s' already exists", name)
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	data, err := yaml.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return os.WriteFile(fullPath, data, 0644)
}

// Update overwrites an existing schema definition file.
func Update(projectDir, name string, schema *model.Schema) error {
	meta, err := project.GetMeta(projectDir)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(meta.SchemasDir, name+".yaml")
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		fullPath = filepath.Join(meta.SchemasDir, name+".yml")
		if _, errYml := os.Stat(fullPath); os.IsNotExist(errYml) {
			return fmt.Errorf("schema '%s' does not exist", name)
		}
	}

	data, err := yaml.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return os.WriteFile(fullPath, data, 0644)
}

// Delete removes a schema definition file.
func Delete(projectDir, name string) error {
	meta, err := project.GetMeta(projectDir)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(meta.SchemasDir, name+".yaml")
	err = os.Remove(fullPath)
	if os.IsNotExist(err) {
		fullPath = filepath.Join(meta.SchemasDir, name+".yml")
		err = os.Remove(fullPath)
	}

	if err != nil {
		return fmt.Errorf("failed to delete schema '%s': %w", name, err)
	}

	return nil
}
