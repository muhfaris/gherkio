# gherkio run

Execute a test scenario

### Synopsis

Executes a Gherkio test YAML file and displays the results.

If no test file is provided, all tests in .gherkio/tests/ are executed.
If a directory is provided, all tests in that directory are executed.

The test file path is resolved in the following order:
1. As provided (relative to current working directory)
2. Relative to .gherkio/tests/ directory

Example:
```bash
  gherkio run                    # Run all tests
  gherkio run tests/login.yaml
  gherkio run restful-api/       # Run all tests in restful-api/ directory
  gherkio run login.yaml --env staging
  gherkio run login.yaml --verbose
  gherkio run login.yaml --report html
  gherkio run login.yaml --env staging --account alpha
  gherkio run login.yaml --env staging --all-accounts
  gherkio run login.yaml --section steps        # Run all steps in 'steps' section only
  gherkio run login.yaml --section setup        # Run all setup steps only
  gherkio run login.yaml --section teardown     # Run all teardown steps only

```
```
gherkio run [test-file] [flags]
```

### Options

```
      --account string   Account name from credentials file (e.g. alpha, beta)
      --all-accounts     Run tests against all accounts in the credentials file
      --dry-run          Preview test execution without making HTTP requests
  -e, --env string       Environment to use (e.g. local, staging, production) (default "local")
  -h, --help             help for run
      --line int         Line number containing the step to run (default -1)
  -p, --parallel int     Number of tests to run in parallel (0 = auto-detect CPU count)
      --report string    Generate a report (format: html, json, or html,json)
      --report-raw       Skip sensitive data masking in JSON reports (cURL commands remain masked)
      --section string   Section to run (setup, steps, teardown). When used without --step or --line, runs ALL steps in that section only.
      --step int         Index of the step to run (0-indexed) (default -1)
  -t, --tag strings      Filter tests by tags (AND logic: test must have ALL specified tags)
  -v, --verbose          Show full request/response payloads
```

### SEE ALSO

* [gherkio](gherkio.md)	 - Gherkio is a testing and validation framework

