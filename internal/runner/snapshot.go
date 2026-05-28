package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// SnapshotConfig holds the configuration for failure snapshot generation.
type SnapshotConfig struct {
	Enabled       bool
	Path          string
	MaskSensitive bool
	MaskFields    []string
	RetainCount   int
}

// FailureSnapshot is a copy of the model type for use within the runner package.
// This avoids circular import issues between runner and model packages.
type FailureSnapshot struct {
	Timestamp    time.Time             `json:"timestamp"`
	Scenario     string                `json:"scenario"`
	TestFile     string                `json:"testFile"`
	FailedStep   int                   `json:"failedStepIndex"`
	FailedLine   int                   `json:"failedStepLine"`
	Role         string                `json:"role"`
	ErrorMessage string                `json:"error"`
	Diagnostics  SnapshotDiagnostics   `json:"diagnostics"`
}

// SnapshotDiagnostics contains detailed request/response state at the moment of failure.
type SnapshotDiagnostics struct {
	Request       SnapshotRequest       `json:"request"`
	Response      SnapshotResponse      `json:"response"`
	VariableStore map[string]interface{} `json:"variableStoreAtFailure"`
}

// SnapshotRequest captures the executed HTTP request details at failure time.
type SnapshotRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    interface{}       `json:"body"`
}

// SnapshotResponse captures the response details at failure time.
type SnapshotResponse struct {
	StatusCode int               `json:"statusCode"`
	DurationMs int64             `json:"durationMs"`
	Headers    map[string]string `json:"headers"`
	Body       interface{}       `json:"body"`
}

var (
	// slugifyRegex matches non-alphanumeric characters for filename sanitization.
	slugifyRegex = regexp.MustCompile(`[^a-zA-Z0-9]+`)
)

// writeFailureSnapshot writes a failure snapshot to a JSON file.
func writeFailureSnapshot(
	cfg SnapshotConfig,
	result *StepResult,
	stepIndex int,
	scenario string,
	testFile string,
	role string,
	vars map[string]interface{},
) error {
	if !cfg.Enabled || cfg.Path == "" {
		return nil
	}

	// Determine the step line number
	stepLine := 0
	locations, err := ScanSteps(testFile)
	if err == nil {
		for _, loc := range locations {
			if loc.Section == role && loc.Index == stepIndex {
				stepLine = loc.StartLine
				break
			}
		}
	}

	// Build error message - get the specific assertion failures
	var errorMessages []string
	if result.Error != "" {
		errorMessages = append(errorMessages, result.Error)
	}
	for _, a := range result.Assertions {
		if !a.Passed {
			if a.Reason != "" {
				errorMessages = append(errorMessages, fmt.Sprintf("assertion '%s' failed: %s (expected: %s, actual: %s)", a.Path, a.Reason, a.Expected, a.Actual))
			} else {
				errorMessages = append(errorMessages, fmt.Sprintf("assertion '%s' failed: expected %s, got %s", a.Path, a.Expected, a.Actual))
			}
		}
	}

	errorMsg := ""
	if len(errorMessages) > 0 {
		errorMsg = strings.Join(errorMessages, "; ")
	}

	// Build the snapshot
	snapshot := FailureSnapshot{
		Timestamp:    time.Now().UTC(),
		Scenario:     scenario,
		TestFile:     testFile,
		FailedStep:   stepIndex,
		FailedLine:   stepLine,
		Role:         role,
		ErrorMessage: errorMsg,
	}

	// Add request diagnostics
	if result.Request != nil {
		req := SnapshotRequest{
			Method:  result.Request.Method,
			URL:     result.Request.URL,
			Headers: result.Request.Headers,
		}
		if result.Request.Body != "" {
			req.Body = parseBodyAsInterface(result.Request.Body, cfg.MaskFields, cfg.MaskSensitive)
		}
		snapshot.Diagnostics.Request = req
	}

	// Add response diagnostics
	if result.Response != nil {
		var body interface{}
		if result.Response.Body != "" {
			body = parseBodyAsInterface(result.Response.Body, cfg.MaskFields, cfg.MaskSensitive)
		}
		snapshot.Diagnostics.Response = SnapshotResponse{
			StatusCode: result.Response.Status,
			DurationMs: int64(result.Duration / time.Millisecond),
			Headers:    result.Response.Headers,
			Body:       body,
		}
	}

	// Add variable store
	snapshot.Diagnostics.VariableStore = snapshotVars(vars, cfg.MaskFields, cfg.MaskSensitive)

	// Ensure the directory exists
	if err := os.MkdirAll(cfg.Path, 0755); err != nil {
		return fmt.Errorf("failed to create failure snapshot directory: %w", err)
	}

	// Generate unique filename
	timestamp := time.Now().UTC().Format("20060102-150405.000")
	scenarioSlug := slugifyRegex.ReplaceAllString(scenario, "-")
	filename := fmt.Sprintf("failure-%s-step%d-%s.json", scenarioSlug, stepIndex, timestamp)
	filePath := filepath.Join(cfg.Path, filename)

	// Write the snapshot
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal failure snapshot: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write failure snapshot: %w", err)
	}

	fmt.Fprintf(os.Stderr, "📸 Failure snapshot written: %s\n", filename)

	// Perform housekeeping if retain count is set
	if cfg.RetainCount > 0 {
		if err := pruneFailureSnapshots(cfg.Path, cfg.RetainCount); err != nil {
			// Log but don't fail
			fmt.Fprintf(os.Stderr, "Warning: failed to prune old snapshots: %v\n", err)
		}
	}

	return nil
}

// parseBodyAsInterface parses a JSON body and optionally masks sensitive fields.
func parseBodyAsInterface(body string, maskFields []string, maskSensitive bool) interface{} {
	if body == "" {
		return nil
	}

	var parsed interface{}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return body
	}

	if maskSensitive && len(maskFields) > 0 {
		parsed = MaskSensitiveData(parsed, maskFields)
	}

	return parsed
}


// pruneFailureSnapshots removes old snapshots beyond the retain count.
func pruneFailureSnapshots(dir string, retainCount int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	// Find all failure snapshot files
	var snapshotFiles []os.FileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), "failure-") && strings.HasSuffix(entry.Name(), ".json") {
			info, err := entry.Info()
			if err == nil {
				snapshotFiles = append(snapshotFiles, info)
			}
		}
	}

	// If we're within the limit, nothing to prune
	if len(snapshotFiles) <= retainCount {
		return nil
	}

	// Sort by modification time (oldest first)
	sort.Slice(snapshotFiles, func(i, j int) bool {
		return snapshotFiles[i].ModTime().Before(snapshotFiles[j].ModTime())
	})

	// Remove the oldest files beyond the retain count
	filesToRemove := len(snapshotFiles) - retainCount
	for i := 0; i < filesToRemove; i++ {
		filePath := filepath.Join(dir, snapshotFiles[i].Name())
		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("failed to remove old snapshot %s: %w", snapshotFiles[i].Name(), err)
		}
	}

	return nil
}
