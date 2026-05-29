# MCP Server Setup & Editor Integrations

Learn how to connect Gherkio's Model Context Protocol (MCP) server to your favorite AI assistant clients, IDEs, and environments.

---

## 🚀 Connecting to Clients

You can connect Gherkio's Model Context Protocol (MCP) server to your client using two execution paths:
1.  **Binary Mode (Standard)**: Invokes the locally installed `gherkio mcp` executable.
2.  **Zero-Install Mode (Go Run)**: Dynamically fetches and runs the server using standard Go remote module execution:
    ```bash
    go run github.com/muhfaris/gherkio@v0.1.0-alpha.1 mcp
    ```

---

## 🛠️ Configuration Blocks

### 1. Claude Desktop Setup
To enable Claude Desktop to invoke Gherkio's testing suite, add the server command to your local Claude configuration file:

- **Mac Path**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows Path**: `%APPDATA%\Claude\claude_desktop_config.json`

#### Option A: Local Binary (Recommended for speed)
```json
{
  "mcpServers": {
    "gherkio": {
      "command": "gherkio",
      "args": ["mcp"],
      "env": {
        "PATH": "/usr/local/bin:/opt/homebrew/bin:/usr/bin"
      }
    }
  }
}
```

#### Option B: Zero-Install (Go Run)
```json
{
  "mcpServers": {
    "gherkio": {
      "command": "go",
      "args": ["run", "github.com/muhfaris/gherkio@v0.1.0-alpha.1", "mcp"],
      "env": {
        "PATH": "/usr/local/bin:/opt/homebrew/bin:/usr/bin"
      }
    }
  }
}
```

---

### 2. Cursor IDE Integration
To use Gherkio dynamically within Cursor's Composer or Chat:

1. Open **Cursor Settings** ➔ **Features** ➔ **MCP**.
2. Click **+ Add New MCP Server**.
3. Configure the following fields:

#### Option A: Local Binary (Recommended for speed)
- **Name**: `gherkio`
- **Type**: `stdio`
- **Command**: `/absolute/path/to/gherkio mcp` (e.g. `/home/username/go/bin/gherkio mcp`)

#### Option B: Zero-Install (Go Run)
- **Name**: `gherkio`
- **Type**: `stdio`
- **Command**: `go run github.com/muhfaris/gherkio@v0.1.0-alpha.1 mcp`

Click **Save**. The status dot should immediately turn green.

---

### 3. VS Code Integration (Roo Code / Continue)
If you use **Roo Code (Roo Cline)** or **Continue** extensions in VS Code, add Gherkio to your custom MCP servers list:

#### Option A: Local Binary (Recommended for speed)
```json
{
  "mcpServers": {
    "gherkio": {
      "command": "gherkio",
      "args": ["mcp"]
    }
  }
}
```

#### Option B: Zero-Install (Go Run)
```json
{
  "mcpServers": {
    "gherkio": {
      "command": "go",
      "args": ["run", "github.com/muhfaris/gherkio@v0.1.0-alpha.1", "mcp"]
    }
  }
}
```

---

### 4. Zed Editor Configuration
To activate Gherkio's MCP capabilities within the Zed editor's built-in assistant, edit your local `~/.config/zed/settings.json` file:

#### Option A: Local Binary (Recommended for speed)
```json
{
  "context_servers": {
    "gherkio": {
      "command": "gherkio",
      "args": ["mcp"]
    }
  }
}
```

#### Option B: Zero-Install (Go Run)
```json
{
  "context_servers": {
    "gherkio": {
      "command": "go",
      "args": ["run", "github.com/muhfaris/gherkio@v0.1.0-alpha.1", "mcp"]
    }
  }
}
```


---

## ✅ Troubleshooting

### Red / Offline Connection State
If your client fails to connect or Gherkio's status dot is red:
1. **Check PATH**: Ensure the directory where `gherkio` is installed (e.g. `/home/username/go/bin` or `/usr/local/bin`) is present in the `PATH` environment variable defined in the configuration block.
2. **Execute manually**: Run `gherkio mcp` in your system shell. It should start silently and wait for stdin. Press `Ctrl+C` to terminate it. If it fails with `command not found`, re-run the `go install` procedure.
