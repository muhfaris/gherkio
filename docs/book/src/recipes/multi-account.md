# Multi-Account Role Boundary Testing

Ensure your APIs isolate resources according to client roles. This recipe shows how to write a scenario validating role limits across multiple accounts.

---

## 📝 The Scenario

```yaml
scenario: Role Boundary Validation
tags:
  - security
  - permissions

steps:
  # Step 1: Login as administrator to create resource
  - request:
      method: POST
      url: /v1/restricted-resource
      headers:
        Authorization: "Bearer ${adminToken}"   # adminToken saved in setup block
      body:
        title: "Confidential Strategy Document"
    expect:
      status: 201
    save:
      resourceId: body.id

  # Step 2: Attempt to read resource using viewer token
  - request:
      method: GET
      url: /v1/restricted-resource/$resourceId
      headers:
        Authorization: "Bearer ${viewerToken}"  # viewerToken saved in setup block
    expect:
      status: 200
      body.title: exists

  # Step 3: Attempt to delete resource as viewer (should fail)
  - request:
      method: DELETE
      url: /v1/restricted-resource/$resourceId
      headers:
        Authorization: "Bearer ${viewerToken}"
    expect:
      status: 403
      body.error: "PermissionDenied"
```

---

## 💡 Key Design Patterns Used

1. **Setup Token Caching**: Authenticates once inside a `setup` block using credentials defined under `.gherkio/credentials/staging.yaml` for both `admin` and `viewer` accounts.
2. **Access Isolation Assertion**: Direct verification of resource endpoints returning `403 Forbidden` for restricted action blocks while successfully returning `200 OK` for authorized actions.
