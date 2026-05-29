# gherkio validate

Validate a test file without executing HTTP requests

### Synopsis

Performs static analysis on Gherkio test YAML files without making HTTP calls.

Validates:
- YAML syntax and structure
- Scenario structure (required fields)
- Request method validity
- Variable references ($var, $accounts.*, built-in generators)
- Account references from credentials
- Schema references from .gherkio/schemas/
- use file existence
- Retry configuration

Example:
```bash
  gherkio validate                           # Validate all test files
  gherkio validate login.yaml                # Validate single file
  gherkio validate --verbose                 # Show detailed results
  gherkio validate --env staging             # Validate with staging credentials

```
```
gherkio validate [test-file] [flags]
```

### Options

```
  -e, --env string   Environment for credentials validation (default "local")
  -h, --help         help for validate
  -v, --verbose      Show detailed validation results
```

### SEE ALSO

* [gherkio](gherkio.md)	 - Gherkio is a testing and validation framework

