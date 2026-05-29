# Global Configuration (`config.yaml`)

The `config.yaml` file located in `.gherkio/config.yaml` defines project metadata, execution targets, schema paths, auto-masking configurations, and reporting defaults.

---

## 📝 Complete Configuration Schema

Here is a fully documented, production-ready example of the `config.yaml` layout:

```yaml
# Unique identifier of Gherkio project
project:
  name: billing-system
  version: "1.0.0"

# Target environment mapping rules
environments:
  default: local                    # Sourced if --env flag is omitted

# Core directory conventions
tests:
  path: tests                       # Location of test scripts under .gherkio/

schemas:
  path: schemas                     # Location of validation schemas under .gherkio/

# Log security and sensitive field masking
security:
  mask:
    enabled: true                   # Enable dynamic log sanitization
    fields:
      - token
      - password
      - secret
      - creditCard
      - customSecret

# Automation run reports
reports:
  format: html                      # 'html' or 'json'
  directory: reports                # Directory to compile results under .gherkio/
```

---

## ⚙️ Key Configuration Details

### 1. `security.mask` (Sensitive Fields)
To guarantee security compliance during test suite executions (such as in public CI runner logs), Gherkio scans all request and response payloads. 

If any field name matches the lists under `security.mask.fields`, Gherkio automatically replaces its value with `[MASKED]` in stdout printing, raw tracebacks, and compiled reports.

### 2. `reports` (Test Artifact Compilation)
Every `gherkio run` execution outputs a clean JSON report by default. If `format: html` is selected, Gherkio compiles a responsive, dark-mode single-page HTML report summarizing performance metrics, request-response payloads, and assertions.
