package model

// Schema represents a reusable JSON Schema-like definition for assertion validation.
type Schema struct {
	Type       string              `yaml:"type"`
	Required   []string            `yaml:"required,omitempty"`
	Properties map[string]*Schema  `yaml:"properties,omitempty"`
	Items      *Schema             `yaml:"items,omitempty"`      // For array item validation
	Format     string              `yaml:"format,omitempty"`     // email, uuid, datetime, uri
	Enum       []interface{}       `yaml:"enum,omitempty"`
	Pattern    string              `yaml:"pattern,omitempty"`
	MinLength  *int                `yaml:"minLength,omitempty"`
	MaxLength  *int                `yaml:"maxLength,omitempty"`
	Minimum    *float64            `yaml:"minimum,omitempty"`
	Maximum    *float64            `yaml:"maximum,omitempty"`
	MinItems   *int                `yaml:"minItems,omitempty"`
	MaxItems   *int                `yaml:"maxItems,omitempty"`
	Nullable   bool                `yaml:"nullable,omitempty"`
}
