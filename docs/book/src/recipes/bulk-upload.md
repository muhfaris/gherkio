# Bulk Upload & Collection Validation

This recipe demonstrates how to submit a list collection in a single POST request and assert properties across the returned array of created items.

---

## 📝 The Scenario

```yaml
scenario: Bulk Item Upload Validation
tags:
  - bulk
  - catalog

steps:
  # Step 1: Submit bulk items list
  - request:
      method: POST
      url: /v1/products/bulk
      headers:
        Content-Type: application/json
      body:
        - sku: "LAP-CORE-I5"
          name: "Developer Laptop v1"
          price: 1200.00
        - sku: "LAP-CORE-I7"
          name: "Developer Laptop v2"
          price: 1800.00
    expect:
      status: 201
      body: array
      
      # Enforce array length is exactly 2
      count(body): 2
      
      # Validate every item in response matches criteria
      all(body.status): "active"
      all(body.sku): startsWith "LAP-"
      
      # Assert specific items in the list using indices
      body[0].sku: "LAP-CORE-I5"
      body[1].price: 1800.00
```

---

## 💡 Key Design Patterns Used

1. **Declarative Arrays**: JSON arrays are mapped directly as standard YAML sequences under the `body` field.
2. **Dynamic Collection Assertions**: `count(body): 2` evaluates array boundaries without looping scripts.
3. **High-Level Array Matchers**: The `all(<path>): <matcher>` syntax validates fields on every member of an array.
