package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Build information. Populated by ldflags at build time.
var (
	Version   = "v0.1.0-alpha.1"
	Commit    = "none"
	BuildDate = "unknown"
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:     "gherkio",
	Short:   "Gherkio is a testing and validation framework",
	Version: version(),
	Long: `Gherkio is a CLI tool for managing test execution,
validation schemas, and generating reports.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
}

func version() string {
	return fmt.Sprintf("%s (commit: %s, built: %s, %s)", Version, Commit, BuildDate, runtime.Version())
}
