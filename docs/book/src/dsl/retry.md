# Retry & Polling

In modern microservice and serverless systems, APIs are frequently eventually consistent. Gherkio offers native, step-level `retry` configurations to pull and assert against resources over time without writing custom loops.

---

## 🔁 Retry Key Reference

| Key | Type | Default | Description | Example |
| :--- | :--- | :--- | :--- | :--- |
| `attempts` | `integer` | `3` | Maximum number of retry attempts to perform | `attempts: 5` |
| `interval` | `string` | `1s` | Duration to wait between attempts | `interval: 2s` |
| `backoff` | `number` | `1.0` | Multiplier applied to `interval` after each attempt | `backoff: 1.5` |
| `maxInterval`| `string` | `10s` | Maximum boundary duration for exponential backoffs | `maxInterval: 5s` |

---

## ⚡ Eventually Consistent Polling Pattern

If a background job processes an asynchronous action (e.g. video processing or payment settlement), the status endpoint initially returns `pending`. We want to poll this endpoint until the status is `completed` or our retry limit is exceeded.

```yaml
steps:
  - request:
      method: POST
      url: /jobs
      body: { type: "compress-video" }
    save:
      jobId: body.id

  - request:
      method: GET
      url: /jobs/$jobId
    retry:
      attempts: 10
      interval: 1s
      backoff: 1.5                   # 1s, 1.5s, 2.25s, 3.37s, etc.
      maxInterval: 5s
    expect:
      status: 200
      body.status: completed        # Retries until this assertion succeeds!
```

### How Retries Work Under the Hood:
1. Gherkio executes the HTTP request.
2. It validates the response against the `expect` block.
3. If all assertions succeed, the step completes immediately.
4. If **any** assertion fails, Gherkio waits for the specified `interval`, updates the interval by multiplying it by `backoff` (capped at `maxInterval`), and re-executes the step.
5. If the maximum `attempts` threshold is reached and assertions are still failing, the step fails, halting the scenario.
