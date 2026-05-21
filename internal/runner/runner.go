package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/muhfaris/gherkio/internal/model"
	"gopkg.in/yaml.v3"
)

// RunConfig holds configuration for a single run.
type RunConfig struct {
	TestPath   string   // Path to the test YAML file
	EnvName    string   // Environment name (e.g. "local", "staging")
	ProjectDir string   // Root project directory (where .gherkio lives)
	Verbose    bool     // Show full request/response payloads
	MaskFields []string // Sensitive field names to mask in output (nil = use defaults)
}

// RunResult holds the overall execution result.
type RunResult struct {
	Scenario  string        `json:"scenario"`
	Steps     []StepResult  `json:"steps"`
	TotalPass int           `json:"totalPass"`
	TotalFail int           `json:"totalFail"`
	Duration  time.Duration `json:"duration"`
	Passed    bool          `json:"passed"`
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
	testFile, err := loadTestFile(cfg.TestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load test file: %w", err)
	}

	// 3. Execute steps
	result := &RunResult{
		Scenario: testFile.Scenario,
	}

	vars := make(map[string]interface{})
	currentDir := filepath.Dir(cfg.TestPath)
	steps, passes, fails, passed := executeSteps(testFile.Steps, env, vars, cfg.ProjectDir, currentDir, 0)

	result.Steps = steps
	result.TotalPass = passes
	result.TotalFail = fails
	result.Passed = passed

	result.Duration = time.Since(start)

	return result, nil
}

func executeSteps(steps []model.Step, env *model.Environment, vars map[string]interface{}, projectDir string, currentDir string, depth int) ([]StepResult, int, int, bool) {
	var stepResults []StepResult
	totalPass := 0
	totalFail := 0
	allPassed := true

	for _, step := range steps {
		stepStart := time.Now()
		stepResult := StepResult{
			Original: step,
			Depth:    depth,
		}

		// Handle 'use' step recursively
		if step.Use != "" {
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

			usedTest, err := loadTestFile(resolvedPath)
			if err != nil {
				stepResult.Error = fmt.Sprintf("failed to load used test file '%s': %v", resolvedPath, err)
				stepResults = append(stepResults, stepResult)
				allPassed = false
				continue
			}

			usedCurrentDir := filepath.Dir(resolvedPath)
			nestedSteps, nestedPass, nestedFail, _ := executeSteps(usedTest.Steps, env, vars, projectDir, usedCurrentDir, depth+1)

			// Flatten the results
			stepResults = append(stepResults, nestedSteps...)
			// add a dummy end step for 'use'
			useEndStep := StepResult{
				Original: step,
				Depth:    depth,
				IsUseEnd: true,
				UseFile:  step.Use,
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
		// Interpolate variables in the request
		interpolatedRequest, err := InterpolateRequest(step.Request, vars)
		if err != nil {
			stepResult.Error = fmt.Sprintf("Variable interpolation failed: %v", err)
			stepResults = append(stepResults, stepResult)
			allPassed = false
			continue
		}

		// Resolve URL using service or global baseUrl
		url := resolveURL(env, interpolatedRequest)

		// Execute HTTP request
		resp, err := executeRequest(interpolatedRequest.Method, url, interpolatedRequest.Headers, interpolatedRequest.Body)
		if err != nil {
			stepResult.Error = err.Error()
			stepResults = append(stepResults, stepResult)
			allPassed = false
			continue
		}

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

		stepResult.Response = resp

		// Decode JWT if present in response (checks common field names)
		var jwtClaims map[string]interface{}
		if resp.Parsed != nil {
			for _, field := range []string{"token", "accessToken", "access_token"} {
				if tokenVal, found := resolvePath(resp.Parsed, field); found {
					if tokenStr, ok := tokenVal.(string); ok {
						claims, err := decodeJWT(tokenStr)
						if err == nil {
							jwtClaims = claims
							break
						}
					}
				}
			}
		}

		// Run assertions
		stepResult.Assertions = runAssertions(resp.Status, resp, jwtClaims, step.Expect.Status, step.Expect.Extra, projectDir)

		// Count pass/fail
		for _, a := range stepResult.Assertions {
			if a.Passed {
				totalPass++
			} else {
				totalFail++
			}
		}

		// Extract variables
		extractValues(vars, step.Save, resp, jwtClaims)

		stepResult.Duration = time.Since(stepStart)

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

// loadTestFile reads and parses a test YAML file.
func loadTestFile(path string) (*model.TestFile, error) {
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
