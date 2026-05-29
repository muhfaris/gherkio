# Setup & Teardown

Managing database seeds, cleanups, and dynamic pre-conditions is critical for reliable integration tests. Gherkio provides native `setup` and `teardown` lifecycle blocks to isolate environment mutations.

---

## 🔁 Lifecycle Execution Path

Gherkio follows a strict execution guarantee for test scenarios:

```text
                  ┌────────────────────┐
                  │   Start Scenario   │
                  └─────────┬──────────┘
                            │
                            ▼
                  ┌────────────────────┐
                  │ Execute SETUP Steps│
                  └─────────┬──────────┘
                            │
              ┌─────────────┴─────────────┐
              │                           │
      Success │                   Failure │
              ▼                           ▼
    ┌──────────────────┐        ┌──────────────────┐
    │   Execute Main   │        │    Skip Main     │
    │      STEPS       │        │      STEPS       │
    └─────────┬────────┘        └─────────┬────────┘
              │                           │
              └─────────────┬─────────────┘
                            │
                            ▼
                  ┌────────────────────┐
                  │  Execute TEARDOWN  │
                  │ Steps (Guaranteed) │
                  └─────────┬──────────┘
                            │
                            ▼
                  ┌────────────────────┐
                  │    End Scenario    │
                  └────────────────────┘
```

- **Setup Failures**: If any step in the `setup` block fails, Gherkio halts execution immediately, skips the primary `steps` block entirely (marking the scenario as failed), and jumps directly to the `teardown` block.
- **Teardown Guarantees**: The `teardown` block is **guaranteed to run** regardless of whether the `setup` block or the primary `steps` block succeeded or failed. This ensures database cleanups, session invalidations, or resource releases are always performed.

---

## 📝 Lifecycle Configuration Example

Here is a typical scenario writing a temporary resource, asserting against it, and then cleaning it up:

```yaml
scenario: Manage User Profile
setup:
  - use: shared/authenticate.yaml   # Get auth token
  - request:
      method: POST
      url: /profiles
      headers:
        Authorization: "Bearer ${token}"
      body:
        username: "temp_user"
    save:
      profileId: body.id

steps:
  - request:
      method: GET
      url: /profiles/$profileId
      headers:
        Authorization: "Bearer ${token}"
    expect:
      status: 200
      body.username: temp_user

teardown:
  - request:
      method: DELETE
      url: /profiles/$profileId     # Clean up target profile
      headers:
        Authorization: "Bearer ${token}"
    expect:
      status: 200
```
