package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/cucumber/godog"
	"github.com/muhfaris/gherkio/internal/loader"
	"github.com/muhfaris/gherkio/internal/report"
	"github.com/muhfaris/gherkio/internal/runner"
	"github.com/spf13/cobra"
)

type htmlSpec struct {
	Path  string
	Debug bool
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run Gherkin features (journey)",
	Long:  `Executes Gherkin feature files to run API journeys.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRun(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().String("env", "", "environment name")
	runCmd.Flags().Bool("debug", false, "print request/response and include debug info in HTML report")
	runCmd.Flags().String("tags", "", "tag expression (e.g. \"@smoke and not @wip\")")
	runCmd.Flags().String("name", "", "filter Scenario name by regex (best-effort)")
	runCmd.Flags().Int("parallel", 1, "number of parallel workers (by feature file)")
	runCmd.Flags().StringArray("feature", []string{}, "repeatable include filter")
	runCmd.Flags().StringArray("exclude-feature", []string{}, "repeatable exclude filter")
	runCmd.Flags().StringArray("report", []string{}, "Reporters (pretty|html|junit|cucumber|csv)")
}

func runRun(cmd *cobra.Command, args []string) (err error) {
	envName, _ := cmd.Flags().GetString("env")
	debug, _ := cmd.Flags().GetBool("debug")
	tags, _ := cmd.Flags().GetString("tags")
	nameRegex, _ := cmd.Flags().GetString("name")
	parallel, _ := cmd.Flags().GetInt("parallel")
	includes, _ := cmd.Flags().GetStringArray("feature")
	excludes, _ := cmd.Flags().GetStringArray("exclude-feature")
	reports, _ := cmd.Flags().GetStringArray("report")

	if envName == "" {
		return errors.New("--env is required")
	}

	env, err := loader.LoadEnv("gherkio/envs", envName)
	if err != nil {
		return err
	}
	cat, err := loader.LoadCatalogs("gherkio/apis")
	if err != nil {
		return err
	}

	// try load flows; optional (avoid unused variable)
	flows, err := loader.LoadFlows("gherkio/flows")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: failed to load flows:", err)
	}
	if len(flows) > 0 {
		fmt.Printf("Loaded %d flow(s) from gherkio/flows\n", len(flows))
	}

	featDir := "gherkio/features"
	features, err := loader.FindFeatures(featDir)
	if err != nil {
		return err
	}
	if len(features) == 0 {
		return fmt.Errorf("no .feature found in %s", featDir)
	}

	// include / exclude by path (substring or glob)
	if len(includes) > 0 {
		features = filterFeatures(features, includes, true)
	}

	if len(excludes) > 0 {
		features = filterFeatures(features, excludes, false)
	}

	// optional: filter by Scenario name (regex) by scanning file content (best-effort)
	if nameRegex != "" {
		f2, err := filterFeaturesByScenarioName(features, nameRegex)
		if err != nil {
			return err
		}
		features = f2
	}

	if len(features) == 0 {
		return fmt.Errorf("no .feature matched filters")
	}

	featureSet := map[string]struct{}{}
	for _, f := range features {
		featureSet[f] = struct{}{}
	}
	meta := report.SummaryMeta{
		Env:          envName,
		Tags:         tags,
		NameFilter:   nameRegex,
		FeatureCount: len(featureSet),
		Parallel:     parallel,
	}
	if len(includes) > 0 {
		meta.Includes = cloneStrings(includes)
	}
	if len(excludes) > 0 {
		meta.Excludes = cloneStrings(excludes)
	}

	csvPaths, htmlSpecs, unknownReports := classifyReports(reports)
	for _, unk := range unknownReports {
		fmt.Fprintf(os.Stderr, "unknown report kind %q\n", unk)
	}

	var (
		agg            *scenarioAggregator
		restoreSink    func()
		restoreDebug   func()
		restoreConsole func()
	)
	agg = newScenarioAggregator(csvPaths, htmlSpecs, meta, debug)
	if agg != nil {
		if agg.debugEnabled {
			restoreDebug = runner.SetDebugCapture(true)
			restoreConsole = runner.SetDebugConsole(true)
			defer restoreDebug()
			defer restoreConsole()
		}
		restoreSink = runner.SetScenarioSink(agg)
		defer restoreSink()
		defer func() {
			if flushErr := agg.Flush(); flushErr != nil {
				if err == nil {
					err = flushErr
				} else {
					fmt.Fprintln(os.Stderr, "failed to write reports:", flushErr)
				}
			}
		}()
	}
	if debug && agg == nil {
		restoreDebug = runner.SetDebugCapture(true)
		restoreConsole = runner.SetDebugConsole(true)
		defer restoreDebug()
		defer restoreConsole()
	}

	if parallel < 1 {
		parallel = 1
	}
	if parallel == 1 {
		opts := &godog.Options{
			Format: "pretty",
			Paths:  features,
			Tags:   tags,
		}
		suite := godog.TestSuite{
			Name:                "gherkio",
			ScenarioInitializer: runner.InitializeScenario(env, cat, flows),
			Options:             opts,
		}
		if status := suite.Run(); status != 0 {
			return fmt.Errorf("test failed with status %d", status)
		}
		return nil
	}

	shards := shard(features, parallel)
	errCh := make(chan error, len(shards))
	var wg sync.WaitGroup
	wg.Add(len(shards))
	for i, paths := range shards {
		i, paths := i, paths
		go func() {
			defer wg.Done()
			if len(paths) == 0 {
				return
			}
			opts := &godog.Options{
				Format: "pretty",
				Paths:  paths,
				Tags:   tags,
			}
			suite := godog.TestSuite{
				Name:                fmt.Sprintf("gherkio-%d", i+1),
				ScenarioInitializer: runner.InitializeScenario(env, cat, flows),
				Options:             opts,
			}
			if status := suite.Run(); status != 0 {
				errCh <- fmt.Errorf("shard %d failed (status %d)", i+1, status)
			}
		}()
	}
	wg.Wait()
	close(errCh)
	if len(errCh) > 0 {
		return <-errCh
	}
	return nil
}

// substring or glob include/exclude
func filterFeatures(list []string, pats []string, include bool) []string {
	out := make([]string, 0, len(list))
	for _, f := range list {
		matched := false
		for _, p := range pats {
			if strings.Contains(f, p) {
				matched = true
				break
			}
			if ok, _ := filepath.Match(p, filepath.Base(f)); ok {
				matched = true
				break
			}
			if ok, _ := filepath.Match(p, f); ok {
				matched = true
				break
			}
		}
		if (include && matched) || (!include && !matched) {
			out = append(out, f)
		}
	}
	return out
}

func filterFeaturesByScenarioName(files []string, rx string) ([]string, error) {
	re, err := regexp.Compile(rx)
	if err != nil {
		return nil, fmt.Errorf("invalid --name regex: %w", err)
	}
	keep := []string{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		lines := strings.Split(string(b), "\n")
		ok := false
		for _, ln := range lines {
			s := strings.TrimSpace(ln)
			if strings.HasPrefix(s, "Scenario:") || strings.HasPrefix(s, "Scenario Outline:") {
				name := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(s, "Scenario:"), "Scenario Outline:"))
				if re.MatchString(name) {
					ok = true
					break
				}
			}
		}
		if ok {
			keep = append(keep, f)
		}
	}
	return keep, nil
}

func shard(paths []string, n int) [][]string {
	if n <= 1 || len(paths) <= 1 {
		return [][]string{paths}
	}
	out := make([][]string, n)
	for i, p := range paths {
		out[i%n] = append(out[i%n], p)
	}
	return out
}

func classifyReports(specs []string) (csvPaths []string, htmlSpecs []htmlSpec, unknown []string) {
	for _, spec := range specs {
		if strings.TrimSpace(spec) == "" {
			continue
		}
		kind, path := splitKindPath(spec)
		lower := strings.ToLower(kind)
		switch lower {
		case "", "pretty":
			continue
		case "csv":
			csvPaths = append(csvPaths, pathOrDefault(path, "reports/run.csv"))
		default:
			if dbg, ok := parseHTMLKind(lower); ok {
				normPath, fromPath := normalizeHTMLPath(path, "reports/run.html")
				htmlSpecs = append(htmlSpecs, htmlSpec{Path: normPath, Debug: dbg || fromPath})
				continue
			}
			unknown = append(unknown, kind)
		}
	}
	return
}

type scenarioAggregator struct {
	csv          *report.CSV
	csvPaths     []string
	htmls        []*report.HTML
	debugEnabled bool
	mu           sync.Mutex
	firstErr     error
}

func newScenarioAggregator(csvPaths []string, htmls []htmlSpec, meta report.SummaryMeta, debugFlag bool) *scenarioAggregator {
	if len(csvPaths) == 0 && len(htmls) == 0 {
		return nil
	}
	inst := &scenarioAggregator{
		csv:          report.NewCSV(),
		csvPaths:     csvPaths,
		debugEnabled: debugFlag,
	}
	inst.htmls = make([]*report.HTML, 0, len(htmls))
	for _, spec := range htmls {
		h := report.NewHTML(spec.Path, spec.Debug || debugFlag, meta)
		inst.htmls = append(inst.htmls, h)
		if spec.Debug || debugFlag {
			inst.debugEnabled = true
		}
	}
	return inst
}

func (a *scenarioAggregator) RecordScenario(feature, scenario, status string, durMs int64, steps []runner.StepLog) {
	if a == nil {
		return
	}
	var stepDetails []report.StepDetail
	if len(a.htmls) > 0 {
		stepDetails = make([]report.StepDetail, len(steps))
		for i, st := range steps {
			detail := report.StepDetail{
				Text:       st.Text,
				Status:     st.Status,
				DurationMs: st.DurationMs,
				Error:      st.Error,
			}
			if st.Debug != nil {
				detail.Debug = &report.DebugInfo{
					RequestBody:    st.Debug.RequestBody,
					ResponseBody:   st.Debug.ResponseBody,
					ResponseStatus: st.Debug.ResponseStatus,
				}
			}
			stepDetails[i] = detail
		}
	}
	res := report.Result{Feature: feature, Scenario: scenario, Status: status, DurationMs: durMs, Steps: stepDetails}
	for _, h := range a.htmls {
		h.Add(res)
	}
	if len(a.csvPaths) == 0 {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.firstErr != nil {
		return
	}
	for _, path := range a.csvPaths {
		if err := a.csv.Append(path, feature, scenario, status, durMs); err != nil {
			a.firstErr = err
			break
		}
	}
}

func (a *scenarioAggregator) Flush() error {
	if a == nil {
		return nil
	}
	var flushErr error
	for _, h := range a.htmls {
		if err := h.Flush(); err != nil && flushErr == nil {
			flushErr = err
		}
	}
	a.mu.Lock()
	first := a.firstErr
	a.mu.Unlock()
	if first != nil {
		return first
	}
	return flushErr
}

func cloneStrings(src []string) []string {
	if len(src) == 0 {
		return nil
	}
	out := make([]string, len(src))
	copy(out, src)
	return out
}