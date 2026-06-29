package runner

import (
	"testing"
)

func TestResolveNestedVar_ArrayIndex(t *testing.T) {
	vars := map[string]interface{}{
		"issueTags": []interface{}{
			map[string]interface{}{"id": "tag_001", "name": "Bug"},
			map[string]interface{}{"id": "tag_002", "name": "Feature"},
			map[string]interface{}{"id": "tag_003", "name": "Support"},
		},
		"nested": map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"val": 10},
				map[string]interface{}{"val": 20},
			},
		},
		"flatArray": []interface{}{"a", "b", "c"},
	}

	tests := []struct {
		path     string
		expected interface{}
		found    bool
	}{
		{"issueTags[0].id", "tag_001", true},
		{"issueTags[1].id", "tag_002", true},
		{"issueTags[2].name", "Support", true},
		{"issueTags[5].id", nil, false},               // out of bounds
		{"nested.items[0].val", 10, true},
		{"nested.items[1].val", 20, true},
		{"nested.items[2].val", nil, false},            // out of bounds
		{"flatArray[0]", "a", true},
		{"flatArray[2]", "c", true},
		{"flatArray[9]", nil, false},                   // out of bounds
		{"missingTags[0].id", nil, false},              // missing field
	}

	for _, tt := range tests {
		val, found := resolveNestedVar(tt.path, vars)
		if found != tt.found {
			t.Errorf("resolveNestedVar(%q): found = %v, want %v", tt.path, found, tt.found)
			continue
		}
		if found && val != tt.expected {
			t.Errorf("resolveNestedVar(%q): val = %v, want %v", tt.path, val, tt.expected)
		}
	}
}

func TestResolveNestedVar_DottedPath_StillWorks(t *testing.T) {
	vars := map[string]interface{}{
		"accounts": map[string]interface{}{
			"budi": map[string]interface{}{
				"username": "budi@test.com",
				"roleId":   "admin_001",
			},
		},
	}

	val, found := resolveNestedVar("accounts.budi.username", vars)
	if !found || val != "budi@test.com" {
		t.Errorf("expected budi@test.com, got %v (found=%v)", val, found)
	}

	val, found = resolveNestedVar("accounts.budi.roleId", vars)
	if !found || val != "admin_001" {
		t.Errorf("expected admin_001, got %v (found=%v)", val, found)
	}
}

func TestInterpolateString_ArrayIndex(t *testing.T) {
	vars := map[string]interface{}{
		"tags": []interface{}{
			map[string]interface{}{"id": "t1"},
			map[string]interface{}{"id": "t2"},
			map[string]interface{}{"id": "t3"},
		},
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"$tags[0].id", "t1"},
		{"$tags[2].id", "t3"},
		{"prefix_${tags[1].id}_suffix", "prefix_t2_suffix"},
	}

	for _, tt := range tests {
		got, err := interpolateString(tt.input, vars)
		if err != nil {
			t.Errorf("interpolateString(%q): unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("interpolateString(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestInterpolateString_ArrayIndex_OutOfBounds(t *testing.T) {
	vars := map[string]interface{}{
		"tags": []interface{}{
			map[string]interface{}{"id": "t1"},
		},
	}

	// Out-of-bounds should leave the match unresolved (no error with default)
	got, err := interpolateString("${tags[5].id:fallback}", vars)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got != "fallback" {
		t.Errorf("expected fallback, got %q", got)
	}

// Out-of-bounds without default should error
	_, err = interpolateString("$tags[5].id", vars)
	if err == nil {
		t.Error("expected error for out-of-bounds access without default, got nil")
	}
}

func TestInterpolateString_EmbeddedGeneratorInBrackets(t *testing.T) {
	vars := map[string]interface{}{
		"tags": []interface{}{
			map[string]interface{}{"id": "tag_a"},
			map[string]interface{}{"id": "tag_b"},
			map[string]interface{}{"id": "tag_c"},
			map[string]interface{}{"id": "tag_d"},
			map[string]interface{}{"id": "tag_e"},
		},
	}

	// ${randomInt(0,4)} inside brackets should resolve first, then array-index works
	result, err := interpolateString("$tags[${randomInt(0,4)}].id", vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should resolve to one of the tags (not the raw array or literal string)
	expected := map[string]bool{"tag_a": true, "tag_b": true, "tag_c": true, "tag_d": true, "tag_e": true}
	if !expected[result] {
		t.Errorf("expected one of %v, got %q", expected, result)
	}

	// Same with full ${} wrapping
	result, err = interpolateString("${tags[${randomInt(0,4)}].id}", vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !expected[result] {
		t.Errorf("expected one of %v, got %q", expected, result)
	}

	// Test with different variable names to confirm pre-pass is name-agnostic
	vars2 := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"val": 10},
			map[string]interface{}{"val": 20},
			map[string]interface{}{"val": 30},
		},
		"users": []interface{}{
			map[string]interface{}{"email": "a@test.com"},
			map[string]interface{}{"email": "b@test.com"},
		},
	}

	result, err = interpolateString("$items[${randomInt(0,2)}].val", vars2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedVals := map[string]bool{"10": true, "20": true, "30": true}
	if !expectedVals[result] {
		t.Errorf("expected one of %v, got %q", expectedVals, result)
	}

	result, err = interpolateString("prefix_${users[${randomInt(0,1)}].email}_suffix", vars2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedEmails := map[string]bool{"prefix_a@test.com_suffix": true, "prefix_b@test.com_suffix": true}
	if !expectedEmails[result] {
		t.Errorf("expected one of %v, got %q", expectedEmails, result)
	}
}
