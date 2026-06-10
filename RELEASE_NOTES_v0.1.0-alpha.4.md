## v0.1.0-alpha.4

Gherkio v0.1.0-alpha.4 adds conditional step execution, improved DSL readability, enhanced HTML reports, and MCP tool fixes.

### ✨ New Features

**Conditional Step Execution**
- **Conditions Values**: Run steps conditionally based on variable values — steps now support `when:` blocks that evaluate variable conditions before execution.
- **Graphify Updates**: Improved graph visualization reflects conditional execution paths.

**DSL Readability Improvements**
- **Step `name` Field**: Added `name` field at step level for better test documentation and readable step labels in reports.
- **Query Map in Request Block**: Added `query` map support in the request block for structured query parameter definition.
- **Scenario `description` Field**: Added `description` field at the scenario level for documenting test intent and purpose.

**HTML Report Enhancements**
- **Structured Request Details**: HTML test reports now include a dedicated Request Details section showing method, URL, headers, and body in a clean, structured layout for faster debugging.

**Editor Support**
- **IDE Autocomplete Schema**: Added JSON Schema for editor autocomplete support, enabling IDEs to provide contextual completion for Gherkio DSL files.

**MCP Server Improvements**
- **`use_project` Tool**: New MCP tool to switch the active project directory at runtime.
- **`get_config` Tool**: New MCP tool to read and return the parsed `.gherkio/config.yaml` contents.
- **Step Completeness Warnings**: Validation now warns when a step name suggests it needs a query/body/multipart field that is missing (non-blocking warnings).

### 🐛 Bug Fixes

- **`update_test` Deep Merge**: `update_test` now performs deep merge instead of full replace — only explicitly present fields are overridden, preserving all other existing fields.
- **Report Field Propagation**: Fixed propagation of step `name` and `query` fields to HTML and JSON test reports.

### 📖 Documentation

- Added *Test Reports & Run Results* reference page to the Gherkio Developer Book.
- Added documentation for `description`, `name`, and `query` DSL fields.
- Added editor autocomplete schema documentation.

### Installation

```bash
# Linux/macOS
curl -fsSL https://raw.githubusercontent.com/muhfaris/gherkio/main/install.sh | sh

# Or build from source
git clone https://github.com/muhfaris/gherkio.git
cd gherkio
go build -o gherkio .
```

### Downloads

- [Linux amd64](https://github.com/muhfaris/gherkio/releases/download/v0.1.0-alpha.4/gherkio_v0.1.0-alpha.4_Linux_x86_64.tar.gz)
- [Linux arm64](https://github.com/muhfaris/gherkio/releases/download/v0.1.0-alpha.4/gherkio_v0.1.0-alpha.4_Linux_arm64.tar.gz)
- [macOS amd64](https://github.com/muhfaris/gherkio/releases/download/v0.1.0-alpha.4/gherkio_v0.1.0-alpha.4_Darwin_x86_64.tar.gz)
- [macOS arm64](https://github.com/muhfaris/gherkio/releases/download/v0.1.0-alpha.4/gherkio_v0.1.0-alpha.4_Darwin_arm64.tar.gz)
- [Windows amd64](https://github.com/muhfaris/gherkio/releases/download/v0.1.0-alpha.4/gherkio_v0.1.0-alpha.4_Windows_x86_64.zip)

### What's Next?

See the [changelog](https://github.com/muhfaris/gherkio/blob/main/CHANGELOG.md) for full details, or visit the [Gherkio Developer Book](https://muhfaris.github.io/gherkio/) for updated documentation.
