# Gherkio Documentation

> **The Declarative Integration Testing Platform** — Build bulletproof API test suites in pure YAML. No boilerplate, no runtime compilation, zero custom glue-code required.

Gherkio is a declarative integration testing platform compiled into a single static Go binary. Teams write API integration tests by describing HTTP request sequences, assertions, and variable extractions in pure YAML. It features a native MCP server for AI assistant integration, outbound network sandboxing for security, and structured reporting for CI/CD pipelines.

---

```yaml
scenario: Authenticate & Retrieve Profile
steps:
  - request:
      method: POST
      url: /auth/login
      body:
        username: $accounts.admin.username
        password: $accounts.admin.password
    expect:
      status: 200
      body.token: exists
    save:
      jwt: body.token

  - request:
      method: GET
      url: /auth/me
      headers:
        Authorization: Bearer $jwt
    expect:
      status: 200
      body.role: admin
```

---

## 🧭 Tiered Learning Paths

Whether you are writing your first test scenario, building reusable enterprise test suites, or integrating AI agent automation into CI/CD, choose your path below:

### 🟢 1. Beginner Path (Zero-to-Hero)
*Ideal for developers & QA engineers getting started with Gherkio.*

| Guide | Description |
| :--- | :--- |
| 🚀 **[Installation](getting-started/installation.md)** | Download static binaries for Linux, macOS, or Windows. |
| ⚡ **[2-Minute Quickstart](getting-started/quickstart.md)** | Scaffold your workspace and run your first scenario in under two minutes. |
| 📁 **[Folder & Project Setup](getting-started/project-setup.md)** | Learn `.gherkio/` directory structure, environments, and configuration. |
| 📝 **[Tutorial: Build Your First Test](getting-started/first-test-tutorial.md)** | Step-by-step guide to modeling HTTP requests, status assertions, and saved variables. |
| 🔍 **[What Happens in a Test: Lifecycle](getting-started/execution-lifecycle.md)** | Detailed breakdown of the internal 9-phase execution pipeline, variable resolution, and step execution. |
| 🎨 **[Interactive Playground](getting-started/playground.md)** | Convert legacy cURL commands to Gherkio YAML DSL instantly. |

---

### 🟡 2. Advanced Practitioner Path
*For engineers building modular, resilient, and multi-environment test suites.*

| Guide | Description |
| :--- | :--- |
| 🧩 **[Scenario Composition](dsl/composition.md)** | Modularize tests using `use: path/to/scenario.yaml` for shared auth & setup flows. |
| 🎯 **[Dynamic Assertions & Matchers](dsl/matchers.md)** | Master value matchers (`contains`, `greaterThan`, `oneOf`, `in`, `matchesRegex`), JWT validation, and timing budgets. |
| 📋 **[JSON Schema Validation](dsl/schemas.md)** | Enforce structural schema rules against response payloads using `.gherkio/schemas/`. |
| 🔄 **[Async Retries & Polling](dsl/retry.md)** | Handle asynchronous background jobs with exponential backoff and retry rules. |
| ⚡ **[Redis Cache State Checks](dsl/redis.md)** | Assert key-value states in Redis directly inside scenario test steps. |
| 🔐 **[Credentials & Environments](reference/environments.md)** | Manage multi-account credentials and staging/production target overrides. |
| 🛠️ **[CLI Tooling & Validation](cli/overview.md)** | Run static analysis (`gherkio validate`), generate JSON Schema (`gherkio schema`), and convert cURL (`gherkio convert`). |

---

### 🔴 3. Expert & Platform Engineer Path
*For architects, DevOps teams, and AI-assisted workflow integration.*

| Guide | Description |
| :--- | :--- |
| 🤖 **[MCP Server & AI Assistant Workflows](mcp/overview.md)** | Connect Gherkio directly to Cursor, Claude Desktop, or Windsurf for AI-generated testing & self-healing runs. |
| 🏗️ **[Engine Architecture & Security Sandboxing](contributing/architecture.md)** | Deep-dive into static Go engine internals, thread safety, and outbound SSRF protection. |
| 📊 **[Enterprise CI/CD & Reporting](reference/reports.md)** | Generate JUnit XML, HTML, and JSON reports for GitHub Actions, GitLab CI, and Jenkins. |
| 🛠️ **[Extending Gherkio](contributing/adding-matchers.md)** | Add custom matchers and extend the static evaluation engine in Go. |

---

## ⚡ Key Value Pillars

* **Built for Speed**: Written in pure Go, Gherkio compiles to a single static binary with zero external dependencies.
* **Declarative YAML Syntax**: Human-readable scenarios double as executable API documentation.
* **Native Tooling**: Out-of-the-box support for JWT checks, Redis assertions, exponential backoff retries, and dynamic generators (`$randomEmail`, `$uuid`).
* **Agentic AI Ready**: Built-in Model Context Protocol (MCP) server enables AI developer tools to run, validate, and write tests autonomously.
