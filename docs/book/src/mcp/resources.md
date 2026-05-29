# MCP Resources Reference

Gherkio's Model Context Protocol (MCP) server exposes a set of **7 read-only resources** containing the active DSL specifications, schema autocomplete files, canonical YAML test examples, available assertion matchers, variables, and project structures. 

These resources are exposed directly to the AI model on connection, allowing it to quickly grasp the syntax and structure of the Gherkio test system without relying on outdated external training data.

---

## 📂 Available Resources

### 1. `gherkio://dsl/spec`
- **Name**: Gherkio DSL Grammar Spec
- **MimeType**: `text/markdown`
- **Description**: Contains the exact structural keys, step properties, request configurations, and variable interpolation guidelines for writing Gherkio tests.

### 2. `gherkio://dsl/schema.json`
- **Name**: Gherkio Autocomplete JSON Schema
- **MimeType**: `application/json`
- **Description**: The automatically generated Draft-07 JSON Schema mapping internal models to DSL fields. Excellent for AI code autocomplete.

### 3. `gherkio://dsl/examples`
- **Name**: Gherkio DSL Canonical Examples
- **MimeType**: `text/yaml`
- **Description**: Working code snippets illustrating requests, saves, assertions, compositions, and collection validation in action.

### 4. `gherkio://dsl/matchers`
- **Name**: Available Matchers
- **MimeType**: `application/json`
- **Description**: A comprehensive JSON map of all 25+ assertion matchers, detailing their expected syntax, type requirements, and behavior.

### 5. `gherkio://dsl/variables`
- **Name**: Built-in Variables
- **MimeType**: `application/json`
- **Description**: Documented built-in generator variables (`$uuid`, `$ulid`, `$randomInt`, `$randomEmail`, `$randomPhone`) available natively in Gherkio test runs.

### 6. `gherkio://dsl/paths`
- **Name**: Canonical Paths
- **MimeType**: `application/json`
- **Description**: Explanation and examples of canonical dot-notation paths for assertions and saves (`body.*`, `headers.*`, `jwt.*`).

### 7. `gherkio://project/structure`
- **Name**: Project Directory Structure
- **MimeType**: `text/markdown`
- **Description**: A breakdown of the Gherkio workspace configuration folder (`.gherkio/`), explaining where configuration, environments, credentials, tests, reports, and schemas reside.

---

## ⚡ Reading a Resource

AI clients can read any resource by calling the `resources/read` protocol method:

```json
{
  "method": "resources/read",
  "params": {
    "uri": "gherkio://dsl/spec"
  }
}
```
 Gherkio returns the raw Markdown or JSON content directly, ensuring perfect, real-time sync with the codebase.
