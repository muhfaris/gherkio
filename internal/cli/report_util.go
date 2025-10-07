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
