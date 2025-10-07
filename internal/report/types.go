package report

type Result struct {
	Feature    string
	Scenario   string
	Status     string // PASSED/FAILED/SKIPPED
	DurationMs int64
}
