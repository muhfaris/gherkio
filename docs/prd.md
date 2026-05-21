
# Gherkio — Declarative Integration Testing Platform

## Product Requirements Document (PRD)

Version: 1.0
Status: Draft
Author: OpenAI Assistant
Date: May 20, 2026

---

# 1. Vision

Gherkio is a declarative integration testing platform designed for API-driven systems.

The core philosophy:

> Integration testing should describe behavior orchestration, not force users to write low-level imperative code.

Gherkio is not intended to replace programming languages.
Instead, it provides a constrained declarative DSL for expressing:

* scenarios
* request flows
* state transitions
* assertions
* data extraction
* orchestration
* reporting

while intentionally limiting unrestricted programming constructs.

The goal is long-term readability, maintainability, observability, and collaboration.

---

# 2. Product Philosophy

## 2.1 Primary Principle

Gherkio is:

* declarative-first
* orchestration-focused
* assertion-centric
* readable by humans
* deterministic
* observable

Gherkio is NOT:

* a general-purpose programming language
* a scripting engine
* a no-code fantasy abstraction
* a replacement for backend test frameworks

---

# 3. Core Problem Statement

Modern integration testing tools suffer from one or more of the following problems:

## 3.1 Imperative Complexity

Most API testing frameworks require writing imperative code:

* async handling
* request orchestration
* variable mutation
* extraction logic
* assertions
* retries
* setup/teardown

This creates high maintenance overhead.

---

## 3.2 Gherkin Ambiguity

BDD/Gherkin-style tools become difficult to maintain because:

* natural language becomes ambiguous
* step definitions explode in complexity
* business language leaks technical implementation
* debugging becomes difficult

---

## 3.3 DSLs Becoming Programming Languages

Most testing DSLs eventually become:

* hidden scripting languages
* difficult to statically analyze
* difficult to report
* difficult to validate
* difficult to onboard

because unrestricted logic is introduced.

---

## 3.4 Reporting Is Treated as Secondary

Most testing frameworks optimize for:

* execution
* assertions
* CI integration

but not:

* readability
* debugging UX
* orchestration visualization
* historical analysis
* human collaboration

---

# 4. Product Goals

## 4.1 Primary Goals

### A. Declarative Scenario Authoring

Users should describe:

* what is happening
* what is expected
* how requests chain together

without writing large amounts of imperative code.

---

### B. Long-Term Readability

A scenario written today should still be readable after 1–2 years.

---

### C. Strong Observability

Every scenario execution must provide:

* request history
* response history
* assertion results
* extracted variables
* execution duration
* step visualization
* failure diagnostics

---

### D. Structured Assertions

Assertions must remain:

* readable
* deterministic
* analyzable
* reportable

without allowing arbitrary inline code.

---

### E. Controlled Complexity

Complexity should be isolated into:

* capabilities
* plugins
* helper systems

instead of leaking into the DSL itself.

---

# 5. Non-Goals

Gherkio will NOT:

## 5.1 Become a Full Programming Language

No unrestricted:

* loops
* arbitrary branching
* inline JavaScript
* inline Python
* arbitrary expressions

inside the DSL.

---

## 5.2 Replace Unit Testing

Gherkio focuses on:

* integration testing
* API orchestration
* scenario validation

not:

* isolated business logic tests
* low-level unit testing

---

## 5.3 Replace E2E Browser Testing

UI/browser automation is not phase 1.

---

# 6. Target Users

## 6.1 Primary Users

### Backend Engineers

Need:

* API integration testing
* regression testing
* workflow validation
* contract verification

---

### QA Automation Engineers

Need:

* readable test scenarios
* orchestration flows
* reusable scenarios
* reporting
* non-imperative structure

---

### Technical Product Teams

Need:

* understandable scenarios
* debugging visibility
* execution traceability

---

# 7. Core Product Architecture

Gherkio architecture consists of 4 layers.

---

## 7.1 Layer 1 — DSL Layer

Purpose:

Human-readable declarative orchestration.

Responsibilities:

* scenario structure
* request declaration
* assertions
* variable interpolation
* chaining

Must NOT contain:

* arbitrary code execution
* unrestricted logic

---

## 7.2 Layer 2 — Execution Engine

Purpose:

Execute scenarios deterministically.

Responsibilities:

* HTTP execution
* variable resolution
* lifecycle management
* extraction
* orchestration
* retries
* assertion execution
* artifact generation

---

## 7.3 Layer 3 — Capability System

Purpose:

Provide controlled extensibility.

Capabilities examples:

* JWT parsing
* schema validation
* JSON path querying
* UUID validation
* date assertions
* auth handling
* pagination validation

Capabilities are structured features, NOT arbitrary scripts.

---

## 7.4 Layer 4 — Escape Hatch

Purpose:

Handle unavoidable edge cases.

Examples:

* helper scripts
* external plugins
* custom processors

Must remain:

* isolated
* explicit
* non-default

The default path should always remain declarative.

---

# 8. DSL Design Principles

## 8.1 Structured Over Natural Language

BAD:

```gherkin
Given user successfully logs in with valid credentials
```

GOOD:

```yaml
login:
  as: admin
```

Reason:

* deterministic
* parseable
* analyzable
* auto-completable

---

## 8.2 Readability Over Cleverness

Scenarios must optimize for:

* scanning
* debugging
* maintenance

not syntax minimalism.

---

## 8.3 Constraints Create Clarity

The DSL should intentionally limit:

* syntax flexibility
* expression complexity
* abstraction depth

to preserve readability.

---

# 9. Scenario Model

## 9.1 Scenario Structure

Example:

```yaml
scenario: update item

steps:
  - login as admin
  - create item
  - update item
  - verify item updated
```

---

## 9.2 Step Model

Each step represents:

* an action
* an assertion
* an extraction
* orchestration behavior

Step types:

* request
* expect
* save
* use
* scenario
* setup
* teardown

---

# 10. Request System

## 10.1 Request Primitive

Example:

```yaml
request:
  method: POST
  url: /auth/login

  headers:
    Content-Type: application/json

  body:
    email: admin@test.com
    password: secret
```

---

## 10.2 Request Features

Supported:

* method
* URL
* query
* headers
* body
* multipart
* timeout
* retry
* auth
* cookies

---

## 10.3 Request Context

Each request generates:

* raw request
* raw response
* parsed body
* metadata
* timing info

stored in execution context.

---

# 11. Variable System

## 11.1 Extraction

Example:

```yaml
save:
  token: response.token
  itemId: response.data.id
```

---

## 11.2 Variable Rules

Variables are:

* immutable by default
* scoped
* typed internally

---

## 11.3 Variable Interpolation

Example:

```yaml
headers:
  Authorization: Bearer $token
```

---

## 11.4 Path Syntax

Supported:

```text
response.id
response.user.role
response.items[0].name
jwt.role
```

Avoid:

* magical syntax
* nested interpolation hell

---

# 12. Assertion Engine

The assertion engine is the heart of Gherkio.

---

## 12.1 Philosophy

Assertions should be:

* readable
* declarative
* structured
* reportable

NOT arbitrary code.

---

## 12.2 Basic Assertions

Example:

```yaml
expect:
  status: 200
```

---

## 12.3 Existence Assertions

```yaml
expect:
  body.id: exists
```

---

## 12.4 Type Assertions

```yaml
expect:
  body.id: uuid
  body.createdAt: datetime
```

---

## 12.5 Equality Assertions

```yaml
expect:
  body.role: admin
```

---

## 12.6 Collection Assertions

```yaml
expect:
  all(response.items.status): active
```

---

## 12.7 Length Assertions

```yaml
expect:
  count(response.items): 10
```

---

## 12.8 JWT Assertions

```yaml
expect:
  jwt.role: admin
  jwt.exp: exists
```

Internally:

* decode JWT
* validate structure
* parse claims

without exposing implementation complexity.

---

## 12.9 Schema Assertions

```yaml
expect:
  schema: user-response
```

---

## 12.10 Matchers

Built-in matchers:

* exists
* uuid
* email
* datetime
* number
* string
* array
* object
* null
* true
* false
* contains
* startsWith
* endsWith
* regex

---

# 13. Scenario Composition

## 13.1 Reusable Scenarios

Example:

```yaml
use:
  - login-as-admin
```

---

## 13.2 Shared Setup

Example:

```yaml
setup:
  - create-user
  - create-role
```

---

## 13.3 Shared Teardown

Example:

```yaml
teardown:
  - cleanup-test-data
```

---

# 14. Capability System

Capabilities expose controlled power.

---

## 14.1 JWT Capability

Provides:

* decode
* claim access
* expiration validation

---

## 14.2 Schema Capability

Provides:

* JSON schema validation
* OpenAPI integration

---

## 14.3 Auth Capability

Provides:

* bearer token injection
* session handling
* cookie persistence

---

## 14.4 Retry Capability

Provides:

* polling
* eventual consistency handling

without requiring loops.

Example:

```yaml
retry:
  attempts: 5
  interval: 1000
```

---

# 15. Reporting System

Reporting is a first-class product feature.

---

# 15.1 Reporting Philosophy

Users do not primarily write tests.

Users primarily:

* debug failures
* inspect execution
* analyze regressions
* validate workflows

Therefore reporting quality is critical.

---

# 15.2 Execution Report

Each run generates:

* step timeline
* request/response logs
* variable extraction logs
* assertion results
* duration metrics
* environment metadata
* retry history

---

# 15.3 Failure UX

BAD:

```text
Assertion failed
```

GOOD:

```text
Expected:
response.user.role = admin

Actual:
response.user.role = staff
```

---

# 15.4 Artifacts

Artifacts include:

* request payload
* response payload
* headers
* execution trace
* logs
* screenshots (future)

---

# 15.5 Historical Reporting

Future support:

* flaky detection
* historical trends
* performance drift
* regression analytics

---

# 16. Execution Model

## 16.1 Deterministic Execution

Scenario execution order must remain deterministic.

---

## 16.2 Isolated Context

Each scenario has isolated:

* variables
* execution state
* artifacts

---

## 16.3 Parallel Execution

Supported in later phases.

---

# 17. Error Handling Philosophy

Errors must:

* explain failure clearly
* include context
* avoid leaking engine internals

---

# 18. Plugin System

## 18.1 Plugin Goals

Allow extensibility without contaminating the DSL.

---

## 18.2 Plugin Boundaries

Plugins may:

* add matchers
* add capabilities
* add integrations

Plugins may NOT:

* modify DSL grammar dynamically
* inject arbitrary runtime syntax

---

# 19. Escape Hatch Design

## 19.1 Purpose

Some edge cases require custom logic.

The escape hatch exists to:

* preserve flexibility
* avoid bloating DSL complexity

---

## 19.2 Rules

Escape hatch usage should be:

* explicit
* isolated
* minimized

---

## 19.3 Example

```yaml
helper:
  use: generateSignedPayload
```

NOT:

```yaml
run: |
  arbitrary code everywhere
```

---

# 20. IDE Experience

Future IDE support should include:

* autocomplete
* validation
* schema hints
* path suggestions
* matcher suggestions
* inline diagnostics

This is only possible because the DSL remains constrained.

---

# 21. AI Compatibility

The DSL should be highly AI-generatable.

Why:

* structured
* deterministic
* predictable
* constrained vocabulary

AI compatibility becomes difficult once unrestricted scripting is introduced.

---

# 22. Security Considerations

## 22.1 Secret Handling

Support:

* environment variables
* secret masking
* encrypted storage

---

## 22.2 Artifact Redaction

Sensitive data should be redactable.

Examples:

* passwords
* tokens
* API keys

---

# 23. Environment System

Support:

* local
* staging
* production-safe read-only mode

Example:

```yaml
env:
  baseUrl: https://staging.api.com
```

---

# 24. CLI Design

Example:

```bash
gherkio run tests/
```

---

## 24.1 CLI Features

* run scenarios
* filter tags
* environment selection
* parallel execution
* report export
* watch mode

---

# 25. Future UI Dashboard

Potential future dashboard:

* execution history
* visual flows
* assertion explorer
* failure analytics
* artifacts viewer
* scenario explorer

---

# 26. MVP Scope

The MVP should intentionally remain narrow.

---

## 26.1 MVP Features

Required:

* request execution
* variable extraction
* structured assertions
* scenario composition
* reporting
* JWT assertions
* schema validation
* retry support

---

## 26.2 MVP Exclusions

Excluded initially:

* browser automation
* visual testing
* arbitrary scripting
* loops
* branching
* distributed execution

---

# 27. Technical Risks

## 27.1 DSL Complexity Creep

The biggest product risk.

Mitigation:

* strict constraints
* controlled capabilities
* escape hatch isolation

---

## 27.2 User Pressure for Arbitrary Logic

Users will request:

* loops
* conditions
* inline scripts

Mitigation:

* helper/plugin architecture
* intentionally constrained DSL

---

## 27.3 Reporting Scalability

Large execution artifacts may become expensive.

Mitigation:

* artifact retention policies
* compression
* lazy loading

---

# 28. Product Positioning

Gherkio is positioned between:

* imperative API testing frameworks
* brittle BDD systems
* low-code orchestration tools

Gherkio aims to become:

> a readable orchestration layer for integration testing.

---

# 29. Product Identity

## What Gherkio Is

* declarative
* readable
* constrained
* orchestration-focused
* assertion-driven
* observable

---

## What Gherkio Is Not

* another scripting language
* natural language parser
* magic no-code abstraction
* unit testing framework

---

# 30. Final Product Thesis

The core thesis behind Gherkio:

> Integration testing is fundamentally orchestration and state verification, not application programming.

Therefore:

* orchestration should be declarative
* assertions should be structured
* complexity should be isolated
* reporting should be first-class
* readability should survive long-term maintenance

The success of Gherkio depends not on how powerful the DSL becomes,

but on how well it prevents uncontrolled complexity while remaining expressive enough for real-world integration workflows.
