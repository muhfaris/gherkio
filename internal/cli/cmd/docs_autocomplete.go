package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var (
	outputVim    string
	outputVscode string
)

var docsAutocompleteCmd = &cobra.Command{
	Use:   "generate-autocomplete",
	Short: "Generate autocomplete files for editors",
	Long:  `Generate autocomplete files for various editors based on Gherkin steps.`,
	Example: `  # Generate for Vim and VS Code
  gherkio docs generate-autocomplete --output-vim completions.vim --output-vscode gherkin.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if outputVim == "" && outputVscode == "" {
			return fmt.Errorf("at least one output format must be specified")
		}

		steps, err := extractGherkinSteps("internal/runner/steps_godog.go")
		if err != nil {
			return fmt.Errorf("failed to extract gherkin steps: %w", err)
		}

		if outputVim != "" {
			content := generateVimAutocomplete(steps)
			err := os.WriteFile(outputVim, []byte(content), 0644)
			if err != nil {
				return fmt.Errorf("failed to write vim autocomplete file: %w", err)
			}
			fmt.Printf("Successfully generated Vim autocomplete file at: %s\n", outputVim)
		}

		if outputVscode != "" {
			content, err := generateVscodeSnippets(steps)
			if err != nil {
				return fmt.Errorf("failed to generate vscode snippets: %w", err)
			}
			err = os.WriteFile(outputVscode, content, 0644)
			if err != nil {
				return fmt.Errorf("failed to write vscode snippets file: %w", err)
			}
			fmt.Printf("Successfully generated VS Code snippets file at: %s\n", outputVscode)
		}

		return nil
	},
}

func extractGherkinSteps(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var steps []string
	scanner := bufio.NewScanner(file)
	re := regexp.MustCompile(`bind\s*\(\s*sc\s*,\s*\x60([^\x60]+)\x60`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			steps = append(steps, matches[1])
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return steps, nil
}

func generateVimAutocomplete(steps []string) string {
	var builder strings.Builder
	for _, step := range steps {
		cleanStep := cleanGherkinRegex(step)
		builder.WriteString(cleanStep)
		builder.WriteString("\n")
	}
	return builder.String()
}

func generateVscodeSnippets(steps []string) ([]byte, error) {
	snippets := make(map[string]interface{})
	for i, step := range steps {
		cleanStep := cleanGherkinRegex(step)
		snippet := map[string]interface{}{
			"prefix":      cleanStep,
			"body":        convertToVsCodeSnippet(step),
			"description": cleanStep,
		}
		snippets[fmt.Sprintf("Gherkin Step %d", i+1)] = snippet
	}

	return json.MarshalIndent(snippets, "", "  ")
}

func cleanGherkinRegex(pattern string) string {
	str := strings.TrimPrefix(pattern, "^")
	str = strings.TrimSuffix(str, "$")
	str = regexp.MustCompile(`\(\?:(?:Given|Then|When|And|But)\s+\)?`).ReplaceAllString(str, "")
	str = strings.TrimSpace(str)

	// More robust replacements
	str = regexp.MustCompile(`\[\\"']\(\[\^\\"']\+\)\[\\"']`).ReplaceAllString(str, "'<string>'")
	str = regexp.MustCompile(`\["']\(\[\^"']\+\)\["']`).ReplaceAllString(str, `"<string>"`)
	str = regexp.MustCompile(`\(\[\^\\"']\+\)`).ReplaceAllString(str, "<string>")
	str = regexp.MustCompile(`\(\[\^"']\+\)`).ReplaceAllString(str, "<string>")
	str = regexp.MustCompile(`\(\\d\+\)`).ReplaceAllString(str, "<number>")
	str = regexp.MustCompile(`\(\\d\{3\}\)-\(\(\\d\{3\}\)`).ReplaceAllString(str, "<range>")
	str = regexp.MustCompile(`\(==\|!=\|>=\|>\|<=\|<\)`).ReplaceAllString(str, "<operator>")
	str = regexp.MustCompile(`\(true\|false\|null\)`).ReplaceAllString(str, "<literal>")
	str = regexp.MustCompile(`\(\.\+\?\)`).ReplaceAllString(str, "<value>")
	str = regexp.MustCompile(`\(\.\+\)`).ReplaceAllString(str, "<value>")

	str = strings.ReplaceAll(str, `\"`, `"`)
	str = strings.ReplaceAll(str, `\'`, `'`)
	str = strings.ReplaceAll(str, `\s+`, " ")
	str = regexp.MustCompile(`\s{2,}`).ReplaceAllString(str, " ")

	return str
}

func convertToVsCodeSnippet(pattern string) string {
	str := strings.TrimPrefix(pattern, "^")
	str = strings.TrimSuffix(str, "$")
    str = regexp.MustCompile(`\(\?:(?:Given|Then|When|And|But)\s+\)?`).ReplaceAllString(str, "")
	str = strings.TrimSpace(str)

	placeholderIndex := 1
	// This regex now specifically targets capturing groups, avoiding non-capturing ones.
	re := regexp.MustCompile(`\((?!\?[:!]).*?\)`)

	body := re.ReplaceAllStringFunc(str, func(group string) string {
		// Extract the content of the group for a better placeholder name
		content := strings.Trim(group, "()")
		// A simple heuristic for placeholder text
		placeholderText := "value"
		if strings.Contains(content, "d+") {
			placeholderText = "number"
		} else if strings.Contains(content, `[^\"']+`) {
			placeholderText = "string"
		}

		placeholder := fmt.Sprintf(`${%d:%s}`, placeholderIndex, placeholderText)
		placeholderIndex++
		return placeholder
	})

	body = strings.ReplaceAll(body, `\"`, `"`)
	body = strings.ReplaceAll(body, `\'`, `'`)
	body = strings.ReplaceAll(body, `\s+`, " ")

	return body
}

func init() {
	docsCmd.AddCommand(docsAutocompleteCmd)
	docsAutocompleteCmd.Flags().StringVar(&outputVim, "output-vim", "", "Output file for Vim autocomplete")
	docsAutocompleteCmd.Flags().StringVar(&outputVscode, "output-vscode", "", "Output file for VS Code snippets")
}
