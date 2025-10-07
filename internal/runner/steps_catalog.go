package runner

import (
	"fmt"
	"html"
	"sort"
	"strings"
)

type StepInfo struct {
	Pattern string
	Desc    string
	Example string
}

type StepCatalog struct {
	items []StepInfo
}

var stepCatalog = &StepCatalog{}

func (c *StepCatalog) Add(pat, desc, ex string) {
	c.items = append(c.items, StepInfo{Pattern: pat, Desc: desc, Example: ex})
}

func (c *StepCatalog) List() []StepInfo {
	out := make([]StepInfo, len(c.items))
	copy(out, c.items)
	sort.Slice(out, func(i, j int) bool { return out[i].Pattern < out[j].Pattern })
	return out
}

// Render as Markdown table.
func (c *StepCatalog) Markdown() string {
	return markdownTable(c.List())
}

// MarkdownFrom renders an arbitrary subset of step infos as Markdown.
func (c *StepCatalog) MarkdownFrom(entries []StepInfo) string {
	rows := make([]StepInfo, len(entries))
	copy(rows, entries)
	sort.Slice(rows, func(i, j int) bool { return rows[i].Pattern < rows[j].Pattern })
	return markdownTable(rows)
}

func markdownTable(entries []StepInfo) string {
	var b strings.Builder
	b.WriteString("| Step pattern | Description | Example |\n|---|---|---|\n")
	for _, it := range entries {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", markdownCode(it.Pattern), markdownText(it.Desc), markdownExample(it.Example))
	}
	return b.String()
}

func markdownText(s string) string {
	if s == "" {
		return ""
	}
	return escapePipes(s)
}

func markdownCode(s string) string {
	if s == "" {
		return ""
	}
	out := escapePipes(s)
	out = strings.ReplaceAll(out, "`", "\\`")
	return "`" + out + "`"
}

func markdownExample(s string) string {
	if s == "" {
		return ""
	}
	if strings.ContainsAny(s, "\n\r") {
		return "<pre><code>" + html.EscapeString(s) + "</code></pre>"
	}
	return markdownCode(s)
}

func escapePipes(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}

// Expose read-only API for CLI
func GetStepCatalog() *StepCatalog { return stepCatalog }
