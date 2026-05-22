package runner

import (
	"fmt"
	"regexp"

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
func interpolateString(s string, vars map[string]interface{}) (string, error) {
	// This regex matches both $var and ${var} syntax
	re := regexp.MustCompile(`\$\{?([a-zA-Z_][a-zA-Z0-9_]*)(?::([^}]*))?}?`)

	result := re.ReplaceAllStringFunc(s, func(match string) string {
		// Extract the variable name and default value
		varName := re.FindStringSubmatch(match)[1]
		defaultValue := re.FindStringSubmatch(match)[2]

		// Check if the variable exists
		if val, ok := vars[varName]; ok {
			return fmt.Sprintf("%v", val)
		}

		// If there's a default value, use it
		if defaultValue != "" {
			return defaultValue
		}

		// Otherwise, leave the original match (or could return an error)
		return match
	})

	// Check if there are any unmatched variables
	matches := re.FindAllStringSubmatch(result, -1)
	for _, match := range matches {
		varName := match[1]
		if _, ok := vars[varName]; !ok && match[2] == "" {
			return "", fmt.Errorf("undefined variable: %s", varName)
		}
	}

	return result, nil
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
