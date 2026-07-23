# Gherkio v0.1.0-alpha.7 Release Notes

> **Release date**: 2026-07-23

## Features

### 🎯 Collection & Array Assertion Matchers
- Added dedicated `04_collection_matchers` showcase test demonstrating array/collection assertion capabilities.
- Extended the matchers engine with new collection-level assertion logic for validating arrays, item counts, and element-wise checks.
- Updated runner executor, interpolator, and matchers (with comprehensive `matchers_test` coverage) to support the new collection assertion pipeline.

### 🔄 Converter Improvements
- Refactored the cURL-to-DSL and DSL-to-cURL converters (`curl.go`, `dsl.go`, `parser.go`) for full compatibility with collection-style steps and complex nested payloads.

### 🧪 Standardized Showcase Test Suite
- Standardised the entire showcase test suite and integrated it into the CI/CD pipeline for consistent regression coverage.
- Added a third-party Stripe mock step to the showcase suite for realistic payment-flow testing.

### 🛠️ MCP & Documentation Updates
- Updated MCP prompts, resources, and server internals to expose new DSL features and collection matchers.
- Extended the test model and report types (`internal/model/test.go`, `internal/report/types.go`, `internal/report/html.go`) to accommodate collection assertion results in output and HTML reports.
- Updated assertion and matchers reference documentation.

### 🧹 Housekeeping
- Cleaned up stale `graphify-out/` artifacts.
- Session files added for `local-alpha` and `mocked` environments.

---

**Full Changelog**: https://github.com/muhfaris/gherkio/compare/v0.1.0-alpha.6...v0.1.0-alpha.7
