# 🥒 Gherkio

Gherkio is a declarative, Gherkin-driven CLI for HTTP API testing. It blends reusable catalogs, templated payloads, flow macros, and rich reporters so QA and backend teams can describe whole API journeys in plain language and run them repeatably.

---

## Highlights

- **Feature-first workflow** – Write scenarios in `.feature` files and back them with an extensive step library.
- **Reusable assets** – Centralized API catalogs (`gherkio/apis/`), environment profiles (`gherkio/envs/`), flows (`gherkio/flows/`), and fixtures (`gherkio/fixtures/`).
- **Template-friendly** – Built-in Go templating with helper functions (random data, UUID, date utilities, env lookups, etc.).
- **Powerful assertions** – JSONPath assertions, numeric comparisons, regex matching, length checks, store comparisons, and more.
- **Report anywhere** – Pretty CLI output plus CSV and HTML (with optional debug payload capture).
- **Integrated tooling** – Curl/OpenAPI importer, step/function documentation commands, dry-run linting, interactive tutorial.

---

## Prerequisites

- Go 1.23 or later
- macOS, Linux, or Windows shell environment
- (Optional) Node.js 18+ if you plan to work on the MD/MDX documentation site under `docs/`

---

## Installation

### Quick installer

```bash
curl -sSL https://raw.githubusercontent.com/muhfaris/gherkio/main/install.sh | sh
```

The script downloads the latest release, verifies the binary, and places it in `~/.local/bin` (or a platform-appropriate location). Ensure that directory is on your `PATH`.

### Manual build

```bash
git clone https://github.com/muhfaris/gherkio.git
cd gherkio
go mod tidy
go build -o gherkio ./cmd/gherkio

# optional: move it somewhere on PATH
mv gherkio ~/.local/bin/
```

### Using `go install`

```bash
go install github.com/muhfaris/gherkio/cmd/gherkio@latest
```

---

## First Run

```bash
# bootstrap the default workspace under ./gherkio/
./gherkio init

# inspect the generated environment/catalog/flow/fixture/feature files
tree gherkio

# execute the sample feature bundle
./gherkio run --env dev --report html:reports/run.html
```

> The HTML report will be written to `reports/run.html`. Add `?debug` suffix (or use `html-debug`) to capture full request/response payloads.

---

## Workspace Layout

`gherkio init` scaffolds the following tree (feel free to adjust or add more folders as needed):

```
gherkio/
├── envs/          # Environment definitions (YAML)
├── apis/          # API catalog files with reusable endpoints
├── flows/         # Flow macros composed of catalog entries
├── fixtures/      # JSON payloads referenced in tests
└── features/      # Gherkin feature files
```

### Environment Files (`gherkio/envs/*.yaml`)

- `baseURL`, `headers`, `timeouts`, `retries`, and `redact` options control defaults applied to every request.
- A nested `vars:` map is loaded into the runtime store (both flattened keys like `login.username` and tree access via `{{ .vars.login.username }}` are available).
- Values are rendered through the template engine, allowing references to other vars or store entries.

**Using `vars`:**

```yaml
# gherkio/envs/dev.yaml
vars:
  login:
    username: admin@example.com
    password: '{{ fnGetEnv "ADMIN_PASSWORD" "secret" }}'
  teams:
    defaultId: 42
```

- Reference values in features or flows with `{{ .vars.login.username }}`.
- Catalog entries can reuse the same keys, e.g. `path: /users/{{ .vars.teams.defaultId }}` or headers.
- Override entries at runtime with `--vars login.password=supersecret` or `--vars teams.defaultId=99`.
- Treat `vars` as read-only configuration—store scenario data with `save`/`set` so it lives under the main store namespace instead of inside `vars`.

### Catalogs (`gherkio/apis/*.yaml`)

- Define endpoints under `endpoints`, keyed as `namespace.action` for readability.
- Support templated `path`, `headers`, `query`, `body`, and inline expectations (`expect.status`).
- Optional `auth` section declares reusable authentication profiles (`setAuth` can activate them in scenarios).

**Defining custom endpoints:**

```yaml
# gherkio/apis/core.yaml
version: 1
endpoints:
  inventory.list:
    method: GET
    path: /v1/inventory
    headers:
      X-Tenant: "{{ .vars.tenant.id }}"
    expect:
      status: 200
  inventory.create:
    method: POST
    path: /v1/inventory
    headers:
      Content-Type: application/json
  inventory.getById:
    method: GET
    path: /v1/inventory/{{ .id }}
```

1. Add the endpoint under an existing catalog file or create a new YAML inside `gherkio/apis/`.
2. Call it from features using `When I call API "inventory.create"` (optionally providing body/query/header overrides).
3. Use `expect` blocks for guarantees that should hold on every invocation (e.g., status codes, JSONPath assertions).

### Flow Macros (`gherkio/flows/*.yaml`)

- Group multiple API calls and utility actions.
- Accept `params`, can save values into the store, and call `setAuth` or nested flows.
- Flow steps mirror what you can do in features: `call`, `expect`, `save`, `setHeaders`, `run`, etc.

### Custom Hosts & Base URLs

- Configure the primary host per environment via `baseURL`:

  ```yaml
  # gherkio/envs/staging.yaml
  baseURL: https://staging-api.example.com
  headers:
    X-App: gherkio
  vars:
    api:
      auth: https://auth.example.com
  ```

- Refer to alternate hosts through `vars` in catalog definitions:

  ```yaml
  endpoints:
    auth.token:
      method: GET
      path: "{{ .vars.api.auth }}/v1/token"
  ```

- Relative catalog paths (e.g., `/v1/users`) resolve against `baseURL`; absolute URLs or templated host values override it.
- Override hosts during execution:
  - Scenario-level: `Given the base URL is "https://mock.local"`
  - Service-specific vars: `./gherkio run --env staging --vars api.auth=https://mock.local/auth`
  - Maintain separate env files per environment when the defaults differ drastically.

### Fixtures (`gherkio/fixtures/…`)

- Store reusable payloads for large JSON bodies or multipart definitions.
- Reference them from features or `gherkio call` via `@fixtures/...` paths.

### Feature Files (`gherkio/features/…`)

- Standard Gherkin syntax (`Feature`, `Scenario`, `Background`, `Given/When/Then`).
- Leverage the built-in step catalog for loading envs, calling APIs, manipulating store values, and asserting responses.

**Gherkin basics:**

```gherkin
Feature: Manage inventory

  Background:
    Given I load env "dev"
    And I load catalogs from "gherkio/apis"

  Scenario: Create a new item
    When I call API "inventory.create" with body:
      """
      {"name": "Widget", "sku": "{{ fnUUID }}"}
      """
    Then response status should be 201
    And save '$.data.id' as "item_id"

  Scenario: Fetch the created item
    Given I set path params:
      | id | {{ .store.item_id }} |
    When I call API "inventory.getById"
    Then response status should be 200
```

- **Feature** groups related behaviour; keep scenarios cohesive.
- **Background** runs before every scenario—ideal for shared setup (loading env/catalogs, resetting headers).
- Use **DocStrings** (`"""`) for JSON payloads and **tables** for path/query params or flow inputs.
- Scenario outlines are supported (`Scenario Outline` + `Examples:`) for data-driven loops; each row produces a new scenario.

---

## Templating & Runtime Store

- All catalog paths, headers, bodies, and feature doc strings/tables are rendered with Go templates (`{{ ... }}`) using the current store.
- Store entries include:
  - Environment vars nested beneath `store["vars"]` (access them with `{{ .vars.* }}`).
  - Saved values from `save` steps (both response and request JSONPaths).
  - Custom entries created via `set 'key' to 'value'`.
- Use `gherkio docs fn` to list helper functions such as `fnRandomEmail`, `fnUUID`, `fnToday`, `fnRandomInt`, `fnBase64Encode`, etc.
- JSONPath resolution uses [`tidwall/gjson`](https://github.com/tidwall/gjson) with templated paths, so constructs like `$.data.{{ .vars.teamIndex }}` resolve dynamically at runtime.

---

## Command Reference

### `gherkio init`

Initializes the `gherkio/` directory (idempotent).

- Creates env, catalog, flow, fixture, and feature folders if missing.
- Generates sample files (`dev.yaml`, `core.yaml`, `auth.yaml`, etc.) only when they do not already exist.
- Use it once per repository or when spinning up a new workspace.

### `gherkio call`

Execute a single catalog endpoint.

| Flag | Description |
| ---- | ----------- |
| `--env <name>` | Environment profile to load (defaults to `dev`). |
| `--api <key>` | Catalog key, e.g. `users.getById`. **Required.** |
| `--body <payload>` | Inline JSON or `@fixtures/path.json`. Multipart fixtures are automatically handled. |
| `--path key=value` | Path parameters (repeatable). |
| `--query key=value` | Query parameters (repeatable). |
| `--header key=value` | Extra headers (repeatable). |
| `--expect-status <code>` | Quick assertion on the final HTTP status. |
| `--report kind[:path]` | Optional reporters (`csv`, `html`, `html-debug`, etc.). Multiple allowed. |

Outputs a pretty-printed summary to stdout and, if requested, writes additional reports.

### `gherkio run`

Run one or more feature files with Godog.

| Flag | Description |
| ---- | ----------- |
| `--env <name>` | Environment profile. |
| `--debug` | Capture request/response payloads for HTML debug output. |
| `--tags "<expression>"` | Tag filter (Godog syntax, e.g. `@smoke and not @wip`). |
| `--name <regex>` | Filter scenarios by name. |
| `--parallel <n>` | Parallelize by feature file. |
| `--feature <path/glob>` | Include specific feature files (repeatable). |
| `--exclude-feature <path/glob>` | Exclude feature files (repeatable). |
| `--report kind[:path]` | Reporter list (`pretty`, `csv`, `html[:path]`, `html-debug[:path]`). |
| `--dry-run` | Lint feature files without hitting external services. |
| `--vars key=value` | Override environment variables at runtime (supports nested keys like `auth.token=abc`). |

Behavior:
- Loads env, catalogs, flows before binding steps.
- Applies include/exclude filters, parallel shards, and scenario name filters.
- Aggregates step logs and writes reporters at the end of the run.
- Fails fast if no feature matches or required files are missing.

### `gherkio import`

Generate catalog/fixture/feature scaffolding from curl or OpenAPI sources.

**Curl mode (`--api` + `--curl`):**
- Parses headers, method, URL, body, and multipart form data.
- Replays the request unless `curl --silent` style flags are present (to capture sample response & default assertions).
- Interactively proposes:
  - Catalog path (default `gherkio/apis/imported.yaml`).
  - Fixture path (derived from API key when body detected).
  - Feature file path and scenario title.
  - Optional JSON assertions generated from the sample payload.
- Copies referenced files in multipart forms into the fixtures directory.

**OpenAPI mode (`--openapi <spec>`):**
- Translates every operation into catalog entries, optionally namespaced via `--prefix`.
- Writes fixtures for example/request bodies to `--fixtures` (default `gherkio/fixtures/openapi`).
- Produces a merged catalog file (`--catalog`, default `gherkio/apis/openapi.yaml`).

### `gherkio docs fn`

Lists all available template helper functions with description, example, and sample output. Supports piping to a file for quick reference.

### `gherkio docs steps`

Prints the registered Gherkin step catalog. Flags:

- `--match <term>` (repeatable) to filter by substring.
- `--format md` to emit Markdown tables (handy for wiki exports).
- `--out <path>` to save the results.

### `gherkio tutor`

Launches an interactive, multi-level tutorial that guides you through simple, advanced, complex, fixture-based, and flow-based scenarios. Ideal for onboarding teammates to the workflow.

---

## Step Library Overview

Gherkio binds a comprehensive set of steps (run `gherkio docs steps` for the full list). Common categories:

- **Environment & Utilities**
  - `Given I load env "dev"`
  - `Given the base URL is "https://..."` (scenario override)
  - `I include feature "other.feature"`
  - `I wait 500ms`
- **Request Setup**
  - `I set path params:`, `I set query params:`, `I clear query params`
  - `I set headers:` (tables or doc strings)
  - `set 'key' to 'value'` (store injection)
- **Calling APIs & Flows**
  - `I call API "users.getById"`
  - `I call API "auth.login" with body:` (doc string or table)
  - `I call API "users.create" using fixture "users.create.json"`
  - `I run flow "login"` / `I run flow "login" with:` (table parameters)
  - `I set auth "bearer"` to switch profiles defined in catalogs.
- **Assertions**
  - Status checks: `response status should be 200`, `response status should be in 200-299`, `response time should be <= 500ms`
  - Headers: `header "Content-Type" should equal "application/json"`
  - JSONPath: existence, equality (string/number/boolean/null), regex, numeric comparisons, length comparisons, emptiness, set membership.
  - Store comparisons: `json '$.id' should equal store 'resource_id'` (with optional `ignoring order`), `json '$.data' should match store request 'payload'`
- **Store Management**
  - `save '$.token' as "access_token"`
  - `save request json '$.name' as "resource_name"`
  - `save request body as "request_payload"`
  - `Then the store should contain:` (table of expected values)
  - `show variable "access_token"`
- **Debugging & Persistence**
  - `response body should contain "..."`, `print response`
  - `save response body to file "artifacts/response.json"`

**Assertion tips:**

- Prefer JSONPath assertions over substring checks—they fail fast and produce precise error messages.
- Save values with `save` and compare them later using `json '$.field' should equal store 'key'` (add `ignoring order` for arrays).
- Validate list endpoints with `json '$.items' length should be > 0` before accessing indexes.
- Use regex steps (`json '$.email' should match "[^@]+@example.com"`) when the exact value changes but the format must hold.
- To assert specific items inside arrays, use indexed JSONPath (`$.data.0.id`) when the position is stable, or search with dotted indices (`$.data.#(name=="Widget").id`) for dynamic lists. Save the result before chaining additional asserts.
- Combine status checks with content asserts to catch partial failures (e.g., 200 but missing fields).
- When debugging flaky tests, enable HTML debug or `save response body to file` to inspect payloads offline.

Specialized helpers (`I create <n> document groups`, `I delete created document groups`) demonstrate how domain-specific automation can live alongside generic steps; extend the bindings to match your own workflows.

---

## Reports

| Kind | CLI Spec | Output |
| ---- | -------- | ------ |
| Pretty | default | Structured console output with per-step status. |
| CSV | `--report csv[:path]` | Scenario-level log (defaults to `reports/run.csv`). |
| HTML | `--report html[:path]` | Rich report with feature/scenario/step drilldown. |
| HTML Debug | `--report html-debug[:path]` or `--debug` | Same as HTML plus prettified request/response payloads. |

Reports accumulate across the run; ensure the `reports/` directory exists or let Gherkio create it on demand.

---

## Function Templates

Function templates extend standard Go templating. A non-exhaustive sampling:

- `{{ fnRandomEmail "example.com" }}` – Random email.
- `{{ fnRandomString "alphanum" 12 }}` – Random string of given charset and length.
- `{{ fnRandomInt 100 999 }}` – Random integer between bounds (inclusive).
- `{{ fnUUID }}` – New UUID v4.
- `{{ fnToday "2006-01-02" "UTC" }}` – Current date in format/timezone.
- `{{ fnFutureDate 7 "2006-01-02" }}` – Date offset.
- `{{ fnBase64Encode "value" }}`, `{{ fnGetEnv "VAR" "fallback" }}`, etc.

Run `gherkio docs fn` to see the full list with examples and descriptions.

---

## CI/CD Integration

Example GitHub Actions snippet:

```yaml
- uses: actions/setup-go@v5
  with:
    go-version: "1.23"
- run: go build ./cmd/gherkio
- run: ./gherkio run --env dev --report csv:reports/run.csv --report html:reports/run.html
- uses: actions/upload-artifact@v4
  with:
    name: gherkio-reports
    path: reports/
```

Tips:
- Combine `--tags` or `--name` filters to slice suites for parallel jobs.
- Use `--vars key=value` to inject CI secrets without editing YAML files.
- For smoke checks, append `--dry-run` first to catch missing step bindings or syntax errors quickly.

---

## Troubleshooting & Best Practices

- **Missing catalog/env/flow** – Ensure `gherkio init` has been run and file names match the lookup path.
- **JSONPath not found** – Remember that `save`/`json` steps operate on the last HTTP response; if you expect data from another call, save it explicitly.
- **Authorization headers** – Define auth profiles in catalogs and call `setAuth`. For per-request overrides, use `I set headers:` or catalog-level `headers`.
- **Large payloads** – Store them in fixtures and reference via `@fixtures/...` to keep feature files concise.
- **Templating errors** – Most functions fail silently to preserve original string; use `show variable "<key>"` to debug store content.
- **Parallel runs** – Be mindful of shared resources when using `--parallel`; isolate fixtures or use unique identifiers (`fnRandomUnix`, `fnUUID`).
- **Extending steps** – Add new bindings in `internal/runner/steps_godog.go` following existing patterns, then recompile the CLI.

---

## License

MIT

Happy testing! 🥒
