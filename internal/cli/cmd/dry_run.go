package cmd

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode"

	gherkin "github.com/cucumber/gherkin/go/v26"
	messages "github.com/cucumber/messages/go/v21"

	"github.com/muhfaris/gherkio/internal/runner"
)

type lintProblem struct {
	File     string
	Line     int
	Text     string
	Keyword  string
	Examples []string
}

type compiledStep struct {
	re   *regexp.Regexp
	info runner.StepInfo
}

func lintFeatures(paths []string) error {
	patterns, err := compileStepPatterns()
	if err != nil {
		return err
	}
	var (
		problems   []lintProblem
		totalSteps int
	)

	for _, path := range paths {
		doc, err := parseFeature(path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		steps := collectSteps(doc)
		for _, step := range steps {
			if step.Text == "" {
				continue
			}
			totalSteps++
			if matchStep(patterns, step.Text) {
				continue
			}
			problem := lintProblem{
				File:    path,
				Line:    int(step.Line),
				Text:    step.Text,
				Keyword: step.Keyword,
			}
			problem.Examples = suggestSteps(patterns, step.Text)
			problems = append(problems, problem)
		}
	}

	if len(problems) > 0 {
		for _, p := range problems {
			display := strings.TrimSpace(strings.Join([]string{p.Keyword, p.Text}, " "))
			fmt.Fprintf(os.Stderr, "%s:%d undefined step: %s\n", p.File, p.Line, display)
			if len(p.Examples) > 0 {
				fmt.Fprintln(os.Stderr, "  Did you mean:")
				for _, ex := range p.Examples {
					fmt.Fprintf(os.Stderr, "    %s\n", ex)
				}
			}
		}
		return fmt.Errorf("dry run failed: %d undefined step(s)", len(problems))
	}

	fmt.Printf("Dry run OK: %d steps validated\n", totalSteps)
	return nil
}

func compileStepPatterns() ([]compiledStep, error) {
	catalog := runner.GetStepCatalog().List()
	compiled := make([]compiledStep, 0, len(catalog))
	for _, info := range catalog {
		re, err := regexp.Compile(info.Pattern)
		if err != nil {
			return nil, fmt.Errorf("compile step pattern %q: %w", info.Pattern, err)
		}
		compiled = append(compiled, compiledStep{re: re, info: info})
	}
	return compiled, nil
}

func parseFeature(path string) (*messages.GherkinDocument, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return gherkin.ParseGherkinDocument(f, (&messages.Incrementing{}).NewId)
}

type lintStep struct {
	Text    string
	Keyword string
	Line    int64
}

func collectSteps(doc *messages.GherkinDocument) []lintStep {
	if doc == nil || doc.Feature == nil {
		return nil
	}
	var steps []lintStep
	walkChildren(doc.Feature.Children, &steps)
	return steps
}

func walkChildren(children []*messages.FeatureChild, steps *[]lintStep) {
	for _, child := range children {
		if child == nil {
			continue
		}
		if bg := child.Background; bg != nil {
			appendSteps(bg.Steps, steps)
		}
		if sc := child.Scenario; sc != nil {
			appendSteps(sc.Steps, steps)
		}
		if rule := child.Rule; rule != nil {
			walkRule(rule, steps)
		}
	}
}

func walkRule(rule *messages.Rule, steps *[]lintStep) {
	if rule == nil {
		return
	}
	for _, child := range rule.Children {
		if child == nil {
			continue
		}
		if bg := child.Background; bg != nil {
			appendSteps(bg.Steps, steps)
		}
		if sc := child.Scenario; sc != nil {
			appendSteps(sc.Steps, steps)
		}
	}
}

func appendSteps(src []*messages.Step, dst *[]lintStep) {
	for _, st := range src {
		if st == nil {
			continue
		}
		text := strings.TrimSpace(st.Text)
		keyword := strings.TrimSpace(st.Keyword)
		line := int64(0)
		if st.Location != nil {
			line = st.Location.Line
		}
		*dst = append(*dst, lintStep{Text: text, Keyword: keyword, Line: line})
	}
}

func matchStep(patterns []compiledStep, text string) bool {
	for _, pat := range patterns {
		if pat.re.MatchString(text) {
			return true
		}
	}
	return false
}

func suggestSteps(patterns []compiledStep, text string) []string {
	const maxSuggestions = 3
	type candidate struct {
		score int
		info  runner.StepInfo
	}
	tokens := tokenize(text)
	if len(tokens) == 0 {
		tokens = []string{text}
	}
	var cands []candidate
	for _, pat := range patterns {
		score := similarityScore(tokens, pat.info)
		if score == 0 {
			continue
		}
		cands = append(cands, candidate{score: score, info: pat.info})
	}
	if len(cands) == 0 {
		return nil
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].score == cands[j].score {
			return cands[i].info.Pattern < cands[j].info.Pattern
		}
		return cands[i].score > cands[j].score
	})
	limit := maxSuggestions
	if len(cands) < limit {
		limit = len(cands)
	}
	out := make([]string, 0, limit)
	for _, cand := range cands[:limit] {
		if cand.info.Example != "" {
			out = append(out, cand.info.Example)
		} else {
			out = append(out, cand.info.Pattern)
		}
	}
	return out
}

func tokenize(text string) []string {
	lower := strings.ToLower(text)
	return strings.FieldsFunc(lower, func(r rune) bool {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return false
		}
		return true
	})
}

func similarityScore(tokens []string, info runner.StepInfo) int {
	if len(tokens) == 0 {
		return 0
	}
	target := strings.ToLower(info.Pattern)
	score := 0
	for _, tok := range tokens {
		if len(tok) < 3 {
			continue
		}
		if strings.Contains(target, tok) {
			score++
		}
	}
	return score
}
