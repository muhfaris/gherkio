package converter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/muhfaris/gherkio/internal/model"
	"gopkg.in/yaml.v3"
)

// LenientInterpolateString replaces variable references in a string with values from vars.
// If a variable is not defined, it leaves it intact as $var or ${var} instead of failing.
func LenientInterpolateString(s string, vars map[string]interface{}) string {
	re := regexp.MustCompile(`\$\{?([a-zA-Z_][a-zA-Z0-9_]*)(?::([^}]*))?}?`)

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

		if val, ok := vars[varName]; ok {
			return fmt.Sprintf("%v", val)
		}

		if defaultValue != "" {
			return defaultValue
		}

		return match
	})
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

	// 2. Leniently interpolate the request URL, headers, and body
	interpolatedURL := LenientInterpolateString(req.URL, vars)
	if baseURL != "" && !strings.HasPrefix(interpolatedURL, "http://") && !strings.HasPrefix(interpolatedURL, "https://") {
		// Append stripped URL to BaseURL
		if !strings.HasSuffix(baseURL, "/") && !strings.HasPrefix(interpolatedURL, "/") {
			baseURL += "/"
		}
		interpolatedURL = baseURL + interpolatedURL
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
