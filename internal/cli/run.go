package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/muhfaris/gherkio/internal/loader"
	"github.com/muhfaris/gherkio/internal/runner"

	"github.com/cucumber/godog"
)

func runRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	var envName, tags string
	_ = multiString(fs, "report")
	fs.StringVar(&envName, "env", "", "environment name")
	fs.StringVar(&tags, "tags", "", "tag expression (e.g. \"@smoke and not @wip\")")
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

	return nil
}
