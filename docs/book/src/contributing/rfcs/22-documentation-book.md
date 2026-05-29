# RFC-22: Public Documentation Book & Interactive Onboarding Playground

**Status:** Fully Implemented & Automated

---

## 📖 1. Context & Motivation

To drive wide-scale developer adoption of Gherkio, it is essential to have a single, highly readable, structured, and search-optimized source of truth. Relying on disorganized static text files inside a Git repository leads to high cognitive load, onboarding abandonment, and out-of-date documentation.

---

## 🎨 2. Architectural Design Decisions

We proposed, designed, and implemented a multi-tiered documentation ecosystem:

### Tier A: Progressive Multi-Page Book (`docs/book/`)
We selected **mdBook** to parse markdown files into an ultra-fast static HTML website. The contents are structured progressively to guide a user from absolute beginner to high-advanced master:
- **Getting Started**: Sequential paths for system installation, interactive scaffolds (`gherkio init`), and sandbox execution.
- **Declarative DSL Reference**: Explaining structured YAML definitions, custom variable scopes, dynamic matchers, authentication credential caching, and scenario compositions (`use:`).
- **Advanced Recipes**: Production-grade scenarios for JWT assertions, negative tests, parallel workers, and exponential polling retries.

### Tier B: Zero-Build Cobra & MCP Reference Autogenerators
To eliminate human maintenance drift:
1. **Cobra CLI Reflection**: Built a pipeline ([gendoc.go](../../../../gendoc.go)) that reads the live command definition tree from [cmd/root.go](../../../../cmd/root.go) and builds perfect CLI manuals in `docs/book/src/cli/`.
2. **MCP Schema Parsers**: Exposes dynamic documentation for Gherkio's **21 background tools** and **7 resources** mapped directly from [internal/mcp/server.go](../../../../internal/mcp/server.go).

### Tier C: Immersive Onboarding Sandbox (`docs/playground/`)
To lower onboarding barriers to zero, created a premium browser companion app:
- **Interactive Stepper**: Renders parsed YAML DSL files into animated graphical timeline nodes.
- **cURL Translator**: Translates standard cURL CLI statements into valid Gherkio steps instantly.

---

## ⚙️ 3. Verification & Deployment

- **Format linting**: Verified via local execution: `mdbook build` inside the `docs/book/` folder.
- **Zero Drift Integration**: Added a dedicated `make docs` build script target to run generators and compile the static build in CI/CD pipelines before deployment to **GitHub Pages**.
