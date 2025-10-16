package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Version: "0.0.1-beta",
	Use:     "gherkio",
	Short:   "gherkio is a declarative API testing & journey runner",
	Long: `A fast and flexible CLI for API testing and journey running.
Complete documentation is available at https://github.com/muhfaris/gherkio`,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Whoops. There was an error while executing your CLI '%s'", err)
		os.Exit(1)
	}
}