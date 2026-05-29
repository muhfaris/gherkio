# RFC-07: Per-Request Timeout

**Status:** Implemented

Added configurable per-request HTTP timeout, separate from the global default of 30 seconds.

**Key decisions:**
- `timeout` field on the request block (e.g. `timeout: 60s`)
- Overrides the default 30-second client timeout
- Distinct from `timing.max` (which is an assertion, not a client deadline)
