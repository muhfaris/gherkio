# Scenario Composition (`use:`)

Scenario Composition allows Gherkio tests to remain modular, reusable, and DRY (Don't Repeat Yourself). By using the `use:` property, a scenario can import and run another scenario file in-line as a single step.

---

## ⚡ Basic Usage

A common testing pattern is performing an authentication sequence before requesting resource endpoints. Instead of copy-pasting the authentication request in dozens of test files, define it once in a shared test:

### Shared Auth Step: `.gherkio/tests/shared/login.yaml`
```yaml
scenario: Authenticate User
steps:
  - request:
      method: POST
      url: /auth/login
      body:
        # Option A: Dynamic runtime account resolution (selected via --account <name> CLI flag)
        username: $accounts.username
        password: $accounts.password

        # Option B: Explicit account name (hardcoded to specific credential block)
        # username: $accounts.admin.username
        # password: $accounts.admin.password
    expect:
      status: 200
    save:
      authToken: body.token
```

> 💡 **Dynamic Account Resolution with `--account`**:
> If you write `$accounts.username` or `$accounts.password` without an account name, Gherkio automatically maps those variables to the active account specified by the `--account` CLI parameter:
> * `gherkio run --account admin` → `$accounts.username` resolves to `$accounts.admin.username`
> * `gherkio run --account alice` → `$accounts.username` resolves to `$accounts.alice.username`
>
> This allows your shared `login.yaml` file to be 100% reusable across different user roles and test accounts without modifying the scenario YAML!

### Main Scenario: `.gherkio/tests/users/list.yaml`
```yaml
scenario: Retrieve User List
steps:
  - use: shared/login.yaml          # Executes the login scenario in-line

  - request:
      method: GET
      url: /users
      headers:
        Authorization: "Bearer ${authToken}"   # Uses variable saved in the used scenario
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

Sometimes you need to pass dynamic input parameters *into* a composed scenario (parameterized test templates) rather than just consuming variables that bubble *out*. The `with:` block lets you inject temporary variable overrides into the target composed scenario's execution context.

```yaml
- use: shared/lookup/status_claim.yaml
  with:
    CLAIM_ID: $CREATED_CLAIM_ID
    EXPECTED_STATUS: "APPROVED"
```

### ⚙️ How `with:` Works

1. **Interpolation**: Each expression in `with:` is evaluated against the current parent context before injection.
2. **Context Scope**: The resolved values are injected as local variables into the target scenario for the duration of its execution.
3. **Automatic Restoration**: Once the composed scenario finishes executing, previous parent variable values are restored so the injected override doesn't pollute subsequent steps in the main scenario.

---

### 📂 Complete Two-File Workflow Example

Below is a complete real-world setup showing both the shared target scenario file and the main parent scenario file using `with:`.

#### 1. Shared Composed Scenario: `.gherkio/tests/shared/lookup/status_claim.yaml`

This scenario expects two input variables (`$CLAIM_ID` and `$EXPECTED_STATUS`), which will be injected via `with:` by any parent scenario:

```yaml
scenario: Shared Claim Status Lookup
steps:
  - name: Fetch claim status details
    request:
      method: GET
      url: /v1/claims/$CLAIM_ID            # Uses injected $CLAIM_ID
      headers:
        Authorization: "Bearer ${authToken}"
    expect:
      status: 200
      body.data.status: "$EXPECTED_STATUS" # Asserts against injected $EXPECTED_STATUS
    save:
      claimAssignee: body.data.assignedTo  # Bubbles up saved variable back to parent
```

#### 2. Main Parent Scenario: `.gherkio/tests/claims/verify-approval.yaml`

The parent scenario imports `shared/lookup/status_claim.yaml` and passes runtime parameters using the `with:` block:

```yaml
scenario: Verify Order Claim Approval Flow

setup:
  - use: shared/login.yaml                # Logs in and provides $authToken

steps:
  - name: Create new insurance claim
    request:
      method: POST
      url: /v1/claims
      headers:
        Authorization: "Bearer ${authToken}"
      body:
        policyId: "POL-1002"
        amount: 500
    expect:
      status: 201
    save:
      NEW_CLAIM_ID: body.data.id          # Saves "CLM-9948"

  - name: Validate claim status using shared lookup template
    use: shared/lookup/status_claim.yaml
    with:
      CLAIM_ID: $NEW_CLAIM_ID             # Passes $NEW_CLAIM_ID into the shared step as $CLAIM_ID
      EXPECTED_STATUS: "PENDING"          # Passes expected status threshold

  - name: Verify assignee bubbled up from shared step
    request:
      method: GET
      url: /v1/users/$claimAssignee       # Uses $claimAssignee saved by the shared lookup step
    expect:
      status: 200
```

---

### 💡 Key Rules for `with:`

- **`use:` Only**: `with:` is **only valid on `use:` steps**. It cannot be combined with standard `request:` steps.
- **Full Interpolation Support**: Values inside `with:` support all Gherkio expressions (`$var`, `${var:default}`, `$accounts.username`, built-in generators, etc.).
- **Scoped Precedence**: Injected variables take highest precedence inside the target composed scenario without modifying the caller's global variable state.


---

## 🔍 Composed Traceability & Debugging

When executing nested scenarios, tracing execution flows and context variables can become challenging. Gherkio provides native composed traceability inside its HTML run reports:

1. **Visual Grouping**: Composed scenarios are rendered as distinct, visually encapsulated boxes containing their inner steps.
2. **Nesting Indentation**: Steps inside composed scenarios are dynamically indented according to their nesting `depth`.
3. **Variable Snapshots**: The entry point of every composed scenario captures a complete snapshot of all active context variables (both inherited and overridden via `with:`), allowing you to inspect execution states during debug.
