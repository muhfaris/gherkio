package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/muhfaris/gherkio/internal/model"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// StepResult holds the result of a single step execution.
type StepResult struct {
	Original     model.Step        `json:"original"`
	Depth        int               `json:"depth"`
	IsUseStart   bool              `json:"isUseStart"`
	IsUseEnd     bool              `json:"isUseEnd"`
	UseFile      string            `json:"useFile,omitempty"`
	ScenarioName string            `json:"scenarioName,omitempty"`
	TestFile     string            `json:"testFile,omitempty"`
	Request      *RequestInfo      `json:"request"`
	Response     *ResponseInfo     `json:"response"`
	Assertions   []AssertionResult `json:"assertions"`
	Duration     time.Duration     `json:"duration"`
	Error        string            `json:"error,omitempty"`
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

// executeRequest performs an HTTP request and returns the result.
func executeRequest(method, url string, headers map[string]string, body interface{}) (*ResponseInfo, error) {
	var bodyReader io.Reader
	if body != nil {
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

	// Auto-detect Content-Type when body is a JSON type and not explicitly set
	if body != nil && req.Header.Get("Content-Type") == "" {
		switch body.(type) {
		case map[string]interface{}, []interface{}:
			req.Header.Set("Content-Type", "application/json")
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
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

// evaluateTiming checks that step duration is within the expected max.
func evaluateTiming(actual time.Duration, maxStr string) AssertionResult {
	maxDuration, err := time.ParseDuration(maxStr)
	if err != nil {
		return AssertionResult{
			Path:     "timing.max",
			Expected: maxStr,
			Actual:   fmt.Sprintf("invalid: %s", err.Error()),
			Passed:   false,
		}
	}

	passed := actual <= maxDuration
	return AssertionResult{
		Path:     "timing.max",
		Expected: maxStr,
		Actual:   FormatDuration(actual),
		Passed:   passed,
	}
}

// runAssertions executes all assertions against the response.
func runAssertions(status int, resp *ResponseInfo, jwtClaims map[string]interface{}, expectStatus int, extra map[string]interface{}, projectDir string) []AssertionResult {
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
		result := evaluateAssertion(path, expectedVal, resp, jwtClaims, projectDir)
		results = append(results, result)
	}

	return results
}

// evaluateAssertion evaluates a single assertion at the given path.
func evaluateAssertion(path string, expected interface{}, resp *ResponseInfo, jwtClaims map[string]interface{}, projectDir string) AssertionResult {
	expectedStr := fmt.Sprintf("%v", expected)
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

		if len(violations) == 0 {
			return AssertionResult{
				Path:     "schema",
				Expected: expectedStr,
				Actual:   "valid",
				Passed:   true,
			}
		}

		var reasonBuilder strings.Builder
		for i, v := range violations {
			if i > 0 {
				reasonBuilder.WriteString("\n\n")
			}
			reasonBuilder.WriteString(fmt.Sprintf("actual: %s\nexpected: field %s %s %s\nreason: %s", v.Actual, v.Field, v.Rule, v.Expected, "validation failed"))
		}

		return AssertionResult{
			Path:     "schema",
			Expected: expectedStr,
			Actual:   "invalid",
			Passed:   false,
			Reason:   reasonBuilder.String(),
		}
	}

	// Collection Matchers: count(path)
	if strings.HasPrefix(path, "count(") && strings.HasSuffix(path, ")") {
		innerPath := path[6 : len(path)-1]
		actualVal, found := resolvePath(resp.Parsed, innerPath)
		if !found {
			return AssertionResult{
				Path:     path,
				Expected: fmt.Sprintf("exactly %s items", expectedStr),
				Actual:   "(not found)",
				Passed:   false,
			}
		}

		arrVal, ok := actualVal.([]interface{})
		if !ok {
			return AssertionResult{
				Path:     path,
				Expected: fmt.Sprintf("exactly %s items", expectedStr),
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

		reason := ""
		if actualLen != expectedLen {
			reason = fmt.Sprintf("array has %d items", actualLen)
		}

		return AssertionResult{
			Path:     path,
			Expected: fmt.Sprintf("exactly %d items", expectedLen),
			Actual:   fmt.Sprintf("%d", actualLen),
			Passed:   actualLen == expectedLen,
			Reason:   reason,
		}
	}

	// Collection Matchers: all(path)
	if strings.HasPrefix(path, "all(") && strings.HasSuffix(path, ")") {
		innerPath := path[4 : len(path)-1]

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
			return AssertionResult{
				Path:        path,
				Expected:    expectedStr,
				Actual:      "(not found)",
				Passed:      expectedStr == "exists" && false,
				Suggestions: getAvailableFields(jwtClaims),
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
			return AssertionResult{
				Path:     path,
				Expected: expectedStr,
				Actual:   "(body not parsed)",
				Passed:   expectedStr == "exists" && false,
			}
		}

		actualVal, found := resolvePath(resp.Parsed, bodyPath)
		if !found {
			return AssertionResult{
				Path:        path,
				Expected:    expectedStr,
				Actual:      "(not found)",
				Passed:      expectedStr == "exists" && false,
				Suggestions: getAvailableFields(resp.Parsed),
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

// extractValues saves values from the response into a variables map.
// Supported path prefixes:
//   - body.<field>          → response JSON body (canonical)
//   - response.body.<field> → same as body.<field>
//   - response.<field>      → backward-compatible alias for body.<field>
//   - jwt.<claim>           → decoded JWT claim
func extractValues(vars map[string]interface{}, save map[string]string, resp *ResponseInfo, jwtClaims map[string]interface{}) {
	for name, path := range save {
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
			// body.<field> → resolve against body root (canonical)
			bodyPath := strings.TrimPrefix(path, "body.")
			if resp.Parsed != nil {
				val, found := resolvePath(resp.Parsed, bodyPath)
				if found {
					vars[name] = val
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
	// Use standard base64 instead of custom
	decoded := make([]byte, len(s))
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z':
			decoded[n] = c - 'A'
		case c >= 'a' && c <= 'z':
			decoded[n] = c - 'a' + 26
		case c >= '0' && c <= '9':
			decoded[n] = c - '0' + 52
		case c == '+':
			decoded[n] = 62
		case c == '/':
			decoded[n] = 63
		default:
			continue
		}
		n++
	}

	// Decode sextets into bytes
	result := make([]byte, 0, n*3/4)
	for i := 0; i+3 < n; i += 4 {
		val := (int(decoded[i]) << 18) | (int(decoded[i+1]) << 12) | (int(decoded[i+2]) << 6) | int(decoded[i+3])
		result = append(result, byte(val>>16), byte(val>>8), byte(val))
	}

	// Handle remaining bytes
	remaining := n % 4
	if remaining >= 2 {
		val := int(decoded[n-2])<<6 | int(decoded[n-1])
		result = append(result, byte(val>>4))
		if remaining == 3 {
			val = (int(decoded[n-3]) << 12) | (int(decoded[n-2]) << 6) | int(decoded[n-1])
			result = append(result, byte(val>>10), byte(val>>2))
		}
	}

	return result, nil
}
