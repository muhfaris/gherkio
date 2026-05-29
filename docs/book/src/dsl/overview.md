# DSL Reference

Gherkio features a fully declarative, human-readable YAML Domain Specific Language (DSL). It eliminates the need to write boilerplates in Go, Python, or JavaScript to test your REST, GraphQL, or HTTP APIs.

---

## 🎨 Design Philosophy

Gherkio's DSL is built on several key tenets:

- **Declarative, Not Imperative**: State *what* you want to request and *what* you expect to receive, rather than *how* to parse, compare, or loop.
- **AI-Native Compatibility**: Structured as dense, well-defined YAML, making it exceptionally easy for Large Language Models (LLMs) to write, audit, and modify tests with 100% precision.
- **Portability**: Test suites can be executed inside any CI/CD pipeline, locally on a laptop, or loaded as resources into an AI agent workspace via MCP.

---

## 🏗️ Basic Scenario Structure

A test scenario file contains three lifecycle blocks: `setup`, `steps`, and `teardown`.

```yaml
# Reusable tags to filter tests (e.g. gherkio run --tag smoke)
tags:
  - smoke
  - payments

# Step 1: Pre-conditions. If setup fails, main steps are skipped, but teardown still runs.
setup:
  - request:
      method: POST
      url: /setup-db
    expect:
      status: 200

# Step 2: Primary test steps executed sequentially.
steps:
  - request:
      method: GET
      url: /payments/methods
    expect:
      status: 200
      body.methods: array

# Step 3: Cleanup steps. Guaranteed to execute even if setup or steps fail.
teardown:
  - request:
      method: POST
      url: /teardown-db
```
