# gherkio mcp

Start Gherkio Model Context Protocol (MCP) server over stdio

### Synopsis

Starts the Gherkio Model Context Protocol (MCP) server over standard I/O (stdio).
This allows Gherkio capabilities (globbing tests, reading scenarios, running tests, resolving environments)
to be programmatically consumed by LLMs and agentic AI clients (e.g. Claude Desktop, VS Code extensions).

Stdout is strictly reserved for compliant JSON-RPC 2.0 frames. Diagnostic logs are output to Stderr.

```
gherkio mcp [flags]
```

### Options

```
  -h, --help   help for mcp
```

### SEE ALSO

* [gherkio](gherkio.md)	 - Gherkio is a testing and validation framework

