# gherkio-mcp

`gherkio-mcp` is a lightweight Model Context Protocol (MCP) bridge for the
Gherkio repository. It exposes the project structure and key CLI operations to
AI agents over JSON-RPC 2.0 via standard input/output. The binary does not
reuse any of the main `gherkio` application code; it is a separate module that
can later be split into its own repository.

## Features

- **Handshake + Capability Negotiation** implementing the MCP handshake flow.
- **Resource catalogue** for everything under `gherkio/` (`envs`, `apis`,
  `flows`, `fixtures`, `features`). Each file is surfaced as
  `gherkio://<relative-path>` with MIME hints.
- **Resource retrieval** delivering UTF-8 text content with change stamps.
- **Tool registry** containing three initial helpers:
  - `gherkio.call` – proxy to `gherkio call ...`
  - `gherkio.run` – proxy to `gherkio run ...`
  - `gherkio.feature.write` – create or overwrite feature files from structured
    arguments.
  - `gherkio.feature.preview` – render the resulting feature text without
    touching the filesystem so users can confirm the scenario first.
  - `gherkio.scenario.suggest` – generate a ready-to-save Gherkin scenario
    skeleton (and corresponding `gherkio.feature.write` payload) from high level
    API intent.
- **Ping** responses for health checks.
- **Official SDK**: built on top of `github.com/modelcontextprotocol/go-sdk` for spec-compliant transports, schemas, and lifecycle.

The CLI transport is stream-based: MCP clients should keep stdin/stdout open
and exchange JSON objects terminated by newlines.

## Build

```bash
cd gherkio-mcp
GOCACHE=$(pwd)/.gocache go build ./cmd/gherkio-mcp
```

To run directly:

```bash
GOCACHE=$(pwd)/.gocache go run ./cmd/gherkio-mcp
```

> Tip: the binary reads from stdin and writes to stdout. When testing manually,
> you can pipe JSON-RPC requests using `jq -n` or a small script.

## Configuration

- `GHERKIO_ROOT`: Overrides the detected repository root used for resource
  discovery (default: current working directory).
- `GHERKIO_BIN`: Absolute or relative path to the main `gherkio` executable
  used by `gherkio.call` and `gherkio.run` (default: `<repo>/gherkio`). Ensure
  the binary exists and is executable.

## Example Session

```
# Terminal 1
./gherkio-mcp/cmd/gherkio-mcp/gherkio-mcp

# Terminal 2 (client side)
{"jsonrpc":"2.0","id":"1","method":"handshake","params":{"protocolVersion":"2024-10-01"}}
```

Expected response:

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "protocolVersion": "2024-10-01",
    "server": {"name": "gherkio-mcp", "version": "0.1.0"},
    "capabilities": {
      "resources": {"list": true, "read": true},
      "tools": {"list": true, "execute": true},
      "events": {"stream": false}
    }
  }
}
```

Subsequent calls can invoke `resources/list`, `resources/read`, `tools/list`,
`tools/execute`, and `ping`.

## Limitations & Next Steps

- Transport is limited to stdio; WebSocket and gRPC transports can be layered in
  a future release.
- Tool execution proxies the existing CLI and inherits its behaviour; it does
  not stream intermediate logs yet.
- Resource content is served as full text without pagination.
- Authentication, sandboxing, and policy enforcement are out of scope for the
  MVP.

When moving this module into its own repository, update the module path in
`go.mod` and adjust the default discovery of the `gherkio` workspace.
