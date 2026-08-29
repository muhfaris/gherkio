package model

// Environment represents an environment configuration YAML file.
type Environment struct {
	BaseURL     string                     `yaml:"baseUrl" jsonschema:"required,description=Base URL for all requests"`
	Services    map[string]Service         `yaml:"services,omitempty" jsonschema:"description=Named service base URL overrides"`
	Connections map[string]RedisConnection `yaml:"connections,omitempty" jsonschema:"description=Named Redis connections available to datastore steps"`
	Mocks       []MockRule                 `yaml:"mocks,omitempty" jsonschema:"description=Outbound request mock definitions"`
}

// RedisConnection configures a named Redis connection. Passwords may reference
// GHERKIO_ environment variables and are interpolated immediately before use.
type RedisConnection struct {
	Type     string         `yaml:"type" jsonschema:"required,enum=redis,description=Connection type"`
	Address  string         `yaml:"address,omitempty" jsonschema:"description=Direct Redis host and port; required when sentinel is not configured"`
	Sentinel *RedisSentinel `yaml:"sentinel,omitempty" jsonschema:"description=Redis Sentinel primary discovery configuration"`
	Username string         `yaml:"username,omitempty" jsonschema:"description=Redis ACL username"`
	Password string         `yaml:"password,omitempty" jsonschema:"description=Redis password; supports Gherkio variable interpolation"`
	Database int            `yaml:"database,omitempty" jsonschema:"description=Redis logical database number"`
	TLS      bool           `yaml:"tls,omitempty" jsonschema:"description=Connect to the discovered Redis primary using TLS"`
	Timeout  string         `yaml:"timeout,omitempty" jsonschema:"description=Discovery connection and command timeout (default 5s)"`
}

// RedisSentinel configures primary discovery through one or more Sentinel nodes.
// Its credentials and TLS setting apply to Sentinel itself, not the Redis primary.
type RedisSentinel struct {
	Master    string   `yaml:"master" jsonschema:"required,description=Sentinel master group name"`
	Addresses []string `yaml:"addresses" jsonschema:"required,minItems=1,description=Sentinel host and port endpoints tried in order"`
	Username  string   `yaml:"username,omitempty" jsonschema:"description=Sentinel ACL username"`
	Password  string   `yaml:"password,omitempty" jsonschema:"description=Sentinel password; supports Gherkio variable interpolation"`
	TLS       bool     `yaml:"tls,omitempty" jsonschema:"description=Connect to Sentinel using TLS"`
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
