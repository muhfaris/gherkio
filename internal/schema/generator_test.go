package schema

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateJSONSchema(t *testing.T) {
	b, err := GenerateJSONSchema()
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	if len(b) == 0 {
		t.Fatal("Expected generated schema, got empty bytes")
	}
}

func TestGenerateAllSchemas(t *testing.T) {
	b, err := GenerateAllSchemas()
	if err != nil {
		t.Fatalf("Failed to generate all schemas: %v", err)
	}

	if len(b) == 0 {
		t.Fatal("Expected generated schemas, got empty bytes")
	}

	// Parse and verify structure
	var result map[string]interface{}
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify $defs key exists
	defs, ok := result["$defs"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected $defs in output")
	}

	// Verify all schema types are present flatly in defs
	expectedTypes := []string{"TestFile", "Config", "Environment", "Credentials", "Schema"}
	for _, typ := range expectedTypes {
		if _, ok := defs[typ]; !ok {
			t.Errorf("Missing flat schema type definition: %s", typ)
		}
	}

	// Verify $schema is present
	if _, ok := result["$schema"]; !ok {
		t.Error("Missing $schema key")
	}
}

func TestGenerateSchemaType(t *testing.T) {
	tests := []struct {
		name    string
		typ     SchemaType
		wantErr bool
	}{
		{"test schema", SchemaTypeTest, false},
		{"config schema", SchemaTypeConfig, false},
		{"environment schema", SchemaTypeEnvironment, false},
		{"credentials schema", SchemaTypeCredentials, false},
		{"schema-definition schema", SchemaTypeSchemaDefinition, false},
		{"invalid schema", SchemaType("invalid"), true}, // Returns nil, not error
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := GenerateSchemaType(tt.typ)
			if err != nil {
				t.Fatalf("GenerateSchemaType() error = %v", err)
			}

			if tt.wantErr && b == nil {
				return // Expected nil for invalid type
			}

			if len(b) == 0 && !tt.wantErr {
				t.Error("Expected non-empty schema")
			}

			if b != nil {
				// Verify valid JSON
				var result interface{}
				if err := json.Unmarshal(b, &result); err != nil {
					t.Errorf("Invalid JSON output: %v", err)
				}
			}
		})
	}
}

func getSchemaDef(schema map[string]interface{}) map[string]interface{} {
	// jsonschema reflector nests properties inside $defs.<TypeName>
	defs, ok := schema["$defs"].(map[string]interface{})
	if !ok {
		return nil
	}
	// Return the first (and usually only) definition
	for _, v := range defs {
		if def, ok := v.(map[string]interface{}); ok {
			return def
		}
	}
	return nil
}

func TestGenerateSchemaType_ContainsExpectedKeys(t *testing.T) {
	tests := []struct {
		typ     SchemaType
		checkFn func(map[string]interface{}) bool
	}{
		{
			SchemaTypeTest,
			func(m map[string]interface{}) bool {
				// Test schema should have $defs with Expect and Matcher
				defs, ok := m["$defs"].(map[string]interface{})
				if !ok {
					return false
				}
				_, hasExpect := defs["Expect"]
				_, hasMatcher := defs["Matcher"]
				return hasExpect && hasMatcher
			},
		},
		{
			SchemaTypeConfig,
			func(m map[string]interface{}) bool {
				// Config schema should have properties
				def := getSchemaDef(m)
				if def == nil {
					return false
				}
				_, hasProps := def["properties"].(map[string]interface{})
				return hasProps
			},
		},
		{
			SchemaTypeEnvironment,
			func(m map[string]interface{}) bool {
				// Environment schema should have baseUrl required
				def := getSchemaDef(m)
				if def == nil {
					return false
				}
				if req, ok := def["required"].([]interface{}); ok {
					for _, r := range req {
						if r == "BaseURL" {
							return true
						}
					}
				}
				return false
			},
		},
		{
			SchemaTypeCredentials,
			func(m map[string]interface{}) bool {
				// Credentials should have accounts property in any definition
				defs, ok := m["$defs"].(map[string]interface{})
				if !ok {
					return false
				}
				for _, v := range defs {
					if def, ok := v.(map[string]interface{}); ok {
						if props, ok := def["properties"].(map[string]interface{}); ok {
							if _, hasAccounts := props["Accounts"]; hasAccounts {
								return true
							}
						}
					}
				}
				return false
			},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.typ), func(t *testing.T) {
			b, err := GenerateSchemaType(tt.typ)
			if err != nil {
				t.Fatalf("GenerateSchemaType() error = %v", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(b, &result); err != nil {
				t.Fatalf("Invalid JSON: %v", err)
			}

			if !tt.checkFn(result) {
				t.Errorf("Schema type %s does not meet expected structure", tt.typ)
			}
		})
	}
}

func TestAvailableSchemaTypes(t *testing.T) {
	types := AvailableSchemaTypes()

	if len(types) != 5 {
		t.Errorf("Expected 5 schema types, got %d", len(types))
	}

	// Verify expected types exist
	typeNames := make(map[string]bool)
	for _, t := range types {
		typeNames[string(t.Type)] = true
	}

	expected := []string{"test", "config", "environment", "credentials", "schema-definition"}
	for _, e := range expected {
		if !typeNames[e] {
			t.Errorf("Missing expected type: %s", e)
		}
	}

	// Verify each type has required fields
	for _, st := range types {
		if st.Name == "" {
			t.Error("SchemaTypeInfo.Name is empty")
		}
		if st.Description == "" {
			t.Error("SchemaTypeInfo.Description is empty")
		}
		if len(st.FilePatterns) == 0 {
			t.Error("SchemaTypeInfo.FilePatterns is empty")
		}
	}
}

func TestSchemaTypes_HaveCorrectFilePatterns(t *testing.T) {
	types := AvailableSchemaTypes()

	patterns := map[string]string{
		"test":              ".gherkio/tests/**/*.yaml",
		"config":            ".gherkio/config.yaml",
		"environment":       ".gherkio/environments/*.yaml",
		"credentials":       ".gherkio/credentials/*.yaml",
		"schema-definition": ".gherkio/schemas/*.yaml",
	}

	for _, st := range types {
		expected, ok := patterns[string(st.Type)]
		if !ok {
			t.Errorf("Unexpected type: %s", st.Type)
			continue
		}
		if len(st.FilePatterns) != 1 || st.FilePatterns[0] != expected {
			t.Errorf("Type %s: expected pattern %q, got %v", st.Type, expected, st.FilePatterns)
		}
	}
}

func TestGenerateAllSchemas_ContainsAllSchemaTypes(t *testing.T) {
	b, err := GenerateAllSchemas()
	if err != nil {
		t.Fatalf("GenerateAllSchemas() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	defs := result["$defs"].(map[string]interface{})

	// Verify flat schema has Expect and Matcher definitions
	if _, hasExpect := defs["Expect"]; !hasExpect {
		t.Error("Schema missing Expect definition")
	}
	if _, hasMatcher := defs["Matcher"]; !hasMatcher {
		t.Error("Schema missing Matcher definition")
	}
}

func TestGenerateAllSchemas_DefaultSchemaOutput(t *testing.T) {
	// This test verifies the default output of GenerateAllSchemas
	// matches the structure expected by LSP configurations
	b, err := GenerateAllSchemas()
	if err != nil {
		t.Fatalf("GenerateAllSchemas() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Check $schema value
	schema, ok := result["$schema"].(string)
	if !ok {
		t.Error("$schema should be a string")
	}
	if !strings.Contains(schema, "json-schema.org") {
		t.Errorf("Expected json-schema.org in $schema, got: %s", schema)
	}

	// Check $defs exists
	if _, ok := result["$defs"]; !ok {
		t.Error("Missing $defs key")
	}
}

func TestGenerateSchemaType_BackwardCompatibility(t *testing.T) {
	// Test that GenerateJSONSchema (deprecated) still works
	oldSchema, err := GenerateJSONSchema()
	if err != nil {
		t.Fatalf("GenerateJSONSchema() error = %v", err)
	}

	// And equals GenerateSchemaType(SchemaTypeTest)
	newSchema, err := GenerateSchemaType(SchemaTypeTest)
	if err != nil {
		t.Fatalf("GenerateSchemaType() error = %v", err)
	}

	if string(oldSchema) != string(newSchema) {
		t.Error("GenerateJSONSchema and GenerateSchemaType(SchemaTypeTest) should produce identical output")
	}
}
