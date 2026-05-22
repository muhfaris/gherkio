# RFC-11: Negative Assertions (not exists, schema: not)

> **Status:** Draft
> **Author:** Faris
> **Date:** May 22, 2026

---

## 1. Summary

Add negative/negation assertions to the DSL, enabling users to assert that a field does **not** exist or a schema does **not** match. This eliminates the need for intentional-failure test cases and aligns the DSL with real-world testing needs.

---

## 2. Motivation

Currently, Gherkio only supports **positive** assertions:

```yaml
expect:
  body.id: exists          # "field MUST exist"
  schema: user-response    # "response MUST match schema"
```

There is no way to express:

- "This field should NOT exist" after a soft delete
- "This deprecated field should NOT appear in the response"
- "This schema should NOT match" (e.g. product data should not match the login schema)
- "This field should be absent until explicitly created"

Users currently work around this by:

1. **Writing intentional-failure tests** — knowing an assertion will fail, then checking the error output manually. This makes the scenario show `✗ FAIL` even though nothing is broken.
2. **Omitting the assertion entirely** — silently ignoring the unwanted field.
3. **Using external scripts** — defeating the purpose of the DSL.

The PRD (§12) states assertions should be "readable, declarative, structured, reportable." Negative assertions follow all four principles without introducing unrestricted logic.

---

## 3. Design

### 3.1 `not exists` — Field Absence Assertion

```yaml
expect:
  body.deletedAt: not exists
  body.nonExistent: not exists
```

**Semantics:** The field must NOT be present in the response body (or JWT claims, headers). If the field exists, the assertion fails.

**Supported path prefixes:**

| Path | Behavior |
|------|----------|
| `body.X: not exists` | Field X must not exist in response body |
| `headers.X: not exists` | Header X must not be present |
| `jwt.X: not exists` | Claim X must not exist in decoded JWT |

### 3.2 `schema: not <name>` — Schema Mismatch Assertion

```yaml
expect:
  schema: not example/login-response
```

**Semantics:** The response body must **fail** validation against the named schema. If the schema validates successfully, the assertion fails. If it fails validation (any violation), the assertion passes.

This is the inverse of `schema: example/login-response`.

### 3.3 Implementation

#### `not exists` in `evaluateMatcher()`

Add a new matcher keyword `not exists` to `internal/runner/matchers.go`:

```go
case "not":
    if len(parts) < 2 || parts[1] != "exists" {
        return AssertionResult{}, false
    }
    // "not exists" — field must NOT be present
    // This is handled before evaluateMatcher is called,
    // so actual will be nil-ish if the field wasn't found
    if actual == nil {
        return AssertionResult{
            Path:     path,
            Expected: "not exists",
            Actual:   "(not found)",
            Passed:   true,
        }, true
    }
    return AssertionResult{
        Path:     path,
        Expected: "not exists",
        Actual:   fmt.Sprintf("%v", actual),
        Passed:   false,
    }, true
```

Wait — the flow is wrong. By the time `evaluateMatcher()` is called, the field has already been resolved. For `body.X: not exists`, if `X` doesn't exist, `evaluateAssertion` returns early with "not found". So the `not exists` logic needs to be handled **before** the path resolution, in `evaluateAssertion()`.

**Revised approach:** In `evaluateAssertion()`, detect `not exists` early:

```go
if expectedStr == "not exists" {
    // Try to resolve the path
    // (path resolution logic here, same as for "exists" but inverted)
    // ...
    // If found → fail (expected it to be absent)
    // If not found → pass (confirmed absence)
}
```

#### `schema: not <name>` in `evaluateAssertion()`

In the schema assertion block, add handling for `not` prefix:

```go
if path == "schema" {
    expectedStr, ok := expected.(string)
    
    isNegated := false
    if strings.HasPrefix(expectedStr, "not ") {
        isNegated = true
        expectedStr = strings.TrimPrefix(expectedStr, "not ")
    }
    
    // ... load and validate schema ...
    
    if isNegated {
        // Schema should NOT match — invert the result
        if len(violations) == 0 {
            // Schema matched — assertion fails
            return AssertionResult{
                Path:     "schema",
                Expected: fmt.Sprintf("not %s", expectedStr),
                Actual:   "valid (unexpectedly)",
                Passed:   false,
                Reason:   "response matches schema but should not",
            }
        }
        // Schema didn't match — assertion passes
        return AssertionResult{
            Path:     "schema",
            Expected: fmt.Sprintf("not %s", expectedStr),
            Actual:   "invalid (expected)",
            Passed:   true,
        }
    }
    // ... normal schema assertion logic ...
}
```

---

## 4. Example Output

### `not exists` — passing (field absent)

```
✓ body.deletedAt: not exists
```

### `not exists` — failing (field present unexpectedly)

```
✗ body.email: not exists
    └─ got: user@example.com
```

### `schema: not` — passing (response doesn't match)

```
✓ schema = not example/login-response (actual: invalid)
```

### `schema: not` — failing (response unexpectedly matches)

```
✗ schema = not features/product-response (actual: valid)
    └─ reason: response matches schema but should not
```

---

## 5. Demo Scenario (Replaces Intentional-Failure Tests)

The existing `demo/schema-formatting.yaml` can be rewritten with proper passing assertions:

```yaml
scenario: negative assertions demo

steps:
  # Product data should NOT match login schema
  - request:
      method: GET
      url: /products/1
    expect:
      status: 200
      schema: not example/login-response

  # Product data SHOULD match product schema
  - request:
      method: GET
      url: /products/1
    expect:
      status: 200
      schema: features/product-response

  # Products don't have a deletedAt field
  - request:
      method: GET
      url: /products/1
    expect:
      status: 200
      body.deletedAt: not exists
```

This scenario would show `✓ PASS` — all assertions test the correct behavior without intentional failures.

---

## 6. Implementation Plan

- [ ] Add `not exists` handling in `evaluateAssertion()` in `internal/runner/executor.go`
  - Detect `expectedStr == "not exists"` before path resolution
  - Invert pass/fail logic: found → fail, not found → pass
- [ ] Add `schema: not <name>` handling in the schema assertion block
  - Parse `not ` prefix from schema name
  - Invert pass/fail logic based on validation result
- [ ] Update `isMatcherKeyword()` in `internal/runner/matchers.go` to include `not exists`
- [ ] Update printer formatting if needed for the new assertion types
- [ ] Update demo test files to use proper negative assertions

---

## 7. Decisions

### 7.1 `not exists` as a single keyword, not a separate `not:` key

Using `not exists` keeps the existing `key: value` structure intact. A separate `not:` YAML key would require structural changes to the `Expect` model and custom unmarshaling.

### 7.2 `schema: not <name>` uses a string prefix

Following the same `key: value` pattern as the positive assertion. The `not ` prefix is simple, readable, and easy to parse. A separate `schema_not:` field would double the model surface area.

### 7.3 Negation is limited to `exists` and `schema`

Not every matcher needs a negated form. `not uuid` or `not email` would test the absence of a type, which is rarely useful. Starting with `exists` and `schema` covers the most common cases. Other negations can be added later if users request them.

---

## 8. Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Ambiguity with `not exists` vs `body.not: exists` | YAML key parsing is unchanged. `not exists` is a matcher value, not a YAML key. |
| Users write `not` without `exists` (e.g. `body.id: not`) | `isMatcherKeyword("not")` returns false → falls through to equality check → `"not" != "1"` → fails as a normal mismatch |
| `schema: not <name>` conflicts with schema names starting with "not" | Schema names starting with "not " would be impossible — schema files are named `<name>.yaml`, not "not <name>.yaml" |
