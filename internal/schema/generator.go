package schema

import (
	"encoding/json"

	"github.com/invopop/jsonschema"
	"github.com/muhfaris/gherkio/internal/model"
)

// GenerateJSONSchema generates a JSON schema for the Gherkio DSL based on internal/model.TestFile.
func GenerateJSONSchema() ([]byte, error) {
	r := new(jsonschema.Reflector)
	r.RequiredFromJSONSchemaTags = true

	// Reflect from TestFile which is the root of our YAML DSL
	schema := r.Reflect(&model.TestFile{})

	// Expect allows dynamic assertions (like body.id: uuid)
	if expectSchema, ok := schema.Definitions["Expect"]; ok {
		expectSchema.AdditionalProperties = &jsonschema.Schema{
			OneOf: []*jsonschema.Schema{
				{Type: "string", Description: "String matcher (e.g., exists, uuid, string, number)"},
				{Type: "number", Description: "Numeric exact match"},
				{Type: "boolean", Description: "Boolean exact match"},
				{Type: "object", Description: "Nested matcher object"},
			},
		}
	}

	// Make all step fields optional EXCEPT request OR use (one of them usually required, but it's hard to express in simple schema, so we leave them optional for flexible YAML writing)
	if stepSchema, ok := schema.Definitions["Step"]; ok {
		stepSchema.Required = []string{}
	}

	// Format as indented JSON
	return json.MarshalIndent(schema, "", "  ")
}
