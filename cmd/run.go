package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/muhfaris/gherkio/internal/model"
	"github.com/muhfaris/gherkio/internal/report"
	"github.com/muhfaris/gherkio/internal/runner"
	"github.com/spf13/cobra"
)

var (
	envName      string
	verbose      bool
	reportFormat string
	reportRaw    bool
)

// runCmd represents the gherkio run command.
var runCmd = &cobra.Command{
	Use:   "run [test-file]",
	Short: "Execute a test scenario",
	Long: `Executes a Gherkio test YAML file and displays the results.

If no test file is provided, all tests in .gherkio/tests/ are executed.
If a directory is provided, all tests in that directory are executed.

The test file path is resolved in the following order:
1. As provided (relative to current working directory)
2. Relative to .gherkio/tests/ directory

Example:
  gherkio run                    # Run all tests
  gherkio run tests/login.yaml
  gherkio run restful-api/       # Run all tests in restful-api/ directory
  gherkio run login.yaml --env staging
  gherkio run login.yaml --verbose
  gherkio run login.yaml --report html`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		testPath := ""
		if len(args) > 0 {
			testPath = args[0]
		}
		return runTest(testPath, envName, verbose, reportFormat, reportRaw)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&envName, "env", "e", "local", "Environment to use (e.g. local, staging, production)")
	runCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show full request/response payloads")
	runCmd.Flags().StringVar(&reportFormat, "report", "", "Generate a report (format: html, json, or html,json)")
	runCmd.Flags().BoolVar(&reportRaw, "report-raw", false, "Skip sensitive data masking in JSON reports (cURL commands remain masked)")
}

func runTest(testPath, env string, verbose bool, reportFormat string, reportRaw bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find project root (directory containing .gherkio)
	projectDir, err := findProjectRoot(cwd)
	if err != nil {
		return fmt.Errorf("not inside a Gherkio project: %w", err)
	}

	// Load Configuration
	appCfg, err := runner.LoadConfig(projectDir)
	if err != nil {
		// Log but don't fail if config is broken/missing, fallback to defaults
		fmt.Printf("Warning: failed to load config: %v\n", err)
	}

	// Merge CLI with Config for Report Format
	if reportFormat == "" && appCfg != nil && appCfg.Reports.Format != "" {
		reportFormat = appCfg.Reports.Format
	}

	maskFields := runner.GetDefaultSensitiveFields()
	if appCfg != nil && appCfg.Security.Mask.Enabled && len(appCfg.Security.Mask.Fields) > 0 {
		maskFields = appCfg.Security.Mask.Fields
	}

	// Build common report configuration
	var reportCfg *report.ReportConfig
	if reportFormat != "" {
		reportCfg = &report.ReportConfig{
			Format:        reportFormat,
			Path:          "",
			MaskSensitive: !reportRaw,
			MaskFields:    maskFields,
		}
		if appCfg != nil {
			reportCfg.Path = appCfg.Reports.Path
			reportCfg.MaskSensitive = appCfg.Reports.MaskSensitive
		}
	}

	// No test path or directory: run all tests
	if testPath == "" {
		return runAllTests(projectDir, env, verbose, reportCfg, maskFields)
	}

	// Check if the path is a directory
	fullPath := filepath.Join(cwd, testPath)
	if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
		testDir := fullPath
		// Also try resolving relative to .gherkio/tests/
		altPath := filepath.Join(projectDir, ".gherkio", "tests", testPath)
		if altInfo, altErr := os.Stat(altPath); altErr == nil && altInfo.IsDir() {
			testDir = altPath
		}
		return runAllInDir(testDir, projectDir, env, verbose, reportCfg, maskFields)
	}

	// Resolve single test file path
	fullPath, err = resolveTestPath(cwd, projectDir, testPath)
	if err != nil {
		return fmt.Errorf("test file not found: %w", err)
	}

	return runSingleTest(fullPath, projectDir, env, verbose, reportCfg, maskFields)
}

func handleReport(result *runner.RunResult, projectDir string, env string, reportCfg *report.ReportConfig) {
	if reportCfg == nil {
		return
	}

	formats := strings.Split(reportCfg.Format, ",")
	for _, format := range formats {
		format = strings.TrimSpace(format)
		switch format {
		case "html":
			html, err := report.RenderHTML(result, *reportCfg, env)
			if err != nil {
				fmt.Printf("Failed to render HTML report: %v\n", err)
				continue
			}
			savedPath, err := report.SaveHTML(html, projectDir, reportCfg.Path)
			if err != nil {
				fmt.Printf("Failed to save HTML report: %v\n", err)
				continue
			}
			fmt.Printf("📄 HTML Report saved: %s\n", savedPath)
		case "json":
			jsonStr, err := report.RenderJSON(result, *reportCfg, env)
			if err != nil {
				fmt.Printf("Failed to render JSON report: %v\n", err)
				continue
			}
			savedPath, err := report.SaveJSON(jsonStr, projectDir, reportCfg.Path)
			if err != nil {
				fmt.Printf("Failed to save JSON report: %v\n", err)
				continue
			}
			fmt.Printf("📄 JSON Report saved: %s\n", savedPath)
		}
	}
}

func runSingleTest(testPath, projectDir, env string, verbose bool, reportCfg *report.ReportConfig, maskFields []string) error {
	cfg := runner.RunConfig{
		TestPath:   testPath,
		EnvName:    env,
		ProjectDir: projectDir,
		Verbose:    verbose,
		MaskFields: maskFields,
	}

	result, err := runner.Run(cfg)
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	runner.PrintResult(result, cfg.Verbose, cfg.MaskFields)

	handleReport(result, projectDir, env, reportCfg)

	if !result.Passed {
		os.Exit(1)
	}

	return nil
}

func runAllTests(projectDir, env string, verbose bool, reportCfg *report.ReportConfig, maskFields []string) error {
	testsDir := filepath.Join(projectDir, ".gherkio", "tests")
	return runAllInDir(testsDir, projectDir, env, verbose, reportCfg, maskFields)
}

func runAllInDir(testDir, projectDir, env string, verbose bool, reportCfg *report.ReportConfig, maskFields []string) error {
	files, err := discoverTestFiles(testDir)
	if err != nil {
		return fmt.Errorf("failed to discover test files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No test files found.")
		return nil
	}

	totalPass := 0
	totalFail := 0
	anyFailed := false

	fmt.Printf("Running %d test(s)...\n\n", len(files))

	// For multiple tests, we technically need an aggregate report.
	// For Phase 1 RFC, let's just create an aggregate RunResult and generate a single report.
	var allSteps []runner.StepResult
	totalDuration := int64(0)

	for i, file := range files {
		relPath, _ := filepath.Rel(projectDir, file)

		cfg := runner.RunConfig{
			TestPath:   file,
			EnvName:    env,
			ProjectDir: projectDir,
			Verbose:    verbose,
			MaskFields: maskFields,
		}

		result, err := runner.Run(cfg)
		if err != nil {
			fmt.Printf("[%d/%d] ✗ %s — error: %v\n", i+1, len(files), relPath, err)
			anyFailed = true
			totalFail++
			// Create a pseudo-step for the failed file execution
			failedStep := runner.StepResult{
				Original: model.Step{Request: model.Request{URL: relPath}},
				Error:    err.Error(),
			}
			allSteps = append(allSteps, failedStep)
			continue
		}

		runner.PrintResult(result, cfg.Verbose, cfg.MaskFields)

		totalPass += result.TotalPass
		totalFail += result.TotalFail
		if !result.Passed {
			anyFailed = true
		}

		allSteps = append(allSteps, result.Steps...)
		totalDuration += result.Duration.Nanoseconds()

		if i < len(files)-1 {
			fmt.Println()
		}
	}

	// Combined summary
	total := totalPass + totalFail
	statusIcon := "✓"
	statusWord := "PASS"
	if anyFailed {
		statusIcon = "✗"
		statusWord = "FAIL"
	}

	fmt.Println(strings.Repeat("═", 40))
	fmt.Printf("%s %s — across %d scenario(s)\n", statusIcon, statusWord, len(files))
	fmt.Printf("%d passed, %d failed, %d total assertions\n", totalPass, totalFail, total)

	// Generate combined report
	if reportCfg != nil {
		combinedResult := &runner.RunResult{
			Scenario:  "Test Suite Run",
			Passed:    !anyFailed,
			Steps:     allSteps,
			TotalPass: totalPass,
			TotalFail: totalFail,
			Duration:  time.Duration(totalDuration),
		}
		handleReport(combinedResult, projectDir, env, reportCfg)
	}

	if anyFailed {
		os.Exit(1)
	}

	return nil
}

// discoverTestFiles finds all .yaml files recursively in the given directory.
func discoverTestFiles(dir string) ([]string, error) {
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

	// Sort for deterministic ordering
	sort.Strings(files)

	return files, nil
}

// findProjectRoot walks up from cwd to find a directory containing .gherkio.
func findProjectRoot(cwd string) (string, error) {
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, ".gherkio")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no .gherkio directory found in any parent directory")
		}
		dir = parent
	}
}

// resolveTestPath tries to find the test file.
func resolveTestPath(cwd, projectDir, testPath string) (string, error) {
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
