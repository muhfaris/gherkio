# Contract Verification with Schemas

Validate complex microservice structures against strict JSON Schema contracts with a single assertion line.

---

## 📝 The Scenario

```yaml
scenario: Verify User List Contract
tags:
  - contract
  - users

steps:
  - request:
      method: GET
      url: /v1/users
      headers:
        Authorization: "Bearer ${adminToken}"
    expect:
      status: 200
      
      # Validates the response body against the target schema
      schema: users/list-response
```

---

## 📐 The Schema File: `.gherkio/schemas/users/list-response.yaml`

Define your structural contracts inside `.gherkio/schemas/`:

```yaml
type: object
required:
  - users
  - pageInfo
properties:
  users:
    type: array
    items:
      type: object
      required:
        - id
        - name
        - email
      properties:
        id:
          type: string
          format: uuid
        name:
          type: string
        email:
          type: string
          format: email
  pageInfo:
    type: object
    required:
      - page
      - totalPages
    properties:
      page:
        type: integer
      totalPages:
        type: integer
```

---

## 💡 Key Design Patterns Used

1. **Zero-Boilerplate Checks**: Validating deep collection trees, formats, and required field lists with one simple `schema:` declaration.
2. **Schema Reuse**: The schema file can be referenced by multiple scenarios, ensuring changes to contracts are updated in one central location.
