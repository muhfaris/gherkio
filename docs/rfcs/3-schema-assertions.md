# RFC-3: Schema Assertions

> **Status:** Ready
> **Author:** Faris
> **Date:** May 21, 2026

---

## 1. Summary

Enable structured response validation against reusable JSON Schema definitions stored in the project. This moves assertion logic from individual field checks to high-level shape validation.

---

## 2. Motivation

Currently, users must assert every field individually:

```yaml
expect:
  status: 200
  body.id: exists
  body.name: exists
  body.email: exists
  body.role: exists
  body.createdAt: exists
```

For a response with 15+ fields, this becomes verbose and fragile. Schema assertions allow:

```yaml
expect:
  status: 200
  schema: user-response
```

Where `user-response` is a reusable schema file. This also enables:
- Required field validation
- Type validation
- Pattern validation
- Nested object validation
- Array item validation

---

## 3. Design

### 3.1 Schema Files

Schemas live in `.gherkio/schemas/` and are organized by domain:

```
.gherkio/schemas/
├── auth/
│   ├── login-response.yaml
│   └── profile-response.yaml
├── users/
│   ├── user-response.yaml
│   └── user-list-response.yaml
└── items/
    ├── item-response.yaml
    └── item-list-response.yaml
```

### 3.2 Schema Format

Schemas use a lightweight YAML format (not full JSON Schema, to keep the DSL constrained and readable):

```yaml
# .gherkio/schemas/users/user-response.yaml
type: object
required:
  - id
  - name
  - email
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
  createdAt:
    type: string
    format: datetime
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

### 3.3 Collection (Array) Schemas

For API endpoints that return arrays at the root:

```yaml
# .gherkio/schemas/users/user-list-response.yaml
type: array
items:
  type: object
  required:
    - id
    - name
  properties:
    id:
      type: integer
    name:
      type: string
```

```yaml
expect:
  status: 200
  schema: users/user-list-response
```

### 3.4 Mixing Schema with Individual Assertions

Schema validation and individual field assertions can be mixed. The schema validates the overall shape, while individual assertions add extra constraints not covered by the schema:

```yaml
expect:
  status: 200
  schema: users/user-response    # shape validation
  body.customFlag: true          # extra assertion beyond schema
  body.timestamp: exists         # field not in schema
```

### 3.5 Usage

```yaml
# In a test file
- request:
    method: GET
    url: /users/me
    headers:
      Authorization: Bearer $token

  expect:
    status: 200
    schema: users/user-response
```

Resolution order for `schema: users/user-response`:

```
1. .gherkio/schemas/users/user-response.yaml
2. .gherkio/schemas/users/user-response.yml
```

Root-level schemas can also be referenced without a directory prefix:

```yaml
expect:
  schema: user-response
# resolves to .gherkio/schemas/user-response.yaml
```

### 3.6 Supported Schema Rules

| Rule | Example | Description |
|------|---------|-------------|
| `type` | `type: string` | Validates type (string, integer, number, boolean, array, object, null) |
| `required` | `required: [id, name]` | Fields that must exist |
| `properties` | `properties: { ... }` | Field-level validation |
| `items` | `items: { type: string }` | Array item validation |
| `format` | `format: email` | Format validation (email, uuid, datetime, uri) |
| `enum` | `enum: [admin, user]` | Allowed values |
| `pattern` | `pattern: "^[A-Z]"` | Regex pattern |
| `minLength` / `maxLength` | `minLength: 1` | String length bounds |
| `minimum` / `maximum` | `minimum: 0` | Numeric bounds |
| `minItems` / `maxItems` | `maxItems: 100` | Array length bounds |
| `nullable` | `type: string, nullable: true` | Allows null |

### 3.7 Reporting

Failure output shows which field and rule failed:

```
✗ schema = users/user-response
    └─ actual: "not-an-email"
    └─ expected: field body.email type string
    └─ reason: got number

✗ schema = users/user-response
    └─ actual: (missing)
    └─ expected: field body.id is required
    └─ reason: field not present in response
```

---

## 4. Edge Cases

### 4.1 Schema Not Found

If the referenced schema file doesn't exist, the assertion fails with a clear message:

```
✗ schema = users/user-response
    └─ actual: (schema file not found)
    └─ expected: .gherkio/schemas/users/user-response.yaml
    └─ reason: file does not exist
```

### 4.2 Schema File Error

If the schema YAML is invalid or contains unsupported rules, fail with a parse error:

```
✗ schema = users/user-response
    └─ schema error: unsupported rule 'uniqueItems'
```

### 4.3 Empty Schema

A schema with no rules passes validation (matches anything). Schemas should warn if empty.

### 4.4 Nested Schema Reuse

Schemas can reference other schemas (future):

```yaml
# items/item-response.yaml
properties:
  id: { type: integer }
  owner:
    $ref: users/user-response.yaml
```

---

## 5. Implementation Plan

### Phase 1 — Basic Schema Validation

- [ ] Create `Schema` model for parsing schema YAML files
- [ ] Implement schema loading from `.gherkio/schemas/`
- [ ] Implement type validation (string, integer, number, boolean, array, object, null)
- [ ] Implement required field validation
- [ ] Implement properties validation with nested objects
- [ ] Add `schema` keyword to assertion engine

### Phase 2 — Format & Constraint Validation

- [ ] Implement format validation (email, uuid, datetime, uri)
- [ ] Implement enum validation
- [ ] Implement string constraints (minLength, maxLength, pattern)
- [ ] Implement numeric constraints (minimum, maximum)
- [ ] Implement array constraints (minItems, maxItems, items)

### Phase 3 — Reporting & Edge Cases

- [ ] Schema not found / parse error handling
- [ ] Detailed failure output with field paths
- [ ] Schema caching (avoid re-parsing on repeated use)

---

## 6. Example Output

```
✓ GET /users/me
  ✓ status = 200
  ✓ schema = users/user-response (8 fields validated)

✗ POST /users
  ✗ status = 201
  ✗ schema = users/user-response
      └─ actual: "not-an-email"
      └─ expected: field body.email format email
      └─ reason: value does not match email pattern

      └─ actual: "superadmin"
      └─ expected: field body.role enum [admin, user, moderator]
      └─ reason: value not in allowed list

      └─ actual: (missing)
      └─ expected: field body.id is required
      └─ reason: field not present in response
```

---

## 7. Decisions

### 7.1 Schema as Single Assertion

Schema validation counts as **a single assertion** — one pass/fail for the entire schema. Individual field violations are reported as detail lines under the single assertion, not as separate assertion results. This keeps the test summary clean (1 schema = 1 pass/fail) while providing rich diagnostics on failure.

### 7.2 Inline Schema Definitions

**Deferred.** Phase 1 only supports file-based schemas under `.gherkio/schemas/`. Inline schemas (`schema: { type: object, ... }`) can be added later if users need ad-hoc validation without creating a file.

### 7.3 Cross-file References (`$ref`)

**Deferred.** Phase 1 requires all schema definitions to be in a single file. `$ref` between schema files can be added later as a Phase 2 or 3 enhancement.

## 8. Remaining Open Questions

1. Should schema validation report **all** violations or stop at the first? (Proposal: report all, as shown in examples)
2. Should the schema assertion output include a **total violations count**? (e.g., `3 violations found`)

## 9. Updated

** RFC-3 is fully implemented.** Here's the status against the RFC:

| Phase | Item | Status |
|-------|------|--------|
| **1** | Schema model (`internal/model/schema.go`) | ✅ |
| **1** | Schema loader (`internal/runner/schema.go`) | ✅ |
| **1** | Type validation (string, integer, number, boolean, array, object, null) | ✅ |
| **1** | Required field validation | ✅ |
| **1** | Properties validation with nested objects | ✅ |
| **1** | `schema:` keyword in assertion engine | ✅ |
| **2** | Format validation (email, uuid, datetime, **uri**) | ✅ |
| **2** | Enum validation | ✅ |
| **2** | String constraints (minLength, maxLength, pattern) | ✅ |
| **2** | Numeric constraints (minimum, maximum) | ✅ |
| **2** | Array constraints (minItems, maxItems, items) | ✅ |
| **3** | Schema not found / parse error handling | ✅ |
| **3** | Detailed failure output with field paths | ✅ |
| **3** | Schema caching | ⏳ Deferred (Phase 3 nice-to-have) |
| — | Collection (array) schemas | ✅ |
| — | Mixing schema + individual assertions | ✅ |
| — | Init scaffold with examples + `schema:` usage | ✅ |
| — | Tests (validator, loader, integration, golden file) | ✅ |

The only item not done is **schema caching** — the RFC lists it as Phase 3, and it's an optimization, not a functional gap. Every time `schema:` is referenced, the YAML file is re-read from disk. For MVP this is fine.


