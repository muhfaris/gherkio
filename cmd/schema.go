package cmd

import (
	"fmt"
	"os"

	"github.com/muhfaris/gherkio/internal/schema"
	"github.com/spf13/cobra"
)

// schemaCmd represents the schema command
var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Generate JSON Schema for Gherkio DSL",
	Long: `Generates a JSON Schema for Gherkio test YAML files.
This schema can be used in editors (like VSCode, Neovim) for autocomplete and validation.

Usage:
  gherkio schema > gherkio-schema.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		b, err := schema.GenerateJSONSchema()
		if err != nil {
			return fmt.Errorf("failed to generate JSON schema: %w", err)
		}

		// Print to stdout
		_, err = os.Stdout.Write(b)
		if err != nil {
			return err
		}

		fmt.Println() // Add a trailing newline
		return nil
	},
}

func init() {
	rootCmd.AddCommand(schemaCmd)
}
