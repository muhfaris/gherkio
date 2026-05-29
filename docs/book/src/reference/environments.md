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
  identity: https://auth-staging.my-company.net
  checkout: https://checkout-staging.my-company.net
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
