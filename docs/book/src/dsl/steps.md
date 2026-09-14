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
| `request` | `object` | Conditional | HTTP request payload block. Mutually exclusive with other step operations. | (See Request Properties below) |
| `redis` | `object` | Conditional | Controlled read-only Redis operation. Mutually exclusive with other step operations. | `redis: { connection: local-cache, command: get, key: "product:42" }` |
| `use` | `string` | Conditional | Scenario composition. Imports and executes another scenario YAML file inline. | `use: shared/login.yaml` |
| `set` | `map[string]string`| Conditional | Inline variable assignment. Explicitly assigns or overrides variables. | `set: { QUEUE_ID: "01KT4EBA37Y" }` |
| `export` | `object` | Conditional | Materializes a saved collection into an Excel (`.xlsx`) file. Mutually exclusive with other step operations. | `export: { file: fixtures/bulk.xlsx, from: "$items", columns: [...] }` |
| `repeat` | `object` | Conditional | Repeats a group of steps until a condition is true or the attempt limit is exhausted. | `repeat: { attempts: 20, until: "$count == 0", steps: [...] }` |
| `for_each` | `object` | Conditional | Executes nested steps sequentially once per item in a saved array. | `for_each: { from: "$items", as: item, steps: [...] }` |
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

## Redis Steps (`redis`)

Redis steps expose only the read-only commands `get`, `exists`, `ttl`, and
`hgetall`. Arbitrary commands and Lua scripts are intentionally unsupported.

```yaml
- name: Verify product cache
  redis:
    connection: local-cache
    command: get
    key: "product:$productId"
  expect:
    redis.exists: true
    redis.value.id: "$productId"
  save:
    cachedName: redis.value.name
  retry:
    attempts: 5
    interval: 200
```

`get` automatically decodes JSON values. Redis assertions and saved values use
the `redis.*` path: `redis.exists`, `redis.value`, `redis.value.<field>`, and
`redis.ttl`. Existing matchers, timing assertions, and retry strategies apply.

See [Redis Cache Checks](redis.md) for complete API-plus-cache scenarios,
command-specific result paths, TTL and hash examples, polling, and Sentinel use.

## Collection Request Loops (`for_each`)

Use `for_each` when an endpoint requires one request per item rather than a
single batch payload. `from` resolves a saved array, `as` names the scoped
current item (default: `item`), and nested `steps` execute sequentially.

```yaml
steps:
  - request:
      method: GET
      url: /api-a/items
    save:
      items: body.items

  - name: Send every item to API B
    for_each:
      from: $items
      as: item
      steps:
        - request:
            method: POST
            url: /api-b/items
            body:
              external_id: $item.id
              name: $item.name
          expect:
            status: 201
```

The loop stops at the first failing nested step. Variables saved inside the
loop remain available afterward, while the item alias is restored when the
loop ends. An empty source array succeeds without executing nested steps; a
missing or non-array source fails the loop. Terminal and report output label
each nested execution as `for_each N/M`.

## Bounded Multi-Step Loops (`repeat`)

Use `repeat` when one polling attempt needs multiple operations. The nested
`steps` run sequentially, then `until` is evaluated against the updated
variables. A true condition preserves those variables for following steps and
ends the loop. An inner failure stops immediately; if the condition remains
false after every attempt, the repeat step fails.

```yaml
- name: Find an unused issue tag
  repeat:
    attempts: 20
    until: $existingTicketCount == 0
    steps:
      - name: Select candidate issue tag
        set:
          ISSUE_TAG_L3: ${randomItem(respIssueTagL3)}

      - name: Check tickets using candidate
        request:
          method: GET
          url: /v1/tickets
          query:
            issue_tag_id: $ISSUE_TAG_L3.id
        expect:
          status: 200
        save:
          existingTicketCount: count(body.data)
```

This is a bounded loop, not unbounded recursion. Reports label every nested
execution as `repeat N/M`. A dry run previews the block once because no live
response exists for evaluating response-dependent exit conditions.

---

## 📤 Excel Export (`export`)

The `export` step materializes a saved collection into an Excel (`.xlsx`) file.
It is a **sink**: it writes data to disk instead of making an HTTP request, so
it pairs naturally with `multipart.files` to generate fixtures for bulk-import
APIs.

```yaml
steps:
  - name: Fetch product catalog
    request:
      method: GET
      url: /products
    save:
      products: body.products

  - name: Generate bulk upload Excel fixture
    export:
      file: fixtures/bulk_upload.xlsx
      from: $products
      sheet: "BulkData"
      columns:
        - header: "product_title"
          value: "$string(item.title)"
        - header: "price"
          value: "$float(item.price)"
        - header: "priority"
          value: "$if(item.price > 500, 'priority', 'normal')"
        - header: "generated_at"
          value: "${dateNow(\"2006-01-02\")}"
```

### Configuration Properties

| Property | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `file` | `string` | **Yes** | Output path for the Excel file. Absolute paths are used as-is; relative paths resolve against the configured `export.path` (default: project root). |
| `from` | `string` | **Yes** | Source collection array variable (must start with `$`). |
| `sheet` | `string` | No | Worksheet name (default `Sheet1`). |
| `columns` | `list` | **Yes** | Ordered column definitions. |

Each column has:

| Property | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `header` | `string` | **Yes** | Column header text. |
| `value` | `string` | **Yes** | Per-cell expression. Supports interpolation, casting (`$string`, `$int`, `$float`, `$bool`), `$if(...)`, and references to saved variables. |

### Key Behaviors
- **Mutual Exclusion**: `export` cannot be combined with `request`, `redis`, `use`, `set`, `repeat`, `for_each`, `with`, `expect`, `save`, `timing`, or `retry`.
- **Row Scoping**: Each row is evaluated with the current item bound to `item` (plus all saved variables).
- **Conditional Values**: `$if(condition, then, else)` supports full boolean expressions with comparison operators (`==`, `!=`, `>`, `>=`, `<`, `<=`, `&&`, `||`, `!`).
- **No HTTP**: The step passes when the file writes successfully.
- **Configurable Output**: Set `export.path` in `.gherkio/config.yaml` to write relative `file` paths under a dedicated directory (e.g. `fixtures`).

---

## 📥 Excel Import (`import`)

The `import` step reads an Excel (`.xlsx`) spreadsheet from disk and parses its rows into an array of objects saved in a runtime variable (`as: <variableName>`). This enables test scenarios to ingest external Excel fixtures, iterate through records with `for_each`, or validate uploaded spreadsheet contents.

```yaml
steps:
  - name: Ingest customer dataset
    import:
      file: fixtures/customers.xlsx
      sheet: "Customers"          # Optional, defaults to active/first sheet
      as: CUSTOMER_LIST          # Target variable storing array of row maps
      header_row: 1              # Optional, 1-based header row index (default: 1)
      data_start_row: 2          # Optional, 1-based data start row (default: 2)

  - name: Create each customer via API
    for_each:
      from: $CUSTOMER_LIST
      as: customer
      steps:
        - name: Send creation request
          request:
            method: POST
            url: /v1/customers
            body:
              full_name: $customer.customer_name
              email: $customer.email_address
              initial_balance: $float(customer.balance)
```

### Configuration Properties

| Property | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `file` | `string` | **Yes** | Target `.xlsx` file path (absolute or relative to project root / export path). |
| `as` | `string` | **Yes** | Target variable name to store parsed rows (without leading `$`). |
| `sheet` | `string` | No | Worksheet name (default: active/first sheet). |
| `header_row` | `int` | No | 1-based row index containing column headers (default: `1`). |
| `data_start_row` | `int` | No | 1-based row index where data rows start (default: `2`). |
| `columns` | `list` | No | Optional list of explicit column alias mappings (`{ header, as }`). |

### Key Behaviors
- **Header Normalization**: When `columns` is omitted, headers containing spaces or punctuation are automatically converted to clean `snake_case` keys (e.g. `"Customer Name ($)"` becomes `customer_name`).
- **Explicit Aliasing**: Specify `columns` mappings to rename specific headers to custom field keys.
- **Mutual Exclusion**: `import` is a primary step operation and cannot be combined with `request`, `redis`, `use`, `set`, `repeat`, `for_each`, `export`, `with`, `expect`, `save`, `timing`, or `retry`.
- **Seamless Expressions**: The loaded variable is a standard array of maps, fully accessible via `$list[0].field`, `${randomItem(list)}`, and `for_each: { from: $list }`.

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
*   `&&` (Logical AND)
*   `||` (Logical OR)
*   `!` (Logical negation)

`&&` has higher precedence than `||`. Use parentheses to group expressions.
Both operators short-circuit: a false left side skips the right side of `&&`,
and a true left side skips the right side of `||`.

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

#### Compound Conditions

```yaml
steps:
  - name: Process complete customer record
    if: $item.ticket_name && $item.customer_name && $item.ticket_code
    request:
      method: POST
      url: /v1/webhooks/premature-tickets
```

```yaml
# Parentheses and negation
if: ($enabled && $count > 0) || !($status == blocked)
```

For backward compatibility, negating a comparison without parentheses (for
example `!$status == blocked`) is rejected as ambiguous. Write
`!($status == blocked)` instead.

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
*   **`count(body.<path>)`**: Saves the length of a response array as an integer. An explicitly present `null` value saves `0`; an empty array also saves `0`. Missing paths and non-array values produce a save warning and are not stored.
    *   *Example*: `4-notesBeforeConflict: count(body.data)`

```yaml
save:
  notes: body.data
  notesCount: count(body.data)
```

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
- **Typed random selection**: An exact `${randomItem(array)}` value preserves the selected object, so later steps can access fields such as `$PARTNER_STATUS.id`. `${randomItem(array,id)}` continues to store only the selected field.
- **CLI Output**: In test logs, a `set` step is formatted to show which variables are being set (e.g., `set variables: QUEUE_ID`).

```yaml
- name: Select one partner status
  set:
    PARTNER_STATUS: ${randomItem(respPartnerStatuses)}

- name: Use fields from the same selected object
  request:
    method: POST
    url: /v1/partners
    body:
      partner_status_id: $PARTNER_STATUS.id
      partner_status_value: $PARTNER_STATUS.value
```
