package cmd

import (
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "gherkio",
	Short: "Gherkio is a testing and validation framework",
	Long: `Gherkio is a CLI tool for managing test execution,
validation schemas, and generating reports.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Here you can define persistent flags for all commands
}
