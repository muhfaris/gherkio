# Request Configuration

The `request` block defines the outgoing HTTP request. Gherkio supports full-featured HTTP client configurations including custom verbs, header maps, service targeting, and flexible request body encodings.

---

## 🌐 HTTP Request Key Reference

| Key | Type | Required | Description | Example |
| :--- | :--- | :--- | :--- | :--- |
| `method` | `string` | Yes | HTTP Method (GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD) | `method: GET` |
| `url` | `string` | Yes | Target path (relative to `baseUrl`) or fully qualified absolute URL | `url: /v1/profile` |
| `query` | `map[string]string` | No | Query parameters appended to the URL. Supports variable interpolation in values. | `query: { status: available, limit: "10" }` |
| `service`| `string` | No | Target service key to override default `baseUrl` | `service: payment` |
| `headers`| `object` | No | Map of custom key-value HTTP headers | `headers: { Content-Type: application/json }` |
| `body` | `any` | No | Request payload. Can be JSON objects, lists, strings, or numbers | (See below) |
| `timeout`| `string` | No | Request-specific HTTP timeout override | `timeout: 45s` |

---

## 🛠️ Detailed Usage & Examples

### Targeting Specific Services

If your system uses a microservice architecture, map service endpoints in your environment file (e.g. `environments/staging.yaml`):
```yaml
baseUrl: https://staging.host.com
services:
  users: https://users-staging.host.com
  billing: https://billing-staging.host.com
```

Invoke service routing in your YAML request:
```yaml
steps:
  - request:
      service: users            # Resolves to https://users-staging.host.com/v1/profile
      method: GET
      url: /v1/profile
```

### JSON Request Payload

JSON bodies are defined natively in Gherkio YAML and fully support runtime variable interpolation:

```yaml
steps:
  - request:
      method: POST
      url: /register
      headers:
        Content-Type: application/json
      body:
        username: $randomEmail   # Interpolated using built-in generator
        profile:
          firstName: "John"
          lastName: "Doe"
          age: ${randomInt(18,65)}
```

### Query Parameters (`query`)

Use the `query` block to append URL query parameters instead of embedding them directly in the URL string. This keeps URLs clean and allows variable interpolation for dynamic values:

```yaml
steps:
  - request:
      method: GET
      url: /pets/findByStatus
      query:
        status: available
        limit: "10"
    expect:
      status: 200
```

Query parameter values support full variable interpolation:

```yaml
steps:
  - request:
      method: GET
      url: /users
      query:
        role: $userRole
        page: "${randomInt(1,5)}"
    expect:
      status: 200
```

---

### Multipart Form Data (`multipart`)

To test form-based submissions or file uploads, Gherkio features an elegant `multipart` block. When `multipart` is defined, Gherkio automatically builds the boundary formatting and sets the proper standard HTTP header (`multipart/form-data; boundary=...`) for you.

You can mix raw fields and multiple file attachments inside a single payload:

```yaml
steps:
  - request:
      method: POST
      url: /v1/user/profile
      multipart:
        fields:
          username: "john_doe"
          role: $accounts.admin.role   # Support dynamic variables
        files:
          # 1. Simple path string syntax:
          avatar: "assets/john-avatar.png"
          
          # 2. Advanced schema mapping (with custom MIME types and filenames):
          document:
            path: "assets/resume.pdf"
            contentType: "application/pdf"
            filename: "john_doe_cv.pdf"
    expect:
      status: 200
      body.updated: true
```

#### 📁 Assets Directory Location & Project Tree

File attachments specified under `multipart.files` are resolved relative to your project workspace root directory (the parent directory of `.gherkio/`). By convention, create an `assets/` directory at your project root to store test media files (images, avatars, PDFs):

You can configure that default directory in `.gherkio/config.yaml`:

```yaml
assets:
  path: assets
```

The path is relative to the project root, although an absolute directory is also accepted. With this configuration, a multipart value such as `avatar: "john-avatar.png"` resolves to `<projectRoot>/assets/john-avatar.png`. If `assets.path` is omitted, the existing project-root and fixtures lookup behavior remains unchanged.

```
my-api-tests/                      <-- Project Workspace Root
├── .gherkio/                      <-- Gherkio Test Engine Config & Suites
│   ├── config.yaml
│   ├── environments/
│   │   └── local.yaml
│   └── tests/
│       └── user/
│           └── upload-avatar.yaml
└── assets/                        <-- Assets Directory for Multipart File Uploads
    ├── john-avatar.png
    ├── sample-image.jpg
    └── resume.pdf
```

#### Custom Assets Directory

To keep upload fixtures inside `.gherkio`, set a custom path relative to the project root:

```yaml
# .gherkio/config.yaml
assets:
  path: .gherkio/test-assets
```

The corresponding project structure is:

```text
my-api-tests/
└── .gherkio/
    ├── config.yaml
    ├── test-assets/
    │   ├── avatars/
    │   │   └── john.png
    │   └── documents/
    │       └── resume.pdf
    └── tests/
        └── upload.yaml
```

Files are then referenced relative to the configured directory:

```yaml
multipart:
  files:
    avatar: "avatars/john.png"
    resume:
      path: "documents/resume.pdf"
      contentType: "application/pdf"
```

The resolved paths are `.gherkio/test-assets/avatars/john.png` and `.gherkio/test-assets/documents/resume.pdf` under the project root.

An absolute assets directory is also supported when the files live outside the project:

```yaml
assets:
  path: /opt/gherkio/shared-assets
```

In that case, `avatar: "avatars/john.png"` resolves to `/opt/gherkio/shared-assets/avatars/john.png`. Absolute paths make a configuration machine-specific, so relative paths are preferable for repositories and CI environments.

#### 💡 Key Details & File Path Resolution:

- **`multipart.fields`**: A flat map of key-value string pairs representing text form fields.
- **`multipart.files`**: A map of files. The field key is the form property name (e.g. `avatar`).
- **Flexible Directory Paths**: You can configure **any custom directory or path name** in your scenario step (e.g. `"assets/avatar.png"`, `"media/uploads/doc.pdf"`, or `"fixtures/test.csv"`).
- **Configured Assets Directory**: A bare relative file such as `"avatar.png"` is also looked up under the optional `assets.path` from `.gherkio/config.yaml`.
- **Search & Resolution Order**: When resolving a file, Gherkio automatically looks up the file in the following order:
  1. **Absolute Path**: If an absolute path is specified (e.g. `/tmp/test-image.png`)
  2. **Declared Path**: `<projectRoot>/<declared-path>` (e.g. `my-project/assets/john-avatar.png` or `my-project/custom-folder/file.jpg`)
  3. **Configured Assets Directory**: `<projectRoot>/<assets.path>/<declared-path>` when configured
  4. **Project Fixtures Fallback**: `<projectRoot>/fixtures/<filename>`
  5. **Gherkio Fixtures Fallback**: `<projectRoot>/.gherkio/fixtures/<filename>`
  6. **Current Working Directory**: Relative path from where `gherkio run` was executed.
- **MIME Detection**: If using the simple string syntax, Gherkio automatically infers the `Content-Type` of the file from its file extension (e.g. `.png` ➔ `image/png`, `.jpg` ➔ `image/jpeg`). Use the advanced map syntax (`path`, `contentType`, `filename`) to specify explicit overrides.



---

### 🔄 Declarative Collection Projections (`transform`)

Gherkio allows you to dynamically filter, slice, project, and reshape array collections from your saved variables directly into your request payload target path before the HTTP request is dispatched.

#### Transform Block Key Reference:
| Key | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `from` | `string` | Yes | The source collection variable name (must start with `$`). |
| `as` | `string` | No | Variable alias to represent each element context during transformation (defaults to `"item"`). |
| `where` | `object` | No | Map of filters applying standard Gherkio matchers. |
| `limit` | `int` | No | Maximum count of matching projected elements to output. |
| `select` | `object` | Yes | Custom structural schema mapping. Supports recursive/nested projection maps. |

#### ⚡ Explicit Type Casting inside Projections
When mapping data from target APIs to your request payloads, different endpoints might expect different types. Gherkio provides explicit type-casting functions inside the projection's `select` block:
*   **`$string(var)`**: Converts a field to a string (e.g., `$string(item.id)` turns `1001` to `"1001"`).
*   **`$int(var)`**: Coerces/parses a field value into an integer.
*   **`$float(var)`**: Coerces/parses a field value into a float64.
*   **`$bool(var)`**: Coerces/parses truthy string or integer values into a boolean.

#### 🌀 Conditional Value Selection (`$if`)
Values can vary conditionally using the **`$if(condition, thenValue, elseValue)`** function. This works inside transform `select` blocks and also directly in request `body` strings.

*   **`condition`**: A scoped variable path to evaluate for truthiness (e.g. `item.is_answered` or `is_active`).
*   **`thenValue`**: Value expression used when the condition is truthy.
*   **`elseValue`**: Value expression used when the condition is falsy (optional — if omitted, evaluates to `null`).

Truthiness rules: `nil`, `false`, `0`, and `""` are falsy; all other values are truthy.

**Inside transform `select` (scoped to item alias):**
```yaml
select:
  # Map different fields based on question type
  answer_value: "$if(q.is_answered, q.free_text_answer, q.default_answer)"
  
  # Use type casting inside conditionals
  quantity: "$if(item.in_stock, $int(item.qty), 0)"
  
  # No else clause — returns null when condition is falsy
  optional_field: "$if(item.has_value, item.value)"
```

**Inside request `body` (use `$` prefix for variable references):**
```yaml
body:
  # Conditionally map based on a saved variable
  status: "$if(is_admin, $admin_endpoint, $user_endpoint)"
  
  # Use type casting with conditionals
  count: "$if(has_items, $int(item_count), 0)"
```

#### Complete Example:
```yaml
steps:
  # Step 1: Fetch details
  - request:
      method: GET
      url: /surveys/132
    save:
      raw_questions: body.questions

  # Step 2: Reshape and submit survey answers
  - request:
      method: POST
      url: /surveys/132/submit
      body:
        survey_id: 132
      transform:
        # Dynamically inject the projected array under body.answers
        answers:
          from: $raw_questions
          as: q
          where:
            q.is_required: true
          limit: 10
          select:
            # Explicitly cast integer id from details response to string for submission!
            question_id: "$string(q.id)"
            
            # Coerce the sequence number to integer
            seq: "$int(q.seq)"
            
            # Static values and clean variable mapping
            is_answered: true
            free_text_answer: q.user_response
```


---

### 💡 Type Preservation & Explicit Casting in Standard Bodies

Outside of the `transform` projection block, Gherkio automatically preserves the original type of standalone variables mapped inside the HTTP request `body`:
*   If you map a saved integer variable: `employee_id: $employee_id`, Gherkio keeps the value as a JSON number (`1234`) in the outgoing request payload.
*   If you explicitly want to coerce it to a string (or other types), wrap the variable in a casting operator inside the request body:
    ```yaml
    body:
      emp_id: "$string(employee_id)"   # Outgoing payload: "emp_id": "1234"
      status: "$bool(is_active)"       # Outgoing payload: "status": true
    ```
