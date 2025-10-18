# Gherkio AI Authoring Guide

This document is the single reference for generating API-testing assets with Gherkio. It covers the mental model, file formats, reusable snippets, and the latest step bindings so an assistant can produce runnable suites on the first try.

---

## 1. Project Primer

**What is Gherkio?**  
CLI tool that runs API journeys described in Gherkin (`.feature`). It loads environment/catalog/flow YAML, executes HTTP calls, stores data in a scenario store, and emits reports (pretty, CSV, HTML, JUnit, Cucumber JSON).

**High-level layers**

| Layer            | Responsibility                                | Key Paths                           |
| ---------------- | ---------------------------------------------- | ----------------------------------- |
| CLI             | Commands like `init`, `call`, `run`, `import`   | `cmd/gherkio`, `internal/cli/cmd`   |
| Loader          | Reads YAML/JSON env, catalogs, flows, fixtures  | `internal/loader`                   |
| Runner          | Steps, store, HTTP executor, Godog bindings     | `internal/runner`                   |
| Reporter        | Pretty/CSV/HTML/JUnit/Cucumber outputs          | `internal/report`                   |

**Default workspace structure (after `gherkio init`)**

```
gherkio/
├── envs/          # <env>.yaml: baseURL, headers, vars (preloaded into store)
├── apis/          # <catalog>.yaml: endpoints + auth profiles
├── flows/         # <flow>.yaml: reusable call chains
├── fixtures/      # JSON payloads referenced by features/flows
└── features/      # *.feature files (atomic/e2e/journey)
```

**Core commands**

- `gherkio run --env dev [--tags ...] [--report html:...]`
- `gherkio run --env dev --dry-run` (lint feature syntax + catalogs/flows, no HTTP)
- `gherkio run --env dev --debug` (captures request/response + prints)
- `gherkio call --env dev --api users.getById --path id=123`
- `gherkio import --api demo.create --curl 'curl ...'`

---

## 2. Authoring Workflow

1. **Plan**: update PRD/tech docs if business behavior changes.
2. **Model**: extend catalogs (`gherkio/apis/*.yaml`), flows (`gherkio/flows/*.yaml`), env vars.
3. **Write Gherkin**: create features under `gherkio/features/` (atomic/e2e/journey).
4. **Use store & templating**: capture IDs/tokens, reference them with `{{ .store.key }}` in payloads, headers, JSONPath expressions, etc.
5. **Lint first**: `gherkio run --env dev --dry-run` (fails on undefined steps, missing flows/endpoints/auth profiles, or invalid Gherkin syntax).
6. **Execute**: `gherkio run --env dev --report html-debug:reports/e2e.html` (strict mode enabled ⇒ undefined/pending steps fail).
7. **Review reports**: HTML report shows nested Feature → Scenario → Step groups with captured request/response, headers, URL, API key, status.
8. **CI**: `go build ./cmd/gherkio && ./gherkio run --env dev --report junit:reports/junit.xml`.

---

## 3. Key Concepts

### Scenario Store

- Map stored in `{{ .store.* }}` namespace.
- Seeded from `envs/<env>.yaml` `vars:` section at scenario start.
- Populated via steps: `save '$.data.id' as 'resource_id'`, `set 'token' to 'foo'`, flows `save` blocks, catalog `save`.
- Accessible inside:
  - feature docstrings/data tables (`{{ .store.order_id }}`)
  - catalog path/headers/body templates
  - flow steps (path/query/headers/body/fixture templating)
  - JSONPath arguments (e.g., `json '$.items.#(id=="{{ .store.item_id }}")' should exist`)

### Templating

- Go `text/template` + Sprig helpers.
- Available contexts: `.store`, flow params (`{{ .username }}`), map/table values, helper funcs (`{{ now | date "2006-01-02" }}`), and custom functions such as `{{ fnRandomString 12 }}` (defaults to alphanumeric) or `{{ fnRandomString "numeric" 8 }}`, `{{ fnUUID }}`, `{{ fnGetEnv "API_KEY" }}`.
- Missing template variables fall back to raw string (helps during authoring).

### Flows

- YAML map under `flows:`. Example:

```yaml
flows:
  authenticate-superadmin:
    steps:
      - call: Login
        body: |
          {"username": "{{ .store.superadmin_username }}", "password": "{{ .store.superadmin_password }}"}
        save:
          "$.data.access_token": access_token
        expect:
          status: 200
```

- Params can be passed via `When I run flow "login" with:` data table. Missing params fallback to `store[param]` or `store["flowName.param"]`.
- In flows, `setAuth`, `expect.status`, `save` behave like in features.

### Catalog

- `endpoints:` map with `method`, `path`, optional `headers`, `auth`, `expect`, `save`.
- `path` supports `{param}` from feature path params and template expressions referencing store.
- Auth profiles (`auth:`) can pull from store (`fromStore`) or env (`usernameEnv/passwordEnv`).

### Environment

- `baseURL`, `headers`, `vars`, `timeouts`, `retries`.
- `vars` injected into store at scenario start.
- `Given the base URL is '...'` overrides `baseURL` per scenario (supports templating).
- Standard Gherkin constructs (Feature/Background/Scenario/Scenario Outline/Tags) are all supported; `Background` steps run before every scenario in the same feature and follow the same binding rules listed below.

---

## 4. Step Binding Quick Reference (Latest)

Only listing frequently used ones. Full catalog in `docs/pages/reference/step-catalog.mdx`.

| Pattern | Description | Example |
| --- | --- | --- |
| `I call API "key"` | Call endpoint using current `lastReq` (path/query/body/headers previously set) | `When I call API "users.getById"` |
| `I call API "key" with:` | Table → JSON body (templated) | See below |
| `I call API "key" with body:` | DocString body (templated) |  |
| `I call API "key" using fixture "file"` | Load fixture JSON (templated) |  |
| `I set path params:` | Populate request path map (templated) |  |
| `I set query params:` / `I clear query params` | Populate or reset query map |  |
| `I set headers:` | Set headers |  |
| `header "Name" should exist/equal "Value"` | Header assertions |  |
| `I run flow "name"` / `with:` | Execute flow | `When I run flow "authenticate-superadmin"` |
| `I set auth "profile"` | Select auth profile |  |
| `save '$.jsonpath' as 'key'` | Save response JSON to store |  |
| `save request json '$.jsonpath' as 'key'` | Save request JSON to store |  |
| `save request body as 'key'` | Persist entire request payload to store |  |
| `save response body to file 'path'` | Persist response to disk |  |
| `set 'key' to 'value'` | Literal → store (templated) |  |
| `json '$.path' should exist` | Assert JSONPath exists |  |
| `json '$.path' should equal store 'key'` | Compare with store |  |
| `json '$.path' should equal store 'key' ignoring order` | Compare arrays ignoring order |  |
| `json '$.path' should equal true|false|null` | Literal bool/null assert |  |
| `json '$.path' should match store request 'key'` | Compare with stored request payload |  |
| `json '$.path' should match 'regex'` | Regex |  |
| `json '$.path' should be == 5` / `length should be == 5` | Numeric/length comparisons |  |
| `json '$.path' should be empty` / `should not be empty` | Emptiness checks |  |
| `json '$.path' should not exist` | Assert absence |  |
| `json '$.path' should be one of:` | Membership (docstring list) |  |
| `response status should be 200` | Assert HTTP status |  |
| `response body should contain "text"` | Substring check |  |
| `the response body should be a valid JSON` | Validate body parses as JSON |  |
| `the store should contain:` | Table asserts store keys/values |  |
| `the store should contain "key"` | Assert key exists |  |
| `the store should not be empty` | Store-level sanity check |  |
| `I print the store` / `print response` | Diagnostic output |  |
| `show variable "key"` | Print single store entry |  |
| `I wait 500ms` | Sleep |  |
| `the base URL is '...'` | Override base URL (templated) |  |

**Table body example**

```gherkin
When I call API "auth.login" with:
  | username | {{ .store.superadmin_username }} |
  | password | {{ .store.superadmin_password }} |
Then response status should be 200
And the response body should be a valid JSON
```

**Store assertion table**

```gherkin
Then the store should contain:
  | key              | value                       |
  | access_token     |                             | # only existence checked
  | created_user_id  | {{ .store.expected_user_id }} |
  | retry_count      | 3                           |
```

> **Tip:** Any JSON path argument (e.g., `json '$.data.id' should exist`) also supports templating, so you can write expressions like `json '$.data.#(code=="{{ .store.document_file_code }}")' should exist` to match values saved in the store.

**Project helpers**  
The repository ships a couple of domain-specific convenience steps (`I create <n> document groups`, `I delete created document groups`). Use them only when you really target the same document-group APIs; otherwise prefer modelling the behavior with dedicated flows or ordinary steps so suites stay portable.

---

## 5. Dry Run & Debugging

- `gherkio run --env dev --dry-run`
  - Parses all features (respecting `--feature`, `--tags`, etc.).
  - Verifies each step matches a registered binding (catalog is auto-populated lazily).
  - Fails on unknown flows/endpoints/auth profiles referenced with static names.
  - Skips checks when flow/API/auth contains template expressions (`{{ ... }}`).
  - Prints suggestions (top 3) for close matches.
- `gherkio run --env dev --debug`
  - Enables console logging + HTML debug attachments.
  - Step log captures: API key, request method, URL, headers, request body, response status, response body.
- `--report html-debug:path` or `--report html:path?debug` persists debug info per step.

---

## 6. Golden Samples

Use these as templates for new suites:

1. **examples/edocument**: Full journey (`features/e2e/*.feature`) using flows (`flows/main.yaml`) and catalog (`apis/core.yaml`).
2. **gherkio/features/atomic/**: Minimal atomic endpoints with store usage.

When generating content, keep:

- Features grouped by Feature/Scenario with clear tags (`@smoke`, `@journey`, etc.).
- Save IDs/tokens immediately after relevant calls.
- Reuse stored values in subsequent steps via templating.
- Assert both status and body (JSON structure, key values).
- Clean up if the system requires (delete calls, flows).

---

## 7. Common Pitfalls Checklist

- ☐ Story uses flow or API key that actually exists in `gherkio/flows` or `gherkio/apis`.  
- ☐ Every new step binding added to code synced here (update table above).  
- ☐ No hardcoded baseURL—use env vars or the base URL override step.  
- ☐ Credentials/tokens taken from env vars or store, not hardcoded.  
- ☐ IDs saved via `save` before reuse.  
- ☐ Always run `--dry-run` prior to full `run`.  
- ☐ HTML debug report path set for ease of triage.  
- ☐ Tags defined for quick filtering (e.g., `@smoke`, `@critical`).  
- ☐ For multi-file projects, features sorted deterministically (already enforced).  
- ☐ Response validations include shape/content (not just status).  
- ☐ Table assertions ensure expected store entries exist post-flow.

---

## 8. Prompt Snippet Template (for external assistants)

```
You are generating Gherkio API tests.

Reference:
- Project summary, structure, templating rules (see Gherkio AI Authoring Guide).
- Use store via {{ .store.* }}, flows in gherkio/flows/*.yaml, endpoints in gherkio/apis/*.yaml.
- Mandatory steps: save important IDs/tokens, assert status + JSON content, run "And the response body should be a valid JSON" where applicable.
- Always ensure generated steps exist in the Step Binding Quick Reference.
- After output, remind to run `gherkio run --env dev --dry-run`.

Deliver:
- Feature file snippet(s) under appropriate directory (atomic/e2e/journeys).
- Any new flow/catalog/env additions in YAML if required.
- Brief note of assumptions or required env vars.
```

Copy this guide + prompt snippet into any AI request so it responds with correct syntax and coverage.

---

## 9. Maintenance Notes

- Update this guide whenever you add/change:
  - Step bindings (regexp, behavior, debug capture).
  - CLI flags/commands (e.g., new reporters, dry-run semantics).
  - Store/templating behavior.
  - Directory conventions or naming.
- Treat `docs/gherkio-ai-guide.md` as the single source before editing other docs.

Happy testing! 🥒
