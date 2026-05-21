# Handoff: RFC-3 Schema Assertions

> **For:** Next AI Agent
> **From:** Faris
> **Date:** May 21, 2026
> **Status:** Ready to implement

---

## What to Build

Reusable YAML schema files that validate response structure — so a user can write:

```yaml
expect:
  status: 200
  schema: users/user-response
```

Instead of asserting every field individually.

Full RFC: `docs/rfcs/3-schema-assertions.md`

---

## Files to Create

### 1. `internal/model/schema.go` — Schema data model

Parse this YAML format:

```yaml
# .gherkio/schemas/users/user-response.yaml
type: object
required:
  - id
  - name
properties:
  id:
    type: integer
  name:
    type: string
  email:
    type: string
    format: email
  role:
    type: string
    enum: [admin, user, moderator]
  tags:
    type: array
    items:
      type: string
  metadata:
    type: object
    properties:
      lastLogin:
        type: string
        format: datetime
```

Struct to define:

```go
package model

type Schema struct {
    Type       string              `yaml:"type"`
    Required   []string            `yaml:"required,omitempty"`
    Properties map[string]*Schema  `yaml:"properties,omitempty"`
    Items      *Schema             `yaml:"items,omitempty"`      // For array item validation
    Format     string              `yaml:"format,omitempty"`     // email, uuid, datetime, uri
    Enum       []interface{}       `yaml:"enum,omitempty"`
    Pattern    string              `yaml:"pattern,omitempty"`
    MinLength  *int                `yaml:"minLength,omitempty"`
    MaxLength  *int                `yaml:"maxLength,omitempty"`
    Minimum    *float64            `yaml:"minimum,omitempty"`
    Maximum    *float64            `yaml:"maximum,omitempty"`
    MinItems   *int                `yaml:"minItems,omitempty"`
    MaxItems   *int                `yaml:"maxItems,omitempty"`
    Nullable   bool                `yaml:"nullable,omitempty"`
}
```

**Important:** Use pointers for `*int` and `*float64` so zero values are distinguishable from "not set".

### 2. `internal/runner/schema.go` — Schema loader + validator

Two responsibilities:

#### a) Schema Loader

```go
// LoadSchema loads a schema by name from the project's schemas directory.
// name example: "users/user-response" or "user-response"
// Resolution: .gherkio/schemas/<name>.yaml, then .gherkio/schemas/<name>.yml
func LoadSchema(name string, projectDir string) (*model.Schema, error)
```

#### b) Schema Validator

```go
// SchemaViolation holds a single validation failure.
type SchemaViolation struct {
    Field    string // e.g. "body.email" or "body.items[2].name"
    Rule     string // e.g. "type", "required", "format", "enum"
    Expected string // e.g. "string", "email", "[admin, user]"
    Actual   string // e.g. "integer", "not-an-email", "superadmin"
}

// ValidateSchema validates parsed response data against a schema.
// Returns all violations (not just first), or nil if valid.
func ValidateSchema(data interface{}, schema *model.Schema, basePath string) []SchemaViolation
```

Implementation approach:

```go
func ValidateSchema(data interface{}, s *model.Schema, basePath string) []SchemaViolation {
    var violations []SchemaViolation

    switch s.Type {
    case "object":
        // 1. Check data is a map
        // 2. Check required fields exist
        // 3. Validate each property recursively
    case "array":
        // 1. Check data is a slice
        // 2. Validate each item against s.Items
        // 3. Check minItems/maxItems
    case "string", "integer", "number", "boolean", "null":
        // 1. Check type matches
        // 2. If string: format, pattern, minLength, maxLength
        // 3. If number/integer: minimum, maximum
        // 4. Check nullable
    }

    // Check enum if present (any type)
    // Check nullable override
}
```

**Key insight:** Reuse the existing `evaluateMatcher()` logic from `internal/runner/matchers.go` for format validation — it already handles `uuid`, `email`, `datetime`. Import and call it directly.

---

## Files to Modify

### 3. `internal/runner/executor.go` — `evaluateAssertion()`

Add handling for `schema:` in the expectation path. In the `evaluateAssertion` function, before or after the existing path handlers, add:

```go
// Schema assertion
if path == "schema" {
    // 1. Call LoadSchema to get the schema
    // 2. Call ValidateSchema to validate resp.Parsed
    // 3. Return a single AssertionResult with violations in Reason
}
```

The `evaluateAssertion` function currently doesn't have access to `projectDir`. You'll need to either:
- Pass it as a parameter (cleaner, but changes the signature)
- Store it in a package-level config
- Use a different approach

**Recommendation:** Change `evaluateAssertion` signature to include a `projectDir string` parameter, or create a wrapper.

### 4. `internal/runner/runner.go` — Pass projectDir to assertion evaluation

When calling `runAssertions()` and `evaluateAssertion()`, pass the project directory so schema files can be resolved.

### 5. `internal/runner/printer.go` — Schema failure output

Schema violations are reported as detail lines under a single assertion:

```
✗ schema = users/user-response
    └─ actual: "not-an-email"
    └─ expected: field body.email format email
    └─ reason: value does not match email pattern

    └─ actual: (missing)
    └─ expected: field body.id is required
    └─ reason: field not present in response
```

The current `AssertionResult` struct uses a single `Reason string`. For schema violations, you have two options:

**Option A (simpler, recommended for MVP):** Concatenate all violations into `Reason` with newlines. The printer already handles multi-line reasons.

**Option B (cleaner):** Add `[]string` field to `AssertionResult` for schema violations. More structured but requires more changes.

Go with **Option A** for MVP — it requires zero struct changes.

### 6. `cmd/init.go` — Update scaffold

When `gherkio init` runs, it should create example schemas:

```
.gherkio/schemas/
└── example/
    ├── login-response.yaml
    └── user-response.yaml
```

With basic schema definitions matching the example test.

---

## Integration Points

| What | Where | How |
|------|-------|-----|
| Load schema | `runner.go` or `executor.go` | Call `LoadSchema(name, projectDir)` |
| Validate response | `executor.go` | Call `ValidateSchema(resp.Parsed, schema, "body")` |
| Report failure | `printer.go` | Format `SchemaViolation` list into assertion reason |
| Reuse matchers | `validator.go` | Call `evaluateMatcher()` for `format` validation |
| Scaffold schemas | `cmd/init.go` | Create example schema files during `gherkio init` |

---

## Existing Patterns to Follow

### Assertion Pattern (from `executor.go`)

The `evaluateAssertion` function already handles `body.X`, `headers.X`, `jwt.X`, `response.X`, `count()`, and `all()`. Add `schema:` in the same function. Follow the same return pattern:

```go
if path == "schema" {
    // ... validate ...
    return AssertionResult{
        Path:     "schema",
        Expected: expectedStr,  // the schema name
        Actual:   fmt.Sprintf("%d fields validated", totalFields),
        Passed:   len(violations) == 0,
        Reason:   formatViolations(violations),
    }
}
```

### Matcher Reuse (from `matchers.go`)

`evaluateMatcher(path, expected, actual)` returns `(AssertionResult, bool)`. You can use it to validate format fields:

```go
result, used := evaluateMatcher(path, "email", actualValue)
if !result.Passed {
    violations = append(violations, SchemaViolation{...})
}
```

### Golden File Testing (from `printer_test.go`)

Add schema-specific golden file tests following the same pattern:

```go
func TestPrintResult_SchemaAssertion(t *testing.T) {
    // Build RunResult with schema assertion
    // Capture stdout
    // Compare against testdata/schema_output.golden
}
```

---

## Test Plan

Same approach as what's already established — pure function tests + golden file tests:

1. **`validator_test.go`** — Test `ValidateSchema`:
   - Valid object with all fields matching
   - Missing required field
   - Wrong type for a field
   - Invalid format (email, uuid, datetime)
   - Enum violation
   - String constraints (minLength, maxLength, pattern)
   - Numeric constraints (minimum, maximum)
   - Array constraints (minItems, maxItems, items)
   - Nested object validation
   - Nullable fields
   - Empty schema matches anything
   - All violations returned (not just first)

2. **`schema_test.go`** — Test `LoadSchema`:
   - Schema file found and parsed
   - Schema file not found
   - Invalid YAML in schema file

3. **Golden file update** — Add to `printer_test.go`:
   - Schema assertion passing
   - Schema assertion failing with single violation
   - Schema assertion failing with multiple violations

---

## Edge Cases to Handle

| Edge Case | Behavior |
|-----------|----------|
| Schema file not found | Fail with clear message: `schema file not found at .gherkio/schemas/users/user-response.yaml` |
| Invalid schema YAML | Fail with parse error |
| Unknown schema rule | Fail with `unsupported rule 'xxx'` |
| Nullable field with null value | Pass (not a violation) |
| Additional properties (not in schema) | Pass (don't fail on extra fields unless specified) |
| Empty properties | Pass all (no rules to validate) |
| Array at root level | Validate as array with items schema |
| Deeply nested property | Recursive validation with path tracking |
| Schema with no required fields | Only validate present fields |
| `enum` with mixed types | Compare actual value against enum list |

---

## Implementation Order

1. **`internal/model/schema.go`** — Schema struct + YAML parsing (can test independently)
2. **`internal/runner/schema.go`** — `LoadSchema` + `ValidateSchema` (can test independently)
3. **Modify `executor.go`** — Add `schema:` handling in `evaluateAssertion`, pass projectDir
4. **Update `printer.go`** — Format schema violation output
5. **Update `cmd/init.go`** — Add example schema files to scaffold
6. **Tests** — `validator_test.go`, golden file updates
7. **Examples** — Schema files in `.gherkio/schemas/`

---

## Reference: Key Existing Files

| File | Purpose |
|------|---------|
| `internal/model/test.go` | Existing YAML model pattern (tags, UnmarshalYAML) |
| `internal/model/environment.go` | Simpler YAML model, good reference |
| `internal/runner/matchers.go` | Reusable `evaluateMatcher()` for format validation |
| `internal/runner/executor.go` | `evaluateAssertion()` — add schema handler here |
| `internal/runner/printer.go` | Output formatting — schema violation display |
| `internal/runner/matchers_test.go` | Pure function test pattern |
| `internal/runner/printer_test.go` | Golden file test pattern |
| `cmd/init.go` | Scaffold — update to create schemas |
