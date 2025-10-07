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
}
