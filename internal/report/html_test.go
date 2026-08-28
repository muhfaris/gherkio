package report

import (
	"strings"
	"testing"
	"time"

	"github.com/muhfaris/gherkio/internal/model"
	"github.com/muhfaris/gherkio/internal/runner"
)

func TestMapResultToReportData(t *testing.T) {
	result := &runner.RunResult{
		Scenario:  "Test Scenario",
		TotalPass: 1,
		TotalFail: 0,
		Passed:    true,
		Duration:  time.Millisecond * 500,
		Steps: []runner.StepResult{
			{
				Original: model.Step{
					Request: model.Request{
						Method: "POST",
						URL:    "/api/login",
					},
				},
				Request: &runner.RequestInfo{
					Method:  "POST",
					URL:     "/api/login",
					Headers: map[string]string{"Content-Type": "application/json"},
					Body:    `{"username":"test","password":"pwd"}`,
				},
				Response: &runner.ResponseInfo{
					Status:  200,
					Headers: map[string]string{"X-Request-Id": "req-12345"},
					Body:    `{"token":"secret-token"}`,
				},
				Duration: time.Millisecond * 300,
				Assertions: []runner.AssertionResult{
					{
						Path:     "status",
						Expected: "200",
						Actual:   "200",
						Passed:   true,
					},
				},
			},
		},
	}

	maskFields := []string{"password", "token"}
	data := MapResultToReportData(result, "local", maskFields, true)

	if data.ScenarioName != "Test Scenario" {
		t.Errorf("expected scenario name 'Test Scenario', got '%s'", data.ScenarioName)
	}
	if data.Environment != "local" {
		t.Errorf("expected env 'local', got '%s'", data.Environment)
	}
	if len(data.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(data.Steps))
	}

	step := data.Steps[0]
	if step.Method != "POST" {
		t.Errorf("expected method 'POST', got '%s'", step.Method)
	}
	if step.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", step.StatusCode)
	}
	if step.RequestID != "req-12345" {
		t.Errorf("expected RequestID 'req-12345', got '%s'", step.RequestID)
	}

	// Check masking in cURL
	if !strings.Contains(step.CurlCommand, "***masked***") || strings.Contains(step.CurlCommand, "pwd") {
		t.Errorf("expected password to be masked in curl command, got: %s", step.CurlCommand)
	}

	// Check masking in Body
	if !strings.Contains(step.RequestBody, "***masked***") || strings.Contains(step.RequestBody, "pwd") {
		t.Errorf("expected password to be masked in request body, got: %s", step.RequestBody)
	}
	if !strings.Contains(step.ResponseBody, "***masked***") || strings.Contains(step.ResponseBody, "secret-token") {
		t.Errorf("expected token to be masked in response body, got: %s", step.ResponseBody)
	}
}

func TestRenderHTML(t *testing.T) {
	result := &runner.RunResult{
		Scenario:  "Test HTML Render",
		TotalPass: 1,
		TotalFail: 0,
		Passed:    true,
		Duration:  time.Millisecond * 500,
		Steps: []runner.StepResult{
			{
				Original: model.Step{
					Request: model.Request{
						Method: "GET",
						URL:    "/api/users",
					},
				},
				Request: &runner.RequestInfo{
					Method: "GET",
					URL:    "/api/users",
				},
				Response: &runner.ResponseInfo{
					Status: 200,
				},
				Duration: time.Millisecond * 300,
				Assertions: []runner.AssertionResult{
					{
						Path:     "status",
						Expected: "200",
						Actual:   "200",
						Passed:   true,
					},
				},
			},
		},
	}

	cfg := ReportConfig{
		Format:        "html",
		MaskSensitive: true,
		MaskFields:    []string{"password"},
	}

	html, err := RenderHTML(result, cfg, "test")
	if err != nil {
		t.Fatalf("RenderHTML failed: %v", err)
	}

	if !strings.Contains(html, "<html") {
		t.Errorf("expected output to contain HTML tags, got: %s...", html[:50])
	}
	if !strings.Contains(html, "Test HTML Render") {
		t.Errorf("expected HTML to contain scenario name")
	}
}

func TestRenderJSON(t *testing.T) {
	result := &runner.RunResult{
		Scenario:  "Test JSON Render",
		TotalPass: 1,
		TotalFail: 0,
		Passed:    true,
		Duration:  time.Millisecond * 500,
		Steps: []runner.StepResult{
			{
				Original: model.Step{
					Request: model.Request{
						Method: "GET",
						URL:    "/api/users",
					},
				},
				Request: &runner.RequestInfo{
					Method: "GET",
					URL:    "/api/users",
				},
				Response: &runner.ResponseInfo{
					Status: 200,
				},
				Duration: time.Millisecond * 300,
				Assertions: []runner.AssertionResult{
					{
						Path:     "status",
						Expected: "200",
						Actual:   "200",
						Passed:   true,
					},
				},
			},
		},
	}

	cfg := ReportConfig{
		Format:        "json",
		MaskSensitive: true,
		MaskFields:    []string{"password"},
	}

	jsonStr, err := RenderJSON(result, cfg, "test")
	if err != nil {
		t.Fatalf("RenderJSON failed: %v", err)
	}

	if !strings.Contains(jsonStr, "\"ScenarioName\": \"Test JSON Render\"") {
		t.Errorf("expected JSON to contain ScenarioName")
	}
}

func TestRenderHTML_ComposedTraceability(t *testing.T) {
	result := &runner.RunResult{
		Scenario:  "Test Composed Scenario",
		TotalPass: 1,
		TotalFail: 0,
		Passed:    true,
		Duration:  time.Millisecond * 500,
		Steps: []runner.StepResult{
			{
				IsUseStart: true,
				UseFile:    "nested_auth.yaml",
				Depth:      1,
				SavedVars: map[string]interface{}{
					"nestedVar": "nestedValue",
				},
				Role: "steps",
			},
			{
				Original: model.Step{
					Request: model.Request{
						Method: "GET",
						URL:    "/api/users",
					},
				},
				Request: &runner.RequestInfo{
					Method: "GET",
					URL:    "/api/users",
				},
				Response: &runner.ResponseInfo{
					Status: 200,
				},
				Depth:    1,
				Duration: time.Millisecond * 300,
				Assertions: []runner.AssertionResult{
					{
						Path:     "status",
						Expected: "200",
						Actual:   "200",
						Passed:   true,
					},
				},
				Role: "steps",
			},
			{
				IsUseEnd: true,
				UseFile:  "nested_auth.yaml",
				Depth:    1,
				Role:     "steps",
			},
		},
	}

	cfg := ReportConfig{
		Format:        "html",
		MaskSensitive: true,
	}

	html, err := RenderHTML(result, cfg, "test")
	if err != nil {
		t.Fatalf("RenderHTML failed: %v", err)
	}

	// Verify that IsUseStart is rendered as a grouping header with Composed Scenario and UseFile
	if !strings.Contains(html, "Composed Scenario:") {
		t.Errorf("expected HTML to contain Composed Scenario header")
	}
	if !strings.Contains(html, "nested_auth.yaml") {
		t.Errorf("expected HTML to contain UseFile 'nested_auth.yaml'")
	}

	// Verify that SavedVars snapshot is rendered
	if !strings.Contains(html, "nestedVar") {
		t.Errorf("expected HTML to contain saved variable key 'nestedVar'")
	}
	if !strings.Contains(html, "nestedValue") {
		t.Errorf("expected HTML to contain saved variable value 'nestedValue'")
	}

	// Verify that IsUseEnd is rendered
	if !strings.Contains(html, "step-compose-end") {
		t.Errorf("expected HTML to contain step-compose-end class")
	}

	// Verify depth indentation is rendered (margin-left style with calc)
	if !strings.Contains(html, "margin-left: calc(1 * 20px)") {
		t.Errorf("expected HTML to contain calc margin-left styling for depth 1")
	}
}

func TestRenderHTMLSuite_LoadRunSummaryAndCollapsibleWorkflows(t *testing.T) {
	started := time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC)
	results := []*runner.RunResult{
		loadRunResult(1, 1, true, started, started.Add(200*time.Millisecond), 100*time.Millisecond),
		loadRunResult(1, 2, false, started.Add(210*time.Millisecond), started.Add(510*time.Millisecond), 300*time.Millisecond),
		loadRunResult(2, 1, true, started.Add(10*time.Millisecond), started.Add(160*time.Millisecond), 150*time.Millisecond),
		loadRunResult(2, 2, true, started.Add(170*time.Millisecond), started.Add(370*time.Millisecond), 200*time.Millisecond),
	}

	data := MapResultsToSuiteReportData(results, "local", nil, true)
	if !data.LoadMode || data.VirtualUsers != 2 || data.IterationsPerUser != 2 || data.WorkflowCount != 4 {
		t.Fatalf("unexpected load metadata: %+v", data)
	}
	if data.PassedWorkflows != 3 || data.FailedWorkflows != 1 || data.RequestCount != 4 {
		t.Fatalf("unexpected workflow/request counts: %+v", data)
	}
	if data.TotalDuration != runner.FormatDuration(510*time.Millisecond) {
		t.Errorf("wall duration = %s, want %s", data.TotalDuration, runner.FormatDuration(510*time.Millisecond))
	}
	if data.P95ResponseTime != runner.FormatDuration(300*time.Millisecond) {
		t.Errorf("p95 = %s, want %s", data.P95ResponseTime, runner.FormatDuration(300*time.Millisecond))
	}

	html, err := RenderHTMLSuite(results, ReportConfig{}, "local")
	if err != nil {
		t.Fatalf("RenderHTMLSuite: %v", err)
	}
	for _, fragment := range []string{
		"2 virtual users", "2 iterations each", "4 workflow executions",
		"VU 1 · iteration 1/2", "details class=\"scenario-block\"",
		"class=\"scenario-block\" open", "p95 latency",
	} {
		if !strings.Contains(html, fragment) {
			t.Errorf("expected rendered report to contain %q", fragment)
		}
	}
}

func loadRunResult(vu, iteration int, passed bool, started, finished time.Time, requestDuration time.Duration) *runner.RunResult {
	step := runner.StepResult{
		Name:     "Request",
		Request:  &runner.RequestInfo{Method: "GET", URL: "/users"},
		Response: &runner.ResponseInfo{Status: 200},
		Duration: requestDuration,
	}
	if !passed {
		step.Error = "assertion failed"
	}
	return &runner.RunResult{
		Scenario:          "Create attachment",
		TestFile:          "1-create-attachment.yaml",
		Steps:             []runner.StepResult{step},
		Passed:            passed,
		Duration:          finished.Sub(started),
		VirtualUser:       vu,
		Iteration:         iteration,
		IterationsPerUser: 2,
		StartedAt:         started,
		FinishedAt:        finished,
	}
}
