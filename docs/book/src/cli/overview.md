# CLI Command & Reference

The Gherkio Command Line Interface (CLI) is the central orchestration tool for validating project settings, starting testing suites, generating reports, and exporting cURL workflows.

---

## 🏃 Execution Rules & File Resolution

When executing `gherkio run <target>`, Gherkio resolves the target pathway under the following rules:

1. **Directories**: If `<target>` is a folder, Gherkio recursively scans all child files ending in `.yaml` or `.yml` under the folder, executing them sequentially or concurrently.
2. **Single Files**: If `<target>` points directly to a YAML file, Gherkio executes only that single file.
3. **Workspace Discovery**: Gherkio looks for a `.gherkio/config.yaml` file in the current working directory, then bubbles up parent directories until one is found. If no `.gherkio` workspace folder is detected, Gherkio errors and prompts you to run `gherkio init`.

---

## 🚦 Exit Status Codes

Gherkio follows standard POSIX process exit boundaries to ensure perfect compatibility with CI/CD platforms (GitHub Actions, GitLab CI, Jenkins):

- `0`: **Success**. All executed test scenarios passed, validation checks returned no warnings, or conversions completed successfully.
- `1`: **Failure / Error**. At least one step failed validation, an API returned a status/body mismatch, a syntax error was detected in your YAML files, or a microservice connection timed out.

---

## 🖥️ Print Output Modes

Modify how Gherkio logs information to terminal windows during execution using stdout flags:

- **Default (Clean)**: Prints structured step checks, colorized status ticks (`✔` / `❌`), and execution durations per step.
- **Verbose (`--verbose` / `-v`)**: Outputs the exact request URL, method, serialized payload body, headers, along with the full response body/headers from the target host. Dynamic credentials and keys are automatically masked.
- **Quiet (`--quiet` / `-q`)**: Suppresses standard output entirely, returning only the final summary block and process exit status code.

---

## 📚 Commands Map

| Command | Link | Description |
| :--- | :--- | :--- |
| `gherkio` | [gherkio](gherkio.md) | Root command showing general usage and flags. |
| `gherkio init` | [gherkio init](init.md) | Scaffolds a new Gherkio workspace `.gherkio/`. |
| `gherkio run` | [gherkio run](run.md) | Executes declarative test scenarios locally or in CI. |
| `gherkio convert`| [gherkio convert](convert.md) | Translates raw cURL statements ⇄ Gherkio YAML step formats. |
| `gherkio validate`| [gherkio validate](validate.md) | Performs static analysis checking on syntax, credentials, and schemas. |
| `gherkio schema` | [gherkio schema](schema.md) | Auto-generates Draft-07 JSON Schema for editor auto-completions. |
| `gherkio mcp` | [gherkio mcp](mcp.md) | Launches stdio Model Context Protocol (MCP) JSON-RPC communication. |
