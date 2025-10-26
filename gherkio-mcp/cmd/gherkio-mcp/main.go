package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/muhfaris/gherkio/gherkio-mcp/server"
)

// This command runs the gherkio-mcp server, which acts as a bridge to allow
// AI agents and other tools to programmatically interact with the gherkio
// API testing framework over the Model Context Protocol (MCP).
func main() {
	// --- 1. Determine the Project Root ---
	// The MCP server needs to know the root of the gherkio project repository
	// to locate the main 'gherkio' binary and other project files.
	// It defaults to the current working directory but can be overridden by the
	// GHERKIO_ROOT environment variable for flexibility.
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	repoRoot := root
	if envRoot := os.Getenv("GHERKIO_ROOT"); envRoot != "" {
		repoRoot = envRoot
	}
	fmt.Fprintf(os.Stderr, "INFO: Using repository root: %s\n", repoRoot)

	// --- 2. Define the Gherkio Resources Path ---
	// The server also needs the path to the directory where all gherkio-specific
	// files are stored (e.g., apis, features, envs). By convention, this is a
	// 'gherkio' subdirectory within the project root.
	resourcesDir := filepath.Join(repoRoot, "gherkio")
	fmt.Fprintf(os.Stderr, "INFO: Using resources directory: %s\n", resourcesDir)

	if _, err := os.Stat(resourcesDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "error: gherkio resources directory not found at %s\n", resourcesDir)
		fmt.Fprintf(os.Stderr, "Please run this from the root of a gherkio project or set GHERKIO_ROOT.\n")
		os.Exit(1)
	}

	// --- 3. Instantiate the Server ---
	// Create a new server instance by providing the repository root and the
	// path to the gherkio resources.
	srv := server.New(repoRoot, resourcesDir)
	fmt.Fprintf(os.Stderr, "INFO: MCP server instance created.\n")

	// --- 4. Set Up Graceful Shutdown ---
	// It's good practice to allow the server to shut down gracefully. This code
	// creates a context that will be canceled when the user presses Ctrl+C
	// (SIGINT) or when the process receives a termination signal (SIGTERM).
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// --- 5. Run the Server ---
	// Start the server. It will begin listening for JSON-RPC 2.0 requests on
	// standard input (stdin) and sending responses to standard output (stdout).
	// The Run method will block until the context is canceled or a fatal error
	// occurs.
	fmt.Fprintf(os.Stderr, "INFO: Starting MCP server. Waiting for requests on stdin...\n")
	if err := srv.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error: server exited with an error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "INFO: Server shut down gracefully.\n")
}
