# Scenarios & Tags

Every Gherkio test file describes a single test scenario. The root of the YAML document contains the top-level keys for execution metadata, tagging, and execution phases.

---

## 🏗️ Top-Level Key Reference

All keys in the Gherkio DSL are case-sensitive and must be written in lowercase.

| Key | Type | Required | Description | Example |
| :--- | :--- | :--- | :--- | :--- |
| `scenario` | `string` | Yes | Human-readable name of the test scenario | `scenario: Create new user` |
| `description` | `string` | No | Detailed description of what this scenario tests. Shown in HTML report header | `description: Verify user can login with valid credentials` |
| `tags` | `array` | No | List of categories/labels for execution filtering | `tags: [smoke, active]` |
| `setup` | `array` | No | Pre-condition HTTP requests or composed files | (See Setup/Teardown chapter) |
| `steps` | `array` | Yes | Primary sequence of API request and assertion steps | (See Steps chapter) |
| `examples` | `array` | No | List of variable blocks for parametrized runs | (See examples section below) |
| `teardown` | `array` | No | Post-condition cleanup steps (guaranteed to run) | (See Setup/Teardown chapter) |

---

## 🏷️ Tagging & Filtering Conventions

Tags are strings that allow you to segment and filter test executions. They are extremely valuable for executing specific subsets of a test suite (e.g. running only light tests on commit, and full suites nightly).

```yaml
scenario: Authenticated checkout flow
description: Verifies a logged-in user can add items to cart and complete checkout with valid payment.
tags:
  - e2e
  - checkout
  - high-priority
```

### Filtering test runs from CLI:

```bash
# Run tests containing the 'smoke' tag
gherkio run tests/ --tag smoke

# Run tests containing BOTH 'core' AND 'user' tags (Logical AND)
gherkio run tests/ --tag core --tag user
```

---

## 📊 Parametrized / Data-Driven Testing (`examples`)

You can execute a single scenario multiple times with different datasets using the `examples` block. Under the hood, the test runner runs the scenario for each set of variables defined in `examples`.

```yaml
scenario: Parametrized login validation
description: Verifies login fails with various bad inputs
examples:
  - username: "alice"
    password: "bad-password"
  - username: bob
    password: ""
steps:
  - name: Attempt login
    request:
      method: POST
      url: /api/login
      body:
        username: "{{username}}"
        password: "{{password}}"
    expect:
      status: 400
```
