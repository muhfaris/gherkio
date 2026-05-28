package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/muhfaris/gherkio/internal/core/project"
	"github.com/spf13/cobra"
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

func runValidate(testPath string, verbose bool, env string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	projectDir, err := project.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("not inside a Gherkio project: %w", err)
	}

	results, err := project.ValidateProject(cwd, projectDir, testPath, env)
	if err != nil {
		return err
	}

	totalIssues := 0
	for _, r := range results {
		totalIssues += len(r.Issues)
	}

	// Print results
	if verbose {
		fmt.Println("═══ Validation Results ═══")
		fmt.Println()
	}

	for _, result := range results {
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
		fmt.Printf("✓ All %d test file(s) are valid\n", len(results))
	} else {
		fmt.Printf("✗ Found %d issue(s) in %d test file(s)\n", totalIssues, len(results))
		for _, r := range results {
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
