# RFC-5: Request Retry

> **Status:** Draft (Revised)
> **Author:** Faris
> **Date:** May 22, 2026
> **Revision:** Removed `until`, deferred `onStatus` to Phase 2, added `timing.max` interaction, idempotency warning, `use:` restriction, connection error handling in retry loop, `Body` field on `RetryEntry`, moved `maxDuration` to Phase 1, retry count = attempts - 1 (only shown when > 0)

---

## 1. Summary

Add configurable retry logic to request steps for handling eventual consistency, polling, and transient failures — without introducing imperative loops into the DSL.

---

## 2. Motivation

Many integration scenarios involve eventual consistency:

- An order is created (202 Accepted), then polling until status = "confirmed"
- A resource is deleted (204), then verifying it returns 404
- A background job completes (202), then checking result appears
- Transient network failures or rate limiting

Without retry, users must either:
- Accept flaky tests — bad for CI
- Write imperative retry logic outside Gherkio — defeats the purpose of the DSL
- Use fixed `time.Sleep` — slow and fragile

---

## 3. Design

### 3.1 Retry Configuration

Retry is specified as a block at the step level, separate from `request` and `expect`:

```yaml
- request:
    method: GET
    url: /orders/$orderId

  retry:
    attempts: 5
    interval: 1000        # milliseconds between attempts
    backoff: linear        # or: exponential, constant (default: constant)

  expect:
    status: 200
    body.status: confirmed
```

The retry loop re-executes the request until **all assertions in `expect` pass**, or until all attempts are exhausted. No separate `until` condition is needed — `expect` is the single source of truth for retry success.

### 3.2 Retry Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `attempts` | Yes | — | Maximum number of attempts |
| `interval` | No | 500 | Base interval in milliseconds |
| `backoff` | No | `constant` | Backoff strategy: `constant`, `linear`, `exponential` |
| `maxDuration` | No | — | Maximum total wall time for all attempts (e.g. `30s`) |

> **Display rule:** Retry info is only shown in output/reports when **actual retries > 0** (i.e. the step didn't pass on the first attempt). The displayed retry count is `attempts - 1` — the number of extra HTTP calls beyond the initial one.
> - `attempts: 3, passed on call 3` → `retry: 2`
> - `attempts: 5, passed on call 1` → retry is invisible (no retry needed)

### 3.3 Backoff Strategies

```yaml
# Constant — same delay between every attempt
retry:
  attempts: 3
  interval: 1000
  backoff: constant
# → wait 1s, wait 1s

# Linear — delay increases linearly
retry:
  attempts: 5
  interval: 1000
  backoff: linear
# → wait 1s, wait 2s, wait 3s, wait 4s

# Exponential — delay doubles each attempt
retry:
  attempts: 5
  interval: 1000
  backoff: exponential
# → wait 1s, wait 2s, wait 4s, wait 8s
```

A simple jitter of ±25% of the interval is applied by default to all strategies to prevent thundering herd. Not configurable.

### 3.4 Retry Loop Semantics

```
for attempt = 1; attempt <= attempts; attempt++:
  response, err = executeRequest(...)

  if err != nil:
    if attempt == attempts:
      break (step fails, store error in RetryEntry)
    wait(backoff(interval, attempt))
    continue

  if all assertions pass:
    break (step passes)

  if attempt == attempts:
    break (step fails, use last response for assertions)

  if maxDuration exceeded:
    break (step fails with timeout)

  wait(backoff(interval, attempt))
```

**Important:** Variables (`$var` / `${var}`) in the request are resolved **once** before the retry loop begins, not re-interpolated per attempt. This ensures variable values remain frozen during polling. JWT claims are re-decoded from each response attempt since the token may change between retries.

### 3.5 Retry Assertions

When all retry attempts are exhausted, the last response is used for assertions:

```
✗ GET /orders/42
    └─ retry: 4 exhausted (5 attempts)
    └─ last response: status 202, body.status = "pending"

  ✗ status = 200
      └─ got: 202
  ✗ body.status = confirmed
      └─ got: pending
```

### 3.6 Retry Reporting

The output shows retry history on success (only when retries > 0):

```
✓ GET /orders/42
  ✓ status = 200
  ✓ body.status = confirmed
  └─ retry: 2, last at retry 2
```

If the step passes on the first attempt, no retry line is shown at all: the step output looks identical to a non-retry step.

### 3.7 Interaction with `timing.max`

When a step has both `retry` and `timing`:

```yaml
retry:
  attempts: 5
  interval: 1000
timing:
  max: 1s
```

`timing.max` applies to the **total duration across all retry attempts**, not per-attempt. This prevents a retrying step from silently passing a timing assertion when it took 5s total.

If the total exceeds `timing.max`, the step still runs all retries but the timing assertion fails, giving clear feedback:

```
✗ timing.max = 1s (actual: 3.2s)
    └─ retry: 2, last at retry 2
```

---

## 4. Edge Cases

### 4.1 Retry on Specific Status Codes (deferred to Phase 2)

Some APIs return 503 (Service Unavailable) or 429 (Rate Limited) during normal operation. A future phase may add:

```yaml
retry:
  attempts: 3
  interval: 1000
  onStatus: [429, 503]
```

If not specified, retry on any failed assertion or connection error.

**Deferred:** Phase 1 retries on any assertion failure. Phase 2 adds `onStatus` for selective retry on specific status codes before assertions are evaluated.

### 4.2 Total Timeout

To prevent runaway polling, a maximum total duration should be enforced:

```yaml
retry:
  attempts: 10
  interval: 1000
  maxDuration: 30s   # fails after 30 seconds regardless
```

If `maxDuration` is exceeded mid-retry, the step fails immediately with a timeout error.

### 4.3 Retry with Save

Variables are extracted from the **last successful** response (the one where all assertions passed). If all attempts fail, the last response is used (no variables extracted).

### 4.4 Idempotency Warning

The engine prints a warning to console **during execution** when retrying non-idempotent methods:

```
⚠ POST /orders — retrying non-idempotent request
```

Triggered for `POST`, `PUT`, `PATCH`, `DELETE` on retry attempt >= 2.

### 4.5 `use:` + Retry (explicitly forbidden)

```yaml
- use: login.yaml
  retry:
    attempts: 3   # ❌ This is rejected with a validation error
```

Retry on a `use:` step is explicitly **not supported**. Place retry inside the referenced scenario if needed. This avoids ambiguity about whether the retry wraps the entire composed scenario or an individual step within it.


---

## 5. Implementation Plan

### Phase 1 — Basic Retry Loop

- [ ] Add `RetryConfig` struct to Step model (`internal/model/test.go`)

```go
type RetryConfig struct {
	Attempts    int    `yaml:"attempts"`
	Interval    int    `yaml:"interval,omitempty"`
	Backoff     string `yaml:"backoff,omitempty"`
	MaxDuration string `yaml:"maxDuration,omitempty"`
}
```

- [ ] Add `Retry` field to `Step` struct
- [ ] In `executeSteps()` in `runner.go`, wrap `executeRequest()` + assertion evaluation in a retry loop
- [ ] Add `RetryCount` and `RetryHistory` to `StepResult`:

```go
type StepResult struct {
	RetryCount   int          `json:"retryCount,omitempty"`   // actual retries (attempts - 1), 0 = no retry
	RetryHistory []RetryEntry `json:"retryHistory,omitempty"`
}

type RetryEntry struct {
	Attempt  int           `json:"attempt"`
	Status   int           `json:"status"`
	Body     string        `json:"body,omitempty"`        // truncated first 500 chars of response body
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
}
```

> `RetryCount` is populated only when > 0 (step required retries). When 0, no retry info is shown in output or reports. The `RetryHistory` still captures all internal attempt info for debugging.

- [ ] Validate that `retry` is not used with `use:` steps
- [ ] Print idempotency warning during execution
- [ ] Variables are resolved once before the retry loop
- [ ] Implement `maxDuration` guard (hard timeout to prevent runaway polling)

### Phase 2 — Backoff Strategies

- [ ] Implement `linear` and `exponential` backoff strategies with jitter (±25%)
- [ ] Handle `timing.max` interaction with retry (total duration across all attempts)
- [ ] Implement `onStatus` for selective retry on specific status codes (deferred from Phase 1)

### Phase 3 — Reporting

- [ ] Show retry history in printer output and HTML/JSON reports
- [ ] Add retry metadata to HTML report (badge showing attempt count, expandable history)

---

## 6. Example Output

**Passing after retry (retries > 0):**

```
✓ GET /orders/42
  ✓ status = 200
  ✓ body.status = confirmed
  └─ retry: 2, last at retry 2
```

**Passing on first attempt (retries = 0, no retry info shown):**

```
✓ GET /orders/42
  ✓ status = 200
  ✓ body.status = confirmed
```

**Failing after all retries:**

```
✗ GET /orders/42
    └─ retry: 4 exhausted (5 attempts)
    └─ last error: connection refused (retry 2)

  ✗ status = 200
      └─ got: 503

✗ FAIL
0 passed, 1 failed, 1 total
```

**Failing due to `maxDuration`:**

```
✗ GET /orders/42
    └─ retry: 2/9, maxDuration 15s exceeded
    └─ last response: status 202, body.status = "processing"
```

---

## 7. Decisions

### 7.1 Step-level, not request-level

Retry is a step-level configuration because it affects the entire step lifecycle — request execution, assertion evaluation, variable saving, and timing measurement.

### 7.2 `use:` + retry forbidden

Retry on `use:` steps is rejected with a validation error. Apply retry inside the referenced scenario instead.

### 7.3 No `until` condition

The `expect` block is the single source of truth for retry success. No separate `until` condition.

### 7.4 `onStatus` deferred to Phase 2

Phase 1 retries on any assertion failure. Phase 2 adds `onStatus` for selective status code retry.

### 7.5 `timing.max` measures total, not per-attempt

Timing assertion applies to aggregate duration across all retry attempts.

### 7.6 Variables frozen before retry

Variables are interpolated once before the retry loop starts, not re-interpolated per attempt.

### 7.7 Retry count displayed = attempts - 1

The output and reports always show **retries** (extra calls beyond the initial one), not total attempts. A step configured with `attempts: 5` that passes on the 4th call shows `retry: 3`. If it passes on the first call, retry is invisible — the step output is indistinguishable from a non-retry step. This avoids noise for the common case.

`RetryCount` on `StepResult` is populated only when > 0.

---

## 8. Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Long-running retries block the entire test suite | `maxDuration` guard; console prints attempt progress every 3s |
| POST/PUT retries create duplicate resources | Idempotency warning; documentation advises using idempotency keys |
| Users rely on flaky tests passing via retry | Retry history is always visible in output and reports |
| Backoff + many attempts = long wait | Default `interval: 500` keeps total reasonable; docs suggest starting with `3` attempts |

---

## 9. Dependencies

| Dependency | Purpose |
|------------|---------|
| `math/rand` (stdlib) | Jitter (±25%) |
| `time` (stdlib) | Sleep between attempts, maxDuration checking |

No new external dependencies.
