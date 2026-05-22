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

	// --- Advanced Autocomplete for Expect ---
	if expectSchema, ok := schema.Definitions["Expect"]; ok {
		// Define the Matcher definition
		matcherSchema := &jsonschema.Schema{
			Type: "string",
			Enum: []interface{}{
				"exists", "not exists",
				"uuid", "email", "datetime",
				"string", "number", "boolean", "array", "object", "null",
				"true", "false",
			},
			Description: "Gherkio assertion matchers",
		}
		schema.Definitions["Matcher"] = matcherSchema

		// 1. Explicit properties for Autocomplete suggestions
		// By adding these explicit keys, the editor will suggest them when typing inside 'expect:'
		expectSchema.Properties.Set("body.", &jsonschema.Schema{
			Ref:         "#/$defs/Matcher",
			Description: "Assert against the JSON response body. Example: body.id, body.data.0.name",
		})
		expectSchema.Properties.Set("headers.", &jsonschema.Schema{
			Ref:         "#/$defs/Matcher",
			Description: "Assert against response headers. Example: headers.content-type",
		})
		expectSchema.Properties.Set("jwt.", &jsonschema.Schema{
			Ref:         "#/$defs/Matcher",
			Description: "Assert against decoded JWT claims. Example: jwt.role, jwt.exp",
		})
		expectSchema.Properties.Set("schema", &jsonschema.Schema{
			Type:        "string",
			Description: "Validate response body against a predefined JSON schema name",
		})

		// 2. Pattern Properties for dynamic paths
		// This allows ANY path starting with body., headers., jwt. to be valid and use the Matcher autocomplete
		if expectSchema.PatternProperties == nil {
			expectSchema.PatternProperties = make(map[string]*jsonschema.Schema)
		}
		expectSchema.PatternProperties["^(body|headers|jwt)\\..+$"] = &jsonschema.Schema{
			Ref: "#/$defs/Matcher",
		}

		// Remove the generic AdditionalProperties since we are using PatternProperties and specific keys
		expectSchema.AdditionalProperties = nil
	}

	// Make all step fields optional EXCEPT request OR use
	if stepSchema, ok := schema.Definitions["Step"]; ok {
		stepSchema.Required = []string{}
	}

	// Format as indented JSON
	return json.MarshalIndent(schema, "", "  ")
}
