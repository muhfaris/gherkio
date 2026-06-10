package report

// ReportData represents the top-level data structure passed to the HTML template.
// For single-scenario runs, Steps is populated and Scenarios is nil.
// For multi-scenario runs, Scenarios is populated and Steps is nil.
type ReportData struct {
	ScenarioName  string
	Description   string
	Environment   string
	Timestamp     string
	TotalDuration string
	TotalSteps    int
	PassCount     int
	FailCount     int
	PassPercent   float64
	FailPercent   float64
	Steps         []ReportStep   // populated for single-scenario runs
	Scenarios     []ScenarioData // populated for multi-scenario runs
}

// ScenarioData holds the report data for a single scenario within a suite run.
type ScenarioData struct {
	Name          string
	Description   string
	TestFile      string
	Account       string `json:"account,omitempty"` // Account name used (if any)
	TotalDuration string
	TotalSteps    int
	PassCount     int
	FailCount     int
	PassPercent   float64
	FailPercent   float64
	Steps         []ReportStep
}

// ReportStep represents a single step in the report.
type ReportStep struct {
	Index        int
	Method       string
	URL          string
	StatusCode   int
	StatusText   string
	Duration     string
	TimingFailed bool
	RequestID    string
	CurlCommand  string
	RequestBody  string
	ResponseBody string
	Passed       bool
	Assertions   []ReportAssertion
	Error        string
	RetryCount   int
	RetryHistory []RetryEntry
	Role         string `json:"role,omitempty"` // "setup", "steps", "teardown"
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
