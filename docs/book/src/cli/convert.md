# gherkio convert

Convert cURL to Gherkio YAML or Gherkio YAML to cURL

### Synopsis

Bidirectionally converts between standard cURL commands and Gherkio DSL YAML.

### cURL to YAML (Default)
Provide the cURL command as an argument, pipe it via stdin, or read from a file using --file.
Automatically resolves environment base URLs and converts JSON bodies into native YAML.

### YAML to cURL (Reverse)
Use the --reverse (-r) flag and provide the Gherkio test YAML file path.
Optionally extract a specific step with --step.

Examples:
```bash
  # Convert cURL from argument
  gherkio convert "curl -X POST https://api.com/login -d '{\"user\":\"x\"}'"
  
  # Convert cURL from file to a YAML file
  gherkio convert --file curl.txt --output tests/login.yaml --scenario "Login Flow"

  # Convert YAML step back to cURL
  gherkio convert --reverse tests/login.yaml --step 0 --env staging


```
```
gherkio convert [command/test-file] [flags]
```

### Options

```
      --account string    Account name from credentials file to interpolate variables
  -e, --env string        Environment to resolve base URLs (e.g. local, staging) (default "local")
  -f, --file string       Read cURL command from a file
  -h, --help              help for convert
  -o, --output string     Write output directly to a file
  -r, --reverse           Convert Gherkio step/YAML to cURL command(s)
  -s, --scenario string   Custom scenario name for the generated YAML (default "untitled")
      --step int          Index of the step to convert in reverse mode (0-indexed) (default -1)
      --step-only         Output just the step block without scenario wrapper
```

### SEE ALSO

* [gherkio](gherkio.md)	 - Gherkio is a testing and validation framework

