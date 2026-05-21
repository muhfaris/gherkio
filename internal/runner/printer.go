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
func maskSensitiveData(data interface{}, fields []string) interface{} {
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
				masked[key] = maskSensitiveData(val, fields)
			}
		}
		return masked
	case []interface{}:
		masked := make([]interface{}, len(v))
		for i, val := range v {
			masked[i] = maskSensitiveData(val, fields)
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

// formatRequestBody pretty-prints a JSON body, optionally masking sensitive fields.
func formatRequestBody(body string, maskFields []string) string {
	var parsed interface{}
	if err := json.Unmarshal([]byte(body), &parsed); err == nil {
		if len(maskFields) > 0 {
			parsed = maskSensitiveData(parsed, maskFields)
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

	for i, step := range result.Steps {
		stepNum := i + 1

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
			stepLabel = "Nested Step"
		}
		fmt.Printf("%d. %s\n", stepNum, stepLabel)
		if stepPassed {
			fmt.Printf("   ✓ success\n")
		} else {
			fmt.Printf("   ✗ failed\n")
		}

		if verbose {
			// ── Verbose: full request/response payloads ──
			fmt.Println()

			// Request details
			if step.Request != nil {
				fmt.Printf("Request:\n%s %s\n", step.Request.Method, step.Request.URL)
				if step.Request.Body != "" {
					fmt.Printf("Body: %s\n", formatRequestBody(step.Request.Body, maskFields))
				}
				fmt.Println()
			}

			// Response (always shown in verbose)
			if step.Response != nil {
				fmt.Printf("Response:\nStatus: %d\n\nBody:\n%s\n", step.Response.Status, formatRequestBody(step.Response.Body, maskFields))
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
						fmt.Printf("  %s %s = %s\n", icon, a.Path, a.Expected)
					}

					if !a.Passed {
						if strings.HasPrefix(a.Actual, "(not found)") || a.Actual == "(unresolved)" {
							fmt.Printf("    └─ path not found\n")
							if len(a.Suggestions) > 0 {
								fmt.Println()
								fmt.Println("Available fields:")
								for _, f := range a.Suggestions {
									fmt.Printf("  - %s\n", f)
								}
							}
						} else {
							fmt.Printf("    └─ got: %s\n", a.Actual)
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
							fmt.Printf("   %s %s %s (actual: %s)\n", icon, a.Path, a.Expected, a.Actual)
						} else {
							fmt.Printf("   %s %s %s\n", icon, a.Path, a.Expected)
						}
					} else if isTiming {
						fmt.Printf("   %s %s = %s (actual: %s)\n", icon, a.Path, a.Expected, a.Actual)
					} else {
						fmt.Printf("   %s %s = %s\n", icon, a.Path, a.Expected)
					}

					// Inline failure info in summary
					if !a.Passed {
						if strings.HasPrefix(a.Actual, "(not found)") || a.Actual == "(unresolved)" {
							fmt.Printf("     └─ path not found")
							if len(a.Suggestions) > 0 {
								fmt.Printf(" (available: %s)", strings.Join(a.Suggestions, ", "))
							}
							fmt.Println()
						} else {
							fmt.Printf("     └─ got: %s\n", a.Actual)
						}
					}
				}
			}

			// Show response body only on failure in summary mode
			if !stepPassed && step.Response != nil {
				fmt.Printf("\nResponse:\nStatus: %d\n\nBody:\n%s\n", step.Response.Status, formatRequestBody(step.Response.Body, maskFields))
			}

			if step.Error != "" {
				fmt.Printf("   ✗ Error: %s\n", step.Error)
			}
		}

		if i < len(result.Steps)-1 {
			fmt.Println(strings.Repeat("─", 40))
		}
	}

	fmt.Println()
	fmt.Printf("%s %s\n", statusIcon, statusWord)

	// Summary line
	total := result.TotalPass + result.TotalFail
	summary := fmt.Sprintf("%d passed, %d failed, %d total", result.TotalPass, result.TotalFail, total)
	fmt.Printf("%s\n", summary)

	fmt.Printf("Duration: %s\n", formatDuration(result.Duration))
	fmt.Println()
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		secs := d.Seconds()
		return fmt.Sprintf("%.1fs", secs)
	}
	return d.Round(time.Second).String()
}
