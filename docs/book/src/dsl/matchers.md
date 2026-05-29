# Matcher Library

Gherkio includes a powerful, built-in library of **25+ assertion matchers**. These matchers allow you to perform rich semantic validations—ranging from format verification (like UUIDs, email addresses, and datetimes) to mathematical range boundaries, string manipulations, and collection inspection—without writing any code or custom assertions.

---

## 📚 Complete Matcher Quick-Reference

| Keyword | Example | Target Data Must Be... |
| :--- | :--- | :--- |
| **Existence** | | |
| `exists` | `body.id: exists` | Present in the payload and not null. |
| `not exists` | `body.deletedAt: not exists` | Completely absent from the payload. |
| **Type Verification** | | |
| `string` | `body.name: string` | A JSON string. |
| `number` | `body.price: number` | Any integer or decimal/float value. |
| `boolean` | `body.active: boolean` | A boolean value (`true` or `false`). |
| `array` | `body.tags: array` | A JSON array list. |
| `object` | `body.metadata: object` | A JSON object map. |
| `null` | `body.archivedAt: null` | Strictly a JSON `null`. |
| `true` | `body.completed: true` | Strictly boolean `true`. |
| `false` | `body.pending: false` | Strictly boolean `false`. |
| **String Operations** | | |
| `contains <value>` | `body.msg: contains Success` | A string containing the exact `<value>`. |
| `startsWith <val>` | `body.sku: startsWith LAP-` | A string starting with prefix `<val>`. |
| `endsWith <val>` | `body.email: endsWith .com` | A string ending with suffix `<val>`. |
| `regex <pattern>` | `body.code: regex ^[A-Z]{3}$`| A string matching the Go regular expression. |
| **Numeric Bounds** | | |
| `gt <value>` | `body.rating: gt 4` | Greater than `<value>`. |
| `gte <value>` | `body.price: gte 9.99` | Greater than or equal to `<value>`. |
| `lt <value>` | `body.age: lt 18` | Less than `<value>`. |
| `lte <value>` | `body.limit: lte 100` | Less than or equal to `<value>`. |
| **Format Validators** | | |
| `uuid` | `body.id: uuid` | A valid UUID v4 format string. |
| `email` | `body.email: email` | A valid RFC-compliant email address. |
| `datetime` | `body.createdAt: datetime` | A valid RFC3339 / ISO8601 datetime string. |
| `uri` | `body.url: uri` | A valid absolute URI string. |
| `ipv4` | `body.ip: ipv4` | A valid IPv4 address (e.g. `192.168.1.1`). |
| `ipv6` | `body.ip: ipv6` | A valid IPv6 address (supports compressed). |
| `base64` | `body.data: base64` | A valid Base64 or URL-Safe Base64 string. |
| `mac` | `body.address: mac` | A valid MAC address format. |
| **State Inspection** | | |
| `empty` | `body.items: empty` | An empty string `""`, empty list `[]`, empty map `{}`, or `null`. |

---

## 🔍 Category Deep-Dive & Syntax Examples

### 1. Existence Checkers
These matchers verify whether a key is present or absent in the response payload structure.

*   **`exists`**: Evaluates to true if the field is present, regardless of its value (including empty values).
*   **`not exists`**: Evaluates to true only if the key is completely omitted from the JSON payload.

```yaml
expect:
  body.id: exists
  body.deletedAt: not exists
```

---

### 2. Type Enforcers & Booleans
Ensures response elements conform strictly to expected JSON data types.

*   **`string`**: Fails if the value is a number, boolean, array, or object.
*   **`number`**: Validates integers, floats, and scientific JSON numbers.
*   **`boolean`**: Matches either `true` or `false` booleans.
*   **`array`**: Verifies the field is a JSON list.
*   **`object`**: Verifies the field is a structured map/object.
*   **`null`**: Verifies the value is `null`.
*   **`true` / `false`**: Value-specific boolean enforcers.

```yaml
expect:
  body.profile.bio: string
  body.profile.age: number
  body.profile.isVerified: boolean
  body.tags: array
  body.preferences: object
  body.finished: true
  body.error: null
```

---

### 3. String Operations (Argument-based)
Performs text assertions on target string fields.

*   **`contains <value>`**: Checks if the target string contains the substring `<value>` (case-sensitive).
*   **`startsWith <value>`**: Checks if the target string begins with the prefix `<value>`.
*   **`endsWith <value>`**: Checks if the target string ends with the suffix `<value>`.
*   **`regex <pattern>`**: Matches the string against a standard Go regular expression.

```yaml
expect:
  body.description: contains Mechanical
  body.sku: startsWith KBD-
  body.supportEmail: endsWith @example.com
  body.productCode: regex ^[A-Z]{3}-\d{4}$
```

---

### 4. Mathematical Bounds (Argument-based)
Used to enforce numeric restrictions. Gherkio automatically coerces both the target response field and the matcher argument into `float64` floating points for maximum comparison precision.

*   **`gt <value>`**: Greater than (`>`).
*   **`gte <value>`**: Greater than or equal to (`>=`).
*   **`lt <value>`**: Less than (`<`).
*   **`lte <value>`**: Less than or equal to (`<=`).

```yaml
expect:
  body.rating: gt 4.2
  body.stock: gte 0
  body.discount: lt 1.0
  body.itemsCount: lte 100
```

---

### 5. Format & Protocol Validators
Performs regex-based and parser-based syntax verification on strings to enforce RFC protocols.

*   **`uuid`**: Asserts strict compliance with UUID v4 syntax.
*   **`email`**: Validates structural email address syntax (presence of `@`, valid domain blocks).
*   **`datetime`**: Parses string against standard RFC3339 / ISO8601 structures (e.g. `2026-05-29T10:12:15Z`).
*   **`uri`**: Verifies that the string is a valid absolute URI.
*   **`ipv4` / `ipv6`**: Enforces valid IP structures, ensuring IPv4 segments fall between `0-255` and IPv6 fields support compressed forms.
*   **`base64`**: Attempts standard base64 decoding and falls back to URL-safe raw base64 to ensure absolute compatibility.
*   **`mac`**: Validates standard colon-separated or hyphen-separated MAC addresses (e.g., `00:1a:2b:3c:4d:5e`).

```yaml
expect:
  body.userId: uuid
  body.contactEmail: email
  body.createdAt: datetime
  body.imageLink: uri
  body.serverIp: ipv4
  body.payloadHash: base64
  body.deviceMac: mac
```

---

### 6. Emptiness Checker
The **`empty`** matcher is a unified helper designed to handle collections, strings, and missing records consistently:
*   Returns `true` if target is a string equal to `""`.
*   Returns `true` if target is a JSON list containing `0` items.
*   Returns `true` if target is a JSON object map containing `0` properties.
*   Returns `true` if target is `null`.

```yaml
expect:
  # Validates that list is empty
  body.items: empty
  # Validates that custom message string is blank
  body.alertMessage: empty
```

---

## 🔄 Matchers inside Universal Collections (`all`)

Matchers are fully integrated into Gherkio's collection engine. When performing assertions on arrays, you can pass **any matcher keyword** as the target check inside an `all()` block:

```yaml
expect:
  # Enforces that every element in the permissions array is a valid UUID
  all(body.users.id): uuid

  # Enforces that every transaction amount is positive
  all(body.transactions.amount): gt 0

  # Enforces that every tags list is a valid array
  all(body.products.tags): array
```
