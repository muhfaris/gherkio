package cmd

import (
	"fmt"
	"os"

	"github.com/muhfaris/gherkio/internal/schema"
	"github.com/spf13/cobra"
)

var (
	schemaType string
	listTypes  bool
)

// schemaCmd represents the schema command
var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Generate JSON Schema for Gherkio YAML files",
	Long: `Generates a JSON Schema for Gherkio YAML files.
This schema can be used in editors (like VSCode, Neovim) for autocomplete and validation.

Usage:
  gherkio schema                              # Generate all schemas (default)
  gherkio schema --type test                 # Generate only test file schema
  gherkio schema --type config               # Generate only config schema
  gherkio schema --type environment         # Generate only environment schema
  gherkio schema --type credentials         # Generate only credentials schema
  gherkio schema --type schema-definition   # Generate only schema definition schema
  gherkio schema --list                     # List available schema types

Output:
  By default, outputs all schemas in a single JSON file keyed by type.
  Use --type to output a specific schema only.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if listTypes {
			return printSchemaTypes()
		}

		var b []byte
		var err error

		if schemaType == "" {
			// Default: generate all schemas
			b, err = schema.GenerateAllSchemas()
		} else {
			// Specific type requested
			b, err = schema.GenerateSchemaType(schema.SchemaType(schemaType))
		}

		if err != nil {
			return fmt.Errorf("failed to generate JSON schema: %w", err)
		}

		if b == nil {
			return fmt.Errorf("unknown schema type: %s. Use --list to see available types", schemaType)
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
	schemaCmd.Flags().StringVar(&schemaType, "type", "", "Schema type: test, config, environment, credentials, schema-definition")
	schemaCmd.Flags().BoolVar(&listTypes, "list", false, "List available schema types")
}

func printSchemaTypes() error {
	types := schema.AvailableSchemaTypes()
	fmt.Println("Available schema types:")
	fmt.Println()
	for _, t := range types {
		fmt.Printf("  %-20s %s\n", t.Type, t.Description)
		for _, pattern := range t.FilePatterns {
			fmt.Printf("    Pattern: %s\n", pattern)
		}
		fmt.Println()
	}
	return nil
}
