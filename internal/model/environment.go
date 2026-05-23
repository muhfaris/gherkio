package model

// Environment represents an environment configuration YAML file.
type Environment struct {
	BaseURL  string             `yaml:"baseUrl" jsonschema:"required,description=Base URL for all requests"`
	Services map[string]Service `yaml:"services,omitempty" jsonschema:"description=Named service base URL overrides"`
}

// Service represents a named service within an environment.
type Service struct {
	BaseURL string `yaml:"baseUrl" jsonschema:"required,description=Service-specific base URL"`
}
