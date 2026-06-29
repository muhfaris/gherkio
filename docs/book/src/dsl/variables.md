# Variables & Generators

Variables allow you to pass state dynamically, randomize inputs, and secure secrets without hardcoding values in test scenarios. Gherkio evaluates and injects variables dynamically per step.

---

## ⚡ Declaring & Using Variables

Variables are prefixed with `$` (e.g., `$accessToken`). 

For string boundary isolation inside header fields or request bodies, wrap them in curly braces:
- `Authorization: "Bearer ${accessToken}"`

### Default Fallbacks
If a variable might not be defined under the current context, you can specify a default fallback:
- `role: "${role:user}"` ➔ resolves to the value of `$role` if defined, otherwise defaults to `"user"`.

### Array Indexing (Bracket Notation)
If a saved variable contains an array, you can access individual elements by index using bracket notation:
- `$tags[0].id` ➔ accesses the `id` field of the first element in the `$tags` array
- `$tags[${randomInt(0,4)}].id` ➔ picks a random element each time (useful with retry)
- `$items[2].name` ➔ accesses the `name` field of the third element

This works in all variable interpolation contexts: request body, headers, URLs, save paths, and assertion values.

**Example** — randomly pick an issue tag from a saved array:

```yaml
steps:
  - request:
      method: GET
      url: /v1/issue-tags/l3
    save:
      issueTags: body.data          # save the full array

  - request:
      method: POST
      url: /v1/tickets
      body:
        issue_tag_id: $issueTags[${randomInt(0,4)}].id   # random tag each time
```

Combined with `retry`, each attempt picks a different random index — useful for avoiding resource conflicts:

```yaml
  - request:
      method: POST
      url: /v1/tickets
      body:
        issue_tag_id: $issueTags[${randomInt(0,9)}].id
        name: "ticket-${randomInt}"
    retry:
      attempts: 5
      interval: 300
      backoff: constant
      onStatus: [409]
    expect:
      status: 200
```

---

## 🎲 Built-in Single-Value Generators

Gherkio dynamically generates fresh values once per step for these standard single-value variables:

| Variable | Output Type | Example Output | Description |
| :--- | :--- | :--- | :--- |
| `$uuid` | `string` | `a1b2c3d4-e5f6-4789-abcd-ef1234567890` | Valid UUID v4 string |
| `$ulid` | `string` | `01ARZ3NDEKTSV4RRFFQ69G5FAV` | Monotonically increasing ULID |
| `$randomInt` | `integer` | `482910` | Random integer between `0` and `999999` |
| `$randomEmail` | `string` | `user_182938@example.com` | Random email address with a numeric suffix |
| `$randomPhone` | `string` | `+6281234567890` | Random Indonesian format phone number (default fallback) |

---

## 🛠️ Dynamic Generator Functions

For more specialized payloads, Gherkio provides a collection of **parameterized generator functions**. These are declared using the `${functionName(args)}` syntax.

### 1. Numeric & String Randomizers

*   **`${randomInt(min, max)}`**: Generates a random integer within a custom inclusive range.
    *   *Example*: `${randomInt(1, 100)}` ➔ `42`
*   **`${randomString(length, charset)}`**: Generates a random string of a specified length using a selected character set (`alpha`, `numeric`, `alphanumeric`).
    *   *Example*: `${randomString(12, "alphanumeric")}` ➔ `a8F2k9Pq3xLm`
*   **`${randomPhone(countryOrPrefix)}`**: Generates a random phone number based on a country code or a raw dialing prefix.
    *   **Country Codes**: Supports mapping major ISO countries to standard mobile layouts. Valid codes include:
        *   `ID` (Indonesia): `+628...`
        *   `US` / `CA` (USA / Canada): `+1...`
        *   `GB` / `UK` (United Kingdom): `+447...`
        *   `SG` (Singapore): `+658...`
        *   `JP` (Japan): `+8190...`
        *   `DE` (Germany): `+491...`
        *   `FR` (France): `+336...`
        *   `AU` (Australia): `+614...`
        *   `IN` (India): `+919...`
        *   `CN` (China): `+861...`
        *   `BR` (Brazil): `+55119...`
        *   `RU` (Russia): `+79...`
        *   `ZA` (South Africa): `+277...`
        *   `KR` (South Korea): `+8210...`
        *   `NL` (Netherlands): `+316...`
        *   `ES` (Spain): `+346...`
        *   `IT` (Italy): `+393...`
        *   `MY` (Malaysia): `+601...`
        *   `PH` (Philippines): `+639...`
        *   `TH` (Thailand): `+668...`
        *   `VN` (Vietnam): `+849...`
    *   **Raw Dialing Prefix**: You can pass any standard dialing prefix directly (e.g. `+351` or `351`). Gherkio automatically ensures it has the `+` prefix and appends 9 random digits.
    *   **Default Fallback**: Empty/unrecognized inputs automatically fallback to the Indonesian mobile (`ID`) format.
    *   *Example (ISO Code)*: `${randomPhone("SG")}` ➔ `+658591823`
    *   *Example (Raw Prefix)*: `${randomPhone("+351")}` ➔ `+351982738192`

---

### 2. Time & Date Offset Functions

*   **`${timestamp()}`**: Returns the current Unix timestamp (seconds since epoch) as a string.
    *   *Example*: `${timestamp()}` ➔ `1779958329`
*   **`${timestampMs()}`**: Returns the current Unix timestamp in milliseconds.
    *   *Example*: `${timestampMs()}` ➔ `1779958329000`
*   **`${dateNow(format)}`**: Formats the current time using a custom layout. Defaults to Go layout `2006-01-02 15:04:05`.
    *   *Example*: `${dateNow("2006-01-02")}` ➔ `2026-05-29`
*   **`${dateOffset(offset, format)}`**: Offsets the current date/time by days (`d`), hours (`h`), minutes (`m`), or seconds (`s`). The `format` argument is optional (defaults to `2006-01-02`).
    *   *Example (14 days forward)*: `${dateOffset("+14d")}` ➔ `2026-06-12`
    *   *Example (2 hours back)*: `${dateOffset("-2h", "2006-01-02 15:04:05")}` ➔ `2026-05-29 12:30:15`

> [!IMPORTANT]
> **Understanding Go's Date/Time Layouts**
>
> Gherkio uses Go's native reference-time formatting under the hood instead of traditional C-style `strftime` patterns (such as `%Y-%m-%d`). 
> 
> To define a layout, use values from Go's canonical reference date: **`January 2, 2006 at 3:04:05 PM MST`** (representing components `1 2 3 4 5 6 -0700`).
>
> | Desired Format | Go Layout Pattern | Example Output |
> | :--- | :--- | :--- |
> | **ISO 8601 / RFC 3339** | `2006-01-02T15:04:05Z07:00` | `2026-05-29T17:24:14Z` |
> | **YYYY-MM-DD** | `2006-01-02` | `2026-05-29` |
> | **DD/MM/YYYY** | `02/01/2006` | `29/05/2026` |
> | **MM-DD-YYYY** | `01-02-2006` | `05-29-2026` |
> | **24-Hour Time** | `15:04:05` | `17:24:14` |
> | **12-Hour Time AM/PM** | `03:04:05 PM` | `05:24:14 PM` |
> | **Full Month & Day** | `January 2, 2006` | `May 29, 2026` |
> | **Short Month & Year** | `Jan 02, 06` | `May 29, 26` |
>
> ⚠️ **Warning:** Traditional formats like `%Y-%m-%d` or `yyyy-MM-dd` **are not parsed** and will be returned as literal strings (e.g. `%Y-%m-%d`). Only use the reference date numbers above.
>
> ---
>
> ### 🛠️ Creating Custom Layouts
> 
> You are not limited to the presets above! You can construct **any custom date/time format** by mixing your preferred punctuation (such as slashes `/`, dots `.`, hyphens `-`, colons `:`, or spaces) with Go's reference date tokens:
> 
> *   **Slash Separated**: `${dateNow("2006/01/02")}` ➔ `2026/05/29`
> *   **Dotted Format**: `${dateNow("02.01.2006")}` ➔ `29.05.2026`
> *   **Millisecond Precision**: `${dateNow("15:04:05.000")}` ➔ `17:28:04.921`
> *   **Written Sentence**: `${dateNow("Monday, Jan 2, 2006")}` ➔ `Friday, May 29, 2026`

---

### 3. Encoders & Text Formatters

*   **`${base64(data)}`**: Encodes the given string parameter to Base64.
    *   *Example*: `${base64("hello")}` ➔ `aGVsbG8=`
*   **`${base64Decode(encoded)}`**: Decodes a Base64 string.
    *   *Example*: `${base64Decode("aGVsbG8=")}` ➔ `"hello"`
*   **`${urlencode(data)}`**: Escapes special characters for use in URI query parameters.
    *   *Example*: `${urlencode("hello world!")}` ➔ `hello+world%21`
*   **`${urldecode(encoded)}`**: Unescapes URL characters.
    *   *Example*: `${urldecode("hello+world%21")}` ➔ `"hello world!"`
*   **`${toUpper(text)}`**: Converts the string to uppercase.
    *   *Example*: `${toUpper("admin")}` ➔ `"ADMIN"`
*   **`${toLower(text)}`**: Converts the string to lowercase.
    *   *Example*: `${toLower("TOKEN")}` ➔ `"token"`
*   **`${trim(text)}`**: Trims leading and trailing whitespaces.
    *   *Example*: `${trim("   secret   ")}` ➔ `"secret"`

---

### 4. Cryptographic Hashing & HMAC

*   **`${hash(algorithm, data)}`**: Generates a hexadecimal hash of the data string. Supported algorithms: `md5`, `sha1`, `sha256`.
    *   *Example*: `${hash("sha256", "gherkio")}` ➔ `0f845d4c82c3c97d...`
*   **`${hmac(algorithm, key, message)}`**: Generates a hexadecimal keyed-hash message authentication code (HMAC). Supported algorithms: `md5`, `sha1`, `sha256`.
    *   *Example*: `${hmac("sha256", "my-secret", "message-data")}` ➔ `a8d7c49f...`

---

## 📊 Variable Precedence

When variable names overlap under the same test execution context, Gherkio resolves the values in the following order of precedence (highest overrides lowest):

```
[1. Host Environment Variables]  <-- Sourced from host environment (GHERKIO_ prefix only)
               ↓
[2. Selected Account Credentials] <-- Loaded from active credential file
               ↓
[3. Step Saves]                  <-- Extracted dynamically from prior steps
               ↓
[4. Built-in Generators]         <-- Generated fresh per step
```

> ⚠️ **Security Warning:** Only host environment variables prefixed with `GHERKIO_` are loaded. Normal environment variables like `PATH`, `USER`, or `SECRET_KEY` are ignored to prevent accidental leaks.
