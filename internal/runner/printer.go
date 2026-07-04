package runner

import (
	"encoding/json"
	"fmt"
	"sort"
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

	// Build scenario display name with account suffix
	scenarioName := result.Scenario
	if result.Account != "" {
		scenarioName = fmt.Sprintf("%s (%s)", result.Scenario, result.Account)
	}

	// Check if this is a dry-run result (has request but no response)
	isDryRun := false
	for _, step := range result.Steps {
		if step.Request != nil && step.Response == nil && step.Error == "" {
			isDryRun = true
			break
		}
	}

	fmt.Printf("\n%s %s", statusIcon, scenarioName)
	if isDryRun {
		fmt.Printf(" [DRY RUN]")
	}
	fmt.Println()

	// Print resolved variables in verbose mode
	if verbose && result.ResolvedVars != nil && len(result.ResolvedVars) > 0 {
		fmt.Println("── Resolved Variables ──")
		printVariables(result.ResolvedVars, maskFields)
		fmt.Println()
	} else {
		// Blank line between scenario name and first step (when no variables section)
		fmt.Println()
	}

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
		if step.Name != "" {
			stepLabel = step.Name
		} else if step.Request != nil {
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
			fmt.Printf("%s✓ success (%s)\n", statusIndent, FormatDuration(step.Duration))
		} else {
			fmt.Printf("%s✗ failed (%s)\n", statusIndent, FormatDuration(step.Duration))
		}

		// Display save warnings (do not affect pass/fail)
		if len(step.Warnings) > 0 {
			for _, w := range step.Warnings {
				fmt.Printf("%s  ⚠ %s\n", statusIndent, w)
			}
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
				fmt.Printf("%sRequest:\n%s%s %s\n", statusIndent, statusIndent, step.Request.Method, step.Request.URL)
				if len(step.Request.Headers) > 0 {
					fmt.Printf("%sHeaders:\n", statusIndent)
					for k, v := range step.Request.Headers {
						fmt.Printf("%s  %s: %s\n", statusIndent, k, v)
					}
				}
				if step.Request.MultipartSummary != "" {
					fmt.Printf("%sMultipart:\n%s\n", statusIndent, indentBlock(step.Request.MultipartSummary, statusIndent))
				}
				if step.Request.Body != "" {
					fmt.Printf("%sBody: %s\n", statusIndent, indentBlock(FormatRequestBody(step.Request.Body, maskFields), statusIndent))
				}
				fmt.Println()
			}

			// Response (always shown in verbose)
			if step.Response != nil {
				fmt.Printf("%sResponse:\n%sStatus: %d\n", statusIndent, statusIndent, step.Response.Status)
				if len(step.Response.Headers) > 0 {
					fmt.Printf("%sHeaders:\n", statusIndent)
					for k, v := range step.Response.Headers {
						fmt.Printf("%s  %s: %s\n", statusIndent, k, v)
					}
				}
				fmt.Printf("\n%sBody:\n%s\n", statusIndent, indentBlock(FormatRequestBody(step.Response.Body, maskFields), statusIndent))
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
					isTiming := a.Path == "timing.duration"

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
								// For schema assertions, show reason lines directly
								if strings.HasPrefix(a.Path, "schema") {
									for _, line := range strings.Split(a.Reason, "\n") {
										fmt.Printf("      └─ %s\n", line)
									}
								} else {
									fmt.Printf("      └─ actual: %s\n", a.Actual)
									fmt.Printf("      └─ expected: %s\n", a.Expected)
									for _, line := range strings.Split(a.Reason, "\n") {
										fmt.Printf("      └─ reason: %s\n", line)
									}
								}
							} else {
								fmt.Printf("      └─ got: %s\n", a.Actual)
							}
						}
					}
				}
			}

			// Show saved variables in verbose mode
			if step.SavedVars != nil && len(step.SavedVars) > 0 {
				names := make([]string, 0, len(step.SavedVars))
				for name := range step.SavedVars {
					names = append(names, name)
				}
				sort.Strings(names)
				fmt.Println()
				fmt.Println("Saved Variables:")
				for _, name := range names {
					val := step.SavedVars[name]
					if s, ok := val.(string); ok {
						if isSensitiveField(name, maskFields) {
							fmt.Printf("  %s = ***masked***\n", name)
						} else {
							fmt.Printf("  %s = %q\n", name, s)
						}
					} else {
						if isSensitiveField(name, maskFields) {
							fmt.Printf("  %s = ***masked***\n", name)
						} else {
							fmt.Printf("  %s = %v\n", name, val)
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
					isTiming := a.Path == "timing.duration"

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
								// For schema assertions, the actual value is already descriptive
								// No need to repeat it separately
								if strings.HasPrefix(a.Path, "schema") {
									for _, line := range strings.Split(a.Reason, "\n") {
										fmt.Printf("%s└─ %s\n", failureIndent, line)
									}
								} else {
									fmt.Printf("%s└─ actual: %s\n", failureIndent, a.Actual)
									fmt.Printf("%s└─ expected: %s\n", failureIndent, a.Expected)
									for _, line := range strings.Split(a.Reason, "\n") {
										fmt.Printf("%s└─ reason: %s\n", failureIndent, line)
									}
								}
							} else {
								fmt.Printf("%s└─ got: %s\n", failureIndent, a.Actual)
							}
						}
					}
				}
			}

			// Show saved variables in summary mode
			if step.SavedVars != nil && len(step.SavedVars) > 0 {
				names := make([]string, 0, len(step.SavedVars))
				for name := range step.SavedVars {
					names = append(names, name)
				}
				sort.Strings(names)
				var parts []string
				for _, name := range names {
					val := step.SavedVars[name]
					if isSensitiveField(name, maskFields) {
						parts = append(parts, fmt.Sprintf("%s → ***masked***", name))
					} else {
						parts = append(parts, fmt.Sprintf("%s → %v", name, formatSavedVarValue(val)))
					}
				}
				fmt.Printf("%s└─ saved: %s\n", statusIndent, strings.Join(parts, ", "))
			}

		// Show request and response on failure in summary mode
		if !stepPassed {
			fmt.Println()
			if step.Request != nil {
				if len(step.Request.Headers) > 0 {
					fmt.Printf("%sRequest Headers:\n", statusIndent)
					for k, v := range step.Request.Headers {
						if isSensitiveField(k, maskFields) {
							fmt.Printf("%s  %s: ***masked***\n", statusIndent, k)
						} else {
							fmt.Printf("%s  %s: %s\n", statusIndent, k, v)
						}
					}
				}
				if step.Request.Body != "" {
					fmt.Printf("%sRequest Body:\n%s\n\n", statusIndent, indentBlock(FormatRequestBody(step.Request.Body, maskFields), statusIndent))
				}
			}
			if step.Response != nil {
				fmt.Printf("%sResponse:\n%sStatus: %d\n\n%sBody:\n%s\n", statusIndent, statusIndent, step.Response.Status, statusIndent, indentBlock(FormatRequestBody(step.Response.Body, maskFields), statusIndent))
			}
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
	fmt.Printf("%s %s (%s)\n", statusIcon, statusWord, scenarioName)

	// Summary line
	total := result.TotalPass + result.TotalFail
	summary := fmt.Sprintf("%d passed, %d failed, %d total", result.TotalPass, result.TotalFail, total)
	fmt.Printf("%s\n", summary)

	fmt.Printf("Duration: %s\n", FormatDuration(result.Duration))
	fmt.Println()
}

// printVariables outputs the variable map in a formatted way for verbose output.
func printVariables(vars map[string]interface{}, maskFields []string) {
	// Sort keys for deterministic output
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		val := vars[key]

		// Handle special nested types like $accounts
		if nestedMap, ok := val.(map[string]interface{}); ok {
			// Print as $accounts.name.field format
			nestedKeys := make([]string, 0, len(nestedMap))
			for k := range nestedMap {
				nestedKeys = append(nestedKeys, k)
			}
			sort.Strings(nestedKeys)

			for _, subKey := range nestedKeys {
				subVal := nestedMap[subKey]
				if subValMap, ok := subVal.(map[string]interface{}); ok {
					// This is an account entry - show fields
					fieldKeys := make([]string, 0, len(subValMap))
					for k := range subValMap {
						fieldKeys = append(fieldKeys, k)
					}
					sort.Strings(fieldKeys)

					for _, fieldKey := range fieldKeys {
						fieldVal := subValMap[fieldKey]
						displayKey := fmt.Sprintf("$accounts.%s.%s", subKey, fieldKey)

						if isSensitiveField(fieldKey, maskFields) {
							fmt.Printf("  %-30s→ ***masked***\n", displayKey)
						} else {
							fmt.Printf("  %-30s→ %v\n", displayKey, fieldVal)
						}
					}
				}
			}
		} else {
			// Simple variable
			displayKey := "$" + key

			// Check if the value itself should be masked
			if isSensitiveField(key, maskFields) {
				fmt.Printf("  %-30s→ ***masked***\n", displayKey)
			} else {
				fmt.Printf("  %-30s→ %v\n", displayKey, val)
			}
		}
	}
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

// indentBlock prepends a prefix to each non-empty line of a multi-line string.
func indentBlock(text, prefix string) string {
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

// PrintStepResult formats and prints a single step execution result to stdout in a highly compact form.
func PrintStepResult(result *RunResult, verbose bool, maskFields []string) {
	if maskFields == nil {
		maskFields = defaultSensitiveFields
	}

	stepCounter := 1
	for i, step := range result.Steps {
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
			if step.Depth > 0 {
				fmt.Printf("%s\n", indentPrefix(indent))
			} else {
				fmt.Printf("\n")
			}
			continue
		}

		stepPassed := step.Error == ""
		for _, a := range step.Assertions {
			if !a.Passed {
				stepPassed = false
				break
			}
		}

		var stepLabel string
		if step.Name != "" {
			stepLabel = step.Name
		} else if step.Request != nil {
			stepLabel = fmt.Sprintf("%s %s", step.Request.Method, step.Request.URL)
		} else {
			if step.Original.Request.URL != "" {
				stepLabel = fmt.Sprintf("%s %s (failed before execution)", step.Original.Request.Method, step.Original.Request.URL)
			} else {
				stepLabel = "Unknown Step"
			}
		}

		prefix := "▼ "
		if step.Depth > 0 {
			prefix = indentPrefix(indent) + "   ├ ▼ "
		}

		fmt.Printf("%s%s (%s)\n", prefix, stepLabel, FormatDuration(step.Duration))

		// Print assertions
		for _, a := range step.Assertions {
			icon := "  ✓"
			if !a.Passed {
				icon = "  ✗"
			}

			isTiming := a.Path == "timing.duration"
			assertionText := ""
			if a.Expected == "exists" {
				if isTiming {
					assertionText = fmt.Sprintf("%s %s %s (actual: %s)", icon, a.Path, a.Expected, a.Actual)
				} else {
					assertionText = fmt.Sprintf("%s %s %s", icon, a.Path, a.Expected)
				}
			} else if isTiming {
				assertionText = fmt.Sprintf("%s %s = %s (actual: %s)", icon, a.Path, a.Expected, a.Actual)
			} else {
				if strings.HasPrefix(a.Expected, "contains ") || strings.HasPrefix(a.Expected, "startsWith ") || strings.HasPrefix(a.Expected, "endsWith ") || strings.HasPrefix(a.Expected, "pattern ") {
					assertionText = fmt.Sprintf("%s %s %s (actual: %s)", icon, a.Path, a.Expected, a.Actual)
				} else if strings.HasPrefix(a.Expected, "exactly") || strings.HasPrefix(a.Expected, "all elements") {
					assertionText = fmt.Sprintf("%s %s = %s (actual: %s)", icon, a.Path, a.Expected, a.Actual)
				} else if isMatcherKeyword(a.Expected) || a.Reason != "" {
					assertionText = fmt.Sprintf("%s %s = %s (actual: %s)", icon, a.Path, a.Expected, a.Actual)
				} else {
					assertionText = fmt.Sprintf("%s %s = %s", icon, a.Path, a.Expected)
				}
			}

			fmt.Printf("%s%s\n", statusIndent, assertionText)

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
						if strings.HasPrefix(a.Path, "schema") {
							for _, line := range strings.Split(a.Reason, "\n") {
								fmt.Printf("%s└─ %s\n", failureIndent, line)
							}
						} else {
							fmt.Printf("%s└─ actual: %s\n", failureIndent, a.Actual)
							fmt.Printf("%s└─ expected: %s\n", failureIndent, a.Expected)
							for _, line := range strings.Split(a.Reason, "\n") {
								fmt.Printf("%s└─ reason: %s\n", failureIndent, line)
							}
						}
					} else {
						fmt.Printf("%s└─ got: %s\n", failureIndent, a.Actual)
					}
				}
			}
		}

		// Show saved variables in PrintStepResult
		if step.SavedVars != nil && len(step.SavedVars) > 0 {
			names := make([]string, 0, len(step.SavedVars))
			for name := range step.SavedVars {
				names = append(names, name)
			}
			sort.Strings(names)
			var parts []string
			for _, name := range names {
				val := step.SavedVars[name]
				if isSensitiveField(name, maskFields) {
					parts = append(parts, fmt.Sprintf("%s → ***masked***", name))
				} else {
					parts = append(parts, fmt.Sprintf("%s → %v", name, formatSavedVarValue(val)))
				}
			}
			fmt.Printf("%s└─ saved: %s\n", statusIndent, strings.Join(parts, ", "))
		}

		if !stepPassed && step.Response != nil {
			fmt.Println()
			if step.Request != nil {
				if step.Request.Body != "" {
					fmt.Printf("%sRequest Body:\n%s\n\n", statusIndent, indentBlock(FormatRequestBody(step.Request.Body, maskFields), statusIndent))
				}
			}
			fmt.Printf("%sResponse:\n%sStatus: %d\n\n%sBody:\n%s\n", statusIndent, statusIndent, step.Response.Status, statusIndent, indentBlock(FormatRequestBody(step.Response.Body, maskFields), statusIndent))
		}

		if step.Error != "" {
			fmt.Printf("%s   ✗ Error: %s\n", indent, step.Error)
		}

		if verbose {
			fmt.Println()
			if step.Request != nil {
				fmt.Printf("%sRequest:\n%s%s %s\n", statusIndent, statusIndent, step.Request.Method, step.Request.URL)
				if step.Request.Body != "" {
					fmt.Printf("%sBody: %s\n", statusIndent, indentBlock(FormatRequestBody(step.Request.Body, maskFields), statusIndent))
				}
				fmt.Println()
			}
			if step.Response != nil {
				fmt.Printf("%sResponse:\n%sStatus: %d\n\n%sBody:\n%s\n", statusIndent, statusIndent, step.Response.Status, statusIndent, indentBlock(FormatRequestBody(step.Response.Body, maskFields), statusIndent))
				fmt.Println()
			}
		}

		if i < len(result.Steps)-1 {
			nextStep := result.Steps[i+1]
			if nextStep.Depth == 0 && !nextStep.IsUseStart && !nextStep.IsUseEnd {
				fmt.Println(strings.Repeat("─", 40))
			}
		}
	}
}

// formatSavedVarValue formats a saved variable value for display.
func formatSavedVarValue(val interface{}) string {
	switch v := val.(type) {
	case string:
		if len(v) > 40 {
			return fmt.Sprintf("%q...", v[:40])
		}
		return fmt.Sprintf("%q", v)
	case nil:
		return "null"
	case json.Number:
		return v.String()
	case float64:
		if v == float64(int(v)) {
			return fmt.Sprintf("%d", int(v))
		}
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
