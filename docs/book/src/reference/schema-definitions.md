# Schema Definitions Layout

Define structural response validation schemas under `.gherkio/schemas/` to enforce robust API contracts.

---

## 📏 Specification Standards

Gherkio natively parses schemas conforming to the **JSON Schema Draft-07** specification. You can define schemas as standard JSON (`.json`) or YAML (`.yaml`/`.yml`).

---

## 📝 Canonical Schema Layout Example

Example: `.gherkio/schemas/auth/token-payload.yaml`
```yaml
$schema: "http://json-schema.org/draft-07/schema#"
type: object
required:
  - token
  - expiresAt
  - user
properties:
  token:
    type: string
    pattern: "^eyJ[A-Za-z0-9-_=]+\\.[A-Za-z0-9-_=]+\\.?[A-Za-z0-9-_.+/=]*$" # Simple JWT format regex
  expiresAt:
    type: string
    format: date-time
  user:
    type: object
    required:
      - id
      - email
    properties:
      id:
        type: string
        format: uuid
      email:
        type: string
        format: email
```

---

## ⚡ Asserting the Schema inside steps

Refer to the relative path of the schema file (omitting the extension) under the `expect` block:

```yaml
steps:
  - request:
      method: POST
      url: /v1/auth/token
    expect:
      status: 200
      schema: auth/token-payload     # References .gherkio/schemas/auth/token-payload.yaml
```

If the server response contains a different structure (e.g. `expiresAt` is a number, or `user.id` is not a valid UUID), the test run halts with high-precision validation tracebacks.
