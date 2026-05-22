package report


// ReportData represents the top-level data structure passed to the HTML template.
type ReportData struct {
	ScenarioName  string
	Environment   string
	Timestamp     string
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
}

// MapResultToReportData converts a runner.RunResult into a ReportData struct.
// It is expected to be called during rendering.
