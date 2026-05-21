package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/muhfaris/gherkio/internal/runner"
	"github.com/spf13/cobra"
)

var (
	envName string
	verbose bool
)

// runCmd represents the gherkio run command.
var runCmd = &cobra.Command{
	Use:   "run [test-file]",
	Short: "Execute a test scenario",
	Long: `Executes a Gherkio test YAML file and displays the results.

The test file path is resolved in the following order:
1. As provided (relative to current working directory)
2. Relative to .gherkio/tests/ directory

Example:
  gherkio run tests/login.yaml
  gherkio run login.yaml --env staging
  gherkio run login.yaml --verbose`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		testPath := args[0]
		return runTest(testPath, envName, verbose)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&envName, "env", "e", "local", "Environment to use (e.g. local, staging, production)")
	runCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show full request/response payloads")
}

func runTest(testPath, env string, verbose bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find project root (directory containing .gherkio)
	projectDir, err := findProjectRoot(cwd)
	if err != nil {
		return fmt.Errorf("not inside a Gherkio project: %w", err)
	}

	// Resolve test file path
	fullPath, err := resolveTestPath(cwd, projectDir, testPath)
	if err != nil {
		return fmt.Errorf("test file not found: %w", err)
	}

	cfg := runner.RunConfig{
		TestPath:   fullPath,
		EnvName:    env,
		ProjectDir: projectDir,
		Verbose:    verbose,
	}

	result, err := runner.Run(cfg)
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	runner.PrintResult(result, cfg.Verbose, cfg.MaskFields)

	if !result.Passed {
		os.Exit(1)
	}

	return nil
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
