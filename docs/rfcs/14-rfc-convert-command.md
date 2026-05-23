# RFC-14: DX Tooling — Convert & Step Runner

**Status:** Draft  
**Author:** Faris  
**Date:** May 23, 2026  
**PRD Reference:** §7 (DSL Layer), §19 (Escape Hatch), §21 (AI Compatibility)  

---

## 1. Summary

Two CLI features designed to improve daily developer experience:

### Feature A: `gherkio convert` — cURL ↔ DSL Bridge

Convert between cURL commands and Gherkio DSL YAML in both directions.

```
cURL ──→ DSL     # Onboarding: paste cURL, get a working test step
DSL  ──→ cURL    # Sharing: copy a step as a cURL command for debugging/colleagues
```

### Feature B: `gherkio run --step` — Run Step Under Cursor

Execute a single step from a test file without running the entire scenario. Designed for fast iteration when editing YAML tests in Neovim (or any editor).

```
gherkio run login.yaml --step 2     # Run only step 2
gherkio run login.yaml --line 25    # Smart-detect which step contains line 25
gherkio run login.yaml --line 25 --setup-only   # Only run setup steps
```

---

## 2. Motivation

### 2.1 Problem: Convert

Gherkio's YAML DSL is declarative and structured, which is great for readability — but it's not how developers *naturally express HTTP requests* in the moment. When debugging an API, developers:

1. Open Chrome DevTools → copy as cURL
2. Paste into terminal to test
3. If it works, *then* manually translate to Gherkio YAML

Step 3 is mechanical, error-prone, and creates friction between "I have a working request" and "I have a test."

### 2.2 Problem: Step Runner

When iterating on a test (tweaking a request body, adjusting assertions, debugging a failure), the feedback loop is slow:

1. Edit the YAML
2. Run `gherkio run login.yaml` — wait for the *entire* scenario
3. Scroll through output to find the step you care about
4. Repeat

A single-step execution mode cuts this loop to milliseconds by running **only the request under your cursor**.

### 2.3 User Stories

- **As a** backend engineer, **I want to** copy a cURL from Chrome DevTools and paste it into my test file as YAML, **so that** I can create a test in seconds.
- **As a** developer debugging a CI failure, **I want to** generate a cURL command from a failing step, **so that** I can reproduce the issue outside Gherkio.
- **As a** QA engineer reviewing test coverage, **I want to** convert an existing cURL collection into Gherkio tests, **so that** I don't manually type every request.
- **As a** daily Gherkio user, **I want to** put my cursor anywhere inside a step and execute just that request, **so that** I can iterate on it without waiting for the whole scenario.

---

## 3. Feature A: `gherkio convert` — cURL ↔ DSL

### 3.1 CLI Interface

```bash
# Direction 1: cURL → YAML (stdin or argument)
gherkio convert 'curl -X POST https://api.example.com/login -H "Content-Type: application/json" -d "{\"email\":\"test@test.com\"}"'

gherkio convert --file request.txt       # Read cURL from file
echo 'curl ...' | gherkio convert         # Pipe from clipboard (nvim)

# Flags
gherkio convert 'curl ...' --scenario "Login Flow"   # Custom scenario name
gherkio convert 'curl ...' --step-only               # Output just the step (no scenario wrapper)
gherkio convert 'curl ...' --output test.yaml         # Write directly to file

# Direction 2: YAML → cURL
gherkio convert --reverse login.yaml                  # All steps as cURL
gherkio convert --reverse login.yaml --step 2         # Specific step
gherkio convert --reverse login.yaml --step 2 --env staging   # Resolve variables with env
```

### 3.2 Output Formats

**cURL → YAML (full scenario):**

```yaml
scenario: untitled

steps:
  - request:
      method: POST
      url: /login
      headers:
        Content-Type: application/json
      body:
        email: test@test.com
    expect:
      status: 200
```

**cURL → YAML (step-only):**

```yaml
  - request:
      method: POST
      url: /login
      headers:
        Content-Type: application/json
      body:
        email: test@test.com
    expect:
      status: 200
```

**YAML → cURL (with variable resolution):**

```bash
curl -X POST 'https://api.example.com/login' \
  -H 'Content-Type: application/json' \
  -d '{"email":"test@test.com"}'
```

### 3.3 Parsing Scope (cURL → YAML)

We parse the **common flags** that cover ~80% of real-world cURL usage:

| Flag | Maps to | Priority |
|------|---------|----------|
| `-X` / `--request` | `request.method` | High |
| `-H` / `--header` | `request.headers` | High |
| `-d` / `--data` / `--data-raw` | `request.body` (raw string) | High |
| `--data-json` | `request.body` (auto-parse as JSON) | High |
| URL (last positional arg) | `request.url` | High |
| `-F` / `--form` | `request.body` (multipart → will warn) | Low |
| `-u` / `--user` | `request.headers[Authorization]` Basic auth | Medium |
| `-b` / `--cookie` | `request.headers[Cookie]` | Medium |
| `-k` / `--insecure` | Skipped (warning) | Low |
| `--compressed` | Skipped | Low |
| `--max-time` | `request.timeout` | Medium |

**Behavior for unsupported flags:**
- Print a warning to stderr but still produce YAML
- Example: `⚠  Ignored flag: --compressed`
- Never fail on unknown flags

### 3.4 URL Resolution Logic

When `--env` is provided (or a `.gherkio` project is detected):

```yaml
# Input cURL:
# curl https://api.example.com/v1/users/me

# Output YAML (with env):
request:
  method: GET
  url: /v1/users/me    # <-- stripped base URL

# Output YAML (without env):
request:
  method: GET
  url: https://api.example.com/v1/users/me    # <-- full URL kept
```

**Heuristic:** Find the longest suffix that matches `env.baseUrl` or known service URLs. If no match, keep the full URL.

### 3.5 Variable Detection

Auto-detect patterns that look like Gherkio variables:

```yaml
# cURL: -H "Authorization: Bearer eyJhbG..."
# → Kept as literal value (no magic)

# cURL: -H "Authorization: Bearer $token"
# → Gherkio variable interpolation works natively — pass through unchanged

# cURL: -d "{\"username\":\"$username\"}"
# → Pass through as-is; Gherkio will interpolate at runtime
```

No special variable detection logic needed — Gherkio's existing `$var` / `${var}` interpolation in `interpolator.go` handles this naturally.

### 3.6 Auth Header Detection

```yaml
# cURL: -H "Authorization: Bearer eyJhbG..."
# → Output as-is

# cURL: -u admin:secret
# → Convert to Base64 and output as Basic auth header
#   request:
#     headers:
#       Authorization: Basic $(echo -n 'admin:secret' | base64)
```

User should be warned that literal tokens in cURL commands will be embedded in the YAML and masked at runtime.

### 3.7 YAML → cURL (Reverse)

This is the simpler direction. The logic already exists in `internal/report/helpers.go` via `generateCurl()`. The command just surfaces it:

```
gherkio convert --reverse <test-file> [--step <N>] [--env <name>]
```

- Reads the test file
- Interpolates variables with values from the environment (if `--env` provided)
- Outputs one or more cURL commands
- Variable interpolation with credentials from the credentials file if `--account` is provided

### 3.8 Neovim Integration

No plugin needed. The command is designed for piping:

```vim
" Convert selected cURL to YAML (visual mode)
:'<,'>!gherkio convert --step-only

" Grab step as cURL
:r !gherkio convert --reverse % --step 2

" Convert whole file
:%!gherkio convert --step-only
```

The `--step-only` flag is specifically for this — paste into an existing scenario without the wrapper.

---

## 4. Feature B: `gherkio run --step` — Run Step Under Cursor

### 4.1 CLI Interface

```bash
# By step index (explicit)
gherkio run login.yaml --step 0              # Run first step (0-indexed)
gherkio run login.yaml --step 2              # Run third step

# By line number (smart detection)
gherkio run login.yaml --line 25             # Detect which step contains line 25

# With section filter
gherkio run login.yaml --line 25 --section setup    # Only search setup block
gherkio run login.yaml --line 25 --section steps    # Only search main steps
gherkio run login.yaml --line 25 --section teardown # Only search teardown block

# Standard flags still work
gherkio run login.yaml --step 2 --verbose
gherkio run login.yaml --step 2 --env staging
gherkio run login.yaml --step 2 --account alpha
```

### 4.2 Smart Step Detection (Line-Based)

When using `--line`, the tool parses the raw YAML text (line-number-aware) to find which step the cursor is inside:

```yaml
# line 1:  scenario: login flow
# line 2:  
# line 3:  steps:
# line 4:    - request:            <-- step 0 boundary
# line 5:        method: POST
# line 6:        url: /auth/login
# line 7:        body:             <-- cursor here (anywhere between line 4-9)
# line 8:          username: $username
# line 9:      expect:
# line 10:       status: 200
# line 11:
# line 12:   - request:            <-- step 1 boundary
# line 13:       method: GET
# line 14:       url: /auth/me
```

**Algorithm:**

```
1. Read file as raw lines
2. Scan for `setup:`, `steps:`, `teardown:` blocks
3. Within each block, find all `- request:` or `- use:` lines at the same indent level
4. Each match = a step boundary with start/end line numbers
5. Given a cursor line, find which step boundary contains it
6. Extract that step by index
```

**Edge cases:**
- Cursor on blank line between steps → nearest preceding step
- Cursor before any step → error with available steps list
- Cursor outside `steps:`/`setup:`/`teardown:` block → error with available sections
- `--line` used with `--section` → searches only within that section

### 4.3 Execution Mode: Isolated

When running a single step, variables from previous steps are **not available**. Only credential variables (`$username`, `$password`, `$role`) are injected. This keeps the execution clean and predictable — you're testing *this request*, not the entire chain.

```bash
# Login step with credentials works:
gherkio run login.yaml --step 0
# $username = emilys, $password = emilyspass (from credentials file)
# $token = NOT available (from save: in a previous run)
```

If the step depends on a saved variable, the runner shows a clear warning:

```
⚠  Step references $token which is only available from:
    → login.yaml:step 2 (save: token: body.accessToken)
  Run the full scenario first, or use --deps to include dependencies.
```

### 4.4 Execution Mode: With Dependencies (Future)

A future `--deps` flag could run all steps before the target:

```bash
gherkio run login.yaml --step 2 --deps
# Runs steps 0, 1, then 2 sequentially
# Variables from save: in steps 0-1 are available in step 2
```

Not included in MVP.

### 4.5 Output Format (Compact)

Single-step output is intentionally compact — no scenario header, no summary footer:

```
▼ POST /auth/login
  ✓ status = 200
  ✓ body.accessToken = exists
  ✗ body.email = emilys@test.com
    └─ got: "emilys@example.com"

Response:
Status: 200

Body:
{
  "accessToken": "eyJ...",
  "email": "emilys@example.com"
}
```

**Verbose mode (`--verbose` / `-v`):** Shows full request/response payloads inline:

```
▼ POST /auth/login

Request:
POST /auth/login
Body:
{
  "username": "emilys",
  "password": "***masked***"
}

Response:
Status: 200

Body:
{
  "accessToken": "***masked***",
  "email": "emilys@example.com"
}

Assertions:
  ✓ status = 200
  ✓ body.accessToken = exists
  ✗ body.email = emilys@test.com
    └─ got: "emilys@example.com"

Duration: 342ms
```

No scenario header, no total summary. Just the step.

### 4.6 Use Steps Handling

When the target step is a `use:` step (composition), the runner:

1. Resolves the referenced file
2. Executes all steps in that file (not just one — composition is all-or-nothing)
3. Outputs results flattened (same as normal `run` does for use steps)

```
gherkio run login.yaml --line 30
# If line 30 is inside:  - use: refresh.yaml
# → Executes ALL steps in refresh.yaml, shows them inline
```

### 4.7 Neovim Workflow

The primary motivation is Neovim integration. The command is designed to be called from editor commands:

```vim
" Map to a key — run step under cursor
nnoremap <leader>rt :!gherkio run % --line <C-r>=line('.')<CR><CR>

" Run step with verbose output
nnoremap <leader>rT :!gherkio run % --line <C-r>=line('.')<CR> --verbose<CR>

" Run step for a specific account
nnoremap <leader>ra :!gherkio run % --line <C-r>=line('.')<CR> --account alpha<CR>
```

Or using `g:gherkio` Neovim plugin (future):

```lua
-- Hypothetical plugin
require('gherkio').setup({
  keymaps = {
    run_step = '<leader>rt',
    run_step_verbose = '<leader>rT',
  }
})
```

No Neovim plugin is required for MVP — just the shell command.

---

## 5. Implementation Plan

### 5.1 Files to Create

| File | Feature | Responsibility |
|------|---------|---------------|
| `cmd/convert.go` | Convert | Cobra command definition, flag parsing, orchestration |
| `cmd/run.go` (modify) | Step Runner | Add `--step` and `--line` flags, modify `runTest()` to handle single-step mode |
| `internal/converter/parser.go` | Convert | cURL string tokenizer + parser |
| `internal/converter/parser_test.go` | Convert | Parser tests (golden file approach) |
| `internal/converter/dsl.go` | Convert | Convert parsed cURL → Gherkio YAML (string) |
| `internal/converter/curl.go` | Convert | Convert Gherkio step → cURL string (wrapper around `generateCurl`) |
| `internal/runner/steplocator.go` | Step Runner | Parse YAML text → step boundaries by line number |
| `internal/runner/steplocator_test.go` | Step Runner | Tests for step detection |
| `internal/runner/executor.go` (modify) | Step Runner | Extract single-step execution from `executeSteps()` |
| `internal/runner/printer.go` (modify) | Step Runner | Compact single-step output format |

### 5.2 Why `internal/converter/` package?

- Keeps parsing logic isolated from the runner and report packages
- The converter has zero dependency on execution logic
- Easy to iterate without touching core runner code
- Clean separation if we later add Postman/HAR converters

### 5.3 Parser Architecture (Convert)

```
Input: "curl -X POST https://... -H 'Content-Type: application/json' -d '...'"
                            │
                    ┌───────▼───────┐
                    │   Tokenizer    │  Splits by shell-like tokenization (handles quotes)
                    └───────┬───────┘
                            │
                    ┌───────▼───────┐
                    │    Parser     │  Flag → value mapping
                    └───────┬───────┘
                            │
                    ┌───────▼───────┐
                    │  Convert to   │  Maps to model.Request + model.Expect
                    │  Gherkio DSL  │
                    └───────┬───────┘
                            │
                    ┌───────▼───────┐
                    │   YAML Output │  Uses gopkg.in/yaml.v3 for serialization
                    └───────────────┘
```

### 5.4 Step Locator Architecture (Step Runner)

```
Input: file path + line number
              │
      ┌───────▼───────┐
      │  Read raw file │  bufio.Scanner, line-by-line
      └───────┬───────┘
              │
      ┌───────▼───────┐
      │  Find sections │  Scan for "setup:", "steps:", "teardown:"
      └───────┬───────┘
              │
      ┌───────▼───────┐
      │  Find steps   │  Match "  - request:" / "  - use:" at block indent
      └───────┬───────┘
              │
      ┌───────▼───────┐
      │  Locate step  │  Binary search: which step range contains line?
      └───────┬───────┘
              │
      ┌───────▼───────┐
      │  Extract step │  Return step index → runner executes it
      └───────────────┘
```

### 5.5 Tokenizer Approach (Convert)

Shell-level tokenization is the hardest part. Two options:

**Option A: Use `strings.Fields()` + manual quote handling** (~150 lines)
- Pros: No dependency, full control
- Cons: Edge cases with nested quotes, escaped chars
- Verdict: Acceptable for MVP — we can warn on unparseable input

**Option B: Use `mvdan.cc/sh/v3` shell parser** (external dependency)
- Pros: Handles all shell quoting, piping, redirects
- Cons: Adds dependency, overkill for our use case
- Verdict: Not recommended for MVP

**Recommendation:** Option A with clear error messaging. If users hit parsing issues, they can simplify the cURL (remove shell-specific syntax).

### 5.6 Changes to Existing Code

- **`cmd/run.go`**: Add `--step` (int) and `--line` (int) flags. Modify `runTest()` to detect single-step mode and call `executeSingleStep()` instead of the full pipeline.
- **`internal/runner/runner.go`**: Add `RunSingleStep()` function — loads env + credentials, executes one step, returns `RunResult`.
- **`internal/runner/executor.go`**: Extract `executeSingleStep()` from the loop in `executeSteps()`. Currently the execution logic for one step is embedded in the `for _, step := range steps` loop — we need to isolate it.
- **`internal/report/helpers.go`**: `generateCurl()` is already public. No changes needed — the converter calls it directly.
- **`internal/runner/printer.go`**: Add `PrintStepResult()` for compact single-step output.

### 5.7 Testing Strategy

**Convert parser tests (golden file):**

```
internal/converter/testdata/
  parser/
    simple_get.txt          → simple_get.yaml
    post_json.txt           → post_json.yaml
    post_form.txt           → post_form.yaml
    with_headers.txt        → with_headers.yaml
    bearer_auth.txt         → bearer_auth.yaml
    full_url.txt            → full_url.yaml
    unknown_flag.txt        → unknown_flag.yaml (warning on stderr)
    edge_case_empty.txt     → edge_case_empty.yaml
```

Use `cmp` (or similar) for structural comparison of the parsed result.

**Reverse direction tests:**

```
internal/converter/testdata/
  reverse/
    simple_step.yaml        → expected_curl.txt
```

**Step locator tests:**

```
internal/runner/testdata/
  steplocator/
    basic_steps.yaml        → step 0 boundary (lines 4-10), step 1 (lines 12-18)
    with_setup.yaml         → setup step 0, main steps 0-1, teardown step 0
    with_use.yaml           → use step at index 1
    cursor_blank_line.yaml  → blank line between steps → nearest preceding
    cursor_outside.yaml     → cursor before steps → error
    empty_steps.yaml        → no steps → error
```

---

## 6. Open Questions

### 6.1. Should Convert auto-generate assertions beyond `status:200`?

cURL gives us zero assertion info. The safe default is `status:200`. But we could **optionally**: detect JSON response structure and add `body.X: exists` for each top-level key. Not recommended for MVP — leads to noisy, overly-permissive tests.

**Decision for MVP:** Only `status:200`. No auto-generated field assertions.

### 6.2. How do we handle `-d` with JSON that has `$` strings?

If `-d '{"token": "$test123"}'` — is `$test123` a Gherkio variable or a literal string? We can't distinguish. Gherkio's interpolator handles this by:
- If variable `test123` exists → interpolates
- If not → leaves as literal string

This is acceptable behavior. No special handling needed.

### 6.3. Should `--step-only` output valid YAML that can be appended to a file?

Yes. It outputs a single step entry starting with `  - request:`, which can be appended to an existing `steps:` block. The indentation (2 spaces) must match Gherkio's convention.

### 6.4. For `--line`, should setup/teardown steps be discoverable?

Yes. The step locator scans all three sections (`setup`, `steps`, `teardown`) by default. The `--section` flag narrows the search. This way, you can debug a teardown step the same way as a main step.

### 6.5. Overlap with `--step` and `--line` in same command?

If both are provided, `--step` wins (explicit over inferred). `--line` is ignored with a warning.

---

## 7. Future Iterations

### Post-MVP (not in scope for this RFC)

- **HAR file converter**: `gherkio convert --from-har recordings.har`
- **Postman collection converter**: `gherkio convert --from-postman collection.json`
- **OpenAPI → scenario skeleton**: `gherkio convert --from-openapi spec.yaml`
- **Interactive mode**: `gherkio convert --interactive` that walks through cURL parts and lets you add assertions inline
- **Step runner `--deps`**: Run all previous steps before the target
- **Neovim plugin**: First-class Neovim plugin with floating result windows
- **`gherkio watch`**: Auto-rerun on file change

---

## 8. Appendix

### 8.1 Comparison with similar tools

| Tool | Direction | Approach |
|------|-----------|----------|
| `curlconverter` (Python/JS) | cURL → many formats | Mature parser, used by Postman |
| `httpie` | CLI native | Has `--output` for OpenAPI |
| Gherkio (this RFC) | Bidirectional | Gherkio DSL focused, shell-level parsing |

### 8.2 Existing `generateCurl` in Report Package

The reverse direction is already partially implemented in `internal/report/helpers.go`:

```go
func generateCurl(req *runner.RequestInfo, maskFields []string) string
```

This function takes a `RequestInfo` (method, URL, headers, body) and outputs a cURL command string. The `convert --reverse` command wraps this, adding:
- Reading the YAML test file
- Extracting the request from a specific step
- Variable interpolation with environment values
- Output formatting

### 8.3 Example: Full Round-Trip

```bash
# Original cURL from browser DevTools
curl 'https://dummyjson.com/auth/login' \
  -H 'Content-Type: application/json' \
  --data-raw '{"username":"emilys","password":"emilyspass","expiresInMins":30}'

# Convert to Gherkio DSL
gherkio convert 'curl ...' --scenario "Login Flow"

# Output:
scenario: Login Flow

steps:
  - request:
      method: POST
      url: /auth/login
      headers:
        Content-Type: application/json
      body:
        username: emilys
        password: emilyspass
        expiresInMins: 30
    expect:
      status: 200

# Later, convert back to share with a teammate
gherkio convert --reverse login.yaml --step 1

# Output:
curl -X POST 'https://dummyjson.com/auth/login' \
  -H 'Content-Type: application/json' \
  -d '{"username":"emilys","password":"emilyspass","expiresInMins":30}'
```

### 8.4 Example: Run Step Under Cursor

```yaml
# login.yaml
scenario: login flow

setup:
  - request:
      method: POST
      url: /auth/setup
    expect:
      status: 200

steps:
  - request:
      method: POST
      url: /auth/login
      body:
        username: $username
    expect:
      status: 200
      body.accessToken: exists
    save:
      token: body.accessToken

  - request:
      method: GET
      url: /auth/me
      headers:
        Authorization: Bearer $token
    expect:
      status: 200
```

```bash
# Cursor on line 14 (body.username), running:
gherkio run login.yaml --line 14

# Output:
▼ POST /auth/login
  ✓ status = 200
  ✓ body.accessToken = exists

Response:
Status: 200

Body:
{
  "accessToken": "eyJ...",
  "email": "emilys@test.com"
}
```

Note: `$token` is NOT available when running step 2 in isolation — it was saved by step 1. The step runner warns about this.

---

