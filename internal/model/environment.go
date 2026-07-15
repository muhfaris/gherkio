package model

// Environment represents an environment configuration YAML file.
type Environment struct {
	BaseURL  string             `yaml:"baseUrl" jsonschema:"required,description=Base URL for all requests"`
	Services map[string]Service `yaml:"services,omitempty" jsonschema:"description=Named service base URL overrides"`
	Mocks    []MockRule         `yaml:"mocks,omitempty" jsonschema:"description=Outbound request mock definitions"`
}

// Service represents a named service within an environment.
type Service struct {
	BaseURL string `yaml:"baseUrl" jsonschema:"required,description=Service-specific base URL"`
}

// MockRule defines a matching rule and mock response for service virtualization.
type MockRule struct {
	Request  MockRequest  `yaml:"request" json:"request"`
	Response MockResponse `yaml:"response" json:"response"`
}

// MockRequest defines matching criteria for mock interception.
type MockRequest struct {
	Method string `yaml:"method" json:"method" jsonschema:"required,description=HTTP Method to match (e.g. GET, POST)"`
	URL    string `yaml:"url" json:"url" jsonschema:"required,description=URL pattern or exact URL to match"`
}

// MockResponse defines the virtual response returned when a mock request matches.
type MockResponse struct {
	Status  int               `yaml:"status" json:"status" jsonschema:"required,description=HTTP status code to return"`
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty" jsonschema:"description=Response headers"`
	Body    interface{}       `yaml:"body,omitempty" json:"body,omitempty" jsonschema:"description=Response body (can be object or string)"`
}
