# Project Setup

Learn how Gherkio structures its folders, configures environments, and maps services for declarative integration testing.

---

## 📁 The `.gherkio/` Directory Layout

All Gherkio configuration, variables, tests, schemas, and reports reside inside a single, hidden `.gherkio/` folder at the root of your repository:

```
.gherkio/
├── config.yaml                     # Global test execution config
├── credentials/                    # Environment-specific usernames & keys
│   └── local.yaml
├── environments/                   # Target API hosts and service mappings
│   └── local.yaml
├── schemas/                        # Reusable JSON/YAML validation schemas
│   └── example/
│       ├── login-response.yaml
│       └── user-response.yaml
├── tests/                          # Your declarative YAML test scenario suites
│   └── example/
│       ├── accounts/
│       │   └── login-as-alice.yaml
│       ├── auth/
│       │   ├── login.yaml
│       │   ├── me.yaml
│       │   └── refresh.yaml
│       └── builtins/
│           └── login-with-generators.yaml
└── reports/                        # Automatically generated test run logs & HTML reports
    ├── archive/
    ├── failures/                   # Dynamic JSON snapshots for failed assertions
    └── latest/
```

---

## ⚙️ Global Configuration (`config.yaml`)

The global configuration file handles execution defaults, reporting, credential masking, and request sandboxing. Gherkio uses these fields to coordinate runtime behavior:

```yaml
gherkio_version: 0.1.0

# ----------------------------------------------------------------------
# 1. Project Metadata
# ----------------------------------------------------------------------
project:
  name: "my-api-testing-suite"
  version: "1.0.0"

# ----------------------------------------------------------------------
# 2. Environments Directory Configuration
# ----------------------------------------------------------------------
environments:
  default: local                    # Default environment if --env is omitted
  path: .gherkio/environments       # Path to environment configurations

# ----------------------------------------------------------------------
# 3. Tests & Scenarios Configuration
# ----------------------------------------------------------------------
tests:
  path: .gherkio/tests              # Directory where test scenarios reside

# ----------------------------------------------------------------------
# 4. JSON Schemas Configuration
# ----------------------------------------------------------------------
schemas:
  path: .gherkio/schemas            # Directory containing validation schemas

# ----------------------------------------------------------------------
# 5. Reports & Core Failure Snapshots
# ----------------------------------------------------------------------
reports:
  path: .gherkio/reports            # Output path for test summaries
  format: html                      # Report format ("html", "json", or "html,json")
  archive: true                     # Keep a history of past reports
  retention: 10                     # Keep up to 10 historical reports
  maskSensitive: true               # Mask credentials in HTML report pages
  failures:
    enabled: true                   # Write detailed JSON debug snapshots on failures
    path: .gherkio/reports/failures # Directory for failure snapshots
    maskSensitive: true             # Mask credentials in failure dumps
    retainCount: 50                 # Retain up to 50 debug snapshots

# ----------------------------------------------------------------------
# 6. Security and Sandboxing (Policy Engine)
# ----------------------------------------------------------------------
security:
  mask:
    enabled: true                   # Enable masking of sensitive fields in outputs
    fields:                         # Case-insensitive sensitive keys to redact
      - password
      - token
      - secret
      - Authorization
  sandboxing:
    enabled: true                   # Enforce outbound network boundaries
    allowedDomains:                 # Allowed target domains (supports wildcards)
      - "*.company.com"
      - "api.stripe.com"
    blockedDomains:                 # Explicitly blocked targets
      - "analytics.untrusted.com"
    blockPrivateSubnets: true       # Prevent loopback (127.0.0.1) and private IP calls
```

---

## 🔒 Security & Sandboxing Policies

Gherkio includes a native, industrial-grade **outbound traffic sandbox** and **credential masking engine** to secure enterprise testing pipelines and prevent security risks (like SSRF or private data leaks).

### 1. Credentials & Token Masking (`security.mask`)

To prevent sensitive keys, tokens, or credentials from leaking into test summaries, build logs, and console outputs, Gherkio scans response JSON keys and HTTP headers to mask matching fields.

*   **`enabled`**: (`boolean`) Turns output redacting on or off.
*   **`fields`**: (`[]string`) List of case-insensitive JSON paths or header keys to mask. When found, values are replaced with `[MASKED]`.
    *   **Built-in Defaults (automatically masked)**: `token`, `accessToken`, `access_token`, `refreshToken`, `refresh_token`, `password`, `secret`, `clientSecret`, `client_secret`, `apiKey`, `api_key`, and `authorization` (case-insensitive). You can add custom fields to this list; the built-in defaults remain active.

### 2. Domain & Outbound Sandboxing (`security.sandboxing`)

The sandboxing engine controls which external network servers Gherkio is allowed to communicate with. If a test scenario attempts to hit a blocked or unlisted host, Gherkio halts the step immediately before performing the HTTP call.

*   **`enabled`**: (`boolean`) Enforces network boundaries.
*   **`allowedDomains`**: (`[]string`) White-list of target endpoints. Supports simple wildcards (e.g. `*.company.com` matches `api.company.com` and `web.company.com`). If left empty, all domains not explicitly blocked are allowed.
*   **`blockedDomains`**: (`[]string`) Explicit black-list of endpoints. Take precedence over allowed domains.
*   **`blockPrivateSubnets`**: (`boolean`) When set to `true`, Gherkio resolves the target domain's IP addresses via DNS lookup prior to execution. If the resolved IP belongs to a local or private subnet (e.g. `127.0.0.1`, `::1`, `10.0.0.0/8`, `192.168.0.0/16`), the request is immediately rejected. This prevents SSRF (Server-Side Request Forgery) attacks and unauthorized internal port scanning.


---

## 🌐 Configuring Environments

Managing tests across local, staging, and production environments can easily lead to configuration bloat. Gherkio solves this using a **declarative environment mapping strategy**. 

All target environment host files reside as separate YAML configurations inside `.gherkio/environments/` (e.g. `local.yaml`, `staging.yaml`, `production.yaml`). You switch environments instantly via the CLI flag:
```bash
gherkio run example/auth/login.yaml --env staging
```

---

### 1. The Environment Configuration Schema

Each environment file has a simple, clean, but highly extensible schema:

| YAML Key | Type | Required | Description | Example |
| :--- | :--- | :--- | :--- | :--- |
| `baseUrl` | `string` | **Yes** | The default host URL. Used as a prefix for all relative URL request steps. | `baseUrl: https://api.staging.company.com` |
| `services` | `map[string]Service` | No | Overrides for specific microservices or downstream third-party APIs. | (See below) |
| `services.<name>.baseUrl` | `string` | **Yes** | The service-specific host URL prefix. | `baseUrl: http://auth-service:3000` |

---

### 2. Multi-Service Microservice Routing

In modern cloud-native architectures, single scenarios often span multiple distinct backend microservices (e.g. creating a user on the Auth API, making a charge on the Payment API, and verifying stock on the Inventory API). 

Gherkio allows you to declare all service endpoints in a single environment configuration, and route requests dynamically on a step-by-step basis:

#### Environment Config: `.gherkio/environments/staging.yaml`
```yaml
# Global default fallback
baseUrl: https://api.staging.company.com

# Microservice host overrides
services:
  auth:
    baseUrl: https://auth.staging.company.internal
  payments:
    baseUrl: https://payments.staging.company.internal
  notifications:
    baseUrl: http://localhost:8080   # Port-forwarded or local mock service
```

#### Test Scenario Routing: `.gherkio/tests/orders/checkout.yaml`
```yaml
scenario: Complete Order Checkout Flow
steps:
  # 1. Routes to Auth Service (https://auth.staging.company.internal/session)
  - request:
      method: POST
      service: auth
      url: /session
      body:
        token: $accounts.customer.apiKey
    save:
      authToken: body.accessToken

  # 2. Routes to Payments Service (https://payments.staging.company.internal/charges)
  - request:
      method: POST
      service: payments
      url: /charges
      headers:
        Authorization: "Bearer ${authToken}"
      body:
        amount: 4500
        currency: usd
    expect:
      status: 201

  # 3. Routes to Default baseUrl (https://api.staging.company.com/orders)
  # (No service tag is declared, so it defaults to the root baseUrl)
  - request:
      method: GET
      url: /orders
      headers:
        Authorization: "Bearer ${authToken}"
    expect:
      status: 200
```

---

### 3. URL Precedence & Resolution Formula

When Gherkio executes a test step, it parses the target endpoint according to the following strict order of precedence (highest to lowest):

```
[1. Absolute Step URL]    -->  - request: { url: "https://google.com" } (Overrides everything)
            ↓
[2. Named Service Route]  -->  - request: { service: "auth", url: "/login" } (Resolves via services map)
            ↓
[3. Default Base URL]     -->  - request: { url: "/login" } (Resolves via root baseUrl)
```

> 💡 **Best Practice:** Keep scenario steps generic by using relative paths (e.g., `/users`). Never hardcode staging or production domain names inside your YAML test scenarios. This ensures that the same scenario runs flawlessly across all pipelines simply by swapping the `--env` flag.

---

## 🛠️ Editor Autocomplete & JSON Schema

To make your development workflow fast and error-free, Gherkio includes a built-in JSON Schema generator (`gherkio schema`). This provides inline autocompletions, real-time syntax checking, and hover documentation inside editors like VSCode, Neovim, and others.

### 1. Generate the Autocomplete Schema

Generate the schema file in your project's workspace root:

```bash
gherkio schema > .gherkio-schema.json
```

> 💡 **Zero-Install Alternative:** If you do not have the binary installed on your local system path, you can run the generator dynamically using remote `go run`:
> ```bash
> go run github.com/muhfaris/gherkio@v0.1.0-alpha.5 schema > .gherkio-schema.json
> ```

---

### 2. VS Code Integration

1. Install the official [YAML Extension by Red Hat](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml) inside VS Code.
2. Open or create your workspace `settings.json` file under `.vscode/settings.json` and associate the schema with your Gherkio test folders:

```json
{
  "yaml.schemas": {
    "./.gherkio-schema.json": [".gherkio/tests/**/*.yaml", ".gherkio/tests/**/*.yml"]
  }
}
```

---

### 3. Inline Alternative (Any Editor)

If you don't use VS Code or prefer localized configuration, add this directive comment as the **very first line** of your Gherkio YAML test files:

```yaml
# yaml-language-server: $schema=../../.gherkio-schema.json
```

