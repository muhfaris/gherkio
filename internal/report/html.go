package report

import (
	"bytes"
	"html/template"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/muhfaris/gherkio/internal/fsutil"
)

type HTML struct {
	path    string
	rows    []Result
	mu      sync.Mutex
	created time.Time
	debug   bool
	meta    SummaryMeta
}

func NewHTML(path string, debug bool, meta SummaryMeta) *HTML {
	return &HTML{path: path, debug: debug, meta: meta}
}

func (h *HTML) Add(res Result) {
	h.mu.Lock()
	h.rows = append(h.rows, res)
	h.mu.Unlock()
}

func (h *HTML) Flush() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := fsutil.EnsureDir(dirOf(h.path)); err != nil {
		return err
	}
	rows := make([]Result, len(h.rows))
	copy(rows, h.rows)
	groups := groupByFeature(rows)
	buf := &bytes.Buffer{}
	summary := buildSummary(rows, h.meta, h.debug)
	data := struct {
		Generated time.Time
		Features  []featureGroup
		Summary   summaryData
		Debug     bool
	}{
		Generated: time.Now(),
		Features:  groups,
		Summary:   summary,
		Debug:     h.debug,
	}
	if err := htmlTemplate.Execute(buf, data); err != nil {
		return err
	}
	return os.WriteFile(h.path, buf.Bytes(), 0o644)
}

type featureGroup struct {
	Name      string
	Passed    int
	Failed    int
	Pending   int
	Scenarios []scenarioEntry
}

type scenarioEntry struct {
	Name       string
	Status     string
	DurationMs int64
	Groups     []stepGroup
}

type stepGroup struct {
	Title      string
	Status     string
	DurationMs int64
	Items      []StepDetail
	Debug      *DebugInfo
}

type summaryData struct {
	Env              string
	Tags             string
	Includes         []string
	Excludes         []string
	NameFilter       string
	FeatureCount     int
	Parallel         int
	TotalScenarios   int
	ScenariosPassed  int
	ScenariosFailed  int
	ScenariosPending int
	TotalSteps       int
	StepsPassed      int
	StepsFailed      int
	StepsPending     int
	TotalDurationMs  int64
	LongestScenario  string
	LongestDuration  int64
	Debug            bool
}

func groupByFeature(rows []Result) []featureGroup {
	if len(rows) == 0 {
		return nil
	}
	acc := map[string]*featureGroup{}
	order := []string{}
	for _, r := range rows {
		fg, ok := acc[r.Feature]
		if !ok {
			fg = &featureGroup{Name: r.Feature}
			acc[r.Feature] = fg
			order = append(order, r.Feature)
		}
		switch r.Status {
		case "PASSED":
			fg.Passed++
		case "FAILED":
			fg.Failed++
		case "PENDING":
			fg.Pending++
		}
		fg.Scenarios = append(fg.Scenarios, scenarioEntry{
			Name:       r.Scenario,
			Status:     r.Status,
			DurationMs: r.DurationMs,
			Groups:     buildGroups(r.Steps),
		})
	}
	groups := make([]featureGroup, 0, len(order))
	for _, name := range order {
		fg := acc[name]
		groups = append(groups, *fg)
	}
	return groups
}

func buildGroups(steps []StepDetail) []stepGroup {
	if len(steps) == 0 {
		return nil
	}
	groups := []stepGroup{}
	var current *stepGroup
	commit := func() {
		if current != nil {
			groups = append(groups, *current)
			current = nil
		}
	}
	for _, st := range steps {
		if isAnchorStep(st.Text) {
			commit()
			current = &stepGroup{
				Title:      st.Text,
				Status:     st.Status,
				DurationMs: st.DurationMs,
				Debug:      st.Debug,
			}
			continue
		}
		if current != nil {
			current.Items = append(current.Items, st)
		} else {
			groups = append(groups, stepGroup{
				Title:      st.Text,
				Status:     st.Status,
				DurationMs: st.DurationMs,
				Debug:      st.Debug,
			})
		}
	}
	commit()
	return groups
}

func isAnchorStep(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if strings.Contains(lower, " call api ") || strings.HasPrefix(lower, "call api ") {
		return true
	}
	if strings.Contains(lower, " run flow ") || strings.HasPrefix(lower, "run flow ") {
		return true
	}
	return false
}

func buildSummary(rows []Result, meta SummaryMeta, debug bool) summaryData {
	sum := summaryData{
		Env:          meta.Env,
		Tags:         meta.Tags,
		Includes:     cloneMetaSlice(meta.Includes),
		Excludes:     cloneMetaSlice(meta.Excludes),
		NameFilter:   meta.NameFilter,
		FeatureCount: meta.FeatureCount,
		Parallel:     meta.Parallel,
		Debug:        debug,
	}
	featureSet := map[string]struct{}{}
	for _, r := range rows {
		featureSet[r.Feature] = struct{}{}
		sum.TotalScenarios++
		switch strings.ToUpper(r.Status) {
		case "PASSED":
			sum.ScenariosPassed++
		case "FAILED":
			sum.ScenariosFailed++
		case "PENDING":
			sum.ScenariosPending++
		default:
			sum.ScenariosPending++
		}
		sum.TotalDurationMs += r.DurationMs
		if r.DurationMs > sum.LongestDuration {
			sum.LongestDuration = r.DurationMs
			sum.LongestScenario = r.Scenario
		}
		for _, st := range r.Steps {
			sum.TotalSteps++
			switch strings.ToUpper(st.Status) {
			case "PASSED":
				sum.StepsPassed++
			case "FAILED":
				sum.StepsFailed++
			case "PENDING", "SKIPPED":
				sum.StepsPending++
			default:
				sum.StepsPending++
			}
		}
	}
	if sum.FeatureCount == 0 {
		sum.FeatureCount = len(featureSet)
	}
	return sum
}

func cloneMetaSlice(src []string) []string {
	if len(src) == 0 {
		return nil
	}
	out := make([]string, len(src))
	copy(out, src)
	return out
}

var htmlTemplate = template.Must(template.New("gherkio-report").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>🥒 Gherkio Report</title>
  <style>
    body { font-family: system-ui, sans-serif; margin: 24px; background-color: #f5f5f5; color: #222; }
    h1 { margin-bottom: 8px; }
    details { margin-bottom: 12px; }
    summary { cursor: pointer; font-weight: 600; padding: 12px 16px; background: #fff; border: 1px solid #eee; box-shadow: 0 2px 6px rgba(0,0,0,0.08); border-radius: 6px; }
    summary::-webkit-details-marker { display: none; }
    summary::before { content: "▸"; display: inline-block; margin-right: 8px; transition: transform 0.2s ease; }
    details[open] summary::before { transform: rotate(90deg); }
    .summary-text { display: inline-flex; gap: 12px; align-items: center; }
    .badges span { display: inline-block; padding: 2px 8px; border-radius: 999px; font-size: 12px; background: #eef2f6; color: #425466; }
    .badges .passed { background: #e6f4ea; color: #0f7a2d; }
    .badges .failed { background: #fdecea; color: #b00020; }
    .badges .pending { background: #fff4e5; color: #8a5300; }
    .summary-section { margin: 18px 0 28px; }
    .summary-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 14px; margin-bottom: 16px; }
    .summary-card { background: #fff; border: 1px solid #e3e8ee; border-radius: 8px; padding: 16px; box-shadow: 0 1px 4px rgba(15, 23, 42, 0.08); }
    .summary-label { text-transform: uppercase; font-size: 12px; letter-spacing: 0.08em; color: #64748b; margin-bottom: 6px; }
    .summary-value { font-size: 24px; font-weight: 600; color: #1f2937; }
    .summary-meta { font-size: 12px; color: #475569; margin-top: 4px; }
    .meta-info { display: flex; flex-wrap: wrap; gap: 8px; font-size: 12px; color: #475569; margin-top: 4px; }
    .meta-pill { background: #fff; border: 1px dashed #cbd5f5; border-radius: 999px; padding: 4px 10px; }
    .status-PASSED { color: #17803d; font-weight: 600; }
    .status-FAILED { color: #d93025; font-weight: 600; }
    .status-PENDING { color: #d08701; font-weight: 600; }
    .status-SKIPPED { color: #1a73e8; font-weight: 600; }
    .status-UNDEFINED, .status-AMBIGUOUS { color: #8a5300; font-weight: 600; }
    .small { font-size: 12px; color: #666; }
    .scenario-summary { display: flex; align-items: center; width: 100%; gap: 16px; }
    .scenario-summary .scenario-name { flex: 1 1 auto; }
    .scenario-meta { font-size: 13px; color: #666; }
    .scenario-list { margin-top: 12px; display: flex; flex-direction: column; gap: 12px; }
    details.scenario { border: 1px solid #eee; border-radius: 6px; background: #fff; box-shadow: 0 1px 4px rgba(0,0,0,0.06); }
    details.scenario summary { border: none; box-shadow: none; border-radius: 6px; padding: 10px 14px; display: flex; align-items: center; gap: 16px; }
    details.scenario summary::before { content: "▸"; margin-right: 8px; transition: transform 0.2s ease; }
    details.scenario[open] summary::before { transform: rotate(90deg); }
    .scenario-summary { display: flex; align-items: center; width: 100%; gap: 16px; }
    .scenario-summary .scenario-name { flex: 1 1 auto; }
    .scenario-meta { font-size: 13px; color: #666; white-space: nowrap; }
    .step-line { display: flex; align-items: center; gap: 12px; padding: 10px 14px; background: #fafafa; border-radius: 6px; border: 1px solid #eee; font-size: 13px; }
    .step-line .step-duration { margin-left: auto; font-size: 12px; color: #666; }
    details.step-group { border: 1px solid #eee; border-radius: 6px; background: #fbfbfb; }
    details.step-group summary { display: flex; align-items: center; gap: 12px; padding: 10px 14px; font-size: 13px; cursor: pointer; }
    details.step-group summary::before { content: "▸"; margin-right: 4px; transition: transform 0.2s ease; }
    details.step-group[open] summary::before { transform: rotate(90deg); }
    .group-title { display: flex; align-items: center; gap: 8px; flex: 1 1 auto; }
    .group-meta { font-size: 12px; color: #666; white-space: nowrap; }
    .steps { margin: 0; padding: 10px 24px 12px 32px; }
    .steps li { margin-bottom: 6px; font-size: 13px; }
    .step-status { font-weight: 600; margin-right: 8px; }
    .step-error { color: #d93025; margin-left: 24px; }
    .step-duration { font-size: 12px; color: #666; margin-left: 24px; }
    details.debug-block, details.debug-inline { margin-top: 10px; }
    details.debug-block summary, details.debug-inline summary { font-size: 12px; color: #1a73e8; font-weight: 600; cursor: pointer; }
    details.debug-block summary::before, details.debug-inline summary::before { content: "▸"; margin-right: 6px; transition: transform 0.2s ease; display: inline-block; }
    details.debug-block[open] summary::before, details.debug-inline[open] summary::before { transform: rotate(90deg); }
    .debug-content { background: #fff; border: 1px solid #e3e8ee; border-radius: 6px; padding: 12px 16px; display: grid; gap: 12px; }
    .debug-section pre { background: #1f2933; color: #f1f5f9; padding: 12px; border-radius: 4px; overflow-x: auto; font-size: 12px; }
    .debug-title { font-size: 12px; font-weight: 600; color: #334155; margin-bottom: 4px; }
  </style>
</head>
<body>
  <h1>🥒 Gherkio Report</h1>
  <p class="small">Generated at: {{ .Generated.Format "2006-01-02 15:04:05 MST" }}</p>
  <section class="summary-section">
    <div class="summary-grid">
      <div class="summary-card">
        <div class="summary-label">Features</div>
        <div class="summary-value">{{.Summary.FeatureCount}}</div>
        <div class="summary-meta">{{if gt .Summary.Parallel 1}}Parallel {{.Summary.Parallel}} workers{{else}}Sequential run{{end}}</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">Scenarios</div>
        <div class="summary-value">{{.Summary.TotalScenarios}}</div>
        <div class="summary-meta">Passed {{.Summary.ScenariosPassed}} · Failed {{.Summary.ScenariosFailed}} · Pending {{.Summary.ScenariosPending}}</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">Steps</div>
        <div class="summary-value">{{.Summary.TotalSteps}}</div>
        <div class="summary-meta">Passed {{.Summary.StepsPassed}} · Failed {{.Summary.StepsFailed}} · Pending {{.Summary.StepsPending}}</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">Duration</div>
        <div class="summary-value">{{.Summary.TotalDurationMs}} ms</div>
        {{if .Summary.LongestScenario}}<div class="summary-meta">Longest: {{.Summary.LongestScenario}} ({{.Summary.LongestDuration}} ms)</div>{{end}}
      </div>
    </div>
    <div class="meta-info">
      {{if .Summary.Env}}<span class="meta-pill"><strong>Env:</strong> {{.Summary.Env}}</span>{{end}}
      {{if .Summary.Tags}}<span class="meta-pill"><strong>Tags:</strong> {{.Summary.Tags}}</span>{{end}}
      {{if .Summary.NameFilter}}<span class="meta-pill"><strong>Name filter:</strong> {{.Summary.NameFilter}}</span>{{end}}
      {{if .Summary.Includes}}
        <span class="meta-pill"><strong>Includes:</strong>
          {{range $i, $v := .Summary.Includes}}{{if $i}}, {{end}}{{$v}}{{end}}
        </span>
      {{end}}
      {{if .Summary.Excludes}}
        <span class="meta-pill"><strong>Excludes:</strong>
          {{range $i, $v := .Summary.Excludes}}{{if $i}}, {{end}}{{$v}}{{end}}
        </span>
      {{end}}
      {{if .Summary.Debug}}<span class="meta-pill">Debug payloads enabled</span>{{end}}
    </div>
  </section>
  {{if .Features}}
    {{range .Features}}
      <details open>
        <summary>
          <span class="summary-text">{{.Name}}
            <span class="badges">
              <span class="passed">Passed: {{.Passed}}</span>
              <span class="failed">Failed: {{.Failed}}</span>
              {{if gt .Pending 0}}<span class="pending">Pending: {{.Pending}}</span>{{end}}
            </span>
          </span>
        </summary>
        <div class="scenario-list">
          {{range .Scenarios}}
            <details class="scenario" open>
              <summary>
                <div class="scenario-summary">
                  <span class="scenario-name">{{.Name}}</span>
                  <span class="scenario-meta">
                    <span class="status-{{.Status}}">{{.Status}}</span>
                    · {{.DurationMs}} ms
                  </span>
                </div>
              </summary>
              <div class="scenario-steps">
                {{range .Groups}}
                  {{if .Items}}
                    <details class="step-group" open>
                      <summary>
                        <span class="group-title">
                          <span class="step-status status-{{.Status}}">{{.Status}}</span>{{.Title}}
                        </span>
                        <span class="group-meta">{{.DurationMs}} ms</span>
                      </summary>
                      <ol class="steps">
                        {{range .Items}}
                          <li>
                            <span class="step-status status-{{.Status}}">{{.Status}}</span>{{.Text}}
                            {{if .Error}}<div class="step-error">{{.Error}}</div>{{end}}
                            {{if ne .DurationMs 0}}<div class="step-duration">{{.DurationMs}} ms</div>{{end}}
                          </li>
                        {{end}}
                      </ol>
                      {{if and $.Debug .Debug}}
                        <details class="debug-block">
                          <summary>Show payload</summary>
                          <div class="debug-content">
                            <div class="debug-section">
                              <div class="debug-title">Request Body</div>
                              <pre>{{.Debug.RequestBody}}</pre>
                            </div>
                            <div class="debug-section">
                              <div class="debug-title">Response (status {{.Debug.ResponseStatus}})</div>
                              <pre>{{.Debug.ResponseBody}}</pre>
                            </div>
                          </div>
                        </details>
                      {{end}}
                    </details>
                  {{else}}
                    <div class="step-line">
                      <span class="step-status status-{{.Status}}">{{.Status}}</span>{{.Title}}
                      {{if ne .DurationMs 0}}<span class="step-duration">{{.DurationMs}} ms</span>{{end}}
                    </div>
                    {{if and $.Debug .Debug}}
                      <details class="debug-inline">
                        <summary>Show payload</summary>
                        <div class="debug-content">
                          <div class="debug-section">
                            <div class="debug-title">Request Body</div>
                            <pre>{{.Debug.RequestBody}}</pre>
                          </div>
                          <div class="debug-section">
                            <div class="debug-title">Response (status {{.Debug.ResponseStatus}})</div>
                            <pre>{{.Debug.ResponseBody}}</pre>
                          </div>
                        </div>
                      </details>
                    {{end}}
                  {{end}}
                {{end}}
              </div>
            </details>
          {{end}}
        </div>
      </details>
    {{end}}
  {{else}}
    <p style="text-align:center; padding:24px; color:#999;">No scenarios recorded</p>
  {{end}}
</body>
</html>`))
