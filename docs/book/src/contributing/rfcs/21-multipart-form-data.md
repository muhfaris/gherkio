# RFC-21: Multipart Form Data

**Status:** Proposed

Add native support for `multipart/form-data` payloads, enabling file upload testing.

**Key decisions:**
- New `multipart:` block on request with `fields:` and `files:` sections
- File paths resolved relative to project root
- Automatic `Content-Type` header with boundary
- Static validation for file existence before execution
