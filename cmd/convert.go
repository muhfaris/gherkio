package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/muhfaris/gherkio/internal/converter"
	"github.com/muhfaris/gherkio/internal/core/project"
	"github.com/muhfaris/gherkio/internal/model"
	"github.com/muhfaris/gherkio/internal/runner"
	"github.com/spf13/cobra"
)

var (
	inputFile    string
	scenarioName string
	stepOnly     bool
	outputFile   string
	reverse      bool
	stepIndex    int
)

// convertCmd represents the gherkio convert command.
var convertCmd = &cobra.Command{
	Use:   "convert [command/test-file]",
	Short: "Convert cURL to Gherkio YAML or Gherkio YAML to cURL",
	Long: `Bidirectionally converts between standard cURL commands and Gherkio DSL YAML.

cURL to YAML (Default):
  Provide the cURL command as an argument, pipe it via stdin, or read from a file using --file.
  Automatically resolves environment base URLs and converts JSON bodies into native YAML.

YAML to cURL (Reverse):
  Use the --reverse (-r) flag and provide the Gherkio test YAML file path.
  Optionally extract a specific step with --step.

Examples:
  # Convert cURL from argument
  gherkio convert "curl -X POST https://api.com/login -d '{\"user\":\"x\"}'"
  
  # Convert cURL from file to a YAML file
  gherkio convert --file curl.txt --output tests/login.yaml --scenario "Login Flow"

  # Convert YAML step back to cURL
  gherkio convert --reverse tests/login.yaml --step 0 --env staging
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Find project root if possible (optional for simple conversions)
		projectDir, _ := project.FindRoot(cwd)

		if reverse {
			// YAML -> cURL
			if len(args) == 0 {
				return fmt.Errorf("please provide the Gherkio test YAML file path")
			}
			return runReverseConvert(args[0], projectDir, envName, accountName, stepIndex, outputFile)
		}

		// cURL -> YAML
		var curlCmd string
		if len(args) > 0 {
			curlCmd = args[0]
		} else if inputFile != "" {
			data, err := os.ReadFile(inputFile)
			if err != nil {
				return fmt.Errorf("failed to read input file: %w", err)
			}
			curlCmd = string(data)
		} else {
			// Read from stdin (check if pipe/redirect)
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				bytes, err := io.ReadAll(os.Stdin)
				if err == nil {
					curlCmd = string(bytes)
				}
			}
		}

		curlCmd = strings.TrimSpace(curlCmd)
		if curlCmd == "" {
			return fmt.Errorf("please provide a cURL command argument, pipe it via stdin, or use --file flag")
		}

		return runForwardConvert(curlCmd, projectDir, envName, scenarioName, stepOnly, outputFile)
	},
}

func init() {
	rootCmd.AddCommand(convertCmd)
	convertCmd.Flags().StringVarP(&inputFile, "file", "f", "", "Read cURL command from a file")
	convertCmd.Flags().StringVarP(&scenarioName, "scenario", "s", "untitled", "Custom scenario name for the generated YAML")
	convertCmd.Flags().BoolVar(&stepOnly, "step-only", false, "Output just the step block without scenario wrapper")
	convertCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write output directly to a file")
	convertCmd.Flags().BoolVarP(&reverse, "reverse", "r", false, "Convert Gherkio step/YAML to cURL command(s)")
	convertCmd.Flags().IntVar(&stepIndex, "step", -1, "Index of the step to convert in reverse mode (0-indexed)")
	convertCmd.Flags().StringVarP(&envName, "env", "e", "local", "Environment to resolve base URLs (e.g. local, staging)")
	convertCmd.Flags().StringVar(&accountName, "account", "", "Account name from credentials file to interpolate variables")
}

func runForwardConvert(curlCmd string, projectDir string, env string, scenName string, stepOnly bool, outPath string) error {
	req, warnings, err := converter.ParseCurl(curlCmd)
	if err != nil {
		return err
	}

	// Print ignored flags warnings
	if len(warnings) > 0 {
		for _, w := range warnings {
			fmt.Fprintf(os.Stderr, "⚠ %s\n", w)
		}
		fmt.Fprintln(os.Stderr)
	}

	yamlStr, err := converter.FormatYAML(req, scenName, stepOnly, projectDir, env)
	if err != nil {
		return err
	}

	if outPath != "" {
		err := os.WriteFile(outPath, []byte(yamlStr), 0644)
		if err != nil {
			return fmt.Errorf("failed to write output to %s: %w", outPath, err)
		}
		fmt.Printf("✓ Converted Gherkio YAML saved to: %s\n", outPath)
	} else {
		fmt.Print(yamlStr)
	}

	return nil
}

func runReverseConvert(testPath string, projectDir string, env string, accountName string, stepIdx int, outPath string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	fullPath, err := resolveTestPath(cwd, projectDir, testPath)
	if err != nil {
		return err
	}

	testFile, err := runner.LoadTestFile(fullPath)
	if err != nil {
		return err
	}

	// Load environment and credentials if in a project
	var creds *model.Credentials
	if projectDir != "" {
		creds, _ = runner.LoadCredentials(projectDir, env)
	}

	// Determine credential vars and all accounts for $accounts.<name>.<field> access
	var credentialVars map[string]interface{}
	allAccountsMap := make(map[string]interface{})
	if creds != nil {
		allAccountsMap = creds.ToMap()
		if accountName != "" {
			if acc, exists := creds.GetAccount(accountName); exists {
				credentialVars = runner.CredentialsToVars(acc)
			}
		} else {
			// Fallback to first/auto account if only one exists
			names := creds.AccountNames()
			if len(names) == 1 {
				if acc, exists := creds.GetAccount(names[0]); exists {
					credentialVars = runner.CredentialsToVars(acc)
				}
			}
		}
	}

	// Merge all accounts into credential vars for $accounts.<name>.<field> access
	if len(allAccountsMap) > 0 {
		if credentialVars == nil {
			credentialVars = make(map[string]interface{})
		}
		credentialVars["accounts"] = allAccountsMap
	}

	// Determine which steps to convert
	var targetSteps []model.Step
	if stepIdx >= 0 {
		if stepIdx >= len(testFile.Steps) {
			return fmt.Errorf("step index %d out of bounds (contains %d steps)", stepIdx, len(testFile.Steps))
		}
		targetSteps = []model.Step{testFile.Steps[stepIdx]}
	} else {
		targetSteps = testFile.Steps
	}

	var curls []string
	for _, step := range targetSteps {
		// Inject fresh built-in generator variables per step so each cURL command
		// gets unique $uuid, $ulid, $randomInt, $randomEmail, $randomPhone values.
		stepVars := make(map[string]interface{})
		for key, val := range runner.LoadGherkioEnvVars() {
			stepVars[key] = val
		}
		for k, v := range credentialVars {
			stepVars[k] = v
		}
		for key, val := range runner.BuiltinVars() {
			stepVars[key] = val
		}

		if step.Use != "" {
			curls = append(curls, fmt.Sprintf("# use: %s (skipped composition)", step.Use))
			continue
		}

		c, err := converter.ConvertStepToCurl(step.Request, projectDir, env, stepVars)
		if err != nil {
			return fmt.Errorf("failed to convert step to cURL: %w", err)
		}
		curls = append(curls, c)
	}

	outStr := strings.Join(curls, "\n")
	if outPath != "" {
		err := os.WriteFile(outPath, []byte(outStr+"\n"), 0644)
		if err != nil {
			return fmt.Errorf("failed to write output to %s: %w", outPath, err)
		}
		fmt.Printf("✓ Converted cURL command(s) saved to: %s\n", outPath)
	} else {
		fmt.Println(outStr)
	}

	return nil
}
