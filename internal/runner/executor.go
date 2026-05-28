package runner

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/muhfaris/gherkio/internal/model"
)

// StepResult holds the result of a single step execution.
type StepResult struct {
	Original     model.Step             `json:"original"`
	Depth        int                    `json:"depth"`
	IsUseStart   bool                   `json:"isUseStart"`
	IsUseEnd     bool                   `json:"isUseEnd"`
	UseFile      string                 `json:"useFile,omitempty"`
	ScenarioName string                 `json:"scenarioName,omitempty"`
	TestFile     string                 `json:"testFile,omitempty"`
	Request      *RequestInfo           `json:"request"`
	Response     *ResponseInfo          `json:"response"`
	Assertions   []AssertionResult      `json:"assertions"`
	SavedVars    map[string]interface{} `json:"savedVars,omitempty"`
	Duration     time.Duration          `json:"duration"`
	Error        string                 `json:"error,omitempty"`
	RetryCount   int                    `json:"retryCount,omitempty"`
	RetryHistory []RetryEntry           `json:"retryHistory,omitempty"`
	Role         string                 `json:"role,omitempty"` // "setup", "steps", "teardown"
}

// RetryEntry captures the outcome of a single retry attempt.
type RetryEntry struct {
	Attempt  int           `json:"attempt"`
	Status   int           `json:"status"`
	Body     string        `json:"body,omitempty"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
}

// RequestInfo captures the executed request details.
type RequestInfo struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body,omitempty"`
}

// ResponseInfo captures the response details.
type ResponseInfo struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
	Parsed  interface{}       `json:"-"` // parsed JSON body, if applicable
}

// AssertionResult holds a single assertion outcome.
type AssertionResult struct {
	Path        string   `json:"path"`
	Expected    string   `json:"expected"`
	Actual      string   `json:"actual"`
	Passed      bool     `json:"passed"`
	Reason      string   `json:"reason,omitempty"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// calculateBackoff computes the sleep duration based on strategy and attempt count, with jitter.
func calculateBackoff(strategy string, intervalMs int, attempt int) time.Duration {
	baseInterval := float64(intervalMs)
	var sleepFloat float64

	switch strings.ToLower(strategy) {
	case "linear":
		sleepFloat = baseInterval * float64(attempt)
	case "exponential":
		// exponential: interval * 2^(attempt-1)
		multiplier := float64(int(1) << (attempt - 1))
		sleepFloat = baseInterval * multiplier
	case "constant":
		fallthrough
	default:
		sleepFloat = baseInterval
	}

	// Apply ±25% jitter
	jitterFactor := 0.75 + (rand.Float64() * 0.5)
	sleepFloat = sleepFloat * jitterFactor

	return time.Duration(sleepFloat) * time.Millisecond
}

// getAvailableFields returns a sorted list of top-level keys from a parsed JSON object.
func getAvailableFields(data interface{}) []string {
	m, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// formatViolation formats a single schema violation into a human-readable string.
func formatViolation(v SchemaViolation) string {
	if v.Rule == "required" {
		return fmt.Sprintf("field %s: %s but %s", v.Field, v.Expected, v.Actual)
	}
	return fmt.Sprintf("field %s: expected %s %s, got %s", v.Field, v.Rule, v.Expected, v.Actual)
}

// executeRequest performs an HTTP request and returns the result.
// It supports both regular body requests and multipart/form-data uploads.
func executeRequest(method, url string, headers map[string]string, body interface{}, multipart *model.MultipartConfig, timeoutStr string, projectDir string) (*ResponseInfo, error) {
	var bodyReader io.Reader
	var contentType string

	// Handle multipart form-data if configured
	if multipart != nil {
		var err error
		bodyReader, contentType, err = buildMultipartBody(multipart, projectDir)
		if err != nil {
			return nil, fmt.Errorf("failed to build multipart body: %w", err)
		}
	} else if body != nil {
		// Handle regular body
		switch b := body.(type) {
		case map[string]interface{}:
			jsonBody, err := json.Marshal(b)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewReader(jsonBody)
		case string:
			bodyReader = strings.NewReader(b)
		default:
			jsonBody, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewReader(jsonBody)
		}
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Apply user-provided headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Set Content-Type for multipart requests
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	} else if body != nil {
		// Auto-detect Content-Type when body is a JSON type and not explicitly set
		if req.Header.Get("Content-Type") == "" {
			switch body.(type) {
			case map[string]interface{}, []interface{}:
				req.Header.Set("Content-Type", "application/json")
			}
		}
	}

	// Parse timeout with fallback to 30s default
	timeout := 30 * time.Second
	if timeoutStr != "" {
		parsed, err := time.ParseDuration(timeoutStr)
		if err == nil {
			timeout = parsed
		}
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	respHeaders := make(map[string]string)
	for k, v := range resp.Header {
		respHeaders[k] = strings.Join(v, ", ")
	}

	info := &ResponseInfo{
		Status:  resp.StatusCode,
		Headers: respHeaders,
		Body:    string(respBody),
	}

	// Try to parse as JSON
	var parsed interface{}
	if err := json.Unmarshal(respBody, &parsed); err == nil {
		info.Parsed = parsed
	}

	return info, nil
}

// buildMultipartBody constructs a multipart/form-data body from the given configuration.
// It returns the body bytes, the content-type header value, and any error encountered.
func buildMultipartBody(mp *model.MultipartConfig, projectDir string) (io.Reader, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Write form fields
	for key, value := range mp.Fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, "", fmt.Errorf("failed to write field '%s': %w", key, err)
		}
	}

	// Write file fields
	for key, item := range mp.Files {
		if err := writeMultipartFile(writer, key, item, projectDir); err != nil {
			return nil, "", err
		}
	}

	// Close the writer to finalize the boundary
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return bytes.NewReader(body.Bytes()), writer.FormDataContentType(), nil
}

// writeMultipartFile writes a single file part to the multipart writer.
func writeMultipartFile(writer *multipart.Writer, fieldName string, item model.MultipartItem, projectDir string) error {
	filePath, err := resolveMultipartFilePath(item.Path, projectDir)
	if err != nil {
		return fmt.Errorf("failed to resolve file path for field '%s': %w", fieldName, err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file '%s' for field '%s': %w", filePath, fieldName, err)
	}
	defer file.Close()

	// Determine filename from item or from the file system
	filename := item.Filename
	if filename == "" {
		filename = filepath.Base(filePath)
	}

	// Create the form file part with custom headers
	if item.ContentType != "" {
		// Use CreatePart to set custom Content-Type header
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, filename))
		header.Set("Content-Type", item.ContentType)
		part, err := writer.CreatePart(header)
		if err != nil {
			return fmt.Errorf("failed to create form file part for field '%s': %w", fieldName, err)
		}
		// Copy file content to the part
		if _, err := io.Copy(part, file); err != nil {
			return fmt.Errorf("failed to write file content for field '%s': %w", fieldName, err)
		}
	} else {
		// Use CreateFormFile for auto content-type detection
		part, err := writer.CreateFormFile(fieldName, filename)
		if err != nil {
			return fmt.Errorf("failed to create form file for field '%s': %w", fieldName, err)
		}
		// Copy file content to the part
		if _, err := io.Copy(part, file); err != nil {
			return fmt.Errorf("failed to write file content for field '%s': %w", fieldName, err)
		}
	}

	return nil
}

// resolveMultipartFilePath resolves a file path according to Gherkio's path resolution rules.
// It checks in order: absolute paths, project root relative, then fixtures fallbacks.
func resolveMultipartFilePath(filePath, projectDir string) (string, error) {
	// If already absolute and exists, use it
	if filepath.IsAbs(filePath) {
		if _, err := os.Stat(filePath); err == nil {
			return filePath, nil
		}
		return "", fmt.Errorf("absolute path does not exist: %s", filePath)
	}

	// Try project root relative path
	if projectDir != "" {
		absPath := filepath.Join(projectDir, filePath)
		if _, err := os.Stat(absPath); err == nil {
			return absPath, nil
		}
	}

	// Try fixtures directory fallback
	if projectDir != "" {
		fixturesPath := filepath.Join(projectDir, "fixtures", filepath.Base(filePath))
		if _, err := os.Stat(fixturesPath); err == nil {
			return fixturesPath, nil
		}
	}

	// Try as-is relative to current working directory
	if _, err := os.Stat(filePath); err == nil {
		absPath, _ := filepath.Abs(filePath)
		return absPath, nil
	}

	return "", fmt.Errorf("file not found: %s (checked: absolute, project root, fixtures/)", filePath)
}

// detectContentType returns the MIME type for a file based on its extension.
func detectContentType(filePath string) string {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	case ".csv":
		return "text/csv"
	case ".txt":
		return "text/plain"
	case ".html", ".htm":
		return "text/html"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".zip":
		return "application/zip"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		return "application/octet-stream"
	}
}

// evaluateTiming checks that step duration is within the expected max.
// Uses the lte matcher for consistency with the matcher system.
func evaluateTiming(actual time.Duration, maxStr string) AssertionResult {
	maxDuration, err := time.ParseDuration(maxStr)
	if err != nil {
		return AssertionResult{
			Path:     "timing.duration",
			Expected: "lte " + maxStr,
			Actual:   fmt.Sprintf("invalid: %s", err.Error()),
			Passed:   false,
		}
	}

	passed := actual <= maxDuration
	return AssertionResult{
		Path:     "timing.duration",
		Expected: "lte " + maxStr,
		Actual:   FormatDuration(actual),
		Passed:   passed,
	}
}

// runAssertions executes all assertions against the response.
func runAssertions(status int, resp *ResponseInfo, jwtClaims map[string]interface{}, expectStatus int, extra map[string]interface{}, projectDir string, requestBody interface{}) []AssertionResult {
	var results []AssertionResult

	// Status assertion
	if expectStatus > 0 {
		passed := status == expectStatus
		results = append(results, AssertionResult{
			Path:     "status",
			Expected: fmt.Sprintf("%d", expectStatus),
			Actual:   fmt.Sprintf("%d", status),
			Passed:   passed,
		})
	}

	// Extra assertions (e.g. response.token: exists, jwt.role: admin)
	for path, expectedVal := range extra {
		result := evaluateAssertion(path, expectedVal, resp, jwtClaims, projectDir, requestBody)
		results = append(results, result)
	}

	return results
}

// evaluateAssertion evaluates a single assertion at the given path.
func evaluateAssertion(path string, expected interface{}, resp *ResponseInfo, jwtClaims map[string]interface{}, projectDir string, requestBody interface{}) AssertionResult {
	expectedStr := fmt.Sprintf("%v", expected)

	// Resolve request.body.<field> references in the expected value
	// e.g. expect: body.data.name: request.body.name
	// This resolves the request body field and uses its value as the expected value.
	if strings.HasPrefix(expectedStr, "request.body.") && requestBody != nil {
		reqPath := strings.TrimPrefix(expectedStr, "request.body.")
		if parsedReq, ok := requestBody.(map[string]interface{}); ok {
			if resolved, found := resolvePath(parsedReq, reqPath); found {
				expectedStr = fmt.Sprintf("%v", resolved)
			}
		}
	}
	if path == "schema" {
		expectedStr, ok := expected.(string)
		if !ok {
			return AssertionResult{
				Path:     "schema",
				Expected: "string (schema name)",
				Actual:   fmt.Sprintf("%T", expected),
				Passed:   false,
				Reason:   "schema name must be a string",
			}
		}

		isNegated := false
		if strings.HasPrefix(expectedStr, "not ") {
			isNegated = true
			expectedStr = strings.TrimPrefix(expectedStr, "not ")
		}

		schema, err := LoadSchema(expectedStr, projectDir)
		if err != nil {
			return AssertionResult{
				Path:     "schema",
				Expected: expectedStr,
				Actual:   "(schema file not found)",
				Passed:   false,
				Reason:   err.Error(),
			}
		}

		violations := ValidateSchema(resp.Parsed, schema, "body")

		if isNegated {
			// schema: not <name> — response should NOT match
			if len(violations) == 0 {
				displayName := "not " + expectedStr
				return AssertionResult{
					Path:     "schema",
					Expected: displayName,
					Actual:   "valid (unexpectedly)",
					Passed:   false,
					Reason:   "response matches schema but should not",
				}
			}
			displayName := "not " + expectedStr
			return AssertionResult{
				Path:     "schema",
				Expected: displayName,
				Actual:   "invalid (expected)",
				Passed:   true,
			}
		}

		// schema: <name> — normal positive assertion
		if len(violations) == 0 {
			return AssertionResult{
				Path:     "schema",
				Expected: expectedStr,
				Actual:   "valid",
				Passed:   true,
			}
		}

		var reasonBuilder strings.Builder
		firstViolation := violations[0]
		summary := formatViolation(firstViolation)

		for i, v := range violations {
			if i > 0 {
				reasonBuilder.WriteString("\n")
			}
			reasonBuilder.WriteString(formatViolation(v))
		}

		return AssertionResult{
			Path:     "schema",
			Expected: expectedStr,
			Actual:   summary,
			Passed:   false,
			Reason:   reasonBuilder.String(),
		}
	}

	// Collection Matchers: count(path) with optional comparator suffix
	// Supports:
	//   count(path): <N>        — exact match (e.g. count(items): 3)
	//   count(path).gte: <N>    — >= N items (e.g. count(items).gte: 1)
	//   count(path).gt: <N>     — > N items
	//   count(path).lte: <N>    — <= N items
	//   count(path).lt: <N>     — < N items
	if strings.HasPrefix(path, "count(") {
		// Check for comparator suffix: count(path).gte, count(path).gt, etc.
		comparator := "eq" // default: exact match
		innerPath := path

		closeParen := strings.LastIndex(path, ")")
		if closeParen < 0 {
			return AssertionResult{
				Path:     path,
				Expected: expectedStr,
				Actual:   "(invalid syntax)",
				Passed:   false,
				Reason:   "missing closing parenthesis in count()",
			}
		}

		// Check if there's a suffix after the closing paren
		if closeParen < len(path)-1 {
			suffix := path[closeParen+1:]
			switch suffix {
			case ".gte":
				comparator = "gte"
			case ".gt":
				comparator = "gt"
			case ".lte":
				comparator = "lte"
			case ".lt":
				comparator = "lt"
			default:
				return AssertionResult{
					Path:     path,
					Expected: expectedStr,
					Actual:   "(invalid syntax)",
					Passed:   false,
					Reason:   fmt.Sprintf("unknown count comparator: %s (use .gte, .gt, .lte, or .lt)", suffix),
				}
			}
			innerPath = path[6:closeParen]
		} else {
			innerPath = path[6 : len(path)-1]
		}

		// Strip body. prefix for consistency with regular assertions
		innerPath = strings.TrimPrefix(innerPath, "body.")
		actualVal, found := resolvePath(resp.Parsed, innerPath)
		if !found {
			return AssertionResult{
				Path:     path,
				Expected: fmt.Sprintf("%s %s items", comparatorLabel(comparator), expectedStr),
				Actual:   "(not found)",
				Passed:   false,
			}
		}

		arrVal, ok := actualVal.([]interface{})
		if !ok {
			return AssertionResult{
				Path:     path,
				Expected: fmt.Sprintf("%s %s items", comparatorLabel(comparator), expectedStr),
				Actual:   fmt.Sprintf("%v", actualVal),
				Passed:   false,
				Reason:   "value is not an array",
			}
		}

		actualLen := len(arrVal)
		expectedLen, err := strconv.Atoi(expectedStr)
		if err != nil {
			return AssertionResult{
				Path:     path,
				Expected: expectedStr,
				Actual:   fmt.Sprintf("%d", actualLen),
				Passed:   false,
				Reason:   "invalid expected count format",
			}
		}

		var passed bool
		switch comparator {
		case "gte":
			passed = actualLen >= expectedLen
		case "gt":
			passed = actualLen > expectedLen
		case "lte":
			passed = actualLen <= expectedLen
		case "lt":
			passed = actualLen < expectedLen
		default:
			passed = actualLen == expectedLen
		}

		reason := ""
		if !passed {
			reason = fmt.Sprintf("array has %d items", actualLen)
		}

		expectedDesc := fmt.Sprintf("%s %d items", comparatorLabel(comparator), expectedLen)
		if comparator == "eq" {
			expectedDesc = fmt.Sprintf("exactly %d items", expectedLen)
		}

		return AssertionResult{
			Path:     path,
			Expected: expectedDesc,
			Actual:   fmt.Sprintf("%d", actualLen),
			Passed:   passed,
			Reason:   reason,
		}
	}

	// Collection Matchers: all(path)
	if strings.HasPrefix(path, "all(") && strings.HasSuffix(path, ")") {
		innerPath := path[4 : len(path)-1]
		// Strip body. prefix for consistency with regular assertions
		innerPath = strings.TrimPrefix(innerPath, "body.")

		// Determine base array path and the field to check
		lastDot := strings.LastIndex(innerPath, ".")
		var arrayPath, fieldName string
		if lastDot != -1 {
			arrayPath = innerPath[:lastDot]
			fieldName = innerPath[lastDot+1:]
		} else {
			arrayPath = innerPath
			fieldName = "" // Check array elements directly
		}

		actualArrVal, found := resolvePath(resp.Parsed, arrayPath)
		if !found {
			return AssertionResult{
				Path:     path,
				Expected: fmt.Sprintf("all elements match %q", expectedStr),
				Actual:   "(not found)",
				Passed:   false,
			}
		}

		arrVal, ok := actualArrVal.([]interface{})
		if !ok {
			return AssertionResult{
				Path:     path,
				Expected: fmt.Sprintf("all elements match %q", expectedStr),
				Actual:   fmt.Sprintf("%v", actualArrVal),
				Passed:   false,
				Reason:   "value is not an array",
			}
		}

		if len(arrVal) == 0 {
			dummyResult, used := evaluateMatcher("", expectedStr, nil)
			expectedDesc := expectedStr
			if used {
				expectedDesc = dummyResult.Expected
			}
			return AssertionResult{
				Path:     path,
				Expected: expectedDesc,
				Actual:   "[]",
				Passed:   true,
			}
		}

		var formattedActuals []string
		var expectedDesc string
		var failedReason string
		passed := true

		for i, elem := range arrVal {
			var valToCheck interface{} = elem
			if fieldName != "" {
				mapVal, isMap := elem.(map[string]interface{})
				if !isMap {
					passed = false
					failedReason = fmt.Sprintf("failed at index %d (element is not an object)", i)
					formattedActuals = append(formattedActuals, fmt.Sprintf("%v", elem))
					break
				}
				val, valFound := mapVal[fieldName]
				if !valFound {
					passed = false
					failedReason = fmt.Sprintf("failed at index %d (field %q missing)", i, fieldName)
					formattedActuals = append(formattedActuals, "(missing)")
					break
				}
				valToCheck = val
			}

			// Try Matcher first
			res, used := evaluateMatcher("", expectedStr, valToCheck)
			if used {
				expectedDesc = res.Expected
				if !res.Passed {
					passed = false
					failedReason = fmt.Sprintf("failed at index %d (got %s)", i, formatActual(valToCheck))
					formattedActuals = append(formattedActuals, formatActual(valToCheck))
					break
				}
			} else {
				// Fallback to Equality
				expectedDesc = fmt.Sprintf("all elements equal %q", expectedStr)
				actualStr := fmt.Sprintf("%v", valToCheck)
				if actualStr != expectedStr {
					passed = false
					failedReason = fmt.Sprintf("failed at index %d (got %q)", i, actualStr)
					formattedActuals = append(formattedActuals, actualStr)
					break
				}
			}
			formattedActuals = append(formattedActuals, formatActual(valToCheck))
		}

		// If failed, populate the rest of formattedActuals quickly for display
		if !passed {
			for j := len(formattedActuals); j < len(arrVal); j++ {
				elem := arrVal[j]
				if fieldName != "" {
					if mapVal, isMap := elem.(map[string]interface{}); isMap {
						if val, valFound := mapVal[fieldName]; valFound {
							formattedActuals = append(formattedActuals, formatActual(val))
							continue
						}
					}
				}
				formattedActuals = append(formattedActuals, formatActual(elem))
			}
		}

		if expectedDesc == "" {
			dummyResult, used := evaluateMatcher("", expectedStr, nil)
			if used {
				expectedDesc = dummyResult.Expected
			} else {
				expectedDesc = fmt.Sprintf("all elements equal %q", expectedStr)
			}
		}

		return AssertionResult{
			Path:     path,
			Expected: expectedDesc,
			Actual:   "[" + strings.Join(formattedActuals, ", ") + "]",
			Passed:   passed,
			Reason:   failedReason,
		}
	}

	// JWT assertions
	if strings.HasPrefix(path, "jwt.") {
		claimPath := strings.TrimPrefix(path, "jwt.")
		actualVal, found := resolvePath(jwtClaims, claimPath)
		if !found {
			// not exists — field absent is a pass
			if expectedStr == "not exists" {
				return AssertionResult{
					Path:     path,
					Expected: "not exists",
					Actual:   "(not found)",
					Passed:   true,
				}
			}
			return AssertionResult{
				Path:        path,
				Expected:    expectedStr,
				Actual:      "(not found)",
				Passed:      expectedStr == "exists" && false,
				Suggestions: getAvailableFields(jwtClaims),
			}
		}

		// not exists — field found is a fail
		if expectedStr == "not exists" {
			return AssertionResult{
				Path:     path,
				Expected: "not exists",
				Actual:   fmt.Sprintf("%v", actualVal),
				Passed:   false,
			}
		}

		// Try Matchers
		if result, used := evaluateMatcher(path, expectedStr, actualVal); used {
			return result
		}

		// Equality check
		actualStr := fmt.Sprintf("%v", actualVal)
		return AssertionResult{
			Path:     path,
			Expected: expectedStr,
			Actual:   actualStr,
			Passed:   actualStr == expectedStr,
		}
	}

	// Headers assertions (canonical: headers.X)
	if strings.HasPrefix(path, "headers.") {
		headerName := strings.TrimPrefix(path, "headers.")
		actualVal, ok := resp.Headers[headerName]
		if !ok {
			// not exists — field absent is a pass
			if expectedStr == "not exists" {
				return AssertionResult{
					Path:     path,
					Expected: "not exists",
					Actual:   "(not found)",
					Passed:   true,
				}
			}
			available := make([]string, 0, len(resp.Headers))
			for k := range resp.Headers {
				available = append(available, k)
			}
			sort.Strings(available)
			return AssertionResult{
				Path:        path,
				Expected:    expectedStr,
				Actual:      "(not found)",
				Passed:      expectedStr == "exists" && false,
				Suggestions: available,
			}
		}
		// not exists — field found is a fail
		if expectedStr == "not exists" {
			return AssertionResult{
				Path:     path,
				Expected: "not exists",
				Actual:   actualVal,
				Passed:   false,
			}
		}
		// Try Matchers
		if result, used := evaluateMatcher(path, expectedStr, actualVal); used {
			return result
		}
		return AssertionResult{Path: path, Expected: expectedStr, Actual: actualVal, Passed: actualVal == expectedStr}
	}

	// Body assertions (canonical: body.<field>)
	if strings.HasPrefix(path, "body.") {
		bodyPath := strings.TrimPrefix(path, "body.")

		if resp.Parsed == nil {
			if expectedStr == "not exists" {
				return AssertionResult{
					Path:     path,
					Expected: "not exists",
					Actual:   "(body not parsed)",
					Passed:   true,
				}
			}
			return AssertionResult{
				Path:     path,
				Expected: expectedStr,
				Actual:   "(body not parsed)",
				Passed:   expectedStr == "exists" && false,
			}
		}

		actualVal, found := resolvePath(resp.Parsed, bodyPath)
		if !found {
			// not exists — field absent is a pass
			if expectedStr == "not exists" {
				return AssertionResult{
					Path:     path,
					Expected: "not exists",
					Actual:   "(not found)",
					Passed:   true,
				}
			}
			return AssertionResult{
				Path:        path,
				Expected:    expectedStr,
				Actual:      "(not found)",
				Passed:      expectedStr == "exists" && false,
				Suggestions: getAvailableFields(resp.Parsed),
			}
		}

		// not exists — field found is a fail
		if expectedStr == "not exists" {
			return AssertionResult{
				Path:     path,
				Expected: "not exists",
				Actual:   fmt.Sprintf("%v", actualVal),
				Passed:   false,
			}
		}

		// Try Matchers
		if result, used := evaluateMatcher(path, expectedStr, actualVal); used {
			return result
		}

		actualStr := fmt.Sprintf("%v", actualVal)
		return AssertionResult{
			Path:     path,
			Expected: expectedStr,
			Actual:   actualStr,
			Passed:   actualStr == expectedStr,
		}
	}

	// Response.* assertions (backward-compatible, delegates to evaluateResponseAssertion)
	if strings.HasPrefix(path, "response.") {
		return evaluateResponseAssertion(path, expectedStr, resp)
	}

	// Fallback: try direct body path (backward-compatible)
	if resp.Parsed != nil {
		actualVal, found := resolvePath(resp.Parsed, path)
		if found {
			if expectedStr == "exists" {
				return AssertionResult{
					Path:     path,
					Expected: "exists",
					Actual:   fmt.Sprintf("%v", actualVal),
					Passed:   true,
				}
			}
			actualStr := fmt.Sprintf("%v", actualVal)
			return AssertionResult{
				Path:     path,
				Expected: expectedStr,
				Actual:   actualStr,
				Passed:   actualStr == expectedStr,
			}
		}
	}

	return AssertionResult{
		Path:        path,
		Expected:    expectedStr,
		Actual:      "(not found)",
		Passed:      false,
		Suggestions: getAvailableFields(resp.Parsed),
	}
}

// evaluateResponseAssertion handles "response.X" paths for headers etc.
func evaluateResponseAssertion(path, expected string, resp *ResponseInfo) AssertionResult {
	// Handle response.headers.X
	if strings.HasPrefix(path, "response.headers.") {
		headerName := strings.TrimPrefix(path, "response.headers.")
		actualVal, ok := resp.Headers[headerName]
		if !ok {
			available := make([]string, 0, len(resp.Headers))
			for k := range resp.Headers {
				available = append(available, k)
			}
			sort.Strings(available)
			return AssertionResult{
				Path:        path,
				Expected:    expected,
				Actual:      "(not found)",
				Passed:      expected == "exists" && false,
				Suggestions: available,
			}
		}
		// Try Matchers
		if result, used := evaluateMatcher(path, expected, actualVal); used {
			return result
		}
		return AssertionResult{Path: path, Expected: expected, Actual: actualVal, Passed: actualVal == expected}
	}

	// Handle response.body.X
	if strings.HasPrefix(path, "response.body.") || strings.HasPrefix(path, "response.") {
		bodyPath := strings.TrimPrefix(path, "response.")
		bodyPath = strings.TrimPrefix(bodyPath, "body.")

		if resp.Parsed == nil {
			return AssertionResult{
				Path:     path,
				Expected: expected,
				Actual:   "(body not parsed)",
				Passed:   expected == "exists" && false,
			}
		}

		actualVal, found := resolvePath(resp.Parsed, bodyPath)
		if !found {
			return AssertionResult{
				Path:        path,
				Expected:    expected,
				Actual:      "(not found)",
				Passed:      expected == "exists" && false,
				Suggestions: getAvailableFields(resp.Parsed),
			}
		}

		if expected == "exists" {
			return AssertionResult{Path: path, Expected: "exists", Actual: fmt.Sprintf("%v", actualVal), Passed: true}
		}

		actualStr := fmt.Sprintf("%v", actualVal)
		return AssertionResult{Path: path, Expected: expected, Actual: actualStr, Passed: actualStr == expected}
	}

	return AssertionResult{Path: path, Expected: expected, Actual: "(unresolved)", Passed: false}
}

// resolvePath navigates a nested structure using dot-notation (e.g. "data.id", "items[0].name").
func resolvePath(data interface{}, path string) (interface{}, bool) {
	if data == nil {
		return nil, false
	}

	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		// Handle array index: "items[0]"
		if idxStart := strings.Index(part, "["); idxStart >= 0 {
			fieldName := part[:idxStart]
			idxStr := part[idxStart+1 : len(part)-1]
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				return nil, false
			}

			// Navigate into the field
			if fieldName != "" {
				mapVal, ok := current.(map[string]interface{})
				if !ok {
					return nil, false
				}
				current, ok = mapVal[fieldName]
				if !ok {
					return nil, false
				}
			}

			// Navigate into array
			arrVal, ok := current.([]interface{})
			if !ok || idx >= len(arrVal) {
				return nil, false
			}
			current = arrVal[idx]
		} else {
			mapVal, ok := current.(map[string]interface{})
			if !ok {
				return nil, false
			}
			val, ok := mapVal[part]
			if !ok {
				return nil, false
			}
			current = val
		}
	}

	return current, true
}

// extractValues saves values from the response or request into a variables map.
// Supported path prefixes:
//   - body.<field>          → response JSON body (canonical)
//   - response.body.<field> → same as body.<field>
//   - response.<field>      → backward-compatible alias for body.<field>
//   - request.body.<field>  → interpolated request body
//   - jwt.<claim>           → decoded JWT claim
func extractValues(vars map[string]interface{}, save map[string]string, resp *ResponseInfo, jwtClaims map[string]interface{}, requestBody interface{}) {
	for name, path := range save {
		// Interpolate the path to resolve variables like $randomInt(1,10) or $previousVar
		// before using it as a path expression. In practice the save key acts as the
		// variable name and the path supports variables inside array indexes etc.
		interpolatedPath, err := interpolateString(path, vars)
		if err == nil {
			path = interpolatedPath
		}
		// If interpolation fails (e.g. undefined var), fall back to the original path

		switch {
		case strings.HasPrefix(path, "jwt."):
			claimPath := strings.TrimPrefix(path, "jwt.")
			val, found := resolvePath(jwtClaims, claimPath)
			if found {
				vars[name] = val
			}

		case strings.HasPrefix(path, "response.body."):
			// response.body.<field> → resolve against body root
			bodyPath := strings.TrimPrefix(path, "response.body.")
			if resp.Parsed != nil {
				val, found := resolvePath(resp.Parsed, bodyPath)
				if found {
					vars[name] = val
				}
			}

		case strings.HasPrefix(path, "body."):
			// body.<field> → resolve against response body root (canonical)
			bodyPath := strings.TrimPrefix(path, "body.")
			if resp.Parsed != nil {
				val, found := resolvePath(resp.Parsed, bodyPath)
				if found {
					vars[name] = val
				}
			}

		case strings.HasPrefix(path, "request.body."):
			// request.body.<field> → resolve against interpolated request body
			bodyPath := strings.TrimPrefix(path, "request.body.")
			if requestBody != nil {
				if parsed, ok := requestBody.(map[string]interface{}); ok {
					val, found := resolvePath(parsed, bodyPath)
					if found {
						vars[name] = val
					}
				}
			}

		case strings.HasPrefix(path, "response."):
			// response.<field> → backward-compatible alias for body.<field>
			bodyPath := strings.TrimPrefix(path, "response.")
			if resp.Parsed != nil {
				val, found := resolvePath(resp.Parsed, bodyPath)
				if found {
					vars[name] = val
				}
			}
		}
	}
}

// decodeJWT parses the JWT payload without verification and returns the claims.
func decodeJWT(tokenString string) (map[string]interface{}, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	// URL-safe base64 decode the payload
	payload := parts[1]
	// Add padding
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	// Replace URL-safe chars
	payload = strings.ReplaceAll(payload, "-", "+")
	payload = strings.ReplaceAll(payload, "_", "/")

	decoded, err := base64Decode(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse JWT claims: %w", err)
	}

	return claims, nil
}

func base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// comparatorLabel returns a human-readable label for count() comparator suffixes.
func comparatorLabel(c string) string {
	switch c {
	case "gte":
		return ">="
	case "gt":
		return ">"
	case "lte":
		return "<="
	case "lt":
		return "<"
	default:
		return "=="
	}
}
