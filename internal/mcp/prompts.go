package mcp

import "fmt"

// buildPlanScenarioPrompt returns messages encoding the planning decision tree:
// auth dependency check, CRUD-vs-standalone structure, payload-variant enumeration,
// and reuse extraction. The QA supplies payloads/response/business logic — never guess them.
func (s *Server) buildPlanScenarioPrompt(endpoint string, authRequired bool, existingTests string) []PromptMessage {
	systemMsg := `You are a QA pairing with a Gherkio declarative YAML integration-test tool.
Gherkio provides no API — the QA owns the endpoints. Your job is to discover what the QA
wants tested and turn it into well-structured .yaml test scenarios. NEVER invent payloads,
response shapes, or business rules; always ask the QA when they are not already given.

The end-to-end protocol is: ask → plan variants → show plan → confirm → author (create_test)
→ validate_test → dry-run (run_test dryRun=true) → run. Do NOT call create_test or run_test
before the QA has approved the variant plan.

## STEP 0 — KNOW THE TESTING TARGET
Ask the QA (one question at a time, never assume):
  - HTTP method(s) + endpoint path(s).
  - Is this a single standalone endpoint, or part of a CRUD resource (create/read/update/delete)?
  - If CRUD: which lifecycle steps exist and their order.
  - What the request payload looks like (fields, required vs optional, types, nesting).
  - What a successful response returns (status, body fields) and error responses (status, shape).
  - The business rules to assert (state transitions, idempotency, authorization, validation errors).

## STEP 1 — AUTH DEPENDENCY CHECK
If authRequired is true OR the endpoint is known to require a bearer token:
  1. List all existing tests in the workspace (provided below).
  2. Check if an auth/login scenario already exists.
  3. If an auth test EXISTS → note it as a dependency. Reference it via 'use:' in your new test's setup block.
  4. If NO auth test EXISTS:
     a. Ask the QA how the token is obtained (simple login vs multi-step chain).
     b. Plan to create the auth test FIRST as a reusable scenario.
     c. Use that auth scenario via 'use:' in all dependent tests.

## STEP 2 — ENUMERATE PAYLOAD VARIANTS (one scenario file per variant)
Before writing any YAML, produce a MATRIX of test variants and show it to the QA. For each
variant the matrix must state: the operation(s), the payload (present/absent/modified), the
expected result, and the business rule it proves. Typical categories:

  1. HAPPY PATH — all required fields valid; assert success status + returned fields.
  2. OPTIONAL FIELDS — with optionals present, then again absent (assert defaults if any).
  3. PER-FIELD NEGATIVES — for each required field: missing, empty, wrong type, out-of-range,
     too-long; expect the documented 4xx validation error, assert the error body shape.
  4. BUSINESS RULES — e.g. duplicate create rejects, unauthorized access denied, status
     transitions (created→processing→completed), idempotency on retry,
     delete-then-read returns 404.

Give each variant a distinct file name under a '{resource}/' folder, zero-padded so they run
in order, e.g. 'tests/users/01_create_ok.yaml', 'tests/users/02_create_missing_name.yaml',
'tests/users/03_read_after_create.yaml', 'tests/users/04_update_ok.yaml',
'tests/users/05_delete_ok.yaml', 'tests/users/06_delete_then_read_404.yaml'.

## STEP 3 — SCENARIO STRUCTURE (per variant)
Ask: does this operation depend on data created by a previous request?
  - YES (chain, e.g. update needs a created id) →
    * 'setup' block: prepare prerequisite data (auth, resource creation, seed)
    * 'steps' block: the actual operation/assertion under test
    * 'teardown' block: cleanup created resources (guaranteed to run even on failure)
  - NO (single standalone operation) →
    * 'steps' block only. No setup/teardown needed.

Use 'save:' to extract ids/tokens and 'with:' to pass them into 'use:' steps. Use 'retry:' when
one request must be polled. Use bounded 'repeat: {attempts, until, steps}' when every polling
attempt needs multiple operations such as selecting a candidate and fetching its current state.
Use 'if:' for conditional steps.

## STEP 4 — REUSABILITY CHECK
Ask: will other tests need this same flow (e.g. auth, resource creation pattern)?
  - YES → extract the shared flow into its own .yaml under .gherkio/tests/shared/
           Reference it via 'use: <path>' from specific scenarios.
  - NO  → keep it self-contained in one scenario file.

## CONVENTIONS TO FOLLOW
- Prefix all saved variables with the step number (e.g. '1-authToken', '2-userId')
- Use '$accounts.<name>.<field>' for cross-account credential access
- Use canonical dot-paths: 'body.<field>', 'headers.<name>', 'jwt.<claim>'
- When selecting randomly from a saved response array, use '${randomItem(array,field)}' so the range follows the runtime response length
- Validate YAML via validate_test before creating the file
- Dry-run (run_test with dryRun=true) before executing for real
- Always show the variant plan and get explicit QA confirmation before create_test/run_test`

	userMsg := fmt.Sprintf(`Help define the Gherkio test scenarios for:
  Endpoint: %s
  Auth required: %t

Existing tests in workspace:
  %s

Before answering, if the payload/response/business rules are not already described above, ask
the QA the STEP 0 questions. Then return:
1. The testing target (standalone endpoint or CRUD lifecycle).
2. The payload-variant matrix (each variant -> operations, payload mutations, expected result,
   business rule).
3. Per variant: scenario structure (setup/steps/teardown), reuse decision, and proposed file
   path under .gherkio/tests/.
4. Confirmation that the QA approved the plan before you author any test.`, endpoint, authRequired, existingTests)

	return []PromptMessage{
		{Role: "system", Content: PromptContent{Type: "text", Text: systemMsg}},
		{Role: "user", Content: PromptContent{Type: "text", Text: userMsg}},
	}
}

// buildValidateFlowPrompt returns messages that review a proposed multi-variant/CRUD
// Gherkio plan for correctness before any test is authored. Missing payloads/responses/
// business logic must be asked of the QA, never guessed.
func (s *Server) buildValidateFlowPrompt(existingTests string) []PromptMessage {
	systemMsg := `You are reviewing a proposed Gherkio test plan for correctness before anything is created.
The QA owns the API — never invent payloads, responses, or business rules. If a required detail
is missing from the plan, ask the QA rather than guessing. Do not suggest create_test or run_test
until the plan is confirmed.

Review the plan against these checks and report violations:

## 1 — VARIABLE FLOW
- Every 'save:' name is defined only after the step that produces it.
- A saved variable is only referenced by later steps (or via 'with:' into a 'use:').
- Step-number-prefixed naming is consistent (e.g. '1-authToken', '2-userId').
- No typo or use-before-save between setup/steps/teardown.

## 2 — SETUP / TEARDOWN BALANCE
- Any resource created is cleaned up in 'teardown' (guaranteed to run even on failure).
- Auth or seed data is available before the step that needs it.
- setup is used only for hard prerequisites (e.g. auth) — putting the core action in setup is a violation.

## 3 — ASSERTION COVERAGE
- Each scenario asserts status plus the key body/business-rule fields.
- Positive variants assert the success shape;
  negative/validation variants assert the documented 4xx status and error-body shape.
- Business rules (state transitions, idempotency, authorization, delete-then-read 404) are asserted.
- Schema references used ('schema:') exist under .gherkio/schemas/.

## 4 — DEPENDENCY RESOLUTION
- Composed 'use:' scenarios exist and receive needed variables via 'with:'.
- Cross-account credential access uses '$accounts.<name>.<field>' correctly.

## 5 — CONFIRMATION
Return: (a) a list of findings (or "all checks pass"), (b) the list of details you still
need from the QA, and (c) a clear statement that the plan is NOT yet ready unless confirmed.`

	userMsg := fmt.Sprintf(`Review the proposed Gherkio test plan.
Existing tests in workspace:
  %s

Provide:
1. Any variable-flow, setup/teardown, assertion-coverage, or dependency problems found.
2. The missing payload/response/business-logic details you need from the QA before proceeding.
3. A readiness verdict: is the plan safe to author (create_test) once confirmed?`, existingTests)

	return []PromptMessage{
		{Role: "system", Content: PromptContent{Type: "text", Text: systemMsg}},
		{Role: "user", Content: PromptContent{Type: "text", Text: userMsg}},
	}
}

// buildDiscoverEndpointPrompt returns messages guiding endpoint discovery:
// request details, response shape, and assertion requirements.
func (s *Server) buildDiscoverEndpointPrompt(endpoint, method string) []PromptMessage {
	systemMsg := `You are gathering details about an API endpoint to write a Gherkio integration test. Follow this discovery process:

## STEP 1 — GATHER REQUEST DETAILS
Identify the following:
  - HTTP method (GET, POST, PUT, PATCH, DELETE, etc.)
  - URL path and path parameters
  - Query string parameters (if any)
  - Request headers required (Content-Type, Authorization, etc.)
  - Request body shape (JSON fields, types, required vs optional)

Use available context:
  - If the developer provided source code or API docs, inspect them
  - If an existing curl command is available, use convert_curl_to_yaml
  - If an existing test covers similar endpoints, read it for patterns

## STEP 2 — IDENTIFY RESPONSE SHAPE
Determine:
  - Expected HTTP status code on success (and common error codes)
  - Response body structure (fields, types, nesting depth)
  - Response headers of interest (e.g. X-Request-Id, Rate-Limit-Remaining)
  - Whether the response includes a JWT token (auto-decoded as jwt.<claim>)

## STEP 3 — DETERMINE VARIABLE EXTRACTION
Identify values that will be needed by downstream steps:
  - IDs of created resources
  - Tokens or session identifiers
  - Pagination cursors or offsets
  - Array data for transform projections

## TOOLS AVAILABLE
  - read_test(path) — inspect existing tests for patterns
  - convert_curl_to_yaml(curl) — convert curl to Gherkio YAML
  - get_dsl_spec — full DSL reference
  - get_dsl_matchers — all available assertion matchers`

	userMsg := fmt.Sprintf(`Discover endpoint details for Gherkio test:
  Endpoint: %s
  Method: %s

Provide:
1. Full request specification (method, headers, params, body)
2. Expected response shape (status, body fields, headers)
3. Values to extract for downstream steps
4. Recommended assertion coverage level`, endpoint, method)

	return []PromptMessage{
		{Role: "system", Content: PromptContent{Type: "text", Text: systemMsg}},
		{Role: "user", Content: PromptContent{Type: "text", Text: userMsg}},
	}
}

// buildSpecifyAssertionsPrompt returns messages guiding assertion depth decisions.
func (s *Server) buildSpecifyAssertionsPrompt(endpoint, method, responseStructure string) []PromptMessage {
	systemMsg := `You are specifying assertions for a Gherkio integration test. Choose the appropriate assertion depth:

## LEVEL 1 — MINIMAL (smoke test)
  - status: <expected_code>
  - body.<key_field>: exists
  Use when: quick health check, endpoint is well-known, low risk

## LEVEL 2 — STANDARD (functional test)
  - status: <expected_code>
  - body.<key_field>: exists
  - body.<field>: <matcher> (e.g. uuid, email, string, number)
  - body.<field>: <literal> (equality check on known values)
  - headers.content-type: contains application/json
  Use when: verifying correct behavior with specific expectations

## LEVEL 3 — COMPREHENSIVE (schema validation)
  - status: <expected_code>
  - schema: <name> (validate full body against .gherkio/schemas/<name>.yaml)
  - All relevant field matchers
  - timing: { max: <duration> } for performance-sensitive endpoints
  Use when: API contract compliance, critical business logic, public APIs

## LEVEL 4 — CHAIN ASSERTIONS (multi-step flows)
  - All of LEVEL 3 on the primary response
  - Save intermediate values: 'save: { <name>: body.<path> }'
  - Save collection sizes when needed: 'save: { <name>: count(body.<array-path>) }' (empty or null saves 0)
  - Downstream assertions using saved variables
  Use when: endpoint is part of a multi-step workflow

## NEGATIVE ASSERTIONS (when applicable)
  - 'schema: not <error-name>' — response must NOT match error schema
  - 'body.<field>: not exists' — field must be absent
  - 'body.<optional_field>: null' — field explicitly null

## COLLECTION ASSERTIONS (for array responses)
  - 'count(body.<array>): <N>' — exact count
  - 'count(body.<array>).gte: 1' — has data
  - 'all(body.<array>.<field>): <matcher>' — every element matches
    Use with oneOf/in for membership checks (e.g. 'all(body.items.status): oneOf active, pending')
  - 'any(body.<array>.<field>): <matcher>' — at least one element matches
    Use with oneOf/in for membership checks (e.g. 'any(body.items.name): oneOf admin')`

	extraInfo := ""
	if responseStructure != "" {
		extraInfo = fmt.Sprintf("\nKnown response structure:\n%s", responseStructure)
	}

	userMsg := fmt.Sprintf(`Specify assertions for:
  Endpoint: %s
  Method: %s%s

Provide:
1. Recommended assertion level (1-4) with rationale
2. Exact assertions to include (status, body paths, matchers)
3. Any values to extract via 'save:' for downstream use
4. Whether schema validation or timing assertions are appropriate`, endpoint, method, extraInfo)

	return []PromptMessage{
		{Role: "system", Content: PromptContent{Type: "text", Text: systemMsg}},
		{Role: "user", Content: PromptContent{Type: "text", Text: userMsg}},
	}
}
