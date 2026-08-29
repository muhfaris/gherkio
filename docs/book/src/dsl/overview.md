# DSL Reference

> 🟡 **Advanced Guide** — Gherkio features a fully declarative, human-readable YAML Domain Specific Language (DSL). It eliminates the need to write boilerplate code to test your REST, HTTP, or Redis-backed APIs.

---

## 🎨 Design Philosophy

Gherkio's DSL is built on four core tenets:

- **Declarative, Not Imperative**: State *what* you want to request and *what* you expect to receive, rather than *how* to parse, compare, or loop.
- **AI-Native Compatibility**: Structured as dense, well-defined YAML, making it exceptionally easy for Large Language Models (LLMs) to write, audit, and modify tests with 100% precision.
- **Zero Glue-Code**: Native support for JWT assertions, Redis state verification, JSON Schema enforcement, and dynamic backoff retries.
- **Portability**: Test suites execute inside any CI/CD pipeline, locally on a developer workstation, or loaded into AI agent workspaces via MCP.

---

## 🏗️ Scenario Lifecycle & Structure

A test scenario file is organized into three lifecycle sections (`setup`, `steps`, `teardown`):

```yaml
# Categorization tags for filtering (e.g. gherkio run --tag smoke)
tags:
  - smoke
  - payments

# Pre-conditions: Executed first. If setup fails, steps are skipped.
setup:
  - request:
      method: POST
      url: /setup-db
    expect:
      status: 200

# Core Test Steps: Executed sequentially. Variables saved here propagate downstream.
steps:
  - request:
      method: GET
      url: /payments/methods
    expect:
      status: 200
      body.methods: array
    save:
      paymentMethodId: body.methods[0].id

  - request:
      method: POST
      url: /payments/charge
      body:
        methodId: $paymentMethodId
        amount: 1500
    expect:
      status: 201
      body.status: paid

# Post-conditions: Guaranteed to execute even if setup or steps fail.
teardown:
  - request:
      method: POST
      url: /teardown-db
```

---

## 📚 Advanced DSL Capabilities

Explore specific features in the Advanced Practitioner path:

| Topic | Description |
| :--- | :--- |
| 🧩 **[Scenarios & Lifecycle](scenarios.md)** | Master scenario structure, tags, setup/teardown mechanics, and exit rules. |
| 🔄 **[Scenario Composition](composition.md)** | Reuse test files via `use: path/to/scenario.yaml` for modular auth and data setup. |
| 🎯 **[Assertions & Matchers](assertions.md)** | Learn value matchers (`contains`, `greaterThan`, `oneOf`, `in`, `matchesRegex`), JWT validation, and timing budgets. |
| 📋 **[Schema Validation](schemas.md)** | Enforce structural schema rules against response payloads using `.gherkio/schemas/`. |
| ⚡ **[Redis Cache State Checks](redis.md)** | Assert key-value states in Redis directly inside scenario test steps. |
| 🎭 **[Service Mocking & Virtualization](mocking.md)** | Intercept outbound HTTP calls with zero-dependency virtual responses and parameter reflection. |
| 🔁 **[Retries & Polling](retry.md)** | Handle asynchronous background jobs with exponential backoff and retry rules. |
| 🔐 **[Variables & Credentials](variables.md)** | Dynamic variable generators (`$randomEmail`, `$uuid`) and environment credential injection. |
