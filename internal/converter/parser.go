package converter

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// ParsedRequest represents the HTTP request properties extracted from a cURL command.
type ParsedRequest struct {
	Name    string
	Method  string
	URL     string
	Query   map[string]string
	Headers map[string]string
	Body    interface{}
	Timeout string
}

// Tokenize splits a shell command string into separate arguments, correctly
// respecting single quotes, double quotes, line continuations, and escaped characters.
func Tokenize(s string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false

	for i := 0; i < len(s); i++ {
		r := s[i]

		if escaped {
			current.WriteByte(r)
			escaped = false
			continue
		}

		if r == '\\' {
			// Handle line continuation: backslash followed by newline
			if i+1 < len(s) && s[i+1] == '\n' {
				i++ // skip backslash and newline
				continue
			}
			if inSingleQuote {
				current.WriteByte(r)
			} else {
				escaped = true
			}
			continue
		}

		if inSingleQuote {
			if r == '\'' {
				inSingleQuote = false
			} else {
				current.WriteByte(r)
			}
			continue
		}

		if inDoubleQuote {
			if r == '"' {
				inDoubleQuote = false
			} else {
				current.WriteByte(r)
			}
			continue
		}

		switch r {
		case '\'':
			inSingleQuote = true
		case '"':
			inDoubleQuote = true
		case ' ', '\t', '\n', '\r':
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(r)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens, nil
}

// isArgFlag returns true if the flag is known to take an argument.
func isArgFlag(flag string) bool {
	argFlags := map[string]bool{
		"-X":                true,
		"--request":         true,
		"-H":                true,
		"--header":          true,
		"-d":                true,
		"--data":            true,
		"--data-raw":        true,
		"--data-ascii":      true,
		"--data-binary":     true,
		"--data-json":       true,
		"-u":                true,
		"--user":            true,
		"-b":                true,
		"--cookie":          true,
		"-A":                true,
		"--user-agent":      true,
		"-e":                true,
		"--referer":         true,
		"--max-time":        true,
		"-m":                true,
		"--connect-timeout": true,
		"--retry":           true,
		"--retry-delay":     true,
		"--retry-max-time":  true,
		"-o":                true,
		"--output":          true,
	}
	return argFlags[flag]
}

// ParseCurl tokenizes a cURL string and extracts request properties, returning
// a ParsedRequest and a slice of warnings for ignored flags.
func ParseCurl(cmdStr string) (*ParsedRequest, []string, error) {
	tokens, err := Tokenize(strings.TrimSpace(cmdStr))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to tokenize cURL command: %w", err)
	}

	if len(tokens) == 0 {
		return nil, nil, fmt.Errorf("empty cURL command")
	}

	// Skip initial "curl" token if present
	startIndex := 0
	if strings.ToLower(tokens[0]) == "curl" {
		startIndex = 1
	}

	req := &ParsedRequest{
		Method:  "",
		URL:     "",
		Headers: make(map[string]string),
		Body:    nil,
		Timeout: "",
	}

	var warnings []string
	hasBodyFlag := false

	for i := startIndex; i < len(tokens); i++ {
		tok := tokens[i]

		// Skip empty tokens
		if tok == "" {
			continue
		}

		if strings.HasPrefix(tok, "-") {
			// It's a flag
			takesArg := isArgFlag(tok)
			var argVal string
			if takesArg {
				if i+1 < len(tokens) {
					argVal = tokens[i+1]
					i++ // consume the argument
				} else {
					return nil, nil, fmt.Errorf("flag %s requires an argument", tok)
				}
			}

			switch tok {
			case "-X", "--request":
				req.Method = strings.ToUpper(argVal)
			case "-H", "--header":
				// Split header into Key: Value
				parts := strings.SplitN(argVal, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					val := strings.TrimSpace(parts[1])
					req.Headers[key] = val
				} else {
					warnings = append(warnings, fmt.Sprintf("Malformed header ignored: %q", argVal))
				}
			case "-d", "--data", "--data-raw", "--data-ascii", "--data-binary", "--data-json":
				hasBodyFlag = true
				// Attempt to parse JSON body
				trimmed := strings.TrimSpace(argVal)
				if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
					(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
					var jsonParsed interface{}
					if err := json.Unmarshal([]byte(argVal), &jsonParsed); err == nil {
						req.Body = jsonParsed
					} else {
						req.Body = argVal
					}
				} else {
					req.Body = argVal
				}
			case "-u", "--user":
				// Convert user:password basic auth to Authorization header
				encoded := base64.StdEncoding.EncodeToString([]byte(argVal))
				req.Headers["Authorization"] = "Basic " + encoded
			case "-b", "--cookie":
				req.Headers["Cookie"] = argVal
			case "--max-time", "-m":
				// Map simple integer timeout to seconds Go duration format
				if strings.ContainsAny(argVal, "smh") {
					req.Timeout = argVal
				} else {
					req.Timeout = argVal + "s"
				}
			default:
				// Unhandled flags
				if takesArg {
					warnings = append(warnings, fmt.Sprintf("Ignored flag with argument: %s %s", tok, argVal))
				} else {
					warnings = append(warnings, fmt.Sprintf("Ignored flag: %s", tok))
				}
			}
		} else {
			// Positional parameter: treat as URL (keep the last one encountered, or first non-empty)
			req.URL = tok
		}
	}

	if req.URL == "" {
		return nil, nil, fmt.Errorf("no URL found in cURL command")
	}

	// Heuristics:
	// 1. If method is not explicitly set but a body exists, default to POST
	if req.Method == "" {
		if hasBodyFlag {
			req.Method = "POST"
		} else {
			req.Method = "GET"
		}
	}

	// 2. Ensure application/json header is set if body is JSON
	if req.Body != nil && req.Headers["Content-Type"] == "" {
		switch req.Body.(type) {
		case map[string]interface{}, []interface{}:
			req.Headers["Content-Type"] = "application/json"
		}
	}

	return req, warnings, nil
}
