# RFC-11: Negative Assertions

**Status:** Implemented

Added the ability to assert that something does NOT exist or does NOT match.

**Key decisions:**
- `not exists` — field must be completely absent from the response
- `schema: not <name>` — response must NOT match the given schema
- Consistent with Gherkio's declarative assertion style
