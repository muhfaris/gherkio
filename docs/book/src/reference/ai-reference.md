# AI and Machine-Readable Reference

This page is the compact reference for generating valid Gherkio tests. The detailed DSL chapters remain canonical for explanations and edge cases. Historical RFCs are not current syntax references.

## Minimal Scenario

```yaml
scenario: Fetch user
tags: [smoke, users]
steps:
  - name: Fetch users
    request:
      method: GET
      url: /users
      query:
        page: 1
    expect:
      status: 200
      body.data: array
    save:
      users: body.data

  - name: Select a user
    set:
      SELECTED_USER: ${randomItem(users)}

  - name: Fetch selected user
    if: $SELECTED_USER.id
    request:
      method: GET
      url: /users/$SELECTED_USER.id
    expect:
      status: 200
```

Top-level fields are `scenario`, `description`, `tags`, `setup`, `steps`, `teardown`, and `examples`. `steps` is required. Setup runs before the main workflow; failed setup skips the main workflow; teardown always runs.

## Step Operations

A step performs one primary operation:

- `request:` sends HTTP.
- `redis:` runs one controlled read-only Redis command.
- `set:` assigns variables without I/O.
- `use:` composes another scenario; `with:` supplies local overrides.
- `repeat:` runs nested `steps` up to `attempts` times and checks `until` after each successful block.

All operation types support `name` and `if`. Request and Redis steps also support `expect`, `save`, `timing`, and `retry` where applicable.

For a multi-step polling cycle, use a bounded repeat block:

```yaml
- name: Find unused candidate
  repeat:
    attempts: 20
    until: $existingCount == 0
    steps:
      - set:
          candidate: ${randomItem(candidates)}
      - request:
          method: GET
          url: /items
          query: { candidate_id: $candidate.id }
        expect: { status: 200 }
        save: { existingCount: count(body.data) }
```

Variables written by nested steps remain available after success. Any nested
failure stops the loop, and unmet `until` after the final attempt fails it.

Negate a simple condition by quoting it:

```yaml
- name: Authenticate when token is absent
  if: "!$accessToken"
  use: shared/auth/auth.yaml
```

## HTTP Request

```yaml
request:
  service: billing             # optional named environment service
  method: POST                 # GET, POST, PUT, DELETE, PATCH
  url: /v1/orders
  query:
    include: [customer, items] # arrays create repeated query keys
  headers:
    Authorization: Bearer $accessToken
  body:
    user_id: $SELECTED_USER.id
  timeout: 30s
```

Use `multipart` instead of `body` for uploads:

```yaml
request:
  method: POST
  url: /attachments
  multipart:
    fields:
      title: evidence
    files:
      attachment:
        path: documents/evidence.pdf
        contentType: application/pdf
        filename: evidence.pdf
```

Relative file names are checked against `assets.path` after the project-relative path. When `contentType` is omitted, the multipart writer uses `application/octet-stream`.

## Variables and Functions

- `$name`, `${name}`: variable reference.
- `${name:default}`: fallback when missing.
- `$accounts.<name>.<field>`: credential value.
- `$value.path`, `$items[0].id`: nested paths.
- `$string(value)`, `$int(value)`, `$float(value)`, `$bool(value)`: request-body casts.
- `$uuid`, `$ulid`, `$randomInt`, `$randomEmail`, `$randomPhone`, `$timestamp`, `$timestampMs`.
- `${randomInt(min,max)}`.
- `${randomItem(array[,field])}`; an exact `${randomItem(array)}` in `set:` preserves the selected object.
- `${dateNow(format)}`, `${dateOffset(duration,format)}`.
- `${base64(string)}`, `${base64Decode(encoded)}`, `${urlencode(string)}`, `${urldecode(encoded)}`.
- `${hash(algo,data)}`, `${hmac(algo,key,message)}`.
- `${randomString(length,charset)}`, `${toUpper(string)}`, `${toLower(string)}`, `${trim(string)}`.
- `${trimPrefix(value,prefix)}`, `${trimSuffix(value,suffix)}`, `${split(value,delimiter,index)}`.
- `$if(condition,thenValue,elseValue)` inside transform selections.

## Assertions and Saved Values

```yaml
expect:
  status: 200
  body.id: uuid
  body.email: email
  body.name: contains Gramedia
  body.role: oneOf admin,manager
  body.deleted_at: not exists
  count(body.items): 3
  all(body.items): object
timing:
  max: 500ms
save:
  userId: body.id
  itemCount: count(body.items)
```

Canonical paths include `body.<field>`, `headers.<name>`, `jwt.<claim>`, and `redis.<field>`. Supported matchers include `exists`, `not exists`, `uuid`, `email`, `datetime`, `uri`, `string`, `number`, `boolean`, `array`, `object`, `null`, `true`, `false`, `contains`, `startsWith`, `endsWith`, `regex`, `oneOf`, `in`, `gt`, `gte`, `lt`, `lte`, `empty`, `ipv4`, `ipv6`, `base64`, and `mac`.

In `save:`, `count(body.<path>)` stores an array length. Empty arrays and explicit `null` values store `0`; missing paths and non-array values warn and are not stored.

## Retry

```yaml
retry:
  attempts: 5
  interval: 500
  backoff: exponential          # constant, linear, exponential
  maxDuration: 15s
  onStatus: [404, 409]
```

## Redis

Environment:

```yaml
connections:
  cache:
    type: redis
    sentinel:
      master: mymaster
      addresses: [sentinel-1:26379, sentinel-2:26379]
    username: $REDIS_USERNAME
    password: $REDIS_PASSWORD
    database: 0
```

Step:

```yaml
- name: Verify cached user
  redis:
    connection: cache
    command: get               # get, exists, ttl, hgetall
    key: user:$userId
  expect:
    redis.exists: true
    redis.value.id: $userId
```

## Execution Models

```bash
# Different test files concurrently
gherkio run .gherkio/tests --parallel 4

# Same workflow: two isolated users, three sequential iterations each
gherkio run workflow.yaml --virtual-users 2 --iterations-per-user 3

# Reports
gherkio run workflow.yaml --report html,json
```

`examples:` is data-driven scenario iteration. Virtual-user mode cannot be combined with `examples`, directory runs, `--parallel`, `--all-accounts`, or partial-step selection.

## Common Invalid Syntax

- Do not use unsupported report flags such as `--report-html`, `--report-json`, or `--report-junit`; use `--report`.
- Do not use a fixed random index when the array length comes from a response; use `randomItem`.
- Do not select an object twice to obtain related fields; store one `${randomItem(array)}` object and access its fields.
- Do not configure both `address` and `sentinel` for one Redis connection.
- Do not use Redis mutation commands; Redis steps are intentionally read-only.
