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
}

var (
	currentSink ScenarioSink
	sinkMu      sync.RWMutex
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
