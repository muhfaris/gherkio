package converter

import (
	"strings"
	"testing"

	"github.com/muhfaris/gherkio/internal/model"
)

func TestLenientInterpolateString(t *testing.T) {
	vars := map[string]interface{}{
		"host":     "api.example.com",
		"version":  "v1",
		"userId":   123,
		"defvalue": "actual",
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple variable",
			input:    "https://$host/users",
			expected: "https://api.example.com/users",
		},
		{
			name:     "Braced variable",
			input:    "https://${host}/users",
			expected: "https://api.example.com/users",
		},
		{
			name:     "Multiple variables",
			input:    "https://$host/$version/users/$userId",
			expected: "https://api.example.com/v1/users/123",
		},
		{
			name:     "Default value used",
			input:    "https://$host/${missing:default}/users",
			expected: "https://api.example.com/default/users",
		},
		{
			name:     "Default value not used when variable is present",
			input:    "https://$host/${defvalue:default}/users",
			expected: "https://api.example.com/actual/users",
		},
		{
			name:     "Lenient missing variable kept intact",
			input:    "https://$host/$missing/users",
			expected: "https://api.example.com/$missing/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LenientInterpolateString(tt.input, vars)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestConvertStepToCurl(t *testing.T) {
	vars := map[string]interface{}{
		"token": "secret123",
		"email": "user@test.com",
	}

	req := model.Request{
		Method: "POST",
		URL:    "https://api.example.com/login",
		Headers: map[string]string{
			"Authorization": "Bearer $token",
			"Content-Type":  "application/json",
		},
		Body: map[string]interface{}{
			"email": "$email",
			"role":  "admin",
		},
	}

	curl, err := ConvertStepToCurl(req, "", "", vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedPrefix := "curl -X POST 'https://api.example.com/login'"
	if !strings.HasPrefix(curl, expectedPrefix) {
		t.Errorf("expected prefix %q, got %q", expectedPrefix, curl)
	}

	expectedHeaderAuth := "-H 'Authorization: Bearer secret123'"
	if !strings.Contains(curl, expectedHeaderAuth) {
		t.Errorf("expected header %q to be present in cURL, got %q", expectedHeaderAuth, curl)
	}

	expectedBody := "-d '{\"email\":\"user@test.com\",\"role\":\"admin\"}'"
	if !strings.Contains(curl, expectedBody) {
		t.Errorf("expected body %q to be present in cURL, got %q", expectedBody, curl)
	}
}
