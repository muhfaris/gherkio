package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/muhfaris/gherkio/internal/loader"
	"github.com/muhfaris/gherkio/internal/runner"

	"github.com/cucumber/godog"
)

func runRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	var envName, tags, nameRegex string
	var parallel int
	includes := multiString(fs, "feature")         // repeatable include filter
	excludes := multiString(fs, "exclude-feature") // repeatable exclude filter

	_ = multiString(fs, "report")
	fs.StringVar(&envName, "env", "", "environment name")
	fs.StringVar(&tags, "tags", "", "tag expression (e.g. \"@smoke and not @wip\")")
	fs.StringVar(&nameRegex, "name", "", "filter Scenario name by regex (best-effort)")
	fs.IntVar(&parallel, "parallel", 1, "number of parallel workers (by feature file)")

	// parallel, name, feature filters: placeholders for MVP
	if err := fs.Parse(args); err != nil {
		return err
	}
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
		fmt.Printf("Loaded %d flow(s) from gherkio/flows\\n", len(flows))
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
	if len(*includes) > 0 {
		features = filterFeatures(features, *includes, true)
	}

	if len(*excludes) > 0 {
		features = filterFeatures(features, *excludes, false)
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

	opts := &godog.Options{
		Format: "pretty",          // console output style
		Paths:  []string{featDir}, // where your .feature files live
		Tags:   tags,              // --tags filter from CLI
	}
	suite := godog.TestSuite{
		Name:                "gherkio", // suite name (for logs/ID)
		ScenarioInitializer: runner.InitializeScenario(env, cat, flows),
		Options:             opts,
	}

	status := suite.Run()
	if status != 0 {
		return fmt.Errorf("test failed with status %d", status)
	}

	if parallel < 1 {
		parallel = 1
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
		// return first error
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
