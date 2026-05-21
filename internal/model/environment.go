package model

// Environment represents an environment configuration YAML file.
type Environment struct {
	BaseURL  string             `yaml:"baseUrl"`
	Services map[string]Service `yaml:"services,omitempty"`
}

// Service represents a named service within an environment.
type Service struct {
	BaseURL string `yaml:"baseUrl"`
}
