# RFC-01: Scenario Composition

**Status:** Implemented

Allow a Gherkio scenario to import another scenario as a step via `use:`. This enables reusable authentication flows, shared setup/teardown, and test composition without duplicating steps.

**Key decisions:**
- `use:` resolves relative to the importing file's directory, then `.gherkio/tests/`
- Variables saved in a used scenario merge into the parent context
- Circular references are detected and rejected
- Max depth limit of 5 prevents infinite recursion
