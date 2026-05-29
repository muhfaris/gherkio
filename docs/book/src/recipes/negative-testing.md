# Negative Testing & Error Validation

API contracts should explicitly enforce business constraints and output clean error bodies. Negative tests assert that requests containing bad payloads, invalid formatting, or missing authentication are successfully rejected by the server.

---

## 📝 The Scenario

```yaml
scenario: Negative Profile Creation Checks
tags:
  - active
  - negative

steps:
  # Step 1: Attempt to register with a duplicate email
  - request:
      method: POST
      url: /v1/users
      body:
        email: "existing-user@my-domain.com"
        password: "securepassword"
    expect:
      status: 409
      body.error: "DuplicateEmail"
      body.message: contains "already exists"

  # Step 2: Attempt to submit a weak password
  - request:
      method: POST
      url: /v1/users
      body:
        email: $randomEmail
        password: "123"
    expect:
      status: 400
      body.code: "InvalidPassword"
      body.details.password[0]: contains "too short"

  # Step 3: Attempt to fetch billing without credentials
  - request:
      method: GET
      url: /v1/billing
    expect:
      status: 401
      body.error: "Unauthorized"
```

---

## 💡 Key Design Patterns Used

1. **Assert Non-2xx Responses**: In Gherkio, steps expecting `4xx` or `5xx` statuses are fully expected and marked as green when the response matches.
2. **Detailed Error Inspections**: Using `contains` assertions on `body.details` arrays to ensure validation packages output helpful user guidance.
