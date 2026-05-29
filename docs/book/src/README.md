# Gherkio

> **The Declarative Integration Testing Platform** — Build bulletproof API test suites in pure YAML. No boilerplate, no runtime compilation, zero custom glue-code required.

---

API testing has traditionally been trapped between two bad options: click-and-point GUI client workbooks that cannot scale under continuous integration, and heavy codebases written in JavaScript, Python, or Go that suffer from dependency bloat, fragile custom assert frameworks, and poor readability.

**Gherkio** offers a professional third path: a single, self-contained binary that executes highly readable, multi-step declarative testing scenarios defined in simple YAML. 

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

## ⚡ Key Value Pillars

### 🟢 1. Built for Speed and Ephemeral CI
Written in pure Go, Gherkio compiles to a single static binary with zero external runtime dependencies. Boot up tests instantly inside empty Docker instances, local terminals, or standard GitHub runner nodes.

### 🟢 2. Declarative Syntax as Single-Source-of-Truth
Because scenarios are written in pure YAML, your API test files double as live, executable documentation. Developers, QA specialists, and product managers can read, write, and audit scenario steps without needing to understand complex software architecture.

### 🟢 3. Industrial-Grade Native Ecosystem
Forget writing custom wrapper functions. Gherkio features native support for **JWT assertions**, **automatic credential masking**, **exponential-backoff retries**, **multi-role account loops**, **full JSON schema matching**, and **timing budgets** right out of the box.

### 🟢 4. Agentic AI Ready (Model Context Protocol)
Gherkio includes a native Model Context Protocol (MCP) server that exposes your testing sandbox directly to AI developer assistants (like Cursor, Windsurf, or custom Claude Desktop environments). Let AI agents draft, execute, validate, and debug integration tests automatically inside your project workspace.

---

## 🗺️ Developer Onboarding Roadmap

To learn Gherkio systematically, follow our structured, progressive onboarding chapters:

| Step | Section | Description |
| :--- | :--- | :--- |
| **1** | 🚀 **[Getting Started: Installation](getting-started/installation.md)** | Install pre-compiled binaries or run dynamically with zero-install Go run directives. |
| **2** | ⏱️ **[Getting Started: 2-Minute Quickstart](getting-started/quickstart.md)** | Scaffold your first testing sandbox and see Gherkio's console reporting in action. |
| **3** | 📁 **[Getting Started: Project & Folder Setup](getting-started/project-setup.md)** | Master config files, environments overrides, and microservice hosts mapping. |
| **4** | 📝 **[Tutorial: Build Your First Test](getting-started/first-test-tutorial.md)** | Step-by-step walkthrough detailing request modeling, data assertions, and variable saving. |
| **5** | 🎨 **[Interactive Browser Playground](getting-started/playground.md)** | Visualize test steps dynamically and convert legacy cURL statements instantly. |
