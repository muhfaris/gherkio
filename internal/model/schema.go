package model

// Schema represents a reusable JSON Schema-like definition for assertion validation.
type Schema struct {
	Type       string             `yaml:"type" jsonschema:"required,description=JSON schema type (object, array, string, number, boolean, null)"`
	Required   []string           `yaml:"required,omitempty" jsonschema:"description=List of required property names"`
	Properties map[string]*Schema `yaml:"properties,omitempty" jsonschema:"description=Object property definitions"`
	Items      *Schema            `yaml:"items,omitempty" jsonschema:"description=Schema for array items"`
	Format     string             `yaml:"format,omitempty" jsonschema:"enum=email,enum=uuid,enum=datetime,enum=uri,description=Value format constraint"`
	Enum       []interface{}      `yaml:"enum,omitempty" jsonschema:"description=Allowed values (anyOf)"`
	Pattern    string             `yaml:"pattern,omitempty" jsonschema:"description=Regex pattern for string values"`
	MinLength  *int               `yaml:"minLength,omitempty" jsonschema:"description=Minimum string length"`
	MaxLength  *int               `yaml:"maxLength,omitempty" jsonschema:"description=Maximum string length"`
	Minimum    *float64           `yaml:"minimum,omitempty" jsonschema:"description=Minimum numeric value"`
	Maximum    *float64           `yaml:"maximum,omitempty" jsonschema:"description=Maximum numeric value"`
	MinItems   *int               `yaml:"minItems,omitempty" jsonschema:"description=Minimum array length"`
	MaxItems   *int               `yaml:"maxItems,omitempty" jsonschema:"description=Maximum array length"`
	Nullable   bool               `yaml:"nullable,omitempty" jsonschema:"description=Allow null values"`
}
