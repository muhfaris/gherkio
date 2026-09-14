# Gherkio DSL Reference — Human & AI-LLM Guide

> 📖 **Gherkio Documentation Hub**: Looking for structured learning paths? See the **[Beginner Guide](book/src/getting-started/introduction.md)**, **[Advanced Guide](book/src/dsl/overview.md)**, **[What Happens in a Test](book/src/getting-started/execution-lifecycle.md)**, or **[Expert & MCP Guide](book/src/mcp/overview.md)** in the official documentation book.

This document is the single source of truth for Gherkio's declarative YAML testing language. It is optimized for both **human developers** (clear, readable, with examples) and **AI Coding Assistants / LLMs** (structurally dense, semantic, and easy to parse).

---

## 🏗️ 1. Test Scenario Structure

A Gherkio test file contains a single scenario definition. All keys are case-sensitive.

```yaml
# The name of the test scenario (Required)
scenario: User login and profile verification

# Optional metadata tags for filtering executions (e.g. gherkio run --tag smoke)
tags:
  - smoke
  - authentication

# Steps run before main steps. Setup failure skips main steps but triggers teardown.
setup:
  - request: ...

# The primary sequential HTTP request steps
steps:
  - request: ...

# Steps run after main steps. Guaranteed to run even if setup or steps fail.
teardown:
  - request: ...
```

---

## ⚡ 2. Step Properties Reference

Each step block in `setup`, `steps`, or `teardown` supports the following properties. A step MUST define exactly one primary operation: `request`, `redis`, `use`, `set`, `repeat`, `for_each`, `export`, or `import`.

| Key | Type | Required | Description | Example |
| :--- | :--- | :--- | :--- | :--- |
| `if` | `string` | No | Conditional guard supporting comparisons, `&&`, `||`, `!`, and parentheses. Bypasses execution if false. | `if: $name && ($count > 0 \|\| $force)` |
| `request` | `object` | Conditional | HTTP request details (method, URL, body, etc.) | (See section below) |
| `use` | `string` | Conditional | Composes/imports another test YAML as a nested step | `use: auth/login.yaml` |
| `set` | `object` | Conditional | Inline variable assignment / override map | `set: { QUEUE_ID: "01KT4EBA" }` |
| `for_each` | `object` | Conditional | Sequentially execute nested steps once per item in a saved array | `for_each: { from: "$items", as: item, steps: [...] }` |
| `export` | `object` | Conditional | Materializes a saved collection into an Excel (`.xlsx`) file | `export: { file: fixtures/bulk.xlsx, from: "$items", columns: [...] }` |
| `import` | `object` | Conditional | Reads data from an Excel (`.xlsx`) file into runtime variables | `import: { file: fixtures/bulk.xlsx, as: items }` |
| `expect` | `object` | No | Assertions map against status, headers, body, or JWT claims | (See section below) |
| `save` | `object` | No | Extracts response data and saves it to runtime variables | `save: { id: body.id }` |
| `retry` | `object` | No | Polling retry rules for eventual consistency validation | (See section below) |
| `timeout` | `string` | No | Custom HTTP request timeout duration | `timeout: 15s` |
| `with` | `object` | No | Variable overrides injected into a `use:` step | `with: { ROLE: "admin" }` |

---

## 🌐 3. HTTP Request Properties (`request`)

```yaml
- request:
    # HTTP verb: GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD
    method: POST
    
    # URL path (relative to baseUrl) or fully qualified absolute URL
    url: /api/v1/users
    
    # Optional service name to override default baseUrl (configured in environment)
    service: identity
    
    # HTTP headers map
    headers:
      Content-Type: application/json
      Authorization: "Bearer $accessToken"
      
    # Request body (JSON objects, lists, strings, or numbers)
    body:
      name: "Emily Watson"
      role: $GHERKIO_DEFAULT_ROLE
```

---

## 🔍 4. Assertions Map Reference (`expect`)

Assertions validate the HTTP response. If any assertion fails, the step fails immediately.

### A. Special Keys
*   `status`: Matches the HTTP response status code (integer or matcher).
    *   `status: 200`
*   `schema`: Validates the full response body shape against a YAML schema inside `.gherkio/schemas/`.
    *   `schema: users/profile-response`
    *   `schema: not auth/login-response` (Negative validation)

### B. Standard Format Path Prefixes
*   `body.<field>`: Traverses the parsed response JSON body (e.g. `body.data[0].id`).
*   `headers.<header-name>`: Validates response HTTP headers (case-insensitive keys).
*   `jwt.<claim-name>`: Validates claims decoded from JWT tokens found in standard response fields (`token` or `accessToken`).

### C. Complete Matcher Library

| Matcher Keyword | Example | Matches If... |
| :--- | :--- | :--- |
| **Existence** | | |
| `exists` | `body.id: exists` | Field is present in payload and not null |
| `not exists` | `body.deletedAt: not exists` | Field is completely absent |
| **Types** | | |
| `uuid` | `body.id: uuid` | Valid UUID v4 string |
| `email` | `body.email: email` | Valid email address string |
| `datetime` | `body.createdAt: datetime`| Valid RFC3339 / ISO8601 datetime string |
| `uri` | `body.avatar: uri` | Valid absolute URI string |
| `string` | `body.name: string` | Target value is a string type |
| `number` | `body.price: number` | Target value is any integer or float |
| `boolean` | `body.active: boolean` | Target value is `true` or `false` |
| `array` | `body.tags: array` | Target value is a JSON list |
| `object` | `body.meta: object` | Target value is a JSON object map |
| `null` | `body.deletedAt: null` | Target value is null |
| `true` | `body.isActive: true` | Target value is boolean true |
| `false` | `body.completed: false`| Target value is boolean false |
| **String Operations** | | |
| `contains <str>` | `body.name: contains Pro` | Target string contains substring |
| `startsWith <str>`| `body.sku: startsWith LAP-`| Target string starts with prefix |
| `endsWith <str>` | `body.email: endsWith .com`| Target string ends with suffix |
| `regex <pattern>` | `body.code: regex ^[A-Z]{3}$`| Target string matches regular expression |
| **Numeric Comparisons** | | |
| `gt <num>` | `body.rating: gt 4` | Value is greater than `<num>` |
| `gte <num>` | `body.price: gte 9.99` | Value is greater than or equal to `<num>` |
| `lt <num>` | `body.age: lt 18` | Value is less than `<num>` |
| `lte <num>` | `body.index: lte 10` | Value is less than or equal to `<num>` |
| **Format Matchers** | | |
| `ipv4` | `body.ip: ipv4` | Valid IPv4 address |
| `ipv6` | `body.ip: ipv6` | Valid IPv6 address |
| `base64` | `body.data: base64` | Valid base64 encoded string |
| `mac` | `body.addr: mac` | Valid MAC address |
| `empty` | `body.list: empty` | Array, string, or object is empty/null |
| **Collections** | | |
| `count(<array-path>)`| `count(body.items): 3` | Checks exact array size. Supports `.gt`, `.gte`, `.lt`, `.lte` suffixes. e.g. `count(body.items).gte: 1` |
| `all(<array-path>)` | `all(body.items.status): active` | Checks if every element in the array matches the condition (equality or matcher) |

---

## 🔄 5. Retries & Eventual Consistency (`retry`)

For polling and waiting on asynchronous tasks (e.g. queue processing).

```yaml
- request:
    method: GET
    url: /orders/$orderId
  retry:
    attempts: 5           # Maximum retry attempts
    interval: 1000        # Wait duration between retries (milliseconds)
    backoff: exponential  # 'constant', 'linear', or 'exponential' (with jitter)
    maxDuration: 15s      # Total time boundary for all retries combined
    onStatus: [404, 202]  # Only retry if HTTP status is 404 or 202
  expect:
    status: 200
    body.status: confirmed
```

---

## 📤 6. Excel Export (`export`)

The `export` step materializes a saved collection into an Excel (`.xlsx`) file on disk. It is a file sink and does not make HTTP requests.

```yaml
- name: Generate user fixture
  export:
    file: fixtures/users.xlsx
    from: $users
    sheet: "UserData"
    columns:
      - header: "ID"
        value: "$string(item.id)"
      - header: "Full Name"
        value: "$string(item.name)"
      - header: "Score"
        value: "$int(item.score)"
      - header: "Tier"
        value: "$if(item.score > 50, 'High', 'Standard')"
```

### Export Configuration

| Property | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `file` | `string` | **Yes** | Destination `.xlsx` path (relative to `export.path` in `.gherkio/config.yaml` or project root). |
| `from` | `string` | **Yes** | Array collection variable (must start with `$`). |
| `sheet` | `string` | No | Worksheet name (default `Sheet1`). |
| `columns` | `list` | **Yes** | List of `{ header: string, value: string }` column mappings. |

Each column expression supports casting (`$string`, `$int`, `$float`, `$bool`), `$if(cond, then, else)`, and row item access (`item.<field>`).

---

## 📥 7. Excel Import (`import`)

The `import` step reads an Excel (`.xlsx`) file from disk and parses its rows into an array of objects stored in a runtime variable (`as: <variableName>`).

```yaml
- name: Read imported catalog
  import:
    file: fixtures/users.xlsx
    sheet: "UserData"          # Optional, defaults to active/first sheet
    as: users                  # Target variable name for array of row objects
    header_row: 1              # Optional, 1-based header row index (default 1)
    data_start_row: 2          # Optional, 1-based data start row (default 2)
    columns:                   # Optional explicit aliases (otherwise auto-converts to snake_case)
      - header: "Full Name"
        as: name
      - header: "Score"
        as: user_score
```

### Import Configuration

| Property | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `file` | `string` | **Yes** | Path to the `.xlsx` file to read. |
| `as` | `string` | **Yes** | Target variable name to store the array of parsed row objects. |
| `sheet` | `string` | No | Worksheet name to read (defaults to active/first sheet). |
| `header_row` | `int` | No | 1-based row index for headers (default: 1). |
| `data_start_row` | `int` | No | 1-based row index where data begins (default: 2). |
| `columns` | `list` | No | Optional explicit column alias mappings (`{ header: string, as: string }`). |

When `columns` is omitted, headers with spaces or special characters are automatically converted to `snake_case` keys (e.g. `"Customer Name"` becomes `customer_name`).

---

## 🔑 8. Variable Reference & Precedence

Variables are declared with a `$` prefix (e.g. `$uuid`) and can be wrapped in curly braces (`${uuid}`) for literal boundary resolution. Default fallbacks are written as `${var:default_value}`.

### Variable Precedence (Lowest to Highest)

Any overlapping variable name is resolved using this order of precedence (later overrides earlier):

```
[1. Host Environment Variables]  <-- Loaded at start (GHERKIO_ prefix only)
               ↓
[2. Selected Account Credentials] <-- Loaded at start (from account file)
               ↓
[3. Step Saves / Set Variables]   <-- Dynamically saved or set in steps
               ↓
[4. Built-in Generators]         <-- Regenerated fresh per step (e.g., $uuid)
```

### Host Environment Variables
Any environment variable on the host OS prefixed with `GHERKIO_` (case-sensitive) is auto-injected. **All other variables (like `PATH` or `USER`) are strictly ignored** to prevent credential leakage.

### Built-in Generator Variables (Regenerated fresh every step)
*   `$uuid`: Fresh UUID v4 string.
*   `$ulid`: Crockford base32 Monotonic ULID.
*   `$randomInt`: Random integer `0-999999`. Or invoke parameter ranges: `${randomInt(min,max)}` (e.g. `${randomInt(1,100)}`).
*   `$randomEmail`: Random email address (e.g. `user_582910@example.com`).
*   `$randomPhone`: Random Indonesian format phone number starting with `+628`.

### Save Paths (`save:` block)

The `save:` block extracts values from responses for use in subsequent steps. Supported path prefixes:

| Prefix | Source | Example |
| :--- | :--- | :--- |
| `body.<path>` | Response JSON body | `save: { id: body.data.id }` |
| `response.body.<path>` | Same as `body.` | `save: { id: response.body.data.id }` |
| `response.<path>` | Backward-compatible alias for `body.` | `save: { id: response.data.id }` |
| `request.body.<path>` | Interpolated request body | `save: { sentEmail: request.body.email }` |
| `jwt.<claim>` | Decoded JWT payload claim | `save: { role: jwt.user_role, parentCustId: jwt.parent_cust_id }` |

The JWT token is automatically decoded from standard response fields (`token`, `accessToken`, `access_token`) or nested paths like `data.access_token`. If your API stores the token in a non-standard location, configure a custom path in `.gherkio/config.yaml`:

```yaml
# .gherkio/config.yaml
jwt_token_path: "data.access_token"
```

### Multi-Account Access via `$accounts.<name>.<field>`

When running with `--account` or `--all-accounts`, all account credentials are accessible via dotted paths:

```yaml
- request:
    method: POST
    url: /login
    body:
      username: $accounts.alice.username
      password: $accounts.bob.password
```

This is useful for cross-account scenarios (e.g. Alice creates a resource, Bob verifies access).
