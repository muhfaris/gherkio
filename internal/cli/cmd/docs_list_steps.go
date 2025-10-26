package cmd

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var listStepsCmd = &cobra.Command{
	Use:   "list-steps",
	Short: "List all available Gherkin steps",
	Long:  `Scans the codebase to find and list all available Gherkin step definitions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := "internal/runner/steps_godog.go"
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", filePath, err)
		}

		// Regex to find all step definitions from the bind() helper function
		re := regexp.MustCompile("bind\\(sc,\\s*`([^`]+)`")
		matches := re.FindAllStringSubmatch(string(content), -1)

		if len(matches) == 0 {
			fmt.Println("No step definitions found.")
			return nil
		}

		var steps []string
		for _, match := range matches {
			if len(match) > 1 {
				steps = append(steps, match[1])
			}
		}

		sort.Strings(steps)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "Available Gherkin Steps:")
		fmt.Fprintln(w, "------------------------")
		for _, step := range steps {
			fmt.Fprintf(w, "- %s\n", step)
		}
		return w.Flush()
	},
}

func init() {
	docsCmd.AddCommand(listStepsCmd)
}
