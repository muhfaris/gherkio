package runner

import (
	"fmt"
	"sort"
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
	out := "| Step pattern | Description | Example |\n|---|---|---|\n"
	for _, it := range c.List() {
		out += fmt.Sprintf("| `%s` | %s | `%s` |\n", it.Pattern, safe(it.Desc), safe(it.Example))
	}
	return out
}

func safe(s string) string {
	if s == "" {
		return ""
	}
	// naive guard for pipes/backticks in table
	return s
}

// Expose read-only API for CLI
func GetStepCatalog() *StepCatalog { return stepCatalog }
