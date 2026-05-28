package project

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/muhfaris/gherkio/internal/core/teststore"
	"github.com/muhfaris/gherkio/internal/model"
	"github.com/muhfaris/gherkio/internal/runner"
	"gopkg.in/yaml.v3"
)

// ValidationIssue represents a static analysis finding in a test scenario file.
type ValidationIssue struct {
	File  string
	Field string
	Code  string
	Msg   string
}

// ValidationResult groups all static analysis issues for a single file.
type ValidationResult struct {
	File   string
	Issues []ValidationIssue
}

// ValidateProject walks the Gherkio workspace to validate one or all test files.
func ValidateProject(cwd, projectDir, testPath, env string) ([]ValidationResult, error) {
	var files []string
	var err error

	if testPath == "" {
		// Validate all test files
		testsDir := filepath.Join(projectDir, ".gherkio", "tests")
		files, err = discoverTestFilesForValidate(testsDir)
		if err != nil {
			return nil, fmt.Errorf("failed to discover test files: %w", err)
		}
		if len(files) == 0 {
			return []ValidationResult{}, nil
		}
	} else {
		// Validate single file
		fullPath, err := resolveTestPathForValidate(cwd, projectDir, testPath)
		if err != nil {
			return nil, fmt.Errorf("test file not found: %w", err)
		}
		files = []string{fullPath}
	}

	// Load credentials for account validation
	var creds *model.Credentials
	if env != "" {
		creds, _ = runner.LoadCredentials(projectDir, env)
	}

	// Load available schemas for schema reference validation
	schemas, _ := loadAvailableSchemas(projectDir)

	var results []ValidationResult
	for _, file := range files {
		result := ValidateFile(file, projectDir, creds, schemas)
		results = append(results, result)
	}

	return results, nil
}

// ValidateFile performs static analysis on a single Gherkio test file.
func ValidateFile(filePath, projectDir string, creds *model.Credentials, schemas []string) ValidationResult {
	result := ValidationResult{File: filePath}

	// Load and parse YAML
	data, err := os.ReadFile(filePath)
	if err != nil {
		result.Issues = append(result.Issues, ValidationIssue{
			Field: "file",
			Code:  "read_error",
			Msg:   fmt.Sprintf("Failed to read file: %v", err),
		})
		return result
	}

	// Try to parse as YAML first
	var rawData map[string]interface{}
	if err := yaml.Unmarshal(data, &rawData); err != nil {
		result.Issues = append(result.Issues, ValidationIssue{
			Field: "yaml",
			Code:  "syntax_error",
			Msg:   fmt.Sprintf("YAML syntax error: %v", err),
		})
		return result
	}

	// Load as TestFile
	test, err := runner.LoadTestFile(filePath)
	if err != nil {
		result.Issues = append(result.Issues, ValidationIssue{
			Field: "yaml",
			Code:  "parse_error",
			Msg:   fmt.Sprintf("Failed to parse test file: %v", err),
		})
		return result
	}

	// Run structural validation from teststore
	schemasDir := filepath.Join(projectDir, ".gherkio", "schemas")
	if meta, err := GetMeta(projectDir); err == nil {
		schemasDir = meta.SchemasDir
	}
	storeResult, err := teststore.Validate(test, schemasDir)
	if err == nil && !storeResult.Valid {
		for _, e := range storeResult.Errors {
			result.Issues = append(result.Issues, ValidationIssue{
				Field: e.Field,
				Code:  e.Code,
				Msg:   e.Message,
			})
		}
	}

	// Extended validations
	result.Issues = append(result.Issues, validateVariableReferences(test, creds)...)
	result.Issues = append(result.Issues, validateRetryConfig(test)...)
	result.Issues = append(result.Issues, validateUseFiles(test, projectDir, filePath)...)
	result.Issues = append(result.Issues, validateSchemaReferences(test, schemas)...)
	result.Issues = append(result.Issues, validateBodyPaths(test)...)
	result.Issues = append(result.Issues, validateMultipartFiles(test, projectDir, filePath)...)

	return result
}

func validateVariableReferences(test *model.TestFile, creds *model.Credentials) []ValidationIssue {
	var issues []ValidationIssue

	// Regex to match variable references: $accounts.name.field, $var, ${var}, ${randomInt(1,100)}
	varPattern := regexp.MustCompile(`\$(accounts\.[a-zA-Z_][a-zA-Z0-9_]*\.[a-zA-Z_][a-zA-Z0-9_]*|[a-zA-Z_][a-zA-Z0-9_]*|\{[a-zA-Z_][a-zA-Z0-9_.]*(?:\([^}]*\))?\})`)

	// Collect variable sources
	savedVars := make(map[string]bool)
	setupSteps, stepsSteps, teardownSteps := collectAllSteps(test)
	allSteps := append(append(setupSteps, stepsSteps...), teardownSteps...)

	// Add built-in variable names
	builtinVars := map[string]bool{
		"uuid": true, "ulid": true, "randomInt": true, "randomEmail": true, "randomPhone": true,
	}

	for _, step := range allSteps {
		// Collect variables used in this step
		usedVars := extractVariables(&step, varPattern)

		for _, v := range usedVars {
			// Skip built-in variables
			if builtinVars[v] {
				continue
			}

			// Skip account references ($accounts.xxx.yyy)
			if strings.HasPrefix(v, "accounts.") {
				if creds != nil {
					// Extract account name and field
					parts := strings.Split(v, ".")
					if len(parts) >= 3 {
						accountName := parts[1]
						if _, exists := creds.GetAccount(accountName); !exists {
							issues = append(issues, ValidationIssue{
								Field: "variables",
								Code:  "undefined_account",
								Msg:   fmt.Sprintf("account %q not found in credentials for %q", accountName, v),
							})
						}
					}
				}
				continue
			}

			// Check if variable is saved from a previous step
			if !savedVars[v] {
				issues = append(issues, ValidationIssue{
					Field: "variables",
					Code:  "undefined_variable",
					Msg:   fmt.Sprintf("undefined variable: $%s", v),
				})
			}
		}

		// Now add this step's saved variables
		if step.Save != nil {
			for name := range step.Save {
				savedVars[name] = true
			}
		}
	}

	return issues
}

func extractVariables(step *model.Step, pattern *regexp.Regexp) []string {
	var vars []string

	extract := func(match string) string {
		v := strings.TrimPrefix(strings.TrimPrefix(match, "$"), "{")
		v = strings.TrimSuffix(v, "}")
		// Strip parenthesized arguments (e.g. randomInt(1,100) -> randomInt)
		if idx := strings.Index(v, "("); idx >= 0 {
			v = v[:idx]
		}
		return v
	}

	// Check request fields
	req := step.Request

	// URL
	for _, match := range pattern.FindAllString(req.URL, -1) {
		vars = append(vars, extract(match))
	}

	// Headers
	for _, v := range req.Headers {
		for _, match := range pattern.FindAllString(v, -1) {
			vars = append(vars, extract(match))
		}
	}

	// Body (if string)
	if bodyStr, ok := req.Body.(string); ok {
		for _, match := range pattern.FindAllString(bodyStr, -1) {
			vars = append(vars, extract(match))
		}
	}

	// Expect extra (for schema references, etc.)
	for key, val := range step.Expect.Extra {
		if _, ok := val.(string); ok && key == "schema" {
			continue // Schema is handled separately
		}
		valStr := fmt.Sprintf("%v", val)
		for _, match := range pattern.FindAllString(valStr, -1) {
			vars = append(vars, extract(match))
		}
	}

	return vars
}

func validateRetryConfig(test *model.TestFile) []ValidationIssue {
	var issues []ValidationIssue

	validBackoffs := map[string]bool{
		"constant": true, "linear": true, "exponential": true,
	}

	allSteps := append(append(test.Setup, test.Steps...), test.Teardown...)
	for i, step := range allSteps {
		if step.Retry != nil {
			if step.Retry.Backoff != "" {
				if !validBackoffs[strings.ToLower(step.Retry.Backoff)] {
					issues = append(issues, ValidationIssue{
						Field: fmt.Sprintf("steps[%d].retry.backoff", i),
						Code:  "invalid_backoff",
						Msg:   fmt.Sprintf("invalid backoff %q (valid: constant, linear, exponential)", step.Retry.Backoff),
					})
				}
			}
			if step.Retry.Attempts < 1 {
				issues = append(issues, ValidationIssue{
					Field: fmt.Sprintf("steps[%d].retry.attempts", i),
					Code:  "invalid_attempts",
					Msg:   fmt.Sprintf("retry attempts must be >= 1, got %d", step.Retry.Attempts),
				})
			}
		}
	}

	return issues
}

func validateUseFiles(test *model.TestFile, projectDir, currentFile string) []ValidationIssue {
	var issues []ValidationIssue

	currentDir := filepath.Dir(currentFile)

	for i, step := range test.Steps {
		if step.Use != "" {
			// Try to resolve the use path
			usePath := step.Use
			if !filepath.IsAbs(usePath) && filepath.Ext(usePath) == "" {
				usePath += ".yaml"
			}

			var found bool

			// Try relative to current file's directory
			fullPath := filepath.Join(currentDir, usePath)
			if _, err := os.Stat(fullPath); err == nil {
				found = true
			}

			// Try relative to .gherkio/tests/
			if !found {
				fullPath = filepath.Join(projectDir, ".gherkio", "tests", usePath)
				if _, err := os.Stat(fullPath); err == nil {
					found = true
				}
			}

			if !found {
				issues = append(issues, ValidationIssue{
					Field: fmt.Sprintf("steps[%d].use", i),
					Code:  "use_file_not_found",
					Msg:   fmt.Sprintf("referenced scenario not found: %s", step.Use),
				})
			}
		}
	}

	return issues
}

func validateSchemaReferences(test *model.TestFile, schemas []string) []ValidationIssue {
	var issues []ValidationIssue

	if len(schemas) == 0 {
		return issues // No schemas available to validate against
	}

	schemaSet := make(map[string]bool)
	for _, s := range schemas {
		schemaSet[s] = true
	}

	allSteps := append(append(test.Setup, test.Steps...), test.Teardown...)
	for i, step := range allSteps {
		if schemaVal, ok := step.Expect.Extra["schema"]; ok {
			if schemaName, ok := schemaVal.(string); ok {
				if !schemaSet[schemaName] {
					issues = append(issues, ValidationIssue{
						Field: fmt.Sprintf("steps[%d].expect.schema", i),
						Code:  "schema_not_found",
						Msg:   fmt.Sprintf("schema %q not found in .gherkio/schemas/", schemaName),
					})
				}
			}
		}
	}

	return issues
}

func validateBodyPaths(test *model.TestFile) []ValidationIssue {
	var issues []ValidationIssue

	// Check for double-dot paths like body..id
	doubleDotPattern := regexp.MustCompile(`\.\.`)

	allSteps := append(append(test.Setup, test.Steps...), test.Teardown...)

	// Check save paths
	for i, step := range allSteps {
		if step.Save != nil {
			for name, path := range step.Save {
				if doubleDotPattern.MatchString(path) {
					issues = append(issues, ValidationIssue{
						Field: fmt.Sprintf("steps[%d].save.%s", i, name),
						Code:  "invalid_path",
						Msg:   fmt.Sprintf("invalid path %q (double dot)", path),
					})
				}
			}
		}

		// Check expect paths
		for key := range step.Expect.Extra {
			if key == "schema" || key == "status" {
				continue
			}
			if doubleDotPattern.MatchString(key) {
				issues = append(issues, ValidationIssue{
					Field: fmt.Sprintf("steps[%d].expect.%s", i, key),
					Code:  "invalid_path",
					Msg:   fmt.Sprintf("invalid path %q (double dot)", key),
				})
			}
		}
	}

	return issues
}

// validateMultipartFiles checks that all file references in multipart configurations exist.
func validateMultipartFiles(test *model.TestFile, projectDir, testFilePath string) []ValidationIssue {
	var issues []ValidationIssue

	allSteps := append(append(test.Setup, test.Steps...), test.Teardown...)

	for i, step := range allSteps {
		if step.Request.Multipart == nil || step.Request.Multipart.Files == nil {
			continue
		}

		for fieldName, item := range step.Request.Multipart.Files {
			// Interpolate the path first to handle variables
			// Since we're doing static validation, we check the raw path
			// but note that variable-interpolated paths won't be validated statically
			filePath := item.Path

			// Try to resolve the file path according to Gherkio's resolution rules
			resolvedPath := resolveMultipartValidationPath(filePath, projectDir, testFilePath)
			if resolvedPath == "" {
				issues = append(issues, ValidationIssue{
					Field: fmt.Sprintf("steps[%d].request.multipart.files.%s", i, fieldName),
					Code:  "file_not_found",
					Msg:   fmt.Sprintf("multipart file %q does not exist", filePath),
				})
			}
		}
	}

	return issues
}

// resolveMultipartValidationPath resolves a file path for validation purposes.
// It checks: absolute paths, project root relative, fixtures/, and sibling test fixtures/.
func resolveMultipartValidationPath(filePath, projectDir, testFilePath string) string {
	// If already absolute and exists, use it
	if filepath.IsAbs(filePath) {
		if _, err := os.Stat(filePath); err == nil {
			return filePath
		}
		return ""
	}

	// Try project root relative path
	if projectDir != "" {
		absPath := filepath.Join(projectDir, filePath)
		if _, err := os.Stat(absPath); err == nil {
			return absPath
		}
	}

	// Try fixtures directory
	if projectDir != "" {
		fixturesPath := filepath.Join(projectDir, "fixtures", filepath.Base(filePath))
		if _, err := os.Stat(fixturesPath); err == nil {
			return fixturesPath
		}
	}

	// Try test file's sibling fixtures directory (tests/fixtures/ relative to test file)
	if testFilePath != "" {
		testDir := filepath.Dir(testFilePath)
		siblingFixtures := filepath.Join(testDir, "fixtures", filepath.Base(filePath))
		if _, err := os.Stat(siblingFixtures); err == nil {
			return siblingFixtures
		}
	}

	// Try as-is relative to current working directory
	if _, err := os.Stat(filePath); err == nil {
		return filePath
	}

	return ""
}

func collectAllSteps(test *model.TestFile) (setup, steps, teardown []model.Step) {
	return test.Setup, test.Steps, test.Teardown
}

func discoverTestFilesForValidate(dir string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".yaml" {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", dir, err)
	}

	sort.Strings(files)
	return files, nil
}

func resolveTestPathForValidate(cwd, projectDir, testPath string) (string, error) {
	// Try as-is
	if filepath.IsAbs(testPath) {
		if _, err := os.Stat(testPath); err == nil {
			return testPath, nil
		}
		return "", fmt.Errorf("file not found: %s", testPath)
	}

	// Try relative to cwd
	fullPath := filepath.Join(cwd, testPath)
	if _, err := os.Stat(fullPath); err == nil {
		return fullPath, nil
	}

	// Try relative to .gherkio/tests/
	fullPath = filepath.Join(projectDir, ".gherkio", "tests", testPath)
	if _, err := os.Stat(fullPath); err == nil {
		return fullPath, nil
	}

	return "", fmt.Errorf("test file '%s' not found (checked: cwd and .gherkio/tests/)", testPath)
}

func loadAvailableSchemas(projectDir string) ([]string, error) {
	meta, err := GetMeta(projectDir)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(meta.SchemasDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var schemas []string
	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			ext := filepath.Ext(name)
			if ext == ".yaml" || ext == ".yml" {
				schemas = append(schemas, strings.TrimSuffix(name, ext))
			}
		}
	}

	sort.Strings(schemas)
	return schemas, nil
}
