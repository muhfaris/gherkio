# Redis Cache Checks

Gherkio can verify application cache state in the same scenario that exercises
the HTTP API. Redis access is deliberately read-only: scenarios can inspect
keys, but cannot create, update, or delete production data.

Define the connection once in your active environment file (`.gherkio/environments/<env>.yaml`), then reference its connection name from your `redis` scenario step.

---

## 🔌 Defining Redis Connections

Redis connection configurations belong in environment files (`.gherkio/environments/local.yaml`, `staging.yaml`, etc.) under the `connections:` block. This keeps host IPs, ports, and credentials out of your test scenario files.

### 1. Standalone Redis Connection

Add a named connection entry to `.gherkio/environments/local.yaml`:

```yaml
baseUrl: http://localhost:8080

connections:
  application-cache:
    type: redis
    address: localhost:6379
    database: 0
    password: $GHERKIO_REDIS_PASSWORD   # Optional: Supports GHERKIO_ env vars
    timeout: 5s
```

### 2. Redis Sentinel Connection (High Availability)

For Sentinel clusters, replace `address` with a `sentinel` block:

```yaml
connections:
  application-cache:
    type: redis
    sentinel:
      master: mymaster
      addresses:
        - sentinel-1:26379
        - sentinel-2:26379
        - sentinel-3:26379
      username: sentinel-user
      password: $GHERKIO_SENTINEL_PASSWORD
    username: redis-user
    password: $GHERKIO_REDIS_PASSWORD
    database: 0
    timeout: 5s
```

---

## 📝 Complete API and Cache Example

Once your connection (`application-cache`) is declared in your environment file, reference it directly inside your scenario step:


```yaml
scenario: Product API populates Redis cache
steps:
  - name: Fetch product from the API
    request:
      method: GET
      url: /v1/products/42
    expect:
      status: 200
      body.data.id: exists
    save:
      productId: body.data.id
      productName: body.data.name

  - name: Verify the API cached the product
    redis:
      connection: application-cache
      command: get
      key: "product:$productId"
    expect:
      redis.exists: true
      redis.value.id: "$productId"
      redis.value.name: "$productName"
    save:
      cachedProduct: redis.value
      cachedProductName: redis.value.name
    timing:
      max: 500ms
    retry:
      attempts: 10
      interval: 200
      backoff: constant
      maxDuration: 5s
```

The HTTP step saves values from its response. The Redis key and assertions can
use those values normally. If a `GET` value contains valid JSON, Gherkio decodes
it automatically, which makes nested paths such as `redis.value.id` available.
Non-JSON values remain strings.

## Redis Step Fields

| Field | Required | Description |
| :--- | :---: | :--- |
| `connection` | Yes | Named entry under `connections` in the active environment. Supports variable interpolation. |
| `command` | Yes | One of `get`, `exists`, `ttl`, or `hgetall`. |
| `key` | Yes | Redis key to inspect. Supports normal variables and runtime expressions. |

A `redis` step can also use the standard step fields `name`, `if`, `expect`,
`save`, `timing`, and `retry`. It is mutually exclusive with `request`, `set`,
and `use` in the same step.

## Commands and Result Paths

| Command | Purpose | Available assertion/save paths |
| :--- | :--- | :--- |
| `get` | Read a string value, automatically decoding JSON when possible. | `redis.exists`, `redis.value`, `redis.value.<path>` |
| `exists` | Check whether a key exists. | `redis.exists` |
| `ttl` | Read the remaining TTL in seconds using Redis TTL semantics. | `redis.ttl` |
| `hgetall` | Read all fields from a hash as an object. | `redis.exists`, `redis.value`, `redis.value.<field>` |

All normal Gherkio value matchers work on these paths.

### Check Existence

```yaml
- name: Session cache must exist
  redis:
    connection: application-cache
    command: exists
    key: "session:$sessionId"
  expect:
    redis.exists: true
```

### Check TTL

```yaml
- name: Session must still have a useful lifetime
  redis:
    connection: application-cache
    command: ttl
    key: "session:$sessionId"
  expect:
    redis.ttl: gte 60
```

Redis returns `-1` when a key exists without an expiry and `-2` when the key
does not exist. Gherkio preserves those values in `redis.ttl`.

### Check a Hash

```yaml
- name: Verify cached user hash
  redis:
    connection: application-cache
    command: hgetall
    key: "user:$userId"
  expect:
    redis.exists: true
    redis.value.id: "$userId"
    redis.value.status: active
  save:
    cachedStatus: redis.value.status
```

Redis hash field values are returned as strings.

## Polling Eventually Consistent Caches

Cache population may happen asynchronously. Add `retry` to rerun the Redis
operation until every assertion passes or the retry limit is reached:

```yaml
- name: Wait for the worker to populate the cache
  redis:
    connection: application-cache
    command: get
    key: "ticket:$ticketId"
  expect:
    redis.exists: true
    redis.value.id: "$ticketId"
  retry:
    attempts: 20
    interval: 250
    backoff: exponential
    maxDuration: 10s
```

Redis retries support `attempts`, `interval`, `backoff`, and `maxDuration`.
`onStatus` is HTTP-specific and does not apply to Redis operations.

## Sentinel Connections

Scenario syntax does not change when the environment uses Redis Sentinel:

```yaml
- name: Verify ticket cache through Sentinel
  redis:
    connection: application-cache
    command: get
    key: "ticket:$ticketId"
  expect:
    redis.exists: true
```

For every attempt, Gherkio asks Sentinel for the current primary before opening
the Redis connection. A retry can therefore discover a new primary after a
failover. See [Redis Connections](../reference/environments.md#redis-connections)
for Sentinel authentication and TLS configuration.

## Safety Boundary

Only `GET`, `EXISTS`, `TTL`, and `HGETALL` are supported. Gherkio intentionally
does not expose arbitrary Redis commands, Lua scripts, writes, deletes, or key
scanning. This keeps cache verification declarative and safe for shared test
environments.
