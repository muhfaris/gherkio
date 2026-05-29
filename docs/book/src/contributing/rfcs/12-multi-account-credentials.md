# RFC-12: Multi-Account Credentials

**Status:** Implemented

Added a credentials system supporting multiple named accounts per environment, with `$accounts.<name>.<field>` variable access.

**Key decisions:**
- Credentials stored in `.gherkio/credentials/<env>.yaml`
- `--account` and `--all-accounts` CLI flags
- Auto-use single account when only one exists
- `$accounts.<name>.<field>` syntax for cross-account access
- Sensitive fields auto-masked in output
