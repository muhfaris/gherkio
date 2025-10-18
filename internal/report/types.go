package report

type Result struct {
	Feature    string
	Scenario   string
	Status     string // PASSED/FAILED/SKIPPED
	DurationMs int64
	Steps      []StepDetail
}

type StepDetail struct {
	Text       string
	Status     string
	DurationMs int64
	Error      string
	Debug      *DebugInfo
}

type DebugInfo struct {
	APIKey         string
	RequestMethod  string
	RequestURL     string
	RequestHeaders string
	RequestBody    string
	ResponseBody   string
	ResponseStatus int
}

type SummaryMeta struct {
	Env          string
	Tags         string
	Includes     []string
	Excludes     []string
	NameFilter   string
	FeatureCount int
	Parallel     int
}
