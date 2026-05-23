package converter

import (
	"reflect"
	"testing"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Simple GET",
			input:    "curl https://api.com/users",
			expected: []string{"curl", "https://api.com/users"},
		},
		{
			name:     "Headers and Method",
			input:    "curl -X POST https://api.com -H 'Content-Type: application/json'",
			expected: []string{"curl", "-X", "POST", "https://api.com", "-H", "Content-Type: application/json"},
		},
		{
			name:     "Double quotes and escaping",
			input:    `curl -d "{\"key\": \"val\"}" https://api.com`,
			expected: []string{"curl", "-d", `{"key": "val"}`, "https://api.com"},
		},
		{
			name:  "Line continuation",
			input: "curl -X PUT \\\n  -H 'Auth: test' \\\n  https://api.com/update",
			expected: []string{
				"curl", "-X", "PUT", "-H", "Auth: test", "https://api.com/update",
			},
		},
		{
			name:     "Nested quotes",
			input:    `curl -H 'Cookie: name="value"' https://api.com`,
			expected: []string{"curl", "-H", `Cookie: name="value"`, "https://api.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Tokenize(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseCurl(t *testing.T) {
	tests := []struct {
		name         string
		cmd          string
		expectURL    string
		expectMethod string
		expectHeader map[string]string
		expectBody   interface{}
		expectWarn   int
	}{
		{
			name:         "Simple GET",
			cmd:          "curl https://api.example.com/users",
			expectURL:    "https://api.example.com/users",
			expectMethod: "GET",
			expectWarn:   0,
		},
		{
			name:         "POST JSON body",
			cmd:          `curl -X POST https://api.example.com/login -H "Content-Type: application/json" -d '{"email":"test@test.com"}'`,
			expectURL:    "https://api.example.com/login",
			expectMethod: "POST",
			expectHeader: map[string]string{"Content-Type": "application/json"},
			expectBody:   map[string]interface{}{"email": "test@test.com"},
			expectWarn:   0,
		},
		{
			name:         "Basic Auth conversion",
			cmd:          "curl -u 'user:pass' https://api.com",
			expectURL:    "https://api.com",
			expectMethod: "GET",
			expectHeader: map[string]string{"Authorization": "Basic dXNlcjpwYXNz"},
			expectWarn:   0,
		},
		{
			name:         "Warnings for ignored flags",
			cmd:          "curl --insecure -k https://api.com",
			expectURL:    "https://api.com",
			expectMethod: "GET",
			expectWarn:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, warnings, err := ParseCurl(tt.cmd)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if req.URL != tt.expectURL {
				t.Errorf("got URL %q, want %q", req.URL, tt.expectURL)
			}

			if req.Method != tt.expectMethod {
				t.Errorf("got Method %q, want %q", req.Method, tt.expectMethod)
			}

			for k, v := range tt.expectHeader {
				if req.Headers[k] != v {
					t.Errorf("got header %s=%q, want %q", k, req.Headers[k], v)
				}
			}

			if tt.expectBody != nil {
				if !reflect.DeepEqual(req.Body, tt.expectBody) {
					t.Errorf("got Body %v, want %v", req.Body, tt.expectBody)
				}
			}

			if len(warnings) != tt.expectWarn {
				t.Errorf("got %d warnings, want %d: %v", len(warnings), tt.expectWarn, warnings)
			}
		})
	}
}
