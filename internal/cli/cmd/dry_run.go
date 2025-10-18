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

	"github.com/muhfaris/gherkio/internal/loader"
	"github.com/muhfaris/gherkio/internal/runner"
)

type lintProblem struct {
	File     string
	Line     int
	Text     string
	Keyword  string
	Reason   string
	Examples []string
}

type compiledStep struct {
	re   *regexp.Regexp
	info runner.StepInfo
}

func lintFeatures(paths []string, env loader.Env, cat loader.Catalog, flows map[string]loader.Flow) error {
	ensureStepCatalog(env, cat, flows)
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
			if !matchStep(patterns, step.Text) {
				problem := lintProblem{
					File:     path,
					Line:     int(step.Line),
					Text:     step.Text,
					Keyword:  step.Keyword,
					Reason:   "undefined step",
					Examples: suggestSteps(patterns, step.Text),
				}
				problems = append(problems, problem)
				continue
			}
			if msg, hints := semanticProblem(step.Text, cat, flows); msg != "" {
				problems = append(problems, lintProblem{
					File:     path,
					Line:     int(step.Line),
					Text:     step.Text,
					Keyword:  step.Keyword,
					Reason:   msg,
					Examples: hints,
				})
			}
		}
	}

	if len(problems) > 0 {
		for _, p := range problems {
			display := strings.TrimSpace(strings.Join([]string{p.Keyword, p.Text}, " "))
			reason := p.Reason
			if reason == "" {
				reason = "undefined step"
			}
			fmt.Fprintf(os.Stderr, "%s:%d %s: %s\n", p.File, p.Line, reason, display)
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

var (
	runFlowRe = regexp.MustCompile(`(?i)^i run flow ["']([^"']+)["']`)
	callAPIRe = regexp.MustCompile(`(?i)^i call api ["']([^"']+)["']`)
	setAuthRe = regexp.MustCompile(`(?i)^i set auth ["']([^"']+)["']`)
)

func semanticProblem(text string, cat loader.Catalog, flows map[string]loader.Flow) (string, []string) {
	if m := runFlowRe.FindStringSubmatch(text); len(m) == 2 {
		name := m[1]
		if containsTemplate(name) {
			return "", nil
		}
		if _, ok := flows[name]; !ok {
			suggestions := suggestKeys(name, mapKeys(flows))
			return fmt.Sprintf("unknown flow %q", name), suggestions
		}
		return "", nil
	}
	if m := callAPIRe.FindStringSubmatch(text); len(m) == 2 {
		key := m[1]
		if containsTemplate(key) {
			return "", nil
		}
		if _, ok := cat.Endpoints[key]; !ok {
			suggestions := suggestKeys(key, mapKeys(cat.Endpoints))
			return fmt.Sprintf("unknown API endpoint %q", key), suggestions
		}
		return "", nil
	}
	if m := setAuthRe.FindStringSubmatch(text); len(m) == 2 {
		name := m[1]
		if containsTemplate(name) {
			return "", nil
		}
		if _, ok := cat.Auth[name]; !ok {
			suggestions := suggestKeys(name, mapKeys(cat.Auth))
			return fmt.Sprintf("unknown auth profile %q", name), suggestions
		}
	}
	return "", nil
}

func containsTemplate(s string) bool {
	return strings.Contains(s, "{{") && strings.Contains(s, "}}")
}

func mapKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func suggestKeys(name string, keys []string) []string {
	if len(keys) == 0 {
		return nil
	}
	nameTokens := tokenize(name)
	if len(nameTokens) == 0 {
		nameTokens = []string{strings.ToLower(name)}
	}
	type cand struct {
		score int
		key   string
	}
	var cands []cand
	for _, key := range keys {
		score := keySimilarityScore(nameTokens, key)
		if score > 0 {
			cands = append(cands, cand{score: score, key: key})
		}
	}
	if len(cands) == 0 {
		return nil
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].score == cands[j].score {
			return cands[i].key < cands[j].key
		}
		return cands[i].score > cands[j].score
	})
	limit := 3
	if len(cands) < limit {
		limit = len(cands)
	}
	out := make([]string, 0, limit)
	for _, c := range cands[:limit] {
		out = append(out, c.key)
	}
	return out
}

func keySimilarityScore(tokens []string, key string) int {
	lower := strings.ToLower(key)
	score := 0
	for _, tok := range tokens {
		if len(tok) < 2 {
			continue
		}
		if strings.Contains(lower, tok) {
			score++
		}
	}
	return score
}

func ensureStepCatalog(env loader.Env, cat loader.Catalog, flows map[string]loader.Flow) {
	if len(runner.GetStepCatalog().List()) > 0 {
		return
	}
	init := runner.InitializeScenario(env, cat, flows)
	if init != nil {
		init(nil)
	}
}
