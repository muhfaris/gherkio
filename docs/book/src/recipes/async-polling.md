# Async Polling & Eventual Consistency

Verify asynchronous operations (such as data pipeline synchronization or long-running audio processing jobs) that require time to complete.

---

## 📝 The Scenario

```yaml
scenario: Async Order Sync Validation
tags:
  - async
  - orders

steps:
  # Step 1: Submit long-running order sync task
  - request:
      method: POST
      url: /v1/orders/sync
    expect:
      status: 202
      body.jobId: exists
    save:
      syncJobId: body.jobId

  # Step 2: Poll status endpoint with backoff until completed
  - request:
      method: GET
      url: /v1/orders/sync/$syncJobId
    retry:
      attempts: 8
      interval: 1500                 # Interval in milliseconds between retries
      backoff: exponential           # Backoff strategy: "linear" or "exponential"
      maxDuration: 10s               # Maximum total duration allowed for the retry loop
    expect:
      status: 200
      body.status: "success"
      body.processedCount: gt 0
      body.errors: empty
```

---

## 💡 Key Design Patterns Used

1. **Wait State Separation**: Spares resources by avoiding a hardcoded script block that sleeps, utilizing standard intervals.
2. **Backoff Escalation**: Gradually increases interval duration, giving slow-settling tasks a wider execution window while finishing quickly for fast jobs.
3. **Step Halting**: If the status stays `pending` after 8 attempts, Gherkio automatically fails the test run with detailed step diagnostic summaries.
