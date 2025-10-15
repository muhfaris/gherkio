package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/muhfaris/gherkio/internal/loader"
	"github.com/muhfaris/gherkio/internal/report"
	"github.com/muhfaris/gherkio/internal/runner"
	"github.com/spf13/cobra"
)

var callCmd = &cobra.Command{
	Use:   "call",
	Short: "Single-endpoint call using catalogs",
	Long:  `Performs a single API call based on the defined catalogs and flags.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCall(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(callCmd)
	callCmd.Flags().String("env", "", "Environment name (must exist in gherkio/envs/<name>.yaml)")
	callCmd.Flags().String("api", "", "Endpoint key from gherkio/apis/*.yaml (e.g., users.getById)")
	callCmd.Flags().String("body", "", "Request body (JSON or multipart fixture)")
	callCmd.Flags().String("expect-status", "", "Expected HTTP status code")
	callCmd.Flags().StringArray("path", []string{}, "Path params (k=v)")
	callCmd.Flags().StringArray("query", []string{}, "Query params (k=v)")
	callCmd.Flags().StringArray("header", []string{}, "Headers (k=v)")
	callCmd.Flags().StringArray("report", []string{}, "Reporters (pretty|html|junit|cucumber|csv)")
}

func runCall(cmd *cobra.Command, args []string) error {
	envName, _ := cmd.Flags().GetString("env")
	apiKey, _ := cmd.Flags().GetString("api")
	body, _ := cmd.Flags().GetString("body")
	expectStatusStr, _ := cmd.Flags().GetString("expect-status")
	paths, _ := cmd.Flags().GetStringArray("path")
	queries, _ := cmd.Flags().GetStringArray("query")
	headers, _ := cmd.Flags().GetStringArray("header")
	reports, _ := cmd.Flags().GetStringArray("report")

	if envName == "" || apiKey == "" {
		return errors.New("--env and --api are required")
	}

	env, err := loader.LoadEnv("gherkio/envs", envName)
	if err != nil {
		return err
	}
	cat, err := loader.LoadCatalogs("gherkio/apis")
	if err != nil {
		return err
	}

	ctx := runner.NewContext(env, cat)
	req := runner.Request{
		APIKey:  apiKey,
		Path:    kvToMap(paths),
		Query:   kvToMap(queries),
		Headers: kvToMap(headers),
	}
	if body != "" {
		if strings.HasPrefix(body, "@") {
			fixturePath := strings.TrimPrefix(body, "@")
			payload, err := runner.LoadFixtureFile(fixturePath, ctx.Store)
			if err != nil {
				return err
			}
			runner.ApplyFixture(&req, payload)
		} else {
			req.Multipart = nil
			req.Body = []byte(body)
			if req.Headers == nil {
				req.Headers = map[string]string{}
			}
			if _, ok := req.Headers["Content-Type"]; !ok {
				req.Headers["Content-Type"] = "application/json"
			}
		}
	}

	resp, _, err := runner.Call(ctx, req)
	if err != nil {
		return err
	}
	// simple pretty output
	report.NewPretty(os.Stdout).RecordSingle(req, resp)

	if expectStatusStr != "" {
		exp, _ := atoi(expectStatusStr)
		if resp.Status != exp {
			return fmt.Errorf("expect %d got %d", exp, resp.Status)
		}
	}
	// emit opted reporters (csv/html/junit/cucumber placeholder)
	meta := report.SummaryMeta{Env: envName, FeatureCount: 1}
	if len(reports) > 0 {
		csv := report.NewCSV()
		for _, spec := range reports {
			kind, path := splitKindPath(spec)
			lower := strings.ToLower(kind)
			switch lower {
			case "", "pretty":
				continue
			case "csv":
				_ = csv.AppendSingle(pathOrDefault(path, "reports/call.csv"), req, resp)
			default:
				if dbg, ok := parseHTMLKind(lower); ok {
					normPath, fromPath := normalizeHTMLPath(path, "reports/call.html")
					debugEnabled := dbg || fromPath
					h := report.NewHTML(normPath, debugEnabled, meta)
					step := report.StepDetail{
						Text:       fmt.Sprintf("call api %s", req.APIKey),
						Status:     statusLabel(resp.Status),
						DurationMs: 0,
					}
					if debugEnabled {
						step.Debug = &report.DebugInfo{
							RequestBody:    formatPayload(req.Body),
							ResponseBody:   formatPayload(resp.Body),
							ResponseStatus: resp.Status,
						}
					}
					res := report.Result{
						Feature:    "single",
						Scenario:   req.APIKey,
						Status:     statusLabel(resp.Status),
						DurationMs: 0,
						Steps:      []report.StepDetail{step},
					}
					h.Add(res)
					if err := h.Flush(); err != nil {
						return err
					}
					continue
				}
				fmt.Fprintf(os.Stderr, "unknown report kind %q\n", kind)
			}
		}
	}
	return nil
}

func atoi(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

func splitKindPath(spec string) (string, string) {
	parts := strings.SplitN(spec, ":", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func formatPayload(body []byte) string {
	if len(body) == 0 {
		return "<empty>"
	}
	trimmed := bytes.TrimSpace(body)
	if json.Valid(trimmed) {
		var buf bytes.Buffer
		if err := json.Indent(&buf, trimmed, "", "  "); err == nil {
			return truncateRunes(buf.String(), maxDebugRunes)
		}
	}
	return truncateRunes(string(body), maxDebugRunes)
}

const maxDebugRunes = 4000

func truncateRunes(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit]) + "\n... (truncated)"
}

func kvToMap(data []string) map[string]string {
	result := make(map[string]string)
	for _, item := range data {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}
func pathOrDefault(path, defaultPath string) string {
	if path != "" {
		return path
	}
	return defaultPath
}

func parseHTMLKind(kind string) (debug bool, ok bool) {
	s := strings.ToLower(kind)
	switch s {
	case "html":
		return false, true
	case "html-dbg", "html-debug":
		return true, true
	}
	return false, false
}

func normalizeHTMLPath(path, defaultPath string) (norm string, fromDefault bool) {
	if path != "" {
		return path, false
	}
	return defaultPath, true
}

func statusLabel(code int) string {
	if code >= 200 && code < 300 {
		return "passed"
	}
	return "failed"
}