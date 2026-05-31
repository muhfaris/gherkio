package mcp

import (
	"encoding/json"

	"github.com/muhfaris/gherkio/internal/runner"
)

// buildMatchersResource returns JSON describing all available assertion matchers.
// Fully dynamic — uses runner.GetMatchersInfo() as the single source of truth.
func (s *Server) buildMatchersResource() string {
	matcherInfo := runner.GetMatchersInfo()

	type outputInfo struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Usage       string `json:"usage"`
	}

	var out []outputInfo
	for _, m := range matcherInfo {
		usage := m.Name
		if m.HasArg {
			usage = m.Name + " <value>"
		}
		out = append(out, outputInfo{
			Name:        m.Name,
			Description: m.Description,
			Usage:       usage,
		})
	}

	data, _ := json.MarshalIndent(out, "", "  ")
	return string(data)
}

// buildVariablesResource returns JSON describing built-in generator variables.
// Fully dynamic — uses runner.GetVariableInfo() as the single source of truth.
func (s *Server) buildVariablesResource() string {
	vars := runner.GetVariableInfo()
	data, _ := json.MarshalIndent(vars, "", "  ")
	return string(data)
}

// buildPathsResource returns JSON describing canonical assertion paths.
// Fully dynamic — uses runner.GetPathInfo() as the single source of truth.
func (s *Server) buildPathsResource() string {
	paths := runner.GetPathInfo()
	data, _ := json.MarshalIndent(paths, "", "  ")
	return string(data)
}

// buildProjectStructureResource returns markdown describing the .gherkio/ directory layout.
func (s *Server) buildProjectStructureResource() string {
	return `# Gherkio Project Directory Structure

A Gherkio project lives under a .gherkio/ directory in the project root.
Here's what goes where:

## Tests (.gherkio/tests/)
Test scenario files in YAML format.

- **Create**: use create_test tool
- **Read**: use read_test or list_tests tool
- **Update**: use update_test tool
- **Delete**: use delete_test tool

## Credentials (.gherkio/credentials/)
Account credentials per environment.

Format:
- accounts: map of account name to credentials
  - username: account username
  - password: account password (auto-masked)
  - role: account role
  - Any extra fields are passed through

- **Create**: use create_credential tool
- **Read**: use read_credential tool or list_environments tool
- **Update**: use update_credential tool

## Environments (.gherkio/environments/)
Environment configuration with base URL and service overrides.

Format:
- baseUrl: base URL for all requests
- services: optional map of named service overrides
  - <name>: baseUrl for that service

- **Create**: use create_environment tool
- **Read**: use list_environments tool
- **Update**: use update_environment tool

## Schemas (.gherkio/schemas/)
Custom validation schemas for response body assertions.

- **Create**: use create_schema tool
- **Read**: use list_schemas tool
- **Update**: use update_schema tool

## Reports (.gherkio/reports/)
Auto-generated HTML/JSON reports (safe to .gitignore).
`
}

// buildSpecResource returns markdown detailing Gherkio DSL's Grammar Spec.
func (s *Server) buildSpecResource() string {
	return `## Gherkio DSL Grammar Spec

### Structural Keys
- **scenario**: (String, Required) Human readable name of the scenario.
- **setup**: (List of Steps, Optional) Pre-condition HTTP requests or composed files (e.g. login, session setup, data seeding).
- **steps**: (List of Steps, Required) The primary test/execution block.
- **teardown**: (List of Steps, Optional) Post-execution cleanup steps (e.g. deleting created resources).

### Lifecycle Guidelines & Execution Rules
Use setup, steps, and teardown blocks strategically:
1. **Setup**: If any step in setup fails, Gherkio halts immediately, skips the primary 'steps' block, and jumps straight to teardown. Use it only for actions that must succeed before the core test is valid (like authentication).
2. **Steps**: Put the actual user actions/business flow under test inside steps.
3. **Teardown**: The teardown block is *guaranteed* to execute even if setup or steps fail. ALWAYS put cleanup/deletion requests in teardown to prevent test data leaks.

### Step Block
- **use**: (String, Conditional) Path to compose/execute another scenario. Mutually exclusive with request.
- **request**: (Request object, Conditional) HTTP Request config. Mutually exclusive with use.
- **expect**: (Expect object, Optional) Response assertions.
- **save**: (Map of name:path, Optional) Extract dynamic values to context variables. Paths support variable interpolation (e.g. 'body.data[$randomInt(0,9)].id').
- **timing**: (TimingConfig, Optional) Execution latency check.

### Request Config
- **service**: (String, Optional) Named service override matching environments.
- **method**: (String, Required) HTTP Method (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS).
- **url**: (String, Required) Target endpoint url (appends to baseUrl). Supports variable interpolation.
- **headers**: (Map of string:string, Optional) Custom HTTP headers. Supports variable interpolation in values.
- **body**: (Free-form object/string, Optional) Request body content. Supports variable interpolation in string values.
- **transform**: (Map of path:ProjectionConfig, Optional) Declarative collections projected into the request payload.


### Variable Interpolation
All string values in request fields support variable substitution:
- **$var** — Simple variable reference (e.g. $username, $token)
- **${var}** — Explicit braces syntax (e.g. ${accessToken})
- **${var:default}** — With default fallback (e.g. ${role:user})
- **$string(var)**, **$int(var)**, **$bool(var)**, **$float(var)** — Type-casting operators to cast variables in request bodies (e.g. $string(emp_id))
- **$accounts.<name>.<field>** — Access any account's credentials directly from .gherkio/credentials/<env>.yaml without needing --account flag (e.g. $accounts.alice.username)
- **${func(arg1,arg2)}** — Parametrized built-in generator with arguments (e.g. ${randomInt(1,100)})

Variables are sourced from:
1. **Built-in generators** — Pre-populated variables available in every test run:
   - **$uuid** — UUID v4 string (e.g. a1b2c3d4-e5f6-4789-abcd-ef1234567890)
   - **$ulid** — ULID string (e.g. 01ARZ3NDEKTSV4RRFFQ69G5FAV)
   - **$randomInt** — Random integer between 0 and 999999 (e.g. 74291). Use **${randomInt(min,max)}** for custom range (e.g. ${randomInt(1,100)})
   - **$randomEmail** — Random email at @example.com (e.g. user_123456@example.com)
   - **$randomPhone** — Random Indonesian-format phone number (e.g. +6281234567890). Use **${randomPhone(ISO)}** (e.g. SG, JP) or **${randomPhone(prefix)}** (e.g. +351) for global formats.
   - **$timestamp** — Current Unix timestamp in seconds (e.g. 1716942900)
   - **$timestampMs** — Current Unix timestamp in milliseconds (e.g. 1716942900123)
   - **${dateNow(format)}** — Current date/time formatted using custom Go layout, e.g. "2006-01-02" or "2006-01-02T15:04:05Z" (defaults to "2006-01-02 15:04:05")
   - **${dateOffset(offset,format)}** — Calculates current date/time with a duration offset (e.g. "+14d", "-2h", "+30m") and custom layout formatting (e.g. "2006-01-02")
   - **${base64(string)}** — Encodes string to Base64 standard format (e.g. ${base64("hello")})
   - **${base64Decode(encoded)}** — Decodes Base64 string back to plaintext (e.g. ${base64Decode("aGVsbG8=")})
   - **${urlencode(string)}** — Encodes string for safe URL query inclusion (e.g. ${urlencode("hello world")})
   - **${urldecode(encoded)}** — Decodes URL-encoded string back to plaintext (e.g. ${urldecode("hello+world")})
   - **${hash(algo,data)}** — Generates hex-encoded hash using md5, sha1, or sha256 (e.g. ${hash("sha256","secret")})
   - **${hmac(algo,key,message)}** — Generates hex-encoded HMAC using md5, sha1, or sha256 (e.g. ${hmac("sha256","key","msg")})
   - **${randomString(length,charset)}** — Generates random string using charset: alpha, numeric, or alphanumeric (e.g. ${randomString(10,"alphanumeric")})
   - **${toUpper(val)}** — Converts string to UPPERCASE (e.g. ${toUpper("hello")})
   - **${toLower(val)}** — Converts string to lowercase (e.g. ${toLower("HELLO")})
   - **${trim(val)}** — Trims leading and trailing whitespace (e.g. ${trim("  hello  ")})
2. **Credentials** — Account fields from .gherkio/credentials/<env>.yaml (injected automatically when --account is used, or via $accounts.<name>.<field>)
3. **Step saves** — Values extracted from previous step responses via save: blocks
4. **Saved vars override credentials** — When a step saves a variable with the same name as a credential

Built-in variables can be overridden by credentials or step saves with the same name.

### Declarative Collection Projections (Transform)
Reshapes source array collections from the variables pool directly into the request payload path using standard projection rules:
- **from**: (String, Required) Source collection array variable name (must start with $).
- **as**: (String, Optional) Scoped alias variable name to represent each item during transformation (defaults to 'item').
- **where**: (Map of field:matcher, Optional) Filter conditions applied using Gherkio assertions/matchers.
- **limit**: (Integer, Optional) Maximum count of projected items to return.
- **select**: (Map, Required) Structural projection mapping. Can define nested mapping or further sub-projections.

**Explicit Type Casting:**
You can coerce field types during selection by wrapping variable paths in casting operators:
- **$string(var)** — Converts the field value directly to a string.
- **$int(var)** — Parses/coerces the field value directly to an integer.
- **$float(var)** — Parses/coerces the field value directly to a float.
- **$bool(var)** — Parses/coerces the field value directly to a boolean.

**Example:**

    transform:
      survey.answers:
        from: $raw_questions
        select:
          question_id: "$string(item.question_id)"
          is_answered: true

### Assertions (Expect)
- **status**: (Integer) Expected HTTP status (e.g. 'status: 200').
- **body.<path>**: Assert on JSON body fields using a matcher or literal value (e.g. 'body.id: exists', 'body.name: Emily').
- **headers.<name>**: Assert on response header values (e.g. 'headers.content-type: contains application/json').
- **jwt.<claim>**: Assert on decoded JWT claims (e.g. 'jwt.role: admin').
- **schema**: Validate full body against a YAML schema file in .gherkio/schemas/ (e.g. 'schema: user-profile').
  Negative form: 'schema: not <name>' asserts the response does NOT match the schema.

**Available Matchers:**
- 'exists' / 'not exists' — Field present / absent
- 'uuid', 'email', 'datetime', 'uri' — Format validators
- 'string', 'number', 'boolean', 'array', 'object', 'null', 'true', 'false' — Type checkers
- 'empty' — String, array, or object is empty
- 'contains <substring>', 'startsWith <prefix>', 'endsWith <suffix>' — String matchers
- 'regex <pattern>' — Regex match
- 'gt <N>', 'gte <N>', 'lt <N>', 'lte <N>' — Numeric comparisons
- 'ipv4', 'ipv6', 'base64', 'mac' — Format validators

**Collection Matchers (for arrays):**
- 'count(<path>): <N>' — Array has exactly N items (e.g. 'count(body.items): 3')
- 'count(<path>).gte: <N>' — Array has >= N items (e.g. 'count(body.items).gte: 1' means "has data")
- 'count(<path>).gt: <N>' — Array has > N items
- 'count(<path>).lte: <N>' — Array has <= N items
- 'count(<path>).lt: <N>' — Array has < N items
- 'all(<path>): <matcher>' — Every element matches (e.g. 'all(body.items.status): active')
- 'all(<path>.<field>): <matcher>' — Every element's field matches (e.g. 'all(body.items.id): uuid')

**Examples:**

    expect:
      status: 200
      body.data: exists
      body.token: uuid
      body.items: array
      body.email: email
      body.role: admin          # literal equality
      body.count: gt 10         # numeric > 10
      body.name: contains John
      count(body.items): 5      # exactly 5 items
      count(body.items).gte: 1  # at least 1 item (has data)
      schema: user-profile
      schema: not error-payload
`
}

// buildExamplesResource returns Gherkio DSL integration examples.
func (s *Server) buildExamplesResource() string {
	return `# Basic example: login with inline credentials
scenario: login and fetch profile

steps:
  - request:
      method: POST
      url: /auth/login
      body:
        username: emilys
        password: emilyspass
    expect:
      status: 200
      body.accessToken: exists
    save:
      authToken: body.accessToken

  - request:
      method: GET
      url: /auth/me
      headers:
        Authorization: Bearer $authToken
    expect:
      status: 200
      body.username: emilys

---
# Multi-account example: access any account without --account flag
# Uses $accounts.<name>.<field> from .gherkio/credentials/local.yaml
scenario: login as specific account via $accounts

steps:
  - request:
      method: POST
      url: /auth/login
      body:
        username: $accounts.alice.username
        password: $accounts.alice.password
        expiresInMins: 30
    expect:
      status: 200
      body.accessToken: exists
    save:
      accessToken: body.accessToken

  - request:
      method: GET
      url: /auth/me
      headers:
        Authorization: Bearer $accessToken
    expect:
      status: 200
      body.role: $accounts.default.role

---
# Built-in generators example: $uuid, $ulid, $randomInt
# These variables are available in every test with no setup needed
scenario: using built-in generators

steps:
  - request:
      method: POST
      url: /auth/login
      body:
        username: $accounts.default.username
        password: $accounts.default.password
        idempotencyKey: $uuid
        requestId: $ulid
        otpCode: $randomInt
    expect:
      status: 200
      body.accessToken: exists
    save:
      accessToken: body.accessToken

  - request:
      method: GET
      url: /auth/me
      headers:
        Authorization: Bearer $accessToken
        X-Idempotency: $uuid
    expect:
      status: 200
      body.username: $accounts.default.username

---
# Parametrized randomInt example: custom range with ${randomInt(min,max)}
# Also demonstrates count().gte for checking array has data
scenario: parametrized randomInt and array length check

steps:
  - request:
      method: POST
      url: /products
      body:
        name: "Product ${randomInt(1000,9999)}"
        price: ${randomInt(1000,500000)}
        quantity: ${randomInt(1,100)}
    expect:
      status: 201
      count(body.items).gte: 1
    save:
      productId: body.id
      sku: body.sku

---
# Lifecycle example: robust setup, steps, and guaranteed teardown
# Guaranteed to delete the created resource even if steps fail
scenario: manage transient user profile
setup:
  - request:
      method: POST
      url: /auth/login
      body:
        username: $accounts.default.username
        password: $accounts.default.password
    expect:
      status: 200
    save:
      accessToken: body.accessToken

  - request:
      method: POST
      url: /users
      headers:
        Authorization: Bearer $accessToken
      body:
        name: "Test User ${randomInt(100,999)}"
        email: $randomEmail
    expect:
      status: 201
    save:
      newUserId: body.id

steps:
  - request:
      method: GET
      url: /users/$newUserId
      headers:
        Authorization: Bearer $accessToken
    expect:
      status: 200
      body.name: startsWith Test User

teardown:
  - request:
      method: DELETE
      url: /users/$newUserId
      headers:
        Authorization: Bearer $accessToken
    expect:
      status: 200
`
}
