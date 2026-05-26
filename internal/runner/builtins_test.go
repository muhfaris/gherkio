package runner

import (
	"os"
	"testing"
)

func TestLoadGherkioEnvVars(t *testing.T) {
	// Set some mock variables
	os.Setenv("GHERKIO_TEST_VAR_1", "value1")
	os.Setenv("GHERKIO_TEST_VAR_2", "value2")
	os.Setenv("SOME_OTHER_VAR", "value3")
	defer func() {
		os.Unsetenv("GHERKIO_TEST_VAR_1")
		os.Unsetenv("GHERKIO_TEST_VAR_2")
		os.Unsetenv("SOME_OTHER_VAR")
	}()

	vars := LoadGherkioEnvVars()

	// Assertions
	if val, ok := vars["GHERKIO_TEST_VAR_1"]; !ok || val != "value1" {
		t.Errorf("Expected GHERKIO_TEST_VAR_1 to be 'value1', got '%v'", val)
	}

	if val, ok := vars["GHERKIO_TEST_VAR_2"]; !ok || val != "value2" {
		t.Errorf("Expected GHERKIO_TEST_VAR_2 to be 'value2', got '%v'", val)
	}

	if _, ok := vars["SOME_OTHER_VAR"]; ok {
		t.Errorf("Expected SOME_OTHER_VAR to be filtered out, but it was found")
	}
}

func TestInterpolateString_WithGherkioEnvVars(t *testing.T) {
	// Set mock GHERKIO_ prefixed variables
	os.Setenv("GHERKIO_API_URL", "https://api.test.com")
	os.Setenv("GHERKIO_API_KEY", "secret-key-xyz")
	os.Setenv("SOME_OTHER_VAR", "leaked")
	defer func() {
		os.Unsetenv("GHERKIO_API_URL")
		os.Unsetenv("GHERKIO_API_KEY")
		os.Unsetenv("SOME_OTHER_VAR")
	}()

	// Load variables into map
	vars := LoadGherkioEnvVars()

	// Test string interpolation
	interpolatedURL, err := interpolateString("URL: $GHERKIO_API_URL/login", vars)
	if err != nil {
		t.Fatalf("Unexpected error interpolating GHERKIO_API_URL: %v", err)
	}
	expectedURL := "URL: https://api.test.com/login"
	if interpolatedURL != expectedURL {
		t.Errorf("Expected URL '%s', got '%s'", expectedURL, interpolatedURL)
	}

	// Test nested/dotted access or simple interpolation with brackets
	interpolatedKey, err := interpolateString("Key: ${GHERKIO_API_KEY}", vars)
	if err != nil {
		t.Fatalf("Unexpected error interpolating GHERKIO_API_KEY: %v", err)
	}
	expectedKey := "Key: secret-key-xyz"
	if interpolatedKey != expectedKey {
		t.Errorf("Expected Key '%s', got '%s'", expectedKey, interpolatedKey)
	}

	// Test non-prefixed variables are undefined and will throw an error
	_, err = interpolateString("$SOME_OTHER_VAR", vars)
	if err == nil {
		t.Errorf("Expected error for non-prefixed host variable, but got none")
	}
}
