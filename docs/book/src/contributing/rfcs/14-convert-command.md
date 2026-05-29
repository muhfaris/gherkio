# RFC-14: Convert Command

**Status:** Implemented

Added bidirectional conversion between cURL commands and Gherkio YAML DSL.

**Key decisions:**
- `gherkio convert` — converts cURL to YAML
- `gherkio convert -r` — converts YAML to cURL
- Supports environment variable interpolation and credential injection
- Shell tokenizer handles complex cURL arguments
