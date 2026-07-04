# Assertions & Dot-Paths

Assertions are the core validation engine of Gherkio. Placed inside the `expect` block of any step, assertions inspect, type-check, and validate the returned HTTP response status, headers, body structure, and even embedded JWT tokens. 

---

## 🏗️ The Anatomy of `expect`

The `expect` block defines a set of strict contracts that the HTTP response must satisfy. Gherkio evaluates these assertions sequentially upon request completion.

### ⚡ Fail-Fast Execution
Gherkio implements a strict **fail-fast** design. 
- Assertions are checked in the order they are written.
- If any single assertion fails, step execution is immediately aborted, the entire scenario is marked as failed, and any subsequent steps (such as database queries or follow-up requests) are skipped.
- This prevents cascading failures, reduces network bandwidth waste, and keeps your test reports clean and focused on the root cause.

---

## 🧭 Dot-Path Prefixes & Resolvers

Gherkio uses a declarative dot-notation system to target specific parts of the response. The prefix of the path determines which resolver is invoked:

| Path Prefix | Target Component | Key Case-Sensitivity | Purpose | Example |
| :--- | :--- | :--- | :--- | :--- |
| `status` | HTTP Status Code | N/A | Validates the exact status code | `status: 200` |
| `body.<path>` | JSON Response Body | **Case-Sensitive** | Traverses response JSON structures | `body.user.email: email` |
| `headers.<key>` | HTTP Response Headers | **Case-Insensitive** | Inspects response headers | `headers.content-type: contains json` |
| `jwt.<claim>` | Decoded JWT payload | **Case-Sensitive** | Automatically decodes tokens and inspects claims | `jwt.role: admin` |
| `schema` | JSON Schema Contract | N/A | Enforces a full JSON schema contract | `schema: users/create` |
| `timing.duration` | Round-trip request time | N/A | Asserts latency SLA/performance | `timing.duration: lte 500ms` |

---

### 1. Response Body Traversal (`body.<path>`)

The `body.` prefix targets the response body payload (expected to be JSON). Gherkio traverses nested properties, list indices, and dictionary keys.

#### Key Syntax Features:
*   **Nested Navigation**: Use dots to descend into maps (e.g., `body.user.profile.firstName`).
*   **Array Indexing**: Use brackets to access specific list items (e.g., `body.items[0].id`).
*   **Case Sensitivity**: JSON keys are strictly case-sensitive. `body.userId` is different from `body.userid`.

#### Supported Matchers

| Assertion Value | Type | Example | Behavior |
|-----------------|------|---------|----------|
| Literal value | Equality | `body.role: admin` | Exact string match |
| `exists` | Existence | `body.email: exists` | Field must be present (any value) |
| `not exists` | Absence | `body.error: not exists` | Field must be absent |
| `uuid` | Format | `body.id: uuid` | Valid UUID v4 format |
| `email` | Format | `body.email: email` | Valid email format |
| `datetime` | Format | `body.createdAt: datetime` | Valid RFC3339 / ISO8601 datetime |
| `uri` | Format | `body.avatar: uri` | Valid URI format (e.g. https://...) |
| `string` | Type | `body.name: string` | Value is a string |
| `number` | Type | `body.price: number` | Value is numeric (int, float) |
| `boolean` | Type | `body.active: boolean` | Value is boolean |
| `array` | Type | `body.tags: array` | Value is an array |
| `object` | Type | `body.meta: object` | Value is an object/map |
| `null` | Type | `body.deletedAt: null` | Value is null |
| `true` / `false` | Literal | `body.active: true` | Value is boolean true/false |
| `contains <s>` | Substring | `body.message: contains error` | String contains substring |
| `startsWith <s>` | Prefix | `body.id: startsWith usr_` | String starts with prefix |
| `endsWith <s>` | Suffix | `body.email: endsWith @example.com` | String ends with suffix |
| `regex <p>` | Pattern | `body.code: regex ^[A-Z]{3}\d{4}$` | String matches regex |
| `schema: <name>` | Schema | `schema: users/profile` | Full body validated against schema |
| `schema: not <name>` | Negated schema | `schema: not errors/validation` | Body must NOT match error schema |

#### Negative Assertions with `not exists`
You can assert that a field is **absent** from the response body using `not exists`. This is useful for verifying that optional fields are omitted on purpose, or that error/cleanup fields are not present:

```yaml
expect:
  body.deletedAt: not exists    # soft-delete field should not appear
  body.error: not exists        # no error on success path
  body.archived: not exists     # field completely absent from payload
```

#### 💡 Intelligent Spelling Suggestion Engine
If you write an assertion for a path that does not exist in the response payload, Gherkio's smart diff engine doesn't just fail; it analyzes the keys that *were* actually present in the response and prints helpful spelling suggestions in your terminal:

```
✗ timing.duration = lte 500ms (actual: 312ms)
✗ body.accessToken = exists
  Reason: field 'accessToken' not found in response body
  Suggestions:
    - message
    - statusCode
```

---

### 2. HTTP Headers (`headers.<key>`)

The `headers.` prefix allows you to inspect HTTP response headers.

#### Key Syntax Features:
*   **Case-Insensitivity**: In accordance with the HTTP RFC, header lookups are fully case-insensitive. Gherkio normalizes these keys internally, meaning `headers.content-type`, `headers.Content-Type`, and `headers.CONTENT-TYPE` are completely equivalent.
*   **Multi-Value Headers**: If a header has multiple values, Gherkio normalizes them to a single comma-separated string for easy assertion matching (e.g. `headers.cache-control: contains no-cache`).

---

### 3. Automatic JWT Claim Decoding (`jwt.<claim>`)

One of Gherkio's most powerful native features is the automatic discovery and decoding of JSON Web Tokens.

#### How It Works:
1.  When a request completes, Gherkio scans the response body for standard JWT field names:
    *   `token`
    *   `accessToken`
    *   `access_token`
    *   `idToken`
    *   `id_token`
2.  If any of these fields are present and contain a valid base64-encoded, three-part JWT structure (`header.payload.signature`), Gherkio automatically decodes the payload section into memory.
3.  You can assert directly against claims in the payload using the `jwt.` prefix.

#### Example:
If the server returns:
```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjMiLCJyb2xlIjoiYWRtaW4iLCJleHAiOjE3MTY4ODQwMDB9.sig"
}
```
You can immediately assert:
```yaml
expect:
  jwt.sub: "123"
  jwt.role: "admin"
  jwt.exp: datetime  # Asserts that the expiration claim is present
```

---

### 4. JSON Schema Contracts (`schema`)

For absolute validation of deep objects and API payloads, individual key assertions can become unwieldy. The `schema` directive enforces full, deep-object JSON Schema validation.

#### Syntax:
```yaml
expect:
  schema: path/to/schema
```

#### How It Works:
*   **Path Resolution**: Gherkio looks for the schema file under your project's configured schemas directory (usually `.gherkio/schemas/`). It automatically appends `.yaml`, `.yml`, or `.json` to locate the file (e.g., `schema: users/profile` resolves to `.gherkio/schemas/users/profile.yaml`).
*   **Negation (`not`)**: You can assert that a response **does not** match a schema by prefixing the value with `not ` (e.g. `schema: not errors/validation`). This is ideal for testing error states where a validation error structure should *not* be returned.

---

### 5. Response Timing Assertions (`timing.duration`)

To enforce service-level agreements (SLAs) or target performance constraints, Gherkio includes a native latency-checking resolver:

*   **Syntax**: `timing.duration: <comparison-matcher> <duration>`
*   **Allowed Matchers**: Supports comparison matchers like `lte`, `lt`, `gte`, and `gt`.
*   **Duration Syntax**: A numeric value followed by `ms` (milliseconds) or `s` (seconds).

```yaml
expect:
  status: 200
  timing.duration: lte 500ms   # Step fails if request takes longer than 500ms
```

---

## 🔄 Dynamic Variables in Assertions

To avoid hardcoding values and to support end-to-end testing, Gherkio supports dynamic variable lookup directly in your assertion values.

### 1. Interpolating Saved & Global Variables (`$<variable>`)
Expected values inside the `expect` block are fully evaluated through the variable interpolation engine. This includes:
*   **Saved Variables**: Values extracted from previous steps using the `save` block (e.g. `$userId`).
*   **Environment & Credentials Variables**: Values loaded from the current environment YAML or credentials accounts (e.g. `$accounts.admin.email`).

```yaml
steps:
  - request:
      method: POST
      url: /v1/users
      body:
        email: "bob@example.com"
    save:
      newUserId: body.id

  - request:
      method: GET
      url: /v1/users/$newUserId
    expect:
      status: 200
      # Validates that the returned ID matches the ID saved from the previous step
      body.id: $newUserId
      # Validates against a global test account email from your environment config
      body.invitedBy: $accounts.admin.email
```

### 2. Current Request References (`request.body.<field>`)
When verifying creation or update steps, you often want to assert that the response body contains the exact same data sent in the request. You can refer to the *current request's body* fields dynamically using the `request.body.` prefix.

```yaml
steps:
  - request:
      method: POST
      url: /v1/products
      body:
        title: "Mechanical Keyboard"
        price: 99.99
    expect:
      status: 201
      # Asserts that the response payload echoes back the sent request fields dynamically
      body.data.title: request.body.title
      body.data.price: request.body.price
```

---

## 🗄️ Collection & Array Assertions

Gherkio provides advanced built-in collection resolvers to perform validations across lists without writing loops.

### 1. Array Size Validation (`count(<array-path>)`)

Asserts the exact length of a list returned in the response.

*   **Exact Count**: `count(body.items): 5`
*   **Comparators**: Append `.gt`, `.gte`, `.lt`, or `.lte` suffixes to the path to perform numeric bounds checks:
    *   `count(body.items).gte: 1` — Asserts that the array is not empty (has at least 1 item).
    *   `count(body.items).lt: 10` — Asserts that the array contains fewer than 10 items.

> [!NOTE]
> Unlike standard body paths, `count()` paths resolve directly against the parsed response body root. The `body.` prefix inside the parenthesis is optional, but supported.

---

### 2. Universal Collection Asserts (`all(<array-path>)`)

Asserts that **every single element** in a list satisfies a specific condition.

*   **Equality Check**: `all(body.items.status): active` — Asserts that every item in the `items` list has a `status` equal to `"active"`.
*   **Matcher Assertion**: `all(body.items.price): number` — Asserts that the `price` field in every element is a number.
*   **Negative Assertion**: `all(body.items.errors): not exists` — Asserts that the `errors` field is absent from every element.

| Condition | Example | Description |
|-----------|---------|-------------|
| Literal value | `all(body.items.role): admin` | Every element's field equals the value |
| Type matcher | `all(body.items.price): number` | Every element's field matches a type matcher |
| Format matcher | `all(body.items.email): email` | Every element's field matches a format matcher |
| Existence | `all(body.items.id): exists` | Every element has the field |
| **Absence** | `all(body.items.errors): not exists` | **No element has the field** |
| Collection matcher | `all(body.items.count): gte 1` | Every array element satisfies a count condition |

---

## 🌟 Comprehensive Real-World Example

Below is a complete, production-grade Gherkio test step demonstrating status codes, headers, nested body lookups, collection checks, automatic JWT assertions, and schema contract verification running side-by-side:

```yaml
scenario: Complete API contract and session validation
steps:
  - request:
      method: POST
      url: /v1/auth/session
      body:
        username: "admin"
        device: "laptop"
    expect:
      # 1. Enforce exact HTTP status code
      status: 201

      # 2. Case-insensitive header checks
      headers.Content-Type: contains json
      headers.x-transaction-id: exists

      # 3. Deep-path JSON body validations using matchers & dynamic variables
      body.session.id: uuid
      body.session.metadata.device: request.body.device     # Dynamic reference to current request payload
      body.session.metadata.tenant: $tenantId               # Interpolates variable saved from previous steps
      body.session.active: true

      # 4. Collection checks with comparison bounds
      count(body.session.permissions).gte: 3
      all(body.session.permissions): string

      # 5. Automatic JWT discovery and claims validation
      jwt.role: "administrator"
      jwt.sub: uuid

      # 6. Full API schema validation (located at .gherkio/schemas/auth/session-response.yaml)
      schema: auth/session-response
```
