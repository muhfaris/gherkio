package cli

import (
	"errors"
	"flag"
	"fmt"
)

// Execute is the entry for CLI.
func Execute(args []string) error {
	if len(args) == 0 {
		printHelp()
		return nil
	}
	sub := args[0]
	switch sub {
	case "help", "-h", "--help":
		printHelp()
		return nil
	case "init":
		return runInit(args[1:])
	case "call":
		return runCall(args[1:])
	case "run":
		return runRun(args[1:])
	case "steps":
		return runSteps(args[1:])
	case "import":
		return runImport(args[1:])
	default:
		return fmt.Errorf("unknown subcommand: %s", sub)
	}
}

func printHelp() {
	fmt.Println(`🥒 gherkio – declarative API testing & journey runner (MVP)

Usage:
  gherkio init                         Initialize gherkio/ structure with examples
  gherkio call  [flags]                Single-endpoint call using catalogs
  gherkio run   [flags]                Run Gherkin features (journey)
  gherkio steps [flags]                Run Gherkin steps
  gherkio import curl [flags]          Import a curl command into catalog/fixture/feature

Flags (call):
  --env <name>                         Environment name (must exist in gherkio/envs/<name>.yaml)
  --api <key>                          Endpoint key from gherkio/apis/*.yaml (e.g., users.getById)
  --path k=v [--path k=v]              Path params
  --query k=v [--query k=v]            Query params
  --header k=v [--header k=v]          Headers
  --body @file.json | '{json}'         Request body
  --expect-status N                    Expected HTTP status code
  --report kind[:path]                 Reporters (pretty|html|junit|cucumber|csv)

Flags (run):
  --env <name>
  --tags "<expr>"                     (placeholder)
  --parallel N                         (placeholder)
  --report kind[:path]

Flags (import curl):
  --api <key>                          API key to generate (required)
  --curl '<cmd>'                       Raw curl command to import (required)
  --catalog path                       Catalog file to update (default gherkio/apis/imported.yaml)
  --fixture path                       Fixture file path (optional)
  --feature path                       Feature file path (optional)
  --title "Scenario title"            Scenario title override
  --name "Feature name"              Feature name override
`)
}

// helper to consume repeated flags like --path k=v --path a=b
func collectKV(fs *flag.FlagSet, name string) *mapList {
	l := &mapList{}
	fs.Var(l, name, "collect k=v pairs")
	return l
}

type mapList struct{ pairs [][2]string }

func (m *mapList) String() string { return fmt.Sprint(m.pairs) }
func (m *mapList) Set(s string) error {
	var k, v string
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			k = s[:i]
			v = s[i+1:]
			break
		}
	}
	if k == "" {
		return errors.New("format must be k=v")
	}
	m.pairs = append(m.pairs, [2]string{k, v})
	return nil
}

func (m *mapList) ToMap() map[string]string {
	out := map[string]string{}
	for _, kv := range m.pairs {
		out[kv[0]] = kv[1]
	}
	return out
}
