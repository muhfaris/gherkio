package runner

import (
	"testing"
)

func TestGetCanonicalPaths(t *testing.T) {
	paths := GetCanonicalPaths()

	expected := []string{"body", "headers", "jwt", "redis"}
	if len(paths) != len(expected) {
		t.Errorf("GetCanonicalPaths() returned %d items, want %d", len(paths), len(expected))
	}

	for i, exp := range expected {
		if paths[i] != exp {
			t.Errorf("GetCanonicalPaths()[%d] = %q, want %q", i, paths[i], exp)
		}
	}
}

func TestGetCollectionFunctions(t *testing.T) {
	funcs := GetCollectionFunctions()

	expected := []string{"count", "all"}
	if len(funcs) != len(expected) {
		t.Errorf("GetCollectionFunctions() returned %d items, want %d", len(funcs), len(expected))
	}

	for i, exp := range expected {
		if funcs[i] != exp {
			t.Errorf("GetCollectionFunctions()[%d] = %q, want %q", i, funcs[i], exp)
		}
	}
}

func TestGetBackoffStrategies(t *testing.T) {
	strategies := GetBackoffStrategies()

	expected := []string{"constant", "linear", "exponential"}
	if len(strategies) != len(expected) {
		t.Errorf("GetBackoffStrategies() returned %d items, want %d", len(strategies), len(expected))
	}

	for i, exp := range expected {
		if strategies[i] != exp {
			t.Errorf("GetBackoffStrategies()[%d] = %q, want %q", i, strategies[i], exp)
		}
	}
}

func TestGetStepRoles(t *testing.T) {
	roles := GetStepRoles()

	expected := []string{"setup", "steps", "teardown"}
	if len(roles) != len(expected) {
		t.Errorf("GetStepRoles() returned %d items, want %d", len(roles), len(expected))
	}

	for i, exp := range expected {
		if roles[i] != exp {
			t.Errorf("GetStepRoles()[%d] = %q, want %q", i, roles[i], exp)
		}
	}
}

func TestGetAvailableMatchers_ContainsAllMatchers(t *testing.T) {
	matchers := GetAvailableMatchers()

	// These should all be present as per RFC-15
	expectedMatchers := []string{
		"exists", "not exists",
		"uuid", "email", "datetime", "uri",
		"string", "number", "boolean", "array", "object", "null",
		"true", "false",
		"contains", "startsWith", "endsWith", "regex",
	}

	matcherMap := make(map[string]bool)
	for _, m := range matchers {
		matcherMap[m] = true
	}

	for _, exp := range expectedMatchers {
		if !matcherMap[exp] {
			t.Errorf("GetAvailableMatchers() missing matcher: %q", exp)
		}
	}
}

func TestIsMatcherKeyword_UsesSourceOfTruth(t *testing.T) {
	// Verify that isMatcherKeyword works with the updated implementation
	// This test ensures the refactored code maintains the same behavior
	tests := []struct {
		input string
		want  bool
	}{
		{"exists", true},
		{"not exists", true},
		{"uuid", true},
		{"email", true},
		{"datetime", true},
		{"uri", true},
		{"string", true},
		{"number", true},
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
		{"contains", false},   // missing argument
		{"startsWith", false}, // missing argument
		{"endsWith", false},   // missing argument
		{"regex", false},      // missing argument
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
