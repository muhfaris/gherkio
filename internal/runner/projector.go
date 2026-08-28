package runner

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/muhfaris/gherkio/internal/model"
)

// ProjectCollection evaluates a declarative collection projection.
func ProjectCollection(cfg *model.ProjectionConfig, vars map[string]interface{}) ([]interface{}, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil projection config")
	}

	varName := cfg.From
	if strings.HasPrefix(varName, "$") {
		varName = varName[1:]
	}

	sourceVal, found := resolveNestedVar(varName, vars)
	if !found {
		return []interface{}{}, nil
	}

	sourceArray, ok := sourceVal.([]interface{})
	if !ok {
		return nil, fmt.Errorf("source collection %q is not an array, got %T", cfg.From, sourceVal)
	}

	alias := cfg.As
	if alias == "" {
		alias = "item"
	}

	var filteredItems []interface{}
	for _, item := range sourceArray {
		localVars := make(map[string]interface{})
		for k, v := range vars {
			localVars[k] = v
		}
		localVars[alias] = item

		passed := true
		for whereKey, expectedVal := range cfg.Where {
			relPath := getRemainingPath(whereKey, alias)

			var actualVal interface{}
			var pathFound bool
			if relPath == "" {
				actualVal = item
				pathFound = true
			} else {
				actualVal, pathFound = resolvePath(item, relPath)
			}

			if !pathFound {
				passed = false
				break
			}

			if expectedStr, ok := expectedVal.(string); ok {
				expectedStr = normalizeMatcherString(expectedStr)
				if res, used := evaluateMatcher(relPath, expectedStr, actualVal); used {
					if !res.Passed {
						passed = false
						break
					}
				} else {
					if fmt.Sprintf("%v", actualVal) != expectedStr {
						passed = false
						break
					}
				}
			} else {
				if actualVal != expectedVal {
					// Handle float64 vs int comparison
					if actualFloat, ok1 := actualVal.(float64); ok1 {
						if expectedFloat, ok2 := toFloat64(expectedVal); ok2 {
							if actualFloat == expectedFloat {
								continue
							}
						}
					}
					passed = false
					break
				}
			}
		}

		if passed {
			filteredItems = append(filteredItems, item)
		}
	}

	if cfg.Limit > 0 && len(filteredItems) > cfg.Limit {
		filteredItems = filteredItems[:cfg.Limit]
	}

	var projectedItems []interface{}
	for _, item := range filteredItems {
		localVars := make(map[string]interface{})
		for k, v := range vars {
			localVars[k] = v
		}
		localVars[alias] = item

		projectedItem, err := evaluateSelectValue(cfg.Select, localVars)
		if err != nil {
			return nil, err
		}
		projectedItems = append(projectedItems, projectedItem)
	}

	return projectedItems, nil
}

// mapToProjectionConfig converts a map representation of a projection config into a model.ProjectionConfig.
func mapToProjectionConfig(m map[string]interface{}) (*model.ProjectionConfig, bool) {
	fromVal, hasFrom := m["from"]
	selectVal, hasSelect := m["select"]
	if !hasFrom || !hasSelect {
		return nil, false
	}

	fromStr, ok1 := fromVal.(string)
	selectMap, ok2 := selectVal.(map[string]interface{})
	if !ok1 || !ok2 {
		return nil, false
	}

	cfg := &model.ProjectionConfig{
		From:   fromStr,
		Select: selectMap,
	}

	if asVal, ok := m["as"].(string); ok {
		cfg.As = asVal
	}
	if limitVal, ok := m["limit"].(int); ok {
		cfg.Limit = limitVal
	} else if limitFloat, ok := m["limit"].(float64); ok {
		cfg.Limit = int(limitFloat)
	}
	if whereVal, ok := m["where"].(map[string]interface{}); ok {
		cfg.Where = whereVal
	}

	return cfg, true
}

// evaluateSelectValue recursively processes select block values.
func evaluateSelectValue(val interface{}, localVars map[string]interface{}) (interface{}, error) {
	if val == nil {
		return nil, nil
	}

	switch v := val.(type) {
	case map[string]interface{}:
		if subCfg, ok := mapToProjectionConfig(v); ok {
			return ProjectCollection(subCfg, localVars)
		}

		result := make(map[string]interface{})
		for k, valItem := range v {
			res, err := evaluateSelectValue(valItem, localVars)
			if err != nil {
				return nil, err
			}
			result[k] = res
		}
		return result, nil

	case []interface{}:
		result := make([]interface{}, len(v))
		for i, valItem := range v {
			res, err := evaluateSelectValue(valItem, localVars)
			if err != nil {
				return nil, err
			}
			result[i] = res
		}
		return result, nil

	default:
		return resolveTypePreserving(v, localVars)
	}
}

// resolveTypePreserving resolves simple variable paths while preserving their types.
func resolveTypePreserving(val interface{}, localVars map[string]interface{}) (interface{}, error) {
	strVal, ok := val.(string)
	if !ok {
		return val, nil
	}

	trimmed := strings.TrimSpace(strVal)
	if matches := randomItemPattern.FindStringSubmatch(trimmed); len(matches) == 2 && matches[0] == trimmed {
		return selectRandomItem(matches[1], localVars)
	}

	// Check for type casting helpers
	if strings.HasPrefix(trimmed, "$string(") && strings.HasSuffix(trimmed, ")") {
		inner := trimmed[len("$string(") : len(trimmed)-1]
		inner = strings.TrimPrefix(strings.TrimSpace(inner), "$")
		if res, ok := resolveNestedVar(inner, localVars); ok {
			if res == nil {
				return nil, nil
			}
			return fmt.Sprintf("%v", res), nil
		}
		// Fallback for missing property: return nil (JSON null) if parent alias exists
		parts := strings.Split(inner, ".")
		if len(parts) > 0 {
			firstPart := parts[0]
			if _, hasAlias := localVars[firstPart]; hasAlias {
				return nil, nil
			}
		}
		return "", fmt.Errorf("undefined variable for casting: %s", inner)
	}

	if strings.HasPrefix(trimmed, "$int(") && strings.HasSuffix(trimmed, ")") {
		inner := trimmed[len("$int(") : len(trimmed)-1]
		inner = strings.TrimPrefix(strings.TrimSpace(inner), "$")
		if res, ok := resolveNestedVar(inner, localVars); ok {
			if res == nil {
				return nil, nil
			}
			switch v := res.(type) {
			case int:
				return v, nil
			case int64:
				return int(v), nil
			case float64:
				return int(v), nil
			case float32:
				return int(v), nil
			case json.Number:
				if i, err := v.Int64(); err == nil {
					return int(i), nil
				}
				if f, err := v.Float64(); err == nil {
					return int(f), nil
				}
				return nil, fmt.Errorf("failed to cast json.Number %q to int", v)
			case string:
				parsed, err := strconv.Atoi(strings.TrimSpace(v))
				if err != nil {
					// Fallback: try parsing as float then converting to int
					fParsed, fErr := strconv.ParseFloat(strings.TrimSpace(v), 64)
					if fErr != nil {
						return nil, fmt.Errorf("failed to cast string %q to int: %w", v, err)
					}
					return int(fParsed), nil
				}
				return parsed, nil
			default:
				return nil, fmt.Errorf("unsupported type for int casting: %T", res)
			}
		}
		parts := strings.Split(inner, ".")
		if len(parts) > 0 {
			firstPart := parts[0]
			if _, hasAlias := localVars[firstPart]; hasAlias {
				return nil, nil
			}
		}
		return "", fmt.Errorf("undefined variable for casting: %s", inner)
	}

	if strings.HasPrefix(trimmed, "$float(") && strings.HasSuffix(trimmed, ")") {
		inner := trimmed[len("$float(") : len(trimmed)-1]
		inner = strings.TrimPrefix(strings.TrimSpace(inner), "$")
		if res, ok := resolveNestedVar(inner, localVars); ok {
			if res == nil {
				return nil, nil
			}
			switch v := res.(type) {
			case float64:
				return v, nil
			case float32:
				return float64(v), nil
			case int:
				return float64(v), nil
			case int64:
				return float64(v), nil
			case json.Number:
				if f, err := v.Float64(); err == nil {
					return f, nil
				}
				return nil, fmt.Errorf("failed to cast json.Number %q to float", v)
			case string:
				parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
				if err != nil {
					return nil, fmt.Errorf("failed to cast string %q to float: %w", v, err)
				}
				return parsed, nil
			default:
				return nil, fmt.Errorf("unsupported type for float casting: %T", res)
			}
		}
		parts := strings.Split(inner, ".")
		if len(parts) > 0 {
			firstPart := parts[0]
			if _, hasAlias := localVars[firstPart]; hasAlias {
				return nil, nil
			}
		}
		return "", fmt.Errorf("undefined variable for casting: %s", inner)
	}

	if strings.HasPrefix(trimmed, "$bool(") && strings.HasSuffix(trimmed, ")") {
		inner := trimmed[len("$bool(") : len(trimmed)-1]
		inner = strings.TrimPrefix(strings.TrimSpace(inner), "$")
		if res, ok := resolveNestedVar(inner, localVars); ok {
			if res == nil {
				return false, nil
			}
			switch v := res.(type) {
			case bool:
				return v, nil
			case int:
				return v != 0, nil
			case int64:
				return v != 0, nil
			case float64:
				return v != 0.0, nil
			case json.Number:
				if i, err := v.Int64(); err == nil {
					return i != 0, nil
				}
				if f, err := v.Float64(); err == nil {
					return f != 0.0, nil
				}
				return false, fmt.Errorf("failed to cast json.Number %q to bool", v)
			case string:
				parsed, err := strconv.ParseBool(strings.TrimSpace(strings.ToLower(v)))
				if err != nil {
					return nil, fmt.Errorf("failed to cast string %q to bool: %w", v, err)
				}
				return parsed, nil
			default:
				return nil, fmt.Errorf("unsupported type for bool casting: %T", res)
			}
		}
		parts := strings.Split(inner, ".")
		if len(parts) > 0 {
			firstPart := parts[0]
			if _, hasAlias := localVars[firstPart]; hasAlias {
				return nil, nil
			}
		}
		return "", fmt.Errorf("undefined variable for casting: %s", inner)
	}

	// $if(condition, thenValue, elseValue) - conditional value selection
	// condition: variable path to check for truthiness (e.g. "q.is_answered")
	// thenValue: value expression to use if condition is truthy
	// elseValue: value expression to use if condition is falsy (optional, defaults to null)
	if strings.HasPrefix(trimmed, "$if(") && strings.HasSuffix(trimmed, ")") {
		inner := trimmed[len("$if(") : len(trimmed)-1]

		args := strings.SplitN(inner, ",", 3)
		if len(args) < 2 {
			return "", fmt.Errorf("$if requires at least 2 arguments: condition, thenValue[, elseValue]")
		}

		conditionPath := strings.TrimSpace(args[0])
		thenExpr := strings.TrimSpace(args[1])
		elseExpr := ""
		if len(args) >= 3 {
			elseExpr = strings.TrimSpace(args[2])
		}

		// Strip optional $ prefix from condition path
		conditionPath = strings.TrimPrefix(conditionPath, "$")

		conditionVal, found := resolveNestedVar(conditionPath, localVars)

		var conditionTrue bool
		if found {
			conditionTrue = isTruthy(conditionVal)
		}

		if conditionTrue {
			return resolveTypePreserving(thenExpr, localVars)
		} else if elseExpr != "" {
			return resolveTypePreserving(elseExpr, localVars)
		}
		// No else clause -> return nil (JSON null)
		return nil, nil
	}

	var varName string
	if strings.HasPrefix(trimmed, "${") && strings.HasSuffix(trimmed, "}") {
		varName = trimmed[2 : len(trimmed)-1]
	} else if strings.HasPrefix(trimmed, "$") && !strings.Contains(trimmed, " ") {
		varName = trimmed[1:]
	}

	if varName != "" {
		if !strings.Contains(varName, "(") && !strings.Contains(varName, ":") {
			if res, ok := resolveNestedVar(varName, localVars); ok {
				return normalizeJSONNumber(res), nil
			}
			// Missing target property handling: return nil (JSON null) if parent alias exists
			parts := strings.Split(varName, ".")
			if len(parts) > 0 {
				firstPart := parts[0]
				if _, hasAlias := localVars[firstPart]; hasAlias {
					return nil, nil
				}
			}
		}
	}

	return interpolateString(strVal, localVars)
}

// normalizeJSONNumber converts json.Number to a native Go numeric type.
// json.Number is a string underneath and doesn't implement json.Marshaler,
// so json.Marshal would serialize it as a quoted string (">"55691"") instead
// of a JSON number (55691). This function converts it to int64 or float64
// so the downstream JSON encoder produces the correct numeric representation.
func normalizeJSONNumber(v interface{}) interface{} {
	switch n := v.(type) {
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return i
		}
		if f, err := n.Float64(); err == nil {
			return f
		}
		return string(n) // fallback
	}
	return v
}

// getRemainingPath strips the alias prefix from a key path.
func getRemainingPath(key string, alias string) string {
	prefix1 := "$" + alias + "."
	prefix2 := alias + "."
	if strings.HasPrefix(key, prefix1) {
		return strings.TrimPrefix(key, prefix1)
	}
	if strings.HasPrefix(key, prefix2) {
		return strings.TrimPrefix(key, prefix2)
	}
	if key == "$"+alias || key == alias {
		return ""
	}
	return strings.TrimPrefix(key, "$")
}

// normalizeMatcherString converts "$gt(0)" style matchers to standard Gherkio matcher syntax.
func normalizeMatcherString(s string) string {
	if strings.HasPrefix(s, "$") {
		openParen := strings.Index(s, "(")
		closeParen := strings.LastIndex(s, ")")
		if openParen > 0 && closeParen > openParen {
			keyword := s[1:openParen]
			arg := s[openParen+1 : closeParen]
			return keyword + " " + arg
		}
	}
	return s
}

// isTruthy checks if a value is truthy for $if condition evaluation.
// Returns false for nil, false, zero, and empty string; true for everything else.
func isTruthy(val interface{}) bool {
	if val == nil {
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0.0
	case string:
		return v != ""
	default:
		return true
	}
}

// writePath writes a value to a nested path in a map.
func writePath(data map[string]interface{}, path string, val interface{}) {
	parts := strings.Split(path, ".")
	current := data

	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = val
			break
		}

		next, ok := current[part]
		if !ok || next == nil {
			nextMap := make(map[string]interface{})
			current[part] = nextMap
			current = nextMap
		} else if nextMap, ok := next.(map[string]interface{}); ok {
			current = nextMap
		} else {
			nextMap := make(map[string]interface{})
			current[part] = nextMap
			current = nextMap
		}
	}
}
