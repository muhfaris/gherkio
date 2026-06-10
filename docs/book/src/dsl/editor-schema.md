# Editor Autocomplete Schema

Gherkio ships a machine-readable JSON Schema that editors like VS Code and Neovim can use to provide inline autocomplete, real-time validation, and hover documentation when writing Gherkio YAML test files.

---

## 🎯 What Is This?

When you open a Gherkio test file (e.g. `.gherkio/tests/login.yaml`) in a compatible editor, the JSON Schema tells the editor:

- Which top-level keys are valid (`scenario`, `setup`, `steps`, `teardown`, etc.)
- What types each field expects (`string`, `array`, `object`, etc.)
- Which keys are required in each block
- What values are allowed for enum-like fields (HTTP methods, matcher names, etc.)
- Descriptive comments shown as hover tooltips

This is powered by the `gherkio schema` CLI command, which introspects the Go model structs and emits a standard **JSON Schema Draft-07** document.

---

## 🏗️ Schema Types

The `gherkio schema` command generates schemas for all first-class Gherkio file types. You can generate all of them at once, or target a specific type:

| Type | File Pattern | Description |
| :--- | :--- | :--- |
| `test` | `.gherkio/tests/**/*.yaml` | Full scenario test files (setup + steps + teardown) |
| `config` | `.gherkio/config.yaml` | Workspace configuration |
| `environment` | `.gherkio/environments/*.yaml` | Environment definitions (baseUrl, services) |
| `credentials` | `.gherkio/credentials/*.yaml` | Account credentials per environment |
| `schema-definition` | `.gherkio/schemas/**/*.yaml` | JSON Schema Draft-07 contract files used in `expect.schema` assertions |

---

## 📦 Generating the Schema

### Generate all schemas at once

```bash
gherkio schema > .gherkio-schema.json
```

This produces a single JSON file keyed by type:

```json
{
  "test": { "$schema": "http://json-schema.org/draft-07/schema#", ... },
  "config": { ... },
  "environment": { ... },
  "credentials": { ... },
  "schema-definition": { ... }
}
```

### Generate a specific schema type

```bash
gherkio schema --type test > gherkio-test-schema.json
gherkio schema --type environment > gherkio-env-schema.json
```

### List available schema types

```bash
gherkio schema --list
```

---

## 🖥️ VS Code Integration

### 1. Install the YAML extension

Install the official [YAML Language Server (Red Hat)](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml) extension in VS Code.

### 2. Configure schema associations

Open or create `.vscode/settings.json` in your workspace root:

```json
{
  "yaml.schemas": {
    "./.gherkio-schema.json": [".gherkio/tests/**/*.yaml", ".gherkio/tests/**/*.yml"]
  }
}
```

### 3. Per-file schema directive (alternative)

Add this comment as the **very first line** of any Gherkio YAML file to use a locally generated schema:

```yaml
# yaml-language-server: $schema=../../.gherkio-schema.json
scenario: Login flow
steps:
  ...
```

---

## ⚡ Neovim Integration

### gherkio.nvim (recommended)

The official [gherkio.nvim](https://github.com/muhfaris/gherkio/tree/main/neovim/gherkio.nvim) Neovim plugin automatically downloads and caches the schema from the MCP server (`gherkio://dsl/schema.json`) on first use. No manual configuration required.

For manual schema configuration in Neovim, point your YAML LSP or `vim.schema` settings to the generated schema file:

```lua
-- Example: vim.schema (Neovim 0.10+)
vim.schema('.gherkio-schema.json', {
  filepattern = '.gherkio/tests/**/*.yaml',
})
```

Or configure via `yaml.schemas` in `vim.lsp.config` if using `yaml-language-server` manually.

---

## 📁 Where to Place the Schema File

Place the generated `.gherkio-schema.json` in your **workspace root** (same level as `.gherkio/`):

```
.
├── .gherkio/
│   ├── tests/
│   │   └── login.yaml
│   ├── environments/
│   │   └── staging.yaml
│   └── schemas/
│       └── auth/token.yaml
└── .gherkio-schema.json    ← generated here
```

Commit `.gherkio-schema.json` to version control so every teammate gets autocomplete without running any commands.

---

## 🔄 Keeping the Schema Up to Date

The schema is derived from the Go model structs in the `gherkio` binary. After upgrading `gherkio` to a new version, re-generate the schema:

```bash
gherkio schema > .gherkio-schema.json
```

The MCP server always exposes the schema for the **currently running** binary version via the `gherkio://dsl/schema.json` resource URI.

---

## 📋 Schema Structure Overview

Below is a simplified view of the top-level structure the test schema validates:

```yaml
# Top-level keys (all optional except 'steps')
tags:           []string          # Filter tags for selective runs
setup:          []Step            # Pre-condition block; halts on failure
steps:          []Step            # Primary test block (required)
teardown:       []Step            # Cleanup block; always executes

# Each Step is either a 'use' (composition) or a 'request':
Step:
  name:     string               # Human-readable label
  use:      string               # Path to another scenario file (mutually exclusive with request)
  request:  Request              # HTTP request config (mutually exclusive with use)
  expect:   Expect               # Response assertions
  save:     map[string]string    # Extract variables from response
  timing:   TimingConfig        # Latency budget

# Request fields
Request:
  service:  string               # Named service override
  method:   string               # GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS
  url:      string               # URL path (appends to baseUrl)
  query:    map[string]string    # Query parameters
  headers:  map[string]string    # Custom headers
  body:     any                  # Request payload
  transform: map[string]ProjectionConfig  # Declarative array projections

# Expect fields
Expect:
  status:          integer       # Expected HTTP status code
  body.<path>:     MatcherValue  # Body dot-path assertions
  headers.<name>:  MatcherValue  # Header assertions
  jwt.<claim>:     MatcherValue  # JWT claim assertions
  schema:          string        # Schema contract name (from .gherkio/schemas/)
  timing.duration: MatcherValue  # Latency assertion
```

For the full schema with all enum values, type constraints, and descriptions, run:

```bash
gherkio schema --type test | python3 -m json.tool | less
```
