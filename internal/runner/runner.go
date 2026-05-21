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

	for _, step := range testFile.Steps {
		stepStart := time.Now()
		stepResult := StepResult{}

		// Resolve URL using service or global baseUrl
		// Interpolate variables in the request
		interpolatedRequest, err := InterpolateRequest(step.Request, vars)
		if err != nil {
			stepResult.Error = fmt.Sprintf("Variable interpolation failed: %v", err)
			result.Steps = append(result.Steps, stepResult)
			result.Passed = false
			continue
		}

		// Resolve URL using service or global baseUrl
		url := resolveURL(env, interpolatedRequest)

		// Execute HTTP request
		resp, err := executeRequest(step.Request.Method, url, step.Request.Headers, step.Request.Body)
		if err != nil {
			stepResult.Error = err.Error()
			result.Steps = append(result.Steps, stepResult)
			result.Passed = false
			continue
		}

		stepResult.Request = &RequestInfo{
			Method:  step.Request.Method,
			URL:     url,
			Headers: step.Request.Headers,
		}
		if step.Request.Body != nil {
			if bodyJSON, err := json.Marshal(step.Request.Body); err == nil {
				stepResult.Request.Body = string(bodyJSON)
			} else {
				stepResult.Request.Body = fmt.Sprintf("%v", step.Request.Body)
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
		stepResult.Assertions = runAssertions(resp.Status, resp, jwtClaims, step.Expect.Status, step.Expect.Extra)

		// Count pass/fail
		for _, a := range stepResult.Assertions {
			if a.Passed {
				result.TotalPass++
			} else {
				result.TotalFail++
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
				result.TotalPass++
			} else {
				result.TotalFail++
			}
		}

		result.Steps = append(result.Steps, stepResult)
	}

	result.Duration = time.Since(start)
	result.Passed = result.TotalFail == 0

	return result, nil
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
