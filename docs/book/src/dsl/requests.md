# Request Configuration

The `request` block defines the outgoing HTTP request. Gherkio supports full-featured HTTP client configurations including custom verbs, header maps, service targeting, and flexible request body encodings.

---

## 🌐 HTTP Request Key Reference

| Key | Type | Required | Description | Example |
| :--- | :--- | :--- | :--- | :--- |
| `method` | `string` | Yes | HTTP Method (GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD) | `method: GET` |
| `url` | `string` | Yes | Target path (relative to `baseUrl`) or fully qualified absolute URL | `url: /v1/profile` |
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

#### 💡 Key Details:
- **`multipart.fields`**: A flat map of key-value string pairs representing text form fields.
- **`multipart.files`**: A map of files. The field key is the form property name (e.g. `avatar`).
- **File Resolution**: Gherkio resolves relative file paths relative to your Gherkio project directory (`.gherkio` parent directory).
- **MIME Detection**: If using the simple string syntax, Gherkio automatically infers the `Content-Type` of the file from its file extension (e.g. `.png` ➔ `image/png`). Use the advanced map syntax to specify overrides explicitly.

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


