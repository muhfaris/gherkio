package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type commandResult struct {
	Status     string   `json:"status"`
	ExitCode   int      `json:"exitCode"`
	Command    []string `json:"command"`
	DurationMs int64    `json:"durationMs"`
	Stdout     string   `json:"stdout"`
	Stderr     string   `json:"stderr"`
	Error      string   `json:"error,omitempty"`
}

type callInput struct {
	Env          string            `json:"env" jsonschema:"Environment name"`
	API          string            `json:"api" jsonschema:"API catalog key (e.g. users.getById)"`
	Path         map[string]string `json:"path" jsonschema:"Path parameters"`
	Query        map[string]string `json:"query" jsonschema:"Query parameters"`
	Header       map[string]string `json:"header" jsonschema:"Headers"`
	Body         string            `json:"body" jsonschema:"Inline body or @fixture path"`
	ExpectStatus *int              `json:"expectStatus" jsonschema:"Expected HTTP status code"`
	Reports      []string          `json:"reports" jsonschema:"Additional reporters"`
}

type runInput struct {
	Env      string   `json:"env" jsonschema:"Environment name"`
	Tags     string   `json:"tags" jsonschema:"Tag expression"`
	Reports  []string `json:"reports" jsonschema:"Report specs (kind[:path])"`
	Paths    []string `json:"paths" jsonschema:"Optional feature paths"`
	Parallel int      `json:"parallel" jsonschema:"Number of parallel workers"`
}

type featureWriteScenario struct {
	Name  string   `json:"name" jsonschema:"Scenario name"`
	Steps []string `json:"steps" jsonschema:"Scenario steps"`
	Tags  []string `json:"tags" jsonschema:"Scenario tags"`
}

type featureWriteInput struct {
	Path        string                 `json:"path" jsonschema:"Relative path under gherkio/features"`
	Content     string                 `json:"content" jsonschema:"Raw feature file contents"`
	Title       string                 `json:"title" jsonschema:"Feature title (required if content empty)"`
	Description string                 `json:"description" jsonschema:"Optional feature description"`
	Tags        []string               `json:"tags" jsonschema:"Feature level tags"`
	Scenarios   []featureWriteScenario `json:"scenarios" jsonschema:"Generated scenarios when content omitted"`
	Overwrite   bool                   `json:"overwrite" jsonschema:"Allow overwriting existing file"`
}

type featureWriteResult struct {
	Status   string `json:"status"`
	URI      string `json:"uri"`
	Path     string `json:"path"`
	Absolute string `json:"absolute"`
	Created  bool   `json:"created"`
}

type scenarioSuggestInput struct {
	FeatureTitle       string            `json:"featureTitle" jsonschema:"Optional feature title"`
	FeatureDescription string            `json:"featureDescription" jsonschema:"Optional feature description"`
	FeatureTags        []string          `json:"featureTags" jsonschema:"Tags applied to the feature"`
	ScenarioName       string            `json:"scenarioName" jsonschema:"Explicit scenario name"`
	Purpose            string            `json:"purpose" jsonschema:"Business goal of the scenario"`
	Env                string            `json:"env" jsonschema:"Environment under test"`
	API                string            `json:"api" jsonschema:"API catalog key"`
	Method             string            `json:"method" jsonschema:"HTTP method"`
	Endpoint           string            `json:"endpoint" jsonschema:"Concrete endpoint or route template"`
	Preconditions      []string          `json:"preconditions" jsonschema:"Preconditions to satisfy"`
	PathParams         map[string]string `json:"pathParams" jsonschema:"Concrete path parameter values"`
	QueryParams        map[string]string `json:"queryParams" jsonschema:"Query parameter values"`
	Headers            map[string]string `json:"headers" jsonschema:"HTTP headers"`
	RequestBody        string            `json:"requestBody" jsonschema:"Request body or fixture reference"`
	ExpectStatus       int               `json:"expectStatus" jsonschema:"Expected HTTP status code"`
	ResponseChecks     []string          `json:"responseChecks" jsonschema:"Additional response assertions"`
	ScenarioTags       []string          `json:"scenarioTags" jsonschema:"Scenario level tags"`
}

type scenarioSuggestResult struct {
	FeatureTitle       string                 `json:"featureTitle"`
	FeatureDescription string                 `json:"featureDescription,omitempty"`
	FeatureTags        []string               `json:"featureTags,omitempty"`
	Scenario           featureWriteScenario   `json:"scenario"`
	SuggestedPath      string                 `json:"suggestedPath"`
	Gherkin            string                 `json:"gherkin"`
	Summary            string                 `json:"summary"`
	FeatureTemplate    featureWriteInput      `json:"featureTemplate"`
	Command            []string               `json:"command"`
	CallArguments      map[string]interface{} `json:"callArguments"`
}

func registerTools(host *Server, mcpServer *mcp.Server) error {
	registerCallTool(host, mcpServer)
	registerRunTool(host, mcpServer)
	registerFeatureWriteTool(host, mcpServer)
	registerScenarioSuggestTool(host, mcpServer)
	return nil
}

func registerCallTool(host *Server, mcpServer *mcp.Server) {
	mcp.AddTool[callInput, commandResult](mcpServer, &mcp.Tool{
		Name:        "gherkio.call",
		Description: "Execute a single API endpoint using the gherkio CLI",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input callInput) (*mcp.CallToolResult, commandResult, error) {
		if strings.TrimSpace(input.Env) == "" || strings.TrimSpace(input.API) == "" {
			return nil, commandResult{}, fmt.Errorf("env and api are required")
		}

		cliArgs := []string{"call", "--env", input.Env, "--api", input.API}
		for _, pair := range sortedPairs(input.Path) {
			cliArgs = append(cliArgs, "--path", pair)
		}
		for _, pair := range sortedPairs(input.Query) {
			cliArgs = append(cliArgs, "--query", pair)
		}
		for _, pair := range sortedPairs(input.Header) {
			cliArgs = append(cliArgs, "--header", pair)
		}
		if strings.TrimSpace(input.Body) != "" {
			cliArgs = append(cliArgs, "--body", input.Body)
		}
		if input.ExpectStatus != nil {
			cliArgs = append(cliArgs, "--expect-status", fmt.Sprintf("%d", *input.ExpectStatus))
		}
		for _, rep := range input.Reports {
			if strings.TrimSpace(rep) == "" {
				continue
			}
			cliArgs = append(cliArgs, "--report", rep)
		}

		output, err := runGherkioCommand(ctx, host.repoRoot, cliArgs)
		if err != nil {
			return nil, commandResult{}, err
		}

		summary := fmt.Sprintf("gherkio call %s exited %d (%s)", input.API, output.ExitCode, output.Status)
		result := &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: summary}},
		}
		return result, output, nil
	})
}

func registerRunTool(host *Server, mcpServer *mcp.Server) {
	mcp.AddTool[runInput, commandResult](mcpServer, &mcp.Tool{
		Name:        "gherkio.run",
		Description: "Execute feature journeys via the gherkio CLI",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input runInput) (*mcp.CallToolResult, commandResult, error) {
		cliArgs := []string{"run"}
		if strings.TrimSpace(input.Env) != "" {
			cliArgs = append(cliArgs, "--env", input.Env)
		}
		if strings.TrimSpace(input.Tags) != "" {
			cliArgs = append(cliArgs, "--tags", input.Tags)
		}
		if input.Parallel > 0 {
			cliArgs = append(cliArgs, "--parallel", fmt.Sprintf("%d", input.Parallel))
		}
		for _, rep := range input.Reports {
			if strings.TrimSpace(rep) == "" {
				continue
			}
			cliArgs = append(cliArgs, "--report", rep)
		}
		cliArgs = append(cliArgs, input.Paths...)

		output, err := runGherkioCommand(ctx, host.repoRoot, cliArgs)
		if err != nil {
			return nil, commandResult{}, err
		}

		summary := fmt.Sprintf("gherkio run completed with exit %d (%s)", output.ExitCode, output.Status)
		result := &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: summary}},
		}
		return result, output, nil
	})
}

func registerFeatureWriteTool(host *Server, mcpServer *mcp.Server) {
	mcp.AddTool[featureWriteInput, featureWriteResult](mcpServer, &mcp.Tool{
		Name:        "gherkio.feature.write",
		Description: "Create or overwrite a feature file inside gherkio/features",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input featureWriteInput) (*mcp.CallToolResult, featureWriteResult, error) {
		if strings.TrimSpace(input.Path) == "" {
			return nil, featureWriteResult{}, fmt.Errorf("path is required")
		}

		featuresBase := filepath.Join(host.resourcesDir, "features")
		if err := os.MkdirAll(featuresBase, 0o755); err != nil {
			return nil, featureWriteResult{}, fmt.Errorf("ensure features directory: %w", err)
		}

		cleaned := filepath.Clean(filepath.FromSlash(input.Path))
		target := filepath.Join(featuresBase, cleaned)
		rel, err := filepath.Rel(featuresBase, target)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil, featureWriteResult{}, fmt.Errorf("path escapes features directory: %s", input.Path)
		}

		existed := false
		if _, err := os.Stat(target); err == nil {
			existed = true
		} else if !os.IsNotExist(err) {
			return nil, featureWriteResult{}, fmt.Errorf("stat target: %w", err)
		}

		if existed && !input.Overwrite {
			return nil, featureWriteResult{}, fmt.Errorf("file exists and overwrite is false")
		}

		content := input.Content
		if strings.TrimSpace(content) == "" {
			if strings.TrimSpace(input.Title) == "" {
				return nil, featureWriteResult{}, fmt.Errorf("title is required when content is empty")
			}
			if len(input.Scenarios) == 0 {
				return nil, featureWriteResult{}, fmt.Errorf("at least one scenario is required when content is empty")
			}
			content = buildFeatureContent(input.Title, input.Description, input.Tags, input.Scenarios)
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, featureWriteResult{}, fmt.Errorf("create directories: %w", err)
		}
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			return nil, featureWriteResult{}, fmt.Errorf("write feature: %w", err)
		}

		abs, err := filepath.Abs(target)
		if err != nil {
			abs = target
		}

		relPath := filepath.ToSlash(filepath.Join("features", cleaned))
		uri := "gherkio://" + relPath
		desc := ResourceDescriptor{
			URI:         uri,
			Name:        relPath,
			Description: fmt.Sprintf("Gherkin feature %s", filepath.Base(cleaned)),
			MimeType:    mimeTypeFor(target),
		}
		if err := host.attachResource(mcpServer, desc); err != nil {
			return nil, featureWriteResult{}, err
		}

		summary := fmt.Sprintf("feature written to %s (created=%t)", relPath, !existed)
		result := &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: summary}},
		}
		output := featureWriteResult{
			Status:   "ok",
			URI:      uri,
			Path:     relPath,
			Absolute: abs,
			Created:  !existed,
		}
		return result, output, nil
	})
}

func registerScenarioSuggestTool(host *Server, mcpServer *mcp.Server) {
	mcp.AddTool[scenarioSuggestInput, scenarioSuggestResult](mcpServer, &mcp.Tool{
		Name:        "gherkio.scenario.suggest",
		Description: "Generate a structured Gherkin scenario skeleton for an API call",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input scenarioSuggestInput) (*mcp.CallToolResult, scenarioSuggestResult, error) {
		scenarioName := deriveScenarioName(input)
		scenarioSteps := buildScenarioSteps(input)
		scenario := featureWriteScenario{
			Name:  scenarioName,
			Steps: scenarioSteps,
			Tags:  dedupStrings(input.ScenarioTags),
		}

		featureTitle := deriveFeatureTitle(input)
		featureDescription := strings.TrimSpace(input.FeatureDescription)
		featureTags := dedupStrings(input.FeatureTags)
		gherkin := buildFeatureContent(featureTitle, featureDescription, featureTags, []featureWriteScenario{scenario})
		suggestedPath := suggestedFeaturePath(input, scenarioName)
		summary := fmt.Sprintf("Scenario %q prepared for API %s", scenarioName, strings.TrimSpace(input.API))

		template := featureWriteInput{
			Path:        suggestedPath,
			Title:       featureTitle,
			Description: featureDescription,
			Tags:        featureTags,
			Scenarios:   []featureWriteScenario{scenario},
			Overwrite:   true,
		}

		callArgs := map[string]interface{}{
			"env":            input.Env,
			"api":            input.API,
			"method":         strings.ToUpper(strings.TrimSpace(input.Method)),
			"endpoint":       input.Endpoint,
			"expectStatus":   input.ExpectStatus,
			"pathParams":     input.PathParams,
			"queryParams":    input.QueryParams,
			"headers":        input.Headers,
			"requestBody":    input.RequestBody,
			"responseChecks": input.ResponseChecks,
		}

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: summary},
				&mcp.TextContent{Text: "```gherkin\n" + strings.TrimSpace(gherkin) + "\n```"},
			},
		}

		output := scenarioSuggestResult{
			FeatureTitle:       featureTitle,
			FeatureDescription: featureDescription,
			FeatureTags:        featureTags,
			Scenario:           scenario,
			SuggestedPath:      suggestedPath,
			Gherkin:            gherkin,
			Summary:            summary,
			FeatureTemplate:    template,
			Command:            []string{"gherkio.feature.write"},
			CallArguments:      callArgs,
		}

		return result, output, nil
	})
}

func buildFeatureContent(title, description string, tags []string, scenarios []featureWriteScenario) string {
	var buf strings.Builder
	dtags := dedupStrings(tags)
	if len(dtags) > 0 {
		buf.WriteString("@")
		buf.WriteString(strings.Join(dtags, " @"))
		buf.WriteString("\n")
	}
	buf.WriteString("Feature: ")
	buf.WriteString(strings.TrimSpace(title))
	buf.WriteString("\n")
	if strings.TrimSpace(description) != "" {
		for _, line := range strings.Split(description, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			buf.WriteString("  ")
			buf.WriteString(trimmed)
			buf.WriteString("\n")
		}
	}
	for _, sc := range scenarios {
		buf.WriteString("\n")
		stags := dedupStrings(sc.Tags)
		if len(stags) > 0 {
			buf.WriteString("  @")
			buf.WriteString(strings.Join(stags, " @"))
			buf.WriteString("\n")
		}
		buf.WriteString("  Scenario: ")
		buf.WriteString(strings.TrimSpace(sc.Name))
		buf.WriteString("\n")
		for _, step := range sc.Steps {
			trimmed := strings.TrimSpace(step)
			if trimmed == "" {
				continue
			}
			buf.WriteString("    ")
			buf.WriteString(trimmed)
			buf.WriteString("\n")
		}
	}
	if !strings.HasSuffix(buf.String(), "\n") {
		buf.WriteString("\n")
	}
	return buf.String()
}

func runGherkioCommand(ctx context.Context, repoRoot string, cliArgs []string) (commandResult, error) {
	bin := os.Getenv("GHERKIO_BIN")
	if strings.TrimSpace(bin) == "" {
		bin = filepath.Join(repoRoot, "gherkio")
	}
	if !filepath.IsAbs(bin) {
		bin = filepath.Join(repoRoot, bin)
	}
	if _, err := os.Stat(bin); err != nil {
		if os.IsNotExist(err) {
			return commandResult{}, fmt.Errorf("gherkio binary not found at %s", bin)
		}
		return commandResult{}, fmt.Errorf("stat gherkio binary: %w", err)
	}

	cmd := exec.CommandContext(ctx, bin, cliArgs...)
	cmd.Dir = repoRoot
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if ps := cmd.ProcessState; ps != nil {
		exitCode = ps.ExitCode()
	}

	output := commandResult{
		Status:     "ok",
		ExitCode:   exitCode,
		Command:    append([]string{bin}, cliArgs...),
		DurationMs: duration.Milliseconds(),
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
	}

	if err != nil {
		output.Status = "error"
		output.Error = err.Error()
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return output, nil
		}
		return output, err
	}

	if exitCode != 0 {
		output.Status = "error"
	}

	return output, nil
}

func sortedPairs(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, fmt.Sprintf("%s=%s", k, m[k]))
	}
	return out
}

func dedupStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func deriveScenarioName(input scenarioSuggestInput) string {
	if name := strings.TrimSpace(input.ScenarioName); name != "" {
		return name
	}
	if purpose := strings.TrimSpace(input.Purpose); purpose != "" {
		return purpose
	}
	api := strings.TrimSpace(input.API)
	if api != "" && input.ExpectStatus > 0 {
		return fmt.Sprintf("%s returns %d", api, input.ExpectStatus)
	}
	if api != "" {
		return fmt.Sprintf("Exercise %s", api)
	}
	if input.ExpectStatus > 0 {
		return fmt.Sprintf("Expect status %d", input.ExpectStatus)
	}
	return "Generated API scenario"
}

func deriveFeatureTitle(input scenarioSuggestInput) string {
	if title := strings.TrimSpace(input.FeatureTitle); title != "" {
		return title
	}
	api := strings.TrimSpace(input.API)
	if api != "" {
		return fmt.Sprintf("Validate %s API", api)
	}
	if purpose := strings.TrimSpace(input.Purpose); purpose != "" {
		return purpose
	}
	return "API validation"
}

func buildScenarioSteps(input scenarioSuggestInput) []string {
	var steps []string
	var givens []string

	if env := strings.TrimSpace(input.Env); env != "" {
		givens = append(givens, fmt.Sprintf("the \"%s\" environment is configured", env))
	}
	for _, pre := range input.Preconditions {
		pre = strings.TrimSpace(pre)
		if pre == "" {
			continue
		}
		givens = append(givens, pre)
	}
	for i, g := range givens {
		keyword := "Given"
		if i > 0 {
			keyword = "And"
		}
		steps = append(steps, fmt.Sprintf("%s %s", keyword, g))
	}

	method := strings.ToUpper(strings.TrimSpace(input.Method))
	if method == "" {
		method = "GET"
	}
	endpoint := strings.TrimSpace(input.Endpoint)
	api := strings.TrimSpace(input.API)
	var whenPhrase string
	switch {
	case api != "" && endpoint != "":
		whenPhrase = fmt.Sprintf("I call the \"%s\" API using %s %s", api, method, endpoint)
	case api != "":
		whenPhrase = fmt.Sprintf("I call the \"%s\" API using %s", api, method)
	case endpoint != "":
		whenPhrase = fmt.Sprintf("I send a %s request to %s", method, endpoint)
	default:
		whenPhrase = fmt.Sprintf("I send a %s request", method)
	}
	steps = append(steps, fmt.Sprintf("When %s", whenPhrase))

	for _, kv := range sortedKeyVals(input.PathParams) {
		steps = append(steps, fmt.Sprintf("And the path parameter \"%s\" is \"%s\"", kv.Key, kv.Value))
	}
	for _, kv := range sortedKeyVals(input.QueryParams) {
		steps = append(steps, fmt.Sprintf("And the query parameter \"%s\" is \"%s\"", kv.Key, kv.Value))
	}
	for _, kv := range sortedKeyVals(input.Headers) {
		steps = append(steps, fmt.Sprintf("And the \"%s\" header is \"%s\"", kv.Key, kv.Value))
	}

	if body := strings.TrimSpace(input.RequestBody); body != "" {
		steps = append(steps, formatDocstringStep(body))
	}

	startedThen := false
	if input.ExpectStatus > 0 {
		steps = append(steps, fmt.Sprintf("Then the response status should be %d", input.ExpectStatus))
		startedThen = true
	}
	for i, check := range input.ResponseChecks {
		check = strings.TrimSpace(check)
		if check == "" {
			continue
		}
		keyword := "And"
		if !startedThen && i == 0 {
			keyword = "Then"
			startedThen = true
		}
		steps = append(steps, fmt.Sprintf("%s %s", keyword, check))
	}
	if !startedThen {
		steps = append(steps, "Then the response should match the acceptance criteria")
	}

	return steps
}

func suggestedFeaturePath(input scenarioSuggestInput, scenarioName string) string {
	api := strings.TrimSpace(input.API)
	purpose := strings.TrimSpace(input.Purpose)
	slugSource := api
	if slugSource != "" && purpose != "" {
		slugSource = slugSource + "-" + purpose
	} else if slugSource == "" {
		slugSource = scenarioName
	}
	slug := slugify(slugSource)
	if slug == "" {
		slug = "generated-scenario"
	}
	return filepath.ToSlash(filepath.Join("features", "generated", slug+".feature"))
}

func formatDocstringStep(body string) string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		lines[i] = "      " + strings.TrimRight(line, " ")
	}
	return "And the request body is:\n      \"\"\"\n" + strings.Join(lines, "\n") + "\n      \"\"\""
}

type keyVal struct {
	Key   string
	Value string
}

func sortedKeyVals(m map[string]string) []keyVal {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		if strings.TrimSpace(k) == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]keyVal, 0, len(keys))
	for _, k := range keys {
		v := strings.TrimSpace(m[k])
		if v == "" {
			continue
		}
		out = append(out, keyVal{Key: k, Value: v})
	}
	return out
}

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(in string) string {
	in = strings.ToLower(strings.TrimSpace(in))
	if in == "" {
		return ""
	}
	slug := slugPattern.ReplaceAllString(in, "-")
	slug = strings.Trim(slug, "-")
	return slug
}
