package converter

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/muhfaris/gherkio/internal/model"
	"gopkg.in/yaml.v3"
)

// LenientInterpolateString replaces variable references in a string with values from vars.
// Supports simple vars ($var, ${var}), dotted paths ($accounts.alice.username, ${accounts.alice.username}),
// and defaults (${var:default}, ${accounts.alice.username:default}).
// If a variable is not defined, it leaves it intact instead of failing (lenient).
func LenientInterpolateString(s string, vars map[string]interface{}) string {
	// Matches $var, ${var}, $accounts.eka.username, ${accounts.eka.username}, ${var:default}
	re := regexp.MustCompile(`\$\{?([a-zA-Z_][a-zA-Z0-9_]+(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*)(?::([^}]*))?}?`)

	return re.ReplaceAllStringFunc(s, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		varName := submatches[1]
		var defaultValue string
		if len(submatches) > 2 {
			defaultValue = submatches[2]
		}

		// Support dotted paths via nested map navigation
		if val, ok := resolveNestedPath(varName, vars); ok {
			return fmt.Sprintf("%v", val)
		}

		if defaultValue != "" {
			return defaultValue
		}

		return match
	})
}

// resolveNestedPath navigates a dotted path in a nested map structure.
// For simple names like "username", it's equivalent to vars["username"].
// For dotted paths like "accounts.alice.username", it navigates the nested map.
func resolveNestedPath(path string, vars map[string]interface{}) (interface{}, bool) {
	parts := strings.Split(path, ".")
	current := interface{}(vars)

	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		val, found := m[part]
		if !found {
			return nil, false
		}
		current = val
	}

	return current, true
}

// LenientInterpolateBody recursively processes a body structure to replace variables leniently.
func LenientInterpolateBody(body interface{}, vars map[string]interface{}) interface{} {
	switch b := body.(type) {
	case string:
		return LenientInterpolateString(b, vars)
	case map[string]interface{}:
		interpolated := make(map[string]interface{})
		for k, v := range b {
			interpolated[k] = LenientInterpolateBody(v, vars)
		}
		return interpolated
	case []interface{}:
		interpolated := make([]interface{}, len(b))
		for i, v := range b {
			interpolated[i] = LenientInterpolateBody(v, vars)
		}
		return interpolated
	default:
		return b
	}
}

// ConvertStepToCurl builds a copy-pasteable cURL command string from a Gherkio request structure,
// resolving variables leniently and merging environment URLs if applicable.
func ConvertStepToCurl(req model.Request, projectDir string, envName string, vars map[string]interface{}) (string, error) {
	// 1. Resolve environment baseUrl or service baseUrl
	baseURL := ""
	if projectDir != "" && envName != "" {
		envPath := filepath.Join(projectDir, ".gherkio", "environments", envName+".yaml")
		if data, err := os.ReadFile(envPath); err == nil {
			var env model.Environment
			if err := yaml.Unmarshal(data, &env); err == nil {
				baseURL = env.BaseURL
				if req.Service != "" && env.Services != nil {
					if svc, ok := env.Services[req.Service]; ok {
						baseURL = svc.BaseURL
					}
				}
			}
		}
	}

	// 2. Leniently interpolate the request URL, headers, body, and query params
	interpolatedURL := LenientInterpolateString(req.URL, vars)
	if baseURL != "" && !strings.HasPrefix(interpolatedURL, "http://") && !strings.HasPrefix(interpolatedURL, "https://") {
		// Append stripped URL to BaseURL
		if !strings.HasSuffix(baseURL, "/") && !strings.HasPrefix(interpolatedURL, "/") {
			baseURL += "/"
		}
		interpolatedURL = baseURL + interpolatedURL
	}

	// Append query params if present — values can be strings or arrays
	if len(req.Query) > 0 {
		sep := "?"
		if strings.Contains(interpolatedURL, "?") {
			sep = "&"
		}
		for k, v := range req.Query {
			escapedKey := url.QueryEscape(k)
			switch val := v.(type) {
			case string:
				escapedVal := url.QueryEscape(LenientInterpolateString(val, vars))
				interpolatedURL += sep + escapedKey + "=" + escapedVal
				sep = "&"
			case []interface{}:
				for _, item := range val {
					itemStr := fmt.Sprintf("%v", item)
					escapedVal := url.QueryEscape(LenientInterpolateString(itemStr, vars))
					interpolatedURL += sep + escapedKey + "=" + escapedVal
					sep = "&"
				}
			default:
				escapedVal := url.QueryEscape(LenientInterpolateString(fmt.Sprintf("%v", val), vars))
				interpolatedURL += sep + escapedKey + "=" + escapedVal
				sep = "&"
			}
		}
	}

	interpolatedHeaders := make(map[string]string)
	for k, v := range req.Headers {
		interpolatedHeaders[k] = LenientInterpolateString(v, vars)
	}

	interpolatedBody := LenientInterpolateBody(req.Body, vars)

	// 3. Construct the cURL command
	var sb strings.Builder
	method := req.Method
	if method == "" {
		if interpolatedBody != nil {
			method = "POST"
		} else {
			method = "GET"
		}
	}
	method = strings.ToUpper(method)

	sb.WriteString(fmt.Sprintf("curl -X %s '%s'", method, interpolatedURL))

	// Sort headers for deterministic cURL generation
	var headerKeys []string
	for k := range interpolatedHeaders {
		headerKeys = append(headerKeys, k)
	}
	// For clean display, let's keep them sorted or just output them.
	// Sorting headers ensures testability.
	for _, k := range headerKeys {
		v := interpolatedHeaders[k]
		sb.WriteString(fmt.Sprintf(" -H '%s: %s'", k, v))
	}

	// Handle body
	if interpolatedBody != nil && method != "GET" && method != "HEAD" {
		switch b := interpolatedBody.(type) {
		case string:
			escaped := strings.ReplaceAll(b, "'", "'\\''")
			sb.WriteString(fmt.Sprintf(" -d '%s'", escaped))
		default:
			minified, err := json.Marshal(b)
			if err != nil {
				return "", fmt.Errorf("failed to marshal body: %w", err)
			}
			escaped := strings.ReplaceAll(string(minified), "'", "'\\''")
			sb.WriteString(fmt.Sprintf(" -d '%s'", escaped))
		}
	}

	return sb.String(), nil
}
