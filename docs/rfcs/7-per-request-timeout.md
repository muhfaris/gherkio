# RFC-7: Per-Request Timeout

> **Status:** Draft
> **Author:** Faris
> **Date:** May 22, 2026

---

## 1. Summary

Add a configurable `timeout` field to the request step model, replacing the current hardcoded 30-second HTTP client timeout with a per-request override.

---

## 2. Motivation

Currently, Gherkio's HTTP client uses a hardcoded 30-second timeout in `executor.go`:

```go
client := &http.Client{Timeout: 30 * time.Second}
```

This causes problems in real-world integration testing:

- **Slow staging environments** — A cold API behind a load balancer may take >30s on the first call. Users cannot adjust the timeout without forking Gherkio.
- **Quick-failing tests** — Users may want a shorter timeout (e.g. 5s) to fail fast when an API is down, rather than waiting 30s.
- **Inconsistent UX** — The PRD (§10.2) lists `timeout` as a supported request feature, but the code doesn't expose it.

Without this, users have no control over request timing — a fundamental HTTP client concern.

---

## 3. Design

### 3.1 Model Change

Add a `Timeout` field to the `Request` struct in `internal/model/test.go`:

```go
type Request struct {
	Service string            `yaml:"service,omitempty"`
	Method  string            `yaml:"method"`
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers,omitempty"`
	Body    interface{}       `yaml:"body,omitempty"`
	Timeout string            `yaml:"timeout,omitempty"` // e.g. "5s", "30s", "1m"
}
```

### 3.2 DSL Syntax

```yaml
- request:
    method: GET
    url: /slow-endpoint
    timeout: 60s
```

### 3.3 Execution Change

In `executeRequest()` in `internal/runner/executor.go`, parse the timeout string and apply it to the HTTP client:

```go
func executeRequest(method, url string, headers map[string]string, body interface{}, timeoutStr string) (*ResponseInfo, error) {
    timeout := 30 * time.Second // default
    if timeoutStr != "" {
        parsed, err := time.ParseDuration(timeoutStr)
        if err == nil {
            timeout = parsed
        }
    }
    client := &http.Client{Timeout: timeout}
    // ...
}
```

### 3.4 Runner Integration

In `executeSteps()` in `runner.go`, pass the timeout from the step's request to `executeRequest()`:

```go
resp, err := executeRequest(
    interpolatedRequest.Method,
    url,
    interpolatedRequest.Headers,
    interpolatedRequest.Body,
    interpolatedRequest.Timeout,
)
```

---

## 4. Edge Cases

### 4.1 Invalid Timeout String

If the user provides an unparseable timeout (e.g. `timeout: fast`), fall back to the 30s default and optionally log a warning. This is a UX decision: failing hard would break a test over a typo in a non-critical field.

### 4.2 Zero/Empty Timeout

An empty `timeout` string means "use default" (30s). A value of `0s` means "no timeout" (pass to http.Client as 0, which means no timeout). This is consistent with Go's `http.Client` behavior.

### 4.3 `timeout` vs `timing.max`

These are distinct:
- `timeout` — per-request HTTP client deadline (applied to each individual HTTP call)
- `timing.max` — expected step duration (assertion, applied after the step completes)

Both can coexist on the same step. `timeout` affects execution, `timing.max` affects reporting.

---

## 5. Implementation Plan

- [ ] Add `Timeout` field to `Request` struct in `internal/model/test.go`
- [ ] Update `executeRequest()` signature to accept a `timeoutStr string` parameter
- [ ] Parse timeout duration in `executeRequest()` with fallback to 30s default
- [ ] Pass `step.Request.Timeout` from `executeSteps()` to `executeRequest()`
- [ ] Update `InterpolateRequest` to copy `Timeout` field to the interpolated request

---

## 6. Dependencies

| Dependency | Purpose |
|------------|---------|
| `time` (stdlib) | `time.ParseDuration()` for timeout string parsing |

No new external dependencies.

---

## 7. Decisions

### 7.1 Duration string format

Using Go duration strings (`"5s"`, `"30s"`, `"1m"`) to stay consistent with `timing.max` syntax elsewhere in Gherkio. Millisecond integers would be inconsistent.

### 7.2 Default remains 30s

Changing the default would break existing tests that implicitly rely on the 30s window. The default stays.
