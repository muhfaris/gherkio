# Frequently Asked Questions

## General

### What is Gherkio?
Gherkio is a declarative integration testing platform that lets teams write API integration tests in pure YAML. It compiles to a single static Go binary with zero external runtime dependencies. Tests describe multi-step HTTP request sequences, extract variables, define rich assertions, and enforce security policies — all in a self-documenting YAML DSL.

### What makes Gherkio different from Postman or Bruno?
Postman and Bruno are GUI-centric API clients. Gherkio is a CLI-first integration testing platform designed for CI/CD pipelines. Tests are plain YAML files that live in your repository, execute deterministically, and produce structured reports — no GUI required, no vendor lock-in.

### Does Gherkio require Node.js, Python, or a JVM?
No. Gherkio is a single static Go binary with zero external runtime dependencies. It runs anywhere Go compiles — Linux, macOS, Windows, Docker, and even air-gapped environments.

### Is Gherkio free and open source?
Yes. Gherkio is released under the MIT license and is fully open source. You can use it for personal projects, internal tools, and commercial products without restrictions.

## Technical

### What API styles does Gherkio support?
Gherkio's HTTP engine supports REST APIs, GraphQL endpoints (via JSON POST), and any HTTP-based API. Native gRPC support is on the roadmap.

### How does Gherkio handle authentication?
Gherkio supports multi-account credentials, environment variable injection, automatic bearer token injection, cookie persistence, and sensitive field masking. You can run the same test against admin, user, and read-only accounts using `gherkio run --all-accounts`.

### Can Gherkio validate response schemas?
Yes. Gherkio includes native JSON schema validation. Define your expected schema in `.gherkio/schemas/` and assert against it inline:
```yaml
expect:
  schema: user-response
```

### Does Gherkio support retries and polling?
Yes. Gherkio includes a configurable retry engine with exponential backoff, fixed intervals, and status-based exit conditions for eventual consistency testing.

### Can I reuse scenarios across test files?
Yes. Gherkio supports scenario composition with the `use:` keyword, letting you compose complex test flows from reusable building blocks.

## CI/CD & Automation

### Can Gherkio integrate with my CI/CD pipeline?
Yes. Gherkio produces POSIX exit codes, structured JSON reports, and machine-readable output. It integrates with GitHub Actions, GitLab CI, Jenkins, CircleCI, and any CI platform that can run a Go binary.

### Does Gherkio support parallel test execution?
Yes. Use the `-p` or `--parallel` flag to control concurrency: `gherkio run . -p 4`.

### Can Gherkio run tests against multiple environments?
Yes. Gherkio supports environment profiles defined in `.gherkio/environments/`. Switch between local, staging, and production configurations without modifying test files.

## AI Integration

### Can AI coding assistants write Gherkio tests?
Yes. Gherkio ships with a native Model Context Protocol (MCP) server that exposes your testing sandbox to AI assistants. Tools like Cursor, Claude Desktop, Cline, and Copilot can read specifications, generate scenarios, validate structures, and run tests using natural language.

### How do I set up the MCP server?
See the [MCP Setup Guide](../mcp/setup.md) for configuration blocks for Claude Desktop, Cursor, VS Code, Zed, and Neovim.

## Troubleshooting

### How do I convert existing cURL commands to Gherkio YAML?
Use the CLI: `gherkio convert --curl "curl -X POST https://api.example.com/login"`. Or use the [Interactive Playground](../getting-started/playground.md) for a visual cURL-to-YAML translator.

### How do I debug a failing test?
Run your test with the `-v` (verbose) flag to see full request/response payloads, assertion results, and execution traces. Gherkio also generates structured HTML and JSON reports under `.gherkio/reports/`.

### Where can I get help?
- Open an issue on [GitHub](https://github.com/muhfaris/gherkio/issues)
- Check the [Changelog](../changelog.md) for recent changes and known issues
- Review the [Contributing Guide](../contributing/overview.md) for development questions
