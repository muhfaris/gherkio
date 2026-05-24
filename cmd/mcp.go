package cmd

import (
	"fmt"
	"os"

	"github.com/muhfaris/gherkio/internal/core/project"
	"github.com/muhfaris/gherkio/internal/mcp"
	"github.com/spf13/cobra"
)

// mcpCmd represents the Gherkio mcp command.
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start Gherkio Model Context Protocol (MCP) server over stdio",
	Long: `Starts the Gherkio Model Context Protocol (MCP) server over standard I/O (stdio).
This allows Gherkio capabilities (globbing tests, reading scenarios, running tests, resolving environments)
to be programmatically consumed by LLMs and agentic AI clients (e.g. Claude Desktop, VS Code extensions).

Stdout is strictly reserved for compliant JSON-RPC 2.0 frames. Diagnostic logs are output to Stderr.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to determine current directory: %w", err)
		}

		// Discover project root, default to empty (graceful outside project state)
		projectDir, _ := project.FindRoot(cwd)

		srv := mcp.NewServer(projectDir)
		fmt.Fprintf(os.Stderr, "ℹ Starting Gherkio MCP Server (stdio transport)... resolved root: '%s'\n", projectDir)

		return srv.Start()
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
