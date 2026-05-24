# RFC-6: MCP Server Support — Internal Refactoring for LLM Integration

> **Status:** Implemented
> **Author:** Faris
> **Date:** May 22, 2026
> **Updated:** May 23, 2026 — aligned with current codebase (RFC-14, RFC-15, engine.go)

---

## 1. Summary

Refactor Gherkio's internal architecture to expose reusable APIs for test, environment, and schema management. This enables an MCP (Model Context Protocol) server layer that allows LLMs to discover, create, read, update, validate, and execute tests declaratively.

---

## 2. Motivation

Currently Gherkio is a pure CLI tool. All test management logic (file discovery, path resolution, project root detection) is embedded in `cmd/run.go` and `cmd/init.go`. This makes it impossible for external integrations — like an LLM-powered MCP server — to interact with Gherkio programmatically.

Primary use case: An AI assistant (e.g. Claude) helps users create and manage integration tests by:

- Listing existing tests, environments, and schemas in a project
- Reading test YAML to understand existing scenarios
- Validating test structure before saving
- Creating new tests with proper YAML structure
- Updating existing tests (adding steps, modifying assertions)
- Running tests and returning structured results

To support this, Gherkio needs a **programmable core** — the CLI becomes just one consumer of that core, and the MCP server becomes another.

---

## 3. Design

### 3.1 Architecture Overview

```
gherkio/
├── cmd/                     │  CLI layer (unchanged, but delegates to new packages)
│   ├── root.go
│   ├── init.go              │  ← delegates to scaffolding helpers
│   ├── run.go               │  ← delegates to runner + new packages
│   ├── convert.go           │  cURL ↔ DSL conversion (RFC-14)
│   └── schema.go            │  JSON Schema generation (RFC-15)
│
├── internal/
│   ├── model/               │  Data types (unchanged)
│   ├── runner/              │  Execution engine
│   │   ├── runner.go         │  Orchestrator + RunSingleStep()
│   │   ├── engine.go         │  Source of truth: paths, matchers, backoff (RFC-15)
│   │   ├── executor.go       │  HTTP client, assertions, retries
│   │   ├── interpolator.go   │  Variable interpolation
│   │   ├── matchers.go       │  All matchers + GetAvailableMatchers()
│   │   ├── steplocator.go    │  Step boundary detection for --line (RFC-14)
│   │   ├── printer.go        │  Console output + PrintStepResult()
│   │   ├── credentials.go    │  LoadCredentials() — already exported
│   │   ├── config.go         │  LoadConfig() — already exported
│   │   ├── schema.go         │  LoadSchema() — already exported
│   │   └── validator.go      │  Schema validation engine
│   ├── schema/               │  JSON Schema generator (RFC-15) ✅ ALREADY EXISTS
│   │   └── generator.go      │  Generates multi-type schemas for all YAML types
│   ├── converter/            │  cURL ↔ DSL (RFC-14) ✅ ALREADY EXISTS
│   │   ├── parser.go         │  cURL tokenizer + parser
│   │   ├── dsl.go            │  YAML output
│   │   └── curl.go           │  Reverse: step → cURL
│   ├── report/               │  HTML/JSON reporting
│   ├── core/                 │  NEW: Domain logic for project artifact management
│   │   ├── project/          │  Project discovery & metadata
│   │   │   ├── finder.go     │  Find project root, config, paths
│   │   │   └── meta.go       │  Project metadata
│   │   ├── teststore/        │  Test file management
│   │   │   ├── lister.go     │  List, glob discovery
│   │   │   ├── reader.go     │  Read & parse test files (wraps exported LoadTestFile)
│   │   │   ├── writer.go     │  Create & update test files
│   │   │   ├── remover.go    │  Delete test files
│   │   │   └── validator.go  │  Validate test structure & references
│   │   ├── envstore/         │  Environment management
│   │   │   ├── lister.go
│   │   │   ├── reader.go
│   │   │   ├── writer.go
│   │   │   └── remover.go
│   │   └── schemastore/      │  Schema management
│   │       ├── lister.go
│   │       ├── reader.go
│   │       ├── writer.go
│   │       └── remover.go
│   └── mcp/                  │  NEW: MCP server
│       └── server.go
```

### 3.2 Package: `internal/core/project/`

**Purpose:** Extract project discovery logic currently in `cmd/run.go` and `cmd/init.go` into a reusable package.

```go
// Finder
func FindRoot(cwd string) (string, error)
// Walks up from cwd, finds directory containing .gherkio/
// Returns absolute path to project root.

// Meta
type ProjectMeta struct {
    Name       string `yaml:"name"`
    Version    string `yaml:"version"`
    RootDir    string // absolute path
    TestsDir   string // .gherkio/tests/
    EnvsDir    string // .gherkio/environments/
    SchemasDir string // .gherkio/schemas/
    ReportsDir string // .gherkio/reports/
}

func GetMeta(projectDir string) (*ProjectMeta, error)
// Reads .gherkio/config.yaml and returns resolved project metadata.

// Config
type Config struct {
    Project struct {
        Name    string `yaml:"name"`
        Version string `yaml:"version"`
    } `yaml:"project"`
    Environments struct {
        Default string `yaml:"default"`
        Path    string `yaml:"path"`
    } `yaml:"environments"`
    Tests struct {
        Path string `yaml:"path"`
    } `yaml:"tests"`
    Reports struct {
        Path    string `yaml:"path"`
        Format  string `yaml:"format"`
        Archive bool   `yaml:"archive"`
    } `yaml:"reports"`
    Schemas struct {
        Path string `yaml:"path"`
    } `yaml:"schemas"`
    Security struct {
        Mask struct {
            Enabled bool     `yaml:"enabled"`
            Fields  []string `yaml:"fields"`
        } `yaml:"mask"`
    } `yaml:"security"`
}

func LoadConfig(projectDir string) (*Config, error)
// Reads and parses .gherkio/config.yaml.
```

### 3.3 Package: `internal/core/teststore/`

**Purpose:** Provide CRUD + validation for test files.

```go
// Types
type TestInfo struct {
    RelativePath string // e.g. "auth/login.yaml"
    AbsolutePath string
    Scenario     string // from the YAML scenario field
    StepCount    int
}

type ValidationResult struct {
    Valid  bool
    Errors []ValidationError
}

type ValidationError struct {
    Field   string // e.g. "steps[0].request.url"
    Message string // e.g. "URL is required"
    Code    string // e.g. "missing_field"
}

// Lister
func ListTests(projectDir string) ([]TestInfo, error)
// Discovers all .yaml files under .gherkio/tests/.
// Parses each file to extract scenario name and step count.
// Returns sorted by path.

// Reader
func ReadTest(absPath string) (*model.TestFile, error)
// Loads and parses a single test YAML file.
// Wraps loadTestFile() from runner (currently internal to runner package).

// Writer
func CreateTest(projectDir, relativePath string, test *model.TestFile) error
// Validates test file first.
// Constructs full path: projectDir/.gherkio/tests/ + relativePath.
// Ensures parent directories exist.
// Returns error if file already exists.

func UpdateTest(absPath string, test *model.TestFile) error
// Validates test file first.
// Overwrites existing test file.
// Creates backup with .bak suffix before overwrite.

// Remover
func DeleteTest(absPath string) error
// Removes test file. Returns error if file doesn't exist.

// Validator
func Validate(test *model.TestFile, projectDir string) (*ValidationResult, error)
// Validates:
//   1. Scenario name is non-empty
//   2. At least one step exists
//   3. Each step has either request or use
//   4. request.method is valid HTTP method
//   5. request.url is non-empty
//   6. If expect.schema exists, schema file exists in project
//   7. If use: exists, referenced file can be resolved
// Returns list of errors; empty errors = valid.

// Scaffolding
func ScaffoldExample(projectDir string) error
// Extracted from cmd/init.go — creates example tests.
```

### 3.4 Package: `internal/core/envstore/`

**Purpose:** Provide CRUD for environment files.

```go
type EnvInfo struct {
    Name    string // "local", "staging", etc.
    BaseURL string
    ServicesCount int
}

func List(projectDir string) ([]EnvInfo, error)
// Lists all .yaml files in .gherkio/environments/.
// Returns sorted by name.

func Read(projectDir, name string) (*model.Environment, error)
// Loads .gherkio/environments/{name}.yaml.

func Create(projectDir, name string, env *model.Environment) error
// Creates new environment file.
// Returns error if already exists.

func Update(projectDir, name string, env *model.Environment) error
// Overwrites existing environment file.

func Delete(projectDir, name string) error
// Removes environment file.
// Returns error if it's the default environment (in use).
```

### 3.5 Package: `internal/core/schemastore/`

**Purpose:** Provide CRUD for schema files.

```go
type SchemaInfo struct {
    Name     string // "example/login-response", "restful-api/object-schema"
    FilePath string
}

func List(projectDir string) ([]SchemaInfo, error)
// Lists all .yaml files in .gherkio/schemas/.
// Returns sorted by name.

func Read(projectDir, name string) (*model.Schema, error)
// Loads schema by name.
// Name maps to path: "example/login-response" → .gherkio/schemas/example/login-response.yaml

func Create(projectDir, name string, schema *model.Schema) error
// Creates new schema file.
// Name includes subdirectory, e.g. "example/login-response".

func Update(projectDir, name string, schema *model.Schema) error
// Overwrites existing schema file.

func Delete(projectDir, name string) error
// Removes schema file.
// Returns error if any test references this schema.
```

### 3.6 Package: `internal/mcp/`

**Purpose:** Expose Gherkio capabilities via the Model Context Protocol.

The MCP server will expose:

| Tool | Description |
|------|-------------|
| `list_tests` | List all test files with scenario name and step count |
| `read_test` | Read and return a test file's YAML content |
| `validate_test` | Validate test YAML structure and references |
| `create_test` | Create a new test file from YAML content |
| `update_test` | Update an existing test file |
| `delete_test` | Delete a test file |
| `run_test` | Execute a test and return structured results |
| `list_environments` | List available environments |
| `list_schemas` | List available schemas |
| `get_project_info` | Get project metadata (name, version, paths) |

Resources (read-only data):
| URI | Description |
|-----|-------------|
| `gherkio://project` | Project metadata |
| `gherkio://tests/{path}` | Test file content |
| `gherkio://environments/{name}` | Environment config |
| `gherkio://schemas/{name}` | Schema definition |
| `gherkio://results/latest` | Most recent run result |

Prompts (reusable templates):
| Prompt | Description |
|--------|-------------|
| `new_test` | Scaffold a new test scenario for a given endpoint |
| `new_environment` | Scaffold a new environment config |

---

## 4. Refactoring Plan

### Phase 1: Extract `core/project/` and `core/teststore/`

**Goal:** Move project discovery and test file management out of `cmd/`.

Steps:
1. Create `internal/core/project/finder.go` — copy `findProjectRoot()` from `cmd/run.go`
2. Create `internal/core/project/meta.go` — new `GetMeta()`, `LoadConfig()` functions
3. Create `internal/core/teststore/lister.go` — copy `discoverTestFiles()` + add parsing
4. Create `internal/core/teststore/reader.go` — expose `loadTestFile()` (move from `runner/`)
5. Create `internal/core/teststore/validator.go` — new validation logic
6. Create `internal/core/teststore/writer.go` — new create/update/delete logic
7. Create `internal/core/teststore/remover.go` — delete logic
8. Update `cmd/run.go` to delegate to these packages
9. Update `cmd/init.go` to delegate to `core/teststore.ScaffoldExample()`

### Phase 2: Extract `core/envstore/` and `core/schemastore/`

**Goal:** Environment and schema management APIs.

Steps:
1. Create `internal/core/envstore/` — CRUD for environment files
2. Create `internal/core/schemastore/` — CRUD for schema files
3. Add cross-reference validation (schema exists when referenced in test)

### Phase 3: Add Dynamic Variable Injection (Already Partially Done)

**Goal:** Allow MCP to pass variables when running tests.

Current: `runner.Run(cfg RunConfig)` — `CredentialVars` field already exists on `RunConfig` (added in RFC-12). Variables are injected into the step variable context before execution.

For MCP, this field maps directly: when a user passes variables via the `run_test` MCP tool, they're passed as `CredentialVars` to `RunConfig`. No new struct fields needed.

```go
type RunConfig struct {
    TestPath       string
    EnvName        string
    ProjectDir     string
    Verbose        bool
    MaskFields     []string
    CredentialVars map[string]interface{} // Already exists — used for credential + dynamic variable injection
}
```

**Note:** The `CredentialVars` field merges with `save:` variables during execution (step saves override credential vars). This is the correct behavior for MCP — explicit variables from the user should take precedence over credentials, and step saves should take precedence over both.

### Phase 4: Build MCP Server

**Goal:** Server that LLMs can connect to via stdio.

Steps:
1. Add MCP Go SDK dependency
2. Create `internal/mcp/server.go`
3. Register tools and resources
4. Wire everything together

---

## 5. Edge Cases

### 5.1 Project Confirmation UX

When an LLM operates outside a Gherkio project (no `.gherkio/` directory found), the MCP server should:

```
→ LLM: "List existing tests"
← MCP: "No .gherkio project found in current directory.
    Would you like me to create one at /path/to/dir?"
→ LLM: "Yes"
← MCP: Creates project scaffold, confirms, then responds.
```

This maps to a dedicated MCP tool: `init_project` that requires explicit user confirmation (not automatic).

### 5.2 File Conflicts

If `CreateTest` is called on an existing path, the server returns an error with the existing file's content so the LLM can decide to update or use a different path.

### 5.3 Schema Reference Validation

When validating a test that references `expect.schema: example/login-response`, the validator checks that the file `.gherkio/schemas/example/login-response.yaml` exists. If not, the validation error includes available schemas as suggestions.

### 5.4 Invalid YAML on Create

If the LLM sends malformed YAML via `create_test`, the server returns a parse error with details about the issue, allowing the LLM to correct and retry.

### 5.5 Environment-Service Cross-Reference

When creating/updating a test that uses `request.service: auth`, the validator checks that the referenced environment has a configured service with that name. If the test is run without an environment (default), it only warns, since the user might configure it later.

---

## 6. MCP Implementation Notes

### 6.1 Transport

**stdio only.** MCP will communicate over stdin/stdout. This is the simplest and most secure for local LLM integrations (Claude Desktop, VS Code extensions, etc.).

No HTTP server, no authentication, no TLS. This is a developer tool for local use.

### 6.2 Tool Registration Sketch

```go
// internal/mcp/server.go
package mcp

import (
    "github.com/muhfaris/gherkio/internal/core/project"
    "github.com/muhfaris/gherkio/internal/core/teststore"
    "github.com/muhfaris/gherkio/internal/core/envstore"
    "github.com/muhfaris/gherkio/internal/core/schemastore"
    "github.com/muhfaris/gherkio/internal/runner"
)

func NewServer(cwd string) (*Server, error) {
    projectDir, err := project.FindRoot(cwd)
    if err != nil {
        // Server still works, but requires explicit init
    }

    s := &Server{
        projectDir: projectDir,
        tools:      make(map[string]ToolHandler),
    }

    // Register tools
    s.tools["list_tests"] = s.handleListTests
    s.tools["read_test"] = s.handleReadTest
    s.tools["create_test"] = s.handleCreateTest
    s.tools["update_test"] = s.handleUpdateTest
    s.tools["delete_test"] = s.handleDeleteTest
    s.tools["validate_test"] = s.handleValidateTest
    s.tools["run_test"] = s.handleRunTest
    s.tools["list_environments"] = s.handleListEnvironments
    s.tools["list_schemas"] = s.handleListSchemas
    s.tools["get_project_info"] = s.handleGetProjectInfo
    s.tools["init_project"] = s.handleInitProject // requires confirmation

    return s, nil
}
```

### 6.3 MCP Dependency Module

```go
// go.mod addition
require (
    github.com/mark3labs/mcp-go v1.0.0
)
```

---

## 7. User Interaction Model

The MCP server is designed for conversational use through an LLM client (Claude Desktop, VS Code extension, etc.). The LLM is an **active agent** — it discovers context, proposes plans, and executes tools autonomously, keeping the user informed at each step.

### 7.1 Core Principles

| Principle | Behavior |
|-----------|----------|
| **Proactive discovery** | LLM calls `list_*` tools before responding, even if the user didn't explicitly ask |
| **Validate before write** | Always `validate_test` before `create_test` or `update_test` |
| **Run after create** | After creating a test, run it to confirm it works |
| **Error with context** | Show actual response body on failure, not just "assertion failed" |
| **Ask with options** | When blocked, offer choices rather than open-ended questions |
| **No silent creation** | Always show the YAML diff/plan before writing, get user confirmation |

### 7.2 Interaction Flows

#### Flow A: Onboarding — "Create tests for my API"

```
User:
  "I have a new auth API. Help me create a login test."

LLM (behind the scenes):
  → list_environments()       logs: sees "local" with dummyjson baseUrl
  → list_tests()               logs: sees example/ tests
  → list_schemas()             logs: sees example/login-response schema
  → read_test(example/auth/login.yaml)  logs: understands existing pattern

LLM responds:
  "I see you have a 'local' environment pointing to dummyjson.com.
   The login endpoint is POST /auth/login with username/password.

   Let me create a login test. What credentials should I use?"
```

```
User:
  "Use emilys/emilyspass, and save the token."

LLM (behind the scenes):
  → validate_test(tests/auth/login.yaml)   → passes
  → create_test(tests/auth/login.yaml, ...) → created
  → run_test(tests/auth/login.yaml)         → all pass

LLM responds:
  "✅ Created tests/auth/login.yaml — runs successfully.

   • POST /auth/login → 200
   • body.accessToken ✓ exists
   • body.email ✓ equals 'emilys'
   • schema ✓ matches example/login-response

   Token saved as $accessToken for reuse in subsequent tests."
```

#### Flow B: Exploration — "What tests do we have?"

```
User:
  "What tests exist in this project?"

LLM (behind the scenes):
  → list_tests() → 4 tests returned

LLM responds:
  "You have 4 test scenarios:

   ├─ example/auth/login.yaml       (2 steps, login + schema check)
   ├─ example/auth/me.yaml          (2 steps, uses login + profile check)
   ├─ example/auth/refresh.yaml     (2 steps, uses me + refresh flow)
   └─ restful-api/e2e-workflow.yaml (4 steps, full CRUD flow)

   Want me to show details on any of these?"
```

#### Flow C: Debugging — "The refresh test is failing"

```
User:
  "My refresh token test is failing. Can you check?"

LLM (behind the scenes):
  → run_test(tests/example/auth/refresh.yaml)
  → Sees: status 400, assertion failures

LLM responds:
  "I ran it and it's failing:

   ✗ POST /auth/refresh → 400 (expected 200)
     └─ Response body:
         { "message": "Refresh token expired" }

   The issue: $refreshToken from the login step likely expired.

   Options:
   1. Increase expiresInMins in the login step
   2. Run login + refresh back-to-back (already done via 'use')
   3. Generate a fresh token before refresh

   Want me to try option 1 — bump expiresInMins to 60?"
```

#### Flow D: Outside a project — "Set up Gherkio for this codebase"

```
User:
  "Help me set up Gherkio for this project."

LLM (behind the scenes):
  → get_project_info() → no .gherkio/ found
  → Cannot list_tests, list_environments, etc.

LLM responds:
  "This directory doesn't have a Gherkio project yet.

   Would you like me to create one? This will:
   • Create .gherkio/ with config.yaml
   • Add a default local environment
   • Create example tests to get started

   Shall I proceed?"
```

### 7.3 User Confirmation Rules

| Action | Requires Confirmation? |
|--------|------------------------|
| List/read/explore | No — fire and forget |
| Run a test | No — read-only execution |
| Create a test | Yes — show YAML first |
| Update a test | Yes — show diff first |
| Delete a test | Yes — warn about consequences |
| Init a new project | Yes — confirm with summary of what will be created |

---

## 8. LLM DSL Assistance

LLMs hallucinate field names, path syntax, and matchers. Gherkio provides structured reference material to guide the LLM toward correct YAML without human correction.

### 8.1 Four-Layer Approach

| Layer | Mechanism | Purpose |
|-------|-----------|---------|
| **DSL Spec** | MCP Resource `gherkio://dsl/spec` | Complete field reference (required/optional, types, description) |
| **JSON Schema** | MCP Resource `gherkio://dsl/schema.json` | Auto-generated from Go structs. LLMs natively understand JSON Schema |
| **Examples** | MCP Resource `gherkio://dsl/examples` | 3 canonical examples for pattern-matching |
| **Prompt Templates** | MCP Prompts (`new_test`, `new_environment`) | Injects spec + example into context upfront |

### 8.2 DSL Spec Resource

Resource URI: `gherkio://dsl/spec`

Returns a Markdown reference of the complete DSL grammar, derived from the actual Go structs and assertion engine.

```markdown
## Gherkio DSL Reference

### TestFile
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| scenario | string | yes | Name of the test scenario |
| steps | []Step | yes | List of steps (at least 1) |

### Step
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| use | string | conditional | Path to another test file. Required if no request |
| request | Request | conditional | HTTP request definition. Required if no use |
| expect | Expect | no | Assertions to run against the response |
| save | map[string]string | no | Extract response values into variables |
| timing | TimingConfig | no | Max duration expectation |

### Request
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| service | string | no | Named service key from environment `services:` map |
| method | string | yes | HTTP method: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS |
| url | string | yes | Path appended to the resolved baseUrl |
| headers | map[string]string | no | HTTP request headers |
| body | any | no | JSON body (map or raw string). Auto-sets Content-Type: application/json |

### Expect
`status` is a special integer field; all other keys are path:matcher pairs.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| status | int | no | Expected HTTP status code |
| `<path>: <matcher>` | inline | no | Path-based assertions (see Path Syntax + Matchers) |

Internally, the `expect` block has a custom `UnmarshalYAML` that extracts `status` and stores everything else as extra matchers.

### TimingConfig
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| max | string | yes | Maximum duration. Go format: `500ms`, `1s`, `250ms`, `2s` |

### Environment
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| baseUrl | string | yes | Default base URL for all requests |
| services | map[string]Service | no | Named service overrides (keyed by service name) |

### Service
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| baseUrl | string | yes | Service-specific base URL |

---

### Path Syntax (for expect and save)

Reference response values using dot-notation paths:

| Prefix | Context | Canonical? | Example |
|--------|---------|------------|--------|
| `body.<field>` | expect / save | ✅ Yes | `body.accessToken: exists` |
| `headers.<name>` | expect / save | ✅ Yes | `headers.content-type: application/json` |
| `jwt.<claim>` | expect / save | ✅ Yes | `jwt.role: admin` |
| `timing.max` | expect (auto) | ✅ Yes | Evaluated from the `timing:` block |
| `schema` | expect only | ✅ Yes | `schema: example/login-response` |

Backward-compatible aliases (work but not recommended for new tests):
| Prefix | Maps To |
|--------|---------|
| `response.<field>` | `body.<field>` |
| `response.body.<field>` | `body.<field>` |
| `response.headers.<name>` | `headers.<name>` |
| Bare `<field>` (fallback) | Tries `body.<field>` if no prefix match |

Array index access:
```
body.items[0].name    → first item's name
body.users[2].email   → third user's email
```

---

### Matchers

| Matcher | Description | Example |
|---------|-------------|--------|
| `exists` | Field must exist (any value, including null) | `body.id: exists` |
| `<literal>` | Exact string equality | `body.email: emilys` |
| `uuid` | Valid UUID format `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` | `body.id: uuid` |
| `email` | Valid email format (local@domain.tld) | `body.email: email` |
| `datetime` | Valid RFC3339/ISO8601 datetime | `body.createdAt: datetime` |
| `uri` | Valid URI/URL format | `body.image: uri` |
| `number` | Value is a numeric type | `body.price: number` |
| `string` | Value is a string type | `body.name: string` |
| `boolean` | Value is a boolean type | `body.active: boolean` |
| `array` | Value is an array type | `body.tags: array` |
| `object` | Value is an object type | `body.metadata: object` |
| `null` | Value is null | `body.deletedAt: null` |
| `true` | Value is boolean true | `body.verified: true` |
| `false` | Value is boolean false | `body.disabled: false` |
| `contains <s>` | String contains substring | `body.message: contains successfully` |
| `startsWith <s>` | String starts with prefix | `body.token: startsWith eyJ` |
| `endsWith <s>` | String ends with suffix | `body.file: endsWith .pdf` |
| `regex <p>` | String matches regex pattern | `body.phone: regex ^\\+?[0-9]{7,15}$` |

---

### Collection Matchers

Applied to array fields using wrapper syntax:

| Syntax | Behavior | Example |
|--------|----------|--------|
| `count(<arrayPath>)` | Assert array length | `count(body.items): 5` |
| `all(<arrayPath>.<field>)` | Assert runs against every element's field | `all(body.items.name): exists` |
| `all(<arrayPath>)` | Assert runs against every element directly | `all(body.names): string` |

---

### Variable Interpolation

Applied to: `url`, `headers` values, `body` (recursive string/map/array).

| Syntax | Behavior |
|--------|----------|
| `$varName` | Replaced with saved variable value |
| `${varName}` | Explicit delimiters (useful next to alphanumeric chars) |
| `${varName:default}` | Use default value if variable not set |

---

### Schema Definition

Located in `.gherkio/schemas/<name>.yaml`. Referenced via `expect.schema`.

| Field | Type | Applies To | Description |
|-------|------|------------|-------------|
| type | string | — | `object`, `string`, `integer`, `number`, `boolean`, `array`, `null` |
| required | []string | object | Required field names |
| properties | map[string]Schema | object | Per-field schema definitions |
| items | Schema | array | Schema for array elements |
| format | string | string | `email`, `uuid`, `datetime`, `uri` |
| enum | []any | any | Allowed values |
| pattern | string | string | Regex pattern the value must match |
| minLength / maxLength | int | string | String length constraints |
| minimum / maximum | float | number | Numeric range constraints |
| minItems / maxItems | int | array | Array length constraints |
| nullable | bool | — | Whether null is allowed |

---

### JWT Auto-Decoding

If `body.token`, `body.accessToken`, or `body.access_token` is present in the response and is a string, it's automatically decoded (no signature verification). Claims are available via `jwt.<claim>` paths in assertions and saves.

### Sensitive Field Masking

Default sensitive fields (case-insensitive):
`token`, `accessToken`, `access_token`, `refreshToken`, `refresh_token`, `password`, `secret`, `clientSecret`, `client_secret`, `apiKey`, `api_key`, `authorization`

Configurable via `.gherkio/config.yaml` > `security.mask.fields`.
```

### 8.3 JSON Schema Resource

Resource URI: `gherkio://dsl/schema.json`

Auto-generated from Go structs in `internal/model/` using reflection. Covers `TestFile`, `Step`, `Request`, `Expect`, `TimingConfig`, `Environment`, `Service`, `Schema`.

Example output for the top-level test file:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "TestFile",
  "type": "object",
  "required": ["scenario", "steps"],
  "properties": {
    "scenario": { "type": "string" },
    "steps": {
      "type": "array",
      "minItems": 1,
      "items": { "$ref": "#/definitions/Step" }
    }
  },
  "definitions": {
    "Step": {
      "type": "object",
      "properties": {
        "use": { "type": "string" },
        "request": { "$ref": "#/definitions/Request" },
        "expect": {
          "type": "object",
          "properties": {
            "status": { "type": "integer" }
          },
          "additionalProperties": { "type": "string" }
        },
        "save": {
          "type": "object",
          "additionalProperties": { "type": "string" }
        },
        "timing": {
          "type": "object",
          "properties": {
            "max": { "type": "string" }
          },
          "required": ["max"]
        }
      },
      "oneOf": [
        { "required": ["use"] },
        { "required": ["request"] }
      ]
    },
    "Request": {
      "type": "object",
      "required": ["method", "url"],
      "properties": {
        "service": { "type": "string" },
        "method": { "type": "string", "enum": ["GET","POST","PUT","PATCH","DELETE","HEAD","OPTIONS"] },
        "url": { "type": "string" },
        "headers": { "type": "object", "additionalProperties": { "type": "string" } },
        "body": { "type": ["object", "string"] }
      }
    },
    "Environment": {
      "type": "object",
      "required": ["baseUrl"],
      "properties": {
        "baseUrl": { "type": "string" },
        "services": {
          "type": "object",
          "additionalProperties": {
            "type": "object",
            "required": ["baseUrl"],
            "properties": {
              "baseUrl": { "type": "string" }
            }
          }
        }
      }
    }
  }
}
```

Generated by `internal/mcp/schema_gen.go` using `reflect` and `yaml` struct tags.

### 8.4 Examples Resource

Resource URI: `gherkio://dsl/examples`

Returns 3 canonical examples that demonstrate the core patterns:

```yaml
# Example 1: Simple request + status assert + matcher + save
scenario: login example

steps:
  - request:
      method: POST
      url: /auth/login
      body:
        username: emilys
        password: emilyspass
        expiresInMins: 30
    expect:
      status: 200
      body.accessToken: exists
      body.email: email
      schema: example/login-response
    save:
      accessToken: body.accessToken
```

```yaml
# Example 2: Composition (use) + variable interpolation
scenario: get current user profile

steps:
  - use: login.yaml

  - request:
      method: GET
      url: /auth/me
      headers:
        Authorization: Bearer $accessToken
    expect:
      status: 200
      body.id: exists
      body.username: emilys
```

```yaml
# Example 3: Named service + timing + collection matcher
scenario: Complete E2E Object Workflow

steps:
  - use: 1-create-a-new-object.yaml

  - request:
      service: restful-api
      method: GET
      url: /objects/$created_id
    expect:
      status: 200
      body.name: "Apple MacBook Pro 16"

  - request:
      service: restful-api
      method: PUT
      url: /objects/$created_id
      headers:
        Content-Type: application/json
      body:
        name: "Apple MacBook Pro 16 - M3 Max"
        data:
          year: 2024
          price: 2699.99
    expect:
      status: 200
      body.name: "Apple MacBook Pro 16 - M3 Max"
      body.data.price: 2699.99
    timing:
      max: 2000ms
```

### 8.5 Prompt Templates

| Prompt | Description |
|--------|-------------|
| `new_test` | "I want to create a new Gherkio test." — Injects DSL spec + example 1 + blank template |
| `new_environment` | "I want to add a new environment." — Injects environment spec + template |

### 8.6 LLM Self-Correction Flow

When the LLM writes invalid YAML, the validation error guides it to fix itself:

```
LLM writes:
  steps:
    - request:
        methode: POST  # typo

← MCP returns:
  ValidationError:
    field: "steps[0].request"
    code: "unknown_field"
    message: "Unknown field 'methode'. Did you mean 'method'?"
    hint: "Valid request fields: service, method, url, headers, body"

→ LLM corrects:
  steps:
    - request:
        method: POST  # fixed
```

This creates a self-healing loop — the LLM reads the error, fixes the YAML, and retries without human intervention.

---

### 8.7 Already-Completed Work (Not Needing Rebuild)

The following infrastructure proposed in this RFC has already been built in subsequent RFCs:

| Feature | Built In | Location |
|---------|----------|----------|
| JSON Schema generation | RFC-15 | `internal/schema/generator.go` — multi-type, all YAML types |
| `LoadTestFile()` exported | RFC-12 | `internal/runner/runner.go` — already public |
| `LoadCredentials()` exported | RFC-12 | `internal/runner/credentials.go` — already public |
| `LoadConfig()` exported | RFC-8 | `internal/runner/config.go` — already public |
| `CredentialVars` on `RunConfig` | RFC-12 | `internal/runner/runner.go` — field exists |
| `GetCanonicalPaths()` in engine | RFC-15 | `internal/runner/engine.go` — source of truth |
| `GetAvailableMatchers()` in engine | RFC-15 | `internal/runner/matchers.go` — source of truth |
| `GetCollectionFunctions()` in engine | RFC-15 | `internal/runner/engine.go` — source of truth |
| `GetBackoffStrategies()` in engine | RFC-15 | `internal/runner/engine.go` — source of truth |

These do not need to be rebuilt. The `internal/core/` packages will **wrap** these existing functions rather than duplicating them.

---

## 9. Backward Compatibility

- All existing CLI commands remain unchanged
- `cmd/run.go` and `cmd/init.go` delegate to new packages — no functional change
- The `runner.Run()` function already has `CredentialVars` field on `RunConfig` (added in RFC-12) — no signature change needed.
- No YAML schema changes
- No breaking changes to existing test files

---

## 10. Open Questions

- MCP Go SDK vs building a minimal stdio transport manually (minimal is simpler, SDK adds spec compliance)
- Should `validate_test` also validate environment/service cross-references at runtime, or only at creation time?
- For `run_test`, should results include the full request/response payload inline, or only assertion results plus a ref to saved artifacts?
