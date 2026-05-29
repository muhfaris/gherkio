# Changelog

All notable changes to Gherkio will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.1.0-alpha.1] - 2026-05-29

### Added
- **RFC-18 Automated Failure Debug Snapshots**: Auto-capturing raw request/response payload snapshots to local disk during test failures.
- **RFC-19 Domain Sandboxing & SSRF Prevention**: Outbound network sandboxing policies with user-configurable allowlists and blocklists.
- **RFC-21 Multipart Form-Data & File Upload**: High-performance streaming of file attachments and multipart fields via Go's `io.Pipe`.
- **Bidirectional cURL ↔ Gherkio YAML Translator**: Added `gherkio convert` CLI target and integrated matching API inside Gherkio's MCP server.
- **Dynamic Parameterized Variables Function Generators**: Added generator support for date offsets (`${dateOffset(...)}`), random integers (`${randomInt(min,max)}`), and localized random phones (`${randomPhone(prefixOrCountry)}`) with support for 23 global dialing codes.
- **Sequential Variable Naming Enforcement**: Strict validation ensuring saved response values use sequential step prefixes (e.g., `1-authToken`, `2-userId`) to prevent leakage.
- **Round-Trip SLA Assertions**: Assertions for latency budgets (`timing.duration <= 250ms`) directly inside steps.
- **Parallel Scenario Execution**: Added `--parallel <N>` CLI flag for concurrent test execution.
- **Dry-Run Mode**: Added `--dry-run` flag to preview scenario flow, variables, and assertions offline.
- **Tag Filtering**: Added `--tag` CLI flags with multi-tag AND evaluations.
- **Online Interactive Playground**: Added hosted interactive dashboard with live flow visualization and cURL translation on GitHub Pages.
- **Expanded MCP Server Tools**: Exposed `gherkio init`, `gherkio validate`, and `gherkio convert` as natively discoverable tools to LLM Clients.

### Fixed
- Fixed playground asset bundling path mismatch inside the mdBook build target.
- Corrected various CLI documentation manual page auto-generation schemas.

---

## [0.1.0-alpha] - 2026-05-15

### Added
- **Core Declarative Engine**: YAML runner for sequential HTTP requests, responses, headers, and parameter mapping.
- **Value Matchers & Negative Assertions**: Core validations (`exists`, `equals`, `contains`, `matches`, `not exists`, `schema: not <name>`).
- **Setup & Teardown Blocks**: Support for scenario lifecycle isolation with global setup and robust error-tolerant teardown blocks.
- **Retry & Polling Loops**: Configurable attempts, interval timers, and linear backoff algorithms for eventual consistency testing.
- **Multi-Account Credentials Management**: Namespaced credentials mapping using `$accounts.<name>.<field>` variables with secure external YAML loaders.
- **Static Schema Linter**: Added `gherkio validate` and `gherkio schema` for IDE integration and workspace lint checks.
- **Model Context Protocol (MCP) Server**: Dynamic stdio server mapping workspace tests, environments, credentials, schemas, and live execution triggers directly to AI Clients.
