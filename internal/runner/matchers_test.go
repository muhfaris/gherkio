package runner

import (
	"testing"
)

func TestEvaluateMatcher_TypeMatchers(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		actual   interface{}
		wantPass bool
		wantUsed bool
	}{
		// uuid
		{"uuid valid", "uuid", "550e8400-e29b-41d4-a716-446655440000", true, true},
		{"uuid invalid format", "uuid", "not-a-uuid", false, true},
		{"uuid not string", "uuid", 42, false, true},

		// email
		{"email valid", "email", "user@example.com", true, true},
		{"email no domain", "email", "user@.com", false, true}, // regex allows it
		{"email no at-sign", "email", "invalid", false, true},
		{"email not string", "email", 123, false, true},

		// datetime
		{"datetime RFC3339", "datetime", "2026-05-21T12:00:00Z", true, true},
		{"datetime with tz", "datetime", "2026-05-21T12:00:00+07:00", true, true},
		{"datetime invalid", "datetime", "2026-05-21", false, true},
		{"datetime not string", "datetime", true, false, true},

		// number
		{"number int", "number", 42, true, true},
		{"number float", "number", 3.14, true, true},
		{"number string", "number", "42", false, true},

		// string
		{"string valid", "string", "hello", true, true},
		{"string not string", "string", 42, false, true},

		// boolean
		{"boolean true", "boolean", true, true, true},
		{"boolean false", "boolean", false, true, true},
		{"boolean string", "boolean", "true", false, true},

		// array
		{"array valid", "array", []interface{}{1, 2, 3}, true, true},
		{"array empty", "array", []interface{}{}, true, true},
		{"array not array", "array", "string", false, true},

		// object
		{"object valid", "object", map[string]interface{}{"a": 1}, true, true},
		{"object empty", "object", map[string]interface{}{}, true, true},
		{"object not object", "object", "string", false, true},

		// null
		{"null valid", "null", nil, true, true},
		{"null not null", "null", "string", false, true},

		// true
		{"true valid", "true", true, true, true},
		{"true false bool", "true", false, false, true},
		{"true not bool", "true", "true", false, true},

		// false
		{"false valid", "false", false, true, true},
		{"false true bool", "false", true, false, true},
		{"false not bool", "false", "false", false, true},

		// exists (handled before keyword switch)
		{"exists present", "exists", "anything", true, true},
		{"exists nil", "exists", nil, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, used := evaluateMatcher("", tt.expected, tt.actual)
			if used != tt.wantUsed {
				t.Errorf("evaluateMatcher() used = %v, wantUsed %v", used, tt.wantUsed)
			}
			if result.Passed != tt.wantPass {
				t.Errorf("evaluateMatcher() passed = %v, wantPass %v\n  result: %+v", result.Passed, tt.wantPass, result)
			}
		})
	}
}

func TestEvaluateMatcher_StringMatchers(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		actual   interface{}
		wantPass bool
		wantUsed bool
	}{
		// contains
		{"contains matching", "contains Laptop", "Laptop Baru", true, true},
		{"contains not matching", "contains Laptop", "Smartphone", false, true},
		{"contains case-sensitive", "contains laptop", "Laptop Baru", false, true},
		{"contains non-string", "contains Laptop", 42, false, true},
		{"contains no arg", "contains", "anything", false, false},

		// startsWith
		{"startsWith matching", "startsWith item-", "item-42", true, true},
		{"startsWith not matching", "startsWith item-", "42-item", false, true},
		{"startsWith case-sensitive", "startsWith Item", "item-42", false, true},
		{"startsWith non-string", "startsWith item-", 42, false, true},
		{"startsWith no arg", "startsWith", "anything", false, false},

		// endsWith
		{"endsWith matching", "endsWith ed", "completed", true, true},
		{"endsWith not matching", "endsWith ed", "complete", false, true},
		{"endsWith case-sensitive", "endsWith ED", "completed", false, true},
		{"endsWith non-string", "endsWith ed", 42, false, true},
		{"endsWith no arg", "endsWith", "anything", false, false},

		// regex
		{"regex matching", "regex ^[A-Z]{3}$", "ABC", true, true},
		{"regex not matching", "regex ^[A-Z]{3}$", "abcd", false, true},
		{"regex invalid pattern", "regex [invalid", "anything", false, true},
		{"regex no arg", "regex", "anything", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, used := evaluateMatcher("", tt.expected, tt.actual)
			if used != tt.wantUsed {
				t.Errorf("evaluateMatcher() used = %v, wantUsed %v", used, tt.wantUsed)
			}
			if result.Passed != tt.wantPass {
				t.Errorf("evaluateMatcher() passed = %v, wantPass %v\n  result: %+v", result.Passed, tt.wantPass, result)
			}
		})
	}
}

func TestEvaluateMatcher_Fallback(t *testing.T) {
	// Unknown keyword should fall back (used = false)
	result, used := evaluateMatcher("body.name", "unknown_matcher", "anything")
	if used {
		t.Error("evaluateMatcher() should return used=false for unknown keyword")
	}
	if result.Passed {
		t.Error("evaluateMatcher() should return zero-value for unknown keyword")
	}
}

func TestFormatActual(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		output string
	}{
		{"string", "hello", `"hello"`},
		{"int", 42, "42"},
		{"float", 3.14, "3.14"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"nil", nil, "null"},
		{"array", []interface{}{1, 2}, "[1 2]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatActual(tt.input)
			if got != tt.output {
				t.Errorf("formatActual(%v) = %q, want %q", tt.input, got, tt.output)
			}
		})
	}
}

func TestIsMatcherKeyword(t *testing.T) {
	tests := []struct {
		input  string
		want   bool
	}{
		{"exists", true},
		{"uuid", true},
		{"email", true},
		{"datetime", true},
		{"number", true},
		{"string", true},
		{"boolean", true},
		{"array", true},
		{"object", true},
		{"null", true},
		{"true", true},
		{"false", true},
		{"contains something", true},
		{"startsWith prefix", true},
		{"endsWith suffix", true},
		{"regex pattern", true},
		{"contains", false},      // missing argument
		{"startsWith", false},    // missing argument
		{"endsWith", false},      // missing argument
		{"regex", false},         // missing argument
		{"unknown", false},
		{"", false},
		{"foo bar", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isMatcherKeyword(tt.input)
			if got != tt.want {
				t.Errorf("isMatcherKeyword(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
