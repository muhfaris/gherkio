package runner

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/muhfaris/gherkio/internal/model"
)

// InterpolateRequest processes a request and replaces any variable references with values from the vars map.
func InterpolateRequest(req model.Request, vars map[string]interface{}) (model.Request, error) {
	// Create a copy of the request to avoid modifying the original
	interpolated := model.Request{
		Service: req.Service,
		Method:  req.Method,
		URL:     req.URL,
		Headers: make(map[string]string),
		Timeout: req.Timeout,
	}

	// Interpolate URL
	url, err := interpolateString(req.URL, vars)
	if err != nil {
		return model.Request{}, fmt.Errorf("failed to interpolate URL: %w", err)
	}
	interpolated.URL = url

	// Interpolate Headers
	interpolatedHeaders := make(map[string]string)
	for k, v := range req.Headers {
		interpolatedValue, err := interpolateString(v, vars)
		if err != nil {
			return model.Request{}, fmt.Errorf("failed to interpolate header '%s': %w", k, err)
		}
		interpolatedHeaders[k] = interpolatedValue
	}
	interpolated.Headers = interpolatedHeaders

	// Interpolate Body
	interpolatedBody, err := interpolateBody(req.Body, vars)
	if err != nil {
		return model.Request{}, fmt.Errorf("failed to interpolate body: %w", err)
	}
	interpolated.Body = interpolatedBody

	return interpolated, nil
}

// interpolateString replaces variable references in a string with values from the vars map.
// Supports:
//   - Simple vars: $var, ${var}
//   - Nested/dotted paths: $accounts.eka.username, ${accounts.eka.username}
//   - Default values: ${var:default}, ${accounts.eka.username:default}
func interpolateString(s string, vars map[string]interface{}) (string, error) {
	// This regex matches $var, ${var}, dotted paths like $accounts.eka.username,
	// and defaults like ${var:default}
	re := regexp.MustCompile(`\$\{?([a-zA-Z_][a-zA-Z0-9_]+(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*)(?::([^}]*))?}?`)

	result := re.ReplaceAllStringFunc(s, func(match string) string {
		submatch := re.FindStringSubmatch(match)
		varName := submatch[1]
		defaultValue := submatch[2]

		// Check if the variable exists (supports dotted paths)
		if val, ok := resolveNestedVar(varName, vars); ok {
			return fmt.Sprintf("%v", val)
		}

		// If there's a default value, use it
		if defaultValue != "" {
			return defaultValue
		}

		// Otherwise, leave the original match
		return match
	})

	// Check if there are any unmatched variables
	matches := re.FindAllStringSubmatch(result, -1)
	for _, match := range matches {
		varName := match[1]
		if _, ok := resolveNestedVar(varName, vars); !ok && match[2] == "" {
			return "", fmt.Errorf("undefined variable: %s", varName)
		}
	}

	return result, nil
}

// resolveNestedVar looks up a potentially dotted variable path in the vars map.
// For simple names like "username", it's equivalent to vars["username"].
// For dotted paths like "accounts.eka.username", it navigates the nested map structure.
func resolveNestedVar(path string, vars map[string]interface{}) (interface{}, bool) {
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

// interpolateBody recursively processes a body structure to replace variable references.
func interpolateBody(body interface{}, vars map[string]interface{}) (interface{}, error) {
	switch b := body.(type) {
	case string:
		return interpolateString(b, vars)
	case map[string]interface{}:
		interpolated := make(map[string]interface{})
		for k, v := range b {
			interpolatedValue, err := interpolateBody(v, vars)
			if err != nil {
				return nil, err
			}
			interpolated[k] = interpolatedValue
		}
		return interpolated, nil
	case []interface{}:
		interpolated := make([]interface{}, len(b))
		for i, v := range b {
			interpolatedValue, err := interpolateBody(v, vars)
			if err != nil {
				return nil, err
			}
			interpolated[i] = interpolatedValue
		}
		return interpolated, nil
	default:
		return b, nil
	}
}
