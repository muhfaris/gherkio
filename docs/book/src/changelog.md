# Changelog

All notable changes to Gherkio will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

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

## [0.1.0-alpha.1] - 2026-05-20

### Added
- **RFC-18 Automated Failure Debug Snapshots**: Auto-capturing raw request/response payload snapshots during test failures.
- **RFC-19 Domain Sandboxing & SSRF Prevention**: Outbound network sandboxing policies with user-configurable allowlists and blocklists.
- **RFC-21 Multipart Form-Data & File Upload**: Streaming of multipart files via Go `io.Pipe`.
- **Dynamic Parameterized Generators**: Parameterized variable generators supporting offset dates and randomized values.
- **Expanded MCP Server Tools**: Exposing `gherkio init`, `gherkio validate`, and `gherkio convert` directly as discoverable tools in the MCP layer.
- **Cobra CLI Doc Generator**: Automatically building Cobra CLI manual pages to markdown.

---

## [0.1.0-alpha] - 2026-05-15

### Added
- **Core Declarative Engine**: Sequential HTTP requests, responses, headers, and parameter binding.
- **Value Matchers & Negative Assertions**: Core validations (`exists`, `equals`, `contains`, `matches`, `not exists`, `schema: not <name>`).
- **Setup & Teardown Blocks**: Scenario setup and error-tolerant teardown block executions.
- **Retry & Polling Loops**: Configurable attempts, interval timers, and linear backoff algorithms.
- **Multi-Account Credentials Management**: Namespaced credentials loader with safe YAML integrations.
