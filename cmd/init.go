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

// defaultLocalEnvTemplate is the default environment file content.
const defaultLocalEnvTemplate = `baseUrl: https://dummyjson.com
services:
  auth:
    baseUrl: http://localhost:3001
`

// defaultExampleTestTemplate is the default example test file content.
const defaultExampleTestTemplate = `scenario: login example

steps:
  - request:
      method: POST
      url: /auth/login

      body:
        username: emilys
        password: emilyspass

    expect:
      status: 200
      body.accessToken: exists

    save:
      token: body.accessToken
`

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

	// Create default environment file
	envPath := filepath.Join(baseDir, "environments", "local.yaml")
	if err := os.WriteFile(envPath, []byte(defaultLocalEnvTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write environment file: %w", err)
	}
	fmt.Printf("  📄  %s\n", envPath)

	// Create example test file
	exampleDir := filepath.Join(baseDir, "tests", "example")
	if err := os.MkdirAll(exampleDir, dirPerm); err != nil {
		return fmt.Errorf("failed to create example tests directory: %w", err)
	}
	fmt.Printf("  📁  %s\n", exampleDir)

	testPath := filepath.Join(exampleDir, "login.yaml")
	if err := os.WriteFile(testPath, []byte(defaultExampleTestTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write example test file: %w", err)
	}
	fmt.Printf("  📄  %s\n", testPath)

	fmt.Println("\n✨ Gherkio project initialized successfully!")
	fmt.Println("")
	fmt.Println("  Quick start:")
	fmt.Println("    gherkio run example/login.yaml")
	fmt.Println("    gherkio run example/login.yaml --verbose")
	return nil
}
