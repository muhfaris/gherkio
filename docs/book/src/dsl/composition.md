# Scenario Composition (`use:`)

Scenario Composition allows Gherkio tests to remain modular, reusable, and DRY (Don't Repeat Yourself). By using the `use:` property, a scenario can import and run another scenario file in-line as a single step.

---

## ⚡ Basic Usage

A common testing pattern is performing an authentication sequence before requesting resource endpoints. Instead of copy-pasting the authentication request in dozens of test files, define it once in a shared test:

### Shared Auth Step: `.gherkio/tests/shared/login.yaml`
```yaml
scenario: Authenticate Admin
steps:
  - request:
      method: POST
      url: /auth/login
      body:
        username: $accounts.admin.username
        password: $accounts.admin.password
    expect:
      status: 200
    save:
      adminToken: body.token
```

### Main Scenario: `.gherkio/tests/users/list.yaml`
```yaml
scenario: Retrieve User List
steps:
  - use: shared/login.yaml          # Executes the login scenario in-line

  - request:
      method: GET
      url: /users
      headers:
        Authorization: "Bearer ${adminToken}"   # Uses variable saved in the used scenario
    expect:
      status: 200
```

---

## 🔄 Composition in Setup, Steps, and Teardown

In Gherkio, the `setup`, `steps`, and `teardown` blocks are structurally identical lists of steps. Because of this elegant design, **you can use `use:` composition in any phase of your scenario's lifecycle**:

*   **`setup`**: Use composition to log in, seed initial databases, or construct pre-condition states.
*   **`steps`**: Use composition to execute core nested business operations or user flows.
*   **`teardown`**: Use composition to run post-condition cleanups, delete created test objects, or invalidate session tokens.

### Comprehensive Lifecycle Composition Example:
```yaml
scenario: Complete Order Cycle
setup:
  - use: shared/login.yaml               # Logs in and bubbles up $adminToken
  - use: shared/seed-inventory.yaml      # Seeds item database before run

steps:
  - request:
      method: POST
      url: /orders
      headers:
        Authorization: "Bearer ${adminToken}"
      body:
        itemId: 105
        quantity: 1
    expect:
      status: 201
    save:
      createdOrderId: body.orderId

teardown:
  - use: shared/clear-inventory.yaml     # Resets DB inventory back to baseline
```

---

## 🧭 File Resolution Logic

When Gherkio encounters a `use: <path>` step, it attempts to resolve the filepath in the following sequence:

1. **Relative Path**: Relative to the directory containing the current test file.
2. **Workspace Root**: Relative to the root `.gherkio/tests/` directory.
3. **Absolute Path**: Absolute file paths on the local filesystem.

---

## ⚠️ Recursion & Composition Rules

To prevent infinite loops and ensure robust execution, Gherkio enforces strict structural and execution rules on all composed steps.

### 1. Request/Use Mutual Exclusion
A single step **cannot define both `use:` and `request:`**. They are strictly mutually exclusive:

*   **❌ Illegal Step**:
    ```yaml
    # Fails static analysis!
    - use: shared/login.yaml
      request:
        method: GET
        url: /profile
    ```
*   **➔ Correct Structure**:
    ```yaml
    - use: shared/login.yaml
    - request:
        method: GET
        url: /profile
    ```

### 2. Forbidden Retry Blocks on Composed Steps
You **cannot apply a `retry:` block to a `use:` step**. Retries are only allowed on individual HTTP requests, not entire composed scenarios. If you define a retry block on a composed step, Gherkio halts immediately with a validation error.

*   **❌ Illegal Step**:
    ```yaml
    # Fails validation!
    - use: shared/login.yaml
      retry:
        attempts: 3
        backoff: 1s
    ```
*   **➔ Correct Structure**: Apply the retry block *inside* the shared `login.yaml` step definition directly.

### 3. Nesting & Recursion Limits
To protect runner performance and prevent system freezes:
*   **Default Composition Depth**: Nesting is limited to a maximum depth of **5 nested layers** by default (customizable in your `config.yaml` using the `maxCompositionDepth` property).
*   **Strict Cutoff Limit**: Regardless of configurations, Gherkio enforces a hard ceiling at **10 nested layers**. Exceeding this triggers a static error (`circular reference or max depth exceeded`).
*   **Circular Import Checks**: Gherkio builds an import dependency graph prior to execution. If a circular loop is found (e.g. `A.yaml` ➔ `B.yaml` ➔ `A.yaml`), the executor aborts before making any HTTP requests.

### 4. Context Inheritance & Variables Bubbling
*   Any dynamic variables or state captured (via the `save` keyword) inside a composed scenario are **automatically merged and bubbled up** into the parent scenario's execution context.
*   Once a composed step finishes executing, all subsequent steps in the parent file have full access to its saved variables.

---

## ↔ Variable Overrides (`with:`)

Sometimes you need to pass a value *into* a composed scenario, rather than just consuming values that bubble *out*. The `with:` block lets you inject temporary variables into the used scenario's context:

```yaml
- use: shared/lookup/status_claim.yaml
  with:
    PARENT_CLAIM_ISSUE_ID: $STATUS_APPROVED_ID
```

### How it works

1. Each key in `with:` is **interpolated** against the current execution context before injection.
2. The resolved values are injected as local variables into the used scenario for the duration of its execution.
3. After the `use:` step completes, the **previous values are restored** — the override doesn't leak to subsequent steps.

### Real-world example

```yaml
scenario: check claim status after approval

setup:
  - request:
      method: POST
      url: /auth/login
      body:
        username: $accounts.default.username
        password: $accounts.default.password
    expect:
      status: 200
    save:
      STATUS_APPROVED_ID: body.approvedStatusId

steps:
  # Pass the approved status ID into a shared lookup scenario
  - use: shared/lookup/status_claim.yaml
    with:
      PARENT_CLAIM_ISSUE_ID: $STATUS_APPROVED_ID

  - request:
      method: GET
      url: /claims/$CLAIM_ID
      headers:
        Authorization: "Bearer ${adminToken}"
    expect:
      status: 200
      body.status: Approved
```

### Notes

- `with:` is **only valid on `use:` steps**. It cannot be combined with `request:`.
- Values support full variable interpolation (`$var`, `${var:default}`, `$accounts.<name>.<field>`, built-in generators, etc.).
- Overrides take precedence over any variables with the same name from the parent context, but only inside the used scenario.
