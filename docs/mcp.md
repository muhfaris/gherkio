# MCP Server — AI Integration

Gherkio includes a built-in [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server that lets AI coding assistants interact with your Gherkio project directly — create and run tests, manage environments, validate schemas, and more — all through natural language.

---

## How it works

The MCP server runs over **stdio transport**. Your AI tool (e.g. Claude Desktop, Neovim with codecompanion.nvim, VS Code with Cline) spawns the server as a subprocess and communicates via JSON-RPC 2.0 messages.

```bash
gherkio mcp
```

That's it. Once running, the AI tool can list tests, run scenarios, create credentials, and perform any Gherkio operation programmatically.

---

## Editor setup

> **Note:** Replace `/usr/local/bin/gherkio` with the actual path to Gherkio on your system. Run `which gherkio` to find it.

### Claude Desktop

Add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "gherkio": {
      "command": "/usr/local/bin/gherkio",
      "args": ["mcp"]
    }
  }
}
```

### Neovim (codecompanion.nvim)

```lua
mcp = {
  servers = {
    ["gherkio"] = {
      cmd = { "/usr/local/bin/gherkio", "mcp" },
    },
  },
  opts = {
    default_servers = { "gherkio" },
  },
}
```

### VS Code (Cline / Roo Code extension)

Create or edit `.vscode/mcp.json` in your project:

```json
{
  "servers": {
    "gherkio": {
      "type": "stdio",
      "command": "/usr/local/bin/gherkio",
      "args": ["mcp"]
    }
  }
}
```

### VS Code (Continue extension)

Add to `~/.continue/config.json`:

```json
{
  "experimental": {
    "mcpServers": {
      "gherkio": {
        "command": "/usr/local/bin/gherkio",
        "args": ["mcp"]
      }
    }
  }
}
```

### Cursor

Add to `.cursor/mcp.json` in your project:

```json
{
  "mcpServers": {
    "gherkio": {
      "command": "/usr/local/bin/gherkio",
      "args": ["mcp"]
    }
  }
}
```

### JetBrains IDEs

1. Install the **MCP Server** plugin from the JetBrains Marketplace
2. Go to **Settings → Tools → MCP Server**
3. Add a new server:
   - **Name**: `gherkio`
   - **Command**: `/usr/local/bin/gherkio`
   - **Arguments**: `mcp`

### Zed

Add to your `settings.json`:

```json
{
  "mcp": {
    "gherkio": {
      "command": "/usr/local/bin/gherkio",
      "args": ["mcp"]
    }
  }
}
```

---

## Available tools

Once connected, the AI assistant can use these tools:

| Tool | Description |
|------|-------------|
| `run_test` | Execute a test scenario (supports `--env`, `--step`, `--section`, `--dry-run`, `--verbose`, `--account`) |
| `create_test` | Create a new test scenario YAML file |
| `read_test` | Read a test scenario's full contents |
| `update_test` | Update an existing test scenario |
| `delete_test` | Delete a test scenario |
| `validate_test` | Validate test syntax and structure without running it |
| `list_tests` | List all test scenarios with step counts |
| `list_environments` | List configured environments and their base URLs |
| `list_schemas` | List all custom schemas |
| `create_environment` | Create a new environment config |
| `update_environment` | Update an existing environment config |
| `create_credential` | Create credentials for an environment |
| `update_credential` | Update existing credentials |
| `read_credential` | Read credentials for an environment |
| `create_schema` | Create a schema definition file |
| `update_schema` | Update an existing schema definition |

---

## Available resources

The server exposes DSL reference resources so AI tools understand Gherkio's syntax without prior training:

| Resource | Content |
|----------|---------|
| `gherkio://dsl/spec` | Full DSL specification (scenario structure, steps, requests, assertions) |
| `gherkio://dsl/matchers` | All assertion matchers with descriptions and examples |
| `gherkio://dsl/variables` | Built-in generator variables (`$uuid`, `$randomInt`, etc.) |
| `gherkio://dsl/paths` | Canonical assertion and save paths (`body.*`, `headers.*`, `jwt.*`) |
| `gherkio://dsl/examples` | Complete working example scenarios |
| `gherkio://project/structure` | `.gherkio/` directory layout explanation |
| `gherkio://project/info` | Project metadata (name, version, workspace root) |

---

## Example prompts

Once configured, you can ask your AI assistant things like:

> "Create a login test for dummyjson that saves the access token"

> "Run the login test against staging with verbose output"

> "Show me all available environments"

> "Create a credentials file for staging with two accounts: admin and viewer"

> "Run all tests in the demo/ directory"

> "Validate the login test without running it"

> "Create a schema for user profiles"

---

## Troubleshooting

### "command not found"

Make sure Gherkio is installed and in your `PATH`:

```bash
which gherkio
# → /usr/local/bin/gherkio
```

If not found, install it or use the full path to the binary.

### "not a gherkio project"

The MCP server needs to be started inside (or above) a Gherkio project directory (one with a `.gherkio/` folder). If you see this error, navigate to your project root first.

If the AI tool's working directory is not your project root, provide the full path to the project directory instead, or configure the editor extension to start from the correct directory.
