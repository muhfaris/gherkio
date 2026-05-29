# RFC-06: MCP Server Support

**Status:** Implemented

Added a Model Context Protocol (MCP) server to Gherkio, enabling AI agents to interact with Gherkio directly.

**Key decisions:**
- Uses stdio transport (JSON-RPC 2.0)
- Exposes tools for running tests, validating, converting, and managing resources
- Exposes resources for DSL spec, matchers, variables, and paths
- Zero-dependency JSON-RPC implementation
