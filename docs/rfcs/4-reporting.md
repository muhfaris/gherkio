# RFC-4: Reporting (HTML & JSON)

> **Status:** Ready (Revised)
> **Author:** Faris
> **Date:** May 21, 2026
> **Revision:** Simplified artifact handling, HTML-first phases, no symlinks, added cURL + request ID

---

## 1. Summary

Add structured reporting output in addition to the existing console printer. Generate HTML reports for human review and JSON reports for CI/CD ingestion and historical analysis.

---

## 2. Motivation

The PRD states reporting is a first-class product feature (section 15). Currently, Gherkio only outputs to the console. This is ephemeral — once the terminal scrolls past, the results are lost.

For CI/CD pipelines and team collaboration, users need:

- **HTML reports** — browsable, shareable, with visual pass/fail indicators
- **JSON reports** — machine-readable for CI/CD, historical analysis, and custom dashboards
- **Artifact preservation** — request/response payloads saved for debugging
- **Execution history** — ability to compare runs over time

---

## 3. Design

### 3.1 CLI Interface

```bash
# Default: console output only (current behavior)
gherkio run tests/login.yaml

# Generate HTML report
gherkio run tests/login.yaml --report html

# Generate JSON report
gherkio run tests/login.yaml --report json

# Generate both
gherkio run tests/login.yaml --report html --report json

# Specify output directory (overrides config path)
gherkio run tests/login.yaml --report html --output ./reports/debug-run
```

### 3.2 Report Directory Structure

Request/response payloads are embedded inline in the report files. Separate artifact files are only extracted for payloads exceeding 1MB.

```
.gherkio/reports/
├── latest/                           # always the most recent run
│   ├── report.html
│   └── report.json
├── 2026-05-21_14-30-00/              # timestamped copy
│   ├── report.html
│   └── report.json
├── 2026-05-21_14-15-00/
│   └── ...
└── archive/                          # old runs (compressed)
```

Rationale for removing separate artifact files:
- Simpler code — no file-per-step management
- Easier to share — a single report file contains everything
- The JSON format already has request/response fields inline
- Separate extraction is deferred until payloads exceed 1MB, which is rare for API responses

### 3.3 JSON Report Format

Each step includes:
- `duration` (milliseconds int) + `durationLabel` (human-readable string)
- `curl` — a copy-pasteable cURL command equivalent of the request (with sensitive values masked)
- `requestId` — extracted from response headers (`x-request-id`, `x-request-id`, `request-id`, or `requestId`), if present

```json
{
  "metadata": {
    "version": "1.0.0",
    "timestamp": "2026-05-21T14:30:00Z",
    "duration": 1250,
    "scenario": "login example",
    "testFile": "tests/auth/login.yaml",
    "environment": "local"
  },
  "summary": {
    "total": 4,
    "passed": 4,
    "failed": 0,
    "passedPercent": 100
  },
  "steps": [
    {
      "index": 1,
      "label": "POST https://dummyjson.com/auth/login",
      "duration": 312,
      "durationLabel": "312ms",
      "passed": true,
      "curl": "curl -X POST 'https://dummyjson.com/auth/login' -H 'Content-Type: application/json' -d '{\"username\":\"emilys\",\"password\":\"***masked***\"}'",
      "request": {
        "method": "POST",
        "url": "https://dummyjson.com/auth/login",
        "headers": { "Content-Type": "application/json" },
        "body": { "username": "emilys", "password": "***masked***" }
      },
      "response": {
        "status": 200,
        "headers": {
          "Content-Type": "application/json",
          "x-request-id": "req_abc123"
        },
        "body": { "accessToken": "***masked***", "username": "emilys" }
      },
      "requestId": "req_abc123",
      "assertions": [
        { "path": "status", "expected": "200", "actual": "200", "passed": true },
        { "path": "body.accessToken", "expected": "exists", "actual": "***masked***", "passed": true }
      ]
    }
  ],
  "errors": []
}
```

### 3.4 cURL Generation

A `curl` string is generated for each step to allow one-click debugging. The format:

```
curl -X <METHOD> '<URL>' -H '<Header>: <Value>' -d '<JSON body>'
```

Rules:
- Sensitive field values are masked (`***masked***`) in the cURL output
- Headers are appended as `-H` flags
- JSON body is minified and passed via `-d` (single-quoted for POSIX shells)
- GET requests without a body omit the `-d` flag
- No `--insecure` or extra flags — the cURL reflects the exact request made

### 3.5 Request ID Extraction

When a response contains a request tracing header, it's extracted and displayed prominently in the HTML report (not just buried in the headers table). This helps users correlate test failures with server-side logs.

Detection order (case-insensitive):
1. `x-request-id`
2. `x-trace-id`
3. `request-id`
4. `requestId`
5. `x-correlation-id`

The first match found is used as `requestId`. If none are present, the field is omitted.

### 3.6 HTML Report

A standalone self-contained HTML file using Go's `html/template` with embedded CSS/JS via `//go:embed`. The template lives at `assets/template/report.html` and is embedded into the binary at build time.

**Template:** `assets/template/report.html`

This template handles all rendering — no external dependencies, no CDN references, no separate CSS/JS files. The Go code parses it with `template.New("report").Funcs(template.FuncMap{...}).ParseFS(embedFS, "assets/template/report.html")`.

Registered template functions:

| Function | Purpose |
|----------|---------|
| `add` | `add $i 1` — increment step index for display |
| `statusClass` | Maps HTTP status code to CSS class (`status-2xx`, `status-4xx`, etc.) |

**Template data structs** (Go types that bridge `RunResult`/`StepResult` to the template):

```go
type ReportData struct {
    ScenarioName string
    Environment  string
    Timestamp    string
    TotalDuration string
    TotalSteps   int
    PassCount    int
    FailCount    int
    PassPercent  float64
    FailPercent  float64
    Steps        []ReportStep
}

type ReportStep struct {
    Index        int
    Method       string
    URL          string
    StatusCode   int
    StatusText   string
    Duration     string           // human-readable, e.g. "312ms"
    TimingFailed bool             // true if timing.max assertion failed
    RequestID    string           // extracted from response headers, empty if none
    CurlCommand  string           // copy-pasteable curl command
    RequestBody  string           // pretty-printed JSON, masked
    ResponseBody string           // pretty-printed JSON, masked
    Passed       bool
    Assertions   []ReportAssertion
}

type ReportAssertion struct {
    Label  string // e.g. "status = 200" or "body.email = email (actual: ...)"
    Detail string // failure details, empty if passed
    Passed bool
}
```

The template renders:

- **Header:** Scenario name, timestamp, environment, total duration
- **Summary bar:** Green/red bar with pass/fail/total counts
- **Step list:** Expandable sections for each step
  - Method + URL + status badge (✓/✗)
  - **Response time badge** — prominently displayed next to the step header (e.g. `312ms`, `1.2s`). Red if the step has a `timing.max` assertion that failed, grey otherwise.
  - **Request ID badge** — if a request ID was detected in the response headers, show it as a copyable badge (e.g. `req_abc123`). Useful for cross-referencing with server-side logs.
  - **cURL command** — a copyable code block showing the equivalent cURL command. The user can copy-paste it directly into their terminal to reproduce the request.
  - Request body (expandable, with sensitive fields masked)
  - Response body (expandable, masked)
  - Assertion list with pass/fail icons and inline failure details
- **Dark/light mode** via CSS media query
- **Collapsible sections** — steps, request/response bodies expand on click
- **Copy buttons** — cURL command and request ID have copy-to-clipboard buttons
- **Expand all / Collapse all** — toolbar toggle
- **Auto-expand failures** — failed steps are open on load
- **Zero external dependencies** — single file, all CSS/JS inlined

Reuse existing Go code for formatting:
- `formatRequestBody()` from `printer.go` for pretty-printing JSON
- `maskSensitiveData()` from `printer.go` for sensitive field masking
- `formatDuration()` from `printer.go` for duration display

### 3.7 Config Integration

```yaml
# .gherkio/config.yaml
reports:
  path: .gherkio/reports
  format: html          # default format for `gherkio run`
  archive: true         # compress old runs
  retention: 30         # days to keep reports
  maskSensitive: true   # apply masking in reports too
```

The `--output` flag overrides `reports.path` for a single run.

---

## 4. Edge Cases

### 4.1 Large Responses

Responses larger than 1MB are truncated in the inline display with a note. Extraction to separate artifact files is deferred — revisit only if users encounter this in practice.

### 4.2 Sensitive Data

Reports respect the same masking rules as console output (`defaultSensitiveFields` + config overrides). JSON reports mask by default; `--report-raw` flag outputs unmasked data. cURL commands always mask sensitive fields regardless of `--report-raw`.

### 4.3 Report File Permissions

Default: 0644 (readable by CI/CD systems).

### 4.4 Concurrent Runs

Each run gets a unique timestamped directory. The `latest/` directory is a plain directory that is overwritten (delete contents, write new files — not a symlink). This avoids symlink compatibility issues on Windows.

### 4.5 `gherkio run` Without `--report`

No reports are saved. Console output is the default. Reports are opt-in via `--report` flag.

### 4.6 No Request ID Found

If no request tracing header is detected in the response, the `requestId` field is omitted from the JSON and no badge is shown in the HTML report. No error is raised.

### 4.7 cURL with Complex Payloads

For requests with `null` body or empty body, the `-d` flag is omitted. For binary payloads (future), the cURL shows `--data-binary @-` with a placeholder.

---

## 5. Implementation Plan

### Phase 1 — HTML Report (highest user impact)

- [ ] Embed `assets/template/report.html` via `//go:embed`
- [ ] Register template functions (`add`, `statusClass`) via `template.FuncMap`
- [ ] Implement `ReportData`, `ReportStep`, `ReportAssertion` structs to bridge `RunResult` → template
- [ ] Implement `generateCurl()` — builds cURL command string from `RequestInfo`
- [ ] Implement `extractRequestId()` — scans response headers for tracing headers
- [ ] Implement `RenderHTML(result, cfg)` returning a string
- [ ] Reuse `formatRequestBody()`, `maskSensitiveData()`, `formatDuration()` from `printer.go`
- [ ] Write `report.html` to `.gherkio/reports/latest/`
- [ ] Copy to timestamped directory `.gherkio/reports/<timestamp>/`
- [ ] Console output prints: `📄 Report saved: .gherkio/reports/latest/report.html`

### Phase 2 — JSON Report

- [ ] Implement `RenderJSON(result, cfg)` returning `[]byte`
- [ ] Include `curl`, `durationLabel`, `requestId` in each step object
- [ ] Write `report.json` alongside `report.html` in both `latest/` and timestamp directory
- [ ] Support `--report-raw` flag to skip masking
- [ ] Inline all request/response bodies in JSON (no separate artifact files)

### Phase 3 — Report Management

- [ ] Implement retention policy (delete reports older than N days from config)
- [ ] Implement archive/compression for old runs
- [ ] `gherkio report` subcommand is **deferred** — not part of this RFC

---

## 6. Example Output

```
gherkio run tests/auth/login.yaml --report html

✓ login example
  ✓ status = 200
  ✓ body.accessToken exists

✓ PASS
2 passed, 0 failed, 2 total
Duration: 312ms

📄 Report saved: .gherkio/reports/latest/report.html
```

---

## 7. Decisions

### 7.1 Template Engine

**Use Go's `html/template` from stdlib with `//go:embed`.** No external template engine dependencies. The HTML template is embedded directly into the binary, keeping it a single-file deployment.

### 7.2 Sensitive Data in Reports

**Masked by default.** JSON reports include masked values. `--report-raw` outputs unmasked data. HTML reports always mask (no raw HTML output). cURL commands always mask regardless of `--report-raw`.

### 7.3 Auto-save Without `--report`

**No.** Reports are only generated when `--report` is explicitly specified. Console output remains the default.

### 7.4 `latest/` Directory

**Plain directory, not a symlink.** On each run, the contents of `latest/` are replaced with the new report files. Cross-platform compatible.

### 7.5 Separate Artifact Files

**Deferred.** Request/response payloads are embedded inline in the report files. Separate artifact extraction only if payloads exceed 1MB.

### 7.6 `gherkio report` Subcommand

**Deferred.** Not part of this RFC. Reporting infrastructure (save, load, list) should be built first; a management CLI can be its own RFC.

### 7.7 cURL Masking Always On

Even when `--report-raw` is used, cURL commands in the report always have sensitive fields masked. This prevents accidental credential leaks when sharing cURL commands.

---

## 8. Dependencies

| Dependency | Purpose |
|------------|---------|
| `html/template` (stdlib) | HTML report rendering |
| `//go:embed` (stdlib) | Template file embedding |
| `time` (stdlib) | Timestamp formatting |
| `embed` (stdlib) | Embed `assets/template/report.html` into binary |
| `html/template` (stdlib) | HTML report rendering from embedded template |
| `encoding/json` (stdlib) | JSON report |
| `fmt` (stdlib) | cURL string building |
| `strings` (stdlib) | Header matching for request ID |

**Template file:** `assets/template/report.html` — embedded via `//go:embed` at build time.

No new external dependencies. The `formatRequestBody()`, `maskSensitiveData()`, and `formatDuration()` functions already exist in `internal/runner/printer.go` and are reused directly.
