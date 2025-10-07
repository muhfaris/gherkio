package cli

import "strings"

func pathOrDefault(p, fallback string) string {
	if strings.TrimSpace(p) == "" {
		return fallback
	}
	return p
}

func statusLabel(code int) string {
	if code >= 200 && code < 300 {
		return "PASSED"
	}
	return "FAILED"
}

type htmlSpec struct {
	Path  string
	Debug bool
}

func normalizeHTMLPath(raw, fallback string) (string, bool) {
	debug := false
	path := strings.TrimSpace(raw)
	if path == "" {
		path = fallback
	}
	lower := strings.ToLower(path)
	markers := []string{"?debug", "|debug", ";debug"}
	for _, m := range markers {
		if idx := strings.Index(lower, m); idx != -1 {
			debug = true
			path = strings.TrimSpace(path[:idx])
			lower = strings.ToLower(path)
		}
	}
	if path == "" {
		path = fallback
	}
	return path, debug
}

func parseHTMLKind(kind string) (debug bool, ok bool) {
	switch strings.ToLower(kind) {
	case "html":
		return false, true
	case "html-debug", "html+debug":
		return true, true
	default:
		return false, false
	}
}
