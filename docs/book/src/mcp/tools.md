# MCP Tools Reference

Gherkio's Model Context Protocol (MCP) server exposes a rich suite of **21 tools** that allow AI models (such as Gemini, Claude, or GPT) to interact natively with Gherkio test suites, config files, environments, credentials, schemas, and runners.

---

## 🛠️ Project & Workspace Tools

### 1. `init_project`
Initialize a new Gherkio project structure with default configuration, schemas, environments, and template examples in the specified directory.

- **Arguments**:
  - `path` (string, optional): Subdirectory or absolute path to initialize the Gherkio project. Defaults to the current working directory.
- **Example call**:
  ```json
  {
    "name": "init_project",
    "arguments": { "path": "my-api-tests" }
  }
  ```

### 2. `get_project_info`
Retrieve active Gherkio project metadata, workspace root, and resolved directory structures.

- **Arguments**: None (returns project directory paths and version).

---

## 🧪 Scenario Management

### 3. `list_tests`
Scan and retrieve a list of all Gherkio test scenarios (`.yaml` files) inside `.gherkio/tests/` with their step counts and scenario names.

- **Arguments**: None

### 4. `read_test`
Read the full raw Gherkio YAML test scenario contents of a given test file path.

- **Arguments**:
  - `path` (string, **required**): Absolute or relative path to the Gherkio scenario file.

### 5. `create_test`
Create a new Gherkio test scenario file in the project workspace under `.gherkio/tests/`. Statically validates syntax and schema references before writing to disk.

- **Arguments**:
  - `path` (string, **required**): Relative path under `.gherkio/tests/` to save the file (e.g. `auth/login.yaml`).
  - `scenarioYaml` (string, **required**): Raw YAML string of the Gherkio test scenario. **Rule**: Always prefix saved/referenced dynamic variables with the sequential step number (e.g., `1-authToken` for step 1, `2-resourceId` for step 2) for strict step-level traceability.

### 6. `update_test`
Overwrite an existing Gherkio test file scenario in the project workspace, writing a backup first.

- **Arguments**:
  - `path` (string, **required**): Absolute or relative path to the Gherkio scenario file to update.
  - `scenarioYaml` (string, **required**): Updated Gherkio test scenario YAML content. **Rule**: Always prefix saved/referenced dynamic variables with the sequential step number (e.g., `1-authToken` for step 1, `2-resourceId` for step 2) for strict step-level traceability.

### 7. `delete_test`
Permanently delete a Gherkio test file from the workspace.

- **Arguments**:
  - `path` (string, **required**): Absolute or relative path to the Gherkio scenario file to delete.

---

## 🏃 Execution & Validation

### 8. `run_test`
Execute a Gherkio test scenario (fully or step-isolated) and receive highly detailed, structured JSON results containing request headers/bodies, response details, timing budgets, and assertion outcomes.

- **Arguments**:
  - `path` (string, **required**): Absolute or relative path to the test scenario to run.
  - `env` (string, optional): Environment to execute against (defaults to project config default or `local`).
  - `account` (string, optional): Optional account name from environments credentials to use for dynamic variable injection.
  - `step` (integer, optional): Step index to execute in isolation (0-indexed). Defaults to `-1` (run all).
  - `section` (string, optional): Section to run (`setup`, `steps`, `teardown`).
  - `dryRun` (boolean, optional): Preview test execution without making HTTP requests.
  - `verbose` (boolean, optional): Show full request/response payloads. Defaults to `true`.
  - `failFast` (boolean, optional): Stop executing remaining steps when a step fails. Defaults to `false`.

### 9. `validate_test`
Validate a Gherkio scenario's syntax, structure, request methods, and schema/composition references without executing it.

- **Arguments**:
  - `scenarioYaml` (string, **required**): Gherkio YAML test scenario string to validate.

### 10. `validate_workspace`
Perform full, multi-file static analysis on all workspace tests, validating syntax, variable scopes, credentials, schema, and `use:` scenario references.

- **Arguments**:
  - `path` (string, optional): Optional path to a specific test file.
  - `env` (string, optional): Optional environment (e.g. `staging`) to validate credential credentials schemas.

---

## 🔑 Environment & Credentials Management

### 11. `list_environments`
List all configured Gherkio test environments along with their `baseUrl` and overriding services.

- **Arguments**: None

### 12. `create_environment`
Create a new environment config file in `.gherkio/environments/<name>.yaml`.

- **Arguments**:
  - `name` (string, **required**): Environment name (e.g., `staging`, `production`).
  - `yaml` (string, **required**): Environment YAML content containing `baseUrl` and services.

### 13. `update_environment`
Update an existing environment config file in `.gherkio/environments/<name>.yaml`.

- **Arguments**:
  - `name` (string, **required**): Environment name to update.
  - `yaml` (string, **required**): Updated environment YAML content.

### 14. `read_credential`
Read credentials for a specific environment.

- **Arguments**:
  - `env` (string, **required**): Environment name.

### 15. `create_credential`
Create a new credentials file for an environment under `.gherkio/credentials/<env>.yaml`.

- **Arguments**:
  - `env` (string, **required**): Environment name.
  - `yaml` (string, **required**): Credentials YAML content with accounts mapping.

### 16. `update_credential`
Update an existing credentials file for an environment.

- **Arguments**:
  - `env` (string, **required**): Environment name.
  - `yaml` (string, **required**): Updated credentials YAML content.

---

## 📐 Schema Validation

### 17. `list_schemas`
List all custom assertion schemas configured under `.gherkio/schemas/` in the workspace.

- **Arguments**: None

### 18. `create_schema`
Create a new schema definition file in `.gherkio/schemas/<name>.yaml`.

- **Arguments**:
  - `name` (string, **required**): Schema name (e.g. `user-profile`).
  - `yaml` (string, **required**): Schema YAML content defining expected structure.

### 19. `update_schema`
Update an existing schema definition file.

- **Arguments**:
  - `name` (string, **required**): Schema name to update.
  - `yaml` (string, **required**): Updated schema YAML content.

---

## 🔁 cURL & DSL Conversion

### 20. `convert_curl_to_yaml`
Convert a raw cURL command string into a formatted Gherkio DSL YAML test scenario or step.

- **Arguments**:
  - `curl` (string, **required**): Raw cURL command string to parse and convert.
  - `scenarioName` (string, optional): Optional custom scenario name wrapping the conversion. Defaults to `Converted curl`.
  - `stepOnly` (boolean, optional): If true, returns only the individual step's YAML content rather than the full scenario wrapper.
  - `env` (string, optional): Optional environment to interpolate and format base URL references.

### 21. `convert_yaml_to_curl`
Convert a Gherkio test scenario step (or all steps) into reproducible standard cURL commands.

- **Arguments**:
  - `path` (string, **required**): Relative or absolute path to the Gherkio test YAML file.
  - `step` (integer, optional): Optional step index (0-indexed) to convert.
  - `env` (string, optional): Optional environment name to load credentials.
  - `account` (string, optional): Optional account name to interpolate credential variables.
