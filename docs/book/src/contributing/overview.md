# Contributing Overview

Thank you for helping build Gherkio! This guide helps you get up to speed with setting up your local Go development environment, building the CLI, and running the test suite.

---

## 🛠️ 1. Local Development Setup

### Prerequisites
- **Go**: Version 1.25+ installed on your system.

### Scaffold & Run
Build the CLI tool locally and run tests directly:

```bash
# Build the binary
go build -o gherkio .

# Initialize a default playground project
./gherkio init

# Run a test scenario
./gherkio run example/login.yaml -v
```

---

## 🧪 2. Test Suite & Validation

Gherkio takes testing seriously. Please ensure all tests pass before submitting any pull requests.

### Running Go Tests
Run the entire Go test suite:
```bash
go test ./...
```

Run tests on a specific package (e.g., the runner engine):
```bash
go test -v ./internal/runner/
```

### Golden File Snapshot Tests
Printer rendering tests inside `/internal/runner/` use **golden snapshot files** (stored under `internal/runner/testdata/`) to compare exact terminal printer bytes.

If you intentionally alter terminal output formatting, regenerate these snapshot files using the `-update` flag:
```bash
go test ./internal/runner/ -update
```

---

## 📝 3. Git Commit Standards

We use the [Conventional Commits](https://www.conventionalcommits.org/) specification to keep the Git history clean and readable:

- `feat: add between matcher for numeric bounds`
- `fix: resolve memory leak on retry exponential backoff`
- `docs: update recipes with JWT token authentication`
- `test: expand coverage on built-in generators`
