# RFC-9: Setup and Teardown Steps

> **Status:** Implemented
> **Author:** Faris
> **Date:** May 22, 2026

---

## 1. Summary

Add `setup` and `teardown` blocks to the scenario model, enabling pre-condition and post-condition steps that run before/after the main steps — regardless of whether the main steps pass or fail.

---

## 2. Motivation

Many integration scenarios require:

- **Setup**: Creating test data (a user, an order, a product) before the test scenario runs
- **Teardown**: Cleaning up test data after the test completes, even if it fails

Currently, users handle this by:

1. Adding setup as regular steps at the top of the scenario — but these steps count toward pass/fail statistics, making them indistinguishable from test logic
2. Using `use:` to import a reusable setup scenario — this works but mixes setup concerns with test logic
3. Manually cleaning up resources via separate scripts outside Gherkio — defeats the purpose of the DSL

The PRD (§9.2, §13.2, §13.3) lists `setup` and `teardown` as first-class step types. This RFC implements them.

---

## 3. Design

### 3.1 Model Change

Add `Setup` and `Teardown` fields to `TestFile` in `internal/model/test.go`:

```go
type TestFile struct {
	Scenario string `yaml:"scenario"`
	Setup    []Step `yaml:"setup,omitempty"`    // ← new
	Steps    []Step `yaml:"steps"`
	Teardown []Step `yaml:"teardown,omitempty"` // ← new
}
```

### 3.2 DSL Syntax

```yaml
scenario: create and verify order

setup:
  - request:
      method: POST
      url: /users
      body:
        name: test-user-$timestamp
    save:
      userId: body.id

steps:
  - request:
      method: POST
      url: /orders
      body:
        userId: $userId
    expect:
      status: 201
      body.id: exists
    save:
      orderId: body.id

teardown:
  - request:
      method: DELETE
      url: /orders/$orderId
  - request:
      method: DELETE
      url: /users/$userId
```

### 3.3 Execution Semantics

```
execute setup steps    (always, before steps)
  ↓
execute main steps     (always)
  ↓
execute teardown steps (always, after steps — even if setup or main steps fail)
```

- **Setup failures** — If any setup step fails, the main steps are skipped (execution stops), but teardown still runs.
- **Main step failures** — Teardown always runs, regardless of main step pass/fail.
- **Teardown failures** — Failures in teardown steps are **recorded but do not affect the overall pass/fail** of the scenario. Rationale: failing a test because cleanup broke is misleading — the test itself may have passed, and the cleanup issue is a separate concern.

### 3.4 Step Result Markers

Setup and teardown steps are marked in `StepResult` with a role field so the printer and reports can distinguish them:

```go
type StepResult struct {
	// ... existing fields ...
	Role string `json:"role,omitempty"` // "setup", "steps", "teardown"
}
```

### 3.5 Output

The printer shows setup and teardown sections visually:

```
✓ create and verify order

  ── Setup ──

  1. POST /users
     ✓ success

  ── Steps ──

  2. POST /orders
     ✓ status = 201
     ✓ body.id = exists

  ── Teardown ──

  3. DELETE /orders/42
     ✓ success
```

### 3.6 Reporting

In HTML/JSON reports, setup and teardown steps appear in the same step list but carry a `Role` field. The HTML template can optionally group them under sub-headers.

### 3.7 Setup/Teardown with `use:`

`use:` steps are allowed inside `setup` and `teardown` blocks. This enables reusable setup scenarios:

```yaml
setup:
  - use: create-test-user.yaml
```

---

## 4. Edge Cases

### 4.1 Teardown runs even on failure

This is the key design choice. If a user creates a resource in setup, the teardown must delete it even if the middle steps fail. Otherwise, orphaned resources accumulate.

### 4.2 Teardown failures don't fail the scenario

Rationale: Teardown is cleanup, not test logic. A user who wants to assert cleanup behavior can add explicit steps in the main `steps:` block.

### 4.3 Variables from setup are available in steps and teardown

Variables extracted during setup (`save:`) are available for interpolation in both main steps and teardown steps. Teardown steps can reference variables created during setup (e.g. `$orderId`, `$userId`).

### 4.4 Teardown runs even if setup fails

If setup fails mid-way, some variables may be missing. Teardown steps referencing missing variables will fail interpolation — those errors are silently skipped (the step is marked as an error but doesn't affect the scenario pass/fail).

---

## 5. Implementation Plan

- [ ] Add `Setup` and `Teardown` fields to `TestFile` in `internal/model/test.go`
- [ ] Add `Role` field to `StepResult` in `internal/runner/executor.go`
- [ ] In `Run()` in `runner.go`, execute setup → steps → teardown in sequence
- [ ] Ensure teardown runs even if setup or steps fail (use `defer` or explicit recover)
- [ ] Prevent teardown failures from affecting `TotalPass`/`TotalFail` counts
- [ ] Update `PrintResult()` in `printer.go` to show section headers
- [ ] Update `MapResultToReportData()` in `report/html.go` to pass through `Role`
- [ ] Update HTML template to optionally show section badges

---

## 6. Dependencies

None. All stdlib.

---

## 7. Decisions

### 7.1 Setup/teardown as step arrays on TestFile, not separate YAML keys

Setup and teardown are `[]Step` arrays inside the scenario file, not separate files or top-level YAM L keys. This keeps them close to the scenario they belong to, making the file self-contained.

### 7.2 Teardown failures don't fail the test

Choosing pragmatism over purity. A test that passed its assertions but "fails" because cleanup threw an error is confusing in CI. The teardown error is logged but doesn't flip the scenario to failed.

### 7.3 Shared setup (`use:` inside setup) is allowed

`use:` inside setup/teardown reuses the existing composition engine. This enables patterns like:

```yaml
setup:
  - use: seed-database.yaml
teardown:
  - use: cleanup-database.yaml
```
