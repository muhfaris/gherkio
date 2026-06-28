package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/muhfaris/gherkio/internal/core/envcontext"
	"github.com/muhfaris/gherkio/internal/core/project"
	"github.com/spf13/cobra"
)

var (
	envContextJSON bool
)

func init() {
	rootCmd.AddCommand(envCmd)
	envCmd.AddCommand(envContextCmd)
	envContextCmd.Flags().BoolVar(&envContextJSON, "json", false, "Output as JSON")
}

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage Gherkio environments",
	Long:  `Commands for managing Gherkio test environments and credentials.`,
}

var envContextCmd = &cobra.Command{
	Use:   "context",
	Short: "Get environment context with auto-selection hints",
	Long: `Returns unified environment context including available environments,
accounts for each environment, and auto-selection hints.

Auto-selection hints are computed as follows:
- If only 1 environment exists, it is auto-selected
- If only 1 environment has exactly 1 account, both are auto-selected
- If an environment has only 1 account, that account is auto-selected

This command is designed for programmatic consumption (nvim plugin, MCP).`,
	RunE: runEnvContext,
}

func runEnvContext(cmd *cobra.Command, args []string) error {
	// Find project root
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	projectDir, err := project.FindRoot(cwd)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}

	ctx, err := envcontext.GetContext(projectDir)
	if err != nil {
		return fmt.Errorf("failed to get environment context: %w", err)
	}

	if envContextJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(ctx)
	}

	// Human-readable output
	printEnvContextHuman(ctx)
	return nil
}

func printEnvContextHuman(ctx *envcontext.EnvContext) {
	fmt.Printf("Project: %s\n\n", ctx.ProjectRoot)

	fmt.Printf("Environments (%d):\n", len(ctx.Environments))
	for _, env := range ctx.Environments {
		fmt.Printf("  • %s\n", env.Name)
		if env.BaseURL != "" {
			fmt.Printf("    BaseURL: %s\n", env.BaseURL)
		}
		if env.ServicesCount > 0 {
			fmt.Printf("    Services: %d\n", env.ServicesCount)
		}
		accs := ctx.Accounts[env.Name]
		if len(accs) > 0 {
			fmt.Printf("    Accounts: %v\n", accs)
		} else {
			fmt.Printf("    Accounts: none\n")
		}
	}

	fmt.Println()
	if ctx.AutoSelect != nil {
		fmt.Println("Auto-Select:")
		fmt.Printf("  Reason: %s\n", ctx.AutoSelect.Reason)
		if ctx.AutoSelect.EnvName != "" {
			fmt.Printf("  Env: %s\n", ctx.AutoSelect.EnvName)
		}
		if ctx.AutoSelect.AccountName != "" {
			fmt.Printf("  Account: %s\n", ctx.AutoSelect.AccountName)
		}
	}
}
