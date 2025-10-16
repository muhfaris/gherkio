package cmd

import (
	"github.com/spf13/cobra"
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "gherkio documentation",
	Long:  `gherkio documentation.`,
	Example: `  # Show documentation for function templates
  gherkio docs fn

  # Show documentation for Gherkin steps
  gherkio docs steps`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(docsCmd)
}