package runner

import (
	"testing"
)

func TestEvaluateCondition(t *testing.T) {
	vars := map[string]interface{}{
		"x":      123,
		"y":      "345",
		"flag":   true,
		"empty":  "",
		"nested": map[string]interface{}{
			"val": 42,
		},
		"arr": []interface{}{
			map[string]interface{}{"id": 10},
		},
	}

	tests := []struct {
		condition string
		expected  bool
		expectErr bool
	}{
		// Truthiness shorthand
		{"$flag", true, false},
		{"!$flag", false, false},
		{"$empty", false, false},
		{"!$empty", true, false},
		{"$undef", false, false},
		{"!$undef", true, false},

		// Equality
		{"$x == 123", true, false},
		{"$x == 345", false, false},
		{"$x != 345", true, false},
		{"$y == 345", true, false},
		{"$y == \"345\"", true, false},
		{"$y != 123", true, false},

		// Numeric comparisons
		{"$x > 100", true, false},
		{"$x >= 123", true, false},
		{"$x < 200", true, false},
		{"$x <= 123", true, false},
		{"$x > 150", false, false},

		// Nested paths
		{"$nested.val == 42", true, false},
		{"$arr[0].id == 10", true, false},

		// Nil / Null checks
		{"$undef == null", true, false},
		{"$x != null", true, false},

		// Negation with comparison (error cases)
		{"!$x == 123", false, true},
	}

	for _, tt := range tests {
		got, err := EvaluateCondition(tt.condition, vars)
		if (err != nil) != tt.expectErr {
			t.Errorf("EvaluateCondition(%q) error = %v, expectErr = %v", tt.condition, err, tt.expectErr)
			continue
		}
		if !tt.expectErr && got != tt.expected {
			t.Errorf("EvaluateCondition(%q) = %v, expected %v", tt.condition, got, tt.expected)
		}
	}
}
