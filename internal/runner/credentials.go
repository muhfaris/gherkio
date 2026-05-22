package runner

import (
	"os"
	"path/filepath"

	"github.com/muhfaris/gherkio/internal/model"
	"gopkg.in/yaml.v3"
)

// LoadCredentials loads a credentials file for a given environment.
func LoadCredentials(projectDir, envName string) (*model.Credentials, error) {
	credPath := filepath.Join(projectDir, ".gherkio", "credentials", envName+".yaml")

	data, err := os.ReadFile(credPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No credentials file for this environment
		}
		return nil, err
	}

	var creds model.Credentials
	if err := yaml.Unmarshal(data, &creds); err != nil {
		return nil, err
	}

	if err := creds.Validate(); err != nil {
		return nil, err
	}

	return &creds, nil
}

// CredentialsToVars converts account credentials to a variables map.
func CredentialsToVars(account model.Account) map[string]interface{} {
	vars := make(map[string]interface{})
	vars["username"] = account.Username
	vars["password"] = account.Password
	if account.Role != "" {
		vars["role"] = account.Role
	}
	// Add any extra fields
	for key, val := range account.Extra {
		vars[key] = val
	}
	return vars
}

// GetSensitiveFieldsFromCredentials returns field names to mask based on credentials.
func GetSensitiveFieldsFromCredentials(account model.Account) []string {
	fields := []string{"password"}
	// Also add any field that looks like it contains secrets
	extraFields := []string{"secret", "token", "key", "apiKey", "api_key", "clientSecret"}
	for key, val := range account.Extra {
		keyLower := lowercase(key)
		for _, sensitive := range extraFields {
			if contains(keyLower, sensitive) {
				fields = append(fields, key)
				break
			}
		}
		// Also check if the value looks like a secret
		if isLikelySecret(key, val) {
			fields = append(fields, key)
		}
	}
	return fields
}

func lowercase(s string) string {
	// Simple lowercase for ASCII
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func isLikelySecret(key, value string) bool {
	keyLower := lowercase(key)
	sensitivePatterns := []string{"secret", "token", "key", "password", "credential", "auth"}
	for _, pattern := range sensitivePatterns {
		if contains(keyLower, pattern) {
			return true
		}
	}
	return false
}
