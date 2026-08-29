package report

import "github.com/muhfaris/gherkio/internal/runner"

// ReportData represents the top-level data structure passed to the HTML template.
// For single-scenario runs, Steps is populated and Scenarios is nil.
// For multi-scenario runs, Scenarios is populated and Steps is nil.
type ReportData struct {
	ScenarioName        string
	Description         string
	Environment         string
	Timestamp           string
	TotalDuration       string
	TotalSteps          int
	PassCount           int
	FailCount           int
	SkipCount           int
	PassPercent         float64
	FailPercent         float64
	SkipPercent         float64
	Steps               []ReportStep   // populated for single-scenario runs
	Scenarios           []ScenarioData // populated for multi-scenario runs
	LoadMode            bool           `json:"loadMode,omitempty"`
	VirtualUsers        int            `json:"virtualUsers,omitempty"`
	IterationsPerUser   int            `json:"iterationsPerUser,omitempty"`
	WorkflowCount       int            `json:"workflowCount,omitempty"`
	PassedWorkflows     int            `json:"passedWorkflows,omitempty"`
	FailedWorkflows     int            `json:"failedWorkflows,omitempty"`
	RequestCount        int            `json:"requestCount,omitempty"`
	AverageResponseTime string         `json:"averageResponseTime,omitempty"`
	P95ResponseTime     string         `json:"p95ResponseTime,omitempty"`
	RequestsPerSecond   string         `json:"requestsPerSecond,omitempty"`
}

// ScenarioData holds the report data for a single scenario within a suite run.
type ScenarioData struct {
	Name              string
	Description       string
	TestFile          string
	Account           string `json:"account,omitempty"` // Account name used (if any)
	TotalDuration     string
	TotalSteps        int
	PassCount         int
	FailCount         int
	SkipCount         int
	PassPercent       float64
	FailPercent       float64
	SkipPercent       float64
	Steps             []ReportStep
	Passed            bool
	VirtualUser       int `json:"virtualUser,omitempty"`
	Iteration         int `json:"iteration,omitempty"`
	IterationsPerUser int `json:"iterationsPerUser,omitempty"`
}

// ReportStep represents a single step in the report.
type ReportStep struct {
	Index        int
	Name         string
	Method       string
	URL          string
	Query        map[string]any    `json:"query,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	StatusCode   int
	StatusText   string
	Duration     string
	TimingFailed bool
	RequestID    string
	CurlCommand  string
	SavedVars    map[string]interface{} `json:"savedVars,omitempty"`
	RequestBody  string
	ResponseBody string
	Passed       bool
	Skipped      bool
	Assertions   []ReportAssertion
	Error        string
	Warnings     []string `json:"warnings,omitempty"`
	RetryCount   int
	RetryHistory []RetryEntry
	Role         string `json:"role,omitempty"` // "setup", "steps", "teardown"
	Original     runner.StepResult
}

// RetryEntry mirrors runner.RetryEntry for the report scope.
type RetryEntry struct {
	Attempt  int
	Status   int
	Body     string
	Duration string
	Error    string
}

// ReportAssertion represents a single assertion result in the report.
type ReportAssertion struct {
	Label  string
	Detail string
	Passed bool
}

// ReportConfig holds options for report generation.
type ReportConfig struct {
	Format        string
	Path          string
	MaskSensitive bool
	MaskFields    []string
	Retention     int
}

// MapResultToReportData converts a runner.RunResult into a ReportData struct.
// It is expected to be called during rendering.
