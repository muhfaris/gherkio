package schema

import (
	"encoding/json"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/muhfaris/gherkio/internal/model"
	"github.com/muhfaris/gherkio/internal/runner"
)

// GenerateJSONSchema generates a JSON schema for the Gherkio DSL based on internal/model.TestFile.
func GenerateJSONSchema() ([]byte, error) {
	r := new(jsonschema.Reflector)
	r.RequiredFromJSONSchemaTags = true

	// Reflect from TestFile which is the root of our YAML DSL
	schema := r.Reflect(&model.TestFile{})

	// --- Advanced Autocomplete for Expect ---
	if expectSchema, ok := schema.Definitions["Expect"]; ok {
		// Define the Matcher definition dynamically from the runner engine
		matchers := runner.GetAvailableMatchers()
		matcherEnums := make([]interface{}, len(matchers))
		for i, m := range matchers {
			matcherEnums[i] = m
		}

		matcherSchema := &jsonschema.Schema{
			Type:        "string",
			Enum:        matcherEnums,
			Description: "Gherkio assertion matchers",
		}
		schema.Definitions["Matcher"] = matcherSchema

		// 1. Explicit properties for Autocomplete suggestions
		// Read canonical paths directly from the engine to ensure schema matches the codebase
		paths := runner.GetCanonicalPaths()

		for _, p := range paths {
			desc := "Assert against response " + p
			if p == "body" {
				desc = "Assert against the JSON response body. Example: body.id, body.data.0.name"
			} else if p == "headers" {
				desc = "Assert against response headers. Example: headers.content-type"
			} else if p == "jwt" {
				desc = "Assert against decoded JWT claims. Example: jwt.role, jwt.exp"
			}

			expectSchema.Properties.Set(p+".", &jsonschema.Schema{
				Ref:         "#/$defs/Matcher",
				Description: desc,
			})
		}

		expectSchema.Properties.Set("schema", &jsonschema.Schema{
			Type:        "string",
			Description: "Validate response body against a predefined JSON schema name",
		})

		// 2. Pattern Properties for dynamic paths
		// Join the canonical paths into a regex group (e.g. ^(body|headers|jwt)\..+$)
		if expectSchema.PatternProperties == nil {
			expectSchema.PatternProperties = make(map[string]*jsonschema.Schema)
		}

		regexPattern := "^(" + strings.Join(paths, "|") + ")\\..+$"
		expectSchema.PatternProperties[regexPattern] = &jsonschema.Schema{
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
