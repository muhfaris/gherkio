package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/muhfaris/gherkio/internal/loader"
	"github.com/muhfaris/gherkio/internal/runner"
)

func runSteps(args []string) error {
	var format, out string
	fs := flag.NewFlagSet("steps", flag.ContinueOnError)
	fs.StringVar(&format, "format", "text", "text|md")
	fs.StringVar(&out, "out", "", "output file (optional)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	initScenario := runner.InitializeScenario(loader.Env{}, loader.Catalog{}, map[string]loader.Flow{})
	initScenario(nil) // bind steps into catalog without a Godog context
	cat := runner.GetStepCatalog()
	var data string
	if format == "md" {
		data = cat.Markdown()
	} else {
		// simple text
		data = "Available steps:\n"
		for _, it := range cat.List() {
			data += fmt.Sprintf("- %s\n  desc: %s\n  ex:   %s\n", it.Pattern, it.Desc, it.Example)
		}
	}

	if out == "" {
		fmt.Print(data)
		return nil
	}
	return os.WriteFile(out, []byte(data), 0o644)
}
