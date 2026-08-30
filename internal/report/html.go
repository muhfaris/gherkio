package report

import (
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
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
	suiteTotalSkip := 0
	suiteTotalSteps := 0
	var suiteDuration time.Duration
	var scenarios []ScenarioData
	loadMode := false
	virtualUsers := 0
	iterationsPerUser := 0
	passedWorkflows := 0
	failedWorkflows := 0
	requestCount := 0
	var requestDurations []time.Duration
	var earliestStart time.Time
	var latestFinish time.Time

	for _, result := range results {
		scData := MapResultToReportData(result, env, maskFields, forceCurlMasking)

		// Create a scenario entry from the per-result data
		scenario := ScenarioData{
			Name:              result.Scenario,
			Description:       result.Description,
			TestFile:          result.TestFile,
			Account:           result.Account,
			TotalDuration:     runner.FormatDuration(result.Duration),
			TotalSteps:        scData.TotalSteps,
			PassCount:         scData.PassCount,
			FailCount:         scData.FailCount,
			SkipCount:         scData.SkipCount,
			Steps:             scData.Steps,
			Passed:            result.Passed,
			VirtualUser:       result.VirtualUser,
			Iteration:         result.Iteration,
			IterationsPerUser: result.IterationsPerUser,
			InitialVars:       result.ResolvedVars,
			FinalVars:         result.FinalVars,
		}
		if scData.TotalSteps > 0 {
			scenario.PassPercent = float64(scData.PassCount) / float64(scData.TotalSteps) * 100
			scenario.FailPercent = float64(scData.FailCount) / float64(scData.TotalSteps) * 100
			scenario.SkipPercent = float64(scData.SkipCount) / float64(scData.TotalSteps) * 100
		}

		suiteTotalPass += scData.PassCount
		suiteTotalFail += scData.FailCount
		suiteTotalSkip += scData.SkipCount
		suiteTotalSteps += scData.TotalSteps
		suiteDuration += result.Duration
		if result.Passed {
			passedWorkflows++
		} else {
			failedWorkflows++
		}
		if result.VirtualUser > 0 {
			loadMode = true
			if result.VirtualUser > virtualUsers {
				virtualUsers = result.VirtualUser
			}
			if result.IterationsPerUser > iterationsPerUser {
				iterationsPerUser = result.IterationsPerUser
			}
			if earliestStart.IsZero() || (!result.StartedAt.IsZero() && result.StartedAt.Before(earliestStart)) {
				earliestStart = result.StartedAt
			}
			if result.FinishedAt.After(latestFinish) {
				latestFinish = result.FinishedAt
			}
			for _, step := range result.Steps {
				if step.Skipped || step.Request == nil || (step.Response == nil && step.Error == "" && len(step.RetryHistory) == 0) {
					continue
				}
				if len(step.RetryHistory) > 0 {
					for _, attempt := range step.RetryHistory {
						requestCount++
						requestDurations = append(requestDurations, attempt.Duration)
					}
				} else {
					requestCount++
					requestDurations = append(requestDurations, step.Duration)
				}
			}
		}

		scenarios = append(scenarios, scenario)
	}
	if loadMode && !earliestStart.IsZero() && latestFinish.After(earliestStart) {
		suiteDuration = latestFinish.Sub(earliestStart)
	}

	averageResponseTime := "0s"
	p95ResponseTime := "0s"
	requestsPerSecond := "0.00"
	if len(requestDurations) > 0 {
		var totalRequestDuration time.Duration
		for _, duration := range requestDurations {
			totalRequestDuration += duration
		}
		averageResponseTime = runner.FormatDuration(totalRequestDuration / time.Duration(len(requestDurations)))
		sort.Slice(requestDurations, func(i, j int) bool { return requestDurations[i] < requestDurations[j] })
		p95Index := (95*len(requestDurations)+99)/100 - 1
		p95ResponseTime = runner.FormatDuration(requestDurations[p95Index])
	}
	if suiteDuration > 0 {
		requestsPerSecond = fmt.Sprintf("%.2f", float64(requestCount)/suiteDuration.Seconds())
	}

	scenarioName := "Test Suite"
	description := ""
	timestamp := time.Now()
	if loadMode && len(results) > 0 {
		scenarioName = results[0].Scenario + " · Load Run"
		description = results[0].Description
		if !earliestStart.IsZero() {
			timestamp = earliestStart
		}
	}

	passPercent := 0.0
	failPercent := 0.0
	skipPercent := 0.0
	if suiteTotalSteps > 0 {
		passPercent = float64(suiteTotalPass) / float64(suiteTotalSteps) * 100
		failPercent = float64(suiteTotalFail) / float64(suiteTotalSteps) * 100
		skipPercent = float64(suiteTotalSkip) / float64(suiteTotalSteps) * 100
	}

	data := ReportData{
		ScenarioName:  scenarioName,
		Description:   description,
		Environment:   env,
		Timestamp:     timestamp.Format(time.RFC1123),
		TotalDuration: runner.FormatDuration(suiteDuration),
		TotalSteps:    suiteTotalSteps,
		PassCount:     suiteTotalPass,
		FailCount:     suiteTotalFail,
		SkipCount:     suiteTotalSkip,
		PassPercent:   passPercent,
		FailPercent:   failPercent,
		SkipPercent:   skipPercent,
		Scenarios:     scenarios,
	}
	if loadMode {
		data.LoadMode = true
		data.VirtualUsers = virtualUsers
		data.IterationsPerUser = iterationsPerUser
		data.WorkflowCount = len(results)
		data.PassedWorkflows = passedWorkflows
		data.FailedWorkflows = failedWorkflows
		data.RequestCount = requestCount
		data.AverageResponseTime = averageResponseTime
		data.P95ResponseTime = p95ResponseTime
		data.RequestsPerSecond = requestsPerSecond
	}
	return data
}

// MapResultToReportData converts a runner.RunResult to ReportData.
func MapResultToReportData(result *runner.RunResult, env string, maskFields []string, forceCurlMasking bool) ReportData {
	totalPass := 0
	totalFail := 0
	totalSkip := 0

	var reportSteps []ReportStep
	stepIndex := 1

	for _, step := range result.Steps {
		if step.IsUseStart || step.IsUseEnd {
			reportSteps = append(reportSteps, ReportStep{
				Index:     0,
				Name:      step.Name,
				Passed:    step.Error == "",
				Skipped:   step.Skipped,
				SavedVars: step.SavedVars,
				Role:      step.Role,
				Original:  step,
			})
			continue
		}

		var reqID, curlCmd, reqBody, resBody, statusText string
		var statusCode int
		var timingFailed bool
		stepPassed := step.Error == "" && !step.Skipped

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

		if step.Skipped {
			totalSkip++
		} else if stepPassed {
			totalPass++
		} else {
			totalFail++
		}

		method := ""
		url := ""
		var query map[string]any
		var headers map[string]string
		if step.Request != nil {
			method = step.Request.Method
			url = step.Request.URL
			// Convert map[string]string to map[string]any for consistent typing
			if step.Request.Query != nil {
				query = make(map[string]any, len(step.Request.Query))
				for k, v := range step.Request.Query {
					query[k] = v
				}
			}
			headers = step.Request.Headers
		} else if step.Redis != nil {
			method = "REDIS " + strings.ToUpper(step.Redis.Command)
			url = step.Redis.Key + " @ " + step.Redis.Connection
		} else if step.Original.Request.URL != "" {
			method = step.Original.Request.Method
			url = step.Original.Request.URL
			query = step.Original.Request.Query
			headers = step.Original.Request.Headers
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
			Headers:      headers,
			StatusCode:   statusCode,
			StatusText:   statusText,
			Duration:     runner.FormatDuration(step.Duration),
			TimingFailed: timingFailed,
			RequestID:    reqID,
			CurlCommand:  curlCmd,
			RequestBody:  reqBody,
			ResponseBody: resBody,
			Passed:       stepPassed,
			Skipped:      step.Skipped,
			Assertions:   assertions,
			Error:        step.Error,
			Warnings:     step.Warnings,
			RetryCount:   step.RetryCount,
			RetryHistory: retryHistory,
			SavedVars:    step.SavedVars,
			Role:         step.Role,
			Original:     step,
		})
		stepIndex++
	}

	totalSteps := totalPass + totalFail + totalSkip
	passPercent := 0.0
	failPercent := 0.0
	skipPercent := 0.0
	if totalSteps > 0 {
		passPercent = float64(totalPass) / float64(totalSteps) * 100
		failPercent = float64(totalFail) / float64(totalSteps) * 100
		skipPercent = float64(totalSkip) / float64(totalSteps) * 100
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
		SkipCount:     totalSkip,
		PassPercent:   passPercent,
		FailPercent:   failPercent,
		SkipPercent:   skipPercent,
		Steps:         reportSteps,
		InitialVars:   result.ResolvedVars,
		FinalVars:     result.FinalVars,
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
