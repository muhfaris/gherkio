package runner

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// defaultSensitiveFields are built-in field names whose values are masked in output.
var defaultSensitiveFields = []string{
	"token",
	"accessToken",
	"access_token",
	"refreshToken",
	"refresh_token",
	"password",
	"secret",
	"clientSecret",
	"client_secret",
	"apiKey",
	"api_key",
	"authorization",
}

// maskSensitiveData recursively walks parsed JSON and replaces sensitive field values.
func MaskSensitiveData(data interface{}, fields []string) interface{} {
	if len(fields) == 0 {
		return data
	}

	switch v := data.(type) {
	case map[string]interface{}:
		masked := make(map[string]interface{}, len(v))
		for key, val := range v {
			if isSensitiveField(key, fields) {
				masked[key] = "***masked***"
			} else {
				masked[key] = MaskSensitiveData(val, fields)
			}
		}
		return masked
	case []interface{}:
		masked := make([]interface{}, len(v))
		for i, val := range v {
			masked[i] = MaskSensitiveData(val, fields)
		}
		return masked
	default:
		return data
	}
}

// isSensitiveField checks if a field name matches any sensitive field (case-insensitive).
func isSensitiveField(name string, fields []string) bool {
	lower := strings.ToLower(name)
	for _, f := range fields {
		if strings.ToLower(f) == lower {
			return true
		}
	}
	return false
}

// stepSpacing returns the spacing between indent and status/assertion text.
// Top-level steps align with "1. " (3 chars), nested steps align with "├ " (2 chars after indent prefix).
func stepSpacing(depth int) string {
	if depth > 0 {
		return " " // 1 space after "│" aligns with "├ " prefix
	}
	return "   " // 3 spaces aligns with "1. " prefix
}

// indentPrefix removes the last visual segment (3 spaces + 1 pipe) from an indent string.
// Uses rune-aware slicing to handle multi-byte Unicode characters like │.
func indentPrefix(indent string) string {
	if indent == "" {
		return ""
	}
	runes := []rune(indent)
	if len(runes) >= 4 {
		return string(runes[:len(runes)-4])
	}
	return indent
}

// formatRequestBody pretty-prints a JSON body, optionally masking sensitive fields.
func FormatRequestBody(body string, maskFields []string) string {
	var parsed interface{}
	if err := json.Unmarshal([]byte(body), &parsed); err == nil {
		if len(maskFields) > 0 {
			parsed = MaskSensitiveData(parsed, maskFields)
		}
		pretty, _ := json.MarshalIndent(parsed, "", "  ")
		return string(pretty)
	}
	return body
}

// PrintResult formats and prints the execution result to stdout.
// When verbose is true, full request/response payloads are shown.
func PrintResult(result *RunResult, verbose bool, maskFields []string) {
	// Use built-in defaults when no mask fields specified
	if maskFields == nil {
		maskFields = defaultSensitiveFields
	}
	statusIcon := "✓"
	statusWord := "PASS"

	if !result.Passed {
		statusIcon = "✗"
		statusWord = "FAIL"
	}

	fmt.Printf("\n%s %s\n\n", statusIcon, result.Scenario)

	stepCounter := 1
	var lastRole string
	for i, step := range result.Steps {
		// Print section header when role changes
		if step.Role != lastRole && (step.Role == "setup" || step.Role == "teardown") {
			if lastRole != "" {
				fmt.Println()
			}
			sectionName := "Setup"
			if step.Role == "teardown" {
				sectionName = "Teardown"
			}
			fmt.Printf("── %s ──\n\n", sectionName)
			lastRole = step.Role
		}

		indent := strings.Repeat("   │", step.Depth)
		statusIndent := indent + stepSpacing(step.Depth)

		if step.IsUseStart {
			prefix := fmt.Sprintf("%d. └─ ", stepCounter)
			if step.Depth > 0 {
				prefix = indentPrefix(indent) + "   ├ └─ "
			} else {
				stepCounter++
			}
			fmt.Printf("%suse: %s\n%s   │\n", prefix, step.UseFile, indent)
			continue
		}
		if step.IsUseEnd {
			// Determine if the use block succeeded
			// (This is tricky since we flattened it, but let's just assume if there's no error it succeeded - actually we shouldn't print success for the block, the individual steps show it)
			if step.Depth > 0 {
				fmt.Printf("%s\n", indentPrefix(indent))
			} else {
				fmt.Printf("\n")
			}
			continue
		}

		// Determine step-level pass/fail
		stepPassed := step.Error == ""
		for _, a := range step.Assertions {
			if !a.Passed {
				stepPassed = false
				break
			}
		}

		// Step header with number and status
		var stepLabel string
		if step.Request != nil {
			stepLabel = fmt.Sprintf("%s %s", step.Request.Method, step.Request.URL)
		} else {
			if step.Original.Request.URL != "" {
				stepLabel = fmt.Sprintf("%s %s (failed before execution)", step.Original.Request.Method, step.Original.Request.URL)
			} else {
				stepLabel = "Unknown Step"
			}
		}

		prefix := fmt.Sprintf("%d. ", stepCounter)
		if step.Depth > 0 {
			prefix = indentPrefix(indent) + "   ├ "
		} else {
			stepCounter++
		}

		fmt.Printf("%s%s\n", prefix, stepLabel)
		if stepPassed {
			fmt.Printf("%s✓ success\n", statusIndent)
		} else {
			fmt.Printf("%s✗ failed\n", statusIndent)
		}

		if step.RetryCount > 0 {
			if stepPassed {
				fmt.Printf("%s└─ retry: %d, last at retry %d\n", statusIndent, step.RetryCount, step.RetryCount)
			} else {
				lastError := ""
				if step.Error != "" {
					lastError = fmt.Sprintf("\n%s└─ last error: %s", statusIndent, step.Error)
				} else if step.Response != nil {
					lastError = fmt.Sprintf("\n%s└─ last response: status %d, body = %q", statusIndent, step.Response.Status, step.Response.Body)
				}

				isTimeout := strings.Contains(step.Error, "maxDuration")
				if isTimeout {
					fmt.Printf("%s└─ retry: %d/%d, %s%s\n", statusIndent, step.RetryCount, step.Original.Retry.Attempts, step.Error, lastError)
				} else {
					fmt.Printf("%s└─ retry: %d exhausted (%d attempts)%s\n", statusIndent, step.RetryCount, step.Original.Retry.Attempts, lastError)
				}
			}
		}

		if verbose {
			// ── Verbose: full request/response payloads ──
			fmt.Println()

			// Request details
			if step.Request != nil {
				fmt.Printf("Request:\n%s %s\n", step.Request.Method, step.Request.URL)
				if step.Request.Body != "" {
					fmt.Printf("Body: %s\n", FormatRequestBody(step.Request.Body, maskFields))
				}
				fmt.Println()
			}

			// Response (always shown in verbose)
			if step.Response != nil {
				fmt.Printf("Response:\nStatus: %d\n\nBody:\n%s\n", step.Response.Status, FormatRequestBody(step.Response.Body, maskFields))
				fmt.Println()
			}

			// Error (if request failed entirely)
			if step.Error != "" {
				fmt.Printf("  ✗ Error: %s\n", step.Error)
				fmt.Println()
			}

			// Assertions with full details
			if len(step.Assertions) > 0 {
				fmt.Println("Assertions:")
				for _, a := range step.Assertions {
					icon := "✓"
					if !a.Passed {
						icon = "✗"
					}

					// Timing assertions always show actual value
					isTiming := a.Path == "timing.max"

					if a.Expected == "exists" {
						if isTiming {
							fmt.Printf("  %s %s %s (actual: %s)\n", icon, a.Path, a.Expected, a.Actual)
						} else {
							fmt.Printf("  %s %s %s\n", icon, a.Path, a.Expected)
						}
					} else if isTiming {
						fmt.Printf("  %s %s = %s (actual: %s)\n", icon, a.Path, a.Expected, a.Actual)
					} else {
						if strings.HasPrefix(a.Expected, "contains ") || strings.HasPrefix(a.Expected, "startsWith ") || strings.HasPrefix(a.Expected, "endsWith ") || strings.HasPrefix(a.Expected, "pattern ") {
							fmt.Printf("  %s %s %s (actual: %s)\n", icon, a.Path, a.Expected, a.Actual)
						} else if strings.HasPrefix(a.Expected, "exactly") || strings.HasPrefix(a.Expected, "all elements") {
							fmt.Printf("  %s %s = %s (actual: %s)\n", icon, a.Path, a.Expected, a.Actual)
						} else if isMatcherKeyword(a.Expected) || a.Reason != "" {
							fmt.Printf("  %s %s = %s (actual: %s)\n", icon, a.Path, a.Expected, a.Actual)
						} else {
							fmt.Printf("  %s %s = %s\n", icon, a.Path, a.Expected)
						}
					}

					if !a.Passed {
						if strings.HasPrefix(a.Actual, "(not found)") || a.Actual == "(unresolved)" {
							fmt.Printf("      └─ path not found\n")
							if len(a.Suggestions) > 0 {
								fmt.Println()
								fmt.Println("Available fields:")
								for _, f := range a.Suggestions {
									fmt.Printf("  - %s\n", f)
								}
							}
						} else {
							if a.Reason != "" {
								fmt.Printf("      └─ actual: %s\n", a.Actual)
								fmt.Printf("      └─ expected: %s\n", a.Expected)
								fmt.Printf("      └─ reason: %s\n", a.Reason)
							} else {
								fmt.Printf("      └─ got: %s\n", a.Actual)
							}
						}
					}
				}
			}
		} else {
			// ── Summary: compact assertions, no payloads ──
			if len(step.Assertions) > 0 {
				for _, a := range step.Assertions {
					icon := "✓"
					if !a.Passed {
						icon = "✗"
					}

					// Timing assertions always show actual value
					isTiming := a.Path == "timing.max"

					if a.Expected == "exists" {
						if isTiming {
							fmt.Printf("%s%s %s %s (actual: %s)\n", statusIndent, icon, a.Path, a.Expected, a.Actual)
						} else {
							fmt.Printf("%s%s %s %s\n", statusIndent, icon, a.Path, a.Expected)
						}
					} else if isTiming {
						fmt.Printf("%s%s %s = %s (actual: %s)\n", statusIndent, icon, a.Path, a.Expected, a.Actual)
					} else {
						if strings.HasPrefix(a.Expected, "contains ") || strings.HasPrefix(a.Expected, "startsWith ") || strings.HasPrefix(a.Expected, "endsWith ") || strings.HasPrefix(a.Expected, "pattern ") {
							fmt.Printf("%s%s %s %s (actual: %s)\n", statusIndent, icon, a.Path, a.Expected, a.Actual)
						} else if strings.HasPrefix(a.Expected, "exactly") || strings.HasPrefix(a.Expected, "all elements") {
							fmt.Printf("%s%s %s = %s (actual: %s)\n", statusIndent, icon, a.Path, a.Expected, a.Actual)
						} else if isMatcherKeyword(a.Expected) || a.Reason != "" {
							fmt.Printf("%s%s %s = %s (actual: %s)\n", statusIndent, icon, a.Path, a.Expected, a.Actual)
						} else {
							fmt.Printf("%s%s %s = %s\n", statusIndent, icon, a.Path, a.Expected)
						}
					}

					// Inline failure info in summary
					if !a.Passed {
						failureIndent := statusIndent + "    "
						if strings.HasPrefix(a.Actual, "(not found)") || a.Actual == "(unresolved)" {
							fmt.Printf("%s└─ path not found", failureIndent)
							if len(a.Suggestions) > 0 {
								fmt.Printf(" (available: %s)", strings.Join(a.Suggestions, ", "))
							}
							fmt.Println()
						} else {
							if a.Reason != "" {
								fmt.Printf("%s└─ actual: %s\n", failureIndent, a.Actual)
								fmt.Printf("%s└─ expected: %s\n", failureIndent, a.Expected)
								fmt.Printf("%s└─ reason: %s\n", failureIndent, a.Reason)
							} else {
								fmt.Printf("%s└─ got: %s\n", failureIndent, a.Actual)
							}
						}
					}
				}
			}

			// Show response body only on failure in summary mode
			if !stepPassed && step.Response != nil {
				fmt.Printf("\nResponse:\nStatus: %d\n\nBody:\n%s\n", step.Response.Status, FormatRequestBody(step.Response.Body, maskFields))
			}

			if step.Error != "" {
				fmt.Printf("%s   ✗ Error: %s\n", indent, step.Error)
			}
		}

		// Separator only if this is a top-level step and not the last one, or if it's the end of a top level block.
		// To keep it simple, we just print the separator if the next step is a top level step.
		if i < len(result.Steps)-1 {
			nextStep := result.Steps[i+1]
			if nextStep.Depth == 0 && !nextStep.IsUseStart && !nextStep.IsUseEnd {
				fmt.Println(strings.Repeat("─", 40))
			}
		}
	}

	fmt.Println()
	fmt.Printf("%s %s\n", statusIcon, statusWord)

	// Summary line
	total := result.TotalPass + result.TotalFail
	summary := fmt.Sprintf("%d passed, %d failed, %d total", result.TotalPass, result.TotalFail, total)
	fmt.Printf("%s\n", summary)

	fmt.Printf("Duration: %s\n", FormatDuration(result.Duration))
	fmt.Println()
}
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		secs := d.Seconds()
		return fmt.Sprintf("%.1fs", secs)
	}
	return d.Round(time.Second).String()
}

// GetDefaultSensitiveFields returns the default sensitive fields
func GetDefaultSensitiveFields() []string {
	return defaultSensitiveFields
}
