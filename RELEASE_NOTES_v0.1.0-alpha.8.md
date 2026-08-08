# Gherkio v0.1.0-alpha.8 Release Notes

> **Release date**: 2026-08-08

## Features

### 🧠 MCP Payload-Aware Scenario Authoring
The MCP server now empowers an AI agent to collaborate with a QA to define
what to test, rather than mechanically mapping a single endpoint:

- Reworked the `plan-scenario` prompt as a payload-variant / CRUD-aware planner.
  A standalone endpoint can yield many test variants (happy path, optional
  fields, per-field negatives, business rules); the agent enumerates a variant
  matrix and proposes one file per variant (e.g. `tests/users/01_create_ok.yaml`).
- The agent always asks the QA for payloads, response shapes, and business
  logic instead of inventing them — Gherkio provides no API of its own.
- Added a `validate_flow` prompt that reviews a proposed multi-variant plan for
  variable flow, setup/teardown balance, assertion coverage, and dependency
  resolution before anything is authored or run.
- Updated the `create_test` tool description to reference the new plan →
  validate_flow → confirm → author → dry-run flow.

### 🐛 Bug Fixes
- Fixed `run_test` step isolation so `step: 0` (the first, 0-indexed step) can
  be targeted explicitly instead of being mistaken for "no step argument",
  matching the CLI runner's `-1` default semantics.
- Corrected the "Execution execution failed" message to "Execution failed".

---

**Full Changelog**: https://github.com/muhfaris/gherkio/compare/v0.1.0-alpha.7...v0.1.0-alpha.8
