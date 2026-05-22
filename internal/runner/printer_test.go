package runner

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "update golden files")

// loadGolden loads a golden file, creating it with -update flag.
func loadGolden(t *testing.T, name, actual string) string {
	t.Helper()

	goldenPath := filepath.Join("testdata", name+".golden")

	if *update {
		err := os.MkdirAll(filepath.Dir(goldenPath), 0755)
		if err != nil {
			t.Fatalf("failed to create testdata directory: %v", err)
		}
		err = os.WriteFile(goldenPath, []byte(actual), 0644)
		if err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}
		return actual
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("no golden file at %s. run with -update to create", goldenPath)
	}

	return string(expected)
}

// captureStdout calls fn and returns everything written to stdout.
func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old

	return buf.String()
}

func mkResp(status int, parsed interface{}) *ResponseInfo {
	return &ResponseInfo{
		Status: status,
		Parsed: parsed,
	}
}

func TestPrintResult_SummaryOutput(t *testing.T) {
	result := &RunResult{
		Scenario: "login example",
		Steps: []StepResult{
			{
				Request: &RequestInfo{
					Method:  "POST",
					URL:     "https://dummyjson.com/auth/login",
					Headers: map[string]string{"Content-Type": "application/json"},
					Body:    `{"username":"emilys","password":"emilyspass","expiresInMins":30}`,
				},
				Response: &ResponseInfo{
					Status: 200,
					Body:   `{"accessToken":"abc123","refreshToken":"def456","username":"emilys","email":"emily.johnson@x.dummyjson.com","firstName":"Emily","lastName":"Johnson","gender":"female","id":1,"image":"https://dummyjson.com/icon/emilys/128"}`,
					Parsed: map[string]interface{}{
						"accessToken":  "abc123",
						"refreshToken": "def456",
						"username":     "emilys",
						"email":        "emily.johnson@x.dummyjson.com",
						"firstName":    "Emily",
						"lastName":     "Johnson",
						"gender":       "female",
						"id":           1,
						"image":        "https://dummyjson.com/icon/emilys/128",
					},
				},
				Assertions: []AssertionResult{
					{Path: "status", Expected: "200", Actual: "200", Passed: true},
					{Path: "body.accessToken", Expected: "exists", Actual: "abc123", Passed: true},
					{Path: "body.refreshToken", Expected: "exists", Actual: "def456", Passed: true},
					{Path: "body.username", Expected: "emily", Actual: "emilys", Passed: false},
				},
				Duration: 388 * time.Millisecond,
			},
		},
		TotalPass: 3,
		TotalFail: 1,
		Duration:  388 * time.Millisecond,
		Passed:    false,
	}

	output := captureStdout(func() {
		PrintResult(result, false, nil)
	})

	expected := loadGolden(t, "summary_output", output)
	if output != expected {
		t.Errorf("output mismatch\n\n=== GOT ===\n%s\n\n=== EXPECTED ===\n%s", output, expected)
	}
}

func TestPrintResult_VerboseOutput(t *testing.T) {
	result := &RunResult{
		Scenario: "login example",
		Steps: []StepResult{
			{
				Request: &RequestInfo{
					Method:  "POST",
					URL:     "https://dummyjson.com/auth/login",
					Headers: map[string]string{"Content-Type": "application/json"},
					Body:    `{"username":"emilys","password":"emilyspass","expiresInMins":30}`,
				},
				Response: &ResponseInfo{
					Status: 200,
					Body:   `{"accessToken":"abc123","refreshToken":"def456","username":"emilys","email":"emily.johnson@x.dummyjson.com","firstName":"Emily","lastName":"Johnson","gender":"female","id":1,"image":"https://dummyjson.com/icon/emilys/128"}`,
					Parsed: map[string]interface{}{
						"accessToken":  "abc123",
						"refreshToken": "def456",
						"username":     "emilys",
						"email":        "emily.johnson@x.dummyjson.com",
						"firstName":    "Emily",
						"lastName":     "Johnson",
						"gender":       "female",
						"id":           1,
						"image":        "https://dummyjson.com/icon/emilys/128",
					},
				},
				Assertions: []AssertionResult{
					{Path: "status", Expected: "200", Actual: "200", Passed: true},
					{Path: "body.accessToken", Expected: "exists", Actual: "abc123", Passed: true},
					{Path: "body.refreshToken", Expected: "exists", Actual: "def456", Passed: true},
					{Path: "body.username", Expected: "emily", Actual: "emilys", Passed: false},
				},
				Duration: 388 * time.Millisecond,
			},
		},
		TotalPass: 3,
		TotalFail: 1,
		Duration:  388 * time.Millisecond,
		Passed:    false,
	}

	output := captureStdout(func() {
		PrintResult(result, true, nil)
	})

	expected := loadGolden(t, "verbose_output", output)
	if output != expected {
		t.Errorf("output mismatch\n\n=== GOT ===\n%s\n\n=== EXPECTED ===\n%s", output, expected)
	}
}

func TestPrintResult_WithAdvancedMatchers(t *testing.T) {
	result := &RunResult{
		Scenario: "advanced matchers test",
		Steps: []StepResult{
			{
				Request: &RequestInfo{
					Method: "GET",
					URL:    "https://dummyjson.com/users/1",
				},
				Response: mkResp(200, map[string]interface{}{
					"id":    "550e8400-e29b-41d4-a716-446655440000",
					"email": "user@example.com",
					"name":  "Laptop Baru",
					"count": 3,
					"tags":  []interface{}{"a", "b", "c"},
				}),
				Assertions: []AssertionResult{
					{Path: "status", Expected: "200", Actual: "200", Passed: true},
					{Path: "body.id", Expected: "valid UUID format", Actual: "550e8400-e29b-41d4-a716-446655440000", Passed: true},
					{Path: "body.email", Expected: "valid email format", Actual: `"user@example.com"`, Passed: true},
					{Path: "body.count", Expected: "number", Actual: "3", Passed: true},
					{Path: "body.name", Expected: `contains substring "Laptop"`, Actual: `"Laptop Baru"`, Passed: true},
					{Path: "count(body.tags)", Expected: "exactly 3 items", Actual: "3", Passed: true},
				},
				Duration: 150 * time.Millisecond,
			},
		},
		TotalPass: 6,
		TotalFail: 0,
		Duration:  150 * time.Millisecond,
		Passed:    true,
	}

	output := captureStdout(func() {
		PrintResult(result, false, nil)
	})

	expected := loadGolden(t, "advanced_matchers_output", output)
	if output != expected {
		t.Errorf("output mismatch\n\n=== GOT ===\n%s\n\n=== EXPECTED ===\n%s", output, expected)
	}
}

func TestPrintResult_WithTimingAssertion(t *testing.T) {
	result := &RunResult{
		Scenario: "timing test",
		Steps: []StepResult{
			{
				Request: &RequestInfo{
					Method: "GET",
					URL:    "https://dummyjson.com/health",
				},
				Response: mkResp(200, map[string]interface{}{"status": "ok"}),
				Assertions: []AssertionResult{
					{Path: "status", Expected: "200", Actual: "200", Passed: true},
					{Path: "timing.max", Expected: "max 500ms", Actual: "312ms", Passed: true},
				},
				Duration: 312 * time.Millisecond,
			},
		},
		TotalPass: 2,
		TotalFail: 0,
		Duration:  312 * time.Millisecond,
		Passed:    true,
	}

	output := captureStdout(func() {
		PrintResult(result, false, nil)
	})

	expected := loadGolden(t, "timing_output", output)
	if output != expected {
		t.Errorf("output mismatch\n\n=== GOT ===\n%s\n\n=== EXPECTED ===\n%s", output, expected)
	}
}

// Tests for helper functions

func TestPrintResult_SchemaAssertion(t *testing.T) {
	schemaReason := "field body.email: expected format email, got not-an-email\nfield body.id: field body.id is required but (missing)"

	result := &RunResult{
		Scenario: "schema validation",
		Steps: []StepResult{
			{
				Request: &RequestInfo{
					Method: "GET",
					URL:    "https://api.example.com/users",
				},
				Response: &ResponseInfo{
					Status: 200,
					Body:   `{"id":"abc","email":"not-an-email"}`,
					Parsed: map[string]interface{}{
						"id":    "abc",
						"email": "not-an-email",
					},
				},
				Assertions: []AssertionResult{
					{Path: "status", Expected: "200", Actual: "200", Passed: true},
					{Path: "schema", Expected: "users/user-response", Actual: "field body.email: expected format email, got not-an-email", Passed: false, Reason: schemaReason},
				},
				Duration: 50 * time.Millisecond,
			},
		},
		TotalPass: 1,
		TotalFail: 1,
		Duration:  50 * time.Millisecond,
		Passed:    false,
	}

	output := captureStdout(func() {
		PrintResult(result, false, nil)
	})

	expected := loadGolden(t, "schema_output", output)
	if output != expected {
		t.Errorf("output mismatch\n\n=== GOT ===\n%s\n\n=== EXPECTED ===\n%s", output, expected)
	}
}

func TestMaskSensitiveData(t *testing.T) {
	tests := []struct {
		name   string
		data   interface{}
		fields []string
		want   interface{}
	}{
		{
			name:   "token masked",
			data:   map[string]interface{}{"token": "abc123", "name": "Emily"},
			fields: defaultSensitiveFields,
			want:   map[string]interface{}{"token": "***masked***", "name": "Emily"},
		},
		{
			name:   "empty fields no masking",
			data:   map[string]interface{}{"token": "abc123"},
			fields: []string{},
			want:   map[string]interface{}{"token": "abc123"},
		},
		{
			name:   "nested masking",
			data:   map[string]interface{}{"user": map[string]interface{}{"token": "secret", "name": "John"}},
			fields: defaultSensitiveFields,
			want:   map[string]interface{}{"user": map[string]interface{}{"token": "***masked***", "name": "John"}},
		},
		{
			name:   "array masking",
			data:   map[string]interface{}{"items": []interface{}{map[string]interface{}{"token": "secret"}, map[string]interface{}{"token": "secret2"}}},
			fields: defaultSensitiveFields,
			want:   map[string]interface{}{"items": []interface{}{map[string]interface{}{"token": "***masked***"}, map[string]interface{}{"token": "***masked***"}}},
		},
		{
			name:   "primitive pass-through",
			data:   "string",
			fields: defaultSensitiveFields,
			want:   "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskSensitiveData(tt.data, tt.fields)
			gotStr := toString(got)
			wantStr := toString(tt.want)
			if gotStr != wantStr {
				t.Errorf("MaskSensitiveData() = %s, want %s", gotStr, wantStr)
			}
		})
	}
}

// toString is a helper for comparing interface{} values in tests.
func toString(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestIsSensitiveField(t *testing.T) {
	tests := []struct {
		name   string
		field  string
		fields []string
		want   bool
	}{
		{"exact match", "token", []string{"token", "password"}, true},
		{"case insensitive", "Token", []string{"token", "password"}, true},
		{"case insensitive reverse", "TOKEN", []string{"token"}, true},
		{"no match", "name", []string{"token", "password"}, false},
		{"empty fields", "token", []string{}, false},
		{"substring no match", "tokenizer", []string{"token"}, false}, // not a match
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSensitiveField(tt.field, tt.fields)
			if got != tt.want {
				t.Errorf("isSensitiveField(%q, %v) = %v, want %v", tt.field, tt.fields, got, tt.want)
			}
		})
	}
}

func TestFormatRequestBody(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		maskFields []string
		want       string
	}{
		{
			name:       "valid json",
			body:       `{"name":"test","value":42}`,
			maskFields: nil,
			want:       "{\n  \"name\": \"test\",\n  \"value\": 42\n}",
		},
		{
			name:       "valid json with masking",
			body:       `{"token":"secret123","name":"test"}`,
			maskFields: defaultSensitiveFields,
			want:       "{\n  \"name\": \"test\",\n  \"token\": \"***masked***\"\n}",
		},
		{
			name:       "invalid json returns raw",
			body:       `not-json`,
			maskFields: nil,
			want:       "not-json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatRequestBody(tt.body, tt.maskFields)
			if got != tt.want {
				t.Errorf("FormatRequestBody() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{500 * time.Millisecond, "500ms"},
		{1 * time.Second, "1.0s"},
		{1500 * time.Millisecond, "1.5s"},
		{2 * time.Minute, "2m0s"},
		{0, "0ms"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatDuration(tt.d)
			if got != tt.want {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestStepSpacing(t *testing.T) {
	tests := []struct {
		depth int
		want  string
	}{
		{0, "   "},
		{1, " "},
		{5, " "},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("depth_%d", tt.depth), func(t *testing.T) {
			got := stepSpacing(tt.depth)
			if got != tt.want {
				t.Errorf("stepSpacing(%d) = %q, want %q", tt.depth, got, tt.want)
			}
		})
	}
}
