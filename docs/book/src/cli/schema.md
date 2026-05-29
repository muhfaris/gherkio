# gherkio schema

Generate JSON Schema for Gherkio YAML files

### Synopsis

Generates a JSON Schema for Gherkio YAML files.
This schema can be used in editors (like VSCode, Neovim) for autocomplete and validation.

Usage:
  gherkio schema                              # Generate all schemas (default)
  gherkio schema --type test                 # Generate only test file schema
  gherkio schema --type config               # Generate only config schema
  gherkio schema --type environment         # Generate only environment schema
  gherkio schema --type credentials         # Generate only credentials schema
  gherkio schema --type schema-definition   # Generate only schema definition schema
  gherkio schema --list                     # List available schema types

Output:
  By default, outputs all schemas in a single JSON file keyed by type.
  Use --type to output a specific schema only.

```
gherkio schema [flags]
```

### Options

```
  -h, --help          help for schema
      --list          List available schema types
      --type string   Schema type: test, config, environment, credentials, schema-definition
```

### SEE ALSO

* [gherkio](gherkio.md)	 - Gherkio is a testing and validation framework

