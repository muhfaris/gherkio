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

	// Apply Transform Projections
	if len(req.Transform) > 0 {
		var bodyMap map[string]interface{}
		if interpolated.Body == nil {
			bodyMap = make(map[string]interface{})
		} else if m, ok := interpolated.Body.(map[string]interface{}); ok {
			bodyMap = m
		} else {
			return model.Request{}, fmt.Errorf("cannot apply transform: request body is not an object")
		}

		for targetPath, projCfg := range req.Transform {
			projected, err := ProjectCollection(projCfg, vars)
			if err != nil {
				return model.Request{}, fmt.Errorf("transform failed at path %q: %w", targetPath, err)
			}
			writePath(bodyMap, targetPath, projected)
		}
		interpolated.Body = bodyMap
	}


	// Interpolate Multipart config
	if req.Multipart != nil {
		multipart, err := interpolateMultipart(req.Multipart, vars)
		if err != nil {
			return model.Request{}, fmt.Errorf("failed to interpolate multipart: %w", err)
		}
		interpolated.Multipart = multipart
	}

	return interpolated, nil
}

// interpolateMultipart processes multipart configuration to replace variable references.
func interpolateMultipart(mp *model.MultipartConfig, vars map[string]interface{}) (*model.MultipartConfig, error) {
	result := &model.MultipartConfig{
		Fields: make(map[string]string),
		Files:  make(map[string]model.MultipartItem),
	}

	// Interpolate form fields
	for k, v := range mp.Fields {
		interpolatedValue, err := interpolateString(v, vars)
		if err != nil {
			return nil, fmt.Errorf("failed to interpolate multipart field '%s': %w", k, err)
		}
		result.Fields[k] = interpolatedValue
	}

	// Interpolate file items
	for k, item := range mp.Files {
		interpolatedPath, err := interpolateString(item.Path, vars)
		if err != nil {
			return nil, fmt.Errorf("failed to interpolate multipart file path '%s': %w", k, err)
		}

		interpolatedFilename := item.Filename
		if item.Filename != "" {
			interpolatedFilename, err = interpolateString(item.Filename, vars)
			if err != nil {
				return nil, fmt.Errorf("failed to interpolate multipart file filename '%s': %w", k, err)
			}
		}

		result.Files[k] = model.MultipartItem{
			Path:        interpolatedPath,
			ContentType: item.ContentType,
			Filename:    interpolatedFilename,
		}
	}

	return result, nil
}

// interpolateString replaces variable references in a string with values from the vars map.
// Supports:
//   - Simple vars: $var, ${var}
//   - Nested/dotted paths: $accounts.eka.username, ${accounts.eka.username}
//   - Default values: ${var:default}, ${accounts.eka.username:default}
//   - Parametrized generators: ${randomInt(1,100)}, ${randomInt()}
func interpolateString(s string, vars map[string]interface{}) (string, error) {
	// This regex matches $var, ${var}, dotted paths like $accounts.eka.username,
	// defaults like ${var:default}, and parametrized generators like ${randomInt(1,100)}
	// Capture groups:
	//   1: variable/function name (e.g. randomInt, accounts.eka.username)
	//   2: arguments inside parens (e.g. 1,100) — optional
	//   3: default value after colon (e.g. 42 in ${var:42}) — optional
	re := regexp.MustCompile(`\$\{?([a-zA-Z_][a-zA-Z0-9_]+(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*)(?:\(([^)]*)\))?(?::([^}]*))?}?`)

	var evalErr error

	result := re.ReplaceAllStringFunc(s, func(match string) string {
		if evalErr != nil {
			return match
		}

		submatch := re.FindStringSubmatch(match)
		varName := submatch[1]
		args := submatch[2]
		defaultValue := submatch[3]

		hasParens := strings.Contains(match, "(") && strings.Contains(match, ")")

		// Check if this is a parametrized generator function call (has parens or arguments)
		if hasParens {
			funcs := GetGeneratorFuncs()
			if fn, ok := funcs[varName]; ok {
				val, err := fn(args)
				if err != nil {
					evalErr = fmt.Errorf("function '%s' failed: %w", varName, err)
					return match
				}
				return fmt.Sprintf("%v", val)
			}
		}

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

	if evalErr != nil {
		return "", evalErr
	}

	// Check if there are any unmatched variables
	matches := re.FindAllStringSubmatch(result, -1)
	for _, match := range matches {
		varName := match[1]
		defaultValue := match[3]

		hasParens := strings.Contains(match[0], "(") && strings.Contains(match[0], ")")

		// Skip parametrized generator functions (e.g. ${randomInt(1,100)})
		if hasParens {
			funcs := GetGeneratorFuncs()
			if _, ok := funcs[varName]; ok {
				continue
			}
		}

		// Skip if there's a default value
		if defaultValue != "" {
			continue
		}

		// Error if variable is not defined
		if _, ok := resolveNestedVar(varName, vars); !ok {
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
		return resolveTypePreserving(b, vars)
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
