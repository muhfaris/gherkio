package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

func registerTools(host *Server, mcpServer *mcp.Server) error {
	registerCallTool(host, mcpServer)
	registerRunTool(host, mcpServer)
	registerFeatureWriteTool(host, mcpServer)
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
