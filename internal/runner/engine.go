package runner

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
)

type Request struct {
	APIKey  string
	Path    map[string]string
	Query   map[string]string
	Headers map[string]string
	Body    []byte
}

type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

func Call(ctx *Context, req Request) (Response, *http.Request, error) {
	httpReq, err := buildHTTPRequest(ctx, req)
	if err != nil {
		return Response{}, nil, err
	}

	resp, err := ctx.HTTP.Do(httpReq)
	if err != nil {
		return Response{}, httpReq, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return Response{Status: resp.StatusCode, Header: resp.Header, Body: b}, httpReq, nil
}

func buildHTTPRequest(ctx *Context, req Request) (*http.Request, error) {
	def, ok := ctx.Cat.Endpoints[req.APIKey]
	if !ok {
		return nil, fmt.Errorf("unknown api key: %s", req.APIKey)
	}
	path, err := render(def.Path, req.Path, ctx.Store)
	if err != nil {
		return nil, err
	}
	url := strings.TrimRight(ctx.Env.BaseURL, "/") + path
	// TODO: query params

	httpReq, err := http.NewRequest(def.Method, url, bytes.NewReader(req.Body))
	if err != nil {
		return nil, err
	}
	for k, v := range ctx.Env.Headers {
		httpReq.Header.Set(k, v)
	}
	for k, v := range def.Headers {
		httpReq.Header.Set(k, v)
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	authName := def.Auth
	if authName == "" {
		authName = ctx.CurrentAuth
	}
	if authName != "" {
		applyAuth(ctx, httpReq, authName)
	}
	return httpReq, nil
}

func render(tpl string, pathParams map[string]string, store map[string]any) (string, error) {
	if tpl == "" {
		return "", errors.New("empty path template")
	}
	t, err := template.New("p").Funcs(templateFuncs()).Parse(tpl)
	if err != nil {
		return "", err
	}
	ctx := map[string]any{"store": store}
	for k, v := range pathParams {
		ctx[k] = v
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, ctx); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func applyAuth(ctx *Context, r *http.Request, profile string) {
	ap, ok := ctx.Cat.Auth[profile]
	if !ok {
		return
	}
	if ap.FromStore != "" {
		val, ok := ctx.Store[ap.FromStore]
		if !ok {
			return
		}
		tpl := ap.Template
		if tpl == "" {
			tpl = "{{ .value }}"
		}
		out, err := execTemplate(tpl, map[string]any{"value": val, "store": ctx.Store})
		if err == nil && ap.Header != "" {
			r.Header.Set(ap.Header, out)
		}
		return
	}
	if ap.UsernameEnv != "" || ap.PasswordEnv != "" {
		user := os.Getenv(ap.UsernameEnv)
		pass := os.Getenv(ap.PasswordEnv)
		b64 := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
		r.Header.Set("Authorization", "Basic "+b64)
	}
}

func execTemplate(tpl string, ctxs ...map[string]any) (string, error) {
	t, err := template.New("tpl").Funcs(templateFuncs()).Parse(tpl)
	if err != nil {
		return "", err
	}
	ctx := map[string]any{}
	for _, c := range ctxs {
		for k, v := range c {
			ctx[k] = v
		}
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, ctx); err != nil {
		return "", err
	}
	return buf.String(), nil
}

var (
	randSrc = rand.New(rand.NewSource(time.Now().UnixNano()))
	randMu  sync.Mutex
)

func templateFuncs() template.FuncMap {
	fns := sprig.FuncMap()
	fns["randomInt"] = randomInt
	fns["randomUnix"] = randomUnix
	return fns
}

func randomInt(min, max int) (int, error) {
	if max < min {
		return 0, fmt.Errorf("randomInt: max (%d) must be >= min (%d)", max, min)
	}
	randMu.Lock()
	defer randMu.Unlock()
	if max == min {
		return min, nil
	}
	rangeSize := max - min + 1
	if rangeSize <= 0 {
		return 0, fmt.Errorf("randomInt: invalid range [%d,%d]", min, max)
	}
	return randSrc.Intn(rangeSize) + min, nil
}

func randomUnix(args ...any) (int64, error) {
	switch len(args) {
	case 0:
		return time.Now().UnixNano(), nil
	case 2:
		start, err := anyToString(args[0])
		if err != nil {
			return 0, fmt.Errorf("randomUnix: %w", err)
		}
		end, err := anyToString(args[1])
		if err != nil {
			return 0, fmt.Errorf("randomUnix: %w", err)
		}
		from, err := parseTimestamp(start)
		if err != nil {
			return 0, fmt.Errorf("randomUnix: %w", err)
		}
		to, err := parseTimestamp(end)
		if err != nil {
			return 0, fmt.Errorf("randomUnix: %w", err)
		}
		if to.Before(from) {
			return 0, fmt.Errorf("randomUnix: end (%s) must be >= start (%s)", end, start)
		}
		randMu.Lock()
		defer randMu.Unlock()
		startUnix := from.Unix()
		endUnix := to.Unix()
		if startUnix == endUnix {
			return startUnix, nil
		}
		rangeSize := endUnix - startUnix + 1
		if rangeSize <= 0 {
			return 0, fmt.Errorf("randomUnix: invalid range [%d,%d]", startUnix, endUnix)
		}
		return startUnix + randSrc.Int63n(rangeSize), nil
	default:
		return 0, fmt.Errorf("randomUnix: expected 0 or 2 arguments, got %d", len(args))
	}
}

func parseTimestamp(v string) (time.Time, error) {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, trimmed); err == nil {
			return ts, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse %q as timestamp", v)
}

func anyToString(v any) (string, error) {
	switch t := v.(type) {
	case string:
		return t, nil
	case fmt.Stringer:
		return t.String(), nil
	}
	return "", fmt.Errorf("cannot use %T as timestamp", v)
}
