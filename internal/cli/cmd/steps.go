package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/muhfaris/gherkio/internal/loader"
	"github.com/muhfaris/gherkio/internal/runner"
	"github.com/spf13/cobra"
)

var stepsCmd = &cobra.Command{
	Use:   "steps",
	Short: "Run Gherkin steps",
	Long:  `Displays all available Gherkin steps that can be used in feature files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSteps(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(stepsCmd)
	stepsCmd.Flags().String("format", "text", "text|md")
	stepsCmd.Flags().String("out", "", "output file (optional)")
	stepsCmd.Flags().StringArray("match", []string{}, "filter steps containing substring (repeatable, AND logic)")
}

func runSteps(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	out, _ := cmd.Flags().GetString("out")
	matches, _ := cmd.Flags().GetStringArray("match")

	initScenario := runner.InitializeScenario(loader.Env{}, loader.Catalog{}, map[string]loader.Flow{})
	initScenario(nil) // bind steps into catalog without a Godog context
	cat := runner.GetStepCatalog()
	entries := cat.List()
	total := len(entries)
	if len(matches) > 0 {
		entries = filterEntries(entries, newMatchFilters(matches))
	}
	var data string
	if format == "md" {
		if len(entries) == 0 {
			data = renderNoMatches(matches)
		} else {
			data = cat.MarkdownFrom(entries)
		}
	} else {
		useColor := out == "" && isTerminal(os.Stdout)
		data = renderTextCatalog(entries, total, matches, useColor)
	}

	if out == "" {
		fmt.Print(data)
		return nil
	}
	return os.WriteFile(out, []byte(data), 0o644)
}

func renderTextCatalog(entries []runner.StepInfo, total int, filters []string, color bool) string {
	if len(entries) == 0 {
		return renderNoMatches(filters)
	}
	palette := struct {
		header  string
		pattern string
		example string
		reset   string
	}{
		header:  "\033[1;36m",
		pattern: "\033[1;33m",
		example: "\033[2m",
		reset:   "\033[0m",
	}
	if !color {
		palette.header, palette.pattern, palette.example, palette.reset = "", "", "", ""
	}

	groups := map[string][]runner.StepInfo{}
	order := []string{}
	for _, it := range entries {
		cat := classifyPattern(it.Pattern)
		if _, ok := groups[cat]; !ok {
			order = append(order, cat)
		}
		groups[cat] = append(groups[cat], it)
	}
	categoryRank := map[string]int{
		"Environment":       10,
		"API Calls":         20,
		"Request Setup":     30,
		"Authentication":    40,
		"Flows":             50,
		"JSON Assertions":   60,
		"HTTP Response":     70,
		"HTTP Headers":      80,
		"Store & Variables": 90,
		"Debugging":         100,
		"Utilities":         110,
		"Misc":              120,
	}
	sort.Slice(order, func(i, j int) bool {
		left := order[i]
		right := order[j]
		ri, ok := categoryRank[left]
		if !ok {
			ri = 1000
		}
		rj, ok := categoryRank[right]
		if !ok {
			rj = 1000
		}
		if ri != rj {
			return ri < rj
		}
		return left < right
	})

	var b strings.Builder
	if len(filters) > 0 {
		fmt.Fprintf(&b, "Available steps (%d of %d) matching: %s\n\n", len(entries), total, strings.Join(filters, ", "))
	} else {
		fmt.Fprintf(&b, "Available steps (%d)\n\n", total)
	}
	for i, name := range order {
		items := groups[name]
		if len(items) == 0 {
			continue
		}
		if i > 0 {
			b.WriteByte('\n')
		}
		title := name
		if title == "" {
			title = "Misc"
		}
		fmt.Fprintf(&b, "%s%s%s\n", palette.header, title, palette.reset)
		b.WriteString(strings.Repeat("-", len(title)))
		b.WriteByte('\n')
		tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  PATTERN\tDESCRIPTION\tEXAMPLE")
		fmt.Fprintln(tw, "  -------\t-----------\t-------")
		for _, it := range items {
			desc := it.Desc
			if desc == "" {
				desc = "-"
			}
			ex := it.Example
			if ex == "" {
				ex = "-"
			}
			pattern := fmt.Sprintf("  %s%s%s", palette.pattern, it.Pattern, palette.reset)
			example := fmt.Sprintf("%s%s%s", palette.example, ex, palette.reset)
			fmt.Fprintf(tw, "%s\t%s\t%s\n", pattern, desc, example)
		}
		_ = tw.Flush()
	}
	return b.String()
}

func filterEntries(entries []runner.StepInfo, filters *matchFilters) []runner.StepInfo {
	if filters == nil || filters.Len() == 0 {
		return entries
	}
	out := make([]runner.StepInfo, 0, len(entries))
	for _, it := range entries {
		haystack := strings.ToLower(strings.Join([]string{it.Pattern, it.Desc, it.Example}, " "))
		match := true
		for _, needle := range filters.lower {
			if !strings.Contains(haystack, needle) {
				match = false
				break
			}
		}
		if match {
			out = append(out, it)
		}
	}
	return out
}

func classifyPattern(p string) string {
	s := strings.TrimSpace(p)
	s = strings.TrimLeft(s, "^")
	s = strings.TrimSuffix(s, "$")
	ls := strings.ToLower(s)
	switch {
	case strings.HasPrefix(ls, "json "):
		return "JSON Assertions"
	case strings.HasPrefix(ls, "response "):
		return "HTTP Response"
	case strings.HasPrefix(ls, "header "):
		return "HTTP Headers"
	case strings.HasPrefix(ls, "i call api"):
		return "API Calls"
	case strings.HasPrefix(ls, "i run flow") || strings.HasPrefix(ls, "flow"):
		return "Flows"
	case strings.HasPrefix(ls, "i set path") || strings.HasPrefix(ls, "i set query") || strings.HasPrefix(ls, "i set headers"):
		return "Request Setup"
	case strings.HasPrefix(ls, "i set auth"):
		return "Authentication"
	case strings.HasPrefix(ls, "save response") || strings.HasPrefix(ls, "print response") || strings.HasPrefix(ls, "show variable"):
		return "Debugging"
	case strings.HasPrefix(ls, "save ") || strings.HasPrefix(ls, "set "):
		return "Store & Variables"
	case strings.HasPrefix(ls, "i wait"):
		return "Utilities"
	case strings.HasPrefix(ls, "i load"):
		return "Environment"
	default:
		return "Misc"
	}
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func renderNoMatches(filters []string) string {
	if len(filters) == 0 {
		return "No steps registered\n"
	}
	return fmt.Sprintf("No steps match filters: %s\n", strings.Join(filters, ", "))
}

type matchFilters struct {
	raw   []string
	lower []string
}

func newMatchFilters(raw []string) *matchFilters {
	lower := make([]string, len(raw))
	for i, v := range raw {
		lower[i] = strings.ToLower(v)
	}
	return &matchFilters{raw: raw, lower: lower}
}

func (m *matchFilters) Len() int {
	if m == nil {
		return 0
	}
	return len(m.raw)
}