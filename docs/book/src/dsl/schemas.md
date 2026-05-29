# Schema Validation

Gherkio supports robust custom validation schemas. This allows you to enforce schema contracts for complex JSON APIs rather than writing dozens of line-by-line field assertions.

---

## 📐 Schema Definitions Directory

All validation schemas live under `.gherkio/schemas/` and are defined using standard YAML or JSON files.

Gherkio resolves referenced schemas relative to the `.gherkio/schemas/` directory. For example, referencing `users/profile` matches:
- `.gherkio/schemas/users/profile.yaml` (or `.yml`)
- `.gherkio/schemas/users/profile.json`

---

## ⚡ Asserting a Schema in DSL

To validate that an API response matches your contract, reference the schema file relative to `.gherkio/schemas/` inside the `expect` block:

```yaml
steps:
  - request:
      method: GET
      url: /profile
    expect:
      status: 200
      schema: users/profile         # Matches users/profile.yaml or users/profile.json
```

If the API response structure changes or omits a required field, Gherkio will output a clear error path highlighting precisely what failed the contract validation:

```
❌ Step 1: GET /profile failed schema validation
  - body.id: Does not match pattern
  - body.email: Required field is missing
```

---

## ⚙️ Supported Schema Constraints Reference

Gherkio's schema engine is highly optimized and supports standard JSON Schema-like validation keywords. Below is the complete reference of all supported properties.

### 1. Structural & Core Keywords

| Keyword | Type | Description |
| :--- | :--- | :--- |
| `type` | `string` | Defines the expected data type. Allowed types: `object`, `array`, `string`, `number`, `integer`, `boolean`, `null`. |
| `nullable` | `boolean` | If set to `true`, the value is allowed to be `null` even if other type rules are violated. Defaults to `false`. |
| `enum` | `array` | A list of allowed values. The actual value must match one of the items exactly. |

### 2. Object Constraints

| Keyword | Type | Description |
| :--- | :--- | :--- |
| `properties` | `object` | A map of nested key-value schemas defining the structure of object fields. |
| `required` | `array` | A list of mandatory property keys that must exist and cannot be null. |

### 3. Array Constraints

| Keyword | Type | Description |
| :--- | :--- | :--- |
| `items` | `schema` | The validation schema applied to **every** item inside the array. |
| `minItems` | `integer` | Enforces the minimum number of items allowed in the array. |
| `maxItems` | `integer` | Enforces the maximum number of items allowed in the array. |

### 4. String Constraints

| Keyword | Type | Description |
| :--- | :--- | :--- |
| `pattern` | `string` | A standard Go-compliant regular expression pattern that the string must match. |
| `minLength` | `integer` | The minimum length (number of characters) required for the string. |
| `maxLength` | `integer` | The maximum length (number of characters) allowed for the string. |
| `format` | `string` | Pre-defined specialized string format matches. Supported formats are: |
| | | ➔ `email`: Standard email format validation.<br>➔ `uuid`: Standard UUID v4 format validation.<br>➔ `datetime`: Valid RFC 3339/ISO 8601 timestamp formats.<br>➔ `uri`: Valid absolute URI/URL pathway validation. |

### 5. Numeric Constraints (Numbers & Integers)

| Keyword | Type | Description |
| :--- | :--- | :--- |
| `minimum` | `number` | Enforces that the numeric value is greater than or equal to this limit. |
| `maximum` | `number` | Enforces that the numeric value is less than or equal to this limit. |

---

## 💡 Complete Schema Validation Examples

### Example A: Comprehensive Array Schema (`.gherkio/schemas/posts/list.yaml`)
Enforces custom length limits, strict nullable attributes, item structure, and format matching for a list of blog posts.

```yaml
type: array
minItems: 1
maxItems: 50
items:
  type: object
  required:
    - id
    - title
    - views
    - tags
  properties:
    id:
      type: string
      format: uuid
    title:
      type: string
      minLength: 3
      maxLength: 100
    description:
      type: string
      nullable: true           # Description can be null or a string
    views:
      type: integer
      minimum: 0
    tags:
      type: array
      minItems: 1
      items:
        type: string
        pattern: "^[a-z0-9-]+$" # Tags must be lowercase alphanumeric-kebab
```

### Example B: Nested Object Schema (`.gherkio/schemas/users/auth.yaml`)
Validates a standard login response structure.

```yaml
type: object
required:
  - user
  - token
properties:
  token:
    type: string
    minLength: 32
  user:
    type: object
    required:
      - id
      - email
      - role
    properties:
      id:
        type: string
        format: uuid
      email:
        type: string
        format: email
      role:
        type: string
        enum:
          - admin
          - user
          - editor
```
