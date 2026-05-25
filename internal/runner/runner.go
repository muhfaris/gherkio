package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	DryRun         bool                  // Preview without executing HTTP requests
}

// RunResult holds the overall execution result.
type RunResult struct {
	Scenario    string        `json:"scenario"`
	TestFile    string        `json:"testFile,omitempty"`
	Account     string        `json:"account,omitempty"` // Account name used (if any)
	Steps       []StepResult  `json:"steps"`
	ResolvedVars map[string]interface{} `json:"resolvedVars,omitempty"` // Variables available at start of scenario
	TotalPass   int           `json:"totalPass"`
	TotalFail   int           `json:"totalFail"`
	Duration    time.Duration `json:"duration"`
	Passed      bool          `json:"passed"`
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

	// Route to single-step or section-only execution
	if cfg.StepIndex >= 0 || cfg.StepSection != "" {
		return RunSingleStep(cfg, env, testFile)
	}

	// 3. Execute steps
	result := &RunResult{
		Scenario: testFile.Scenario,
		Account:  cfg.AccountName,
	}

	vars := make(map[string]interface{})

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

	// Capture initial variable state for verbose output
	result.ResolvedVars = snapshotVars(vars, cfg.MaskFields)

	currentDir := filepath.Dir(cfg.TestPath)
	var allSteps []StepResult
	setupFailed := false

	// Execute setup steps first
	if len(testFile.Setup) > 0 {
		setupSteps, setupPass, setupFail, setupPassed := executeSteps(testFile.Setup, env, vars, cfg.ProjectDir, currentDir, 0, "setup", cfg.DryRun)
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
		mainSteps, mainPass, mainFail, mainPassed := executeSteps(testFile.Steps, env, vars, cfg.ProjectDir, currentDir, 0, "steps", cfg.DryRun)
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
		teardownSteps, _, _, _ := executeSteps(testFile.Teardown, env, vars, cfg.ProjectDir, currentDir, 0, "teardown", cfg.DryRun)
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

	return result, nil
}

// executeSteps executes a list of steps. If dryRun is true, skips HTTP calls and produces preview output.
func executeSteps(steps []model.Step, env *model.Environment, vars map[string]interface{}, projectDir string, currentDir string, depth int, role string, dryRun bool) ([]StepResult, int, int, bool) {
	var stepResults []StepResult
	totalPass := 0
	totalFail := 0
	allPassed := true

	for _, step := range steps {
		stepStart := time.Now()
		stepResult := StepResult{
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
			nestedSteps, nestedPass, nestedFail, _ := executeSteps(usedTest.Steps, env, vars, projectDir, usedCurrentDir, depth+1, role, dryRun)

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
			continue
		}

		url := resolveURL(env, interpolatedRequest)

		stepResult.Request = &RequestInfo{
			Method:  interpolatedRequest.Method,
			URL:     url,
			Headers: interpolatedRequest.Headers,
		}
		if interpolatedRequest.Body != nil {
			if bodyJSON, err := json.Marshal(interpolatedRequest.Body); err == nil {
				stepResult.Request.Body = string(bodyJSON)
			} else {
				stepResult.Request.Body = fmt.Sprintf("%v", interpolatedRequest.Body)
			}
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
			fmt.Printf("⚠ %s %s — retrying non-idempotent request\n", interpolatedRequest.Method, url)
		}

		var lastResp *ResponseInfo
		var lastJwtClaims map[string]interface{}
		var lastAssertions []AssertionResult

		for i := 1; i <= attempts; i++ {
			if maxDuration > 0 && time.Since(stepStart) >= maxDuration {
				stepErr = fmt.Errorf("maxDuration %s exceeded", step.Retry.MaxDuration)
				break
			}

			attemptStart := time.Now()
			resp, err = executeRequest(interpolatedRequest.Method, url, interpolatedRequest.Headers, interpolatedRequest.Body, interpolatedRequest.Timeout)

			entry := RetryEntry{
				Attempt:  i,
				Duration: time.Since(attemptStart),
			}

			if err != nil {
				entry.Error = err.Error()
				retryHistory = append(retryHistory, entry)
				if i == attempts {
					stepErr = err
					break
				}
				time.Sleep(calculateBackoff(backoffStrat, intervalMs, i))
				continue
			}

			entry.Status = resp.Status
			if len(resp.Body) > 500 {
				entry.Body = resp.Body[:500] + "..."
			} else {
				entry.Body = resp.Body
			}
			retryHistory = append(retryHistory, entry)

			jwtClaims = nil
			if resp.Parsed != nil {
				for _, field := range []string{"token", "accessToken", "access_token"} {
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

			assertions = runAssertions(resp.Status, resp, jwtClaims, step.Expect.Status, step.Expect.Extra, projectDir)

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
			stepResults = append(stepResults, stepResult)
			continue
		}

		stepResult.Response = lastResp
		stepResult.Assertions = lastAssertions

		// Extract variables
		if lastResp != nil {
			extractValues(vars, step.Save, lastResp, lastJwtClaims)
		}

		stepResult.Duration = time.Since(stepStart)
		if hasRetry && len(retryHistory) > 1 {
			stepResult.RetryCount = len(retryHistory) - 1
			stepResult.RetryHistory = retryHistory
		}

		// Count pass/fail
		for _, a := range stepResult.Assertions {
			if a.Passed {
				totalPass++
			} else {
				totalFail++
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
			}
		}

		stepResults = append(stepResults, stepResult)
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
func resolveURL(env *model.Environment, req model.Request) string {
	baseURL := env.BaseURL

	if req.Service != "" && env.Services != nil {
		if svc, ok := env.Services[req.Service]; ok {
			baseURL = svc.BaseURL
		}
	}

	return baseURL + req.URL
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

	if cfg.CredentialVars != nil {
		for key, val := range cfg.CredentialVars {
			vars[key] = val
		}
	}

	// Inject all accounts as $accounts.<name>.<field> for dotted-path access
	if cfg.AllAccounts != nil {
		vars["accounts"] = cfg.AllAccounts
	}

	// Capture initial variable state for verbose output
	initialVars := snapshotVars(vars, cfg.MaskFields)

	currentDir := filepath.Dir(cfg.TestPath)
	runSteps, pass, fail, passed := executeSteps(stepsToRun, env, vars, cfg.ProjectDir, currentDir, 0, section, cfg.DryRun)

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
		Scenario:    testFile.Scenario,
		TestFile:    cfg.TestPath,
		Account:     cfg.AccountName,
		Steps:       runSteps,
		ResolvedVars: initialVars,
		TotalPass:   pass,
		TotalFail:   fail,
		Duration:    time.Since(start),
		Passed:      passed,
	}

	return result, nil
}

// snapshotVars creates a copy of the variables map with sensitive fields masked.
// snapshotVars creates a copy of the variables map with sensitive fields masked.
// This is used for verbose output to show what values were available at runtime.
func snapshotVars(vars map[string]interface{}, maskFields []string) map[string]interface{} {
	if vars == nil {
		return nil
	}

	result := make(map[string]interface{})

	// If no mask fields provided, use defaults
	if len(maskFields) == 0 {
		maskFields = defaultSensitiveFields
	}

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
