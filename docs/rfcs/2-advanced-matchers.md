# RFC-2: Advanced Matchers

> **Status:** Ready 
> **Author:** Faris
> **Date:** May 21, 2026

---

## 1. Summary

Expand the assertion engine with structured matchers beyond the current `exists` and equality checks. This enables type validation, format validation, collection assertions, and length assertions — all within the existing declarative DSL.

---

## 2. Motivation

Currently, users can only assert:

```yaml
expect:
  status: 200
  body.token: exists
  body.role: admin          # equality only
```

To validate that `body.email` is actually an email, or that `body.id` is a valid UUID, users must rely on external tools or write imperative code. This directly contradicts the PRD's assertion-centric philosophy (section 12.1).

Real-world assertions Gherkio can't express today:

| Need | Current workaround | Problem |
|------|-------------------|---------|
| `body.email: email` | Can't assert | No validation |
| `body.id: uuid` | Can't assert | No validation |
| `body.items[0].name: string` | Can't assert type | No type safety |
| `all(response.items.status): active` | Can't assert all elements | No collection validation |
| `count(response.items): 10` | Can't assert length | No size validation |

---

## 3. Design

### 3.1 Matcher Syntax

Matchers are string keywords used as assertion values:

```yaml
expect:
  body.id: uuid
  body.email: email
  body.createdAt: datetime
  body.name: string
  body.count: number
  body.isActive: boolean
  body.tags: array
  body.meta: object
  body.nullable: null
  body.flag: true
  body.completed: false
```

### 3.2 String Matchers

```yaml
expect:
  body.email: email        # valid email format
  body.name: contains Laptop  # substring match
  body.name: startsWith Pre   # prefix match
  body.name: endsWith ium     # suffix match
  body.code: regex ^[A-Z]{3}$ # regex pattern
```

### 3.3 Collection Matchers

```yaml
expect:
  # All elements satisfy a condition
  all(response.items.status): active

  # Array length
  count(response.items): 10

  # Array is not empty (exists covers this, but explicit is clearer)
  body.items: exists
```

### 3.4 Type Assertions

```yaml
expect:
  body.id: number
  body.name: string
  body.isActive: boolean
  body.tags: array
  body.meta: object
  body.nullField: null
```

### 3.5 Implementation

Add an `evaluateMatcher()` function in `executor.go` that switches on the matcher keyword:

```
"exists"    → already implemented
"uuid"      → regex match ^[0-9a-f]{8}-...
"email"     → regex match ^[^@]+@[^@]+\.[^@]+$
"datetime"  → parse RFC3339 / ISO8601
"number"    → json.Number or float64 type check
"string"    → string type check
"boolean"   → bool type check
"array"     → []interface{} type check
"object"    → map[string]interface{} type check
"null"      → nil check
"true"      → bool true
"false"     → bool false
"contains"  → substring match (value: "contains <text>")
"startsWith"→ prefix match
"endsWith"  → suffix match
"regex"     → regex pattern match
"count"     → array length check
"all"       → all elements match condition
```

---

## 4. Edge Cases

### 4.1 Matcher Priority

Matchers that overlap with existing patterns (e.g., `body.token: exists`) should continue to work as-is. The matcher system only activates when the value is a known matcher keyword.

### 4.2 Invalid Matcher

If a matcher keyword is unrecognized, fall back to equality comparison (current behavior). This preserves backward compatibility.

### 4.3 Nested `all()` Paths

`all(response.items.status)` should resolve `response.items` to an array, then check each element's `status` field against the matcher.

---

## 5. Implementation Plan

### Phase 1 — Type Matchers

- [ ] Add `evaluateMatcher(path string, matcher string, actual interface{}) AssertionResult`
- [ ] Implement: `uuid`, `email`, `datetime`, `number`, `string`, `boolean`, `array`, `object`, `null`, `true`, `false`
- [ ] Modify `evaluateAssertion` to detect matcher keywords vs equality

### Phase 2 — String & Collection Matchers

- [ ] Implement: `contains`, `startsWith`, `endsWith`, `regex`
- [ ] Implement: `count()` — resolves path to array, checks length
- [ ] Implement: `all()` — resolves path to array, checks each element

### Phase 3 — Reporting

- [ ] Show matcher name in assertion output: `✓ body.email = email (actual: admin@test.com)`

---

## 6. Example Output

### 6.1 Type Matchers — Passing

```yaml
expect:
  status: 200
  body.id: uuid
  body.email: email
  body.name: string
```

```
  ✓ status = 200
  ✓ body.id = uuid (actual: 550e8400-e29b-41d4-a716-446655440000)
  ✓ body.email = email (actual: emily.johnson@x.dummyjson.com)
  ✓ body.name = string (actual: Emily)
```

### 6.2 Type Matchers — Failing

```yaml
expect:
  body.id: uuid
  body.email: email
  body.isActive: boolean
```

Failing response:
```json
{
  "id": 42,
  "email": "email.com",
  "isActive": "yes"
}
```

```
  ✗ body.id = uuid
      └─ actual: 42
      └─ expected: valid UUID format
      └─ reason: value is not a string

  ✗ body.email = email
      └─ actual: "email.com"
      └─ expected: valid email format
      └─ reason: missing local-part before '@'

  ✗ body.isActive = boolean
      └─ actual: "yes"
      └─ expected: boolean (true/false)
      └─ reason: string cannot be coerced to boolean
```

### 6.3 String Matchers — Passing

```yaml
expect:
  body.name: contains Laptop
  body.slug: startsWith item-
  body.status: endsWith ed
  body.code: regex ^[A-Z]{3}$
```

```
  ✓ body.name contains "Laptop" (actual: "Laptop Baru")
  ✓ body.slug startsWith "item-" (actual: "item-42")
  ✓ body.status endsWith "ed" (actual: "completed")
  ✓ body.code regex "^[A-Z]{3}$" (actual: "ABC")
```

### 6.4 String Matchers — Failing

```yaml
expect:
  body.name: contains Laptop
  body.code: regex ^[A-Z]{3}$
```

```
  ✗ body.name contains "Laptop"
      └─ actual: "Smartphone"
      └─ expected: contains substring "Laptop"
      └─ reason: substring not found at any position

  ✗ body.code regex "^[A-Z]{3}$"
      └─ actual: "abcd"
      └─ expected: pattern ^[A-Z]{3}$
      └─ reason: no match at position 0
```

### 6.5 Collection Matchers — Passing

```yaml
expect:
  count(body.items): 3
  all(body.items.status): active
```

```
  ✓ count(body.items) = 3 (actual: 3)
  ✓ all(body.items.status) = active (actual: [active, active, active])
```

### 6.6 Collection Matchers — Failing

```yaml
expect:
  count(body.items): 3
  all(body.items.status): active
```

Failing response:
```json
{
  "items": [
    { "status": "active" },
    { "status": "inactive" },
    { "status": "active" },
    { "status": "pending" }
  ]
}
```

```
  ✗ count(body.items) = 3
      └─ actual: 4
      └─ expected: exactly 3 items
      └─ reason: array has 4 items

  ✗ all(body.items.status) = active
      └─ actual: [active, inactive, active, pending]
      └─ expected: all elements equal "active"
      └─ reason: failed at index 1 (got "inactive")
      └─ reason: failed at index 3 (got "pending")
```

---

## 7. Open Questions

1. Should matchers support case-insensitive matching for `contains`, `startsWith`, `endsWith`?
2. Should custom matchers be configurable (part of the capability system)?
3. Should `all()` support nested matchers (e.g., `all(items.price): number`)?
