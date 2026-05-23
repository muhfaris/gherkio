# RFC-15: Schema Generator Completeness — Cover All Gherkio YAML Types

**Status:** Draft  
**Author:** Faris  
**Date:** May 23, 2026  
**Depends on:** RFC-14 (convert & step runner) — schema generator should be complete before feature work

---

## 1. Summary

The current `gherkio schema` command generates JSON Schema only for `model.TestFile` (test YAML files). It should generate schemas for **all** Gherkio YAML document types to provide full editor autocomplete and validation.

Additionally, the source of truth in `engine.go` should be expanded to cover **all** runner capabilities so the schema is always in sync with the codebase.

---

## 2. Motivation

### 2.1 Problem

Users edit more than just test files:

```
.gherkio/
├── config.yaml              # model.Config — NO schema
├── credentials/
│   └── local.yaml           # model.Credentials — NO schema
├── environments/
│   └── local.yaml           # model.Environment — NO schema
└── schemas/
    └── login-response.yaml  # model.Schema — NO schema
```

Currently, `gherkio schema` only covers test files. Editing any other Gherkio YAML file gives zero autocomplete or validation.

### 2.2 User Stories

- **As a** developer editing `.gherkio/config.yaml`, **I want to** get autocomplete for `security.mask.enabled`, `reports.format`, etc.
- **As a** developer editing `.gherkio/credentials/local.yaml`, **I want to** get validation that each account has `username` and `password`.
- **As a** developer editing `.gherkio/environments/staging.yaml`, **I want to** get autocomplete for `baseUrl` and `services`.
- **As a** developer writing a schema file, **I want to** get autocomplete for `type`, `properties`, `required`, `format`, etc.

---

## 3. Design

### 3.1 CLI Interface

```bash
# Current — generates only TestFile schema
gherkio schema                                 # → stdout

# New modes:
gherkio schema                                # → all schemas (default)
gherkio schema --type test                    # → only TestFile schema
gherkio schema --type config                  # → only Config schema
gherkio schema --type environment             # → only Environment schema
gherkio schema --type credentials             # → only Credentials schema
gherkio schema --type schema-definition       # → only Schema (definition) schema

# Useful for editor setup:
gherkio schema --list                         # → list available schema types and their file patterns
```

### 3.2 Multi-Schema Output (Default Mode)

By default, `gherkio schema` outputs a JSON object containing all schemas keyed by type:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "definitions": {
    "test": { ... },
    "config": { ... },
    "environment": { ... },
    "credentials": { ... },
    "schema-definition": { ... }
  }
}
```

This allows users to set up their YAML LSP with a single schema file. The LSP resolves the correct sub-schema based on the file path.

### 3.3 Schema Coverage Plan

| Type | Model | jsonschema tags? | Current | Target |
|------|-------|-----------------|---------|--------|
| `test` | `model.TestFile` | ✅ Yes | ✅ Generated | ✅ Keep |
| `config` | `model.Config` | ❌ **None** | ❌ Not generated | ✅ Add tags + generate |
| `environment` | `model.Environment` | ❌ **None** | ❌ Not generated | ✅ Add tags + generate |
| `credentials` | `model.Credentials` | ❌ **None** | ❌ Not generated | ✅ Add tags + generate |
| `schema-definition` | `model.Schema` | ❌ **None** | ❌ Not generated | ✅ Add tags + generate |

### 3.4 Backward Compatibility

- `gherkio schema` with no flags still outputs a schema — but now a **multi-schema container**.
- For users who pipe the output directly to a file for test-file autocomplete, add `--type test` to get exactly what they had before.
- The `--type` flag enables zero-break migration.

---

## 4. Source of Truth Expansion

### 4.1 Current State

`internal/runner/engine.go` exports:

```go
func GetCanonicalPaths() []string      // ["body", "headers", "jwt"]
func GetCollectionFunctions() []string // ["count", "all"]
```

`internal/runner/matchers.go` exports:

```go
func GetAvailableMatchers() []string   // all matcher keywords
```

### 4.2 Missing Capabilities

| Capability | Where it's hardcoded | Should be in `engine.go` |
|------------|---------------------|--------------------------|
| Backoff strategies | `runner.go` / `calculateBackoff()` | `GetBackoffStrategies()` |
| Step roles | `runner.go` / `executeSteps()` | `GetStepRoles()` |
| HTTP methods | `model.Request` struct tag enum | Already in model — fine |
| Retry on-status logic | `runner.go` loop | Already captured by model — fine |

### 4.3 Proposed `engine.go` Additions

```go
// GetBackoffStrategies returns the supported retry backoff strategies.
func GetBackoffStrategies() []string {
    return []string{"constant", "linear", "exponential"}
}

// GetStepRoles returns the supported step lifecycle roles.
func GetStepRoles() []string {
    return []string{"setup", "steps", "teardown"}
}
```

### 4.4 Refactor `isMatcherKeyword()` to Use Source of Truth

Current:

```go
func isMatcherKeyword(expected string) bool {
    parts := strings.SplitN(expected, " ", 2)
    switch parts[0] {
    case "exists", "not":
        return true
    case "uuid", "email", "datetime", "uri", ...
    }
}
```

Target:

```go
func isMatcherKeyword(expected string) bool {
    parts := strings.SplitN(expected, " ", 2)
    for _, matcher := range GetAvailableMatchers() {
        if parts[0] == matcher || (len(parts) > 1 && parts[0]+" "+parts[1] == matcher) {
            return true
        }
    }
    return false
}
```

This eliminates code drift — if a new matcher is added to `GetAvailableMatchers()`, it's automatically recognized as a keyword everywhere.

---

## 5. Model Tag Additions

### 5.1 `model/config.go`

```go
type Config struct {
    Project      ProjectConfig  `yaml:"project,omitempty" jsonschema:"description=Project metadata"`
    Environments EnvConfig      `yaml:"environments,omitempty" jsonschema:"description=Environment configuration"`
    Tests        TestsConfig    `yaml:"tests,omitempty" jsonschema:"description=Test path configuration"`
    Schemas      SchemasConfig  `yaml:"schemas,omitempty" jsonschema:"description=Schema directory path"`
    Security     SecurityConfig `yaml:"security,omitempty" jsonschema:"description=Security and masking configuration"`
    Reports      ReportsConfig  `yaml:"reports,omitempty" jsonschema:"description=Report generation configuration"`
}

type SecurityConfig struct {
    Mask struct {
        Enabled bool     `yaml:"enabled" jsonschema:"description=Enable sensitive field masking"`
        Fields  []string `yaml:"fields,omitempty" jsonschema:"description=Custom field names to mask (case-insensitive)"`
    } `yaml:"mask"`
}

type ReportsConfig struct {
    Path          string `yaml:"path,omitempty" jsonschema:"description=Output path for reports"`
    Format        string `yaml:"format,omitempty" jsonschema:"enum=html,enum=json,description=Report format"`
    Archive       bool   `yaml:"archive,omitempty" jsonschema:"description=Archive previous reports"`
    Retention     int    `yaml:"retention,omitempty" jsonschema:"description=Number of archives to retain"`
    MaskSensitive bool   `yaml:"maskSensitive,omitempty" jsonschema:"description=Mask sensitive data in reports"`
}
```

### 5.2 `model/environment.go`

```go
type Environment struct {
    BaseURL  string             `yaml:"baseUrl" jsonschema:"required,description=Base URL for all requests"`
    Services map[string]Service `yaml:"services,omitempty" jsonschema:"description=Named service base URL overrides"`
}

type Service struct {
    BaseURL string `yaml:"baseUrl" jsonschema:"required,description=Service-specific base URL"`
}
```

### 5.3 `model/credentials.go`

```go
type Credentials struct {
    Accounts map[string]Account `yaml:"accounts" jsonschema:"required,description=Named test accounts"`
}

type Account struct {
    Username string            `yaml:"username" jsonschema:"required,description=Account username"`
    Password string            `yaml:"password" jsonschema:"required,description=Account password"`
    Role     string            `yaml:"role,omitempty" jsonschema:"description=Account role for authorization"`
    Extra    map[string]string `yaml:",inline" jsonschema:"description=Additional account properties"`
}
```

### 5.4 `model/schema.go`

```go
type Schema struct {
    Type       string             `yaml:"type" jsonschema:"required,enum=object,enum=array,enum=string,enum=integer,enum=number,enum=boolean,enum=null,description=Schema type"`
    Required   []string           `yaml:"required,omitempty" jsonschema:"description=Required field names (for object type)"`
    Properties map[string]*Schema `yaml:"properties,omitempty" jsonschema:"description=Field definitions (for object type)"`
    Items      *Schema            `yaml:"items,omitempty" jsonschema:"description=Item schema (for array type)"`
    Format     string             `yaml:"format,omitempty" jsonschema:"enum=email,enum=uuid,enum=datetime,enum=uri,description=Expected string format"`
    Enum       []interface{}      `yaml:"enum,omitempty" jsonschema:"description=List of allowed values"`
    Pattern    string             `yaml:"pattern,omitempty" jsonschema:"description=Regex pattern for string validation"`
    MinLength  *int               `yaml:"minLength,omitempty" jsonschema:"description=Minimum string length"`
    MaxLength  *int               `yaml:"maxLength,omitempty" jsonschema:"description=Maximum string length"`
    Minimum    *float64           `yaml:"minimum,omitempty" jsonschema:"description=Minimum numeric value"`
    Maximum    *float64           `yaml:"maximum,omitempty" jsonschema:"description=Maximum numeric value"`
    MinItems   *int               `yaml:"minItems,omitempty" jsonschema:"description=Minimum array length"`
    MaxItems   *int               `yaml:"maxItems,omitempty" jsonschema:"description=Maximum array length"`
    Nullable   bool               `yaml:"nullable,omitempty" jsonschema:"description=Allow null values"`
}
```

---

## 6. Schema Generator Code Changes

### 6.1 `internal/schema/generator.go` — Multi-Schema Support

Current: `GenerateJSONSchema()` reflects only `model.TestFile`.

Target:

```go
// GenerateJSONSchema generates JSON schemas for all Gherkio YAML document types.
func GenerateJSONSchema() ([]byte, error) {
    return generateAllSchemas()
}

// GenerateJSONSchemaForType generates a JSON schema for a specific type.
func GenerateJSONSchemaForType(schemaType string) ([]byte, error) {
    // ...
}
```

The `generateAllSchemas()` function:

```go
func generateAllSchemas() ([]byte, error) {
    r := new(jsonschema.Reflector)
    r.RequiredFromJSONSchemaTags = true

    // Build individual schemas for each document type
    testSchema := r.Reflect(&model.TestFile{})
    configSchema := r.Reflect(&model.Config{})
    envSchema := r.Reflect(&model.Environment{})
    credSchema := r.Reflect(&model.Credentials{})
    schemaDefSchema := r.Reflect(&model.Schema{})

    // Apply Expect patching only to test schema (same as current logic)
    patchExpectSchema(testSchema)

    // Combine into a single output
    combined := map[string]interface{}{
        "$schema": "http://json-schema.org/draft-07/schema#",
        "definitions": map[string]interface{}{
            "test":              testSchema,
            "config":            configSchema,
            "environment":       envSchema,
            "credentials":       credSchema,
            "schema-definition": schemaDefSchema,
        },
    }

    return json.MarshalIndent(combined, "", "  ")
}
```

### 6.2 Extracted `patchExpectSchema()` Function

The current Expect-patching logic (adding PatternProperties for `body.X`, `headers.X`, etc.) should be extracted into its own function so it's applied only to the test schema:

```go
func patchExpectSchema(schema *jsonschema.Schema) {
    if expectSchema, ok := schema.Definitions["Expect"]; ok {
        // ... current patching logic ...
    }
}
```

### 6.3 `cmd/schema.go` — Add `--type` and `--list` Flags

```go
var (
    schemaType string
    listTypes  bool
)

var schemaCmd = &cobra.Command{
    Use:   "schema",
    Short: "Generate JSON Schema for Gherkio YAML files",
    RunE: func(cmd *cobra.Command, args []string) error {
        if listTypes {
            return printSchemaTypes()
        }
        return generateSchema(schemaType)
    },
}

func init() {
    rootCmd.AddCommand(schemaCmd)
    schemaCmd.Flags().StringVar(&schemaType, "type", "", "Schema type: test, config, environment, credentials, schema-definition")
    schemaCmd.Flags().BoolVar(&listTypes, "list", false, "List available schema types")
}
```

### 6.4 Backward-Compatible Output

When `--type` is **not** specified, the output is a multi-schema container. Users who currently run:

```bash
gherkio schema > gherkio-schema.json
```

Will get a different JSON structure. To avoid breakage:

```bash
# Legacy behavior (single test schema):
gherkio schema --type test > gherkio-schema.json

# New behavior (all schemas):
gherkio schema > gherkio-schemas.json
```

Document this change clearly.

---

## 7. Documentation Fixes

### 7.1 README — Add `uri` Matcher

The `uri` matcher is implemented in `matchers.go` but missing from the README's type matchers table. Add:

```yaml
expect:
  body.avatar: uri         # Valid URI format (e.g. https://example.com/avatar.png)
```

### 7.2 `llm-map.txt` — Update Matchers Section

The matchers section at line 330 is severely outdated — it only lists `exists` and literal values. Update it to list all 18+ matchers with their behavior.

---

## 8. Implementation Order

| Step | What | Why |
|------|------|-----|
| 1 | Add `jsonschema` tags to `config.go`, `environment.go`, `credentials.go`, `schema.go` | Foundation — without tags, generated schemas are bare |
| 2 | Add `GetBackoffStrategies()` and `GetStepRoles()` to `engine.go` | Complete the source of truth |
| 3 | Refactor `isMatcherKeyword()` to use `GetAvailableMatchers()` | Eliminate code drift |
| 4 | Refactor `generator.go` to support multi-schema output | Core feature |
| 5 | Add `--type` / `--list` flags to `cmd/schema.go` | CLI interface |
| 6 | Fix README — add `uri` matcher | Documentation gap |
| 7 | Fix `llm-map.txt` — update matchers section | Documentation gap |

---

## 9. Open Questions

### 9.1. Should we use `oneOf` for Step's `request` / `use` mutual exclusion?

Currently the schema makes both optional and doesn't express the constraint. Using `oneOf` would make the editor reject invalid steps:
- Step with neither `request` nor `use` → ❌ invalid
- Step with both `request` and `use` → ❌ invalid
- Step with exactly one → ✅ valid

This requires manual schema patching (the `jsonschema` reflector doesn't generate `oneOf` from Go structs).

**Recommendation:** Include in this RFC. It's a validation improvement that directly prevents user errors.

### 9.2. Should the multi-schema output use `$id` for file-path-based resolution?

Some YAML LSPs support `$id` fields to associate a sub-schema with a file glob pattern. For example:

```json
{
    "definitions": {
        "test": {
            "$id": "gherkio://schemas/test",
            "description": "Schema for .gherkio/tests/**/*.yaml"
        },
        "config": {
            "$id": "gherkio://schemas/config",
            "description": "Schema for .gherkio/config.yaml"
        }
    }
}
```

This depends on the LSP's capabilities. **Not required for MVP** — users can set up file associations manually in their editor settings.

### 9.3. Should `$ref` resolution work between sub-schemas?

For example, the `schema-definition` sub-schema references `model.Schema`, which is also referenced by `model.Expect` in the test schema. If a user edits both test files and schema definition files, having cross-references would be useful.

**Recommendation:** Skip for MVP. The schemas are small enough that duplication is acceptable.

---

## 10. Appendix

### 10.1 Example: Multi-Schema Output Structure

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "definitions": {
    "test": {
      "$ref": "#/$defs/TestFile",
      "$defs": {
        "TestFile": { ... },
        "Step": { ... },
        "Request": { ... },
        "Expect": { ... },
        "RetryConfig": { ... },
        "TimingConfig": { ... }
      }
    },
    "config": {
      "$ref": "#/$defs/Config",
      "$defs": {
        "Config": { ... },
        "ProjectConfig": { ... },
        "SecurityConfig": { ... },
        "ReportsConfig": { ... }
      }
    },
    "environment": {
      "$ref": "#/$defs/Environment",
      "$defs": {
        "Environment": { ... },
        "Service": { ... }
      }
    },
    "credentials": {
      "$ref": "#/$defs/Credentials",
      "$defs": {
        "Credentials": { ... },
        "Account": { ... }
      }
    },
    "schema-definition": {
      "$ref": "#/$defs/Schema",
      "$defs": {
        "Schema": { ... }
      }
    }
  }
}
```

### 10.2 Example: Neovim LSP Configuration

After this RFC, a user can configure `yaml-language-server`:

```lua
-- ~/.config/nvim/init.lua or through lspconfig
require('lspconfig').yamlls.setup({
  settings = {
    yaml = {
      schemas = {
        ["gherkio-schemas.json#/definitions/test"] = {
          ".gherkio/tests/**/*.yaml",
        },
        ["gherkio-schemas.json#/definitions/config"] = {
          ".gherkio/config.yaml",
        },
        ["gherkio-schemas.json#/definitions/environment"] = {
          ".gherkio/environments/*.yaml",
        },
        ["gherkio-schemas.json#/definitions/credentials"] = {
          ".gherkio/credentials/*.yaml",
        },
        ["gherkio-schemas.json#/definitions/schema-definition"] = {
          ".gherkio/schemas/**/*.yaml",
        },
      },
    },
  },
})
```

### 10.3 Step `oneOf` Implementation Detail

The `oneOf` constraint for Step must be applied manually after reflection:

```go
if stepSchema, ok := schema.Definitions["Step"]; ok {
    stepSchema.OneOf = []*jsonschema.Schema{
        {
            Required: []string{"request"},
            Title:    "Step with HTTP request",
        },
        {
            Required: []string{"use"},
            Title:    "Step composing another scenario",
        },
    }
}
```

This ensures editor validation catches invalid steps at write-time rather than run-time.
