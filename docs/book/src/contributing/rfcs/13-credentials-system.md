# RFC-13: Credentials System (Initial)

**Status:** Implemented

The initial credentials system design, providing the foundation for the multi-account credentials feature.

**Key decisions:**
- Account model with `username`, `password`, `role`, and `Extra` (inline) fields
- Credentials are environment-specific (`local.yaml`, `staging.yaml`)
- Passwords and sensitive fields are auto-flagged for masking
