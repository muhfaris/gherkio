# Gherkio v0.1.0-alpha.6 Release Notes

> **Release date**: 2026-07-15

## Features

### 🚀 Advanced Testing & Traceability Features
- **Feature A: Data-Driven Testing (Parametrization)**: Support for scenario-level iteration via the `examples` block, allowing scenarios to run repeatedly using a set of variables.
- **Feature B: Composed Traceability**: Visual debug groupings, dynamic nesting indentation based on step depth, and variable state snapshots in the generated HTML reports.
- **Feature C: Outbound Service Mocking / Virtualization**: Define mock request/response pairs directly inside environment configurations to stub third-party APIs with support for variable interpolation.
- **Feature D: JSONPath Assertions**: Standard JSONPath support (`$.`) for evaluating complex payload assertions.

### 🔄 Step Locator & Reverse Conversion Enhancements
- **Session Variables in Reverse Conversion**: Integrated session variable loading when converting Gherkio steps back into cURL commands.
- **Robust Step Parsing**: Improved `ScanSteps` to correctly parse nested list structures (such as lists in request bodies or retry parameters) without misidentifying step indexes.
- **Absolute URL Fix**: Prevented `resolveURL` from prepending the environment base URL to absolute URLs (starting with `http://` or `https://`).

### 📚 Rich Schema Hover Documentation
- Enhanced auto-generated JSON schemas with comprehensive hover descriptions, listing available fields, options, and matchers for:
  - `expect` and `timing` blocks
  - `retry` configuration
  - `setup`, `steps`, and `teardown` blocks

### ➕ Declarative Variable Assignment with `set:`
- Added a new declarative `set:` step style allowing direct, inline variable assignments within test scenarios.

### 🐛 General Bug Fixes
- Fixed multipart form-data display output.
- Corrected `json.Number` serialization behaviors.
- Resolved edge cases with `all()` combined with `not exists` assertions.
- Improved runner error output formatting.

### 🛠️ MCP & Session Enhancements
- Added MCP prompt capabilities.
- Improved runner session persistence and stderr progress indication.
- Added support for custom JWT paths.

### 🔢 Bracket Notation & Retry Interpolation
- Supported array-index bracket notation for accessing variables (e.g., `users[0]`).
- Enabled dynamic variable re-interpolation on every retry attempt.

### 🌍 Environment Context Command
- Added the new `env context` command to output context details alongside auto-selection hints.

---

**Full Changelog**: https://github.com/muhfaris/gherkio/compare/v0.1.0-alpha.5...v0.1.0-alpha.6
