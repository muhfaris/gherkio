# RFC-1: Scenario Composition (Shared/Reusable Tests)

> **Status:** Completed
> **Author:** Faris
> **Date:** May 21, 2026

---

## 1. Summary

Allow a Gherkio scenario to **use/import** another scenario as a step. This enables reusable authentication flows, shared setup/teardown, and test composition without duplicating steps across files.

---

## 2. Motivation

Currently each test file is standalone. Common flows (e.g. login, token refresh, data seeding) must be copy-pasted across every scenario. This violates the DRY principle and makes maintenance expensive.

Real-world example:
- 10 test files all need an admin login step
- If the login endpoint or payload changes, all 10 must be updated
- A `use: login-as-admin` step would solve this in one place

---

## 3. Design

### 3.1 The `use` Step Type

A new step type that references another scenario file by name:

```yaml
steps:
  - use: login-as-admin
```

The file is resolved relative to `.gherkio/tests/` (with optional `.yaml` extension).

### 3.2 Resolution Order (Flexible)

No enforced folder convention. You organize tests however you want.

Resolution when `use: <path>` is specified:

```
1. Relative to the importing file's directory
2. Relative to .gherkio/tests/
```

Examples:

```yaml
use: login-as-admin            # .gherkio/tests/login-asadmin.yaml or same dir
use: auth/login-as-admin       # .gherkio/tests/auth/login-as-admin.yaml
use: shared/login-as-admin     # .gherkio/tests/shared/login-as-admin.yaml
use: ../common/login           # relative path from importing file
```

You control the folder structure. No convention to learn.

### 3.3 Variable Flow

Variables saved in a `use`d scenario are **merged into the importing scenario's variable context**. This is the critical integration point.

```yaml
# .gherkio/tests/auth/login-as-admin.yaml
steps:
  - request:
      method: POST
      url: /auth/login
      body:
        email: admin@test.com
        password: secret

    expect:
      status: 200
      body.token: exists

    save:
      token: body.token         # ← exported to importing context

# .gherkio/tests/items/update-item.yaml
steps:
  - use: auth/login-as-admin     # ← resolves to .gherkio/tests/auth/login-as-admin.yaml

  - request:
      method: POST
      url: /items
      headers:
        Authorization: Bearer $token   # ← used here
```

### 3.4 Recursive Composition

A `use`d scenario can itself contain `use` steps. The engine must handle this with:

- **Depth-first execution** — nested `use`s are resolved before the parent continues
- **Circular detection** — detect and error on circular references (A uses B uses A)
- **Max depth limit** — configurable limit (default: 5) to prevent infinite recursion bugs

### 3.5 Multi-level Composition

```yaml
# .gherkio/tests/auth/login-as-admin.yaml
steps:
  - use: secrets/get-api-keys    # nested composition, resolves to .gherkio/tests/secrets/get-api-keys.yaml
  - request: ...
    save:
      token: body.token

# .gherkio/tests/items/update-item.yaml
steps:
  - use: auth/login-as-admin     # resolves to .gherkio/tests/auth/login-as-admin.yaml
  - request: ...                 # has access to both get-api-keys AND login-as-admin variables
```

---

## 4. Edge Cases

### 4.1 Variable Shadowing

If both the parent and the `use`d scenario save a variable with the same name, the **last write wins** (sequential). This is consistent with how variables already work within a single scenario.

```yaml
# shared: saves token = "abc"
# parent: saves token = "xyz"
# result: token = "xyz"
```

### 4.2 Circular References

```yaml
# a.yaml: uses b
# b.yaml: uses a
# → Error: circular reference detected: a → b → a
```

Detected via a **visited set** during resolution.

### 4.3 Assertions in Used Scenarios

If a `use`d scenario's assertions fail, the entire step fails. The failure output should indicate which `use`d scenario and file produced the failure, so users can trace the error source.

### 4.4 Timing in Used Scenarios

If a `use`d scenario has its own `timing` configuration, it's evaluated within that scenario's scope. The parent sees the total duration of the `use` step (including all nested execution).

### 4.5 Environment Isolation

A `use`d scenario executes within the **same environment** as the parent. It does NOT reload the environment file — the parent's `env` applies.

---

## 5. Implementation Plan

### Phase 1 — Basic `use` (minimal viable)

- [ ] Add `Use string` field to `Step` model (`internal/model/test.go`)
- [ ] In `runner.Run()`, detect `use` steps and load target scenario
- [ ] Execute loaded scenario steps in-place
- [ ] Merge variables back into parent context
- [ ] Flatten error/assertion reporting

### Phase 2 — Resolution & Safety

- [ ] File resolution logic (relative to importing file → relative to `.gherkio/tests/`)
- [ ] Circular reference detection (visited set)
- [ ] Max depth guard (default: 5)
- [ ] Clear error messages for resolution failures

### Phase 3 — Reporting

- [ ] Show `use` steps in output: `3. └─ use: login-as-admin`
- [ ] Indent nested assertions under the `use` step
- [ ] Show source file path for `use`d scenario failures

---

## 6. Example Output

Consistent with current Gherkio output format:

```
✓ update item

1. └─ use: auth/login-as-admin
   │
   ├ POST https://api.example.com/auth/login
   │  ✓ success
   │  ✓ status = 200
   │  ✓ body.token exists
   │

2. POST https://api.example.com/items
   ✓ success
   ✓ status = 201
   ✓ body.id exists

3. PUT https://api.example.com/items/42
   ✓ success
   ✓ status = 200
   ✓ body.name = Laptop Baru

✓ PASS
6 passed, 0 failed, 6 total
Duration: 1.2s
```

Failure case (nested failure inside use):

```
✗ update item

1. └─ use: auth/login-as-admin
   │
   ├ POST https://api.example.com/auth/login
   │  ✗ failed
   │  ✓ status = 200
   │  ✗ body.token exists
   │     └─ path not found
   │

2. POST https://api.example.com/items
   ✓ success
   ✓ status = 201
   ✓ body.id exists

✗ FAIL
5 passed, 1 failed, 6 total
Duration: 1.2s
```

---

## 7. Open Questions

1. **Should `use`d scenarios support parameterization?** e.g. `use: login-as { role: admin }` — or keep it simple for now?
2. **Should assertions from `use`d scenarios count toward the parent's pass/fail total?** (Proposal: yes, they contribute)
3. **Should `teardown` steps be supported?** A scenario could declare teardown steps that run even if the parent fails?

---

## 8. Alternatives Considered

### 8.1 Copy-paste

Rejected. Violates DRY, high maintenance cost.

### 8.2 Helper scripts / Plugins

Rejected for MVP. The `use` approach keeps composition within the DSL — declarative, readable, no imperative code. Plugin system is a separate future concern.

### 8.3 YAML Anchors & Aliases

Rejected. YAML anchors (`&login`, `*login`) are a YAML-level feature, not a Gherkio-level one. They require all scenarios to be in the same file and don't support variable merging.
