package report

import (
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/muhfaris/gherkio/internal/runner"
)

//go:embed template.html
var reportTemplateStr string

// MapResultsToSuiteReportData converts multiple RunResults into a single suite-level ReportData.
func MapResultsToSuiteReportData(results []*runner.RunResult, env string, maskFields []string, forceCurlMasking bool) ReportData {
	suiteTotalPass := 0
	suiteTotalFail := 0
	suiteTotalSteps := 0
	var suiteDuration time.Duration
	var scenarios []ScenarioData

	for _, result := range results {
		scData := MapResultToReportData(result, env, maskFields, forceCurlMasking)

		// Create a scenario entry from the per-result data
		scenario := ScenarioData{
			Name:          result.Scenario,
			Description:   result.Description,
			TestFile:      result.TestFile,
			Account:       result.Account,
			TotalDuration: runner.FormatDuration(result.Duration),
			TotalSteps:    scData.TotalSteps,
			PassCount:     scData.PassCount,
			FailCount:     scData.FailCount,
			Steps:         scData.Steps,
		}
		if scData.TotalSteps > 0 {
			scenario.PassPercent = float64(scData.PassCount) / float64(scData.TotalSteps) * 100
			scenario.FailPercent = float64(scData.FailCount) / float64(scData.TotalSteps) * 100
		}

		suiteTotalPass += scData.PassCount
		suiteTotalFail += scData.FailCount
		suiteTotalSteps += scData.TotalSteps
		suiteDuration += result.Duration

		scenarios = append(scenarios, scenario)
	}

	passPercent := 0.0
	failPercent := 0.0
	if suiteTotalSteps > 0 {
		passPercent = float64(suiteTotalPass) / float64(suiteTotalSteps) * 100
		failPercent = float64(suiteTotalFail) / float64(suiteTotalSteps) * 100
	}

	return ReportData{
		ScenarioName:  "Test Suite",
		Environment:   env,
		Timestamp:     time.Now().Format(time.RFC1123),
		TotalDuration: runner.FormatDuration(suiteDuration),
		TotalSteps:    suiteTotalSteps,
		PassCount:     suiteTotalPass,
		FailCount:     suiteTotalFail,
		PassPercent:   passPercent,
		FailPercent:   failPercent,
		Scenarios:     scenarios,
	}
}

// MapResultToReportData converts a runner.RunResult to ReportData.
func MapResultToReportData(result *runner.RunResult, env string, maskFields []string, forceCurlMasking bool) ReportData {
	totalPass := 0
	totalFail := 0

	var reportSteps []ReportStep
	stepIndex := 1

	for _, step := range result.Steps {
		if step.IsUseStart || step.IsUseEnd {
			continue // Skip use directives for the report step index
		}

		var reqID, curlCmd, reqBody, resBody, statusText string
		var statusCode int
		var timingFailed bool
		stepPassed := step.Error == ""

		if step.Request != nil {

			curlMaskFields := maskFields
			if forceCurlMasking {
				curlMaskFields = runner.GetDefaultSensitiveFields()
			}
			curlCmd = generateCurl(step.Request, curlMaskFields)
			reqBody = runner.FormatRequestBody(step.Request.Body, maskFields)
		}

		if step.Response != nil {
			reqID = extractRequestId(step.Response.Headers)
			resBody = runner.FormatRequestBody(step.Response.Body, maskFields)
			statusCode = step.Response.Status
			statusText = http.StatusText(statusCode)
		}

		var assertions []ReportAssertion
		for _, a := range step.Assertions {
			if !a.Passed {
				stepPassed = false
			}

			if a.Path == "timing.duration" && !a.Passed {
				timingFailed = true
			}

			label := fmt.Sprintf("%s = %s", a.Path, a.Expected)
			if a.Expected == "exists" {
				label = fmt.Sprintf("%s exists", a.Path)
			}

			detail := ""
			if !a.Passed {
				if strings.HasPrefix(a.Actual, "(not found)") || a.Actual == "(unresolved)" {
					detail = "path not found"
					if len(a.Suggestions) > 0 {
						detail += fmt.Sprintf(" (available: %s)", strings.Join(a.Suggestions, ", "))
					}
				} else if a.Reason != "" {
					detail = fmt.Sprintf("expected: %s, actual: %s, reason: %s", a.Expected, a.Actual, a.Reason)
				} else {
					detail = fmt.Sprintf("got: %s", a.Actual)
				}
			}

			assertions = append(assertions, ReportAssertion{
				Label:  label,
				Detail: detail,
				Passed: a.Passed,
			})
		}

		if stepPassed {
			totalPass++
		} else {
			totalFail++
		}

		method := ""
		url := ""
		var query map[string]string
		if step.Request != nil {
			method = step.Request.Method
			url = step.Request.URL
			query = step.Request.Query
		} else if step.Original.Request.URL != "" {
			method = step.Original.Request.Method
			url = step.Original.Request.URL
			query = step.Original.Request.Query
		}

		var retryHistory []RetryEntry
		for _, entry := range step.RetryHistory {
			retryHistory = append(retryHistory, RetryEntry{
				Attempt:  entry.Attempt,
				Status:   entry.Status,
				Body:     entry.Body,
				Duration: runner.FormatDuration(entry.Duration),
				Error:    entry.Error,
			})
		}

		reportSteps = append(reportSteps, ReportStep{
			Index:        stepIndex,
			Name:         step.Name,
			Method:       method,
			URL:          url,
			Query:        query,
			StatusCode:   statusCode,
			StatusText:   statusText,
			Duration:     runner.FormatDuration(step.Duration),
			TimingFailed: timingFailed,
			RequestID:    reqID,
			CurlCommand:  curlCmd,
			RequestBody:  reqBody,
			ResponseBody: resBody,
			Passed:       stepPassed,
			Assertions:   assertions,
			Error:        step.Error,
			RetryCount:   step.RetryCount,
			RetryHistory: retryHistory,
			Role:         step.Role,
		})
		stepIndex++
	}

	totalSteps := totalPass + totalFail
	passPercent := 0.0
	failPercent := 0.0
	if totalSteps > 0 {
		passPercent = float64(totalPass) / float64(totalSteps) * 100
		failPercent = float64(totalFail) / float64(totalSteps) * 100
	}

	return ReportData{
		ScenarioName:  result.Scenario,
		Description:   result.Description,
		Environment:   env,
		Timestamp:     time.Now().Format(time.RFC1123),
		TotalDuration: runner.FormatDuration(result.Duration),
		TotalSteps:    totalSteps,
		PassCount:     totalPass,
		FailCount:     totalFail,
		PassPercent:   passPercent,
		FailPercent:   failPercent,
		Steps:         reportSteps,
	}
}

// RenderHTMLSuite generates an HTML report for a suite of multiple scenarios.
func RenderHTMLSuite(results []*runner.RunResult, cfg ReportConfig, env string) (string, error) {
	data := MapResultsToSuiteReportData(results, env, cfg.MaskFields, true)
	return renderHTMLTemplate(data)
}

// renderHTMLTemplate executes the HTML template with the given data.
func renderHTMLTemplate(data ReportData) (string, error) {
	funcs := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"statusClass": func(code int) string {
			switch {
			case code >= 200 && code < 300:
				return "status-2xx"
			case code >= 300 && code < 400:
				return "status-3xx"
			case code >= 400 && code < 500:
				return "status-4xx"
			case code >= 500:
				return "status-5xx"
			default:
				return "status-unknown"
			}
		},
	}

	tmpl, err := template.New("report").Funcs(funcs).Parse(reportTemplateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return sb.String(), nil
}

// RenderHTML generates the HTML report string for a single scenario.
func RenderHTML(result *runner.RunResult, cfg ReportConfig, env string) (string, error) {
	data := MapResultToReportData(result, env, cfg.MaskFields, true)
	return renderHTMLTemplate(data)
}

// SaveHTML saves the rendered HTML to the specified paths.
func SaveHTML(html, projectDir string, customPath string) (string, error) {
	basePath := filepath.Join(projectDir, ".gherkio", "reports")
	if customPath != "" {
		basePath = customPath
	}

	latestDir := filepath.Join(basePath, "latest")
	timestampDir := filepath.Join(basePath, time.Now().Format("20060102_150405"))

	if err := os.MkdirAll(latestDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create latest report dir: %w", err)
	}
	if err := os.MkdirAll(timestampDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create timestamp report dir: %w", err)
	}

	latestFile := filepath.Join(latestDir, "report.html")
	if err := os.WriteFile(latestFile, []byte(html), 0644); err != nil {
		return "", fmt.Errorf("failed to write latest report: %w", err)
	}

	timestampFile := filepath.Join(timestampDir, "report.html")
	if err := os.WriteFile(timestampFile, []byte(html), 0644); err != nil {
		return "", fmt.Errorf("failed to write timestamp report: %w", err)
	}

	return latestFile, nil
}
