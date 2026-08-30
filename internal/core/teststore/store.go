package teststore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// ListTests discovers and extracts metadata of all YAML test scenarios under the given testsDir.
func ListTests(testsDir string) ([]TestInfo, error) {
	var infos []TestInfo
	err := filepath.Walk(testsDir, func(path string, info os.FileInfo, err error) error {
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

		relPath, err := filepath.Rel(testsDir, path)
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
func CreateTest(testsDir string, schemasDir string, relativePath string, test *model.TestFile) error {
	// Validate test scenario
	res, err := Validate(test, schemasDir)
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

	fullPath := filepath.Join(testsDir, relativePath)
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
func UpdateTest(absPath string, test *model.TestFile, schemasDir string) error {
	res, err := Validate(test, schemasDir)
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
func Validate(test *model.TestFile, schemasDir string) (*ValidationResult, error) {
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
		operationCount := 0
		if step.Use != "" {
			operationCount++
		}
		if step.Set != nil {
			operationCount++
		}
		if step.Redis != nil {
			operationCount++
		}
		if step.Repeat != nil {
			operationCount++
		}
		if step.Request.Method != "" || step.Request.URL != "" {
			operationCount++
		}
		if operationCount == 0 {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix,
				Message: "Step must define one of 'request', 'redis', 'use', 'set', or 'repeat'",
				Code:    "invalid_step",
			})
			continue
		}

		if operationCount > 1 {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix,
				Message: "Step operations 'request', 'redis', 'use', 'set', and 'repeat' are mutually exclusive",
				Code:    "mutually_exclusive",
			})
		}

		if step.Repeat != nil && len(step.Repeat.Steps) > 0 {
			nested, err := Validate(&model.TestFile{Scenario: "repeat block", Steps: step.Repeat.Steps}, schemasDir)
			if err != nil {
				return nil, err
			}
			if !nested.Valid {
				result.Valid = false
				for _, nestedError := range nested.Errors {
					nestedError.Field = prefix + ".repeat." + nestedError.Field
					result.Errors = append(result.Errors, nestedError)
				}
			}
		}

		if step.Redis != nil {
			validCommands := map[string]bool{"get": true, "exists": true, "ttl": true, "hgetall": true}
			if strings.TrimSpace(step.Redis.Connection) == "" {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{Field: prefix + ".redis.connection", Message: "Redis connection is required", Code: "missing_field"})
			}
			if !validCommands[strings.ToLower(step.Redis.Command)] {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{Field: prefix + ".redis.command", Message: "Redis command must be one of: get, exists, ttl, hgetall", Code: "invalid_command"})
			}
			if strings.TrimSpace(step.Redis.Key) == "" {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{Field: prefix + ".redis.key", Message: "Redis key is required", Code: "missing_field"})
			}
			if step.Expect.Status != 0 {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{Field: prefix + ".expect.status", Message: "HTTP status assertions are not valid on Redis steps", Code: "invalid_assertion"})
			}
			if step.Retry != nil && len(step.Retry.OnStatus) > 0 {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{Field: prefix + ".retry.onStatus", Message: "HTTP status retry conditions are not valid on Redis steps", Code: "invalid_retry"})
			}
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
		if schemaNameVal, ok := step.Expect.Extra["schema"]; ok && schemasDir != "" {
			if schemaName, ok := schemaNameVal.(string); ok && schemaName != "" {
				schemaName = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(schemaName), "not "))
				schemaFile := filepath.Join(schemasDir, schemaName+".yaml")
				if _, err := os.Stat(schemaFile); os.IsNotExist(err) {
					// Also check .yml
					schemaFileYml := filepath.Join(schemasDir, schemaName+".yml")
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

		// Validate transform projections
		if len(step.Request.Transform) > 0 {
			for targetPath, projCfg := range step.Request.Transform {
				validateProjectionConfig(projCfg, fmt.Sprintf("%s.request.transform.%s", prefix, targetPath), result)
			}
		}
	}

	return result, nil
}

// validateProjectionConfig checks for structural correctness of projection configs.
func validateProjectionConfig(cfg *model.ProjectionConfig, prefix string, result *ValidationResult) {
	if cfg == nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   prefix,
			Message: "Projection configuration cannot be nil",
			Code:    "invalid_transform",
		})
		return
	}

	if strings.TrimSpace(cfg.From) == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   prefix + ".from",
			Message: "from variable is required",
			Code:    "missing_field",
		})
	} else if !strings.HasPrefix(cfg.From, "$") {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   prefix + ".from",
			Message: "from source variable must start with '$'",
			Code:    "invalid_variable_reference",
		})
	}

	if cfg.Limit < 0 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   prefix + ".limit",
			Message: "limit must be a positive integer",
			Code:    "invalid_limit",
		})
	}

	if len(cfg.Select) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   prefix + ".select",
			Message: "select structure is required",
			Code:    "missing_field",
		})
	} else {
		for k, v := range cfg.Select {
			if m, ok := v.(map[string]interface{}); ok {
				if subCfg, ok := parseProjectionConfig(m); ok {
					validateProjectionConfig(subCfg, prefix+".select."+k, result)
				}
			}
		}
	}
}

// parseProjectionConfig parses a map into ProjectionConfig for validation.
func parseProjectionConfig(m map[string]interface{}) (*model.ProjectionConfig, bool) {
	fromVal, hasFrom := m["from"]
	selectVal, hasSelect := m["select"]
	if !hasFrom || !hasSelect {
		return nil, false
	}

	fromStr, ok1 := fromVal.(string)
	selectMap, ok2 := selectVal.(map[string]interface{})
	if !ok1 || !ok2 {
		return nil, false
	}

	cfg := &model.ProjectionConfig{
		From:   fromStr,
		Select: selectMap,
	}

	if asVal, ok := m["as"].(string); ok {
		cfg.As = asVal
	}
	if limitVal, ok := m["limit"].(int); ok {
		cfg.Limit = limitVal
	} else if limitFloat, ok := m["limit"].(float64); ok {
		cfg.Limit = int(limitFloat)
	}
	if whereVal, ok := m["where"].(map[string]interface{}); ok {
		cfg.Where = whereVal
	}

	return cfg, true
}
