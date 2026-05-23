package runner

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// evaluateMatcher evaluates the expected matcher against the actual value.
// It returns the AssertionResult and a boolean indicating if a matcher was found and used.
func evaluateMatcher(path string, expected string, actual interface{}) (AssertionResult, bool) {
	if expected == "exists" {
		return AssertionResult{
			Path:     path,
			Expected: "exists",
			Actual:   fmt.Sprintf("%v", actual),
			Passed:   true,
		}, true
	}

	parts := strings.SplitN(expected, " ", 2)
	keyword := parts[0]

	switch keyword {
	case "uuid":
		actualStr := fmt.Sprintf("%v", actual)
		matched, _ := regexp.MatchString(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`, actualStr)
		reason := ""
		if !matched {
			if _, ok := actual.(string); !ok {
				reason = "value is not a string"
			} else {
				reason = "string does not match UUID format"
			}
		}
		return AssertionResult{
			Path:     path,
			Expected: "valid UUID format",
			Actual:   actualStr,
			Passed:   matched,
			Reason:   reason,
		}, true

	case "email":
		actualStr := fmt.Sprintf("%v", actual)
		matched, _ := regexp.MatchString(`^[^@\s]+@[^@\s]+\.[^@\s]+$`, actualStr)
		reason := ""
		if !matched {
			if _, ok := actual.(string); !ok {
				reason = "value is not a string"
			} else if !strings.Contains(actualStr, "@") {
				reason = "missing local-part before '@'"
			} else {
				reason = "string does not match email format"
			}
		}
		return AssertionResult{
			Path:     path,
			Expected: "valid email format",
			Actual:   fmt.Sprintf("%q", actualStr),
			Passed:   matched,
			Reason:   reason,
		}, true

	case "datetime":
		actualStr := fmt.Sprintf("%v", actual)
		_, errRFC3339 := time.Parse(time.RFC3339, actualStr)
		passed := errRFC3339 == nil
		reason := ""
		if !passed {
			if _, ok := actual.(string); !ok {
				reason = "value is not a string"
			} else {
				reason = "string does not match RFC3339/ISO8601 datetime format"
			}
		}
		return AssertionResult{
			Path:     path,
			Expected: "valid datetime format",
			Actual:   actualStr,
			Passed:   passed,
			Reason:   reason,
		}, true

	case "uri":
		actualStr := fmt.Sprintf("%v", actual)
		matched, _ := regexp.MatchString(`^[a-zA-Z][a-zA-Z0-9+.-]*://[^\s]*$`, actualStr)
		reason := ""
		if !matched {
			if _, ok := actual.(string); !ok {
				reason = "value is not a string"
			} else {
				reason = "string does not match URI format"
			}
		}
		return AssertionResult{
			Path:     path,
			Expected: "valid URI format",
			Actual:   actualStr,
			Passed:   matched,
			Reason:   reason,
		}, true

	case "number":
		passed := false
		reason := ""
		switch actual.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, json.Number:
			passed = true
		default:
			reason = "value is not a number"
		}
		return AssertionResult{
			Path:     path,
			Expected: "number",
			Actual:   fmt.Sprintf("%v", actual),
			Passed:   passed,
			Reason:   reason,
		}, true

	case "string":
		_, passed := actual.(string)
		reason := ""
		if !passed {
			reason = "value is not a string"
		}
		return AssertionResult{
			Path:     path,
			Expected: "string",
			Actual:   fmt.Sprintf("%v", actual),
			Passed:   passed,
			Reason:   reason,
		}, true

	case "boolean":
		_, passed := actual.(bool)
		reason := ""
		if !passed {
			reason = "string cannot be coerced to boolean"
		}
		return AssertionResult{
			Path:     path,
			Expected: "boolean (true/false)",
			Actual:   fmt.Sprintf("%v", actual),
			Passed:   passed,
			Reason:   reason,
		}, true

	case "array":
		_, passed := actual.([]interface{})
		reason := ""
		if !passed {
			reason = "value is not an array"
		}
		return AssertionResult{
			Path:     path,
			Expected: "array",
			Actual:   fmt.Sprintf("%v", actual),
			Passed:   passed,
			Reason:   reason,
		}, true

	case "object":
		_, passed := actual.(map[string]interface{})
		reason := ""
		if !passed {
			reason = "value is not an object"
		}
		return AssertionResult{
			Path:     path,
			Expected: "object",
			Actual:   fmt.Sprintf("%v", actual),
			Passed:   passed,
			Reason:   reason,
		}, true

	case "null":
		passed := actual == nil
		reason := ""
		if !passed {
			reason = "value is not null"
		}
		return AssertionResult{
			Path:     path,
			Expected: "null",
			Actual:   fmt.Sprintf("%v", actual),
			Passed:   passed,
			Reason:   reason,
		}, true

	case "true":
		b, ok := actual.(bool)
		passed := ok && b == true
		reason := ""
		if !passed {
			if !ok {
				reason = "value is not a boolean"
			} else {
				reason = "value is not true"
			}
		}
		return AssertionResult{
			Path:     path,
			Expected: "true",
			Actual:   fmt.Sprintf("%v", actual),
			Passed:   passed,
			Reason:   reason,
		}, true

	case "false":
		b, ok := actual.(bool)
		passed := ok && b == false
		reason := ""
		if !passed {
			if !ok {
				reason = "value is not a boolean"
			} else {
				reason = "value is not false"
			}
		}
		return AssertionResult{
			Path:     path,
			Expected: "false",
			Actual:   fmt.Sprintf("%v", actual),
			Passed:   passed,
			Reason:   reason,
		}, true

	case "contains":
		if len(parts) < 2 {
			return AssertionResult{}, false
		}
		target := parts[1]
		actualStr := fmt.Sprintf("%v", actual)
		passed := strings.Contains(actualStr, target)
		reason := ""
		if !passed {
			reason = "substring not found at any position"
		}
		return AssertionResult{
			Path:     path,
			Expected: fmt.Sprintf("contains substring %q", target),
			Actual:   fmt.Sprintf("%q", actualStr),
			Passed:   passed,
			Reason:   reason,
		}, true

	case "startsWith":
		if len(parts) < 2 {
			return AssertionResult{}, false
		}
		target := parts[1]
		actualStr := fmt.Sprintf("%v", actual)
		passed := strings.HasPrefix(actualStr, target)
		reason := ""
		if !passed {
			reason = "prefix not found"
		}
		return AssertionResult{
			Path:     path,
			Expected: fmt.Sprintf("startsWith %q", target),
			Actual:   fmt.Sprintf("%q", actualStr),
			Passed:   passed,
			Reason:   reason,
		}, true

	case "endsWith":
		if len(parts) < 2 {
			return AssertionResult{}, false
		}
		target := parts[1]
		actualStr := fmt.Sprintf("%v", actual)
		passed := strings.HasSuffix(actualStr, target)
		reason := ""
		if !passed {
			reason = "suffix not found"
		}
		return AssertionResult{
			Path:     path,
			Expected: fmt.Sprintf("endsWith %q", target),
			Actual:   fmt.Sprintf("%q", actualStr),
			Passed:   passed,
			Reason:   reason,
		}, true

	case "regex":
		if len(parts) < 2 {
			return AssertionResult{}, false
		}
		pattern := parts[1]
		actualStr := fmt.Sprintf("%v", actual)
		matched, err := regexp.MatchString(pattern, actualStr)
		passed := err == nil && matched
		reason := ""
		if err != nil {
			reason = fmt.Sprintf("invalid regex pattern: %v", err)
		} else if !passed {
			reason = "no match at position 0"
		}
		return AssertionResult{
			Path:     path,
			Expected: fmt.Sprintf("pattern %s", pattern),
			Actual:   fmt.Sprintf("%q", actualStr),
			Passed:   passed,
			Reason:   reason,
		}, true
	}

	return AssertionResult{}, false
}

// formatActual returns a string representation of the actual value suitable for output.
func formatActual(actual interface{}) string {
	switch v := actual.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// isMatcherKeyword fast checks if the expected string is formatted as a matcher
// GetAvailableMatchers returns the list of all supported assertion matchers.
// This is used by the schema generator to provide dynamic autocomplete.
func GetAvailableMatchers() []string {
	return []string{
		"exists", "not exists",
		"uuid", "email", "datetime", "uri",
		"string", "number", "boolean", "array", "object", "null",
		"true", "false",
		"contains", "startsWith", "endsWith", "regex",
	}
}

// isMatcherKeyword checks if the expected string is formatted as a matcher.
// It uses GetAvailableMatchers() and GetArgMatchers() as the source of truth
// to avoid code drift.
func isMatcherKeyword(expected string) bool {
	matchers := GetAvailableMatchers()
	argMatchers := GetArgMatchers()
	argMatcherSet := make(map[string]bool, len(argMatchers))
	for _, am := range argMatchers {
		argMatcherSet[am] = true
	}

	// First pass: exact match with available matchers
	for _, m := range matchers {
		if expected == m {
			// Arg matchers like "contains" need an argument — bare keyword alone is not valid
			if argMatcherSet[m] {
				return false
			}
			return true
		}
	}

	// Second pass: check for "<argMatcher> <value>" form
	// These appear as bare keywords in the matchers list for schema autocomplete,
	// but their full form "contains <value>" is needed to be a valid matcher usage.
	for am := range argMatcherSet {
		if strings.HasPrefix(expected, am+" ") {
			return true
		}
	}

	return false
}
