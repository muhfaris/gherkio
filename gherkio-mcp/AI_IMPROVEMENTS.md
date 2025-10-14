# AI-Centric Enhancements for `gherkio-mcp`

This note captures adjustments that would make `gherkio-mcp` more helpful when an AI
assistant is generating API testing scenarios on behalf of users. The
recommendations build on the current MCP bridge implementation in
`internal/server`.

## 1. Richer Resource Catalogue

* **Expose structured descriptors for APIs and fixtures** – Extend
  `internal/server/resources.go` so `ResourceDescriptor` includes
  machine-readable metadata (HTTP method, path, sample payload hints).
  AI agents can then query `resources/read` and understand how to
  exercise an endpoint without manual lookups.
* **Surface step templates** – Add a `templates/` directory to the
  catalogue containing canonical Given/When/Then snippets for common API
  actions (authentication, negative cases, etc.). Listing these through
  MCP gives the AI concrete building blocks when expanding terse user
  prompts into full scenarios.

## 2. Scenario Authoring Tools

* **New `gherkio.scenario.suggest` tool** – Layered on top of the
  existing feature writer, this helper now accepts structured inputs
  (HTTP method, endpoint, request/response expectations, fixtures to
  reuse) and returns polished scenario steps alongside a ready-to-save
  feature template. The AI can review the generated Gherkin before
  persisting via `gherkio.feature.write`.
* **Validation hooks** – Before writing a feature, call the main
  `gherkio` CLI in dry-run mode to ensure referenced envs/apis/fixtures
  exist. Returning validation errors early stops the AI from generating
  broken artefacts.

## 3. Execution Utilities

* **Idempotent sample calls** – Enhance `gherkio.call` so it can return
  redacted snippets of the executed request/response (headers, body
  shapes). AI tools can use these artefacts to justify the generated
  scenarios or craft `Then` assertions automatically.
* **Scenario preview** – Done. `gherkio.feature.preview` renders the final
  Gherkin text without touching disk so the AI can request sign-off before
  persisting via `gherkio.feature.write`.

## 4. Documentation & Schemas

* **JSON Schema exports** – Publish the tool input schemas (for call,
  run, feature writing/generation) as separate resources. When an AI
  requests `resources/read`, it can load the schema and guide users on
  required fields.
* **Usage playbooks** – Add docs with end-to-end examples showing how a
  short natural-language prompt maps to tool calls. This helps align
  expectations when the module is split into its own repository.

Implementing the above will make `gherkio-mcp` a stronger bridge between
natural-language prompts and executable API test journeys.
