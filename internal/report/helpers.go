package report

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/muhfaris/gherkio/internal/runner"
)

// generateCurl builds a copy-pasteable curl command from a RequestInfo.
func generateCurl(req *runner.RequestInfo, maskFields []string) string {
	if req == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("curl -X %s '%s'", req.Method, req.URL))

	for k, v := range req.Headers {
		if containsFold(maskFields, k) {
			v = "***masked***"
		}
		sb.WriteString(fmt.Sprintf(" -H '%s: %s'", k, v))
	}

	if req.Body != "" && req.Method != "GET" && req.Method != "HEAD" {
		var parsed interface{}
		if err := json.Unmarshal([]byte(req.Body), &parsed); err == nil {
			if len(maskFields) > 0 {
				parsed = runner.MaskSensitiveData(parsed, maskFields)
			}
			minified, _ := json.Marshal(parsed)
			sb.WriteString(fmt.Sprintf(" -d '%s'", string(minified)))
		} else {
			sb.WriteString(fmt.Sprintf(" -d '%s'", strings.ReplaceAll(req.Body, "'", "'\\''")))
		}
	}

	return sb.String()
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

// extractRequestId scans headers for common request tracing IDs.
func extractRequestId(headers map[string]string) string {
	targetHeaders := []string{
		"x-request-id",
		"x-trace-id",
		"request-id",
		"requestid",
		"x-correlation-id",
	}

	// Create case-insensitive map for easier lookup
	lowerHeaders := make(map[string]string)
	for k, v := range headers {
		lowerHeaders[strings.ToLower(k)] = v
	}

	for _, target := range targetHeaders {
		if val, ok := lowerHeaders[target]; ok {
			return val
		}
	}

	return ""
}
