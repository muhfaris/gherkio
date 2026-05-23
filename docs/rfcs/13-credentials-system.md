# Credentials System

## Overview

The credentials system allows running the same test scenarios against multiple accounts without duplicating test files or environment files. A single test file runs once per account, with account-specific variables injected automatically.

## Directory Structure

```
.gherkio/
├── environments/
│   ├── local.yaml
│   └── staging.yaml
├── credentials/                  # ← new
│   └── staging.yaml              # credentials for staging environment
├── tests/
│   └── ...
└── ...
```

## Credentials File Format

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
```

## CLI Usage

```bash
# Run tests with a specific account
gherkio run tests/login.yaml --env staging --account alpha

# Run tests against ALL accounts in the credentials file
gherkio run tests/login.yaml --env staging --all-accounts

# Run all tests in a directory, each test against all accounts
gherkio run tests/ --env staging --all-accounts

# Run all tests (auto-uses single account if only one exists)
gherkio run --env staging
```

## Variable Injection

Account credentials are injected as variables before step execution:

```yaml
# tests/login.yaml
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
```

When run with `--account alpha`:
- `$username` = `alpha@test.com`
- `$password` = `alpha-secret`

These variables behave like `save:` variables — available for interpolation in URLs, headers, and body, and pass through `use:` steps.

## Output Format

### Single account with account suffix:
```
✓ staging login (alpha)

1. POST /auth/login
   ✓ status = 200
   ✓ body.accessToken exists

✓ PASS (staging login (alpha))
```

### Multi-account summary:
```
Running account: alpha (1/3)
✓ staging login (alpha)
  ...
✓ PASS (staging login (alpha))

Running account: beta (2/3)
✗ staging login (beta)
  ...
✗ FAIL (staging login (beta))

════════════════════════════════════════
✗ FAIL — across 3 account(s)
15 passed, 5 failed, 20 total assertions
```

## Edge Cases

### No credentials file
If `--env staging` is used but no `credentials/staging.yaml` exists, the command behaves exactly as today — no accounts, no change.

### Single account with no flag
If credentials file has exactly 1 account and neither `--account` nor `--all-accounts` is given, the single account is automatically used.

### Multiple accounts with no flag
If credentials file has multiple accounts but no flag is given:
```
⚠ 3 accounts found in credentials. Use --account <name> or --all-accounts to use them.
```
Tests run without credential injection.

### Account specified but file missing
```
✗ account "alpha" not found in .gherkio/credentials/staging.yaml.
  Available accounts: beta, gamma, delta
```

## Sensitive Field Masking

Password fields are automatically added to the sensitive mask when credentials are loaded. This prevents passwords from appearing in console output or reports.

## Implementation Details

- Credentials are loaded from `.gherkio/credentials/<env>.yaml`
- Variables are injected before step execution (can be overridden by `save:`)
- Tests run sequentially for each account (no parallelization to avoid rate limiting)
- Account name is included in `RunResult.Account` and `ScenarioData.Account` for reporting