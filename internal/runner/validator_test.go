package runner

import (
	"testing"

	"github.com/muhfaris/gherkio/internal/model"
)

func ptrInt(i int) *int             { return &i }
func ptrFloat(f float64) *float64   { return &f }

func TestValidateSchema(t *testing.T) {
	tests := []struct {
		name       string
		data       interface{}
		schema     *model.Schema
		wantViolations int
	}{
		{
			name: "empty schema matches anything",
			data: map[string]interface{}{"foo": "bar"},
			schema: &model.Schema{},
			wantViolations: 0,
		},
		{
			name: "valid simple object",
			data: map[string]interface{}{"id": 1, "name": "Alice"},
			schema: &model.Schema{
				Type: "object",
				Required: []string{"id", "name"},
				Properties: map[string]*model.Schema{
					"id":   {Type: "integer"},
					"name": {Type: "string"},
				},
			},
			wantViolations: 0,
		},
		{
			name: "missing required field",
			data: map[string]interface{}{"name": "Alice"},
			schema: &model.Schema{
				Type: "object",
				Required: []string{"id", "name"},
			},
			wantViolations: 1, // "id" is missing
		},
		{
			name: "wrong field type",
			data: map[string]interface{}{"id": "1", "name": "Alice"},
			schema: &model.Schema{
				Type: "object",
				Properties: map[string]*model.Schema{
					"id": {Type: "integer"},
				},
			},
			wantViolations: 1, // "id" is string, expected integer
		},
		{
			name: "valid enum",
			data: "admin",
			schema: &model.Schema{
				Type: "string",
				Enum: []interface{}{"admin", "user"},
			},
			wantViolations: 0,
		},
		{
			name: "invalid enum",
			data: "guest",
			schema: &model.Schema{
				Type: "string",
				Enum: []interface{}{"admin", "user"},
			},
			wantViolations: 1,
		},
		{
			name: "valid format email",
			data: "test@example.com",
			schema: &model.Schema{
				Type: "string",
				Format: "email",
			},
			wantViolations: 0,
		},
		{
			name: "invalid format email",
			data: "not-an-email",
			schema: &model.Schema{
				Type: "string",
				Format: "email",
			},
			wantViolations: 1,
		},
		{
			name: "valid array items",
			data: []interface{}{"apple", "banana"},
			schema: &model.Schema{
				Type: "array",
				Items: &model.Schema{Type: "string"},
			},
			wantViolations: 0,
		},
		{
			name: "invalid array items",
			data: []interface{}{"apple", 42},
			schema: &model.Schema{
				Type: "array",
				Items: &model.Schema{Type: "string"},
			},
			wantViolations: 1, // 42 is not a string
		},
		{
			name: "array constraints min/max",
			data: []interface{}{"apple", "banana"},
			schema: &model.Schema{
				Type: "array",
				MinItems: ptrInt(1),
				MaxItems: ptrInt(2),
			},
			wantViolations: 0,
		},
		{
			name: "array constraints max exceeded",
			data: []interface{}{"apple", "banana", "cherry"},
			schema: &model.Schema{
				Type: "array",
				MaxItems: ptrInt(2),
			},
			wantViolations: 1,
		},
		{
			name: "string constraints pattern",
			data: "A123",
			schema: &model.Schema{
				Type: "string",
				Pattern: "^[A-Z][0-9]{3}$",
			},
			wantViolations: 0,
		},
		{
			name: "string constraints length",
			data: "abcd",
			schema: &model.Schema{
				Type: "string",
				MinLength: ptrInt(3),
				MaxLength: ptrInt(5),
			},
			wantViolations: 0,
		},
		{
			name: "numeric constraints",
			data: 10,
			schema: &model.Schema{
				Type: "integer",
				Minimum: ptrFloat(5),
				Maximum: ptrFloat(15),
			},
			wantViolations: 0,
		},
		{
			name: "numeric constraint failure",
			data: 20,
			schema: &model.Schema{
				Type: "integer",
				Maximum: ptrFloat(15),
			},
			wantViolations: 1,
		},
		{
			name: "nullable passes with nil",
			data: nil,
			schema: &model.Schema{
				Type: "string",
				Nullable: true,
			},
			wantViolations: 0,
		},
		{
			name: "not nullable fails with nil",
			data: nil,
			schema: &model.Schema{
				Type: "string",
				Nullable: false,
			},
			wantViolations: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := ValidateSchema(tt.data, tt.schema, "body")
			if len(violations) != tt.wantViolations {
				t.Errorf("ValidateSchema() returned %d violations, want %d", len(violations), tt.wantViolations)
				for i, v := range violations {
					t.Logf("Violation %d: field=%s rule=%s expected=%s actual=%s", i, v.Field, v.Rule, v.Expected, v.Actual)
				}
			}
		})
	}
}
