# Variable Assignment (`set`)

`set:` is one of Gherkio's eight primary step operations. It writes variables into the runtime context **without performing any I/O** — no HTTP request, no Redis command, no file access.

Reach for it when you need to:

- Seed deterministic fixtures instead of hardcoding values in five different requests.
- Override a saved or credential value for a single local/debug run.
- **Freeze** a generated value (a correlation ID, an idempotency key) so several later steps agree on it.
- Snapshot a decision — such as a random pick — once, so every later step reads the same object.
- Hand state out of a composed scenario, or derive a value per iteration inside a loop.

---

## 🧱 Syntax

```yaml
steps:
  - name: Seed the queue fixture
    set:
      QUEUE_ID: "01KT4EBA37Y"
      REGION: "ap-southeast-1"

  - name: Use the seeded value downstream
    request:
      method: GET
      url: /v1/queues/$QUEUE_ID
    expect:
      status: 200
```text

| Element | Rule |
| :--- | :--- |
| Step key | `set:` — a map of `VARIABLE_NAME: value` |
| Keys | Bare names, **no** `$` prefix. Write `QUEUE_ID`, then reference it as `$QUEUE_ID` |
| Values | **Scalars only** — text with optional interpolation. Nested maps or lists are rejected at load time (`cannot unmarshal !!map into string`) |
| Network | None. A `set` step never calls anything, and has no response to assert |
| Counted as | A normal step in terminal output, HTML reports, and pass/fail totals. A step skipped by `if:` counts as neither a pass nor a failure |

Quote values that look numeric or boolean (`"5"`, `"true"`) so their intent is obvious — see [Value Semantics](#-value-semantics) for how they are typed at run time.

---

## 🔒 Allowed & Forbidden Keys

Every step must define **exactly one** primary operation:

```text
request | redis | use | set | repeat | for_each | export | import
```text

Declaring two of them is rejected before execution:

```text
Step operations 'request', 'redis', 'use', 'set', 'repeat', 'for_each', 'export', and 'import' are mutually exclusive
```text

With `set`, only these side keys are read:

| Key | On a `set` step |
| :--- | :--- |
| `name` | ✅ Labels the step in terminal and HTML output |
| `if` | ✅ Full guard expression. When it evaluates to false the step is skipped and **nothing is set** |
| `request`, `use`, `redis`, `repeat`, `for_each`, `export`, `import` | ❌ Validation error (`mutually_exclusive`) |
| `expect`, `save`, `timing`, `retry`, `with` | ⚠️ Not rejected by the validator, but **silently ignored** — there is no response to assert or extract, and nothing to retry |

> [!IMPORTANT]
> Never pair `set:` with `expect:` or `save:`. The step still passes, which hides the mistake: your assertions never run. Split it into a `set` step followed by the request step that carries the assertions.

---

## 🔤 Value Semantics

- **Interpolation happens at execution time** against the live variable store, so a `set` value can read credentials, `save` results, built-in generators, and values from earlier `set` steps.
- **One `set` block is not ordered.** `set: { A: "1", B: "$A-2" }` is a race — the lookup of `$A` may run before `A` is written. Derive dependent values in a **separate** step.
- **Values are stored as text.** Use the casting helpers (`$int()`, `$float()`, `$bool()`, `$string()`) when a JSON body needs a real number or boolean, and remember that `if:` comparisons already coerce numeric strings, so `if: $COUNT > 3` works with `COUNT: "5"`.
- **An exact `${randomItem(...)}` keeps the runtime type.** When the *entire* trimmed value is `${randomItem(items)}` the selected object is stored (so `$PICK.id` works). `${randomItem(items,id)}` stores only that field. Any expression with surrounding text falls back to ordinary string interpolation.
- **Last write wins.** A later `set:`, a later `save:`, or a `with:` injection under the same name replaces the earlier value ([variable precedence](variables.md#-variable-precedence)).
- **Built-in generators refresh every step**, so `set` is the way to capture `$uuid` / `${randomInt(1,100)}` once and reuse it.

---

## 🍳 Examples

### 1. Seed a deterministic fixture

One place to change a fixture ID, instead of scattering it across requests.

```yaml
steps:
  - name: Seed the catalogue fixture
    set:
      PRODUCT_ID: "1001"
      CURRENCY: "SGD"

  - request:
      method: GET
      url: /products/$PRODUCT_ID
      query:
        currency: $CURRENCY
    expect:
      status: 200
```text

### 2. Override a saved or credential value for one run

Handy when a teammate's environment returns a different ID, or when you want to pin a test to a known record while debugging.

```yaml
steps:
  - request:
      method: GET
      url: /v1/users/me
    save:
      userId: body.data.id

  - name: Pin to the seeded account instead of the live one
    set:
      userId: "01KT4EBA37Y"

  - request:
      method: GET
      url: /v1/users/$userId
    expect:
      status: 200
```text

### 3. Rotate state between steps

Swap a "previous" and "current" pair so a later assertion can compare them. Use two steps — never one block.

```yaml
steps:
  - name: Keep the previous queue ID
    set:
      PREVIOUS_QUEUE_ID: "$QUEUE_ID"

  - name: Move to the next queue ID
    set:
      QUEUE_ID: "02HT5FCA38Z"

  - request:
      method: GET
      url: /v1/queues/$PREVIOUS_QUEUE_ID/events
    expect:
      body.migratedTo: $QUEUE_ID
```text

### 4. Pick one object once, then read many fields

`${randomItem(array)}` on its own line stores the whole object, which prevents the classic bug of picking a *different* record for each field.

```yaml
steps:
  - request:
      method: GET
      url: /v1/partner-statuses
    save:
      statuses: body.data

  - name: Choose a single partner status
    set:
      PICKED: ${randomItem(statuses)}

  - request:
      method: POST
      url: /v1/partners
      body:
        partner_status_id: $PICKED.id
        partner_status_value: $PICKED.value
    expect:
      status: 201
```text

### 5. Guard a default with `if:`

Branch on the current context — role, feature flag, or a value saved earlier — and skip the assignment entirely when the guard is false.

```yaml
steps:
  - request:
      method: POST
      url: /auth/login
    save:
      role: jwt.role

  - name: Give admins a wider page size
    if: $role == "admin"
    set:
      PAGE_SIZE: "100"

  - name: Default page size for everyone else
    if: $role != "admin"
    set:
      PAGE_SIZE: "25"

  - request:
      method: GET
      url: /v1/audit-logs
      query:
        limit: $PAGE_SIZE
    expect:
      status: 200
```text

### 6. Freeze a generated value that must stay identical

`$uuid` is regenerated for every step. If the same correlation ID has to appear in a create call and its follow-up lookup, capture it once.

```yaml
steps:
  - name: Freeze one correlation ID for the whole flow
    set:
      CORRELATION_ID: "$uuid"

  - request:
      method: POST
      url: /v1/orders
      headers:
        X-Correlation-ID: $CORRELATION_ID
      body:
        order_ref: $CORRELATION_ID
    expect:
      status: 201

  - request:
      method: GET
      url: /v1/orders
      query:
        correlation_id: $CORRELATION_ID
    expect:
      status: 200
      body.items: array
```text

### 7. Derive a value inside a `for_each` loop

A `set` nested in a loop body turns the current item into a reusable variable. Variables set inside the loop remain available after it, while the loop alias (`item`) does not.

```yaml
steps:
  - request:
      method: GET
      url: /v1/warehouses
    save:
      warehouses: body.data

  - name: Provision a shelf per warehouse
    for_each:
      from: $warehouses
      as: warehouse
      steps:
        - name: Build the shelf code for this warehouse
          set:
            SHELF_CODE: "${toUpper($warehouse.code)}-01"

        - request:
            method: POST
            url: /v1/warehouses/$warehouse.id/shelves
            body:
              code: "$SHELF_CODE"
          expect:
            status: 201
```text

### 8. Hand state out of a composed scenario

`use:` shares the parent variable store, so anything a composed scenario (or its `set` steps) writes is visible afterwards. `with:` is the opposite: its injected overrides are restored when the composed scenario finishes.

```yaml
# steps/auth/login.yaml
scenario: Login and tag the session
steps:
  - request:
      method: POST
      url: /auth/login
      body:
        username: "$username"
        password: "$password"
    save:
      token: body.accessToken

  - name: Tag the session so the parent can branch on it
    set:
      SESSION_TIER: "gold"
```text

```yaml
# steps/checkout.yaml
scenario: Compose login, then branch on the tier
steps:
  - use: auth/login.yaml
    with:
      username: $accounts.alpha.username

  - name: Only gold sessions get the promo
    if: $SESSION_TIER == "gold"
    set:
      PROMO_CODE: "GOLD-2026"
```text

### 9. Bookkeeping in `setup:` and `teardown:`

`teardown` is guaranteed to run even when earlier steps fail, which makes it a reliable place to set the flag that a cleanup request will read.

```yaml
scenario: Order cleanup bookkeeping

setup:
  - name: Assume nothing was created yet
    set:
      ORDER_ID: ""
      ORDER_CREATED: "false"

steps:
  - request:
      method: POST
      url: /v1/orders
    save:
      ORDER_ID: body.data.id

  - name: Remember there is something to clean up
    set:
      ORDER_CREATED: "true"

  - request:
      method: GET
      url: /v1/orders/$ORDER_ID
    expect:
      status: 200

teardown:
  - name: Delete the order only when it exists
    if: $ORDER_CREATED == "true"
    request:
      method: DELETE
      url: /v1/orders/$ORDER_ID
    expect:
      status: 204
```text

### 10. Compose a value from generators

Anything available to interpolation can be captured, mixed with literals, or nested.

```yaml
steps:
  - name: Seed the run constants
    set:
      REGION: "ap-southeast-1"

  - name: Build a synthetic registration payload once
    set:
      EMAIL: "qa_${randomString(8, \"numeric\")}@example.com"
      PASSWORD: "Pw-${randomInt(100000,999999)}"
      REFERRAL: "${toUpper($REGION)}-${base64(\"signup\")}"
      SIGNED_UP_AT: "${dateNow(\"2006-01-02 15:04:05\")}"
      TRIAL_ENDS: "${dateOffset(\"+14d\")}"

  - request:
      method: POST
      url: /v1/registrations
      body:
        email: "$EMAIL"
        password: "$PASSWORD"
        referral_code: "$REFERRAL"
        signed_up_at: "$SIGNED_UP_AT"
    expect:
      status: 201
```text

Note the two-step split: `REFERRAL` reads `$REGION`, and reading a variable from the *same* block would be a race (see [Value Semantics](#-value-semantics)).

### 11. Feed a `repeat` loop until the target state appears

The nested `set` re-picks a candidate on every attempt, and `until:` is evaluated after each successful block, so the loop exits as soon as the candidate is free.

```yaml
steps:
  - request:
      method: GET
      url: /v1/candidates
    save:
      candidates: body.data

  - name: Find an unused candidate
    repeat:
      attempts: 20
      until: $existingCount == 0
      steps:
        - set:
            CANDIDATE: ${randomItem(candidates)}

        - request:
            method: GET
            url: /v1/bookings
            query:
              candidate_id: $CANDIDATE.id
          expect:
            status: 200
          save:
            existingCount: count(body.data)

  - request:
      method: POST
      url: /v1/bookings
      body:
        candidate_id: $CANDIDATE.id
    expect:
      status: 201
```text

---

## 🖥️ Observability & Dry-Run

- Terminal and report output labels the step with the keys it writes, sorted: `set variables: QUEUE_ID, REGION`.
- `gherkio run --dry-run` behaves identically here — there is no request to suppress, and later steps still resolve `$QUEUE_ID`.
- The written variables are reported as that step's saved variables, so `--verbose` traces show exactly what a `set` contributed.
- A failure to interpolate fails the step (and the run), with a precise message:

```text
Failed to interpolate set variable 'PARTNER': array variable "items" is not defined
```text

---

## ⚠️ Common Mistakes

| Symptom | Cause | Fix |
| :--- | :--- | :--- |
| `Step operations ... are mutually exclusive` | `set:` shares a step with `request:`, `use:`, `redis:`, `repeat:`, `for_each:`, `export:`, or `import:` | Split into two steps |
| `cannot unmarshal !!map into string` | A nested map or list was used as a value | Store scalars only; capture objects with `${randomItem(array)}` or `save:` |
| `undefined variable` on the next step, sometimes | A dependent value was referenced inside the same `set` block | Derive it in a following `set` step |
| Assertions silently pass | `expect:` / `save:` / `retry:` / `timing:` were added to a `set` step | Move them to the request step |
| A random pick returns inconsistent fields | `${randomItem(array,field)}` was used, storing only one field | Store the object with `${randomItem(array)}` and read `$PICK.field` |
| A body field arrives as a string | `set` stores text | Cast at the payload: `$int(COUNT)`, `$bool(FLAG)` |
| A different ID appears per step | `$uuid` / `${randomInt(...)}` regenerate each step | Freeze it once with `set` (example 6) |

---

## 🔗 Related Reading

- [Steps & Actions](steps.md) — the full list of primary operations.
- [Variables & Context](variables.md) — generators, precedence, and interpolation rules.
- [Scenario Composition](composition.md) — what `use:` and `with:` share with the parent.
- [AI & Machine-Readable Reference](../reference/ai-reference.md) — dense form of the same rules for assistants.
