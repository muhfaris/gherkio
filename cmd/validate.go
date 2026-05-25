package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/muhfaris/gherkio/internal/core/project"
	"github.com/muhfaris/gherkio/internal/core/teststore"
	"github.com/muhfaris/gherkio/internal/model"
	"github.com/muhfaris/gherkio/internal/runner"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	validateVerbose bool
	validateEnv     string
)

var validateCmd = &cobra.Command{
	Use:   "validate [test-file]",
	Short: "Validate a test file without executing HTTP requests",
	Long: `Performs static analysis on Gherkio test YAML files without making HTTP calls.

Validates:
- YAML syntax and structure
- Scenario structure (required fields)
- Request method validity
- Variable references ($var, $accounts.*, built-in generators)
- Account references from credentials
- Schema references from .gherkio/schemas/
- use file existence
- Retry configuration

Example:
  gherkio validate                           # Validate all test files
  gherkio validate login.yaml                # Validate single file
  gherkio validate --verbose                 # Show detailed results
  gherkio validate --env staging             # Validate with staging credentials`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		testPath := ""
		if len(args) > 0 {
			testPath = args[0]
		}
		return runValidate(testPath, validateVerbose, validateEnv)
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
	validateCmd.Flags().BoolVarP(&validateVerbose, "verbose", "v", false, "Show detailed validation results")
	validateCmd.Flags().StringVarP(&validateEnv, "env", "e", "local", "Environment for credentials validation")
}

type ValidationIssue struct {
	File  string
	Field string
	Code  string
	Msg   string
}

type ValidationResult struct {
	File   string
	Issues []ValidationIssue
}

func runValidate(testPath string, verbose bool, env string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	projectDir, err := project.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("not inside a Gherkio project: %w", err)
	}

	var files []string

	if testPath == "" {
		// Validate all test files
		testsDir := filepath.Join(projectDir, ".gherkio", "tests")
		files, err = discoverTestFilesForValidate(testsDir)
		if err != nil {
			return fmt.Errorf("failed to discover test files: %w", err)
		}
		if len(files) == 0 {
			fmt.Println("No test files found to validate.")
			return nil
		}
	} else {
		// Validate single file
		fullPath, err := resolveTestPathForValidate(cwd, projectDir, testPath)
		if err != nil {
			return fmt.Errorf("test file not found: %w", err)
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

	var allResults []ValidationResult
	totalIssues := 0

	for _, file := range files {
		result := validateFile(file, projectDir, creds, schemas)
		allResults = append(allResults, result)
		totalIssues += len(result.Issues)
	}

	// Print results
	if verbose {
		fmt.Println("═══ Validation Results ═══")
		fmt.Println()
	}

	for _, result := range allResults {
		relPath, _ := filepath.Rel(projectDir, result.File)

		if verbose {
			if len(result.Issues) == 0 {
				fmt.Printf("✓ %s — valid\n", relPath)
			} else {
				fmt.Printf("✗ %s — %d issue(s)\n", relPath, len(result.Issues))
				for _, issue := range result.Issues {
					fmt.Printf("  [%s] %s: %s\n", issue.Code, issue.Field, issue.Msg)
				}
			}
		} else {
			if len(result.Issues) == 0 {
				fmt.Printf("✓ %s — valid\n", relPath)
			} else {
				fmt.Printf("✗ %s: %s\n", relPath, result.Issues[0].Msg)
				if len(result.Issues) > 1 {
					for _, issue := range result.Issues[1:] {
						fmt.Printf("  %s: %s\n", issue.Field, issue.Msg)
					}
				}
			}
		}
	}

	// Summary
	fmt.Println()
	if totalIssues == 0 {
		fmt.Printf("✓ All %d test file(s) are valid\n", len(files))
	} else {
		fmt.Printf("✗ Found %d issue(s) in %d test file(s)\n", totalIssues, len(allResults))
		for _, r := range allResults {
			if len(r.Issues) > 0 {
				fmt.Printf("  - %s: %d issue(s)\n", r.File, len(r.Issues))
			}
		}
		if totalIssues > 0 {
			os.Exit(1)
		}
	}

	return nil
}

func validateFile(filePath, projectDir string, creds *model.Credentials, schemas []string) ValidationResult {
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
	storeResult, err := teststore.Validate(test, projectDir)
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

	return result
}

func validateVariableReferences(test *model.TestFile, creds *model.Credentials) []ValidationIssue {
	var issues []ValidationIssue

	// Regex to match variable references: $var, ${var}, $accounts.name.field
	varPattern := regexp.MustCompile(`\$([a-zA-Z_][a-zA-Z0-9_]*|\{[a-zA-Z_][a-zA-Z0-9_.]*\}|accounts\.[a-zA-Z_][a-zA-Z0-9_]*\.[a-zA-Z_][a-zA-Z0-9_]*)`)

	// Collect variable sources
	savedVars := make(map[string]bool)
	setupSteps, stepsSteps, teardownSteps := collectAllSteps(test)
	allSteps := append(append(setupSteps, stepsSteps...), teardownSteps...)

	// First pass: collect all saved variables (from previous steps only)
	visitedOrder := make(map[*model.Step]bool)

	// Add built-in variable names
	builtinVars := map[string]bool{
		"uuid": true, "ulid": true, "randomInt": true, "randomEmail": true, "randomPhone": true,
	}

	for _, step := range allSteps {
		// Collect variables used in this step
		usedVars := extractVariables(step, varPattern)

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
						fieldName := parts[2]
						if _, exists := creds.GetAccount(accountName); !exists {
							issues = append(issues, ValidationIssue{
								Field:   "variables",
								Code:    "undefined_account",
								Msg:     fmt.Sprintf("account %q not found in credentials for %q", accountName, v),
							})
						} else {
							// Check if field exists
							acc, _ := creds.GetAccount(accountName)
							if acc != nil {
								fieldExists := false
								switch acc.(type) {
								case map[string]interface{}:
									if _, ok := acc.(map[string]interface{})[fieldName]; ok {
										fieldExists = true
									}
								}
								_ = fieldExists // We don't require fields to exist in validation
							}
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

	// Check request fields
	req := step.Request

	// URL
	for _, match := range pattern.FindAllString(req.URL, -1) {
		vars = append(vars, strings.TrimPrefix(strings.TrimPrefix(match, "$"), "{"))
	}

	// Headers
	for _, v := range req.Headers {
		for _, match := range pattern.FindAllString(v, -1) {
			v := strings.TrimPrefix(strings.TrimPrefix(match, "$"), "{")
			vars = append(vars, v)
		}
	}

	// Body (if string)
	if bodyStr, ok := req.Body.(string); ok {
		for _, match := range pattern.FindAllString(bodyStr, -1) {
			v := strings.TrimPrefix(strings.TrimPrefix(match, "$"), "{")
			vars = append(vars, v)
		}
	}

	// Expect extra (for schema references, etc.)
	for key, val := range step.Expect.Extra {
		if schemaVal, ok := val.(string); ok && key == "schema" {
			continue // Schema is handled separately
		}
		valStr := fmt.Sprintf("%v", val)
		for _, match := range pattern.FindAllString(valStr, -1) {
			v := strings.TrimPrefix(strings.TrimPrefix(match, "$"), "{")
			vars = append(vars, v)
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

			var resolvedPath string
			var found bool

			// Try relative to current file's directory
			fullPath := filepath.Join(currentDir, usePath)
			if _, err := os.Stat(fullPath); err == nil {
				resolvedPath = fullPath
				found = true
			}

			// Try relative to .gherkio/tests/
			if !found {
				fullPath = filepath.Join(projectDir, ".gherkio", "tests", usePath)
				if _, err := os.Stat(fullPath); err == nil {
					resolvedPath = fullPath
					found = true
				}
			}

			if !found {
				issues = append(issues, ValidationIssue{
					Field:   fmt.Sprintf("steps[%d].use", i),
					Code:    "use_file_not_found",
					Msg:     fmt.Sprintf("referenced scenario not found: %s", step.Use),
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
						Field:   fmt.Sprintf("steps[%d].expect.schema", i),
						Code:    "schema_not_found",
						Msg:     fmt.Sprintf("schema %q not found in .gherkio/schemas/", schemaName),
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
						Field:   fmt.Sprintf("steps[%d].save.%s", i, name),
						Code:    "invalid_path",
						Msg:     fmt.Sprintf("invalid path %q (double dot)", path),
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
					Field:   fmt.Sprintf("steps[%d].expect.%s", i, key),
					Code:    "invalid_path",
					Msg:     fmt.Sprintf("invalid path %q (double dot)", key),
				})
			}
		}
	}

	return issues
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
	meta, err := project.GetMeta(projectDir)
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