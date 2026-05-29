# RFC-04: HTML & JSON Reporting

**Status:** Implemented

Added standalone HTML and JSON report generation for test runs, with masked sensitive data and copyable cURL commands.

**Key decisions:**
- Reports saved to `.gherkio/reports/latest/` and timestamped directories
- HTML reports include masked payloads, cURL commands, and Request IDs
- JSON reports support `--report-raw` for unmasked data
- Multi-scenario grouping when running test directories
