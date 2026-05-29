# RFC-15: Schema Completeness

**Status:** Implemented

Completed the JSON Schema generator with support for all Gherkio document types.

**Key decisions:**
- `--type` flag to generate specific schemas (test, config, environment, credentials, schema-definition)
- `--list` flag to enumerate available types
- `jsonschema` tags added to all model structs
- Generated schemas provide IDE autocomplete for YAML files
