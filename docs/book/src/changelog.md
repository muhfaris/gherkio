# Changelog

All notable changes to Gherkio will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- **Redis Cache Assertions**: Added controlled, read-only Redis steps for `GET`, `EXISTS`, `TTL`, and `HGETALL`, including `redis.*` assertions, saved values, retries, timing checks, authentication, TLS, database selection, and Redis Sentinel primary discovery.
- **Virtual-User Load Runs**: Added `--virtual-users` and `--iterations-per-user` for concurrent isolated users running complete workflows sequentially, with virtual-user and iteration metadata included in reports.
- **Response-Aware Random Selection**: Added `${randomItem(array[,field])}` using the runtime collection length, including typed object preservation when the expression is used as an exact `set:` value.
- **String Helpers**: Added `${trimPrefix(...)}`, `${trimSuffix(...)}`, and `${split(...,delimiter,index)}` interpolation helpers.
- **Configurable Multipart Assets**: Added `assets.path` to `.gherkio/config.yaml` while retaining the existing project-root and fixtures fallbacks when it is omitted.
- **Bounded Multi-Step Loops**: Added `repeat.attempts`, `repeat.until`, and nested `repeat.steps` for workflows that must reselect data and refetch an API until a response-derived condition succeeds, with attempt labels in terminal and HTML reports.
- **Executable Feature Catalog**: Expanded the built-in `.gherkio` project into consolidated, self-contained mocked workflows covering composition, multipart uploads, collections, repeat loops, runtime helpers, matchers, and load execution, plus opt-in direct Redis and Sentinel examples.

### Changed
- **Request Pacing**: Added `--request-delay` with sensible defaults for individual files and directory runs.
- **Step-Prefixed Variables**: Saved names such as `1-ticketId` and `2-userId` now validate, interpolate, and convert consistently.
- **Load-Test Reporting**: HTML reports now present load-run summaries and virtual-user workflow executions with clearer expandable details.
- **Documentation Theme**: Set Navy as the default mdBook theme and placed Mermaid diagrams on a light canvas for readable labels across Light, Navy, Ayu, and Coal.
- **AI Documentation Reliability**: Added a compact machine-readable reference, refreshed `llms.txt`, corrected stale runtime claims, and introduced consistency tests for CLI flags, book links, MCP request fields, matchers, and built-in functions.
- **Saved Collection Counts**: `save:` now accepts `count(body.<path>)`; arrays save their length and explicit `null` values save `0` without hiding missing or incorrectly typed paths.
- **Variable Snapshot Reporting**: Reports now distinguish variables available before execution from final variables preserved after repeat blocks; generated cURL commands mask authorization headers, and `authToken` variables use built-in masking.

### Fixed
- **Transport Failure Accounting**: Request construction and transport errors now fail the scenario and contribute to failure totals instead of allowing a false-green result.
- **Composed Variable Validation**: Project validation now recognizes example variables and outputs saved or set by nested `use` workflows, and accepts negated schema assertions.
- **Strict MCP Schemas**: MCP tool input schemas now emit explicit object properties and strict validation metadata required by stricter clients.
- **Deterministic Query Parameters**: Request URLs now serialize query parameters in a stable order.

## [0.1.0-alpha.8] - 2026-08-08

### Added
- **Payload-Aware MCP Authoring**: MCP scenario creation prompts can derive useful request bodies, queries, headers, expectations, and saved values from supplied API payload context.

### Fixed
- **MCP Step Isolation**: Running step index `0` through `run_test` now executes only that step instead of falling back to the complete scenario.

## [0.1.0-alpha.7] - 2026-07-23

### Added
- **Collection Assertions**: Added `any(path)` and expanded `all(path)` collection matching, including nested collection fields.
- **Session Persistence**: Saved scenario values can persist between runs and are available to reverse cURL conversion and subsequent workflows.
- **Collection Diagnostics**: HTML reports show collection assertion details and clearer matching failures.

### Changed
- **Showcase Suite**: Standardized the example scenarios, added third-party Stripe virtualization, and integrated the showcase into CI.

## [0.1.0-alpha.6] - 2026-07-15

### Added
- **Data-Driven Testing (Parametrization)**: Scenario-level execution iteration using the `examples` block.
- **Composed Traceability**: Visually grouped composed scenario blocks, dynamic nesting indentation by depth, and local context variable snapshots in HTML reports.
- **Outbound Service Mocking & Interception**: Direct request interception and virtualized responses defined in environment files, with full variable interpolation support.
- **JSONPath Assertions**: Support for standard JSONPath queries (`$.`) inside response body expectation assertions.
- **Step Locator & Reverse Conversion Enhancements**: Integrated session variables and robust parsing of nested list structures when converting steps back to cURL.
- **Rich Schema Hover Documentation**: Enhanced autocomplete JSON schemas with detailed description strings for editor tooltips.
- **Declarative Variable Assignment (`set:`)**: Direct inline variable assignment step style.
- **Environment Context Command (`env context`)**: Interactive auto-selection hints command.

## [0.1.0-alpha.5] - 2026-06-22

### Added
- **Fail-Fast Mode**: Added runner and MCP support for stopping after the first failure while still guaranteeing teardown execution.
- **Composition Overrides**: Added `with:` variables for `use:` steps, with interpolation and restoration of the caller's values after composition completes.
- **Save-Path Warnings**: Missing response paths referenced by `save:` now produce non-fatal warnings in terminal, JSON, and HTML output.
- **Resolved Variables in Reports**: Reports now expose resolved workflow variables for debugging and traceability.
- **MCP Documentation Improvements**: Expanded DSL resources with multipart, query, bracket-path, assertion, and granular execution guidance.

### Changed
- **Documentation Discoverability**: Added AI-crawler metadata, FAQs, and denser answer-first project documentation.

## [0.1.0-alpha.4] - 2026-06-10

### Added
- **Conditional Steps**: Added value-based conditional execution for declarative workflows.
- **Scenario and Step Metadata**: Added scenario `description`, step `name`, and request `query` fields.
- **Structured Request Reports**: HTML and JSON reports now preserve step names and expose request method, URL, headers, query parameters, and body details.
- **Editor and Report Documentation**: Added dedicated references for JSON Schema editor integration and generated test reports.

### Fixed
- **MCP Authoring Feedback**: Improved MCP test creation and execution behavior based on early user feedback.

## [0.1.0-alpha.3] - 2026-05-31

### Added
- **Declarative Collection Transforms**: Added filtering, slicing, projection, and explicit type casting for response collections used in later requests.
- **Partial Execution with `--until`**: Added execution through a selected step boundary.
- **Swagger Petstore Showcase**: Added a complete public API example suite.

### Fixed
- **JSON Number Precision**: Preserved large integer values without lossy floating-point conversion.
- **Report Error Handling**: Corrected failures encountered while producing reports.
- **MCP DSL Resources**: Exposed variable and DSL guidance through MCP resources.

## [0.1.0-alpha.2] - 2026-05-29

### Added
- **Gherkio Developer Book**: Introduced a fully-fledged developer guide containing 23 chapters built with mdBook covering CLI tools, DSL syntax, and advanced recipes.
- **Interactive Browser Playground**: Added a self-contained, offline-first visual interface supporting:
  * *Visual DSL Stepper*: Dynamic execution flowchart rendering.
  * *cURL ↔ YAML Step Translator*: Real-time legacy terminal logs conversion.
- **Hosted GitHub Pages Support**: Configured static deploy workflows to serve both the developer book and the playground in a unified workspace.
- **Stunning System Architecture Visual**: Embedded a premium, dark-mode glassmorphic diagram explaining Gherkio's MCP daemon and execution flow.
- **Expanded Phone Number Support**: Added broader localized randomized phone number formats supporting international validations.

### Fixed
- Fixed playground asset bundling path mapping inside the `docs-build` compilation targets.
- Corrected search index generation schemas for mdBook chapter queries.

---

## [0.1.0-alpha.1] - 2026-05-28

### Added
- **RFC-18 Automated Failure Debug Snapshots**: Auto-capturing raw request/response payload snapshots during test failures.
- **RFC-19 Domain Sandboxing & SSRF Prevention**: Outbound network sandboxing policies with user-configurable allowlists and blocklists.
- **RFC-21 Multipart Form-Data & File Upload**: Streaming of multipart files via Go `io.Pipe`.
- **Dynamic Parameterized Generators**: Parameterized variable generators supporting offset dates and randomized values.
- **Expanded MCP Server Tools**: Exposing `gherkio init`, `gherkio validate`, and `gherkio convert` directly as discoverable tools in the MCP layer.
- **Cobra CLI Doc Generator**: Automatically building Cobra CLI manual pages to markdown.

---

## [0.1.0-alpha] - 2026-05-26

### Added
- **Core Declarative Engine**: Sequential HTTP requests, responses, headers, and parameter binding.
- **Value Matchers & Negative Assertions**: Core validations (`exists`, `equals`, `contains`, `matches`, `not exists`, `schema: not <name>`).
- **Setup & Teardown Blocks**: Scenario setup and error-tolerant teardown block executions.
- **Retry & Polling Loops**: Configurable attempts, interval timers, and linear backoff algorithms.
- **Multi-Account Credentials Management**: Namespaced credentials loader with safe YAML integrations.
