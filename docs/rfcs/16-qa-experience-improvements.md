# RFC-16: QA Experience Improvements

**Status:** Draft
**Date:** 2026-05-24
**Author:** Faris
**PRD References:** $4.1.C (Observability), $6.2 (QA Automation), $15.3 (Failure UX), $16.3 (Parallel Execution), $24.1 (CLI Features)

---

## 1. Summary

Feedback from a QA engineer perspective on using Gherkio for API integration testing. Identifies five gaps in the current workflow and proposes solutions aligned with the PRD's core philosophy.

---

## 2. Motivation

The Gherkio execution engine is solid — the DSL is declarative, assertions are expressive, and failure diagnostics are already better than most tools. However, a QA engineer writing and debugging tests day-to-day would encounter several friction points:

1. No way to validate/lint a test before running it
2. No way to preview resolved variable values when debugging failures
3. No way to preview what a test will do without executing HTTP calls
4. No way to organize or filter tests as the suite grows
5. No parallel execution for large test suites

Each of these is explicitly called out in the PRD as either a core goal or a planned feature.

---

## 3. Proposals

### 3.1 `gherkio validate` — Pre-run Lint Command

**PRD Alignment:** $4.1.C (Strong Observability — failure diagnostics), $20 (IDE Experience — inline diagnostics)

**Problem:** QA writes a YAML file, runs it, and it fails because of a typo in a path, a missing variable reference, or an invalid schema name. The feedback loop is slower than it could be.

**Solution:** Add a `gherkio validate` CLI command that performs static analysis without executing HTTP requests:

```bash
gherkio validate login.yaml          # Validate single file
gherkio validate                     # Validate all test files
gherkio validate --verbose           # Show structured results per file
```

**Checks performed:**

| Check | Example Error |
|-------|--------------|
| YAML syntax | `yaml: line 12: mapping keys are not allowed in this context` |
| Scenario structure | `missing required field: steps` |
| Request method validity | `invalid method: GETT` (typo) |
| Path syntax | `body..id: invalid path (double dot)` |
| Variable references | `undefined variable: $tokn` (typo, suggests `$token` if nearby) |
| Account references | `account "eka" not found in credentials` |
| Schema references | `schema "user" not found in .gherkio/schemas/` |
| `use` file existence | `referenced scenario not found: auth/login.yaml` |
| Retry config | `retry.backoff "fibonacci" invalid (valid: constant, linear, exponential)` |

**Implementation approach:** Reuse existing validation logic from `teststore.Validate()` and `runner` package functions. The `validate_test` MCP tool already exists — this is surfacing it as a CLI command.

**Non-goal:** This is NOT a full type-checker or static analysis engine. It catches structural issues, not semantic ones (e.g. it won't validate that a response body shape matches expectations before runtime).

---

### 3.2 Verbose Variable Resolution — `--verbose` Enhancement

**PRD Alignment:** $4.1.C (Strong Observability — execution traceability), $15.3 (Failure UX — context-rich errors), $6.2 (QA Automation Engineers — debugging visibility)

**Problem:** When a test fails because of a variable resolution issue (e.g. `$accounts.eka.username` resolves to wrong value), the QA has to mentally trace through the variable chain. There's no way to see what values were actually injected without reading the test YAML and guessing.

**Solution:** Include resolved variable state as part of the existing `--verbose` output. The expectation for `-v` is "show me everything" — variable resolution is internal state the user should see when they ask for verbose mode.

Example output with `--verbose`:

```
✓ User Login

── Resolved Variables ──
  $uuid                      → a1b2c3d4-e5f6-4789-abcd-ef1234567890
  $accounts.eka.username     → "eka_ali"
  $accounts.eka.password     → ***masked***
  $token                     → (undefined — available from: login.yaml:steps step 0 at line 14)

── Steps ──

1. POST https://dummyjson.com/auth/login
   ✓ success
   ✓ status = 200
   ✓ body.id exists

   ├ Request: POST https://dummyjson.com/auth/login
   ├ Headers: Content-Type: application/json
   ├ Body: {"username":"eka_ali","password":"***masked***"}
   │
   ├ Response: 200 OK
   ├ Body: {"id":1,"token":"eyJ..."}
   └ Duration: 312ms
```

**Design decisions:**

| Decision | Rationale |
|----------|-----------|
| Variables shown once per scenario (pre-execution) | Avoids repetition; user can see what values were available going in |
| Saved variables shown inline in step output as they become available | Gives dynamic traceability without cluttering the summary |
| Sensitive fields masked (`***masked***`) | Security — same masking logic as request/response bodies |
| Undefined variables show "undefined" + source hint | Helps QA trace where a variable should come from |
| Not a separate flag — part of `--verbose` | Users expect `-v` to show everything; a separate flag violates that convention |

**Implementation approach:** The runner already tracks `vars map[string]interface{}` through execution. The printer captures a snapshot of the variable map at the start of each scenario and after each save, then renders it in the verbose output tree alongside request/response payloads.

---

### 3.3 Tags System — Test Organization

**PRD Alignment:** $24.1.2 (CLI Features — filter tags), $6.2 (QA Automation Engineers — readable test scenarios)

**Problem:** After 20+ test files, the QA needs a way to organize and filter tests. Currently the only grouping is by directory.

**Solution:** Add an optional `tags` field to scenarios and a `--tag` CLI filter:

```yaml
scenario: User Login
tags: [auth, critical, smoke]
```

```bash
gherkio run --tag smoke                # Run all smoke tests
gherkio run --tag auth                 # Run all auth tests
gherkio run --tag "critical,smoke"     # AND logic: all critical + smoke
```

**Design decisions:**

- Tags are optional — existing tests without tags work unchanged
- Tags are simple strings — no key=value, no expressions
- Multiple `--tag` flags use AND logic (test must have ALL specified tags)
- A single tag with commas uses AND logic too
- Directory-based grouping still works independently

**Non-goal:** No tag inheritance, tag expressions, or tag-based variable injection. The PRD explicitly warns against complexity creep ($27.1).

---

### 3.4 `--parallel` — Concurrent Test Execution

**PRD Alignment:** $16.3 (Parallel Execution — "Supported in later phases"), $24.1.4 (CLI Features — parallel execution)

**Problem:** A suite of 30 tests at ~1.5s each takes 45 seconds sequentially. The PRD explicitly marks this as a future feature.

**Solution:** Add `--parallel N` flag to run tests concurrently:

```bash
gherkio run --parallel 4               # Run up to 4 tests at a time
gherkio run --parallel                 # Auto-detect CPU count
```

**Constraints (PRD-aligned):**

| Constraint | Rationale |
|-----------|-----------|
| **Per-scenario isolation only** — each test file runs in its own goroutine with an independent variable context | The PRD mandates isolated execution contexts ($16.2) |
| **No parallel steps within a scenario** — steps within one scenario remain sequential | Preserves determinism ($16.1) |
| **Environment sharing** — tests share the same environment config but get independent HTTP clients | Prevents client-state pollution |
| **Output interleaving** — output is buffered per scenario and printed as a group, not interleaved line-by-line | Preserves readability |
| **Report merging** — parallel results are merged into a single suite report | Required for CI integration |

**Failure behavior:** If one test fails, other tests continue. The exit code reflects the aggregate result.

**Non-goal:** Distributed execution across machines. This is explicitly out of scope ($26.2).

---

### 3.5 `--dry-run` — Preview Without Execution

**PRD Alignment:** $4.1.C (Strong Observability — execution traceability), $20 (IDE Experience — validation), $17 (Error Handling — explain failures clearly)

**Problem:** QA wants to verify a test is correctly structured before hitting real APIs. This is especially important when targeting production-like environments where accidental writes are costly. Currently there's no way to "preview" what a test will do without executing it.

**Solution:** Add `--dry-run` to `gherkio run` that parses the test, resolves variables, and prints the expanded requests — but never sends HTTP calls:

```bash
gherkio run login.yaml --dry-run
gherkio run --dry-run                    # Dry-run all tests
gherkio run login.yaml --dry-run -v      # Dry-run with verbose variable resolution
```

Example output:

```
── Dry Run: User Login ──

── Resolved Variables ──
  $uuid                     → a1b2c3d4-e5f6-4789-abcd-ef1234567890
  $accounts.eka.username    → "eka_ali"
  $accounts.eka.password    → ***masked***

── Validation ──
  ✓ YAML syntax
  ✓ scenario structure
  ✓ request methods
  ✓ variable references
  ✓ account references
  ✓ schema references

── Steps (not executed) ──

1. POST https://dummyjson.com/auth/login
     Content-Type: application/json
     Body: {"username":"eka_ali","password":"***masked***"}

2. GET https://dummyjson.com/auth/me
     Authorization: Bearer \$token
     ⚠  \$token would be resolved at runtime from step 1's response (save: body.accessToken)
```

**Design decisions:**

| Decision | Rationale |
|----------|-----------|
| Dry-run validates AND previews — not just validation | A raw "no errors" message is less useful than seeing the expanded requests |
| Shows step-by-step expanded requests with resolved variables | QA can spot incorrect URL interpolation, wrong header values, etc. before they cause runtime failures |
| Marks variables that can't be pre-resolved (e.g. `$token` from a previous step's response) | Sets expectations — QA knows which values are dynamic and can't be verified statically |
| Safe to run against any environment (including production) | Zero HTTP calls — impossible to accidentally mutate data |
| Compatible with `--verbose` | `--verbose` in dry-run mode shows more detail (e.g. full header map, resolved body structure) |

**Relationship to `gherkio validate`:** `--dry-run` is a superset of validation. It does everything `validate` does plus preview. Both are useful — `validate` for quick CI/linting checks, `--dry-run` for interactive debugging.

**Relationship to `--verbose`:** `--verbose` shows what *did* happen (runtime). `--dry-run` shows what *would* happen (pre-execution). They're complementary — a typical workflow: `--dry-run` → fix issues → `run` → if failure → `run -v` to debug.

**Implementation approach:** The runner already has all the parsing (YAML → `TestFile`), variable resolution (interpolator), and path resolution (URL building) logic. Dry-run would reuse the same code paths but skip `executeRequest()`. The printer already knows how to format expanded requests — it uses this in verbose mode for actual executions.

---

## 4. Effort Estimation

| Feature | Files to Change | Complexity | Notes |
|---------|---------------|-----------|-------|
| `gherkio validate` | `cmd/validate.go` (new), reuses `teststore.Validate` | Low | Thin CLI wrapper over existing validation |
| `--verbose` enh. (variable resolution) | `runner/printer.go`, `runner/runner.go` | Medium | Adds variable tree to verbose output section |
| `--dry-run` | `cmd/run.go`, `runner/runner.go` | Medium | Reuses parsing/interpolation, skips HTTP execution |
| Tags | `model/test.go` (new field), `cmd/run.go` (filter), `runner/runner.go` (filter logic) | Medium | New field, filter logic, directory scanner update |
| `--parallel` | `cmd/run.go`, `runner/runner.go` | High | Goroutine pool, buffered output, merged reporting |

---

## 5. Open Questions

1. **`gherkio validate`** — Should it also validate against the environment (e.g. check that services referenced in `request.service` exist in the environment file)? Or stay purely static?
2. **`--dry-run`** — Should unresolved variables (from future step saves) show as `⚠ unresolved` or be silently omitted from the preview?
3. **Tags** — Should tags be allowed on individual steps, or only at the scenario level?
4. **`--parallel`** — Should there be a `--parallel-timeout` to cap the total suite duration?

---

## 6. Out of Scope (for this RFC)

These were identified as QA friction points but are intentionally excluded:

| Item | Reason |
|------|--------|
| Response time range assertions (e.g. `timing.min`, `timing.< 500ms`) | Adds expression complexity to the DSL — risk of scope creep ($27.1). Better as a future capability ($14). |
| Watch mode (`gherkio run --watch`) | Nice-to-have but not in PRD. Post-MVP quality-of-life. |
| Flaky test detection | PRD mentions historical reporting ($15.5) as future. Would build on top of that. |

---

## 7. Summary

| Feature | PRD Backing | Effort | Priority |
|---------|-----------|--------|----------|
| `gherkio validate` | $4.1.C, $20 | Low | High — closes a daily-friction gap |
| `--verbose` enh. (variable resolution) | $4.1.C, $15.3, $6.2 | Medium | High — directly improves debugging UX |
| `--dry-run` | $4.1.C, $20, $17 | Medium | High — preview without side effects |
| Tags / `--tag` | $24.1 | Medium | Medium — needed as suite grows |
| `--parallel` | $16.3, $24.1 | High | Low — explicitly post-MVP |
