# RFC-03: Schema Assertions

**Status:** Implemented

Introduced `schema:` as a special assertion key that validates the full response body against a YAML schema definition stored in `.gherkio/schemas/`.

**Key decisions:**
- Schemas are defined as YAML files, not JSON Schema
- Support for `schema: not <name>` negative assertions
- Schema validation checks types, required fields, enums, and format constraints
