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
	data := MapResultToReportData(result, "local", maskFields)

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
