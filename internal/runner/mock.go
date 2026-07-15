package runner

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/muhfaris/gherkio/internal/model"
)

// matchMock checks if a request matches the configured mock rule criteria.
func matchMock(reqMethod, reqURL string, mock model.MockRule) bool {
	if strings.ToUpper(reqMethod) != strings.ToUpper(mock.Request.Method) {
		return false
	}

	mockURL := mock.Request.URL
	// Remove query parameters from reqURL for comparison
	cleanReqURL := reqURL
	if idx := strings.Index(cleanReqURL, "?"); idx != -1 {
		cleanReqURL = cleanReqURL[:idx]
	}

	if cleanReqURL == mockURL {
		return true
	}
	if strings.HasSuffix(cleanReqURL, mockURL) {
		return true
	}
	if strings.Contains(cleanReqURL, mockURL) {
		return true
	}
	return false
}

// interpolateMockBody processes mock response body structure to interpolate request parameters.
func interpolateMockBody(bodyVal interface{}, reqBody interface{}, reqHeaders map[string]string) interface{} {
	switch v := bodyVal.(type) {
	case string:
		return interpolateMockString(v, reqBody, reqHeaders)
	case map[string]interface{}:
		res := make(map[string]interface{}, len(v))
		for k, val := range v {
			res[k] = interpolateMockBody(val, reqBody, reqHeaders)
		}
		return res
	case []interface{}:
		res := make([]interface{}, len(v))
		for i, val := range v {
			res[i] = interpolateMockBody(val, reqBody, reqHeaders)
		}
		return res
	default:
		return v
	}
}

// interpolateMockString interpolates a string against request body and headers.
func interpolateMockString(s string, reqBody interface{}, reqHeaders map[string]string) interface{} {
	if strings.HasPrefix(s, "$request.body.") {
		path := strings.TrimPrefix(s, "$request.body.")
		if val, found := resolvePath(reqBody, path); found {
			return val
		}
	}
	if strings.HasPrefix(s, "$request.headers.") {
		name := strings.TrimPrefix(s, "$request.headers.")
		for k, v := range reqHeaders {
			if strings.EqualFold(k, name) {
				return v
			}
		}
	}

	re := regexp.MustCompile(`\$request\.(body|headers)\.([a-zA-Z0-9_.-]+)`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}
		source := parts[1]
		path := parts[2]
		if source == "body" {
			if val, found := resolvePath(reqBody, path); found {
				return fmt.Sprintf("%v", val)
			}
		} else if source == "headers" {
			for k, v := range reqHeaders {
				if strings.EqualFold(k, path) {
					return v
				}
			}
		}
		return match
	})
}
