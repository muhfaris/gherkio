package runner

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/muhfaris/gherkio/internal/model"
	"gopkg.in/yaml.v3"
)

// RunConfig holds configuration for a single run.
type RunConfig struct {
	TestPath       string                 // Path to the test YAML file
	EnvName        string                 // Environment name (e.g. "local", "staging")
	ProjectDir     string                 // Root project directory (where .gherkio lives)
	Verbose        bool                   // Show full request/response payloads
	MaskFields     []string               // Sensitive field names to mask in output (nil = use defaults)
	AccountName    string                 // Account name to use from credentials (optional)
	CredentialVars map[string]interface{} // Pre-injected credential variables (optional)
	AllAccounts    map[string]interface{} // All accounts for $accounts.<name>.<field> access (optional)
	StepIndex      int                    // Index of step to run (0-indexed). Negative means run all steps.
	StepSection    string                 // Section of the step ("setup", "steps", "teardown")
	DryRun         bool                   // Preview without executing HTTP requests
	Snapshot      SnapshotConfig          // Configuration for failure snapshots
	Until          string                 // Execute steps until a specific target, e.g. "steps:1" or "2"
	FailFast       bool                   // Stop executing remaining steps when a step fails
	SessionVars map[string]interface{} // Session-persistent variables across step runs (mutated in place, may be nil)
	SessionFile  string                 // Path to session file for CLI persistence (empty = no file persistence)
}

// RunResult holds the overall execution result.
type RunResult struct {
	Scenario     string                 `json:"scenario"`
	Description  string                 `json:"description,omitempty"`
	TestFile     string                 `json:"testFile,omitempty"`
	Account      string                 `json:"account,omitempty"` // Account name used (if any)
	Steps        []StepResult           `json:"steps"`
	ResolvedVars map[string]interface{} `json:"resolvedVars,omitempty"` // Variables available at start of scenario
	TotalPass    int                    `json:"totalPass"`
	TotalFail    int                    `json:"totalFail"`
	Duration     time.Duration          `json:"duration"`
	Passed       bool                   `json:"passed"`
}

// Run executes a test file and returns the result.
func Run(cfg RunConfig) (*RunResult, error) {
	start := time.Now()

	// 1. Load environment
	env, err := loadEnvironment(cfg.ProjectDir, cfg.EnvName)
	if err != nil {
		return nil, fmt.Errorf("failed to load environment: %w", err)
	}

	// 2. Load test file
	testFile, err := LoadTestFile(cfg.TestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load test file: %w", err)
	}

	// Apply dynamic execution slicing (until parameter)
	if cfg.Until != "" {
		untilSection, untilIndex, err := parseUntil(cfg.Until, testFile)
		if err != nil {
			return nil, err
		}
		if untilSection != "" {
			switch untilSection {
			case "setup":
				testFile.Setup = testFile.Setup[:untilIndex+1]
				testFile.Steps = nil
				testFile.Teardown = nil
			case "steps":
				testFile.Steps = testFile.Steps[:untilIndex+1]
				testFile.Teardown = nil
			case "teardown":
				testFile.Teardown = testFile.Teardown[:untilIndex+1]
			}
		}
	}

	// Route to single-step or section-only execution
	if cfg.StepIndex >= 0 || cfg.StepSection != "" {
		return RunSingleStep(cfg, env, testFile)
	}

	// 3. Execute steps
	result := &RunResult{
		Scenario:    testFile.Scenario,
		Description: testFile.Description,
		Account:     cfg.AccountName,
	}

	vars := make(map[string]interface{})

	// Inject OS env vars starting with GHERKIO_
	for key, val := range LoadGherkioEnvVars() {
		vars[key] = val
	}

	// Inject credential variables (can be overridden by save:)
	if cfg.CredentialVars != nil {
		for key, val := range cfg.CredentialVars {
			vars[key] = val
		}
	}

	// Inject all accounts as $accounts.<name>.<field> for dotted-path access
	if cfg.AllAccounts != nil {
		vars["accounts"] = cfg.AllAccounts
	}

	// Merge session vars from previous step runs (overrides credentials, overridden by builtins)
	if cfg.SessionVars != nil {
		for key, val := range cfg.SessionVars {
			vars[key] = val
		}
	}

	// Load session vars from file (CLI file-based persistence across processes)
	if cfg.SessionFile != "" {
		loaded, err := loadSessionVars(cfg.SessionFile)
		if err == nil && loaded != nil {
			for key, val := range loaded {
				vars[key] = val
			}
		}
	}

	// Capture initial variable state for verbose output
	result.ResolvedVars = snapshotVars(vars, cfg.MaskFields)

	// Load project configuration for sandbox policy checks
	appCfg, _ := LoadConfig(cfg.ProjectDir)
	var sandbox *model.SandboxConfig
	if appCfg != nil {
		if appCfg.Security.Sandboxing.Enabled {
			sandbox = &appCfg.Security.Sandboxing
		} else if appCfg.Security.Sandbox.Enabled {
			sandbox = &appCfg.Security.Sandbox
		}
	}

	currentDir := filepath.Dir(cfg.TestPath)
	var allSteps []StepResult
	setupFailed := false

	// Execute setup steps first
	if len(testFile.Setup) > 0 {
		setupSteps, setupPass, setupFail, setupPassed := executeSteps(testFile.Setup, env, vars, cfg.ProjectDir, currentDir, 0, "setup", cfg.DryRun, cfg.FailFast, sandbox, cfg.Snapshot, testFile.Scenario, cfg.TestPath)
		for i := range setupSteps {
			setupSteps[i].ScenarioName = testFile.Scenario
			setupSteps[i].TestFile = cfg.TestPath
		}
		allSteps = append(allSteps, setupSteps...)
		result.TotalPass += setupPass
		result.TotalFail += setupFail
		if !setupPassed {
			setupFailed = true
		}
	}

	// Execute main steps (skip if setup failed)
	if !setupFailed {
		mainSteps, mainPass, mainFail, mainPassed := executeSteps(testFile.Steps, env, vars, cfg.ProjectDir, currentDir, 0, "steps", cfg.DryRun, cfg.FailFast, sandbox, cfg.Snapshot, testFile.Scenario, cfg.TestPath)
		for i := range mainSteps {
			mainSteps[i].ScenarioName = testFile.Scenario
			mainSteps[i].TestFile = cfg.TestPath
		}
		allSteps = append(allSteps, mainSteps...)
		result.TotalPass += mainPass
		result.TotalFail += mainFail
		if !mainPassed {
			result.Passed = false
		}
	}

	// Execute teardown steps (always, even if setup or steps failed)
	// Teardown failures are recorded but don't affect overall pass/fail
	if len(testFile.Teardown) > 0 {
		teardownSteps, _, _, _ := executeSteps(testFile.Teardown, env, vars, cfg.ProjectDir, currentDir, 0, "teardown", cfg.DryRun, cfg.FailFast, sandbox, cfg.Snapshot, testFile.Scenario, cfg.TestPath)
		for i := range teardownSteps {
			teardownSteps[i].ScenarioName = testFile.Scenario
			teardownSteps[i].TestFile = cfg.TestPath
		}
		allSteps = append(allSteps, teardownSteps...)
	}

	result.Steps = allSteps
	result.TestFile = cfg.TestPath
	result.Duration = time.Since(start)

	// Determine overall pass/fail (teardown failures don't affect this)
	// If we haven't set Passed to false yet, check main steps pass/fail
	if result.TotalFail == 0 && setupFailed == false {
		result.Passed = true
	}

	// Write back session vars for subsequent step runs
	if cfg.SessionVars != nil {
		for key, val := range vars {
			cfg.SessionVars[key] = val
		}
	}

	// Persist session to file for CLI cross-process usage
	if cfg.SessionFile != "" {
		_ = saveSessionVars(cfg.SessionFile, vars)
	}

	return result, nil
}

// executeSteps executes a list of steps. If dryRun is true, skips HTTP calls and produces preview output.
func executeSteps(steps []model.Step, env *model.Environment, vars map[string]interface{}, projectDir string, currentDir string, depth int, role string, dryRun bool, failFast bool, sandbox *model.SandboxConfig, snapCfg SnapshotConfig, scenario string, testFile string) ([]StepResult, int, int, bool) {
	var stepResults []StepResult
	totalPass := 0
	totalFail := 0
	allPassed := true
	stepIndex := 0

	for _, step := range steps {
		stepStart := time.Now()
		stepResult := StepResult{
			Name:     step.Name,
			Original: step,
			Depth:    depth,
			Role:     role,
		}

		// Inject fresh built-in generator variables per step so each request
		// gets unique $uuid, $ulid, $randomInt, $randomEmail, $randomPhone values.
		for key, val := range BuiltinVars() {
			vars[key] = val
		}

		// Handle 'use' step recursively
		if step.Use != "" {
			if step.Retry != nil {
				stepResult.Error = "validation error: retry block is not allowed on 'use:' steps"
				stepResults = append(stepResults, stepResult)
				allPassed = false
				continue
			}
			useStartStep := StepResult{
				Original:   step,
				Depth:      depth,
				IsUseStart: true,
				UseFile:    step.Use,
			}
			stepResults = append(stepResults, useStartStep)
			if depth > 10 {
				stepResult.Error = fmt.Sprintf("circular reference or max depth exceeded for use: %s", step.Use)
				stepResults = append(stepResults, stepResult)
				allPassed = false
				continue
			}

			resolvedPath, err := resolveUsePath(currentDir, projectDir, step.Use)
			if err != nil {
				stepResult.Error = fmt.Sprintf("failed to resolve use step '%s': %v", step.Use, err)
				stepResults = append(stepResults, stepResult)
				allPassed = false
				continue
			}

			usedTest, err := LoadTestFile(resolvedPath)
			if err != nil {
				stepResult.Error = fmt.Sprintf("failed to load used test file '%s': %v", resolvedPath, err)
				stepResults = append(stepResults, stepResult)
				allPassed = false
				continue
			}

			usedCurrentDir := filepath.Dir(resolvedPath)

			// Apply 'with' variable overrides before executing the used scenario
			// Interpolate each value against current vars, inject into vars,
			// and restore after the nested execution.
			var restoredVars []struct {
				name  string
				value interface{}
				exist bool
			}
			if step.With != nil {
				for name, rawVal := range step.With {
					interpolated, err := interpolateString(rawVal, vars)
					if err != nil {
						continue // skip overrides that fail to interpolate
					}
					oldVal, exists := vars[name]
					restoredVars = append(restoredVars, struct {
						name  string
						value interface{}
						exist bool
					}{name: name, value: oldVal, exist: exists})
					vars[name] = interpolated
				}
			}

			nestedSteps, nestedPass, nestedFail, _ := executeSteps(usedTest.Steps, env, vars, projectDir, usedCurrentDir, depth+1, role, dryRun, failFast, sandbox, snapCfg, scenario, testFile)

			// Restore previous variable values after the used scenario completes
			if step.With != nil {
				for _, rv := range restoredVars {
					if rv.exist {
						vars[rv.name] = rv.value
					} else {
						delete(vars, rv.name)
					}
				}
			}

			// Flatten the results
			// Flatten the results
			stepResults = append(stepResults, nestedSteps...)
			// add a dummy end step for 'use'
			useEndStep := StepResult{
				Original: step,
				Depth:    depth,
				IsUseEnd: true,
				UseFile:  step.Use,
				Role:     role,
			}
			stepResults = append(stepResults, useEndStep)
			totalPass += nestedPass
			totalFail += nestedFail
			if nestedFail > 0 {
				allPassed = false
				if failFast {
					break
				}
			}
			continue
		}

		// Resolve URL using service or global baseUrl
		// Interpolate variables in the request once before the loop
		interpolatedRequest, err := InterpolateRequest(step.Request, vars)
		if err != nil {
		stepResult.Error = fmt.Sprintf("Variable interpolation failed: %v", err)
			stepResults = append(stepResults, stepResult)
			allPassed = false
			if failFast {
				break
			}
			continue
		}

		url := resolveURL(env, interpolatedRequest)

		stepResult.Request = &RequestInfo{
			Method:  interpolatedRequest.Method,
			URL:     url,
			Query:   interpolatedRequest.Query,
			Headers: interpolatedRequest.Headers,
		}
		if interpolatedRequest.Body != nil {
			if bodyJSON, err := json.Marshal(interpolatedRequest.Body); err == nil {
				stepResult.Request.Body = string(bodyJSON)
			} else {
				stepResult.Request.Body = fmt.Sprintf("%v", interpolatedRequest.Body)
			}
		}

		// Run domain sandboxing validation
		if err := ValidateURL(url, sandbox); err != nil {
			stepResult.Error = fmt.Sprintf("security policy check failed: %v", err)
			stepResult.Duration = time.Since(stepStart)
			stepResults = append(stepResults, stepResult)
			allPassed = false
			totalFail++
			if failFast {
				break
			}
			continue
		}

		// Dry-run mode: skip HTTP execution and mark step as preview
		if dryRun {
			stepResult.Duration = time.Since(stepStart)
			stepResults = append(stepResults, stepResult)
			continue
		}

		var (
			resp         *ResponseInfo
			jwtClaims    map[string]interface{}
			assertions   []AssertionResult
			stepErr      error
			retryHistory []RetryEntry
			attempts     = 1
			intervalMs   = 500
			backoffStrat = "constant"
			hasRetry     = step.Retry != nil
		)

		if hasRetry {
			if step.Retry.Attempts > 0 {
				attempts = step.Retry.Attempts
			}
			if step.Retry.Interval > 0 {
				intervalMs = step.Retry.Interval
			}
			if step.Retry.Backoff != "" {
				backoffStrat = step.Retry.Backoff
			}
		}

		var maxDuration time.Duration
		if hasRetry && step.Retry.MaxDuration != "" {
			var dErr error
			maxDuration, dErr = time.ParseDuration(step.Retry.MaxDuration)
			if dErr != nil {
				stepResult.Error = fmt.Sprintf("invalid maxDuration: %v", dErr)
				stepResults = append(stepResults, stepResult)
				allPassed = false
				continue
			}
		}

		isIdempotent := func(m string) bool {
			return m == "GET" || m == "HEAD" || m == "OPTIONS" || m == "TRACE"
		}
		if hasRetry && !isIdempotent(interpolatedRequest.Method) {
			fmt.Fprintf(os.Stderr, "⚠ %s %s — retrying non-idempotent request\n", interpolatedRequest.Method, url)
		}

		var lastResp *ResponseInfo
		var lastJwtClaims map[string]interface{}
		var lastAssertions []AssertionResult

		for i := 1; i <= attempts; i++ {
			// Re-inject fresh built-in generator variables per retry attempt so
			// ${randomInt}, ${uuid}, and other dynamic values change on each attempt.
			// This also causes array-index references like $issueTags[${randomInt(0,4)}].id
			// to resolve to a different element on each retry.
			for key, val := range BuiltinVars() {
				vars[key] = val
			}

			// Re-interpolate request with fresh variables so each retry attempt gets
			// a potentially different body (e.g. different random index for issue tag).
			freshReq, reqErr := InterpolateRequest(step.Request, vars)
			if reqErr == nil {
				interpolatedRequest = freshReq
				url = resolveURL(env, interpolatedRequest)

				// Update stepResult.Request to reflect the current attempt's values
				stepResult.Request = &RequestInfo{
					Method:  interpolatedRequest.Method,
					URL:     url,
					Query:   interpolatedRequest.Query,
					Headers: interpolatedRequest.Headers,
				}
				if interpolatedRequest.Body != nil {
					if bodyJSON, err := json.Marshal(interpolatedRequest.Body); err == nil {
						stepResult.Request.Body = string(bodyJSON)
					} else {
						stepResult.Request.Body = fmt.Sprintf("%v", interpolatedRequest.Body)
					}
				}
			}

			if maxDuration > 0 && time.Since(stepStart) >= maxDuration {
				stepErr = fmt.Errorf("maxDuration %s exceeded", step.Retry.MaxDuration)
				break
			}

			attemptLabel := fmt.Sprintf("%d/%d", i, attempts)
			fmt.Fprintf(os.Stderr, "→ [%s] %s %s...\n", attemptLabel, interpolatedRequest.Method, url)

			attemptStart := time.Now()
			resp, err = executeRequest(interpolatedRequest.Method, url, interpolatedRequest.Headers, interpolatedRequest.Body, interpolatedRequest.Multipart, interpolatedRequest.Timeout, projectDir)

			entry := RetryEntry{
				Attempt:  i,
				Duration: time.Since(attemptStart),
			}

			attemptDuration := time.Since(attemptStart)
			if err != nil {
				entry.Error = err.Error()
				retryHistory = append(retryHistory, entry)
				if i == attempts {
					fmt.Fprintf(os.Stderr, "✗ [%s] %s %s → error: %v\n", attemptLabel, interpolatedRequest.Method, url, err)
					stepErr = err
					break
				}
				fmt.Fprintf(os.Stderr, "✗ [%s] %s %s → error: %v (will retry)\n", attemptLabel, interpolatedRequest.Method, url, err)
				time.Sleep(calculateBackoff(backoffStrat, intervalMs, i))
				continue
			}

			entry.Status = resp.Status
			fmt.Fprintf(os.Stderr, "← [%s] %s %s → %d (%s)\n", attemptLabel, interpolatedRequest.Method, url, resp.Status, attemptDuration.Round(time.Millisecond))
			if len(resp.Body) > 500 {
				entry.Body = resp.Body[:500] + "..."
			} else {
				entry.Body = resp.Body
			}
			retryHistory = append(retryHistory, entry)

			jwtClaims = nil
			if resp.Parsed != nil {
				// Build token search paths: custom config path first, then defaults
				var jwtTokenPaths []string
				appCfg, _ := LoadConfig(projectDir)
				if appCfg != nil && appCfg.JWTTokenPath != "" {
					jwtTokenPaths = append(jwtTokenPaths, appCfg.JWTTokenPath)
				}
				jwtTokenPaths = append(jwtTokenPaths,
					"data.token", "data.accessToken", "data.access_token",
					"body.data.token", "body.data.accessToken", "body.data.access_token",
					"token", "accessToken", "access_token")
				for _, field := range jwtTokenPaths {
					if tokenVal, found := resolvePath(resp.Parsed, field); found {
						if tokenStr, ok := tokenVal.(string); ok {
							claims, derr := decodeJWT(tokenStr)
							if derr == nil {
								jwtClaims = claims
								break
							}
						}
					}
				}
			}

			// Interpolate variable references in expected assertion values
			interpolatedExtra := make(map[string]interface{}, len(step.Expect.Extra))
			for path, val := range step.Expect.Extra {
				if strVal, ok := val.(string); ok {
					interpolated, err := interpolateString(strVal, vars)
					if err == nil {
						interpolatedExtra[path] = interpolated
					} else {
						// Variable not found — use original, assertion will fail clearly
						interpolatedExtra[path] = strVal
					}
				} else {
					interpolatedExtra[path] = val
				}
			}

			assertions = runAssertions(resp.Status, resp, jwtClaims, step.Expect.Status, interpolatedExtra, projectDir, interpolatedRequest.Body)

			allPass := true
			for _, a := range assertions {
				if !a.Passed {
					allPass = false
					break
				}
			}

			lastResp = resp
			lastJwtClaims = jwtClaims
			lastAssertions = assertions

			if allPass {
				break
			}

			if hasRetry && len(step.Retry.OnStatus) > 0 {
				statusMatch := false
				for _, st := range step.Retry.OnStatus {
					if resp.Status == st {
						statusMatch = true
						break
					}
				}
				if !statusMatch {
					// Condition failed but status is not in onStatus list, so don't retry
					break
				}
			}

			if i == attempts {
				break
			}

			if maxDuration > 0 && time.Since(stepStart) >= maxDuration {
				stepErr = fmt.Errorf("maxDuration %s exceeded", step.Retry.MaxDuration)
				break
			}

			time.Sleep(calculateBackoff(backoffStrat, intervalMs, i))
		}

		if stepErr != nil {
			stepResult.Error = stepErr.Error()
			if hasRetry && len(retryHistory) > 0 {
				stepResult.RetryCount = len(retryHistory) - 1
				stepResult.RetryHistory = retryHistory
				// if we have a last response, use it for context
				if lastResp != nil {
					stepResult.Response = lastResp
					stepResult.Assertions = lastAssertions
					for _, a := range lastAssertions {
						if a.Passed {
							totalPass++
						} else {
							totalFail++
						}
					}
				}
			} else {
				allPassed = false
			}
			stepResult.Duration = time.Since(stepStart)
			if stepResult.Response == nil {
				allPassed = false
			}

			// Write failure snapshot if enabled and this step has failed
			if snapCfg.Enabled && !allPassed {
				writeFailureSnapshot(snapCfg, &stepResult, stepIndex, scenario, testFile, role, vars)
			}

			stepResults = append(stepResults, stepResult)
			stepIndex++
			if failFast {
				break
			}
			continue
		}

		stepResult.Response = lastResp
		stepResult.Assertions = lastAssertions

		// Extract variables
		if lastResp != nil {
			warnings := extractValues(vars, step.Save, lastResp, lastJwtClaims, interpolatedRequest.Body)
			if len(warnings) > 0 {
				stepResult.Warnings = warnings
			}
		}

		// Track saved variable values for display
		if step.Save != nil {
			stepResult.SavedVars = make(map[string]interface{})
			for name := range step.Save {
				if val, ok := vars[name]; ok {
					stepResult.SavedVars[name] = val
				}
			}
		}

		stepResult.Duration = time.Since(stepStart)
		if hasRetry && len(retryHistory) > 1 {
			stepResult.RetryCount = len(retryHistory) - 1
			stepResult.RetryHistory = retryHistory
		}

		// Count pass/fail
		stepHasFailedAssertions := false
		for _, a := range stepResult.Assertions {
			if a.Passed {
				totalPass++
			} else {
				totalFail++
				stepHasFailedAssertions = true
			}
		}

		// Timing assertion (if configured)
		if step.Timing.Max != "" {
			timingResult := evaluateTiming(stepResult.Duration, step.Timing.Max)
			stepResult.Assertions = append(stepResult.Assertions, timingResult)
			if timingResult.Passed {
				totalPass++
			} else {
				totalFail++
				stepHasFailedAssertions = true
			}
		}

		// Write failure snapshot if step has failed assertions and snapshots are enabled
		if snapCfg.Enabled && stepHasFailedAssertions {
			writeFailureSnapshot(snapCfg, &stepResult, stepIndex, scenario, testFile, role, vars)
		}

		stepResults = append(stepResults, stepResult)
		stepIndex++

		if failFast && stepHasFailedAssertions {
			break
		}
	}

	allPassed = totalFail == 0
	return stepResults, totalPass, totalFail, allPassed
}

// LoadTestFile reads and parses a test YAML file.
func LoadTestFile(path string) (*model.TestFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var test model.TestFile
	if err := yaml.Unmarshal(data, &test); err != nil {
		return nil, fmt.Errorf("failed to parse test file: %w", err)
	}

	return &test, nil
}

// loadEnvironment reads the environment YAML file.
func loadEnvironment(projectDir, envName string) (*model.Environment, error) {
	envPath := filepath.Join(projectDir, ".gherkio", "environments", envName+".yaml")
	data, err := os.ReadFile(envPath)
	if err != nil {
		return nil, err
	}

	var env model.Environment
	if err := yaml.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("failed to parse environment file: %w", err)
	}

	return &env, nil
}

// resolveURL builds the full URL using environment baseUrl or service-specific baseUrl.
// If req.Query is non-empty, its key-value pairs are appended as URL query parameters.
func resolveURL(env *model.Environment, req model.Request) string {
	baseURL := env.BaseURL

	if req.Service != "" && env.Services != nil {
		if svc, ok := env.Services[req.Service]; ok {
			baseURL = svc.BaseURL
		}
	}

	finalURL := baseURL + req.URL

	if len(req.Query) > 0 {
		sep := "?"
		if strings.Contains(finalURL, "?") {
			sep = "&"
		}
		first := true
		for k, v := range req.Query {
			escapedKey := url.QueryEscape(k)
			escapedVal := url.QueryEscape(v)
			if first {
				finalURL += sep + escapedKey + "=" + escapedVal
				first = false
			} else {
				finalURL += "&" + escapedKey + "=" + escapedVal
			}
		}
	}

	return finalURL
}

// resolveUsePath tries to find the file referenced in a 'use' step.
// It checks relative to the current file's directory first, then relative to projectDir/.gherkio/tests/
func resolveUsePath(currentFileDir, projectDir, usePath string) (string, error) {
	// Ensure the usePath has a .yaml extension
	if !filepath.IsAbs(usePath) && filepath.Ext(usePath) == "" {
		usePath += ".yaml"
	}

	// 1. Try relative to the current file's directory
	fullPath := filepath.Join(currentFileDir, usePath)
	if _, err := os.Stat(fullPath); err == nil {
		return fullPath, nil
	}

	// 2. Try relative to .gherkio/tests/
	fullPath = filepath.Join(projectDir, ".gherkio", "tests", usePath)
	if _, err := os.Stat(fullPath); err == nil {
		return fullPath, nil

	}

	return "", fmt.Errorf("file not found (checked relative to '%s' and '.gherkio/tests/')", currentFileDir)
}

// RunSingleStep executes a single step in isolation.
func RunSingleStep(cfg RunConfig, env *model.Environment, testFile *model.TestFile) (*RunResult, error) {
	start := time.Now()

	section := strings.ToLower(cfg.StepSection)
	if section == "" {
		section = "steps"
	}

	var stepList []model.Step
	switch section {
	case "setup":
		stepList = testFile.Setup
	case "steps":
		stepList = testFile.Steps
	case "teardown":
		stepList = testFile.Teardown
	default:
		return nil, fmt.Errorf("invalid step section: %s", cfg.StepSection)
	}

	// Determine which steps to run
	var stepsToRun []model.Step
	if cfg.StepIndex >= 0 {
		// Single step mode
		if cfg.StepIndex >= len(stepList) {
			return nil, fmt.Errorf("step index %d out of bounds for section %q (contains %d steps)", cfg.StepIndex, section, len(stepList))
		}
		stepsToRun = []model.Step{stepList[cfg.StepIndex]}
	} else {
		// Section-only mode: run ALL steps in the section
		stepsToRun = stepList
	}

	vars := make(map[string]interface{})

	// Inject OS env vars starting with GHERKIO_
	for key, val := range LoadGherkioEnvVars() {
		vars[key] = val
	}

	if cfg.CredentialVars != nil {
		for key, val := range cfg.CredentialVars {
			vars[key] = val
		}
	}

	// Inject all accounts as $accounts.<name>.<field> for dotted-path access
	if cfg.AllAccounts != nil {
		vars["accounts"] = cfg.AllAccounts
	}

	// Merge session vars from previous step runs
	if cfg.SessionVars != nil {
		for key, val := range cfg.SessionVars {
			vars[key] = val
		}
	}

	// Load session vars from file (CLI file-based persistence across processes)
	if cfg.SessionFile != "" {
		loaded, err := loadSessionVars(cfg.SessionFile)
		if err == nil && loaded != nil {
			for key, val := range loaded {
				vars[key] = val
			}
		}
	}

	// Capture initial variable state for verbose output
	initialVars := snapshotVars(vars, cfg.MaskFields)

	// Load project configuration for sandbox policy checks
	appCfg, _ := LoadConfig(cfg.ProjectDir)
	var sandbox *model.SandboxConfig
	if appCfg != nil {
		if appCfg.Security.Sandboxing.Enabled {
			sandbox = &appCfg.Security.Sandboxing
		} else if appCfg.Security.Sandbox.Enabled {
			sandbox = &appCfg.Security.Sandbox
		}
	}

	currentDir := filepath.Dir(cfg.TestPath)
	runSteps, pass, fail, passed := executeSteps(stepsToRun, env, vars, cfg.ProjectDir, currentDir, 0, section, cfg.DryRun, cfg.FailFast, sandbox, cfg.Snapshot, testFile.Scenario, cfg.TestPath)

	for i := range runSteps {
		runSteps[i].ScenarioName = testFile.Scenario
		runSteps[i].TestFile = cfg.TestPath
	}

	// Scan for undefined variable error and print helpful suggestion
	for idx, sr := range runSteps {
		if strings.Contains(sr.Error, "undefined variable: ") {
			parts := strings.Split(sr.Error, "undefined variable: ")
			if len(parts) > 1 {
				varName := strings.TrimSpace(parts[1])

				type PrecedingMatch struct {
					Index   int
					Section string
					Line    int
					Path    string
				}
				var matches []PrecedingMatch

				stepLocations, _ := ScanSteps(cfg.TestPath)

				checkPreceding := func(steps []model.Step, secName string, limit int) {
					for i := 0; i < limit && i < len(steps); i++ {
						step := steps[i]
						if step.Save != nil {
							if _, ok := step.Save[varName]; ok {
								line := 0
								for _, loc := range stepLocations {
									if loc.Section == secName && loc.Index == i {
										line = loc.StartLine
										break
									}
								}
								matches = append(matches, PrecedingMatch{
									Index:   i,
									Section: secName,
									Line:    line,
									Path:    filepath.Base(cfg.TestPath),
								})
							}
						}
					}
				}

				// Compute step limit: cfg.StepIndex for single-step, len(stepList) for section-all
				stepLimit := cfg.StepIndex
				if stepLimit < 0 {
					stepLimit = len(stepList)
				}

				if section == "setup" {
					checkPreceding(testFile.Setup, "setup", stepLimit)
				} else if section == "steps" {
					checkPreceding(testFile.Setup, "setup", len(testFile.Setup))
					checkPreceding(testFile.Steps, "steps", stepLimit)
				} else if section == "teardown" {
					checkPreceding(testFile.Setup, "setup", len(testFile.Setup))
					checkPreceding(testFile.Steps, "steps", len(testFile.Steps))
					checkPreceding(testFile.Teardown, "teardown", stepLimit)
				}

				if len(matches) > 0 {
					var hint strings.Builder
					hint.WriteString(fmt.Sprintf("\n\n⚠ Step references $%s which is only available from:\n", varName))
					for _, m := range matches {
						hint.WriteString(fmt.Sprintf("  → %s:%s step %d at line %d (save: %s)\n", m.Path, m.Section, m.Index, m.Line, varName))
					}
					runSteps[idx].Error += hint.String()
				}
			}
		}
	}

	result := &RunResult{
		Scenario:     testFile.Scenario,
		TestFile:     cfg.TestPath,
		Account:      cfg.AccountName,
		Steps:        runSteps,
		ResolvedVars: initialVars,
		TotalPass:    pass,
		TotalFail:    fail,
		Duration:     time.Since(start),
		Passed:       passed,
	}

	// Write back session vars for subsequent step runs
	if cfg.SessionVars != nil {
		for key, val := range vars {
			cfg.SessionVars[key] = val
		}
	}

	// Persist session to file for CLI cross-process usage
	if cfg.SessionFile != "" {
		_ = saveSessionVars(cfg.SessionFile, vars)
	}

	return result, nil
}

// snapshotVars creates a copy of the variables map with sensitive fields masked.
// This is used for verbose output to show what values were available at runtime.
// The maskSensitive parameter controls whether masking is applied; when true,
// fields in maskFields are replaced with "***masked***".
func snapshotVars(vars map[string]interface{}, maskFields []string, maskSensitive ...bool) map[string]interface{} {
	if vars == nil {
		return nil
	}

	// Determine whether to actually mask (default: true when maskFields provided)
	shouldMask := true
	if len(maskFields) == 0 {
		maskFields = defaultSensitiveFields
	}
	if len(maskSensitive) > 0 {
		shouldMask = maskSensitive[0]
	}

	if !shouldMask {
		return vars
	}

	result := make(map[string]interface{})

	for key, val := range vars {
		// Check if this key should be masked
		if isSensitiveField(key, maskFields) {
			result[key] = "***masked***"
		} else if valMap, ok := val.(map[string]interface{}); ok {
			// Handle $accounts map - mask sensitive fields inside accounts
			maskedMap := make(map[string]interface{})
			for k, v := range valMap {
				if isSensitiveField(k, maskFields) {
					maskedMap[k] = "***masked***"
				} else {
					maskedMap[k] = v
				}
			}
			result[key] = maskedMap
		} else {
			result[key] = val
		}
	}

	return result
}

// parseUntil parses a target execution target, e.g. "steps:1" or "2" (default section: "steps").
func parseUntil(untilStr string, testFile *model.TestFile) (string, int, error) {
	untilStr = strings.TrimSpace(untilStr)
	if untilStr == "" {
		return "", -1, nil
	}

	section := "steps"
	indexStr := untilStr

	if strings.Contains(untilStr, ":") {
		parts := strings.SplitN(untilStr, ":", 2)
		section = strings.ToLower(strings.TrimSpace(parts[0]))
		indexStr = strings.TrimSpace(parts[1])
	}

	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return "", -1, fmt.Errorf("invalid step index in until pattern %q: %w", untilStr, err)
	}

	if index < 0 {
		return "", -1, fmt.Errorf("step index cannot be negative in until pattern %q", untilStr)
	}

	// Validate against actual step lists
	switch section {
	case "setup":
		if index >= len(testFile.Setup) {
			return "", -1, fmt.Errorf("until step index %d out of bounds for setup section (contains %d steps)", index, len(testFile.Setup))
		}
	case "steps":
		if index >= len(testFile.Steps) {
			return "", -1, fmt.Errorf("until step index %d out of bounds for steps section (contains %d steps)", index, len(testFile.Steps))
		}
	case "teardown":
		if index >= len(testFile.Teardown) {
			return "", -1, fmt.Errorf("until step index %d out of bounds for teardown section (contains %d steps)", index, len(testFile.Teardown))
		}
	default:
		return "", -1, fmt.Errorf("invalid section name %q in until pattern %q (must be one of: setup, steps, teardown)", section, untilStr)
	}

	return section, index, nil
}


// loadSessionVars reads a session file and returns the stored variables.
// Returns nil without error if the file doesn't exist.
func loadSessionVars(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var vars map[string]interface{}
	if err := yaml.Unmarshal(data, &vars); err != nil {
		return nil, err
	}
	return vars, nil
}

// saveSessionVars persists vars to a session file, excluding built-in generator keys.
func saveSessionVars(path string, vars map[string]interface{}) error {
	filtered := make(map[string]interface{})
	for k, v := range vars {
		if !isBuiltinKey(k) {
			filtered[k] = v
		}
	}
	if len(filtered) == 0 {
		os.Remove(path)
		return nil
	}
	data, err := yaml.Marshal(filtered)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// isBuiltinKey returns true if the key is a built-in generator variable name.
func isBuiltinKey(key string) bool {
	switch key {
	case "uuid", "ulid", "randomInt", "randomEmail", "randomPhone",
		"timestamp", "timestampMs", "accounts":
		return true
	}
	return false
}
