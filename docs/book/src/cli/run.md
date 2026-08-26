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
  gherkio run restful-api/ --request-delay 1000 # Wait 1000ms before every request
  gherkio run restful-api/ --request-delay 1s   # Explicit duration syntax also works

```
```
gherkio run [test-file] [flags]
```

By default, directory/all-tests runs wait `100ms` before each request and single-file runs wait
`50ms`. `--request-delay` overrides that default, including `--request-delay 0` to disable it.
The delay applies before every request attempt, including retries and requests in composed
scenarios. Bare numbers use milliseconds. With `--parallel`, each worker applies its own delay
independently; omit `--parallel` when requests must remain globally sequential.

### Options

```
      --account string   Account name from credentials file (e.g. alpha, beta)
      --all-accounts     Run tests against all accounts in the credentials file
      --dry-run          Preview test execution without making HTTP requests
  -e, --env string       Environment to use (e.g. local, staging, production) (default "local")
  -h, --help             help for run
      --line int         Line number containing the step to run (default -1)
  -p, --parallel int     Number of tests to run in parallel (0 = auto-detect CPU count)
      --request-delay string   Wait before each HTTP request (defaults: 50ms/file, 100ms/directory; bare numbers are milliseconds)
      --report string    Generate a report (format: html, json, or html,json)
      --report-raw       Skip sensitive data masking in JSON reports (cURL commands remain masked)
      --section string   Section to run (setup, steps, teardown). When used without --step or --line, runs ALL steps in that section only.
      --step int         Index of the step to run (0-indexed) (default -1)
  -t, --tag strings      Filter tests by tags (AND logic: test must have ALL specified tags)
  -u, --until string     Execute steps until a specific target, e.g. 'steps:1' or '2'
  -v, --verbose          Show full request/response payloads
```

### SEE ALSO

* [gherkio](gherkio.md)	 - Gherkio is a testing and validation framework
