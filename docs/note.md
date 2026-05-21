## Step 1 Complete — Core Runner

### New files created
```
cmd/
├── root.go          # (existing) - added nothing
├── init.go          # (existing) - unchanged
└── run.go           # NEW - gherkio run <test-file>
internal/
├── model/
│   ├── test.go           # NEW - Test YAML structs + custom unmarshaler
│   └── environment.go    # NEW - Environment YAML structs
└── runner/
    ├── runner.go     # NEW - Orchestrator: load files, execute, assertions
    ├── executor.go   # NEW - HTTP execution, JWT decoding, path resolution
    └── printer.go    # NEW - Console output formatting
```

### What works now

| Feature | Status |
|---------|--------|
| `gherkio init` | ✅ (existing) |
| `gherkio run <file>` | ✅ |
| `--env` flag (default: `local`) | ✅ |
| HTTP request execution | ✅ |
| Status code assertions | ✅ |
| Field existence assertions (`response.X: exists`) | ✅ |
| Equality assertions (`response.title: test post`) | ✅ |
| JWT assertions (`jwt.role: admin`) | ✅ (auto-decodes JWT from `response.token` or `response.access_token`) |
| Variable extraction (`save`) | ✅ |
| Multi-step scenarios | ✅ |
| Service-based URL resolution | ✅ |
| Failure output with diff | ✅ |
| Exit code 1 on failure | ✅ |
| Relative test path (from `.gherkio/tests/`) | ✅ |

### Usage examples
```bash
# Run with default "local" env
gherkio run tests/login.yaml

# Run with staging env
gherkio run tests/login.yaml --env staging

# Run with relative path (resolved under .gherkio/tests/)
gherkio run example/sample1.yaml --env staging
```

### Step 2 Complete — Variable Interpolation

You can now use variables in subsequent requests using the `$var` or `${var}` syntax. Variables saved from responses can be used in:
- Request URLs
- Request headers
- Request bodies

Example:

```yaml
steps:
  - request:
      method: POST
      url: /auth/login
      body:
        email: admin@test.com
        password: secret

    expect:
      status: 200
      response.token: exists

    save:
      token: response.token

  - request:
      method: GET
      url: /api/user
      headers:
        Authorization: "Bearer $token"

    expect:
      status: 200
```

### Step 3b — Path Syntax Standardization

**Problem:** The path syntax was ambiguous — `response.foo`, `response.body.foo`, and `body.foo` all meant different things to users but were inconsistently handled.

**Solution:** Standardized to explicit, non-ambiguous path prefixes:

| Prefix | Context | What it resolves to |
|--------|---------|-------------------|
| `body.X` | expect / save | Response JSON body field **(canonical)** |
| `headers.X` | expect / save | Response header field |
| `status` | expect only | HTTP status code (already handled via `expect.status`) |
| `jwt.X` | expect / save | Decoded JWT claim |

**Changes made:**
- `extractValues()` in `executor.go` — added proper `body.X` and `response.body.X` support with correct prefix ordering
- `evaluateAssertion()` in `executor.go` — added direct `headers.X` support, removed dead `response.body.` code path
- `runner.go` — JWT auto-decoding now also checks `accessToken` (camelCase) for DummyJSON compatibility
- All example/test YAML files updated to use canonical `body.X` syntax
- `model/test.go` comment updated

**Backward compatibility preserved:** `response.X`, `response.body.X`, `response.headers.X`, and bare field paths still work.

### Step 3c — Timing Assertions

**New feature:** Steps can now specify a maximum duration.

```yaml
- request:
    method: POST
    url: /auth/login
    ...

  timing:
    max: 500ms
```

**Changes made:**
- `internal/model/test.go` — Added `TimingConfig` struct with `Max` field, added `Timing` to `Step`
- `internal/runner/executor.go` — Added `evaluateTiming()` function that parses duration and compares against actual
- `internal/runner/runner.go` — Evaluates timing after step execution and appends result to assertions
- `internal/runner/printer.go` — Timing assertions always show actual duration inline (e.g. `timing.max = 500ms (actual: 312ms)`)

**Supported formats:** Go duration strings — `500ms`, `1s`, `250ms`, `2.5s`, `1m`

### Step 3 Complete — Contextual Failure UX

**Changes:**
- Added `Suggestions []string` field to `AssertionResult` in `executor.go`
- Added `getAvailableFields()` helper that extracts top-level keys from parsed JSON
- All "not found" assertion paths now populate suggestions with available fields
- Printer shows "path not found" instead of "got: (not found)"
- Printer lists available fields when path resolution fails
- Printer auto-displays response status + body when any assertion fails in a step

**What it looks like:**
```
✗ response.accessToken exists
    └─ path not found

Available fields:
  - message
  - statusCode

Response:
Status: 400

Body:
{
  "message": "Invalid credentials"
}
```

### Ready for Step 4?
**Step 2** would be **Variable Interpolation** — so you can use `$token` in subsequent request headers/body. Want me to continue, or would you like to adjust anything in Step 1 first?


