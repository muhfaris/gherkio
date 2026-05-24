package teststore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/muhfaris/gherkio/internal/core/project"
	"github.com/muhfaris/gherkio/internal/model"
	"github.com/muhfaris/gherkio/internal/runner"
	"gopkg.in/yaml.v3"
)

// TestInfo details metadata about discovered tests.
type TestInfo struct {
	RelativePath string `json:"relativePath"`
	AbsolutePath string `json:"absolutePath"`
	Scenario     string `json:"scenario"`
	StepCount    int    `json:"stepCount"`
}

// ValidationError holds individual validation issue details.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// ValidationResult aggregates scenario validations outcome.
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors"`
}

// ListTests discovers and extracts metadata of all YAML test scenarios.
func ListTests(projectDir string) ([]TestInfo, error) {
	meta, err := project.GetMeta(projectDir)
	if err != nil {
		return nil, err
	}

	var infos []TestInfo
	err = filepath.Walk(meta.TestsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible files
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".yaml") && !strings.HasSuffix(info.Name(), ".yml") {
			return nil
		}

		// Load test file without executing assertions
		tf, err := runner.LoadTestFile(path)
		if err != nil {
			return nil // skip corrupt files
		}

		relPath, err := filepath.Rel(meta.TestsDir, path)
		if err != nil {
			relPath = path
		}

		infos = append(infos, TestInfo{
			RelativePath: relPath,
			AbsolutePath: path,
			Scenario:     tf.Scenario,
			StepCount:    len(tf.Steps),
		})
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk tests directory: %w", err)
	}

	return infos, nil
}

// ReadTest parses a single test YAML file.
func ReadTest(absPath string) (*model.TestFile, error) {
	return runner.LoadTestFile(absPath)
}

// CreateTest creates a new test file, validating it first.
func CreateTest(projectDir, relativePath string, test *model.TestFile) error {
	meta, err := project.GetMeta(projectDir)
	if err != nil {
		return err
	}

	// Validate test scenario
	res, err := Validate(test, projectDir)
	if err != nil {
		return err
	}
	if !res.Valid {
		var errs []string
		for _, e := range res.Errors {
			errs = append(errs, fmt.Sprintf("%s: %s", e.Field, e.Message))
		}
		return fmt.Errorf("validation failed: %s", strings.Join(errs, "; "))
	}

	fullPath := filepath.Join(meta.TestsDir, relativePath)
	if _, err := os.Stat(fullPath); err == nil {
		return fmt.Errorf("test file already exists at %s", fullPath)
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	data, err := yaml.Marshal(test)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return os.WriteFile(fullPath, data, 0644)
}

// UpdateTest overwrites an existing Gherkio test file, writing a backup first.
func UpdateTest(absPath string, test *model.TestFile, projectDir string) error {
	res, err := Validate(test, projectDir)
	if err != nil {
		return err
	}
	if !res.Valid {
		var errs []string
		for _, e := range res.Errors {
			errs = append(errs, fmt.Sprintf("%s: %s", e.Field, e.Message))
		}
		return fmt.Errorf("validation failed: %s", strings.Join(errs, "; "))
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("test file does not exist at %s", absPath)
	}

	// Read existing to backup
	oldData, err := os.ReadFile(absPath)
	if err == nil {
		_ = os.WriteFile(absPath+".bak", oldData, 0644)
	}

	newData, err := yaml.Marshal(test)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return os.WriteFile(absPath, newData, 0644)
}

// DeleteTest deletes the scenario file.
func DeleteTest(absPath string) error {
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("test file does not exist at %s", absPath)
	}
	return os.Remove(absPath)
}

// Validate checks a test file's Gherkio structural and dependency references.
func Validate(test *model.TestFile, projectDir string) (*ValidationResult, error) {
	result := &ValidationResult{Valid: true}

	if strings.TrimSpace(test.Scenario) == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "scenario",
			Message: "Scenario name cannot be empty",
			Code:    "missing_field",
		})
	}

	if len(test.Steps) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "steps",
			Message: "At least one step must be defined",
			Code:    "empty_steps",
		})
	}

	// Validate steps
	for i, step := range test.Steps {
		prefix := fmt.Sprintf("steps[%d]", i)
		if step.Use == "" && step.Request.Method == "" && step.Request.URL == "" {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix,
				Message: "Step must define either a 'use' reference or a 'request' configuration",
				Code:    "invalid_step",
			})
			continue
		}

		if step.Use != "" && (step.Request.Method != "" || step.Request.URL != "") {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix,
				Message: "Step cannot define both 'use' and 'request'",
				Code:    "mutually_exclusive",
			})
		}

		if step.Request.Method != "" || step.Request.URL != "" {
			m := strings.ToUpper(step.Request.Method)
			validMethods := map[string]bool{
				"GET":     true,
				"POST":    true,
				"PUT":     true,
				"PATCH":   true,
				"DELETE":  true,
				"HEAD":    true,
				"OPTIONS": true,
			}
			if !validMethods[m] {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:   prefix + ".request.method",
					Message: fmt.Sprintf("Invalid HTTP method '%s'. Supported: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS", step.Request.Method),
					Code:    "invalid_method",
				})
			}

			if strings.TrimSpace(step.Request.URL) == "" {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:   prefix + ".request.url",
					Message: "Request URL is required",
					Code:    "missing_field",
				})
			}
		}

		// Validate referenced schema file if it exists
		if schemaNameVal, ok := step.Expect.Extra["schema"]; ok && projectDir != "" {
			if schemaName, ok := schemaNameVal.(string); ok && schemaName != "" {
				meta, err := project.GetMeta(projectDir)
				if err == nil {
					schemaFile := filepath.Join(meta.SchemasDir, schemaName+".yaml")
					if _, err := os.Stat(schemaFile); os.IsNotExist(err) {
						// Also check .yml
						schemaFileYml := filepath.Join(meta.SchemasDir, schemaName+".yml")
						if _, errYml := os.Stat(schemaFileYml); os.IsNotExist(errYml) {
							result.Valid = false
							result.Errors = append(result.Errors, ValidationError{
								Field:   prefix + ".expect.schema",
								Message: fmt.Sprintf("Referenced schema '%s' does not exist in .gherkio/schemas/", schemaName),
								Code:    "schema_not_found",
							})
						}
					}
				}
			}
		}
	}

	return result, nil
}
