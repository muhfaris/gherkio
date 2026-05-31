package runner

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/muhfaris/gherkio/internal/model"
)

// SchemaViolation holds a single validation failure.
type SchemaViolation struct {
	Field    string // e.g. "body.email" or "body.items[2].name"
	Rule     string // e.g. "type", "required", "format", "enum"
	Expected string // e.g. "string", "email", "[admin, user]"
	Actual   string // e.g. "integer", "not-an-email", "superadmin"
}

// ValidateSchema validates parsed response data against a schema.
func ValidateSchema(data interface{}, schema *model.Schema, basePath string) []SchemaViolation {
	var violations []SchemaViolation

	if schema == nil {
		return violations // Empty schema passes everything
	}

	// Handle nil data
	if data == nil {
		if schema.Nullable {
			return violations
		}
		violations = append(violations, SchemaViolation{
			Field:    basePath,
			Rule:     "nullable",
			Expected: "not null",
			Actual:   "null",
		})
		return violations
	}

	// Type validation and structural checks
	violations = append(violations, validateTypeAndStructure(data, schema, basePath)...)

	// Value constraints (only if data is present)
	violations = append(violations, validateConstraints(data, schema, basePath)...)

	return violations
}

func validateTypeAndStructure(data interface{}, schema *model.Schema, basePath string) []SchemaViolation {
	var violations []SchemaViolation

	switch schema.Type {
	case "object":
		m, ok := data.(map[string]interface{})
		if !ok {
			return []SchemaViolation{{
				Field:    basePath,
				Rule:     "type",
				Expected: "object",
				Actual:   getTypeName(data),
			}}
		}

		// Required fields
		for _, req := range schema.Required {
			if val, exists := m[req]; !exists || val == nil {
				violations = append(violations, SchemaViolation{
					Field:    fmt.Sprintf("%s.%s", basePath, req),
					Rule:     "required",
					Expected: fmt.Sprintf("field %s.%s is required", basePath, req),
					Actual:   "(missing)",
				})
			}
		}

		// Properties
		if schema.Properties != nil {
			for propName, propSchema := range schema.Properties {
				if val, exists := m[propName]; exists {
					propPath := fmt.Sprintf("%s.%s", basePath, propName)
					violations = append(violations, ValidateSchema(val, propSchema, propPath)...)
				}
			}
		}

	case "array":
		arr, ok := data.([]interface{})
		if !ok {
			return []SchemaViolation{{
				Field:    basePath,
				Rule:     "type",
				Expected: "array",
				Actual:   getTypeName(data),
			}}
		}

		// Items validation
		if schema.Items != nil {
			for i, item := range arr {
				itemPath := fmt.Sprintf("%s[%d]", basePath, i)
				violations = append(violations, ValidateSchema(item, schema.Items, itemPath)...)
			}
		}

	case "string", "integer", "number", "boolean", "null":
		actualType := getTypeName(data)
		if !typesMatch(schema.Type, actualType) {
			violations = append(violations, SchemaViolation{
				Field:    basePath,
				Rule:     "type",
				Expected: schema.Type,
				Actual:   actualType,
			})
		}
	}

	return violations
}

func validateConstraints(data interface{}, schema *model.Schema, basePath string) []SchemaViolation {
	var violations []SchemaViolation

	// Enum
	if len(schema.Enum) > 0 {
		matched := false
		strData := fmt.Sprintf("%v", data)
		var allowedStrs []string

		for _, enumVal := range schema.Enum {
			allowedStrs = append(allowedStrs, fmt.Sprintf("%v", enumVal))
			if fmt.Sprintf("%v", enumVal) == strData { // Simplistic equality comparison
				matched = true
				break
			}
		}
		if !matched {
			violations = append(violations, SchemaViolation{
				Field:    basePath,
				Rule:     "enum",
				Expected: fmt.Sprintf("[%s]", strings.Join(allowedStrs, ", ")),
				Actual:   strData,
			})
		}
	}

	// Format
	if schema.Format != "" {
		strData, ok := data.(string)
		if ok {
			result, _ := evaluateMatcher(basePath, schema.Format, strData)
			if !result.Passed {
				violations = append(violations, SchemaViolation{
					Field:    basePath,
					Rule:     "format",
					Expected: fmt.Sprintf("field %s format %s", basePath, schema.Format),
					Actual:   strData,
				})
			}
		}
	}

	// Pattern
	if schema.Pattern != "" {
		strData, ok := data.(string)
		if ok {
			matched, err := regexp.MatchString(schema.Pattern, strData)
			if err != nil || !matched {
				violations = append(violations, SchemaViolation{
					Field:    basePath,
					Rule:     "pattern",
					Expected: fmt.Sprintf("match pattern %s", schema.Pattern),
					Actual:   strData,
				})
			}
		}
	}

	// MinLength / MaxLength
	if strData, ok := data.(string); ok {
		length := len(strData) // Note: this is byte length, for runes use utf8.RuneCountInString
		if schema.MinLength != nil && length < *schema.MinLength {
			violations = append(violations, SchemaViolation{
				Field:    basePath,
				Rule:     "minLength",
				Expected: fmt.Sprintf("minLength %d", *schema.MinLength),
				Actual:   fmt.Sprintf("%d", length),
			})
		}
		if schema.MaxLength != nil && length > *schema.MaxLength {
			violations = append(violations, SchemaViolation{
				Field:    basePath,
				Rule:     "maxLength",
				Expected: fmt.Sprintf("maxLength %d", *schema.MaxLength),
				Actual:   fmt.Sprintf("%d", length),
			})
		}
	}

	// Minimum / Maximum
	if numData, isNum := toFloat64(data); isNum {
		if schema.Minimum != nil && numData < *schema.Minimum {
			violations = append(violations, SchemaViolation{
				Field:    basePath,
				Rule:     "minimum",
				Expected: fmt.Sprintf("minimum %v", *schema.Minimum),
				Actual:   fmt.Sprintf("%v", numData),
			})
		}
		if schema.Maximum != nil && numData > *schema.Maximum {
			violations = append(violations, SchemaViolation{
				Field:    basePath,
				Rule:     "maximum",
				Expected: fmt.Sprintf("maximum %v", *schema.Maximum),
				Actual:   fmt.Sprintf("%v", numData),
			})
		}
	}

	// MinItems / MaxItems
	if arrData, ok := data.([]interface{}); ok {
		length := len(arrData)
		if schema.MinItems != nil && length < *schema.MinItems {
			violations = append(violations, SchemaViolation{
				Field:    basePath,
				Rule:     "minItems",
				Expected: fmt.Sprintf("minItems %d", *schema.MinItems),
				Actual:   fmt.Sprintf("%d", length),
			})
		}
		if schema.MaxItems != nil && length > *schema.MaxItems {
			violations = append(violations, SchemaViolation{
				Field:    basePath,
				Rule:     "maxItems",
				Expected: fmt.Sprintf("maxItems %d", *schema.MaxItems),
				Actual:   fmt.Sprintf("%d", length),
			})
		}
	}

	return violations
}

// toFloat64 is a helper to reliably convert numeric types to float64
func toFloat64(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f, true
		}
		return 0, false
	default:
		return 0, false
	}
}

func getTypeName(data interface{}) string {
	if data == nil {
		return "null"
	}
	switch data.(type) {
	case string:
		return "string"
	case int, int8, int16, int32, int64, float32, float64, json.Number:
		return "number" // YAML parser usually unmarshals into float64 or int
	case bool:
		return "boolean"
	case map[string]interface{}:
		return "object"
	case []interface{}:
		return "array"
	default:
		return fmt.Sprintf("%T", data)
	}
}

func typesMatch(expected string, actual string) bool {
	if expected == "number" && (actual == "number" || actual == "integer") {
		return true
	}
	if expected == "integer" && actual == "number" { // Because Go yaml unmarshals to float64 often
		return true // Further validation needed to ensure it's a whole number if strictly required
	}
	return expected == actual
}
