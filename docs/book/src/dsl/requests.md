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

