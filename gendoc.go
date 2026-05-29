//go:build docgen
// +build docgen

package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/muhfaris/gherkio/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {
	outDir := "docs/book/src/cli"
	
	// Create a temporary directory to generate standard Cobra docs
	tmpDir, err := os.MkdirTemp("", "gherkio-docs")
	if err != nil {
		log.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	root := cmd.GetRootCmd()
	err = doc.GenMarkdownTree(root, tmpDir)
	if err != nil {
		log.Fatalf("failed to generate cobra markdown: %v", err)
	}

	// Read generated files and copy/rename to target output directory
	err = filepath.WalkDir(tmpDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		filename := d.Name()
		// Cobra generates: gherkio.md, gherkio_init.md, gherkio_run.md, etc.
		targetName := filename
		if filename != "gherkio.md" {
			targetName = strings.TrimPrefix(filename, "gherkio_")
		}

		targetPath := filepath.Join(outDir, targetName)

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Let's make some nice custom adjustments to links inside generated files
		// (e.g. gherkio_init.md -> init.md)
		adjusted := strings.ReplaceAll(string(content), "gherkio_", "")

		// Promote top-level H2 headers to H1 headers for correct HTML and SEO structure
		if strings.HasPrefix(adjusted, "## ") {
			adjusted = "# " + strings.TrimPrefix(adjusted, "## ")
		}

		// Clean up, format examples, and strip the raw spf13/cobra auto-generation footer lines to keep manual pages premium
		lines := strings.Split(adjusted, "\n")
		var cleanLines []string
		for i := 0; i < len(lines); i++ {
			line := lines[i]
			if strings.HasPrefix(line, "###### Auto generated") {
				continue
			}

			cleanLines = append(cleanLines, line)

			// Automatically discover CLI examples blocks and wrap them in beautiful markdown code blocks
			if strings.HasPrefix(line, "Examples:") || strings.HasPrefix(line, "Example:") {
				cleanLines = append(cleanLines, "```bash")
				for i+1 < len(lines) {
					nextLine := lines[i+1]
					trimmed := strings.TrimSpace(nextLine)

					// If next line is empty or starts with spaces, it belongs to the examples block
					if trimmed == "" {
						cleanLines = append(cleanLines, nextLine)
						i++
						continue
					}
					if strings.HasPrefix(nextLine, "  ") || strings.HasPrefix(nextLine, "\t") {
						cleanLines = append(cleanLines, nextLine)
						i++
						continue
					}
					break
				}
				cleanLines = append(cleanLines, "```")
			}
		}
		adjusted = strings.Join(cleanLines, "\n")

		err = os.WriteFile(targetPath, []byte(adjusted), 0644)
		if err != nil {
			return err
		}

		fmt.Printf("✓ Auto-generated %s -> %s\n", filename, targetPath)
		return nil
	})

	if err != nil {
		log.Fatalf("failed to process generated docs: %v", err)
	}

	fmt.Println("✓ CLI reference auto-generation complete!")
}
