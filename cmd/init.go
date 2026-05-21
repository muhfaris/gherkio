package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const (
	dirPerm os.FileMode = 0755
)

// defaultConfigTemplate is the default config.yaml template content.
const defaultConfigTemplate = `# Gherkio Configuration
project:
  name: ""
  version: "1.0.0"

environments:
  default: local
  path: .gherkio/environments

tests:
  path: .gherkio/tests

reports:
  path: .gherkio/reports
  format: html
  archive: true

schemas:
  path: .gherkio/schemas

security:
  mask:
    # Sensitive field names to mask in output.
    # Override defaults by listing custom field names here.
    # Set to empty to disable masking.
    enabled: true
    fields: []
`

// initCmd represents the gherkio init command.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Gherkio project structure",
	Long: `Creates the .gherkio directory in the current working directory
with the default project structure including config, environments, tests,
reports, and schemas.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit()
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	baseDir := filepath.Join(cwd, ".gherkio")

	// Directories to create
	dirs := []string{
		baseDir,
		filepath.Join(baseDir, "environments"),
		filepath.Join(baseDir, "tests"),
		filepath.Join(baseDir, "reports"),
		filepath.Join(baseDir, "reports", "latest"),
		filepath.Join(baseDir, "reports", "archive"),
		filepath.Join(baseDir, "schemas"),
	}

	fmt.Println("Creating .gherkio project structure...")

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, dirPerm); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		fmt.Printf("  📁  %s\n", dir)
	}

	// Create default config.yaml
	configPath := filepath.Join(baseDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(defaultConfigTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write config.yaml: %w", err)
	}
	fmt.Printf("  📄  %s\n", configPath)

	fmt.Println("\n✨ Gherkio project initialized successfully!")
	return nil
}
