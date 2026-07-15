package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/muhfaris/gherkio/internal/model"
)

func TestParseUntil(t *testing.T) {
	// Create a mock TestFile
	testFile := &model.TestFile{
		Setup: []model.Step{
			{Request: model.Request{URL: "http://setup1"}},
			{Request: model.Request{URL: "http://setup2"}},
		},
		Steps: []model.Step{
			{Request: model.Request{URL: "http://step1"}},
			{Request: model.Request{URL: "http://step2"}},
			{Request: model.Request{URL: "http://step3"}},
		},
		Teardown: []model.Step{
			{Request: model.Request{URL: "http://teardown1"}},
		},
	}

	tests := []struct {
		name        string
		untilStr    string
		wantSection string
		wantIndex   int
		expectError bool
	}{
		{
			name:        "empty until string",
			untilStr:    "",
			wantSection: "",
			wantIndex:   -1,
			expectError: false,
		},
		{
			name:        "default section (steps) with index",
			untilStr:    "1",
			wantSection: "steps",
			wantIndex:   1,
			expectError: false,
		},
		{
			name:        "setup section with index",
			untilStr:    "setup:0",
			wantSection: "setup",
			wantIndex:   0,
			expectError: false,
		},
		{
			name:        "steps section with index",
			untilStr:    "steps:2",
			wantSection: "steps",
			wantIndex:   2,
			expectError: false,
		},
		{
			name:        "teardown section with index",
			untilStr:    "teardown:0",
			wantSection: "teardown",
			wantIndex:   0,
			expectError: false,
		},
		{
			name:        "case insensitivity of section",
			untilStr:    "SeTuP:1",
			wantSection: "setup",
			wantIndex:   1,
			expectError: false,
		},
		{
			name:        "whitespace trimming",
			untilStr:    "  steps  :  1  ",
			wantSection: "steps",
			wantIndex:   1,
			expectError: false,
		},
		{
			name:        "invalid index format",
			untilStr:    "steps:abc",
			expectError: true,
		},
		{
			name:        "negative index",
			untilStr:    "steps:-1",
			expectError: true,
		},
		{
			name:        "setup index out of bounds",
			untilStr:    "setup:2",
			expectError: true,
		},
		{
			name:        "steps index out of bounds",
			untilStr:    "steps:3",
			expectError: true,
		},
		{
			name:        "teardown index out of bounds",
			untilStr:    "teardown:1",
			expectError: true,
		},
		{
			name:        "invalid section name",
			untilStr:    "invalidsec:1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sec, idx, err := parseUntil(tt.untilStr, testFile)
			if (err != nil) != tt.expectError {
				t.Errorf("parseUntil(%q) error = %v, expectError = %v", tt.untilStr, err, tt.expectError)
				return
			}
			if !tt.expectError {
				if sec != tt.wantSection {
					t.Errorf("parseUntil(%q) got section = %q, want %q", tt.untilStr, sec, tt.wantSection)
				}
				if idx != tt.wantIndex {
					t.Errorf("parseUntil(%q) got index = %d, want %d", tt.untilStr, idx, tt.wantIndex)
				}
			}
		})
	}
}

func TestExecuteSteps_Set(t *testing.T) {
	steps := []model.Step{
		{
			Set: map[string]string{
				"FOO": "bar",
				"BAZ": "qux",
			},
		},
		{
			Set: map[string]string{
				"INTERPOLATED": "$FOO-value",
			},
		},
	}

	vars := make(map[string]interface{})
	results, pass, fail, ok := executeSteps(
		steps,
		nil, // env
		vars,
		"", // projectDir
		"", // currentDir
		0,  // depth
		"steps",
		false, // dryRun
		false, // failFast
		nil,   // sandbox
		SnapshotConfig{},
		"scenario",
		"testfile.yaml",
	)

	if !ok {
		t.Error("Expected execution to succeed")
	}
	if pass != 2 {
		t.Errorf("Expected 2 passing steps, got %d", pass)
	}
	if fail != 0 {
		t.Errorf("Expected 0 failing steps, got %d", fail)
	}
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// Verify variables in the map
	if vars["FOO"] != "bar" {
		t.Errorf("Expected FOO=bar, got %v", vars["FOO"])
	}
	if vars["BAZ"] != "qux" {
		t.Errorf("Expected BAZ=qux, got %v", vars["BAZ"])
	}
	if vars["INTERPOLATED"] != "bar-value" {
		t.Errorf("Expected INTERPOLATED=bar-value, got %v", vars["INTERPOLATED"])
	}

	// Verify SavedVars in step results
	s1 := results[0]
	if s1.SavedVars["FOO"] != "bar" || s1.SavedVars["BAZ"] != "qux" {
		t.Errorf("Unexpected SavedVars in step 1: %v", s1.SavedVars)
	}

	s2 := results[1]
	if s2.SavedVars["INTERPOLATED"] != "bar-value" {
		t.Errorf("Unexpected SavedVars in step 2: %v", s2.SavedVars)
	}
}

func TestExecuteSteps_Set_Error(t *testing.T) {
	steps := []model.Step{
		{
			Set: map[string]string{
				"BAD": "$MISSING_VAR",
			},
		},
	}

	vars := make(map[string]interface{})
	results, pass, fail, ok := executeSteps(
		steps,
		nil, // env
		vars,
		"", // projectDir
		"", // currentDir
		0,  // depth
		"steps",
		false, // dryRun
		false, // failFast
		nil,   // sandbox
		SnapshotConfig{},
		"scenario",
		"testfile.yaml",
	)

	if ok {
		t.Error("Expected execution to fail")
	}
	if pass != 0 {
		t.Errorf("Expected 0 passing steps, got %d", pass)
	}
	if fail != 1 {
		t.Errorf("Expected 1 failing steps, got %d", fail)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].Error == "" {
		t.Error("Expected error message in step result, got empty")
	}
}

func TestMockMatchingAndInterpolation(t *testing.T) {
	mockRule := model.MockRule{
		Request: model.MockRequest{
			Method: "POST",
			URL:    "/v1/charges",
		},
		Response: model.MockResponse{
			Status: 201,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: map[string]interface{}{
				"id":      "ch_123",
				"amount":  "$request.body.amount",
				"auth":    "Bearer $request.headers.Authorization",
				"custom":  "static-val",
				"nested": map[string]interface{}{
					"currency": "$request.body.currency",
				},
				"list": []interface{}{
					"$request.body.amount",
					"static-list-val",
				},
			},
		},
	}

	// 1. Test MatchMock
	if !matchMock("POST", "https://api.stripe.com/v1/charges", mockRule) {
		t.Error("expected matchMock to pass for exact path suffix matching")
	}
	if !matchMock("POST", "https://api.stripe.com/v1/charges?foo=bar", mockRule) {
		t.Error("expected matchMock to ignore query parameters")
	}
	if matchMock("GET", "https://api.stripe.com/v1/charges", mockRule) {
		t.Error("expected matchMock to reject mismatched HTTP method")
	}
	if matchMock("POST", "https://api.stripe.com/v1/refunds", mockRule) {
		t.Error("expected matchMock to reject mismatched URLs")
	}

	// 2. Test Interpolation
	reqBody := map[string]interface{}{
		"amount":   2000,
		"currency": "usd",
	}
	reqHeaders := map[string]string{
		"Authorization": "token123",
	}

	interpolated := interpolateMockBody(mockRule.Response.Body, reqBody, reqHeaders)
	resMap, ok := interpolated.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{} result, got %T", interpolated)
	}

	if resMap["id"] != "ch_123" {
		t.Errorf("expected id='ch_123', got %v", resMap["id"])
	}
	if resMap["amount"] != 2000 {
		t.Errorf("expected amount=2000 (int preserved), got %T (%v)", resMap["amount"], resMap["amount"])
	}
	if resMap["auth"] != "Bearer token123" {
		t.Errorf("expected auth='Bearer token123', got %v", resMap["auth"])
	}
	nestedMap, ok := resMap["nested"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected nested map, got %T", resMap["nested"])
	}
	if nestedMap["currency"] != "usd" {
		t.Errorf("expected currency='usd', got %v", nestedMap["currency"])
	}
	listVal, ok := resMap["list"].([]interface{})
	if !ok {
		t.Fatalf("expected list, got %T", resMap["list"])
	}
	if len(listVal) != 2 {
		t.Fatalf("expected list length 2, got %d", len(listVal))
	}
	if listVal[0] != 2000 {
		t.Errorf("expected listVal[0]=2000, got %v", listVal[0])
	}
}

func TestExecuteSteps_Mocks(t *testing.T) {
	steps := []model.Step{
		{
			Request: model.Request{
				Method: "POST",
				URL:    "https://api.stripe.com/v1/charges",
				Headers: map[string]string{
					"Authorization": "token_xyz",
				},
				Body: map[string]interface{}{
					"amount": 5000,
				},
			},
			Expect: model.Expect{
				Status: 201,
				Extra: map[string]interface{}{
					"body.id":     "ch_123",
					"body.amount": 5000,
					"body.auth":   "Bearer token_xyz",
				},
			},
		},
	}

	env := &model.Environment{
		Mocks: []model.MockRule{
			{
				Request: model.MockRequest{
					Method: "POST",
					URL:    "/v1/charges",
				},
				Response: model.MockResponse{
					Status: 201,
					Body: map[string]interface{}{
						"id":     "ch_123",
						"amount": "$request.body.amount",
						"auth":   "Bearer $request.headers.Authorization",
					},
				},
			},
		},
	}

	vars := make(map[string]interface{})
	results, pass, fail, ok := executeSteps(
		steps,
		env,
		vars,
		"", // projectDir
		"", // currentDir
		0,  // depth
		"steps",
		false, // dryRun
		false, // failFast
		nil,   // sandbox
		SnapshotConfig{},
		"scenario",
		"testfile.yaml",
	)

	if !ok {
		t.Error("Expected mocked execution to succeed")
	}
	if pass != 4 {
		t.Errorf("Expected 4 passing assertions, got %d", pass)
	}
	if fail != 0 {
		t.Errorf("Expected 0 failing assertions, got %d", fail)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if len(results[0].Assertions) == 0 {
		t.Error("Expected assertions to have run")
	}
	for _, assertion := range results[0].Assertions {
		if !assertion.Passed {
			t.Errorf("Assertion %s failed: %+v", assertion.Path, assertion)
		}
	}
}

func TestRun_Parametrization(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create a test file with examples
	testYAML := `
scenario: Parametrized User Login
description: Test login for multiple roles
examples:
  - username: alice
    expected_role: admin
  - username: bob
    expected_role: user
steps:
  - name: Mock login for $username
    request:
      method: POST
      url: https://api.example.com/login
      body:
        user: $username
    expect:
      status: 200
      body.role: $expected_role
`
	testPath := filepath.Join(tmpDir, "login_test.yaml")
	if err := os.WriteFile(testPath, []byte(testYAML), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create a project config to make the run happy
	projectDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(filepath.Join(projectDir, ".gherkio", "environments"), 0755); err != nil {
		t.Fatalf("failed to create env dir: %v", err)
	}
	
	// Write environment with a mock matching the request
	envYAML := `
baseUrl: https://api.example.com
mocks:
  - request:
      method: POST
      url: /login
    response:
      status: 200
      body:
        role: $request.body.user
        # For 'alice', role is 'alice'. Wait, we want to return expected_role?
        # But expected_role is not in request body.
        # Wait, if we mock it to return the user name as role, then:
        # alice -> role: alice. bob -> role: bob.
        # Let's adjust the examples or mock!
`
	// Actually, let's mock it to return role: admin for alice and role: user for bob.
	// But how does mock know?
	// We can use a request parameter or we can just mock it returning:
	// body: { role: "$request.body.user" }
	// And then expected_role in examples can be "alice" and "bob"!
	// Yes! That is extremely elegant and requires no conditional mock rules!
	
	testYAMLWithAdjustedExpectedRole := `
scenario: Parametrized User Login
description: Test login for multiple roles
examples:
  - username: alice
    expected_role: alice
  - username: bob
    expected_role: bob
steps:
  - name: Mock login for $username
    request:
      method: POST
      url: https://api.example.com/login
      body:
        user: $username
    expect:
      status: 200
      body.role: $expected_role
`
	if err := os.WriteFile(testPath, []byte(testYAMLWithAdjustedExpectedRole), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	envYAML = `
baseUrl: https://api.example.com
mocks:
  - request:
      method: POST
      url: /login
    response:
      status: 200
      body:
        role: $request.body.user
`
	if err := os.WriteFile(filepath.Join(projectDir, ".gherkio", "environments", "local.yaml"), []byte(envYAML), 0644); err != nil {
		t.Fatalf("failed to write environment: %v", err)
	}

	cfg := RunConfig{
		TestPath:   testPath,
		EnvName:    "local",
		ProjectDir: projectDir,
		StepIndex:  -1,
	}

	res, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !res.Passed {
		t.Errorf("Expected run to pass, but failed")
	}

	t.Logf("TotalPass: %d, TotalFail: %d, Passed: %v", res.TotalPass, res.TotalFail, res.Passed)
	for i, step := range res.Steps {
		t.Logf("Step %d: Name=%q, Role=%q, Iteration=%d, Error=%q, Passed=%v", i, step.Name, step.Role, step.Iteration, step.Error, step.Error == "")
		for _, a := range step.Assertions {
			t.Logf("  Assertion: Path=%s, Expected=%v, Actual=%v, Passed=%v, Reason=%s", a.Path, a.Expected, a.Actual, a.Passed, a.Reason)
		}
	}

	if len(res.Steps) != 2 {
		t.Fatalf("Expected 2 step results, got %d", len(res.Steps))
	}

	// Check iterations are marked correctly
	if res.Steps[0].Iteration != 1 {
		t.Errorf("Expected first step iteration to be 1, got %d", res.Steps[0].Iteration)
	}
	if res.Steps[1].Iteration != 2 {
		t.Errorf("Expected second step iteration to be 2, got %d", res.Steps[1].Iteration)
	}
}


