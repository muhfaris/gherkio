package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/muhfaris/gherkio/internal/loader"
	"github.com/muhfaris/gherkio/internal/report"
	"github.com/muhfaris/gherkio/internal/runner"
)

func runCall(args []string) error {
	fs := flag.NewFlagSet("call", flag.ContinueOnError)
	var envName, apiKey, body, expectStatusStr string
	paths := collectKV(fs, "path")
	queries := collectKV(fs, "query")
	headers := collectKV(fs, "header")
	reports := multiString(fs, "report")
	fs.StringVar(&envName, "env", "", "environment name")
	fs.StringVar(&apiKey, "api", "", "endpoint key")
	fs.StringVar(&body, "body", "", "body: @file or inline JSON")
	fs.StringVar(&expectStatusStr, "expect-status", "", "expected status code")
	if err := fs.Parse(args); err != nil {
		return err
	}
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
		Path:    paths.ToMap(),
		Query:   queries.ToMap(),
		Headers: headers.ToMap(),
	}
	if body != "" {
		if strings.HasPrefix(body, "@") {
			b, err := os.ReadFile(strings.TrimPrefix(body, "@"))
			if err != nil {
				return err
			}
			req.Body = b
		} else {
			req.Body = []byte(body)
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
	if len(*reports) > 0 {
		csv := report.NewCSV()
		for _, spec := range *reports {
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

func atoi(s string) (int, error) { var n int; _, err := fmt.Sscanf(s, "%d", &n); return n, err }

func multiString(fs *flag.FlagSet, name string) *[]string {
	var out []string
	fs.Func(name, "repeatable", func(v string) error { out = append(out, v); return nil })
	return &out
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
