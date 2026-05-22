# RFC-10: Capability System

> **Status:** Draft
> **Author:** Faris
> **Date:** May 22, 2026

---

## 1. Summary

Formalize the **Capability System** as a first-class architectural layer in Gherkio. JWT decoding and schema validation currently exist as loose helper functions. This RFC introduces a structured registration and extension mechanism for capabilities, making them discoverable, documented, and independently testable.

---

## 2. Motivation

The PRD (§7.3, §14) describes capabilities as a core architectural layer:

> Capabilities are structured features, NOT arbitrary scripts.

Currently:

- **JWT parsing** is a standalone function (`decodeJWT()`) called inline in `runner.go`
- **Schema validation** is a standalone package (`LoadSchema()`, `ValidateSchema()`) called inline in `executor.go`
- **Matchers** are a flat switch statement in `matchers.go`
- There is no registry, no interface, no discoverability

This leads to several problems:

1. **No discoverability** — A new contributor must read the entire codebase to find all capabilities
2. **No uniform interface** — JWT and Schema have different calling conventions, error handling, and configuration patterns
3. **No isolation** — Capability logic leaks into the runner/executor loops, making them harder to test
4. **No extension point** — Adding a new capability (e.g. OAuth, pagination, UUID generation) requires modifying core runner files rather than adding a self-contained capability

---

## 3. Design

### 3.1 Capability Interface

A capability is a Go interface in a new `internal/capability/` package:

```go
// internal/capability/capability.go

package capability

import "context"

// A Capability is a named, self-contained feature that can be invoked
// during step execution.
type Capability interface {
	// Name returns the unique identifier for this capability (e.g. "jwt", "schema")
	Name() string

	// Description returns a human-readable summary of what this capability does
	Description() string

	// Execute runs the capability with the given context and input.
	// The input/output are free-form maps to avoid coupling to the runner types.
	Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
}
```

### 3.2 Capability Registry

A global (or per-execution) registry holds all registered capabilities:

```go
// internal/capability/registry.go

package capability

import "fmt"

var registry = make(map[string]Capability)

// Register adds a capability to the global registry.
func Register(c Capability) {
	registry[c.Name()] = c
}

// Get retrieves a capability by name.
func Get(name string) (Capability, error) {
	c, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("capability not found: %s", name)
	}
	return c, nil
}

// List returns all registered capabilities.
func List() []Capability {
	var caps []Capability
	for _, c := range registry {
		caps = append(caps, c)
	}
	return caps
}
```

### 3.3 JWT Capability (refactored)

```go
// internal/capability/jwt.go

type JWT struct{}

func (j *JWT) Name() string        { return "jwt" }
func (j *JWT) Description() string { return "Decode and inspect JWT tokens without signature verification" }

func (j *JWT) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	token, ok := input["token"].(string)
	if !ok {
		return nil, fmt.Errorf("jwt: missing 'token' string input")
	}

	claims, err := decodeJWT(token)
	if err != nil {
		return nil, fmt.Errorf("jwt: %w", err)
	}

	return map[string]interface{}{
		"claims": claims,
	}, nil
}
```

### 3.4 Schema Capability (refactored)

```go
// internal/capability/schema.go

type Schema struct {
	ProjectDir string // schemas directory root
}

func (s *Schema) Name() string        { return "schema" }
func (s *Schema) Description() string { return "Validate JSON response bodies against YAML schema definitions" }

func (s *Schema) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	// ... load and validate schema ...
}
```

### 3.5 Runner Integration

Instead of calling `decodeJWT()` directly, the runner uses the capability registry:

```go
// Current (before)
claims, err := decodeJWT(tokenStr)

// After
jwtCap, _ := capability.Get("jwt")
result, err := jwtCap.Execute(ctx, map[string]interface{}{"token": tokenStr})
claims := result["claims"]
```

This is opt-in: the runner can continue to call capabilities directly for performance-critical paths, but the capability interface provides a standardized contract.

### 3.6 Capability Configuration via YAML

Capabilities that require configuration (e.g. schema directory) can be configured in `.gherkio/config.yaml`:

```yaml
capabilities:
  jwt:
    enabled: true
  schema:
    enabled: true
    path: .gherkio/schemas
```

---

## 4. Edge Cases

### 4.1 Backward Compatibility

All existing functionality (JWT decoding, schema validation) must continue to work unchanged. The capability system is an **internal refactoring** — the DSL syntax (`jwt.role`, `schema: user-response`) does not change.

### 4.2 Capability Ordering

Some capabilities may have implicit dependencies (e.g. schema depends on file system access). Dependencies between capabilities are out of scope for this RFC — each capability is assumed to be self-contained.

### 4.3 Performance

The capability interface introduces a slight overhead (map serialization). For performance-critical paths, capabilities may expose additional direct methods. The interface is the **default contract**; optimizations are opt-in.

---

## 5. Implementation Plan

### Phase 1 — Capability Package + JWT Refactor

- [ ] Create `internal/capability/` package
- [ ] Define `Capability` interface
- [ ] Implement `Registry` (Register, Get, List)
- [ ] Refactor `decodeJWT()` into `JWT` capability
- [ ] Register JWT capability at startup (`init()` or `main()`)
- [ ] Update runner to use `capability.Get("jwt")` instead of direct call

### Phase 2 — Schema Refactor

- [ ] Refactor `LoadSchema()` + `ValidateSchema()` into `Schema` capability
- [ ] Pass `ProjectDir` via capability configuration
- [ ] Update runner to use `capability.Get("schema")`

### Phase 3 — Configuration & CLI

- [ ] Add `Capabilities` field to `Config` model
- [ ] Add `gherkio capabilities` CLI command to list registered capabilities
- [ ] Document capability lifecycle (registration → configuration → execution)

---

## 6. Example Output

```bash
$ gherkio capabilities
Available capabilities:

  jwt     Decode and inspect JWT tokens without signature verification
  schema  Validate JSON response bodies against YAML schema definitions
```

---

## 7. Decisions

### 7.1 Capabilities are internal, not DSL-exposed

Capabilities are an internal architectural layer. Users don't write `capability: jwt` in their YAML — they write `jwt.role: admin` as before. The capability system is how Gherkio organizes its own features, not a user-facing extension API.

### 7.2 Plugin system is separate

The Capability System (this RFC) formalizes Gherkio's own internal features. The **Plugin System** (PRD §18) will be a separate RFC for third-party extensibility. Plugins may implement the `Capability` interface to integrate with Gherkio's execution pipeline.

### 7.3 No DSL change

This is purely an internal refactoring. No YAML syntax changes. No user-facing impact except:
- Better error messages from capabilities
- The new `gherkio capabilities` list command
- Easier onboarding for contributors

---

## 8. Dependencies

| Dependency | Purpose |
|------------|---------|
| `context` (stdlib) | Standard Go context for cancellation/timeout in capability execution |

No new external dependencies.
