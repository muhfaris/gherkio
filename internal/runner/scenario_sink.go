package runner

import "sync"

type ScenarioSink interface {
	RecordScenario(feature, scenario, status string, durationMs int64, steps []StepLog)
}

type StepLog struct {
	Text       string
	Status     string
	DurationMs int64
	Error      string
	Debug      *StepDebug
}

type StepDebug struct {
	APIKey         string
	RequestMethod  string
	RequestURL     string
	RequestHeaders string
	RequestBody    string
	ResponseBody   string
	ResponseStatus int
	RequestCurl    string
}

var (
	currentSink  ScenarioSink
	sinkMu       sync.RWMutex
	debugCapture bool
	debugConsole bool
	debugMu      sync.RWMutex
)

func SetScenarioSink(s ScenarioSink) func() {
	sinkMu.Lock()
	prev := currentSink
	currentSink = s
	sinkMu.Unlock()
	return func() {
		sinkMu.Lock()
		currentSink = prev
		sinkMu.Unlock()
	}
}

func recordScenario(feature, scenario, status string, durMs int64, steps []StepLog) {
	sinkMu.RLock()
	defer sinkMu.RUnlock()
	if currentSink != nil {
		currentSink.RecordScenario(feature, scenario, status, durMs, steps)
	}
}

func SetDebugCapture(enabled bool) func() {
	debugMu.Lock()
	prev := debugCapture
	debugCapture = enabled
	debugMu.Unlock()
	return func() {
		debugMu.Lock()
		debugCapture = prev
		debugMu.Unlock()
	}
}

func SetDebugConsole(enabled bool) func() {
	debugMu.Lock() // reuse same mutex/state
	prev := debugConsole
	debugConsole = enabled
	debugMu.Unlock()
	return func() {
		debugMu.Lock()
		debugConsole = prev
		debugMu.Unlock()
	}
}

func isDebugCapture() bool {
	debugMu.RLock()
	defer debugMu.RUnlock()
	return debugCapture
}

func isDebugConsole() bool {
	debugMu.RLock()
	defer debugMu.RUnlock()
	return debugConsole
}
