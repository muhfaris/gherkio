package runner

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/muhfaris/gherkio/internal/model"
)

var randomItemPattern = regexp.MustCompile(`\$\{?randomItem\(([^)]*)\)\}?`)
var trimAffixPattern = regexp.MustCompile(`\$\{?(trimPrefix|trimSuffix)\(([^)]*)\)\}?`)
var splitPattern = regexp.MustCompile(`\$\{?split\(([^)]*)\)\}?`)

// InterpolateRequest processes a request and replaces any variable references with values from the vars map.
func InterpolateRequest(req model.Request, vars map[string]interface{}) (model.Request, error) {
	// Create a copy of the request to avoid modifying the original
	interpolated := model.Request{
		Service: req.Service,
		Method:  req.Method,
		URL:     req.URL,
		Headers: make(map[string]string),
		Query:   make(map[string]any),
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

	// Interpolate Query — values can be strings or []interface{} (array)
	interpolatedQuery := make(map[string]any)
	for k, v := range req.Query {
		switch val := v.(type) {
		case string:
			interpolatedValue, err := interpolateString(val, vars)
			if err != nil {
				return model.Request{}, fmt.Errorf("failed to interpolate query '%s': %w", k, err)
			}
			interpolatedQuery[k] = interpolatedValue
		case []interface{}:
			interpolatedArr := make([]interface{}, len(val))
			for i, item := range val {
				itemStr := fmt.Sprintf("%v", item)
				interpolatedValue, err := interpolateString(itemStr, vars)
				if err != nil {
					return model.Request{}, fmt.Errorf("failed to interpolate query '%s[%d]': %w", k, i, err)
				}
				interpolatedArr[i] = interpolatedValue
			}
			interpolatedQuery[k] = interpolatedArr
		default:
			interpolatedQuery[k] = val
		}
	}
	interpolated.Query = interpolatedQuery

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
//   - Step-prefixed vars: $1-authToken, ${2-userId}
//   - Nested/dotted paths: $accounts.alice.username, ${accounts.alice.username}
//   - Array-index notation: $issueTags[0].id, ${issueTags[${randomInt(0,4)}].id}
//   - Default values: ${var:default}, ${accounts.alice.username:default}
//   - Parametrized generators: ${randomInt(1,100)}, ${randomInt()}
//   - Response-aware selection: ${randomItem(users,id)}
func interpolateString(s string, vars map[string]interface{}) (string, error) {
	// Resolve generators that depend on runtime variables before the ordinary
	// self-contained generator pass. The optional second argument is a field
	// path selected from the randomly chosen array element.
	var randomItemErr error
	s = randomItemPattern.ReplaceAllStringFunc(s, func(match string) string {
		if randomItemErr != nil {
			return match
		}
		submatch := randomItemPattern.FindStringSubmatch(match)
		value, err := selectRandomItem(submatch[1], vars)
		if err != nil {
			randomItemErr = err
			return match
		}
		return fmt.Sprintf("%v", value)
	})
	if randomItemErr != nil {
		return "", fmt.Errorf("function 'randomItem' failed: %w", randomItemErr)
	}

	var trimAffixErr error
	s = trimAffixPattern.ReplaceAllStringFunc(s, func(match string) string {
		if trimAffixErr != nil {
			return match
		}
		submatch := trimAffixPattern.FindStringSubmatch(match)
		value, err := applyTrimAffix(submatch[1], submatch[2], vars)
		if err != nil {
			trimAffixErr = err
			return match
		}
		return value
	})
	if trimAffixErr != nil {
		return "", trimAffixErr
	}

	var splitErr error
	s = splitPattern.ReplaceAllStringFunc(s, func(match string) string {
		if splitErr != nil {
			return match
		}
		submatch := splitPattern.FindStringSubmatch(match)
		value, err := applySplit(submatch[1], vars)
		if err != nil {
			splitErr = err
			return match
		}
		return value
	})
	if splitErr != nil {
		return "", splitErr
	}

	// Pre-pass: resolve ${func(args)} generator calls embedded anywhere in the string
	// (e.g. inside bracket notation like $issueTags[${randomInt(0,4)}].id).
	// These are self-contained (no variable dependencies), so resolving them first
	// lets the main pass handle the now-literal bracket indices.
	//
	// This regex matches ${funcName(args)} patterns only (must have parens).
	reFunc := regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)\(([^)]*)\)\}`)
	funcs := GetGeneratorFuncs()
	s = reFunc.ReplaceAllStringFunc(s, func(match string) string {
		submatch := reFunc.FindStringSubmatch(match)
		if len(submatch) < 3 {
			return match
		}
		funcName := submatch[1]
		args := submatch[2]
		if fn, ok := funcs[funcName]; ok {
			val, err := fn(args)
			if err != nil {
				return match // leave unresolved if fn fails
			}
			return fmt.Sprintf("%v", val)
		}
		return match
	})

	// Main pass: resolve variable references including dotted paths, array-index
	// bracket notation $issueTags[0].id, defaults, and parametrized generators.
	// Capture groups:
	//   1: variable/function name (e.g. randomInt, accounts.eka.username, issueTags[0].id)
	//   2: arguments inside parens (e.g. 1,100) — optional
	//   3: default value after colon (e.g. 42 in ${var:42}) — optional
	re := regexp.MustCompile(`\$\{?((?:[0-9]+-[a-zA-Z_][a-zA-Z0-9_-]*|[a-zA-Z_][a-zA-Z0-9_]*)(?:\.(?:[0-9]+-[a-zA-Z_][a-zA-Z0-9_-]*|[a-zA-Z_][a-zA-Z0-9_]*)|\[\d+\])*)(?:\(([^)]*)\))?(?::([^}]*))?}?`)

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

func applyTrimAffix(name, args string, vars map[string]interface{}) (string, error) {
	parts := strings.SplitN(args, ",", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("function '%s' failed: %s requires exactly 2 arguments (value,affix)", name, name)
	}
	value, err := resolveStringTransformArg(parts[0], vars)
	if err != nil {
		return "", fmt.Errorf("function '%s' failed: %w", name, err)
	}
	affix, err := resolveStringTransformArg(parts[1], vars)
	if err != nil {
		return "", fmt.Errorf("function '%s' failed: %w", name, err)
	}
	if name == "trimPrefix" {
		return strings.TrimPrefix(value, affix), nil
	}
	return strings.TrimSuffix(value, affix), nil
}

func applySplit(args string, vars map[string]interface{}) (string, error) {
	parts := strings.SplitN(args, ",", 3)
	if len(parts) != 3 {
		return "", fmt.Errorf("function 'split' failed: split requires exactly 3 arguments (value,delimiter,index)")
	}
	value, err := resolveStringTransformArg(parts[0], vars)
	if err != nil {
		return "", fmt.Errorf("function 'split' failed: %w", err)
	}
	delimiter, err := resolveStringTransformArg(parts[1], vars)
	if err != nil {
		return "", fmt.Errorf("function 'split' failed: %w", err)
	}
	if delimiter == "" {
		return "", fmt.Errorf("function 'split' failed: delimiter cannot be empty")
	}
	indexRaw, err := resolveStringTransformArg(parts[2], vars)
	if err != nil {
		indexRaw = strings.TrimSpace(parts[2])
	}
	index, err := strconv.Atoi(indexRaw)
	if err != nil || index < 0 {
		return "", fmt.Errorf("function 'split' failed: index must be a non-negative integer, got %q", indexRaw)
	}
	segments := strings.Split(value, delimiter)
	if index >= len(segments) {
		return "", fmt.Errorf("function 'split' failed: index %d out of range (split produced %d segments)", index, len(segments))
	}
	return segments[index], nil
}

func resolveStringTransformArg(raw string, vars map[string]interface{}) (string, error) {
	arg := strings.TrimSpace(raw)
	if len(arg) >= 2 && ((arg[0] == '"' && arg[len(arg)-1] == '"') ||
		(arg[0] == '\'' && arg[len(arg)-1] == '\'')) {
		return arg[1 : len(arg)-1], nil
	}

	path := strings.TrimPrefix(arg, "$")
	if strings.HasPrefix(path, "{") && strings.HasSuffix(path, "}") {
		path = path[1 : len(path)-1]
	}
	if value, ok := resolveNestedVar(path, vars); ok {
		return fmt.Sprintf("%v", value), nil
	}
	return "", fmt.Errorf("undefined variable: %s", path)
}

// resolveSetValue preserves the runtime type of an exact randomItem expression.
// Embedded expressions still use ordinary string interpolation so existing set
// behavior remains unchanged.
func resolveSetValue(raw string, vars map[string]interface{}) (interface{}, error) {
	trimmed := strings.TrimSpace(raw)
	if matches := randomItemPattern.FindStringSubmatch(trimmed); len(matches) == 2 && matches[0] == trimmed {
		return selectRandomItem(matches[1], vars)
	}
	return interpolateString(raw, vars)
}

func selectRandomItem(args string, vars map[string]interface{}) (interface{}, error) {
	parts := strings.SplitN(args, ",", 2)
	arrayPath := strings.Trim(strings.TrimSpace(parts[0]), "\"'")
	arrayPath = strings.TrimPrefix(arrayPath, "$")
	if arrayPath == "" {
		return nil, fmt.Errorf("randomItem requires an array variable")
	}
	value, found := resolveNestedVar(arrayPath, vars)
	if !found {
		return nil, fmt.Errorf("array variable %q is not defined", arrayPath)
	}
	items, ok := value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("%q is %T, expected array", arrayPath, value)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("array variable %q is empty", arrayPath)
	}
	selected := items[generateRandomIntInRange(0, len(items)-1)]
	if len(parts) == 1 {
		return normalizeJSONNumber(selected), nil
	}
	fieldPath := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
	fieldPath = strings.TrimPrefix(fieldPath, "$")
	if fieldPath == "" {
		return nil, fmt.Errorf("randomItem field path cannot be empty")
	}
	selectedMap, ok := selected.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("selected item is %T, cannot read field %q", selected, fieldPath)
	}
	fieldValue, found := resolveNestedVar(fieldPath, selectedMap)
	if !found {
		return nil, fmt.Errorf("field %q does not exist on selected item", fieldPath)
	}
	return normalizeJSONNumber(fieldValue), nil
}

// resolveNestedVar looks up a potentially dotted variable path in the vars map.
// For simple names like "username", it's equivalent to vars["username"].
// For dotted paths like "accounts.alice.username", it navigates the nested map structure.
// For array-index paths like "issueTags[0].id", it navigates into arrays by index.
func resolveNestedVar(path string, vars map[string]interface{}) (interface{}, bool) {
	parts := strings.Split(path, ".")
	current := interface{}(vars)

	for _, part := range parts {
		// Handle array index: "name[0]" or "field[3]"
		if idxStart := strings.Index(part, "["); idxStart >= 0 {
			fieldName := part[:idxStart]
			idxStr := part[idxStart+1 : len(part)-1]
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				return nil, false
			}

			// Navigate into the field if there's a name before the bracket
			if fieldName != "" {
				m, ok := current.(map[string]interface{})
				if !ok {
					return nil, false
				}
				val, found := m[fieldName]
				if !found {
					return nil, false
				}
				current = val
			}

			// Navigate into array by index
			arr, ok := current.([]interface{})
			if !ok || idx < 0 || idx >= len(arr) {
				return nil, false
			}
			current = arr[idx]
		} else {
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
