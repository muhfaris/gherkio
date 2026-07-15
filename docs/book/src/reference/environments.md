# Environments Configuration

Target different deployment environments dynamically. All configurations are stored as standard YAML files inside the `.gherkio/environments/` directory.

---

## 📝 Directory Resolution & Naming

Gherkio matches the environment filename with the CLI `--env` (or `-e`) execution flag:

- `--env staging` ➔ loads `.gherkio/environments/staging.yaml`
- `--env local` ➔ loads `.gherkio/environments/local.yaml`

---

## ⚙️ Environment YAML Properties

| Key | Type | Required | Description | Example |
| :--- | :--- | :--- | :--- | :--- |
| `baseUrl` | `string` | Yes | Target domain for relative requests | `baseUrl: https://api.staging.net` |
| `services`| `object` | No | Microservice override pathways | (See below) |

---

## ⚡ Microservice Endpoint Overrides

In multi-service networks, different features (identity, checkout, inventory) are hosted on independent servers. Use the `services` map to cleanly route specific API requests:

```yaml
# .gherkio/environments/staging.yaml
baseUrl: https://staging.my-company.com

services:
  identity:
    baseUrl: https://auth-staging.my-company.net
  checkout:
    baseUrl: https://checkout-staging.my-company.net
```

When writing a step, reference the service key:
```yaml
steps:
  # Routes to https://auth-staging.my-company.net/v1/token
  - request:
      service: identity
      method: POST
      url: /v1/token
```
 This allows you to migrate identical testing logic between `local` (where all services run on local ports) and `production` with zero script modifications.

---

## 🎭 Service Mocking / Virtualization

Gherkio supports service virtualization directly within your environment file. You can define outbound mock intercepts to stub third-party API dependencies or simulate specific response conditions (e.g. failure states) without hitting real external endpoints.

```yaml
# .gherkio/environments/local.yaml
baseUrl: http://localhost:8080

mocks:
  - request:
      method: GET
      url: /api/external-service/status
    response:
      status: 200
      headers:
        Content-Type: application/json
      body:
        status: "healthy"
        service: "virtualized-dependency"
```

When the test runner encounters a step requesting a URL matching the defined mock request (`method` and `url`), it intercepts the request and instantly returns the configured `response` status, headers, and body.

### Intercept Variable Interpolation
Mock responses support dynamic variable interpolation using the context variables from the active test runner session, allowing you to return customized responses:
```yaml
mocks:
  - request:
      method: POST
      url: /api/echo
    response:
      status: 201
      body:
        message: "Hello {{username}}"
```
