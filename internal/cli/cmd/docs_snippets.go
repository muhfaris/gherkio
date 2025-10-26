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
	output string
)

var docsSnippetsCmd = &cobra.Command{
	Use:   "generate-snippets",
	Short: "Generate snippets for editors",
	Long:  `Generate snippets for various editors based on Gherkin steps.`,
	Example: `  # Generate for VS Code
  gherkio docs generate-snippets --output vscode.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if output == "" {
			return fmt.Errorf("output format must be specified")
		}

		steps, err := extractGherkinSteps("internal/runner/steps_godog.go")
		if err != nil {
			return fmt.Errorf("failed to extract gherkin steps: %w", err)
		}

		content, err := generateSnippets(steps)
		if err != nil {
			return fmt.Errorf("failed to generate vscode snippets: %w", err)
		}
		err = os.WriteFile(output, content, 0644)
		if err != nil {
			return fmt.Errorf("failed to write vscode snippets file: %w", err)
		}
		fmt.Printf("Successfully generated VS Code snippets file at: %s\n", output)

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

func generateSnippets(steps []string) ([]byte, error) {
	snippets := make(map[string]interface{})
	for _, step := range steps {
		cleanStep := cleanGherkinRegex(step)
		snippet := map[string]interface{}{
			"prefix":      cleanStep,
			"body":        convertToVsCodeSnippet(step),
			"description": cleanStep,
		}
		snippets[cleanStep] = snippet
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
	// The negative lookahead `(?!)` is not supported in Go's regexp engine, causing a panic.
	// We replace it with a simpler regex that finds any parenthesized group, and then
	// manually check if it's a non-capturing group within the replacement function.
	re := regexp.MustCompile(`\([^)]+\)`)

	body := re.ReplaceAllStringFunc(str, func(group string) string {
		// Check if it's a non-capturing group (e.g., "(?:...)") and return it as is.
		if strings.HasPrefix(group, "(?:") {
			return group
		}

		// For capturing groups, convert them to VS Code snippet placeholders.
		content := strings.Trim(group, "()")
		placeholderText := "value"
		if strings.Contains(content, "d+") {
			placeholderText = "number"
		} else if strings.Contains(content, `[^\"']+`) || strings.Contains(content, `[^"']+`) {
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
	docsCmd.AddCommand(docsSnippetsCmd)
	docsSnippetsCmd.Flags().StringVar(&output, "output", "", "Output file for snippets")
}
