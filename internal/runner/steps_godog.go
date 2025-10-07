package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/cucumber/godog"
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
			sc.Before(func(c context.Context, _ *godog.Scenario) (context.Context, error) {
				// reset per-scenario
				w.ctx = NewContext(env, cat)
				w.lastReq = Request{}
				w.lastRes = Response{}
				w.flows = flows
				// snapshot default env headers for this scenario
				w.savedEnvHeaders = map[string]string{}
				for k, v := range w.ctx.Env.Headers {
					w.savedEnvHeaders[k] = v
				}
				return c, nil
			})

			sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
				// restore env headers
				w.ctx.Env.Headers = map[string]string{}
				for k, v := range w.savedEnvHeaders {
					w.ctx.Env.Headers[k] = v
				}
				return ctx, nil
			})
		}

		// No-op (env & catalogs sudah diload di CLI)
		bind(sc, `^I load env [\"']([^\"']+)[\"']$`, "Load env", "", func(_ string) error { return nil })
		bind(sc, `^I load flows from [\"']([^\"']+)[\"']$`, "Load flows", "", func(_ string) error { return nil })

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
			w.lastReq.Query = tableToMap(table)
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
			res, err := Call(w.ctx, w.lastReq)
			w.lastDurMs = time.Since(t0).Milliseconds()
			w.lastRes = res
			return err
		})

		// Call API with body: """ {json} """
		bind(sc, `^I call API [\"']([^\"']+)[\"'] with body:$`, "Call API with docstring body", "I call API 'auth.login' with body:\n\"\"\"\n{\"username\": \"demo\"}\n\"\"\"", func(key string, body *godog.DocString) error {
			w.lastReq.APIKey = key
			w.lastReq.Body = []byte(body.Content)
			t0 := time.Now()
			res, err := Call(w.ctx, w.lastReq)
			w.lastDurMs = time.Since(t0).Milliseconds()
			w.lastRes = res
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
			res, err := Call(w.ctx, w.lastReq)
			w.lastDurMs = time.Since(t0).Milliseconds()
			w.lastRes = res
			return err
		})

		// I call API "<key>" using fixture "<path>"
		bind(sc, `^I call API [\"']([^\"']+)[\"'] using fixture [\"']([^\"']+)[\"']$`, "Call API using fixture file", "I call API 'users.create' using fixture 'fixtures/user.json'", func(key, fpath string) error {
			b, err := os.ReadFile(fpath)
			if err != nil {
				return err
			}
			if w.lastReq.Headers == nil {
				w.lastReq.Headers = map[string]string{}
			}
			if _, ok := w.lastReq.Headers["Content-Type"]; !ok {
				w.lastReq.Headers["Content-Type"] = "application/json"
			}
			w.lastReq.APIKey = key
			w.lastReq.Body = b
			t0 := time.Now()
			res, err := Call(w.ctx, w.lastReq)
			w.lastDurMs = time.Since(t0).Milliseconds()
			w.lastRes = res
			return err
		})

		// I run flow "<name>" with:
		bind(sc, `^I run flow [\"']([^\"']+)[\"'] with:$`, "Run flow with parameters", "I run flow 'login' with:\n| username | demo |\n| password | secret |", func(name string, table *godog.Table) error {
			args := tableToMap(table)
			return w.runFlow(name, args)
		})

		// I run flow "<name>"
		bind(sc, `^I run flow [\"']([^\"']+)[\"']$`, "Run flow without parameters", "I run flow 'auth'", func(name string) error {
			return w.runFlow(name, map[string]string{})
		})

		// Assertions
		bind(sc, `^response status should be (\\d+)$`, "Assert response status code", "response status should be 200", func(code int) error {
			if w.lastRes.Status != code {
				return fmt.Errorf("expect %d got %d", code, w.lastRes.Status)
			}
			return nil
		})

		// response status should be in 200-299 (generic range)
		bind(sc, `^response status should be in (\\d{3})-(\\d{3})$`, "Assert response status in range", "response status should be in 200-299", func(lo, hi int) error {
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
		bind(sc, `^response time should be (==|!=|>=|>|<=|<) (\\d+)ms$`, "Assert response time", "response time should be <= 500ms", func(op string, want int64) error {
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

		// json "<path>" should exist
		bind(sc, `^json [\"']([^\"']+)[\"'] should exist$`, "Assert JSON path exists", "json '$.data.id' should exist", func(path string) error {
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
		bind(sc, `^save [\"']([^\"']+)[\"'] as [\"']([^\"']+)[\"']$`, "Save JSONPath to store", "save '$.access_token' as 'token'", func(path, key string) error {
			v := getJSONPath(w.lastRes.Body, path)
			if !v.Exists() {
				return fmt.Errorf("json path %q not found", path)
			}
			w.ctx.Store[key] = v.Value()
			return nil
		})

		// json "<path>" should equal "<value>"
		bind(sc, `^json [\"']([^\"']+)[\"'] should equal [\"']([^\"']+)[\"']$`, "Assert JSON value equals", "json '$.status' should equal 'success'", func(path, want string) error {
			val := getJSONPath(w.lastRes.Body, path)
			if !val.Exists() {
				return fmt.Errorf("json path %q not found", path)
			}
			if val.Str != want && val.Raw != want {
				return fmt.Errorf("expect %s got %s", want, val.Raw)
			}
			return nil
		})

		// json "<path>" should not be empty
		bind(sc, `^json [\"']([^\"']+)[\"'] should not be empty$`, "Assert JSON path is not empty", "json '$.data' should not be empty", func(path string) error {
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

		// json "<path>" should not exist
		bind(sc, `^json [\"']([^\"']+)[\"'] should not exist$`, "Assert JSON path does not exist", "json '$.error' should not exist", func(path string) error {
			v := getJSONPath(w.lastRes.Body, path)
			if v.Exists() {
				return fmt.Errorf("json path %q exists (value: %s)", path, v.Raw)
			}
			return nil
		})

		// json "<path>" should match "<regex>"
		bind(sc, `^json [\"']([^\"']+)[\"'] should match [\"']([^\"']+)[\"']$`, "Assert JSON value matches regex", "json '$.email' should match '[a-z]+@example.com'", func(path, rx string) error {
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
		bind(sc, `^json [\"']([^\"']+)[\"'] should be (==|!=|>=|>|<=|<) ([\\d\\.]+)$`, "Assert JSON numeric value", "json '$.age' should be >= 18", func(path, op, wantStr string) error {
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
		bind(sc, `^json [\"']([^\"']+)[\"'] length should be (==|!=|>=|>|<=|<) (\\d+)$`, "Assert JSON length", "json '$.items' length should be == 5", func(path, op string, n int) error {
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
		bind(sc, `^set [\"']([^\"']+)[\"'] to [\"']([^\"']+)[\"']$`, "Set value in store", "set 'base_url' to 'https://api.example.com'", func(k, v string) error {
			w.ctx.Store[k] = v
			return nil
		})

		// json "<path>" should be one of:
		bind(sc, `^json [\"']([^\"']+)[\"'] should be one of:$`, "Assert JSON value in list", "json '$.status' should be one of:\nactive\ninactive", func(path string, ds *godog.DocString) error {
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
		bind(sc, `^I wait (\\d+)(ms|s)$`, "Wait for duration", "I wait 100ms", func(n int, unit string) error {
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
			fmt.Printf("\n--- response ---\nstatus: %d\n", w.lastRes.Status)

			if len(w.lastRes.Header) > 0 {
				fmt.Println("headers:")
				for k, vals := range w.lastRes.Header {
					fmt.Printf("  %s: %s\n", k, strings.Join(vals, ", "))
				}
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
	// context for templating
	ctxMap := map[string]any{"store": w.ctx.Store}
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
		if st.Body != "" {
			req.Body = []byte(mustExec(st.Body, ctxMap))
		}
		// default Content-Type if body present
		if len(req.Body) > 0 {
			if req.Headers == nil {
				req.Headers = map[string]string{}
			}
			if _, ok := req.Headers["Content-Type"]; !ok {
				req.Headers["Content-Type"] = "application/json"
			}
		}
		res, err := Call(w.ctx, req)
		w.lastReq, w.lastRes = req, res
		if err != nil {
			return fmt.Errorf("flow %s step %d (%s): %w", name, i+1, st.Call, err)
		}
		if st.Expect != nil && st.Expect.Status != 0 && res.Status != st.Expect.Status {
			return fmt.Errorf("flow %s step %d (%s): expect %d got %d", name, i+1, st.Call, st.Expect.Status, res.Status)
		}
		// saves
		for jp, key := range st.Save {
			v := gjson.GetBytes(res.Body, jp)
			if !v.Exists() {
				return fmt.Errorf("flow %s step %d: jsonpath %s not found", name, i+1, jp)
			}
			w.ctx.Store[key] = v.Value()
		}
	}
	return nil
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
