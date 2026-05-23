# RFC-12: Multi-Account Test Execution (Credentials System)

> **Status:** Implemented
> **Author:** Faris
> **Date:** May 22, 2026

---

## 1. Summary

Add a credentials system that allows running the same test scenarios against multiple accounts without duplicating test files or environment files. A single test file runs once per account, with account-specific variables injected automatically.

---

## 2. Motivation

Users often need to test the same API against different accounts:

- **10 tenant accounts** on staging — verify all tenants can log in and perform basic operations
- **Admin vs regular user** — confirm different permission sets work correctly
- **Data isolation testing** — ensure account A cannot access account B's data

Currently, users must either:

1. **Create one test file per account** — 10 accounts = 10 nearly identical files
2. **Create one environment file per account** — pollutes the environment directory with account credentials
3. **Use shell scripts to inject variables** — defeats the purpose of the DSL

All three approaches violate the DRY principle and make maintenance expensive.

---

## 3. Design

### 3.1 Credentials File

A new credentials file lives alongside the environment files:

```
.gherkio/
├── environments/
│   ├── local.yaml
│   └── staging.yaml
└── credentials/                  # ← new
    └── staging.yaml              # credentials for staging environment
```

Each credentials file maps account names to variable sets:

```yaml
# .gherkio/credentials/staging.yaml
accounts:
  alpha:
    username: alpha@test.com
    password: alpha-secret
    role: admin

  beta:
    username: beta@test.com
    password: beta-secret
    role: viewer

  gamma:
    username: gamma@test.com
    password: gamma-secret
    role: editor

  # ... up to 10 or more accounts
```

Credentials are loaded only when the matching environment is active. `gherkio run --env staging` loads `credentials/staging.yaml`.

### 3.2 CLI Interface

```bash
# Run tests with a specific account
gherkio run tests/login.yaml --env staging --account alpha

# Run tests against ALL accounts in the credentials file
gherkio run tests/login.yaml --env staging --all-accounts

# Run all tests in a directory, each test against all accounts
gherkio run accounts/ --env staging --all-accounts
```

### 3.3 Variable Injection

Account credentials are injected as variables **before** step execution, just like environment variables:

```yaml
# tests/staging-login.yaml
steps:
  - request:
      method: POST
      url: /auth/login
      body:
        username: $username
        password: $password

    expect:
      status: 200
      body.accessToken: exists

    save:
      token: body.accessToken

  - use: workflows/order-creation.yaml    # downstream steps use $token
```

When run with `--account alpha`:
- `$username` = `alpha@test.com`
- `$password` = `alpha-secret`
- `$role` = `admin`

These variables behave exactly like `save:d` variables — they're available for interpolation in URLs, headers, and body, and they pass through `use:` steps.

### 3.4 Multi-Account Output

When running with `--all-accounts`, each account produces a separate result group:

```
Running account: alpha

✓ staging login (alpha)
  1. POST /auth/login
     ✓ status = 200
     ✓ body.accessToken exists

✓ PASS

Running account: beta

✗ staging login (beta)
  1. POST /auth/login
     ✗ status = 401
         └─ got: 403

Response:
Status: 403

Body:
{
  "error": "Account locked"
}

✗ FAIL

════════════════════════════════════════
✗ FAIL — 1 passed, 1 failed across 2 accounts
```

The scenario name is suffixed with the account name to differentiate results in reports:

```
✗ staging login (alpha)
✗ staging login (beta)
```

### 3.5 Report Integration

In HTML/JSON reports, each account-scenario pair appears as a separate `ScenarioData` entry. The account name is included in metadata:

```go
type ScenarioData struct {
    Name          string
    TestFile      string
    Account       string   // ← new: "alpha", "beta", etc.
    TotalDuration string
    // ... existing fields
}
```

---

## 4. Edge Cases

### 4.1 No credentials file for the selected environment

If `--env staging` is used but no `credentials/staging.yaml` exists, the command behaves exactly as today — no accounts, no change. Credentials are purely additive.

### 4.2 Credentials file exists but no account flag specified

If credentials exist for the active environment but neither `--account` nor `--all-accounts` is given:

- **1 account** → auto-use it (no need to type `--account` every time)
- **2+ accounts** → print a hint and run without credentials:

```
⚠ 3 accounts found in credentials/staging.yaml. Use --account <name> or --all-accounts to use them.
```

This keeps the zero-config experience clean for the common case (one account per environment) while being explicit when ambiguity exists.

### 4.3 `--account` without `--env`

If `--account alpha` is used without `--env`, Gherkio uses the default environment (e.g. `local`) and looks for `credentials/local.yaml`. This is consistent with how `--env` defaults work.

### 4.4 `--account` specified but credentials file missing

If `--account alpha` is specified but no credentials file exists for the active environment, the command fails with a clear error:

```
✗ No credentials file found for environment "staging" at .gherkio/credentials/staging.yaml.
  Create one or remove the --account flag.
```

If the credentials file exists but the named account doesn't exist in it:

```
✗ Account "alpha" not found in .gherkio/credentials/staging.yaml.
  Available accounts: beta, gamma, delta
```

### 4.4 Variable conflicts

If a credential variable (`$username`) conflicts with a variable saved from a previous step (`save:`), the saved value wins. Credential variables are injected first and can be overridden by step execution.

### 4.5 Sensitive field masking

Account credentials (especially `password`) should be automatically added to the sensitive fields mask when credentials are loaded. This prevents passwords from appearing in console output or reports.

---

## 5. Implementation Plan

### Phase 1 — Credentials Loader

- [ ] Create `.gherkio/credentials/` directory in init scaffolding
- [ ] Add `Credentials` model struct to `internal/model/`
- [ ] Add `LoadCredentials(projectDir, envName)` function to load credentials YAML
- [ ] Add `--account` and `--all-accounts` flags to `gherkio run`

### Phase 2 — Variable Injection

- [ ] In `Run()`, inject credential variables into the vars map before step execution
- [ ] When `--all-accounts`, iterate over all accounts in the credentials file
- [ ] For each account, run the full test suite with injected variables
- [ ] Collect results per account and display grouped output

### Phase 3 — Reporting

- [ ] Add `Account` field to `ScenarioData`
- [ ] Update HTML/JSON report templates to show account names
- [ ] Show account-per-account results in the summary

---

## 6. Example Output

### Single account:

```bash
$ gherkio run tests/workflows/ --env staging --account alpha

✓ staging login (alpha)

1. POST /auth/login
   ✓ status = 200
   ✓ body.accessToken exists
...
✓ PASS
```

### All accounts:

```bash
$ gherkio run tests/workflows/ --env staging --all-accounts

Running account: alpha (1/3)
✓ staging login (alpha) — 5 passed, 0 failed

Running account: beta (2/3)
✗ staging login (beta) — 3 passed, 2 failed

Running account: gamma (3/3)
✓ staging login (gamma) — 5 passed, 0 failed

════════════════════════════════════════
✗ FAIL — 13 passed, 2 failed across 3 accounts
```

---

## 7. Decisions

### 7.1 Separate credentials file, not inline in environment

Keeping credentials in a separate `credentials/` directory keeps the environment files infrastructure-only. This also makes it easier to `.gitignore` credentials while keeping environment files version-controlled.

### 7.2 Account credentials as variables, not overrides

Account credentials are injected as variables (`$username`, `$password`) rather than as config overrides. This keeps the model simple — no new YAML fields, no new structs in the test file parser.

### 7.3 `--all-accounts` executes sequentially

Each account runs sequentially to avoid rate limiting and resource contention. Parallel multi-account execution is a future optimization.

### 7.4 No credential inheritance

Each account is fully self-contained in its YAML entry. There is no "extends: alpha" pattern. This keeps credentials explicit and auditable.

---

## 8. Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Credentials file committed to git by accident | Default path `.gherkio/credentials/` should be added to `.gitignore`; init template includes a `.gitkeep` and a warning comment |
| Password shown in console output | Credential variables auto-added to sensitive field mask |
| Many accounts = slow test runs | Sequential execution is explicit; users choose which accounts to run with `--account` |
