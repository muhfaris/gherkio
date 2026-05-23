# RFC-8: Configuration Alignment — schemas path

> **Status:** Implemented
> **Author:** Faris
> **Date:** May 22, 2026

---

## 1. Summary

Add the missing `schemas` path field to the `Config` model so that the config YAML and the Go struct are consistent, and the schema loader respects the configured path instead of using a hardcoded one.

---

## 2. Motivation

When running `gherkio init`, the generated `.gherkio/config.yaml` includes:

```yaml
schemas:
  path: .gherkio/schemas
```

However, the Go `Config` struct in `internal/model/config.go` has no `Schemas` field — it's silently ignored during parsing. Meanwhile, `LoadSchema()` in `internal/runner/schema.go` hardcodes the path:

```go
schemasDir := filepath.Join(projectDir, ".gherkio", "schemas")
```

This misalignment means:

- The config file documents a setting that the code doesn't use (misleading)
- If a user changes `schemas.path` in config.yaml, nothing happens — their schemas still resolve against the hardcoded path
- Schema path is not configurable, violating the pattern set by `tests.path` and `reports.path`

---

## 3. Design

### 3.1 Model Change

Add a `Schemas` field to `Config` in `internal/model/config.go`:

```go
type Config struct {
	Project      ProjectConfig  `yaml:"project,omitempty"`
	Environments EnvConfig      `yaml:"environments,omitempty"`
	Tests        TestsConfig    `yaml:"tests,omitempty"`
	Schemas      SchemasConfig  `yaml:"schemas,omitempty"`  // ← new
	Security     SecurityConfig `yaml:"security,omitempty"`
	Reports      ReportsConfig  `yaml:"reports,omitempty"`
}

type SchemasConfig struct {
	Path string `yaml:"path,omitempty"`
}
```

### 3.2 Schema Loader Change

Update `LoadSchema()` in `internal/runner/schema.go` to accept a configurable schemas directory:

```go
func LoadSchema(name string, projectDir string) (*model.Schema, error) {
    // Default path
    schemasDir := filepath.Join(projectDir, ".gherkio", "schemas")

    // Override from config if available
    cfg, err := LoadConfig(projectDir)
    if err == nil && cfg.Schemas.Path != "" {
        if filepath.IsAbs(cfg.Schemas.Path) {
            schemasDir = cfg.Schemas.Path
        } else {
            schemasDir = filepath.Join(projectDir, cfg.Schemas.Path)
        }
    }
    // ... rest of function unchanged
}
```

Alternatively, have `LoadSchema` receive the schemas directory as a parameter and resolve it upstream in the runner.

### 3.3 Runner Integration

The runner already calls `LoadSchema()` during assertion evaluation. The `RunConfig` could optionally carry the resolved schemas directory, but the simplest approach is to resolve it inside `LoadSchema()` via `LoadConfig()` as shown above.

---

## 4. Edge Cases

### 4.1 Config file doesn't exist

`LoadConfig()` already returns a default empty `Config` when the file is missing. The schema loader falls back to the hardcoded path. No regression.

### 4.2 Config exists but `schemas` section is missing

The `SchemasConfig` zero value has an empty `Path`. The loader falls back to the hardcoded path. No regression.

### 4.3 Absolute vs relative path

Absolute paths are used as-is. Relative paths are resolved against `projectDir`, consistent with how `tests.path` and `reports.path` work.

---

## 5. Implementation Plan

- [ ] Add `SchemasConfig` struct and `Schemas` field to `Config` in `internal/model/config.go`
- [ ] Update `LoadSchema()` to check config for overridden schemas path
- [ ] Update `cmd/init.go` template to match (it already has `schemas:` — no change needed)

---

## 6. Dependencies

None. All stdlib.

---

## 7. Decisions

### 7.1 Resolve inside LoadSchema vs pass as parameter

Resolving inside `LoadSchema()` is simpler and avoids changing the function signature across the codebase. The config load is cached at the OS level (file read), so the extra `LoadConfig()` call is negligible in the assertion path.
