package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cucumber/godog"
	messages "github.com/cucumber/messages/go/v21"
	"github.com/muhfaris/gherkio/internal/loader"
	"github.com/tidwall/gjson"
)

type world struct {
	ctx             *Context
	lastReq         Request
	lastRes         Response
	flows           map[string]loader.Flow
	savedEnvHeaders map[string]string
	lastDurMs       int64
	lastStepArg     *messages.PickleStepArgument
	lastHTTPReq     *http.Request
	currentFeature  string
	currentScenario string
	scenarioStarted time.Time
	stepLogs        []StepLog
	stepStart       time.Time
	stepText        string
	pendingDebug    *StepDebug
}

// helper to bind + register docs in catalog
func bind(sc *godog.ScenarioContext, pattern, desc, example string, fn interface{}) {
	if sc != nil {
		sc.Step(pattern, fn)
	}
	stepCatalog.Add(pattern, desc, example)
}

func InitializeScenario(env loader.Env, cat loader.Catalog, flows map[string]loader.Flow) func(*godog.ScenarioContext) {
	return func(sc *godog.ScenarioContext) {
		w := &world{ctx: NewContext(env, cat), flows: flows}
		if sc != nil {
			sc.StepContext().Before(func(ctx context.Context, st *godog.Step) (context.Context, error) {
				w.lastStepArg = st.Argument
				if st != nil {
					w.stepText = st.Text
				} else {
					w.stepText = ""
				}
				w.stepStart = time.Now()
				w.pendingDebug = nil
				return ctx, nil
			})
			sc.StepContext().After(func(ctx context.Context, st *godog.Step, status godog.StepResultStatus, err error) (context.Context, error) {
				w.lastStepArg = nil
				dur := int64(0)
				if !w.stepStart.IsZero() {
					dur = time.Since(w.stepStart).Milliseconds()
				}
				text := w.stepText
				if text == "" && st != nil {
					text = st.Text
				}
				log := StepLog{Text: text, Status: mapStepStatus(status, err), DurationMs: dur, Error: errorText(err), Debug: w.pendingDebug}
				w.stepLogs = append(w.stepLogs, log)
				w.pendingDebug = nil
				w.stepStart = time.Time{}
				w.stepText = ""
				return ctx, nil
			})
			sc.Before(func(c context.Context, s *godog.Scenario) (context.Context, error) {
				// reset per-scenario
				w.ctx = NewContext(env, cat)
				w.lastReq = Request{}
				w.lastRes = Response{}
				w.flows = flows
				w.lastStepArg = nil
				w.lastHTTPReq = nil
				w.stepLogs = nil
				w.pendingDebug = nil
				// snapshot default env headers for this scenario
				w.savedEnvHeaders = map[string]string{}
				for k, v := range w.ctx.Env.Headers {
					w.savedEnvHeaders[k] = v
				}
				if s != nil {
					w.currentScenario = s.Name
					w.currentFeature = featureTitleFor(s.Uri)
				} else {
					w.currentScenario = ""
					w.currentFeature = ""
				}
				w.scenarioStarted = time.Now()
				normalizedVars := normalizeEnvVars(w.ctx.Env.Vars)
				if normalizedVars == nil {
					normalizedVars = map[string]any{}
				}
				w.ctx.Store["vars"] = deepCopyMap(normalizedVars)
				flat := flattenEnvVars(normalizedVars)
				keys := make([]string, 0, len(flat))
				for k := range flat {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					rendered := mustExec(flat[k], w.templateCtx())
					w.ctx.Store[k] = rendered
					setNestedStoreValue(w.ctx.Store, strings.Split(k, "."), rendered)
				}
				return c, nil
			})

			sc.After(func(ctx context.Context, s *godog.Scenario, err error) (context.Context, error) {
				// restore env headers
				w.ctx.Env.Headers = map[string]string{}
				for k, v := range w.savedEnvHeaders {
					w.ctx.Env.Headers[k] = v
				}
				status := "PASSED"
				if err != nil {
					if errors.Is(err, godog.ErrPending) {
						status = "PENDING"
					} else {
						status = "FAILED"
					}
				}
				feature := w.currentFeature
				if feature == "" && s != nil {
					feature = featureTitleFor(s.Uri)
				}
				scenarioName := w.currentScenario
				if scenarioName == "" && s != nil {
					scenarioName = s.Name
				}
				dur := int64(0)
				if !w.scenarioStarted.IsZero() {
					dur = time.Since(w.scenarioStarted).Milliseconds()
				}
				logs := append([]StepLog(nil), w.stepLogs...)
				recordScenario(feature, scenarioName, status, dur, logs)
				w.stepLogs = nil
				w.stepStart = time.Time{}
				w.stepText = ""
				w.currentFeature = ""
				w.currentScenario = ""
				w.scenarioStarted = time.Time{}
				w.pendingDebug = nil
				return ctx, nil
			})
		}

		// No-op (env & catalogs sudah diload di CLI)
		bind(sc, `^I load env [\"']([^\"']+)[\"']$`, "Load env", "", func(_ string) error { return nil })
		bind(sc, `^I load flows from [\"']([^\"']+)[\"']$`, "Load flows", "", func(_ string) error { return nil })

		// Override base URL for the current scenario
		bind(sc, `^(?:Given\s+)?the base URL is ["']([^"']*)["']$`, "Override base URL", "Given the base URL is 'https://api.example.com'", func(raw string) error {
			rendered := mustExec(raw, w.templateCtx())
			w.ctx.Env.BaseURL = rendered
			return nil
		})

		bind(sc, `^I include feature ["']([^"']+)["']$`, "Include feature", "Include feature \"users.feature\"",
			func(path string) error {
				full := filepath.Join("gherkio/features", path)
				if !strings.HasSuffix(full, ".feature") {
					full += ".feature"
				}
				if _, err := os.Stat(full); err != nil {
					return fmt.Errorf("included feature not found: %s", full)
				}

				// Parse and execute via new godog suite (isolated)
				opts := &godog.Options{Format: "pretty", Paths: []string{full}}
				suite := godog.TestSuite{
					Name:                fmt.Sprintf("include-%s", filepath.Base(full)),
					ScenarioInitializer: InitializeScenario(w.ctx.Env, w.ctx.Cat, w.flows),
					Options:             opts,
				}
				if code := suite.Run(); code != 0 {
					return fmt.Errorf("included feature %s failed with status %d", full, code)
				}
				return nil
			})

		// header "<name>" should exist
		bind(sc, `^header [\"']([^\"']+)[\"'] should exist$`, "Assert header exists", "header 'Content-Type' should exist", func(name string) error {
			key := http.CanonicalHeaderKey(name)
			if _, ok := w.lastRes.Header[key]; !ok {
				return fmt.Errorf("header %q not found", key)
			}
			return nil
		})

		// header "<name>" should equal "<value>"
		bind(sc, `^header [\"']([^\"']+)[\"'] should equal [\"']([^\"']+)[\"']$`, "Assert header value", "header 'Content-Type' should equal 'application/json'", func(name, want string) error {
			key := http.CanonicalHeaderKey(name)
			vals, ok := w.lastRes.Header[key]
			if !ok || len(vals) == 0 {
				return fmt.Errorf("header %q not found", key)
			}
			got := strings.Join(vals, ", ")
			if got != want {
				return fmt.Errorf("expect header %s=%q, got %q", key, want, got)
			}
			return nil
		})

		// Set params via table
		bind(sc, `^I set path params:$`, "Set path parameters", "I set path params:\n| id | 123 |", func(table *godog.Table) error {
			w.lastReq.Path = tableToMap(table)
			return nil
		})
		bind(sc, `^I set query params:$`, "Set query parameters", "I set query params:\n| page | 1 |\n| limit | 10 |", func(table *godog.Table) error {
			rendered := map[string]string{}
			if table != nil {
				ctx := w.templateCtx()
				for _, r := range table.Rows {
					if len(r.Cells) < 2 {
						continue
					}
					key := strings.TrimSpace(r.Cells[0].Value)
					val := strings.TrimSpace(r.Cells[1].Value)
					out, err := execTemplate(val, ctx)
					if err != nil {
						return fmt.Errorf("render query param %s: %w", key, err)
					}
					rendered[key] = out
				}
			}
			w.lastReq.Query = rendered
			return nil
		})
		bind(sc, `^I clear query params$`, "Clear query parameters", "I clear query params", func() error {
			w.lastReq.Query = nil
			return nil
		})
		bind(sc, `^I set headers:$`, "Set headers", "I set headers:\n| Authorization | Bearer token |\n| Content-Type | application/json |", func(table *godog.Table) error {
			w.lastReq.Headers = tableToMap(table)
			return nil
		})

		// Call API (no body)
		bind(sc, `^I call API [\"']([^\"']+)[\"']$`, "Call API endpoint", "I call API 'users.getById'", func(key string) error {
			w.lastReq.APIKey = key
			t0 := time.Now()
			res, httpReq, err := Call(w.ctx, w.lastReq)
			w.lastDurMs = time.Since(t0).Milliseconds()
			w.lastRes = res
			w.lastHTTPReq = httpReq
			w.captureDebug(w.lastReq, w.lastRes, httpReq)
			return err
		})

		// Call API with body: """ {json} """
		bind(sc, `^I call API [\"']([^\"']+)[\"'] with body:$`, "Call API with structured body", "I call API 'auth.login' with body:\n| username | superadmin |\n| password | Admin@123 |",
			func(key string) error {
				w.lastReq.APIKey = key
				arg := w.lastStepArg
				if arg == nil {
					return errors.New("body is empty")
				}

				ctx := w.templateCtx()
				switch {
				case arg.DocString != nil:
					raw := arg.DocString.Content
					if strings.TrimSpace(raw) == "" {
						return errors.New("body is empty")
					}
					rendered := mustExec(raw, ctx)
					w.lastReq.Body = []byte(rendered)
				case arg.DataTable != nil:
					m := pickleTableToMap(arg.DataTable)
					if len(m) == 0 {
						return errors.New("body is empty")
					}
					rendered := renderMap(m, ctx)
					b, err := json.Marshal(rendered)
					if err != nil {
						return err
					}
					w.lastReq.Body = b
				default:
					return errors.New("body is empty")
				}

				if w.lastReq.Headers == nil {
					w.lastReq.Headers = map[string]string{}
				}
				if _, ok := w.lastReq.Headers["Content-Type"]; !ok {
					w.lastReq.Headers["Content-Type"] = "application/json"
				}

				t0 := time.Now()
				res, httpReq, err := Call(w.ctx, w.lastReq)
				w.lastDurMs = time.Since(t0).Milliseconds()
				w.lastRes = res
				w.lastHTTPReq = httpReq
				w.captureDebug(w.lastReq, w.lastRes, httpReq)
				return err
			})

		// Call API with: (table → JSON body)
		// example:
		// When I call API "auth.login" with:
		//   | username | superadmin |
		//   | password | Admin@123  |
		bind(sc, `^I call API [\"']([^\"']+)[\"'] with:$`, "Call API with table as JSON body", "I call API 'auth.login' with:\n| username | superadmin |\n| password | Admin@123 |", func(key string, table *godog.Table) error {
			w.lastReq.APIKey = key
			m := tableToMap(table)
			ctx := w.templateCtx()
			m = renderMap(m, ctx)
			b, err := json.Marshal(m)
			if err != nil {
				return err
			}
			// set default Content-Type if not provided
			if w.lastReq.Headers == nil {
				w.lastReq.Headers = map[string]string{}
			}
			if _, ok := w.lastReq.Headers["Content-Type"]; !ok {
				w.lastReq.Headers["Content-Type"] = "application/json"
			}
			w.lastReq.Body = b
			t0 := time.Now()
			res, httpReq, err := Call(w.ctx, w.lastReq)
			w.lastDurMs = time.Since(t0).Milliseconds()
			w.lastRes = res
			w.lastHTTPReq = httpReq
			w.captureDebug(w.lastReq, w.lastRes, httpReq)
			return err
		})

		// I call API "<key>" using fixture "<path>"
		bind(sc, `^I call API [\"']([^\"']+)[\"'] using fixture [\"']([^\"']+)[\"']$`, "Call API using fixture file", "I call API 'users.create' using fixture 'user.json'", func(key, fpath string) error {
			fpath = filepath.Join("gherkio/fixtures/", fpath)
			payload, err := LoadFixtureFile(fpath, w.ctx.Store)
			if err != nil {
				return fmt.Errorf("load fixture %s: %w", fpath, err)
			}
			ApplyFixture(&w.lastReq, payload)

			w.lastReq.APIKey = key
			t0 := time.Now()
			res, httpReq, err := Call(w.ctx, w.lastReq)
			w.lastDurMs = time.Since(t0).Milliseconds()
			w.lastRes = res
			w.lastHTTPReq = httpReq
			w.captureDebug(w.lastReq, w.lastRes, httpReq)
			return err
		})

		// I run flow "<name>" with:
		bind(sc, `^I run flow [\"']([^\"']+)[\"'] with:$`, "Run flow with parameters", "I run flow 'login' with:\n| username | demo |\n| password | secret |", func(name string, table *godog.Table) error {
			args := tableToMap(table)
			args = renderMap(args, w.templateCtx())
			return w.runFlow(name, args)
		})

		// I run flow "<name>"
		bind(sc, `^I run flow [\"']([^\"']+)[\"']$`, "Run flow without parameters", "I run flow 'auth'", func(name string) error {
			return w.runFlow(name, map[string]string{})
		})

		// Assertions
		bind(sc, `^response status should be (\d+)$`, "Assert response status code", "response status should be 200", func(code int) error {
			if w.lastRes.Status != code {
				return fmt.Errorf("expect %d got %d", code, w.lastRes.Status)
			}
			return nil
		})

		// response status should be in 200-299 (generic range)
		bind(sc, `^response status should be in (\d{3})-(\d{3})$`, "Assert response status in range", "response status should be in 200-299", func(lo, hi int) error {
			if w.lastRes.Status < lo || w.lastRes.Status > hi {
				return fmt.Errorf("status %d not in range %d-%d", w.lastRes.Status, lo, hi)
			}
			return nil
		})

		// response body should contain "<text>"
		bind(sc, `^response body should contain [\"']([^\"']+)[\"']$`, "Assert substring in response body", "response body should contain 'success'", func(sub string) error {
			if !strings.Contains(string(w.lastRes.Body), sub) {
				return fmt.Errorf("response body does not contain %q", sub)
			}
			return nil
		})

		// response time should be <op> <ms>
		bind(sc, `^response time should be (==|!=|>=|>|<=|<) (\d+)ms$`, "Assert response time", "response time should be <= 500ms", func(op string, want int64) error {
			got := w.lastDurMs
			ok := false
			switch op {
			case ">":
				ok = got > want
			case ">=":
				ok = got >= want
			case "<":
				ok = got < want
			case "<=":
				ok = got <= want
			case "==":
				ok = got == want
			case "!=":
				ok = got != want
			}
			if !ok {
				return fmt.Errorf("assert failed: %d ms %s %d ms", got, op, want)
			}
			return nil
		})

		bind(sc, `^(?:Then\s+)?the store should contain:$`, "Assert store entries", "Then the store should contain:\n| key | value |", func(table *godog.Table) error {
			if table == nil || len(table.Rows) == 0 {
				return errors.New("table required")
			}
			rows := table.Rows
			start := 0
			if len(rows) > 0 && len(rows[0].Cells) >= 1 {
				head := strings.TrimSpace(strings.ToLower(rows[0].Cells[0].Value))
				if head == "key" || head == "name" {
					start = 1
				}
			}
			storeCtx := w.templateCtx()
			for i := start; i < len(rows); i++ {
				cells := rows[i].Cells
				if len(cells) == 0 {
					continue
				}
				key := strings.TrimSpace(cells[0].Value)
				if key == "" {
					return fmt.Errorf("row %d: empty key", i+1)
				}
				val, ok := w.ctx.Store[key]
				if !ok {
					return fmt.Errorf("store key %q not found", key)
				}
				if len(cells) > 1 {
					wantRaw := cells[1].Value
					want := mustExec(strings.TrimSpace(wantRaw), storeCtx)
					if !storeValueMatches(val, want) {
						return fmt.Errorf("store[%s]=%s does not match %q", key, formatAny(val), want)
					}
				}
			}
			return nil
		})

		// json "<path>" should exist
		bind(sc, `^json ["'](.+?)["'] should exist$`, "Assert JSON path exists", "json '$.data.id' should exist", func(path string) error {
			path = w.renderPath(path)
			v := getJSONPath(w.lastRes.Body, path)
			if !v.Exists() {
				return fmt.Errorf("json path %q not found", path)
			}
			return nil
		})

		// I set auth "<name>"
		bind(sc, `^I set auth [\"']([^\"']+)[\"']$`, "Set authentication profile", "I set auth 'bearer'", func(name string) error {
			w.ctx.CurrentAuth = name
			return nil
		})

		// save "<jsonpath>" as "<key>"
		bind(sc, `^save ["'](.+?)["'] as ["']([^"']+)["']$`, "Save JSONPath to store", "save '$.access_token' as 'token'", func(path, key string) error {
			path = w.renderPath(path)
			v := getJSONPath(w.lastRes.Body, path)
			if !v.Exists() {
				return fmt.Errorf("json path %q not found", path)
			}
			w.ctx.Store[key] = v.Value()
			return nil
		})

		bind(sc, `^save request json ["'](.+?)["'] as ["']([^"']+)["']$`, "Save request JSONPath to store", "save request json '$.name' as 'room_name'", func(path, key string) error {
			if len(w.lastReq.Body) == 0 {
				return errors.New("last request body is empty")
			}
			path = w.renderPath(path)
			v := getJSONPath(w.lastReq.Body, path)
			if !v.Exists() {
				return fmt.Errorf("request json path %q not found", path)
			}
			w.ctx.Store[key] = v.Value()
			return nil
		})

		bind(sc, `^save request body as [\"']([^\"']+)[\"']$`, "Save full request body", "save request body as 'request_payload'", func(key string) error {
			if len(w.lastReq.Body) == 0 {
				return errors.New("last request body is empty")
			}
			var payload any
			if err := json.Unmarshal(w.lastReq.Body, &payload); err != nil {
				return fmt.Errorf("parse request body: %w", err)
			}
			w.ctx.Store[key] = payload
			return nil
		})

		// json "<path>" should equal "<value>"
		bind(sc, `^json ["'](.+?)["'] should equal ["']([^"']+)["']$`, "Assert JSON value equals", "json '$.status' should equal 'success'", func(path, want string) error {
			path = w.renderPath(path)
			val := getJSONPath(w.lastRes.Body, path)
			if !val.Exists() {
				return fmt.Errorf("json path %q not found", path)
			}
			if val.Str != want && val.Raw != want {
				return fmt.Errorf("expect %s got %s", want, val.Raw)
			}
			return nil
		})

		// json "<path>" should equal true|false|null
		bind(sc, `^json ["'](.+?)["'] should equal (true|false|null)$`, "Assert JSON literal equals", "json '$.flags.isEnabled' should equal true", func(path, literal string) error {
			path = w.renderPath(path)
			val := getJSONPath(w.lastRes.Body, path)
			if !val.Exists() {
				return fmt.Errorf("json path %q not found", path)
			}
			switch strings.ToLower(literal) {
			case "true":
				if val.Type != gjson.True {
					return fmt.Errorf("expect true got %s", val.Raw)
				}
			case "false":
				if val.Type != gjson.False {
					return fmt.Errorf("expect false got %s", val.Raw)
				}
			case "null":
				if val.Type != gjson.Null {
					return fmt.Errorf("expect null got %s", val.Raw)
				}
			}
			return nil
		})

		bind(sc, `^json ["'](.+?)["'] should equal store ["']([^"']+)["']$`, "Assert JSON equals stored value", "json '$.data.id' should equal store 'resource_id'", func(path, key string) error {
			path = w.renderPath(path)
			val := getJSONPath(w.lastRes.Body, path)
			if !val.Exists() {
				return fmt.Errorf("json path %q not found", path)
			}
			storeVal, ok := w.ctx.Store[key]
			if !ok {
				return fmt.Errorf("store key %q not found", key)
			}
			actual := val.Value()
			if compareJSON(actual, storeVal, false) {
				return nil
			}
			return fmt.Errorf("json %s=%s does not equal store[%s]=%s", path, val.Raw, key, formatAny(storeVal))
		})

		bind(sc, `^json ["'](.+?)["'] should equal store ["']([^"']+)["'] ignoring order$`, "Assert JSON equals stored value (ignore order)", "json '$.data.tags' should equal store 'expected_tags' ignoring order", func(path, key string) error {
			path = w.renderPath(path)
			val := getJSONPath(w.lastRes.Body, path)
			if !val.Exists() {
				return fmt.Errorf("json path %q not found", path)
			}
			storeVal, ok := w.ctx.Store[key]
			if !ok {
				return fmt.Errorf("store key %q not found", key)
			}
			if compareJSON(val.Value(), storeVal, true) {
				return nil
			}
			return fmt.Errorf("json %s=%s does not equal store[%s]=%s (ignoring order)", path, val.Raw, key, formatAny(storeVal))
		})

		bind(sc, `^json ["'](.+?)["'] should match store request ["']([^"']+)["']$`, "Assert JSON equals stored request JSON", "json '$.data' should match store request 'meeting_room_payload'", func(path, key string) error {
			path = w.renderPath(path)
			actual := getJSONPath(w.lastRes.Body, path)
			if !actual.Exists() {
				return fmt.Errorf("json path %q not found", path)
			}
			src, ok := w.ctx.Store[key]
			if !ok {
				return fmt.Errorf("store key %q not found", key)
			}
			actualVal := actual.Value()
			if compareJSON(actualVal, src, false) {
				return nil
			}
			actualJSON, _ := json.Marshal(actualVal)
			expectedJSON, _ := json.Marshal(src)
			return fmt.Errorf("json %s does not match store[%s]\nexpected: %s\nactual:   %s", path, key, expectedJSON, actualJSON)
		})

		// json "<path>" should not be empty
		bind(sc, `^json ["'](.+?)["'] should not be empty$`, "Assert JSON path is not empty", "json '$.data' should not be empty", func(path string) error {
			path = w.renderPath(path)
			val := getJSONPath(w.lastRes.Body, path)
			if !val.Exists() {
				return fmt.Errorf("json path %q not found", path)
			}
			// consider empty if "", null, [] or {}
			raw := strings.TrimSpace(val.Raw)
			if raw == `""` || raw == "null" || raw == "[]" || raw == "{}" || raw == "" {
				return fmt.Errorf("json %q is empty: %s", path, raw)
			}
			// if it's string type, also ensure non-empty
			if val.Type == gjson.String && strings.TrimSpace(val.Str) == "" {
				return fmt.Errorf("json %q is empty string", path)
			}
			return nil
		})

		// I create <n> document groups
		bind(sc, `^I create (\\d+) document groups$`, "Create multiple document groups", "I create 5 document groups", func(count int) error {
			if count <= 0 {
				return nil
			}
			suffix := fmt.Sprintf("%d", time.Now().UnixNano())
			ids := make([]string, 0, count)
			for i := 1; i <= count; i++ {
				name := fmt.Sprintf("Document Group %s-%d", suffix, i)
				code := fmt.Sprintf("DOC-%s-%d", suffix, i)
				payload := map[string]any{"name": name, "code": code}
				body, err := json.Marshal(payload)
				if err != nil {
					return err
				}
				req := Request{APIKey: "document.group.create", Body: body, Headers: map[string]string{"Content-Type": "application/json"}}
				res, httpReq, err := Call(w.ctx, req)
				w.lastReq, w.lastRes, w.lastHTTPReq = req, res, httpReq
				if err != nil {
					return err
				}
				if res.Status < 200 || res.Status >= 300 {
					return fmt.Errorf("document.group.create returned status %d", res.Status)
				}
				if id := gjson.GetBytes(res.Body, "data.id"); id.Exists() {
					ids = append(ids, id.String())
					w.ctx.Store["document_group_id"] = id.String()
				}
				if i == 1 {
					w.ctx.Store["document_group_search_name"] = name
				}
			}
			w.ctx.Store["document_group_ids"] = ids
			w.lastReq = Request{}
			return nil
		})

		// I delete created document groups
		bind(sc, `^I delete created document groups$`, "Delete generated document groups", "I delete created document groups", func() error {
			raw, ok := w.ctx.Store["document_group_ids"]
			if !ok {
				return errors.New("document_group_ids not found in store")
			}
			ids, ok := raw.([]string)
			if !ok {
				return fmt.Errorf("document_group_ids has unexpected type %T", raw)
			}
			if len(ids) == 0 {
				return errors.New("document_group_ids is empty")
			}
			for _, id := range ids {
				w.ctx.Store["document_group_id"] = id
				req := Request{APIKey: "document.group.delete"}
				res, httpReq, err := Call(w.ctx, req)
				w.lastReq, w.lastRes, w.lastHTTPReq = req, res, httpReq
				if err != nil {
					return err
				}
				if res.Status < 200 || res.Status >= 300 {
					return fmt.Errorf("document.group.delete returned status %d", res.Status)
				}
			}
			delete(w.ctx.Store, "document_group_ids")
			return nil
		})

		// json "<path>" should be empty
		bind(sc, `^json ["'](.+?)["'] should be empty$`, "Assert JSON path is empty", "json '$.data' should be empty", func(path string) error {
			path = w.renderPath(path)
			val := getJSONPath(w.lastRes.Body, path)
			if !val.Exists() {
				return fmt.Errorf("json path %q not found", path)
			}
			raw := strings.TrimSpace(val.Raw)
			if raw == "" || raw == `""` || raw == "null" || raw == "[]" || raw == "{}" {
				return nil
			}
			if val.Type == gjson.String && strings.TrimSpace(val.Str) == "" {
				return nil
			}
			return fmt.Errorf("json %q is not empty: %s", path, raw)
		})

		// json "<path>" should not exist
		bind(sc, `^json ["'](.+?)["'] should not exist$`, "Assert JSON path does not exist", "json '$.error' should not exist", func(path string) error {
			path = w.renderPath(path)
			val := getJSONPath(w.lastRes.Body, path)
			if val.Exists() {
				return fmt.Errorf("json path %q exists (value: %s)", path, val.Raw)
			}
			return nil
		})

		// json "<path>" should match "<regex>"
		bind(sc, `^json ["'](.+?)["'] should match ["']([^"']+)["']$`, "Assert JSON value matches regex", "json '$.email' should match '[a-z]+@example.com'", func(path, rx string) error {
			path = w.renderPath(path)
			v := getJSONPath(w.lastRes.Body, path)
			if !v.Exists() {
				return fmt.Errorf("json path %q not found", path)
			}
			re, err := regexp.Compile(rx)
			if err != nil {
				return fmt.Errorf("invalid regex %q: %w", rx, err)
			}
			// Prefer string value; fallback to raw
			s := v.Str
			if v.Type != gjson.String {
				s = v.Raw
			}
			if !re.MatchString(s) {
				return fmt.Errorf("value %q does not match %q", s, rx)
			}
			return nil
		})

		// json "<path>" should be <op> <number>
		bind(sc, `^json ["'](.+?)["'] should be (==|!=|>=|>|<=|<) ([0-9.]+)$`, "Assert JSON numeric value", "json '$.age' should be >= 18", func(path, op, wantStr string) error {
			path = w.renderPath(path)
			v := getJSONPath(w.lastRes.Body, path)
			if !v.Exists() {
				return fmt.Errorf("json path %q not found", path)
			}

			// parse actual number
			var got float64
			switch v.Type {
			case gjson.Number:
				got = v.Float()
			case gjson.String:
				f, err := strconv.ParseFloat(strings.TrimSpace(v.Str), 64)
				if err != nil {
					return fmt.Errorf("value at %q is string %q, not numeric", path, v.Str)
				}
				got = f
			default:
				// try parse from raw (e.g., "42" or 42)
				f, err := strconv.ParseFloat(strings.Trim(v.Raw, `"`), 64)
				if err != nil {
					return fmt.Errorf("value at %q is not numeric: %s", path, v.Raw)
				}
				got = f
			}

			want, err := strconv.ParseFloat(wantStr, 64)
			if err != nil {
				return fmt.Errorf("invalid number %q: %w", wantStr, err)
			}

			ok := false
			switch op {
			case ">":
				ok = got > want
			case ">=":
				ok = got >= want
			case "<":
				ok = got < want
			case "<=":
				ok = got <= want
			case "==":
				ok = got == want
			case "!=":
				ok = got != want
			}
			if !ok {
				return fmt.Errorf("assert failed: %v %s %v (path=%s)", got, op, want, path)
			}
			return nil
		})

		// json "<path>" length should be <op> <n>
		bind(sc, `^json ["'](.+?)["'] length should be (==|!=|>=|>|<=|<) (\d+)$`, "Assert JSON length", "json '$.items' length should be == 5", func(path, op string, n int) error {
			path = w.renderPath(path)
			v := getJSONPath(w.lastRes.Body, path)
			if !v.Exists() {
				return fmt.Errorf("json path %q not found", path)
			}

			var got int
			switch v.Type {
			case gjson.JSON:
				// try array length; if not array, use raw string length as fallback
				if v.IsArray() {
					got = len(v.Array())
				} else if v.IsObject() {
					got = len(v.Map())
				} else {
					got = len(v.Raw)
				}
			case gjson.String:
				got = len(v.Str)
			default:
				// for numbers/bools/null → use raw length
				got = len(v.Raw)
			}

			ok := false
			switch op {
			case ">":
				ok = got > n
			case ">=":
				ok = got >= n
			case "<":
				ok = got < n
			case "<=":
				ok = got <= n
			case "==":
				ok = got == n
			case "!=":
				ok = got != n
			}
			if !ok {
				return fmt.Errorf("assert failed: len(%s) %s %d (got %d)", path, op, n, got)
			}
			return nil
		})

		// set "<key>" to "<value>"
		bind(sc, `^set [\"']([^\"']+)[\"'] to [\"'](.+)[\"']$`, "Set value in store", "set 'base_url' to 'https://api.example.com'",
			func(k, v string) error {
				rendered, err := execTemplate(v, w.templateCtx())
				if err != nil {
					return fmt.Errorf("render value for %s: %w", k, err)
				}
				w.ctx.Store[k] = rendered
				setNestedStoreValue(w.ctx.Store, strings.Split(k, "."), rendered)
				return nil
			})

		// show variable "<key>"
		bind(sc, `^show variable [\"']([^\"']+)[\"']$`, "Print value from store", "show variable 'access_token'", func(key string) error {
			val, ok := w.ctx.Store[key]
			if !ok {
				return fmt.Errorf("store key %q not found", key)
			}

			fmt.Printf("store[%s] = ", key)
			switch v := val.(type) {
			case string:
				fmt.Printf("%s\n", v)
			case fmt.Stringer:
				fmt.Printf("%s\n", v.String())
			case []byte:
				fmt.Printf("%s\n", string(v))
			default:
				if b, err := json.MarshalIndent(v, "", "  "); err == nil {
					fmt.Printf("%s\n", string(b))
				} else {
					fmt.Printf("%#v\n", v)
				}
			}
			return nil
		})

		// json "<path>" should be one of:
		bind(sc, `^json ["'](.+?)["'] should be one of:$`, "Assert JSON value in list", "json '$.status' should be one of:\nactive\ninactive", func(path string, ds *godog.DocString) error {
			path = w.renderPath(path)
			v := getJSONPath(w.lastRes.Body, path)
			if !v.Exists() {
				return fmt.Errorf("json path %q not found", path)
			}

			// build set kandidat
			raw := strings.TrimSpace(ds.Content)
			lines := []string{}
			for _, ln := range strings.Split(raw, "\n") {
				ln = strings.TrimSpace(ln)
				if ln == "" {
					continue
				}
				// allow comma-separated on one line too
				parts := strings.Split(ln, ",")
				for _, p := range parts {
					s := strings.TrimSpace(p)
					s = strings.Trim(s, `"'`) // strip quotes if user adds them
					if s != "" {
						lines = append(lines, s)
					}
				}
			}
			if len(lines) == 0 {
				return fmt.Errorf("empty candidates list")
			}

			// normalize actual value
			got := v.Str
			if v.Type != gjson.String {
				got = strings.Trim(v.Raw, `"`)
			}

			valid := slices.Contains(lines, got)
			if valid {
				return nil
			}
			return fmt.Errorf("value %q is not in allowed set %v", got, lines)
		})

		// I wait <duration>
		bind(sc, `^I wait (\d+)(ms|s)$`, "Wait for duration", "I wait 100ms", func(n int, unit string) error {
			var d time.Duration
			if unit == "ms" {
				d = time.Duration(n) * time.Millisecond
			} else {
				d = time.Duration(n) * time.Second
			}
			time.Sleep(d)
			return nil
		})

		// save response body to file "<path>"
		bind(sc, `^save response body to file [\"']([^\"']+)[\"']$`, "Save response to file", "save response body to file 'response.json'", func(path string) error {
			if err := os.WriteFile(path, w.lastRes.Body, 0o644); err != nil {
				return err
			}
			return nil
		})

		// print response
		bind(sc, `^print response$`, "Print last response", "print response", func() error {
			fmt.Println("\n--- request ---")
			if w.lastHTTPReq != nil {
				fmt.Printf("method: %s\n", w.lastHTTPReq.Method)
				fmt.Printf("url: %s\n", w.lastHTTPReq.URL.String())
				if len(w.lastHTTPReq.Header) > 0 {
					fmt.Println("headers:")
					for k, vals := range w.lastHTTPReq.Header {
						fmt.Printf("  %s: %s\n", k, strings.Join(vals, ", "))
					}
				} else {
					fmt.Println("headers: <none>")
				}
			} else {
				fmt.Println("method: <unknown>")
				fmt.Println("url: <unknown>")
				fmt.Println("headers: <none>")
			}
			if len(w.lastReq.Body) > 0 {
				var js any
				if json.Unmarshal(w.lastReq.Body, &js) == nil {
					pretty, _ := json.MarshalIndent(js, "", "  ")
					fmt.Printf("body:\n%s\n", string(pretty))
				} else {
					fmt.Printf("body:\n%s\n", string(w.lastReq.Body))
				}
			} else {
				fmt.Println("body: <empty>")
			}

			fmt.Printf("\n--- response ---\nstatus: %d\n", w.lastRes.Status)

			if len(w.lastRes.Header) > 0 {
				fmt.Println("headers:")
				for k, vals := range w.lastRes.Header {
					fmt.Printf("  %s: %s\n", k, strings.Join(vals, ", "))
				}
			} else {
				fmt.Println("headers: <none>")
			}

			if len(w.lastRes.Body) > 0 {
				var js any
				if json.Unmarshal(w.lastRes.Body, &js) == nil {
					pretty, _ := json.MarshalIndent(js, "", "  ")
					fmt.Printf("body:\n%s\n", string(pretty))
				} else {
					fmt.Printf("body:\n%s\n", string(w.lastRes.Body))
				}
			} else {
				fmt.Println("body: <empty>")
			}
			fmt.Println("----------------")
			return nil
		})
	}
}

func tableToMap(t *godog.Table) map[string]string {
	m := map[string]string{}
	for _, r := range t.Rows {
		if len(r.Cells) >= 2 {
			m[strings.TrimSpace(r.Cells[0].Value)] = strings.TrimSpace(r.Cells[1].Value)
		}
	}
	return m
}

func pickleTableToMap(t *messages.PickleTable) map[string]string {
	out := map[string]string{}
	if t == nil {
		return out
	}
	for _, row := range t.Rows {
		if row == nil || len(row.Cells) < 2 {
			continue
		}
		key := strings.TrimSpace(row.Cells[0].Value)
		val := strings.TrimSpace(row.Cells[1].Value)
		out[key] = val
	}
	return out
}

// Accepts paths like: data.token, $.data.token, $data.token
func getJSONPath(body []byte, path string) gjson.Result {
	p := strings.TrimSpace(path)
	if len(p) > 0 && p[0] == '$' {
		p = p[1:]
		if len(p) > 0 && p[0] == '.' {
			p = p[1:]
		}
	}
	return gjson.GetBytes(body, p)
}

// --- helpers for flow execution ---
func (w *world) runFlow(name string, args map[string]string) error {
	f, ok := w.flows[name]
	if !ok {
		return fmt.Errorf("unknown flow: %s", name)
	}
	if args == nil {
		args = map[string]string{}
	}
	if len(f.Params) > 0 {
		for _, param := range f.Params {
			if _, ok := args[param]; ok {
				continue
			}
			if val, ok := w.ctx.Store[param]; ok {
				args[param] = fmt.Sprint(val)
				continue
			}
			compound := name + "." + param
			if val, ok := w.ctx.Store[compound]; ok {
				args[param] = fmt.Sprint(val)
			}
		}
	}
	// context for templating
	ctxMap := w.templateCtx()
	for k, v := range args {
		ctxMap[k] = v
	}

	for i, st := range f.Steps {
		// switch auth
		if st.SetAuth != "" {
			w.ctx.CurrentAuth = st.SetAuth
			continue
		}
		// build request
		req := Request{APIKey: st.Call}
		if len(st.Path) > 0 {
			req.Path = renderMap(st.Path, ctxMap)
		}
		if len(st.Query) > 0 {
			req.Query = renderMap(st.Query, ctxMap)
		}
		if len(st.Headers) > 0 {
			req.Headers = renderMap(st.Headers, ctxMap)
		}
		fixturePath := strings.TrimSpace(st.Fixture)
		if fixturePath != "" {
			resolved := filepath.Join("gherkio/fixtures/", fixturePath)
			payload, err := LoadFixtureFile(resolved, w.ctx.Store)
			if err != nil {
				return fmt.Errorf("flow %s step %d (%s): load fixture %s: %w", name, i+1, st.Call, fixturePath, err)
			}
			ApplyFixture(&req, payload)
		} else if st.Body != "" {
			req.Multipart = nil
			req.Body = []byte(mustExec(st.Body, ctxMap))
		}
		// default Content-Type if body present (and not multipart)
		if req.Multipart == nil && len(req.Body) > 0 {
			if req.Headers == nil {
				req.Headers = map[string]string{}
			}
			if _, ok := req.Headers["Content-Type"]; !ok {
				req.Headers["Content-Type"] = "application/json"
			}
		}
		res, httpReq, err := Call(w.ctx, req)
		w.lastReq, w.lastRes = req, res
		w.lastHTTPReq = httpReq
		w.captureDebug(w.lastReq, w.lastRes, httpReq)
		if err != nil {
			return fmt.Errorf("flow %s step %d (%s): %w", name, i+1, st.Call, err)
		}
		if st.Expect != nil && st.Expect.Status != 0 && res.Status != st.Expect.Status {
			return fmt.Errorf("flow %s step %d (%s): expect %d got %d", name, i+1, st.Call, st.Expect.Status, res.Status)
		}
		// saves
		for jp, key := range st.Save {
			v := getJSONPath(res.Body, jp)
			if !v.Exists() {
				preview := previewBody(res.Body)
				return fmt.Errorf("flow %s step %d: jsonpath %s not found (status=%d body=%s)", name, i+1, jp, res.Status, preview)
			}
			w.ctx.Store[key] = v.Value()
		}
	}
	return nil
}

func previewBody(body []byte) string {
	if len(body) == 0 {
		return "<empty>"
	}
	max := 512
	if len(body) > max {
		return fmt.Sprintf("%q...", string(body[:max]))
	}
	return fmt.Sprintf("%q", string(body))
}

func formatAny(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	case []byte:
		return string(t)
	default:
		if b, err := json.Marshal(t); err == nil {
			return string(b)
		}
		return fmt.Sprintf("%v", t)
	}
}

func mapStepStatus(status godog.StepResultStatus, err error) string {
	if err != nil {
		return "FAILED"
	}
	s := strings.ToUpper(status.String())
	if s == "UNKNOWN" {
		if err != nil {
			return "FAILED"
		}
		return "UNKNOWN"
	}
	return s
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

const maxDebugRunes = 4000

func (w *world) captureDebug(req Request, res Response, httpReq *http.Request) {
	method := "<unknown>"
	url := "<unknown>"
	headers := "<none>"
	if httpReq != nil {
		if httpReq.Method != "" {
			method = httpReq.Method
		}
		if httpReq.URL != nil {
			url = httpReq.URL.String()
		}
		if hdr := formatDebugHeaders(httpReq.Header); hdr != "" {
			headers = hdr
		}
	}
	reqBody := formatDebugBody(req.Body)
	resBody := formatDebugBody(res.Body)
	if isDebugConsole() {
		fmt.Printf("\n[debug] API: %s (%s %s) -> %d\n", req.APIKey, method, url, res.Status)
		if headers == "<none>" {
			fmt.Println("[debug] request headers: <none>")
		} else {
			fmt.Println("[debug] request headers:")
			for _, line := range strings.Split(headers, "\n") {
				fmt.Printf("[debug]   %s\n", line)
			}
		}
		fmt.Printf("[debug] request body:\n%s\n", reqBody)
		fmt.Printf("[debug] response body:\n%s\n", resBody)
	}
	if !isDebugCapture() {
		w.pendingDebug = nil
		return
	}
	w.pendingDebug = &StepDebug{
		APIKey:         req.APIKey,
		RequestMethod:  method,
		RequestURL:     url,
		RequestHeaders: headers,
		RequestBody:    reqBody,
		ResponseBody:   resBody,
		ResponseStatus: res.Status,
	}
}

func formatDebugBody(body []byte) string {
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

func truncateRunes(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit]) + "\n... (truncated)"
}

func formatDebugHeaders(h http.Header) string {
	if len(h) == 0 {
		return ""
	}
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(strings.Join(h[k], ", "))
	}
	return truncateRunes(b.String(), maxDebugRunes)
}

func storeValueMatches(actual any, expected string) bool {
	expTrim := strings.TrimSpace(expected)
	if actual == nil {
		return expTrim == "" || strings.EqualFold(expTrim, "null")
	}
	if str, ok := actual.(fmt.Stringer); ok {
		return str.String() == expected
	}
	switch v := actual.(type) {
	case string:
		return v == expected
	case []byte:
		return string(v) == expected
	}
	rv := reflect.ValueOf(actual)
	switch rv.Kind() {
	case reflect.Bool:
		if want, err := strconv.ParseBool(expTrim); err == nil {
			return rv.Bool() == want
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if want, err := strconv.ParseInt(expTrim, 10, 64); err == nil {
			return rv.Int() == want
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if want, err := strconv.ParseUint(expTrim, 10, 64); err == nil {
			return rv.Uint() == want
		}
	case reflect.Float32, reflect.Float64:
		if want, err := strconv.ParseFloat(expTrim, 64); err == nil {
			return rv.Float() == want
		}
	case reflect.Slice:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return string(rv.Bytes()) == expected
		}
	}
	if expTrim != "" && (strings.HasPrefix(expTrim, "{") || strings.HasPrefix(expTrim, "[")) {
		var wantJSON any
		if err := json.Unmarshal([]byte(expTrim), &wantJSON); err == nil {
			return compareJSON(actual, wantJSON, false)
		}
	}
	formatted := formatAny(actual)
	return formatted == expected || formatted == expTrim
}

func compareJSON(a, b any, ignoreOrder bool) bool {
	normA := normalizeJSON(a, ignoreOrder)
	normB := normalizeJSON(b, ignoreOrder)
	return reflect.DeepEqual(normA, normB)
}

func normalizeJSON(v any, ignoreOrder bool) any {
	switch val := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(val))
		for k, vv := range val {
			m[k] = normalizeJSON(vv, ignoreOrder)
		}
		return m
	case []any:
		arr := make([]any, len(val))
		for i, vv := range val {
			arr[i] = normalizeJSON(vv, ignoreOrder)
		}
		if ignoreOrder {
			sortSlice(arr)
		}
		return arr
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		arr := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			arr[i] = normalizeJSON(rv.Index(i).Interface(), ignoreOrder)
		}
		if ignoreOrder {
			sortSlice(arr)
		}
		return arr
	case reflect.Map:
		m := map[string]any{}
		iter := rv.MapRange()
		for iter.Next() {
			key := fmt.Sprint(iter.Key().Interface())
			m[key] = normalizeJSON(iter.Value().Interface(), ignoreOrder)
		}
		return m
	}
	return v
}

func sortSlice(arr []any) {
	sort.Slice(arr, func(i, j int) bool {
		ai, _ := json.Marshal(arr[i])
		aj, _ := json.Marshal(arr[j])
		return string(ai) < string(aj)
	})
}

func (w *world) renderPath(tpl string) string {
	if w == nil || w.ctx == nil {
		return tpl
	}
	return mustExec(tpl, w.templateCtx())
}

func (w *world) templateCtx() map[string]any {
	ctx := map[string]any{"store": w.ctx.Store}
	if w != nil && w.ctx != nil {
		if vars, ok := w.ctx.Store["vars"]; ok {
			ctx["vars"] = vars
		}
	}
	return ctx
}

func renderMap(in map[string]string, ctx map[string]any) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = mustExec(v, ctx)
	}
	return out
}

func mustExec(tpl string, ctx map[string]any) string {
	s, err := execTemplate(tpl, ctx) // execTemplate sudah ada di engine.go (package sama)
	if err != nil {
		return tpl
	}
	return s
}

func flattenEnvVars(src map[string]any) map[string]string {
	if len(src) == 0 {
		return nil
	}
	out := map[string]string{}
	for k, v := range src {
		flattenEnvVar(out, k, v)
	}
	return out
}

func flattenEnvVar(out map[string]string, prefix string, val any) {
	switch typed := val.(type) {
	case map[string]any:
		for k, v := range typed {
			flattenEnvVar(out, joinKey(prefix, k), v)
		}
	case map[interface{}]any:
		for rawKey, v := range typed {
			strKey, ok := rawKey.(string)
			if !ok {
				continue
			}
			flattenEnvVar(out, joinKey(prefix, strKey), v)
		}
	case []any:
		b, err := json.Marshal(typed)
		if err != nil {
			out[prefix] = fmt.Sprint(typed)
			return
		}
		out[prefix] = string(b)
	case nil:
		out[prefix] = ""
	default:
		out[prefix] = fmt.Sprint(typed)
	}
}

func joinKey(prefix, key string) string {
	if prefix == "" {
		return key
	}
	if key == "" {
		return prefix
	}
	return prefix + "." + key
}

func normalizeEnvVars(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	out := map[string]any{}
	for k, v := range src {
		out[k] = normalizeEnvValue(v)
	}
	return out
}

func normalizeEnvValue(val any) any {
	switch typed := val.(type) {
	case map[string]any:
		m := map[string]any{}
		for k, v := range typed {
			m[k] = normalizeEnvValue(v)
		}
		return m
	case map[interface{}]any:
		m := map[string]any{}
		for k, v := range typed {
			strKey, ok := k.(string)
			if !ok {
				continue
			}
			m[strKey] = normalizeEnvValue(v)
		}
		return m
	case []any:
		arr := make([]any, len(typed))
		for i, v := range typed {
			arr[i] = normalizeEnvValue(v)
		}
		return arr
	case nil:
		return ""
	default:
		return fmt.Sprint(typed)
	}
}

func setNestedStoreValue(store map[string]any, path []string, value any) {
	if len(path) == 0 {
		return
	}
	if len(path) == 1 {
		store[path[0]] = value
		return
	}
	curr := store
	for i := 0; i < len(path)-1; i++ {
		key := path[i]
		next, ok := curr[key]
		if !ok {
			child := map[string]any{}
			curr[key] = child
			curr = child
			continue
		}
		if child, ok := next.(map[string]any); ok {
			curr = child
			continue
		}
		child := map[string]any{}
		curr[key] = child
		curr = child
	}
	curr[path[len(path)-1]] = value
}

func deepCopyMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = deepCopyValue(v)
	}
	return out
}

func deepCopyValue(val any) any {
	switch typed := val.(type) {
	case map[string]any:
		return deepCopyMap(typed)
	case map[interface{}]any:
		m := map[string]any{}
		for k, v := range typed {
			strKey, ok := k.(string)
			if !ok {
				continue
			}
			m[strKey] = deepCopyValue(v)
		}
		return m
	case []any:
		arr := make([]any, len(typed))
		for i, v := range typed {
			arr[i] = deepCopyValue(v)
		}
		return arr
	default:
		return typed
	}
}
