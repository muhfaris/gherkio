package runner

import (
	"os"
	"strings"
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

func TestParameterizedGenerators(t *testing.T) {
	vars := make(map[string]interface{})

	t.Run("Date and Time generators", func(t *testing.T) {
		// timestamp
		ts, err := interpolateString("$timestamp()", vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(ts) < 10 {
			t.Errorf("Expected timestamp to be at least 10 chars, got %q", ts)
		}

		// timestampMs
		tsMs, err := interpolateString("${timestampMs()}", vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(tsMs) < 13 {
			t.Errorf("Expected timestampMs to be at least 13 chars, got %q", tsMs)
		}

		// dateNow
		dNow, err := interpolateString("$dateNow()", vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(dNow) != 19 { // "2006-01-02 15:04:05" is 19 chars
			t.Errorf("Expected dateNow standard format length 19, got %q", dNow)
		}

		dCustom, err := interpolateString(`$dateNow("2006-01-02")`, vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(dCustom) != 10 {
			t.Errorf("Expected custom formatted date length 10, got %q", dCustom)
		}

		// dateOffset
		dOffsetDays, err := interpolateString(`$dateOffset("+14d", "2006-01-02")`, vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(dOffsetDays) != 10 {
			t.Errorf("Expected dateOffset length 10, got %q", dOffsetDays)
		}

		dOffsetHours, err := interpolateString(`$dateOffset("-5h")`, vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(dOffsetHours) != 10 { // Default format is "2006-01-02"
			t.Errorf("Expected dateOffset default format length 10, got %q", dOffsetHours)
		}

		// Fail cases
		_, err = interpolateString(`$dateOffset("+14x")`, vars)
		if err == nil {
			t.Error("Expected error for invalid duration unit")
		}
		_, err = interpolateString(`$dateOffset()`, vars)
		if err == nil {
			t.Error("Expected error for empty dateOffset args")
		}
	})

	t.Run("Security & Cryptography generators", func(t *testing.T) {
		// base64 & base64Decode
		encoded, err := interpolateString(`$base64("hello world")`, vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		expectedEncoded := "aGVsbG8gd29ybGQ="
		if encoded != expectedEncoded {
			t.Errorf("Expected base64 encoded %q, got %q", expectedEncoded, encoded)
		}

		decoded, err := interpolateString(`$base64Decode("aGVsbG8gd29ybGQ=")`, vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if decoded != "hello world" {
			t.Errorf("Expected base64 decoded 'hello world', got %q", decoded)
		}

		// urlencode & urldecode
		encUrl, err := interpolateString(`$urlencode("a b")`, vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if encUrl != "a+b" && encUrl != "a%20b" {
			t.Errorf("Expected urlencode 'a+b' or 'a%%20b', got %q", encUrl)
		}

		decUrl, err := interpolateString(`$urldecode("a%20b")`, vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if decUrl != "a b" {
			t.Errorf("Expected urldecode 'a b', got %q", decUrl)
		}

		// hashing
		md5Val, err := interpolateString(`$hash(md5, "hello")`, vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		expectedMd5 := "5d41402abc4b2a76b9719d911017c592"
		if md5Val != expectedMd5 {
			t.Errorf("Expected md5 hash %q, got %q", expectedMd5, md5Val)
		}

		sha256Val, err := interpolateString(`$hash(sha256, "hello")`, vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		expectedSha256 := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
		if sha256Val != expectedSha256 {
			t.Errorf("Expected sha256 hash %q, got %q", expectedSha256, sha256Val)
		}

		// hmac
		hmacVal, err := interpolateString(`$hmac(sha256, "secret", "hello")`, vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(hmacVal) != 64 {
			t.Errorf("Expected hmac sha256 length 64, got %q", hmacVal)
		}

		// Fail cases
		_, err = interpolateString(`$hash(sha512, "payload")`, vars)
		if err == nil {
			t.Error("Expected error for unsupported hashing algorithm")
		}
		_, err = interpolateString(`$hash(md5)`, vars)
		if err == nil {
			t.Error("Expected error for missing hash arguments")
		}
	})

	t.Run("Mock Data & Fakers", func(t *testing.T) {
		// randomString
		randStr, err := interpolateString(`$randomString(10, alpha)`, vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(randStr) != 10 {
			t.Errorf("Expected random string length 10, got %q", randStr)
		}

		// randomEmail
		email, err := interpolateString(`$randomEmail()`, vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !strings.HasSuffix(email, "@example.com") {
			t.Errorf("Expected random email to end with @example.com, got %q", email)
		}

		// randomPhone
		phoneID, err := interpolateString(`$randomPhone("ID")`, vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !strings.HasPrefix(phoneID, "+628") {
			t.Errorf("Expected Indonesian phone prefix +628, got %q", phoneID)
		}

		phoneUS, err := interpolateString(`$randomPhone("US")`, vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !strings.HasPrefix(phoneUS, "+1") {
			t.Errorf("Expected US phone prefix +1, got %q", phoneUS)
		}

		phoneSG, err := interpolateString(`$randomPhone("SG")`, vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !strings.HasPrefix(phoneSG, "+658") {
			t.Errorf("Expected SG phone prefix +658, got %q", phoneSG)
		}
		if len(phoneSG) != 11 { // +658 + 7 digits = 11 characters
			t.Errorf("Expected SG phone length 11, got %d (%q)", len(phoneSG), phoneSG)
		}

		phoneRawPlus, err := interpolateString(`$randomPhone("+351")`, vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !strings.HasPrefix(phoneRawPlus, "+351") {
			t.Errorf("Expected custom raw prefix +351, got %q", phoneRawPlus)
		}
		if len(phoneRawPlus) != 13 { // +351 + 9 digits = 13 characters
			t.Errorf("Expected phone length 13 for prefix +351, got %d (%q)", len(phoneRawPlus), phoneRawPlus)
		}

		phoneRawNum, err := interpolateString(`$randomPhone("351")`, vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !strings.HasPrefix(phoneRawNum, "+351") {
			t.Errorf("Expected coerced raw prefix +351, got %q", phoneRawNum)
		}

		phoneEmpty, err := interpolateString(`$randomPhone()`, vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !strings.HasPrefix(phoneEmpty, "+628") {
			t.Errorf("Expected default Indonesian phone prefix +628, got %q", phoneEmpty)
		}
	})

	t.Run("Transformers", func(t *testing.T) {
		upper, err := interpolateString(`$toUpper("hello")`, vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if upper != "HELLO" {
			t.Errorf("Expected uppercase 'HELLO', got %q", upper)
		}

		lower, err := interpolateString(`$toLower("HELLO")`, vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if lower != "hello" {
			t.Errorf("Expected lowercase 'hello', got %q", lower)
		}

		trimmed, err := interpolateString(`$trim("  hello  ")`, vars)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if trimmed != "hello" {
			t.Errorf("Expected trimmed 'hello', got %q", trimmed)
		}
	})
}

