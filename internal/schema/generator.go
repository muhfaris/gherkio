package schema

import (
	"encoding/json"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/muhfaris/gherkio/internal/model"
	"github.com/muhfaris/gherkio/internal/runner"
)

// SchemaType represents a type of Gherkio YAML document.
type SchemaType string

const (
	SchemaTypeTest             SchemaType = "test"
	SchemaTypeConfig           SchemaType = "config"
	SchemaTypeEnvironment      SchemaType = "environment"
	SchemaTypeCredentials      SchemaType = "credentials"
	SchemaTypeSchemaDefinition SchemaType = "schema-definition"
)

// SchemaTypeInfo contains metadata about a schema type.
type SchemaTypeInfo struct {
	Type         SchemaType
	Name         string
	Description  string
	FilePatterns []string
}

// AvailableSchemaTypes returns information about all available schema types.
func AvailableSchemaTypes() []SchemaTypeInfo {
	return []SchemaTypeInfo{
		{SchemaTypeTest, "test", "Test file schema for .gherkio/tests/**/*.yaml", []string{".gherkio/tests/**/*.yaml"}},
		{SchemaTypeConfig, "config", "Configuration schema for .gherkio/config.yaml", []string{".gherkio/config.yaml"}},
		{SchemaTypeEnvironment, "environment", "Environment schema for .gherkio/environments/*.yaml", []string{".gherkio/environments/*.yaml"}},
		{SchemaTypeCredentials, "credentials", "Credentials schema for .gherkio/credentials/*.yaml", []string{".gherkio/credentials/*.yaml"}},
		{SchemaTypeSchemaDefinition, "schema-definition", "Schema definition schema for .gherkio/schemas/*.yaml", []string{".gherkio/schemas/*.yaml"}},
	}
}

// GenerateAllSchemas generates schemas for all Gherkio YAML document types.
// Returns a JSON object containing all schemas keyed by type.
func GenerateAllSchemas() ([]byte, error) {
	r := &jsonschema.Reflector{
		RequiredFromJSONSchemaTags: true,
		Anonymous:                  true,
	}

	// Build individual schemas for each document type
	testSchema := r.Reflect(&model.TestFile{})
	configSchema := r.Reflect(&model.Config{})
	envSchema := r.Reflect(&model.Environment{})
	credSchema := r.Reflect(&model.Credentials{})
	schemaDefSchema := r.Reflect(&model.Schema{})

	// Apply Expect patching only to test schema
	patchExpectSchema(testSchema)

	// Add step oneOf constraint to test schema
	patchStepOneOf(testSchema)

	// Add ProjectionConfig enhancements
	patchProjectionSchema(testSchema)

	// Combine all definitions ($defs) into a single flat map
	allDefs := make(map[string]interface{})

	mergeDefs := func(s *jsonschema.Schema) {
		for k, v := range s.Definitions {
			allDefs[k] = v
		}
	}

	mergeDefs(testSchema)
	mergeDefs(configSchema)
	mergeDefs(envSchema)
	mergeDefs(credSchema)
	mergeDefs(schemaDefSchema)

	// Combine into a single flat output using $defs (draft-07 compatible)
	combined := map[string]interface{}{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"oneOf": []map[string]string{
			{"$ref": "#/$defs/TestFile"},
			{"$ref": "#/$defs/Config"},
			{"$ref": "#/$defs/Environment"},
			{"$ref": "#/$defs/Credentials"},
			{"$ref": "#/$defs/Schema"},
		},
		"$defs": allDefs,
	}

	return json.MarshalIndent(combined, "", "  ")
}

// GenerateSchemaType generates a schema for a specific document type.
func GenerateSchemaType(schemaType SchemaType) ([]byte, error) {
	r := &jsonschema.Reflector{
		RequiredFromJSONSchemaTags: true,
		Anonymous:                  true,
	}

	switch schemaType {
	case SchemaTypeTest:
		schema := r.Reflect(&model.TestFile{})
		patchExpectSchema(schema)
		patchStepOneOf(schema)
		patchProjectionSchema(schema)
		return json.MarshalIndent(schema, "", "  ")
	case SchemaTypeConfig:
		schema := r.Reflect(&model.Config{})
		return json.MarshalIndent(schema, "", "  ")
	case SchemaTypeEnvironment:
		schema := r.Reflect(&model.Environment{})
		return json.MarshalIndent(schema, "", "  ")
	case SchemaTypeCredentials:
		schema := r.Reflect(&model.Credentials{})
		return json.MarshalIndent(schema, "", "  ")
	case SchemaTypeSchemaDefinition:
		schema := r.Reflect(&model.Schema{})
		return json.MarshalIndent(schema, "", "  ")
	default:
		return nil, nil
	}
}

// GenerateJSONSchema generates a JSON schema for the Gherkio DSL based on internal/model.TestFile.
// Deprecated: Use GenerateAllSchemas or GenerateSchemaType instead.
func GenerateJSONSchema() ([]byte, error) {
	return GenerateSchemaType(SchemaTypeTest)
}

// patchExpectSchema applies advanced autocomplete enhancements to the Expect definition.
func patchExpectSchema(schema *jsonschema.Schema) {
	if expectSchema, ok := schema.Definitions["Expect"]; ok {
		// Define the Matcher definition dynamically from the runner engine
		matchers := runner.GetAvailableMatchers()
		matcherEnums := make([]interface{}, len(matchers))
		for i, m := range matchers {
			matcherEnums[i] = m
		}

		matcherSchema := &jsonschema.Schema{
			AnyOf: []*jsonschema.Schema{
				{
					Type:        "string",
					Enum:        matcherEnums,
					Description: "Gherkio assertion matchers (keyword-based)",
				},
				{
					Description: "Literal value for equality comparison (string, number, boolean, null, array, or object)",
				},
			},
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

		// Add dynamic collection functions (e.g. ^count\(.+\)$, ^all\(.+\)$)
		collectionFuncs := runner.GetCollectionFunctions()
		for _, fn := range collectionFuncs {
			pattern := "^" + fn + "\\(.+\\)$"
			if fn == "count" {
				expectSchema.PatternProperties[pattern] = &jsonschema.Schema{
					Type:        "integer",
					Description: "Assert exact length of an array",
				}
			} else {
				expectSchema.PatternProperties[pattern] = &jsonschema.Schema{
					Ref:         "#/$defs/Matcher",
					Description: "Assert every element in an array matches this condition",
				}
			}
		}

		// Remove the generic AdditionalProperties since we are using PatternProperties and specific keys
		expectSchema.AdditionalProperties = nil
	}
}

// patchProjectionSchema adds enhanced descriptions to the ProjectionConfig schema.
func patchProjectionSchema(schema *jsonschema.Schema) {
	if projSchema, ok := schema.Definitions["ProjectionConfig"]; ok {
		if selectProp, ok := projSchema.Properties.Get("select"); ok {
			selectProp.Description = "Structural projection mapping for item fields. " +
				"Supports variable references ($item.field), type casting ($string $int $float $bool), " +
				"conditional selection ($if(condition, then, else)), string interpolation, " +
				"static values, and nested sub-projections."
		}
	}
}

// patchStepOneOf adds oneOf constraint to Step to ensure request OR use is provided, not both.
func patchStepOneOf(schema *jsonschema.Schema) {
	if stepSchema, ok := schema.Definitions["Step"]; ok {
		// Make all step fields optional except request OR use
		stepSchema.Required = []string{}

		// Add oneOf constraint for request/use mutual exclusion
		stepSchema.OneOf = []*jsonschema.Schema{
			{
				Required:    []string{"request"},
				Description: "Step with HTTP request",
			},
			{
				Required:    []string{"use"},
				Description: "Step composing another scenario",
			},
		}
	}
}
