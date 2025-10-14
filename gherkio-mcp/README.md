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

Or install the binary into your `$GOBIN`/`$GOPATH/bin` for reuse from any
workspace:

```bash
make install
```

To run directly:

```bash
GOCACHE=$(pwd)/.gocache go run ./cmd/gherkio-mcp
```

> Tip: the binary reads from stdin and writes to stdout. When testing manually,
> you can pipe JSON-RPC requests using `jq -n` or a small script.

## Usage

1. Start the MCP bridge (either the compiled binary or `go run` command above).
2. Connect an MCP-compatible client over stdio using the regular handshake
   sequence.
3. Use the `resources/*` and `tools/*` methods outlined below to explore and
   generate test scenarios.

### Listing & Reading Resources

List the repository files exposed to the MCP client:

```json
{"jsonrpc":"2.0","id":"2","method":"resources/list","params":{}}
```

Fetch the contents of a specific feature file:

```json
{
  "jsonrpc": "2.0",
  "id": "3",
  "method": "resources/read",
  "params": {"uri": "gherkio://features/petstore/get_pet.feature"}
}
```

### Generating Scenarios with AI Support

The `gherkio.scenario.suggest` tool turns high-level API intent into Gherkin you
can review or save. Invoke it via `tools/execute`:

```json
{
  "jsonrpc": "2.0",
  "id": "4",
  "method": "tools/execute",
  "params": {
    "name": "gherkio.scenario.suggest",
    "arguments": {
      "featureName": "Get pet",
      "api": {
        "name": "petstore",
        "operation": "GET /pet/{id}",
        "responseStatus": 200
      }
    }
  }
}
```

The result contains:

- a docstring-ready `scenario` block,
- suggested file and scenario names, and
- a payload you can feed straight into `gherkio.feature.write`.

### Previewing vs. Writing Features

Use `gherkio.feature.preview` to render the final feature text before touching
disk:

```json
{
  "jsonrpc": "2.0",
  "id": "5",
  "method": "tools/execute",
  "params": {
    "name": "gherkio.feature.preview",
    "arguments": {
      "feature": {"name": "Get pet", "description": "Ensure pets are retrievable"},
      "scenarios": [
        {"name": "Get a pet", "steps": [
          "Given a pet exists",
          "When I request the pet by id",
          "Then the response status is 200"
        ]}
      ]
    }
  }
}
```

Once satisfied, persist the feature using the same payload with
`gherkio.feature.write` (optionally overriding the destination path with
`relativePath`).

### Proxying CLI Commands

To reuse the existing `gherkio` CLI over MCP, call the provided proxies:

```json
{
  "jsonrpc": "2.0",
  "id": "6",
  "method": "tools/execute",
  "params": {"name": "gherkio.call", "arguments": {"args": ["apis", "list"]}}
}
```

`gherkio.run` follows the same shape and expects the workflow slug as the first
argument.

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
