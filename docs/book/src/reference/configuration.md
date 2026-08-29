# Global Configuration (`config.yaml`)

The `config.yaml` file located in `.gherkio/config.yaml` defines project metadata, execution targets, schema paths, auto-masking configurations, security sandboxing, and reporting defaults.

---

## 📝 Complete Configuration Reference (`.gherkio/config.yaml`)

Below is the exhaustive, production-ready `config.yaml` reference containing all available configuration properties and inline documentation:

```yaml
gherkio_version: 0.1.0

# ----------------------------------------------------------------------
# 1. Project Metadata
# ----------------------------------------------------------------------
project:
  name: "billing-system"
  version: "1.0.0"

# ----------------------------------------------------------------------
# 2. Environments Directory Configuration
# ----------------------------------------------------------------------
environments:
  default: local                    # Sourced if --env flag is omitted
  path: .gherkio/environments       # Path to environment configurations

# ----------------------------------------------------------------------
# 3. Tests & Scenarios Configuration
# ----------------------------------------------------------------------
tests:
  path: .gherkio/tests              # Directory where test scenarios reside

# ----------------------------------------------------------------------
# 4. Validation Schemas Configuration
# ----------------------------------------------------------------------
schemas:
  path: .gherkio/schemas            # Directory containing JSON/YAML validation schemas

# ----------------------------------------------------------------------
# 5. Multipart File Assets Directory
# ----------------------------------------------------------------------
assets:
  path: assets                      # Default workspace directory for multipart file uploads

# ----------------------------------------------------------------------
# 6. Security, Masking & Outbound Sandboxing
# ----------------------------------------------------------------------
security:
  mask:
    enabled: true                   # Enable dynamic log sanitization
    fields:                         # Case-insensitive field keys to mask in output/reports
      - token
      - password
      - secret
      - Authorization
      - creditCard
  sandboxing:
    enabled: true                   # Enforce outbound network security boundaries
    allowedDomains:                 # Allowed target domains (supports wildcards)
      - "*.company.com"
      - "api.stripe.com"
    blockedDomains:                 # Explicitly blocked target domains
      - "analytics.untrusted.com"
    blockPrivateSubnets: true       # Prevent loopback (127.0.0.1) & private IP SSRF attacks

# ----------------------------------------------------------------------
# 7. Reports & Failure Snapshot Dumps
# ----------------------------------------------------------------------
reports:
  path: .gherkio/reports            # Output path for test run summaries
  format: html                      # Report format ("html", "json", or "html,json")
  archive: true                     # Keep a history of past reports
  retention: 10                     # Keep up to 10 historical reports
  maskSensitive: true               # Mask credentials in HTML report pages
  failures:
    enabled: true                   # Write detailed JSON debug snapshots on test failures
    path: .gherkio/reports/failures # Directory for failure snapshots
    maskSensitive: true             # Mask credentials in failure dumps
    retainCount: 50                 # Retain up to 50 failure snapshots

# ----------------------------------------------------------------------
# 8. Authentication Defaults (Optional)
# ----------------------------------------------------------------------
jwt_token_path: "body.token"        # JSON path for auto-extracting JWT tokens
```

---

## ⚙️ Detailed Configuration Section Reference

### 1. `project`
Defines the name and version of your API testing suite. Used when generating test run reports and HTML dashboards.

### 2. `environments`
* `default`: The default target environment name (e.g. `local`) if the CLI `--env` flag is omitted.
* `path`: Relative path pointing to target environment host definition files (e.g. `.gherkio/environments/`).

### 3. `tests`, `schemas`, & `assets`
* `tests.path`: Directory where `.yaml` scenario suites are stored.
* `schemas.path`: Directory where reusable JSON/YAML schema validation files are stored.
* `assets.path`: Default folder for storing multipart file attachments (images, PDFs, documents).

### 4. `security.mask` (Credential Redaction)
Scans all stdout console logs, raw HTTP tracebacks, and generated report files. If any field matches `security.mask.fields`, Gherkio automatically replaces its value with `[MASKED]`.

### 5. `security.sandboxing` (Network Protection Engine)
Prevents SSRF and unauthorized network egress:
* `allowedDomains`: List of target domain patterns permitted for HTTP requests.
* `blockedDomains`: List of explicitly forbidden target domains.
* `blockPrivateSubnets`: Resolves domain IPs prior to HTTP calls. Rejects requests targeting private or loopback subnets (`127.0.0.1`, `10.0.0.0/8`, `192.168.0.0/16`).

### 6. `reports` & `reports.failures`
Controls test result artifact generation:
* `format`: Report output format (`html` or `json`).
* `archive` & `retention`: Maintains historical report versions up to `retention` runs.
* `failures`: Creates diagnostic JSON snapshots in `.gherkio/reports/failures/` whenever an assertion fails, preserving full request/response context for debugging.
