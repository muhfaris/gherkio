# gherkio env context

Get environment context with auto-selection hints

### Synopsis

Returns unified environment context including available environments,
accounts for each environment, and auto-selection hints.

Auto-selection hints are computed as follows:
- If only 1 environment exists, it is auto-selected
- If only 1 environment has exactly 1 account, both are auto-selected
- If an environment has only 1 account, that account is auto-selected

This command is designed for programmatic consumption (nvim plugin, MCP).

```
gherkio env context [flags]
```

### Options

```
  -h, --help   help for context
      --json   Output as JSON
```

### SEE ALSO

* [gherkio env](env.md)	 - Manage Gherkio environments

