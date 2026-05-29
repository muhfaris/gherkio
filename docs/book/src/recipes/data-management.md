# Database Seeding & Data Strategies

Ensure tests run in complete isolation and leave zero residue by designing robust setup and teardown architectures.

---

## 🏗️ Strategy A: Transient Setup & Teardown (Recommended)

In this model, every scenario creates its own resource during `setup` and deletes it during `teardown`.

```yaml
scenario: Transient Resource Lifecycle
setup:
  - request:
      method: POST
      url: /items
      body: { name: "Temp Book" }
    save:
      bookId: body.id

steps:
  - request:
      method: PUT
      url: /items/$bookId
      body: { name: "Updated Book" }
    expect:
      status: 200

teardown:
  - request:
      method: DELETE
      url: /items/$bookId
    expect:
      status: 200
```
- **Pros**: Zero DB pollution, highly predictable, immune to concurrent execution conflicts.
- **Cons**: Adds additional HTTP calls per test scenario.

---

## 🏗️ Strategy B: Bulk Environmental Seeding

If tests require extensive reference lists (e.g. currency maps, country codes, permissions catalogues), create an environmental seeding script.

Map this initializer test at the very beginning of your pipeline:

```yaml
# tests/00-seed.yaml
scenario: Seed Database
steps:
  - request:
      method: POST
      url: /admin/seed
      headers:
        X-Seed-Key: $accounts.admin.password
    expect:
      status: 200
```

Configure your test orchestrator to execute this scenario sequentially first, before running the remaining tests in parallel:

```bash
# Seed sequentially
gherkio run .gherkio/tests/00-seed.yaml

# Run tests in parallel
gherkio run .gherkio/tests/ -p 4
```
- **Pros**: Keeps individual scenario scripts small and extremely fast.
- **Cons**: Shared state must be read-only during parallel steps to prevent test interference.
