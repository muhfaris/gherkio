package mcp

import "fmt"

// buildPlanScenarioPrompt returns messages encoding the Phase 1 decision tree:
// auth dependency check, setup/teardown vs standalone, and reuse extraction.
func (s *Server) buildPlanScenarioPrompt(endpoint string, authRequired bool, existingTests string) []PromptMessage {
	systemMsg := `You are planning a Gherkio declarative YAML integration test. Follow this decision tree strictly:

## STEP 1 — AUTH DEPENDENCY CHECK
If authRequired is true OR the endpoint is known to require a bearer token:
  1. List all existing tests in the workspace (provided below).
  2. Check if an auth/login scenario already exists.
  3. If an auth test EXISTS → note it as a dependency. Reference it via 'use:' in your new test's setup block.
  4. If NO auth test EXISTS:
     a. Identify the auth API — how is the token obtained?
        - Simple user/password login (POST /auth/login)?
        - Multi-step chain (login → exchange code → get token)?
     b. Plan to create the auth test FIRST as a reusable scenario.
     c. Use that auth scenario via 'use:' in all dependent tests.

## STEP 2 — SCENARIO STRUCTURE
Ask: does this endpoint depend on data created by a previous request?
  - YES (chain, e.g. create then read) →
    * 'setup' block: prepare prerequisite data (auth, resource creation)
    * 'steps' block: the actual user flow / assertion under test
    * 'teardown' block: cleanup created resources (guaranteed to run even on failure)
  - NO (single standalone endpoint) →
    * 'steps' block only. No setup/teardown needed.

## STEP 3 — REUSABILITY CHECK
Ask: will other tests need this same flow (e.g. auth, resource creation pattern)?
  - YES → extract the shared flow into its own .yaml file under .gherkio/tests/shared/
           Reference it via 'use: <path>' from specific scenarios.
  - NO  → keep it self-contained in one scenario file.

## CONVENTIONS TO FOLLOW
- Prefix all saved variables with the step number (e.g. '1-authToken', '2-userId')
- Use '$accounts.<name>.<field>' for cross-account credential access
- Use canonical dot-paths: 'body.<field>', 'headers.<name>', 'jwt.<claim>'
- Validate YAML via validate_test before creating the file
- Dry-run (run_test with dryRun=true) before executing for real`

	userMsg := fmt.Sprintf(`Plan a Gherkio test for:
  Endpoint: %s
  Auth required: %t

Existing tests in workspace:
  %s

Return a structured plan with:
1. Auth dependency analysis (does auth test exist? if not, identify auth API)
2. Scenario structure choice (setup/steps/teardown vs steps only)
3. Reuse decision (extract shared flow or keep standalone)`, endpoint, authRequired, existingTests)

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
  - Downstream assertions using saved variables
  Use when: endpoint is part of a multi-step workflow

## NEGATIVE ASSERTIONS (when applicable)
  - 'schema: not <error-name>' — response must NOT match error schema
  - 'body.<field>: not exists' — field must be absent
  - 'body.<optional_field>: null' — field explicitly null

## COLLECTION ASSERTIONS (for array responses)
  - 'count(body.<array>): <N>' — exact count
  - 'count(body.<array>).gte: 1' — has data
  - 'all(body.<array>.<field>): <matcher>' — every element matches`

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
