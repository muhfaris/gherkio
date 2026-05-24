package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/muhfaris/gherkio/internal/core/project"
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
	accountName  string
	allAccounts  bool
	stepIdx      int
	lineNum      int
	stepSection  string
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
  gherkio run login.yaml --report html
  gherkio run login.yaml --env staging --account alpha
  gherkio run login.yaml --env staging --all-accounts`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		testPath := ""
		if len(args) > 0 {
			testPath = args[0]
		}
		return runTest(testPath, envName, verbose, reportFormat, reportRaw, accountName, allAccounts)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&envName, "env", "e", "local", "Environment to use (e.g. local, staging, production)")
	runCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show full request/response payloads")
	runCmd.Flags().StringVar(&reportFormat, "report", "", "Generate a report (format: html, json, or html,json)")
	runCmd.Flags().BoolVar(&reportRaw, "report-raw", false, "Skip sensitive data masking in JSON reports (cURL commands remain masked)")
	runCmd.Flags().StringVar(&accountName, "account", "", "Account name from credentials file (e.g. alpha, beta)")
	runCmd.Flags().BoolVar(&allAccounts, "all-accounts", false, "Run tests against all accounts in the credentials file")
	runCmd.Flags().IntVar(&stepIdx, "step", -1, "Index of the step to run (0-indexed)")
	runCmd.Flags().IntVar(&lineNum, "line", -1, "Line number containing the step to run")
	runCmd.Flags().StringVar(&stepSection, "section", "steps", "Section containing the step (setup, steps, teardown)")
}

func runTest(testPath, env string, verbose bool, reportFormat string, reportRaw bool, accountName string, allAccounts bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find project root (directory containing .gherkio)
	projectDir, err := project.FindRoot(cwd)
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

	// Load credentials for the environment
	creds, err := runner.LoadCredentials(projectDir, env)
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	// Warn if --account is specified but no credentials file exists
	if accountName != "" && creds == nil {
		fmt.Printf("⚠ No credentials file found for environment %q at .gherkio/credentials/%s.yaml. --account flag ignored.\n\n", env, env)
	}

	// Validate account usage
	if accountName != "" && creds != nil {
		// Check if specified account exists
		if _, exists := creds.GetAccount(accountName); !exists {
			available := creds.AccountNames()
			return fmt.Errorf("account %q not found in .gherkio/credentials/%s.yaml\n  Available accounts: %s", accountName, env, strings.Join(available, ", "))
		}
	}

	// No test path or directory: run all tests
	if testPath == "" {
		return runAllTests(projectDir, env, verbose, reportCfg, maskFields, creds, accountName, allAccounts)
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
		return runAllInDir(testDir, projectDir, env, verbose, reportCfg, maskFields, creds, accountName, allAccounts)
	}

	// Resolve single test file path
	fullPath, err = resolveTestPath(cwd, projectDir, testPath)
	if err != nil {
		return fmt.Errorf("test file not found: %w", err)
	}

	return runSingleTest(fullPath, projectDir, env, verbose, reportCfg, maskFields, creds, accountName, allAccounts)
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

func handleSuiteReport(results []*runner.RunResult, projectDir string, env string, reportCfg *report.ReportConfig) {
	if reportCfg == nil || len(results) == 0 {
		return
	}

	// If only one result, use the single-scenario path
	if len(results) == 1 {
		handleReport(results[0], projectDir, env, reportCfg)
		return
	}

	formats := strings.Split(reportCfg.Format, ",")
	for _, format := range formats {
		format = strings.TrimSpace(format)
		switch format {
		case "html":
			html, err := report.RenderHTMLSuite(results, *reportCfg, env)
			if err != nil {
				fmt.Printf("Failed to render HTML suite report: %v\n", err)
				continue
			}
			savedPath, err := report.SaveHTML(html, projectDir, reportCfg.Path)
			if err != nil {
				fmt.Printf("Failed to save HTML suite report: %v\n", err)
				continue
			}
			fmt.Printf("📄 HTML Report saved: %s\n", savedPath)
		case "json":
			jsonStr, err := report.RenderJSONSuite(results, *reportCfg, env)
			if err != nil {
				fmt.Printf("Failed to render JSON suite report: %v\n", err)
				continue
			}
			savedPath, err := report.SaveJSON(jsonStr, projectDir, reportCfg.Path)
			if err != nil {
				fmt.Printf("Failed to save JSON suite report: %v\n", err)
				continue
			}
			fmt.Printf("📄 JSON Report saved: %s\n", savedPath)
		}
	}
}

func runSingleTest(testPath, projectDir, env string, verbose bool, reportCfg *report.ReportConfig, maskFields []string, creds *model.Credentials, accountName string, allAccounts bool) error {
	// Determine which accounts to run
	accounts, err := resolveAccounts(creds, accountName, allAccounts)
	if err != nil {
		return err
	}

	// If multiple accounts, run each sequentially
	if len(accounts) > 1 {
		return runSingleTestMultiAccount(testPath, projectDir, env, verbose, reportCfg, maskFields, accounts)
	}

	// Single account (or no credentials)
	var credentialVars map[string]interface{}
	var accName string
	if len(accounts) == 1 {
		for name, acc := range accounts {
			accName = name
			credentialVars = runner.CredentialsToVars(acc)
		}
		// Merge credential-sensitive fields into mask
		for _, acc := range accounts {
			maskFields = append(maskFields, runner.GetSensitiveFieldsFromCredentials(acc)...)
		}
	}

	targetStepIdx := stepIdx
	targetSection := stepSection
	if lineNum >= 0 {
		loc, err := runner.LocateStep(testPath, lineNum)
		if err != nil {
			return err
		}
		targetStepIdx = loc.Index
		targetSection = loc.Section
		fmt.Printf("🎯 Line %d resolved to %s step %d\n", lineNum, targetSection, targetStepIdx)
	}

	cfg := runner.RunConfig{
		TestPath:       testPath,
		EnvName:        env,
		ProjectDir:     projectDir,
		Verbose:        verbose,
		MaskFields:     maskFields,
		AccountName:    accName,
		CredentialVars: credentialVars,
		StepIndex:      targetStepIdx,
		StepSection:    targetSection,
	}

	result, err := runner.Run(cfg)
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	if targetStepIdx >= 0 {
		runner.PrintStepResult(result, cfg.Verbose, cfg.MaskFields)
	} else {
		runner.PrintResult(result, cfg.Verbose, cfg.MaskFields)
	}

	handleReport(result, projectDir, env, reportCfg)

	if !result.Passed {
		os.Exit(1)
	}

	return nil
}

// resolveAccounts determines which accounts to use based on credentials, flags, and edge cases.
// Returns a map of account name → Account for convenient name access.
func resolveAccounts(creds *model.Credentials, accountName string, allAccounts bool) (map[string]model.Account, error) {
	// No credentials file - run with no account
	if creds == nil {
		return nil, nil
	}

	accountNames := creds.AccountNames()

	// If --account specified, use just that account
	if accountName != "" {
		account, exists := creds.GetAccount(accountName)
		if !exists {
			return nil, fmt.Errorf("account %q not found in credentials\n  Available: %s", accountName, strings.Join(accountNames, ", "))
		}
		return map[string]model.Account{accountName: account}, nil
	}

	// If --all-accounts, use all accounts
	if allAccounts {
		accounts := make(map[string]model.Account, len(creds.Accounts))
		for _, name := range accountNames {
			accounts[name] = creds.Accounts[name]
		}
		return accounts, nil
	}

	// No flags - check edge cases
	if len(accountNames) == 0 {
		return nil, nil // No accounts defined
	}

	if len(accountNames) == 1 {
		// Single account - auto-use it
		name := accountNames[0]
		return map[string]model.Account{name: creds.Accounts[name]}, nil
	}

	// Multiple accounts but no flag - print hint and continue without credentials
	fmt.Printf("⚠ %d accounts found in credentials. Use --account <name> or --all-accounts to use them.\n\n", len(accountNames))
	return nil, nil
}

// runSingleTestMultiAccount runs a single test file against multiple accounts.
func runSingleTestMultiAccount(testPath, projectDir, env string, verbose bool, reportCfg *report.ReportConfig, maskFields []string, accounts map[string]model.Account) error {
	var allResults []*runner.RunResult
	totalPass := 0
	totalFail := 0

	accountCount := len(accounts)
	i := 0
	for accountName, account := range accounts {
		i++
		// Merge credential-sensitive fields into mask for this account
		accountMask := append([]string{}, maskFields...)
		accountMask = append(accountMask, runner.GetSensitiveFieldsFromCredentials(account)...)

		fmt.Printf("Running account: %s (%d/%d)\n\n", accountName, i, accountCount)

		cfg := runner.RunConfig{
			TestPath:       testPath,
			EnvName:        env,
			ProjectDir:     projectDir,
			Verbose:        verbose,
			MaskFields:     accountMask,
			AccountName:    accountName,
			CredentialVars: runner.CredentialsToVars(account),
		}

		result, err := runner.Run(cfg)
		if err != nil {
			fmt.Printf("✗ Error running with account %s: %v\n", accountName, err)
			totalFail++
			failedResult := &runner.RunResult{
				Scenario:  filepath.Base(testPath),
				TestFile:  testPath,
				Account:   accountName,
				Passed:    false,
				TotalFail: 1,
			}
			allResults = append(allResults, failedResult)
		} else {
			runner.PrintResult(result, cfg.Verbose, cfg.MaskFields)
			totalPass += result.TotalPass
			totalFail += result.TotalFail
			allResults = append(allResults, result)
		}

		if i < accountCount {
			fmt.Println()
		}
	}

	// Combined summary
	statusIcon := "✓"
	statusWord := "PASS"
	if totalFail > 0 {
		statusIcon = "✗"
		statusWord = "FAIL"
	}

	fmt.Println(strings.Repeat("═", 40))
	fmt.Printf("%s %s — across %d account(s)\n", statusIcon, statusWord, accountCount)
	fmt.Printf("%d passed, %d failed, %d total assertions\n", totalPass, totalFail, totalPass+totalFail)

	// Generate suite report
	if reportCfg != nil {
		handleSuiteReport(allResults, projectDir, env, reportCfg)
	}

	if totalFail > 0 {
		os.Exit(1)
	}

	return nil
}

func runAllTests(projectDir, env string, verbose bool, reportCfg *report.ReportConfig, maskFields []string, creds *model.Credentials, accountName string, allAccounts bool) error {
	testsDir := filepath.Join(projectDir, ".gherkio", "tests")
	return runAllInDir(testsDir, projectDir, env, verbose, reportCfg, maskFields, creds, accountName, allAccounts)
}

func runAllInDir(testDir, projectDir, env string, verbose bool, reportCfg *report.ReportConfig, maskFields []string, creds *model.Credentials, accountName string, allAccounts bool) error {
	files, err := discoverTestFiles(testDir)
	if err != nil {
		return fmt.Errorf("failed to discover test files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No test files found.")
		return nil
	}

	// Determine which accounts to run
	accounts, err := resolveAccounts(creds, accountName, allAccounts)
	if err != nil {
		return err
	}

	// If multiple accounts, run each test against all accounts
	if len(accounts) > 1 {
		return runAllInDirMultiAccount(testDir, projectDir, env, verbose, reportCfg, maskFields, files, accounts)
	}

	// Single account or no accounts - run each test file
	totalPass := 0
	totalFail := 0
	anyFailed := false

	fmt.Printf("Running %d test(s)...\n\n", len(files))

	// Collect individual results for suite-level reporting
	var results []*runner.RunResult

	for i, file := range files {
		relPath, _ := filepath.Rel(projectDir, file)

		var credentialVars map[string]interface{}
		var accName string
		if len(accounts) == 1 {
			for name, acc := range accounts {
				accName = name
				credentialVars = runner.CredentialsToVars(acc)
			}
			// Merge credential-sensitive fields into mask
			for _, acc := range accounts {
				maskFields = append(maskFields, runner.GetSensitiveFieldsFromCredentials(acc)...)
			}
		}

		cfg := runner.RunConfig{
			TestPath:       file,
			EnvName:        env,
			ProjectDir:     projectDir,
			Verbose:        verbose,
			MaskFields:     maskFields,
			AccountName:    accName,
			CredentialVars: credentialVars,
		}

		result, err := runner.Run(cfg)
		if err != nil {
			fmt.Printf("[%d/%d] ✗ %s — error: %v\n", i+1, len(files), relPath, err)
			anyFailed = true
			totalFail++
			// Create a pseudo-result for the failed file
			failedResult := &runner.RunResult{
				Scenario:  filepath.Base(relPath),
				TestFile:  file,
				Account:   accName,
				Passed:    false,
				TotalFail: 1,
				Steps: []runner.StepResult{
					{
						Original:     model.Step{Request: model.Request{URL: relPath}},
						ScenarioName: filepath.Base(relPath),
						TestFile:     file,
						Error:        err.Error(),
					},
				},
			}
			results = append(results, failedResult)
			continue
		}

		runner.PrintResult(result, cfg.Verbose, cfg.MaskFields)

		totalPass += result.TotalPass
		totalFail += result.TotalFail
		if !result.Passed {
			anyFailed = true
		}

		results = append(results, result)

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

	// Generate suite report (keeps scenarios grouped)
	if reportCfg != nil {
		handleSuiteReport(results, projectDir, env, reportCfg)
	}

	if anyFailed {
		os.Exit(1)
	}

	return nil
}

// runAllInDirMultiAccount runs all test files against multiple accounts.
func runAllInDirMultiAccount(testDir, projectDir, env string, verbose bool, reportCfg *report.ReportConfig, maskFields []string, files []string, accounts map[string]model.Account) error {
	var allResults []*runner.RunResult
	totalPass := 0
	totalFail := 0
	accountCount := len(accounts)
	totalScenarios := len(files) * accountCount

	fmt.Printf("Running %d test(s) against %d account(s)...\n\n", len(files), accountCount)

	for i, file := range files {
		relPath, _ := filepath.Rel(projectDir, file)

		j := 0
		for accountName, account := range accounts {
			j++
			// Merge credential-sensitive fields into mask for this account
			accountMask := append([]string{}, maskFields...)
			accountMask = append(accountMask, runner.GetSensitiveFieldsFromCredentials(account)...)

			scenarioIdx := i*accountCount + j
			fmt.Printf("[%d/%d] Running %s with account: %s\n", scenarioIdx, totalScenarios, relPath, accountName)

			cfg := runner.RunConfig{
				TestPath:       file,
				EnvName:        env,
				ProjectDir:     projectDir,
				Verbose:        verbose,
				MaskFields:     accountMask,
				AccountName:    accountName,
				CredentialVars: runner.CredentialsToVars(account),
			}

			result, err := runner.Run(cfg)
			if err != nil {
				fmt.Printf("  ✗ Error: %v\n", err)
				totalFail++
				failedResult := &runner.RunResult{
					Scenario:  filepath.Base(relPath),
					TestFile:  file,
					Account:   accountName,
					Passed:    false,
					TotalFail: 1,
					Steps: []runner.StepResult{
						{
							Original:     model.Step{Request: model.Request{URL: relPath}},
							ScenarioName: filepath.Base(relPath),
							TestFile:     file,
							Error:        err.Error(),
						},
					},
				}
				allResults = append(allResults, failedResult)
				continue
			}

			// Print compact result
			statusIcon := "✓"
			if !result.Passed {
				statusIcon = "✗"
				totalFail++
			} else {
				totalPass++
			}
			fmt.Printf("  %s %s — %d passed, %d failed\n", statusIcon, accountName, result.TotalPass, result.TotalFail)

			allResults = append(allResults, result)
		}

		if i < len(files)-1 {
			fmt.Println()
		}
	}

	// Combined summary
	statusIcon := "✓"
	statusWord := "PASS"
	if totalFail > 0 {
		statusIcon = "✗"
		statusWord = "FAIL"
	}

	fmt.Println(strings.Repeat("═", 40))
	fmt.Printf("%s %s — across %d scenario(s) and %d account(s)\n", statusIcon, statusWord, len(files), accountCount)
	fmt.Printf("%d passed, %d failed, %d total assertions\n", totalPass, totalFail, totalPass+totalFail)

	// Generate suite report
	if reportCfg != nil {
		handleSuiteReport(allResults, projectDir, env, reportCfg)
	}

	if totalFail > 0 {
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
