# RFC-05: Request Retry

**Status:** Implemented

Added polling and retry support for testing eventual consistency scenarios.

**Key decisions:**
- Configurable `attempts`, `interval`, `backoff` (constant/linear/exponential)
- `maxDuration` boundary to prevent infinite polling
- `onStatus` filter to only retry on specific status codes
- Retry history recorded in step results and reports
