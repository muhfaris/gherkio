package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/muhfaris/gherkio/gherkio-mcp/internal/server"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	repoRoot := root
	if envRoot := os.Getenv("GHERKIO_ROOT"); envRoot != "" {
		repoRoot = envRoot
	}

	baseResources := filepath.Join(repoRoot, "gherkio")
	srv := server.New(repoRoot, baseResources)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := srv.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
