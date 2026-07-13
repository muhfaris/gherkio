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

	// Add Request enhancements
	patchRequestSchema(testSchema)

	// Add Retry enhancements
	patchRetrySchema(testSchema)

	// Add Timing enhancements
	patchTimingSchema(testSchema)

	// Add Steps properties enhancements to test schema
	patchStepsProperties(testSchema)

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
		patchRequestSchema(schema)
		patchRetrySchema(schema)
		patchTimingSchema(schema)
		patchStepsProperties(schema)
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

		expectDesc := "Assertions for the step response.\n\n" +
			"Available options:\n" +
			"- **status**: (Integer, Optional) Expected HTTP status code (e.g. 200).\n" +
			"- **schema**: (String, Optional) Validate response body against a predefined JSON schema name.\n" +
			"- **body.<path>**: Assert against json body fields (e.g. 'body.id: exists', 'body.status: equal success').\n" +
			"- **headers.<header>**: Assert against response headers (e.g. 'headers.content-type: contains json').\n" +
			"- **jwt.<claim>**: Assert against decoded JWT claims (e.g. 'jwt.role: equal admin').\n" +
			"- **count(<path>)**: Assert exact length of an array field (e.g. 'count(body.items): 5').\n" +
			"- **all(<path>)**: Assert every element in an array matches a matcher (e.g. 'all(body.items.status): equal active').\n\n" +
			"Available Gherkio matchers:\n" +
			"- " + strings.Join(matchers, ", ")

		expectSchema.Description = expectDesc

		if stepSchema, ok := schema.Definitions["Step"]; ok {
			if expectProp, ok := stepSchema.Properties.Get("expect"); ok {
				expectProp.Description = expectDesc
			}
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

// patchStepOneOf adds oneOf constraint to Step to ensure request OR use OR set is provided, not both.
func patchStepOneOf(schema *jsonschema.Schema) {
	if stepSchema, ok := schema.Definitions["Step"]; ok {
		// Make all step fields optional except request OR use OR set
		stepSchema.Required = []string{}

		stepSchema.Description = "A single test execution step.\n\n" +
			"Available options:\n" +
			"- **name**: (String, Optional) Human-readable label for the step.\n" +
			"- **if**: (String, Optional) Conditional guard expression (e.g. '$status == 200').\n" +
			"- **request**: (Object, Conditional) HTTP Request config. Mutually exclusive with 'use' and 'set'.\n" +
			"- **use**: (String, Conditional) Path to compose/execute another scenario. Mutually exclusive with 'request' and 'set'.\n" +
			"- **set**: (Map, Conditional) Inline variable assignment / override map. Mutually exclusive with 'request' and 'use'.\n" +
			"- **expect**: (Object, Optional) Response assertions.\n" +
			"- **save**: (Map, Optional) Extract dynamic values to context variables.\n" +
			"- **timing**: (Object, Optional) Execution latency check.\n" +
			"- **retry**: (Object, Optional) Retry configuration for transient failures."

		// Add oneOf constraint for request/use/set mutual exclusion
		stepSchema.OneOf = []*jsonschema.Schema{
			{
				Required:    []string{"request"},
				Description: "Step with HTTP request",
			},
			{
				Required:    []string{"use"},
				Description: "Step composing another scenario",
			},
			{
				Required:    []string{"set"},
				Description: "Step setting or overriding variables inline",
			},
		}
	}
}

// patchRequestSchema updates the descriptions for Request options in the JSON schema.
func patchRequestSchema(schema *jsonschema.Schema) {
	requestDesc := "HTTP request definition.\n\n" +
		"Available options:\n" +
		"- **service**: (String, Optional) Name of the service defined in environment.\n" +
		"- **method**: (String, Required) HTTP method (GET, POST, PUT, DELETE, PATCH, etc.).\n" +
		"- **url**: (String, Required) Request URL path or absolute URL. Supports variable interpolation ($var, ${var:default}, $uuid, $ulid, etc.).\n" +
		"- **query**: (Map, Optional) Query parameters to append to the URL.\n" +
		"- **headers**: (Map, Optional) Custom HTTP request headers.\n" +
		"- **body**: (Object/String, Optional) HTTP request body. Supports variable interpolation and type casting.\n" +
		"- **multipart**: (Object, Optional) Multipart form-data configuration for file uploads.\n" +
		"- **transform**: (Object, Optional) Declarative projections reshaped into request payload."

	if stepSchema, ok := schema.Definitions["Step"]; ok {
		if reqProp, ok := stepSchema.Properties.Get("request"); ok {
			reqProp.Description = requestDesc
		}
	}
	if reqSchema, ok := schema.Definitions["Request"]; ok {
		reqSchema.Description = requestDesc
	}
}

// patchStepsProperties updates the descriptions of setup, steps, and teardown arrays.
func patchStepsProperties(schema *jsonschema.Schema) {
	stepsDesc := "Each step supports the following options:\n" +
		"- **name**: (String, Optional) Human-readable label for the step.\n" +
		"- **if**: (String, Optional) Conditional guard expression (e.g. '$status == 200').\n" +
		"- **request**: (Object, Conditional) HTTP Request config. Mutually exclusive with 'use' and 'set'.\n" +
		"- **use**: (String, Conditional) Path to compose/execute another scenario. Mutually exclusive with 'request' and 'set'.\n" +
		"- **set**: (Map, Conditional) Inline variable assignment / override map. Mutually exclusive with 'request' and 'use'.\n" +
		"- **expect**: (Object, Optional) Response assertions.\n" +
		"- **save**: (Map, Optional) Extract dynamic values to context variables.\n" +
		"- **timing**: (Object, Optional) Execution latency check.\n" +
		"- **retry**: (Object, Optional) Retry configuration for transient failures."

	var testFileSchema *jsonschema.Schema
	if schema.Definitions != nil {
		if s, ok := schema.Definitions["TestFile"]; ok {
			testFileSchema = s
		}
	}
	if testFileSchema == nil {
		testFileSchema = schema
	}

	if testFileSchema != nil && testFileSchema.Properties != nil {
		if setupProp, ok := testFileSchema.Properties.Get("setup"); ok {
			setupProp.Description = "Pre-condition steps executed before main steps.\n\n" + stepsDesc
		}
		if stepsProp, ok := testFileSchema.Properties.Get("steps"); ok {
			stepsProp.Description = "Main steps to execute for this scenario.\n\n" + stepsDesc
		}
		if teardownProp, ok := testFileSchema.Properties.Get("teardown"); ok {
			teardownProp.Description = "Post-condition steps that always execute, even on failure.\n\n" + stepsDesc
		}
	}
}

// patchRetrySchema updates the descriptions for RetryConfig options in the JSON schema.
func patchRetrySchema(schema *jsonschema.Schema) {
	retryDesc := "Retry configuration for transient step failures.\n\n" +
		"Available options:\n" +
		"- **attempts**: (Integer, Required) Number of retry attempts.\n" +
		"- **interval**: (Integer, Optional) Interval in milliseconds between retries.\n" +
		"- **backoff**: (String, Optional) Backoff strategy (linear or exponential).\n" +
		"- **maxDuration**: (String, Optional) Maximum duration for the retry loop (e.g. '5s', '1m').\n" +
		"- **onStatus**: (Array of Integers, Optional) List of status codes that trigger a retry."

	if stepSchema, ok := schema.Definitions["Step"]; ok {
		if retryProp, ok := stepSchema.Properties.Get("retry"); ok {
			retryProp.Description = retryDesc
		}
	}
	if retrySchema, ok := schema.Definitions["RetryConfig"]; ok {
		retrySchema.Description = retryDesc
	}
}

// patchTimingSchema updates the descriptions for TimingConfig options in the JSON schema.
func patchTimingSchema(schema *jsonschema.Schema) {
	timingDesc := "Timing expectations for the step.\n\n" +
		"Available options:\n" +
		"- **max**: (String, Required) Maximum duration the step is allowed to take (e.g. '500ms', '1s')."

	if stepSchema, ok := schema.Definitions["Step"]; ok {
		if timingProp, ok := stepSchema.Properties.Get("timing"); ok {
			timingProp.Description = timingDesc
		}
	}
	if timingSchema, ok := schema.Definitions["TimingConfig"]; ok {
		timingSchema.Description = timingDesc
	}
}



