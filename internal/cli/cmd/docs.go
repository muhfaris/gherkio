package cmd

import "github.com/spf13/cobra"

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Commands for documentation generation",
	Long:  `A collection of commands to generate documentation and snippets.`,
}

func init() {
	rootCmd.AddCommand(docsCmd)
}
