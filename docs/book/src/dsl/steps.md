# Step Properties

Steps are the execution blocks inside Gherkio's `setup`, `steps`, and `teardown` lifecycle lists. They are evaluated sequentially, passing context variables, environment tokens, and parsed JSON fields down the scenario chain.

---

## ⚡ The Step Structure

A scenario step is defined as a YAML map containing structural blocks that configure the action, validate the response, extract variables, or control loops.

```yaml
- name: Create checkout order        # Optional human-readable step label
  request:                       # 1. Action: Trigger HTTP request
    method: POST
    url: /v1/checkout
    timeout: 5s
  expect:                        # 2. Assert: Validate status & schema
    status: 201
    body.success: true
  save:                          # 3. Context: Store data for next steps
    orderId: body.id
  timing:                        # 4. Perform: Latency budget check
    max: 500ms
```

---

## 🧭 Step Configuration Properties

Each step in a scenario sequence supports the following top-level keys:

| Property Key | Type | Required | Description | Example |
| :--- | :--- | :--- | :--- | :--- |
| `name` | `string` | No | Human-readable label for the step. Shown in test output and HTML report instead of `METHOD /url`. | `name: Create new order` |
| `if` | `string` | No | Conditional guard clause. Step is skipped if the expression evaluates to false. | `if: $responseCode == 200` |
| `request` | `object` | Conditional | HTTP request payload block. **Mutually exclusive with `use` and `set`**. | (See Request Properties below) |
| `use` | `string` | Conditional | Scenario composition. Imports and executes another scenario YAML file inline. **Mutually exclusive with `request` and `set`**. | `use: shared/login.yaml` |
| `set` | `map[string]string`| Conditional | Inline variable assignment. Explicitly assigns or overrides variables. **Mutually exclusive with `request` and `use`**. | `set: { QUEUE_ID: "01KT4EBA37Y" }` |
| `with` | `map[string]string`| No | Variable overrides injected into a `use:` step. Values interpolated before injection; original values restored after completion. | `with: { PARENT_CLAIM_ISSUE_ID: $STATUS_APPROVED_ID }` |
| `expect` | `object` | No | Assertions mapping target dot-notation paths to expected formats or matchers. | `expect: { status: 200 }` |
| `save` | `map[string]string`| No | Context extraction map. Binds response parameters to dynamic variables. | `save: { token: body.accessToken }` |
| `retry` | `object` | No | Automated polling loop rules for testing eventually consistent resources. | (See Retry & Polling chapter) |
| `timing` | `object` | No | Latency budget validation. Asserts that request execution did not exceed duration limits. | `timing: { max: 300ms }` |

---

## 🌐 Request Configuration Properties (`request`)

The `request` block defines the HTTP action Gherkio will execute. It supports the following keys:

| Property Key | Type | Required | Description | Example |
| :--- | :--- | :--- | :--- | :--- |
| `method` | `string` | **Yes** | HTTP request verb. Sourced as `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `OPTIONS`, `HEAD`. | `method: POST` |
| `url` | `string` | **Yes** | Target URL. Supports relative paths (resolves to environment `baseUrl`) or absolute URLs. | `url: /v1/users` |
| `service` | `string` | No | Routes the request to a specific microservice defined in the active environment. | `service: auth` |
| `headers` | `map[string]string`| No | Key-value mapping of custom HTTP headers. Supports variable interpolation. | `headers: { Content-Type: "application/json" }` |
| `body` | `any` | No | Request body payload. Supports JSON maps, lists, raw strings, and variable injection. | `body: { role: "admin" }` |
| `query` | `map[string]string` | No | Query parameters appended to the URL. Supports variable interpolation in values. | `query: { status: available }` |
| `transform` | `object` | No | Declarative collection projections: filter, slice, and reshape arrays from saved variables into the request payload. | (See Requests chapter) |
| `multipart` | `object` | No | Multipart form-data wrapper used for sending form fields and binary file uploads. | (See Requests chapter) |
| `timeout` | `string` | No | HTTP socket timeout limit (parsed via standard Go duration strings like `5s`, `500ms`, `1m`). | `timeout: 10s` |

---

## 🔀 Conditional Execution (`if`)

Steps can be conditionally executed using the `if` guard property. If the expression evaluates to false, the step is skipped entirely (its HTTP request is not sent, assertions are ignored, and any variable extraction is bypassed). Skipped steps are tracked as `skipped` in test metrics, CLI logs, and HTML reports.

### Syntax and Comparison Operators
The `if` property expects a string expression consisting of variables, comparison operators, and literal values (strings, numbers, or booleans).

Supported operators:
*   `==` (Equal to)
*   `!=` (Not equal to)
*   `>` (Greater than)
*   `>=` (Greater than or equal to)
*   `<` (Less than)
*   `<=` (Less than or equal to)

### Examples

#### Basic Variable Comparison
```yaml
steps:
  - name: Generate Admin Invoice
    if: $USER_ROLE == admin
    request:
      method: POST
      url: /v1/invoices/admin
      body:
        amount: 150.00
    expect:
      status: 201
```

#### Numeric Comparison
```yaml
steps:
  - name: Get Invoice Details
    if: $INVOICE_AMOUNT >= 1000
    request:
      method: GET
      url: /v1/audit/large-invoice/$INVOICE_ID
    expect:
      status: 200
```

#### Truthiness Check (Check if variable exists and is not false/empty)
```yaml
steps:
  - name: Process Refund
    if: $REFUND_ENABLED
    request:
      method: POST
      url: /v1/refunds
      body:
        transaction_id: $TX_ID
```

---

## 💾 Dynamic Variable Saving (`save`)

The `save` block allows steps to bind HTTP response parameters (body fields, headers, or decoded JWT claims) to variables that can be dynamically interpolated in all subsequent request URLs, headers, or bodies.

### Syntax Reference
```yaml
save:
  variable_name: response_source_path
```

*   **`body.<path>`**: Extracts JSON fields. Supports dotted-paths, indexes, and collections.
    *   *Example*: `productId: body.items[0].id`
*   **`headers.<header-key>`**: Extracts HTTP response header values (case-insensitive).
    *   *Example*: `rateLimit: headers.X-Rate-Limit`
*   **`jwt.<claim>`**: Automatically decodes response JWT keys (looks for `token`, `accessToken`, `access_token`) and extracts claims.
    *   *Example*: `adminRole: jwt.role`

---

## ⏱️ Latency Budgets (`timing`)

In performance-critical applications, keeping endpoint response latency within a specific budget is a core contract. Gherkio allows developers to validate performance metrics directly inside test steps using the `timing` block:

### Configuration Syntax
```yaml
timing:
  max: duration_string # e.g. "200ms", "1s", "1.5s"
```

If the combined execution time of the step exceeds the `max` threshold, Gherkio fails the step and reports a detailed latency budget violation error:

```
❌ Step 2: GET /users/profile timing assertion failed
  - Expected latency: <= 200ms
  - Actual latency:   242ms
```

---

## 🔌 Scenario Composition & Context Bubble Up (`use`)

To keep test suites DRY, Gherkio steps can delegate execution to a shared modular test script using the `use:` tag.

### Execution Blueprint
```yaml
# Inside login-and-query.yaml
steps:
  - use: auth/login.yaml            # 1. Runs login sequence, saves $authToken
  - request:
      method: GET
      url: /profile
      headers:
        Authorization: "Bearer ${authToken}" # 2. Automatically inherits $authToken
```

1.  **Monotonic Variables**: Any variable saved (via `save`) inside the composed YAML file is automatically merged and bubbles up to the parent execution context.
2.  **Context Inheritance**: Composed scenarios inherit all variables defined prior to their execution (e.g. host environments, active credential credentials).

---

## ⚙️ Declarative Variable Assignment (`set`)

Gherkio steps can explicitly assign, update, or override variables in the runtime context using the `set` tag. This is particularly useful for overriding defaults during local testing/debugging, managing sequential state, or addressing variable name collisions without executing a full HTTP request or nested scenario.

### Syntax and Usage

The `set` block accepts a map of variable keys to their string values. Values support variable interpolation.

```yaml
steps:
  # 1. Manually set/override a variable
  - name: Define custom queue ID
    set:
      QUEUE_ID: "01KT4EBA37Y"

  # 2. Reference the variable in subsequent steps
  - name: Get Queue info
    request:
      method: GET
      url: /v1/queues/$QUEUE_ID
    expect:
      status: 200

  # 3. Re-assign or interpolate variables
  - name: Rotate queue ID
    set:
      PREVIOUS_QUEUE_ID: "$QUEUE_ID"
      QUEUE_ID: "02HT5FCA38Z"
```

### Key Behaviors
- **Mutual Exclusion**: A step containing `set` must not contain a `request` or `use` key.
- **Interpolation**: Variables referenced in `set` values (e.g. `$QUEUE_ID`) are interpolated immediately at execution time using the active variable store.
- **CLI Output**: In test logs, a `set` step is formatted to show which variables are being set (e.g., `set variables: QUEUE_ID`).
