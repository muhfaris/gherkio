package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/muhfaris/gherkio/internal/converter"
	"github.com/muhfaris/gherkio/internal/core/credentialstore"
	"github.com/muhfaris/gherkio/internal/core/envcontext"
	"github.com/muhfaris/gherkio/internal/core/envstore"
	"github.com/muhfaris/gherkio/internal/core/project"
	"github.com/muhfaris/gherkio/internal/core/schemastore"
	"github.com/muhfaris/gherkio/internal/core/teststore"
	"github.com/muhfaris/gherkio/internal/model"
	"github.com/muhfaris/gherkio/internal/report"
	"github.com/muhfaris/gherkio/internal/runner"
	"github.com/muhfaris/gherkio/internal/schema"
	"gopkg.in/yaml.v3"
)

// Server implements a native Model Context Protocol (MCP) server over stdio.
type Server struct {
	projectDir string
}

// NewServer creates a new instance of the MCP server.
func NewServer(projectDir string) *Server {
	return &Server{
		projectDir: projectDir,
	}
}

// Start initiates the stdin read loop and dispatches JSON-RPC requests.
func (s *Server) Start() error {
	dec := json.NewDecoder(os.Stdin)
	for {
		var req RPCRequest
		if err := dec.Decode(&req); err != nil {
			if err == io.EOF {
				return nil
			}
			s.writeError(nil, ParseError, "Parse error: invalid JSON payload", err.Error())
			continue
		}

		if req.JSONRPC != "2.0" && req.JSONRPC != "" {
			s.writeError(req.ID, InvalidRequest, "Invalid Request: missing or invalid jsonrpc version", nil)
			continue
		}

		// Dispatches standard MCP protocol methods
		switch req.Method {
		case "initialize":
			s.handleInitialize(req.ID, req.Params)
		case "notifications/initialized":
			// Handshake notification (no ID) — ignore or log to stderr
			s.logStderr("Handshake finalized: client initialized")
		case "tools/list":
			s.handleListTools(req.ID, req.Params)
		case "tools/call":
			s.handleCallTool(req.ID, req.Params)
		case "resources/list":
			s.handleListResources(req.ID, req.Params)
		case "resources/read":
			s.handleReadResource(req.ID, req.Params)
		default:
			s.writeError(req.ID, MethodNotFound, fmt.Sprintf("Method not found: '%s'", req.Method), nil)
		}
	}
}

func (s *Server) handleInitialize(id interface{}, params json.RawMessage) {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: ServerCapabilities{
			Resources: struct{}{},
			Tools:     struct{}{},
		},
		ServerInfo: Implementation{
			Name:    "gherkio-mcp-server",
			Version: "0.1.0",
		},
	}
	s.writeResponse(id, result, nil)
}

func (s *Server) handleListTools(id interface{}, params json.RawMessage) {
	result := ListToolsResult{
		Tools: []Tool{
			{
				Name:        "init_project",
				Description: "Initialize a new Gherkio project structure with default configuration, schemas, environments, and template examples.",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Optional subdirectory or absolute path to initialize the Gherkio project. Defaults to the current working directory.",
						},
					},
				},
			},
			{
				Name:        "get_project_info",
				Description: "Retrieve active Gherkio project metadata, workspace root, and resolved directory structures.",
				InputSchema: InputSchema{Type: "object", Properties: map[string]interface{}{}},
			},
			{
				Name:        "use_project",
				Description: "Switch the active Gherkio project directory. Useful when working with multiple projects without re-initializing.",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Absolute or relative path to the Gherkio project directory to switch to.",
						},
					},
					Required: []string{"path"},
				},
			},
			{
				Name:        "get_config",
				Description: "View the parsed contents of .gherkio/config.yaml for the active project.",
				InputSchema: InputSchema{Type: "object", Properties: map[string]interface{}{}},
			},
			{
				Name:        "list_tests",
				Description: "Scan and retrieve a list of all Gherkio test scenarios (.yaml) with their step counts and scenario names.",
				InputSchema: InputSchema{Type: "object", Properties: map[string]interface{}{}},
			},
			{
				Name:        "read_test",
				Description: "Read the full raw Gherkio YAML test scenario contents of a given test file path.",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Absolute or relative path to the Gherkio scenario file.",
						},
					},
					Required: []string{"path"},
				},
			},
			{
				Name:        "validate_test",
				Description: "Validate a Gherkio scenario's syntax, structure, request methods, and schema/composition references.",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"scenarioYaml": map[string]interface{}{
							"type":        "string",
							"description": "Gherkio YAML test scenario string to validate.",
						},
					},
					Required: []string{"scenarioYaml"},
				},
			},
			{
				Name:        "create_test",
				Description: "Create a new Gherkio test scenario file in the project workspace (.gherkio/tests/). Checks validity before creation.",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Relative path under .gherkio/tests/ to save the file (e.g. auth/login.yaml).",
						},
						"scenarioYaml": map[string]interface{}{
							"type":        "string",
							"description": "Raw YAML string of the Gherkio test scenario. IMPORTANT: Always prefix any saved dynamic variables (in the 'save' block or referenced across steps) with the sequential step number (e.g. '1-authToken' for step 1, '2-resourceId' for step 2, depending on the scenario step order) to guarantee strict traceability.",
						},
					},
					Required: []string{"path", "scenarioYaml"},
				},
			},
			{
				Name:        "update_test",
				Description: "Overwrite an existing Gherkio test file scenario in the project workspace, writing a backup first.",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Absolute or relative path to the Gherkio scenario file to update.",
						},
						"scenarioYaml": map[string]interface{}{
							"type":        "string",
							"description": "Updated Gherkio test scenario YAML content. IMPORTANT: Always prefix any saved dynamic variables (in the 'save' block or referenced across steps) with the sequential step number (e.g. '1-authToken' for step 1, '2-resourceId' for step 2, depending on the scenario step order) to guarantee strict traceability.",
						},
					},
					Required: []string{"path", "scenarioYaml"},
				},
			},
			{
				Name:        "delete_test",
				Description: "Permanently delete a Gherkio test file from the workspace.",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Absolute or relative path to the Gherkio scenario file to delete.",
						},
					},
					Required: []string{"path"},
				},
			},
			{
				Name:        "list_environments",
				Description: "List all configured Gherkio test environments along with their baseUrl and overriding services.",
				InputSchema: InputSchema{Type: "object", Properties: map[string]interface{}{}},
			},
			{
				Name:        "get_environment_context",
				Description: "Get unified environment context with auto-selection hints. Returns environments, accounts per environment, and computed hints for automatic selection. Use this to determine what env/account should be auto-selected when only one option exists.",
				InputSchema: InputSchema{Type: "object", Properties: map[string]interface{}{}},
			},
			{
				Name:        "list_schemas",
				Description: "List all custom assertion schemas configured under .gherkio/schemas/ in the workspace.",
				InputSchema: InputSchema{Type: "object", Properties: map[string]interface{}{}},
			},
			{
				Name:        "run_test",
				Description: "Execute a Gherkio test scenario (fully or single-step isolated) and receive highly detailed, structured results.",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Absolute or relative path to the test scenario to run.",
						},
						"env": map[string]interface{}{
							"type":        "string",
							"description": "Environment to execute against (defaults to project config default or 'local').",
						},
						"account": map[string]interface{}{
							"type":        "string",
							"description": "Optional account name from environments credentials to use for dynamic variable injection. Not needed if the test uses $accounts.<name>.<field> syntax directly.",
						},
"step": map[string]interface{}{
							"type":        "integer",
							"description": "Step index to execute in isolation (0-indexed). Defaults to -1 (run entire scenario). When 'step' is set without 'section', defaults to the 'steps' section (use 'section' to target setup/teardown steps).",
						},
						"section": map[string]interface{}{
							"type":        "string",
							"description": "Section to run (setup, steps, teardown). When set without 'step', runs ALL steps in that section only. When combined with 'step', targets a specific step within that section (e.g. section=setup, step=0 runs the first setup step).",
						},
						"dryRun": map[string]interface{}{
							"type":        "boolean",
							"description": "Preview test execution without making HTTP requests (dry-run mode).",
						},
						"verbose": map[string]interface{}{
							"type":        "boolean",
							"description": "Show full request/response payloads and resolved variables. Defaults to true.",
						},
"until": map[string]interface{}{
							"type":        "string",
							"description": "Execute steps until a specific target. Format: '<section>:<index>' (e.g. 'steps:2' runs steps 0,1,2; 'setup:0' runs first setup step only). Or just a raw index '2' to slice the overall steps array. Sections: setup, steps, teardown.",
						},
						"failFast": map[string]interface{}{
							"type":        "boolean",
							"description": "Stop executing remaining steps when a step fails. Defaults to false.",
						},
					},
					Required: []string{"path"},
				},
			},
			{
				Name:        "create_credential",
				Description: "Create a new credentials file for an environment in .gherkio/credentials/<env>.yaml",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"env": map[string]interface{}{
							"type":        "string",
							"description": "Environment name (e.g. local, staging, production)",
						},
						"yaml": map[string]interface{}{
							"type":        "string",
							"description": "Credentials YAML content with accounts mapping",
						},
					},
					Required: []string{"env", "yaml"},
				},
			},
			{
				Name:        "update_credential",
				Description: "Update an existing credentials file for an environment",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"env": map[string]interface{}{
							"type":        "string",
							"description": "Environment name (e.g. local, staging, production)",
						},
						"yaml": map[string]interface{}{
							"type":        "string",
							"description": "Updated credentials YAML content",
						},
					},
					Required: []string{"env", "yaml"},
				},
			},
			{
				Name:        "read_credential",
				Description: "Read a credentials file for a specific environment",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"env": map[string]interface{}{
							"type":        "string",
							"description": "Environment name (e.g. local, staging, production)",
						},
					},
					Required: []string{"env"},
				},
			},
			{
				Name:        "create_environment",
				Description: "Create a new environment config file in .gherkio/environments/<name>.yaml",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Environment name (e.g. staging, production)",
						},
						"yaml": map[string]interface{}{
							"type":        "string",
							"description": "Environment YAML content (baseUrl, services)",
						},
					},
					Required: []string{"name", "yaml"},
				},
			},
			{
				Name:        "update_environment",
				Description: "Update an existing environment config file",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Environment name (e.g. staging)",
						},
						"yaml": map[string]interface{}{
							"type":        "string",
							"description": "Updated environment YAML content",
						},
					},
					Required: []string{"name", "yaml"},
				},
			},
			{
				Name:        "read_environment",
				Description: "Read an existing environment config file's parsed structure (baseUrl and service overrides)",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Environment name (e.g. local, staging)",
						},
					},
					Required: []string{"name"},
				},
			},
			{
				Name:        "create_schema",
				Description: "Create a new schema definition file in .gherkio/schemas/<name>.yaml",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Schema name (e.g. user-profile, api-response)",
						},
						"yaml": map[string]interface{}{
							"type":        "string",
							"description": "Schema YAML content defining expected structure",
						},
					},
					Required: []string{"name", "yaml"},
				},
			},
			{
				Name:        "update_schema",
				Description: "Update an existing schema definition file",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Schema name (e.g. user-profile)",
						},
						"yaml": map[string]interface{}{
							"type":        "string",
							"description": "Updated schema YAML content",
						},
					},
					Required: []string{"name", "yaml"},
				},
			},
			{
				Name:        "convert_curl_to_yaml",
				Description: "Convert a raw cURL command string into a formatted Gherkio DSL YAML test scenario or step.",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"curl": map[string]interface{}{
							"type":        "string",
							"description": "Raw cURL command string to parse and convert.",
						},
						"scenarioName": map[string]interface{}{
							"type":        "string",
							"description": "Optional custom scenario name wrapping the conversion. Defaults to 'Converted curl'.",
						},
						"stepOnly": map[string]interface{}{
							"type":        "boolean",
							"description": "If true, only returns the individual step's YAML content rather than the full scenario wrapper.",
						},
						"env": map[string]interface{}{
							"type":        "string",
							"description": "Optional environment to interpolate and format base URL references.",
						},
					},
					Required: []string{"curl"},
				},
			},
			{
				Name:        "convert_yaml_to_curl",
				Description: "Convert a Gherkio test scenario step (or all steps) into reproducible standard cURL commands.",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Relative or absolute path to the Gherkio test YAML file.",
						},
						"step": map[string]interface{}{
							"type":        "integer",
							"description": "Optional step index (0-indexed) to convert. If omitted, converts all steps in the scenario.",
						},
						"env": map[string]interface{}{
							"type":        "string",
							"description": "Optional environment name (e.g., local, staging) to load credentials and variables for injection.",
						},
						"account": map[string]interface{}{
							"type":        "string",
							"description": "Optional account name from environments credentials to interpolate credentials variables.",
						},
					},
					Required: []string{"path"},
				},
			},
			{
				Name:        "validate_workspace",
				Description: "Perform full, multi-file static analysis on the workspace tests, validating syntax, variable scopes, credentials, schema, and use scenario references.",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Optional relative or absolute path to a specific test file. If omitted, performs a complete scan of all test scenarios in the workspace.",
						},
						"env": map[string]interface{}{
							"type":        "string",
							"description": "Optional environment (e.g., local, staging) to load credentials for accounts variable validation. Defaults to 'local'.",
						},
					},
				},
			},
			{
				Name:        "get_dsl_variables",
				Description: "Get the dynamic reference list of all Gherkio DSL built-in variables and helper/generator functions.",
				InputSchema: InputSchema{Type: "object", Properties: map[string]interface{}{}},
			},
			{
				Name:        "get_dsl_spec",
				Description: "Get the Gherkio DSL grammar specification, lifecycle blocks, and execution rules.",
				InputSchema: InputSchema{Type: "object", Properties: map[string]interface{}{}},
			},
			{
				Name:        "get_dsl_matchers",
				Description: "Get the complete reference list of Gherkio's dynamic assertion matchers (e.g. equal, contains, greaterThan).",
				InputSchema: InputSchema{Type: "object", Properties: map[string]interface{}{}},
			},
		},
	}
	s.writeResponse(id, result, nil)
}

func (s *Server) handleCallTool(id interface{}, params json.RawMessage) {
	var call CallToolRequest
	if err := json.Unmarshal(params, &call); err != nil {
		s.writeError(id, InvalidParams, "Invalid tool call parameters", err.Error())
		return
	}

	if s.projectDir == "" && call.Name != "get_project_info" && call.Name != "use_project" && call.Name != "get_config" && call.Name != "init_project" && call.Name != "convert_curl_to_yaml" && call.Name != "get_dsl_variables" && call.Name != "get_dsl_spec" && call.Name != "get_dsl_matchers" {
		// MCP capability check: let LLM know we are outside of a Gherkio workspace
		s.writeToolError(id, "Gherkio project not initialized in this directory. Call 'init_project' or configure workspace.")
		return
	}

	switch call.Name {
	case "init_project":
		targetDir := s.projectDir
		if targetDir == "" {
			var err error
			targetDir, err = os.Getwd()
			if err != nil {
				s.writeToolError(id, fmt.Sprintf("Failed to get current working directory: %v", err))
				return
			}
		}

		if pathArg, ok := call.Arguments["path"].(string); ok && pathArg != "" {
			if filepath.IsAbs(pathArg) {
				targetDir = pathArg
			} else {
				targetDir = filepath.Join(targetDir, pathArg)
			}
		}

		// Use v0.1.0-alpha as default version
		err := project.Initialize(targetDir, "v0.1.0-alpha")
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to initialize project: %v", err))
			return
		}
		s.projectDir = targetDir
		s.writeToolResponse(id, fmt.Sprintf("✓ Successfully initialized Gherkio project in %s", targetDir))

	case "get_project_info":
		meta, err := project.GetMeta(s.projectDir)
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to get project info: %v", err))
			return
		}
		data, _ := json.MarshalIndent(meta, "", "  ")
		s.writeToolResponse(id, string(data))

	case "use_project":
		pathArg, _ := call.Arguments["path"].(string)
		if pathArg == "" {
			s.writeToolError(id, "Missing required argument 'path'")
			return
		}
		var targetDir string
		if filepath.IsAbs(pathArg) {
			targetDir = pathArg
		} else if s.projectDir != "" {
			targetDir = filepath.Join(s.projectDir, pathArg)
		} else {
			targetDir = pathArg
		}
		// Verify the target directory has a .gherkio/config.yaml
		configPath := filepath.Join(targetDir, ".gherkio", "config.yaml")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			s.writeToolError(id, fmt.Sprintf("No Gherkio project found at %s (missing .gherkio/config.yaml)", targetDir))
			return
		}
		s.projectDir = targetDir
		s.writeToolResponse(id, fmt.Sprintf("✓ Switched to Gherkio project at %s", targetDir))

	case "get_config":
		configPath := filepath.Join(s.projectDir, ".gherkio", "config.yaml")
		data, err := os.ReadFile(configPath)
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to read config: %v", err))
			return
		}
		// Parse YAML to return structured config
		var cfg model.Config
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to parse config: %v", err))
			return
		}
		out, _ := json.MarshalIndent(cfg, "", "  ")
		s.writeToolResponse(id, string(out))

	case "list_tests":
		meta, err := project.GetMeta(s.projectDir)
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to get project info: %v", err))
			return
		}
		tests, err := teststore.ListTests(meta.TestsDir)
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to list tests: %v", err))
			return
		}
		data, _ := json.MarshalIndent(tests, "", "  ")
		s.writeToolResponse(id, string(data))

	case "read_test":
		path, _ := call.Arguments["path"].(string)
		if path == "" {
			s.writeToolError(id, "Missing required argument 'path'")
			return
		}
		fullPath := s.resolvePath(path)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to read test file: %v", err))
			return
		}
		s.writeToolResponse(id, string(data))

	case "validate_test":
		scenarioYaml, _ := call.Arguments["scenarioYaml"].(string)
		if scenarioYaml == "" {
			s.writeToolError(id, "Missing required argument 'scenarioYaml'")
			return
		}

		var tf model.TestFile
		if err := yaml.Unmarshal([]byte(scenarioYaml), &tf); err != nil {
			s.writeToolError(id, fmt.Sprintf("Invalid YAML structure: %v", err))
			return
		}

		meta, err := project.GetMeta(s.projectDir)
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to get project info: %v", err))
			return
		}

		res, err := teststore.Validate(&tf, meta.SchemasDir)
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Validation processing error: %v", err))
			return
		}

		data, _ := json.MarshalIndent(res, "", "  ")
		s.writeToolResponse(id, string(data))

	case "create_test":
		path, _ := call.Arguments["path"].(string)
		scenarioYaml, _ := call.Arguments["scenarioYaml"].(string)
		if path == "" || scenarioYaml == "" {
			s.writeToolError(id, "Missing required arguments 'path' and 'scenarioYaml'")
			return
		}

		var tf model.TestFile
		if err := yaml.Unmarshal([]byte(scenarioYaml), &tf); err != nil {
			s.writeToolError(id, fmt.Sprintf("Invalid YAML structure: %v", err))
			return
		}

		meta, err := project.GetMeta(s.projectDir)
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to get project info: %v", err))
			return
		}

		err = teststore.CreateTest(meta.TestsDir, meta.SchemasDir, path, &tf)
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to create test: %v", err))
			return
		}
		s.writeToolResponse(id, fmt.Sprintf("✓ Successfully created test scenario at .gherkio/tests/%s", path))

	case "update_test":
		path, _ := call.Arguments["path"].(string)
		scenarioYaml, _ := call.Arguments["scenarioYaml"].(string)
		if path == "" || scenarioYaml == "" {
			s.writeToolError(id, "Missing required arguments 'path' and 'scenarioYaml'")
			return
		}

		meta, err := project.GetMeta(s.projectDir)
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to get project info: %v", err))
			return
		}

		fullPath := s.resolvePath(path)

		// Read existing file for deep merge
		existingData, err := os.ReadFile(fullPath)
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to read existing test file: %v", err))
			return
		}

		var existing model.TestFile
		if err := yaml.Unmarshal(existingData, &existing); err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to parse existing test file: %v", err))
			return
		}

		// Parse which top-level keys are present in the incoming YAML
		var rawIncoming map[string]interface{}
		if err := yaml.Unmarshal([]byte(scenarioYaml), &rawIncoming); err != nil {
			s.writeToolError(id, fmt.Sprintf("Invalid YAML structure: %v", err))
			return
		}

		var incoming model.TestFile
		if err := yaml.Unmarshal([]byte(scenarioYaml), &incoming); err != nil {
			s.writeToolError(id, fmt.Sprintf("Invalid YAML structure: %v", err))
			return
		}

		// Deep merge: only apply fields that are explicitly present in incoming YAML
		merged := existing
		if _, ok := rawIncoming["scenario"]; ok {
			merged.Scenario = incoming.Scenario
		}
		if _, ok := rawIncoming["description"]; ok {
			merged.Description = incoming.Description
		}
		if _, ok := rawIncoming["tags"]; ok {
			merged.Tags = incoming.Tags
		}
		if _, ok := rawIncoming["setup"]; ok {
			merged.Setup = incoming.Setup
		}
		if _, ok := rawIncoming["steps"]; ok {
			merged.Steps = incoming.Steps
		}
		if _, ok := rawIncoming["teardown"]; ok {
			merged.Teardown = incoming.Teardown
		}

		err = teststore.UpdateTest(fullPath, &merged, meta.SchemasDir)
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to update test: %v", err))
			return
		}
		s.writeToolResponse(id, fmt.Sprintf("✓ Successfully updated test scenario at %s (backup created)", path))

	case "delete_test":
		path, _ := call.Arguments["path"].(string)
		if path == "" {
			s.writeToolError(id, "Missing required argument 'path'")
			return
		}
		fullPath := s.resolvePath(path)
		err := teststore.DeleteTest(fullPath)
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to delete test: %v", err))
			return
		}
		s.writeToolResponse(id, fmt.Sprintf("✓ Successfully deleted test scenario at %s", path))

	case "list_environments":
		envs, err := envstore.List(s.projectDir)
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to list environments: %v", err))
			return
		}
		data, _ := json.MarshalIndent(envs, "", "  ")
		s.writeToolResponse(id, string(data))

	case "get_environment_context":
		ctx, err := envcontext.GetContext(s.projectDir)
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to get environment context: %v", err))
			return
		}
		data, _ := json.MarshalIndent(ctx, "", "  ")
		s.writeToolResponse(id, string(data))

	case "list_schemas":
		schemas, err := schemastore.List(s.projectDir)
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to list schemas: %v", err))
			return
		}
		data, _ := json.MarshalIndent(schemas, "", "  ")
		s.writeToolResponse(id, string(data))

	case "run_test":
		path, _ := call.Arguments["path"].(string)
		envName, _ := call.Arguments["env"].(string)
		accountName, _ := call.Arguments["account"].(string)
		stepVal, _ := call.Arguments["step"].(float64)
		dryRun, _ := call.Arguments["dryRun"].(bool)
		verbose := true
		if v, ok := call.Arguments["verbose"].(bool); ok {
			verbose = v
		}
		untilArg, _ := call.Arguments["until"].(string)
		failFast, _ := call.Arguments["failFast"].(bool)

		if path == "" {
			s.writeToolError(id, "Missing required argument 'path'")
			return
		}

		fullPath := s.resolvePath(path)
		stepIndex := -1
		if stepVal != 0 {
			stepIndex = int(stepVal)
		}

		sectionArg, _ := call.Arguments["section"].(string)
		// Default to "steps" when step is set without section (backward compat)
		if stepIndex >= 0 && sectionArg == "" {
			sectionArg = "steps"
		}

		// Load config
		appCfg, _ := project.LoadConfig(s.projectDir)

		// Set default environment name
		if envName == "" {
			envName = "local"
			if appCfg != nil && appCfg.Environments.Default != "" {
				envName = appCfg.Environments.Default
			}
		}

		// Expose run setup similar to cmd/run.go
		var credentialVars map[string]interface{}
		var credentialMaskFields []string
		var allAccountsMap map[string]interface{}

		creds, _ := runner.LoadCredentials(s.projectDir, envName)
		if creds != nil {
			allAccountsMap = creds.ToMap()
			// Resolve specific account or single/fallback account
			accName := accountName
			if accName == "" {
				names := creds.AccountNames()
				if len(names) == 1 {
					accName = names[0]
				}
			}
			if acc, exists := creds.GetAccount(accName); exists {
				credentialVars = runner.CredentialsToVars(acc)
				credentialMaskFields = append(credentialMaskFields, runner.GetSensitiveFieldsFromCredentials(acc)...)
			}
		}

		// Determine mask fields
		maskFieldsToUse := runner.GetDefaultSensitiveFields()
		if appCfg != nil && appCfg.Security.Mask.Enabled && len(appCfg.Security.Mask.Fields) > 0 {
			maskFieldsToUse = appCfg.Security.Mask.Fields
		}
		// Append credential specific masks if any
		maskFieldsToUse = append(maskFieldsToUse, credentialMaskFields...)

		// Build snapshot configuration
		var snapshotCfg runner.SnapshotConfig
		if appCfg != nil {
			snapshotCfg = runner.SnapshotConfig{
				Enabled:       appCfg.Reports.Failures.Enabled,
				MaskSensitive: appCfg.Reports.Failures.MaskSensitive,
				MaskFields:    maskFieldsToUse,
				RetainCount:   appCfg.Reports.Failures.RetainCount,
			}
			if appCfg.Reports.Failures.Path != "" {
				snapshotCfg.Path = appCfg.Reports.Failures.Path
			} else {
				snapshotCfg.Path = filepath.Join(s.projectDir, ".gherkio", "reports", "failures")
			}
		} else {
			snapshotCfg = runner.SnapshotConfig{
				Enabled:       false,
				MaskSensitive: true,
				MaskFields:    maskFieldsToUse,
				RetainCount:   10,
				Path:          filepath.Join(s.projectDir, ".gherkio", "reports", "failures"),
			}
		}

		cfg := runner.RunConfig{
			TestPath:       fullPath,
			EnvName:        envName,
			ProjectDir:     s.projectDir,
			Verbose:        verbose,
			MaskFields:     maskFieldsToUse,
			AccountName:    accountName,
			CredentialVars: credentialVars,
			AllAccounts:    allAccountsMap,
			StepIndex:      stepIndex,
			StepSection:    sectionArg,
			DryRun:         dryRun,
			Snapshot:       snapshotCfg,
			Until:          untilArg,
			FailFast:       failFast,
		}

		result, err := runner.Run(cfg)
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Execution execution failed: %v", err))
			return
		}

		// Write HTML/JSON report if configured in the config file
		if appCfg != nil && appCfg.Reports.Format != "" {
			reportCfg := &report.ReportConfig{
				Format:        appCfg.Reports.Format,
				Path:          appCfg.Reports.Path,
				MaskSensitive: appCfg.Reports.MaskSensitive,
				Retention:     appCfg.Reports.Retention,
			}
			formats := strings.Split(reportCfg.Format, ",")
			for _, format := range formats {
				format = strings.TrimSpace(format)
				switch format {
				case "html":
					if html, rerr := report.RenderHTML(result, *reportCfg, envName); rerr == nil {
						_, _ = report.SaveHTML(html, s.projectDir, reportCfg.Path)
					}
				case "json":
					if jsonStr, rerr := report.RenderJSON(result, *reportCfg, envName); rerr == nil {
						_, _ = report.SaveJSON(jsonStr, s.projectDir, reportCfg.Path)
					}
				}
			}
			_ = report.EnforceRetention(s.projectDir, reportCfg.Path, reportCfg.Retention)
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		s.writeToolResponse(id, string(data))

	case "create_credential":
		env, _ := call.Arguments["env"].(string)
		yamlContent, _ := call.Arguments["yaml"].(string)
		if env == "" || yamlContent == "" {
			s.writeToolError(id, "Missing required arguments 'env' and 'yaml'")
			return
		}
		var creds model.Credentials
		if err := yaml.Unmarshal([]byte(yamlContent), &creds); err != nil {
			s.writeToolError(id, fmt.Sprintf("Invalid YAML: %v", err))
			return
		}
		if err := creds.Validate(); err != nil {
			s.writeToolError(id, fmt.Sprintf("Invalid credentials: %v", err))
			return
		}
		if err := credentialstore.Create(s.projectDir, env, &creds); err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to create credentials: %v", err))
			return
		}
		s.writeToolResponse(id, fmt.Sprintf("✓ Created credentials for '%s' at .gherkio/credentials/%s.yaml", env, env))

	case "update_credential":
		env, _ := call.Arguments["env"].(string)
		yamlContent, _ := call.Arguments["yaml"].(string)
		if env == "" || yamlContent == "" {
			s.writeToolError(id, "Missing required arguments 'env' and 'yaml'")
			return
		}
		var creds model.Credentials
		if err := yaml.Unmarshal([]byte(yamlContent), &creds); err != nil {
			s.writeToolError(id, fmt.Sprintf("Invalid YAML: %v", err))
			return
		}
		if err := credentialstore.Update(s.projectDir, env, &creds); err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to update credentials: %v", err))
			return
		}
		s.writeToolResponse(id, fmt.Sprintf("✓ Updated credentials for '%s'", env))

	case "read_credential":
		env, _ := call.Arguments["env"].(string)
		if env == "" {
			s.writeToolError(id, "Missing required argument 'env'")
			return
		}
		creds, err := credentialstore.Read(s.projectDir, env)
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to read credentials: %v", err))
			return
		}
		data, _ := json.MarshalIndent(creds, "", "  ")
		s.writeToolResponse(id, string(data))

	case "create_environment":
		name, _ := call.Arguments["name"].(string)
		yamlContent, _ := call.Arguments["yaml"].(string)
		if name == "" || yamlContent == "" {
			s.writeToolError(id, "Missing required arguments 'name' and 'yaml'")
			return
		}
		var env model.Environment
		if err := yaml.Unmarshal([]byte(yamlContent), &env); err != nil {
			s.writeToolError(id, fmt.Sprintf("Invalid YAML: %v", err))
			return
		}
		if err := envstore.Create(s.projectDir, name, &env); err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to create environment: %v", err))
			return
		}
		s.writeToolResponse(id, fmt.Sprintf("✓ Created environment '%s' at .gherkio/environments/%s.yaml", name, name))

	case "update_environment":
		name, _ := call.Arguments["name"].(string)
		yamlContent, _ := call.Arguments["yaml"].(string)
		if name == "" || yamlContent == "" {
			s.writeToolError(id, "Missing required arguments 'name' and 'yaml'")
			return
		}
		var env model.Environment
		if err := yaml.Unmarshal([]byte(yamlContent), &env); err != nil {
			s.writeToolError(id, fmt.Sprintf("Invalid YAML: %v", err))
			return
		}
		if err := envstore.Update(s.projectDir, name, &env); err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to update environment: %v", err))
			return
		}
		s.writeToolResponse(id, fmt.Sprintf("✓ Updated environment '%s'", name))

	case "read_environment":
		name, _ := call.Arguments["name"].(string)
		if name == "" {
			s.writeToolError(id, "Missing required argument 'name'")
			return
		}
		env, err := envstore.Read(s.projectDir, name)
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to read environment: %v", err))
			return
		}
		data, _ := json.MarshalIndent(env, "", "  ")
		s.writeToolResponse(id, string(data))

	case "create_schema":
		name, _ := call.Arguments["name"].(string)
		yamlContent, _ := call.Arguments["yaml"].(string)
		if name == "" || yamlContent == "" {
			s.writeToolError(id, "Missing required arguments 'name' and 'yaml'")
			return
		}
		var schema model.Schema
		if err := yaml.Unmarshal([]byte(yamlContent), &schema); err != nil {
			s.writeToolError(id, fmt.Sprintf("Invalid YAML: %v", err))
			return
		}
		if err := schemastore.Create(s.projectDir, name, &schema); err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to create schema: %v", err))
			return
		}
		s.writeToolResponse(id, fmt.Sprintf("✓ Created schema '%s' at .gherkio/schemas/%s.yaml", name, name))

	case "update_schema":
		name, _ := call.Arguments["name"].(string)
		yamlContent, _ := call.Arguments["yaml"].(string)
		if name == "" || yamlContent == "" {
			s.writeToolError(id, "Missing required arguments 'name' and 'yaml'")
			return
		}
		var schema model.Schema
		if err := yaml.Unmarshal([]byte(yamlContent), &schema); err != nil {
			s.writeToolError(id, fmt.Sprintf("Invalid YAML: %v", err))
			return
		}
		if err := schemastore.Update(s.projectDir, name, &schema); err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to update schema: %v", err))
			return
		}
		s.writeToolResponse(id, fmt.Sprintf("✓ Updated schema '%s'", name))

	case "convert_curl_to_yaml":
		{
			curl, _ := call.Arguments["curl"].(string)
			if curl == "" {
				s.writeToolError(id, "Missing required argument 'curl'")
				return
			}
			scenarioName, _ := call.Arguments["scenarioName"].(string)
			if scenarioName == "" {
				scenarioName = "Converted curl"
			}
			stepOnly, _ := call.Arguments["stepOnly"].(bool)
			env, _ := call.Arguments["env"].(string)

			req, _, err := converter.ParseCurl(curl)
			if err != nil {
				s.writeToolError(id, fmt.Sprintf("Failed to parse cURL command: %v", err))
				return
			}

			yamlStr, err := converter.FormatYAML(req, scenarioName, stepOnly, s.projectDir, env)
			if err != nil {
				s.writeToolError(id, fmt.Sprintf("Failed to format YAML: %v", err))
				return
			}

			s.writeToolResponse(id, yamlStr)
		}

	case "convert_yaml_to_curl":
		{
			path, _ := call.Arguments["path"].(string)
			if path == "" {
				s.writeToolError(id, "Missing required argument 'path'")
				return
			}
			stepIdxVal, stepHas := call.Arguments["step"]
			env, _ := call.Arguments["env"].(string)
			account, _ := call.Arguments["account"].(string)

			fullPath := s.resolvePath(path)
			testFile, err := runner.LoadTestFile(fullPath)
			if err != nil {
				s.writeToolError(id, fmt.Sprintf("Failed to load test file: %v", err))
				return
			}

			// Load environment and credentials if in a project
			var creds *model.Credentials
			if s.projectDir != "" && env != "" {
				creds, _ = runner.LoadCredentials(s.projectDir, env)
			}

			// Determine credential vars and all accounts for $accounts.<name>.<field> access
			var credentialVars map[string]interface{}
			allAccountsMap := make(map[string]interface{})
			if creds != nil {
				allAccountsMap = creds.ToMap()
				if account != "" {
					if acc, exists := creds.GetAccount(account); exists {
						credentialVars = runner.CredentialsToVars(acc)
					}
				} else {
					// Fallback to first/auto account if only one exists
					names := creds.AccountNames()
					if len(names) == 1 {
						if acc, exists := creds.GetAccount(names[0]); exists {
							credentialVars = runner.CredentialsToVars(acc)
						}
					}
				}
			}

			// Merge all accounts into credential vars for $accounts.<name>.<field> access
			if len(allAccountsMap) > 0 {
				if credentialVars == nil {
					credentialVars = make(map[string]interface{})
				}
				credentialVars["accounts"] = allAccountsMap
			}

			// Determine which steps to convert
			var targetSteps []model.Step
			var singleStepIdx int = -1
			if stepHas && stepIdxVal != nil {
				switch v := stepIdxVal.(type) {
				case float64:
					singleStepIdx = int(v)
				case int:
					singleStepIdx = v
				}
			}

			if singleStepIdx >= 0 {
				if singleStepIdx >= len(testFile.Steps) {
					s.writeToolError(id, fmt.Sprintf("step index %d out of bounds (contains %d steps)", singleStepIdx, len(testFile.Steps)))
					return
				}
				targetSteps = []model.Step{testFile.Steps[singleStepIdx]}
			} else {
				targetSteps = testFile.Steps
			}

			var curls []string
			for idx, step := range targetSteps {
				// Inject fresh built-in generator variables per step
				stepVars := make(map[string]interface{})
				for key, val := range runner.LoadGherkioEnvVars() {
					stepVars[key] = val
				}
				for k, v := range credentialVars {
					stepVars[k] = v
				}
				for key, val := range runner.BuiltinVars() {
					stepVars[key] = val
				}

				if step.Use != "" {
					curls = append(curls, fmt.Sprintf("# use: %s (skipped composition)", step.Use))
					continue
				}

				c, err := converter.ConvertStepToCurl(step.Request, s.projectDir, env, stepVars)
				if err != nil {
					curls = append(curls, fmt.Sprintf("# error converting step %d: %v", idx, err))
					continue
				}
				curls = append(curls, c)
			}

			s.writeToolResponse(id, strings.Join(curls, "\n"))
		}

	case "validate_workspace":
		{
			path, _ := call.Arguments["path"].(string)
			env, _ := call.Arguments["env"].(string)
			if env == "" {
				env = "local"
			}

			results, err := project.ValidateProject(s.projectDir, s.projectDir, path, env)
			if err != nil {
				s.writeToolError(id, fmt.Sprintf("Validation failed: %v", err))
				return
			}

			data, _ := json.MarshalIndent(results, "", "  ")
			s.writeToolResponse(id, string(data))
		}

	case "get_dsl_variables":
		s.writeToolResponse(id, s.buildVariablesResource())

	case "get_dsl_matchers":
		s.writeToolResponse(id, s.buildMatchersResource())

	case "get_dsl_spec":
		s.writeToolResponse(id, s.buildSpecResource())

	default:
		s.writeError(id, MethodNotFound, fmt.Sprintf("Tool not found: '%s'", call.Name), nil)
	}
}

func (s *Server) handleListResources(id interface{}, params json.RawMessage) {
	result := ListResourcesResult{
		Resources: []Resource{
			{
				URI:         "gherkio://dsl/spec",
				Name:        "Gherkio DSL Grammar Spec",
				Description: "Markdown documentation outlining Gherkio DSL fields, dot-paths, and collection matchers.",
				MimeType:    "text/markdown",
			},
			{
				URI:         "gherkio://dsl/schema.json",
				Name:        "Gherkio Autocomplete JSON Schema",
				Description: "Automatically generated Draft-07 JSON Schema mapping internal models to DSL fields.",
				MimeType:    "application/json",
			},
			{
				URI:         "gherkio://dsl/examples",
				Name:        "Gherkio DSL Canonical Examples",
				Description: "YAML code scenarios showing request, saves, assertions, compositions, and collections in action.",
				MimeType:    "text/yaml",
			},
			{
				URI:         "gherkio://dsl/matchers",
				Name:        "Available Matchers",
				Description: "List of all supported assertion matchers with descriptions and usage examples.",
				MimeType:    "application/json",
			},
			{
				URI:         "gherkio://dsl/variables",
				Name:        "Built-in Variables",
				Description: "Built-in generator variables available in every test run ($uuid, $ulid, $randomInt, etc.).",
				MimeType:    "application/json",
			},
			{
				URI:         "gherkio://dsl/paths",
				Name:        "Canonical Paths",
				Description: "Canonical dot-notation paths for assertions and saves (body, headers, jwt).",
				MimeType:    "application/json",
			},
			{
				URI:         "gherkio://project/structure",
				Name:        "Project Directory Structure",
				Description: "The .gherkio/ directory layout explaining where each file type belongs.",
				MimeType:    "text/markdown",
			},
		},
	}
	s.writeResponse(id, result, nil)
}

func (s *Server) handleReadResource(id interface{}, params json.RawMessage) {
	var read struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(params, &read); err != nil {
		s.writeError(id, InvalidParams, "Invalid resource read params", err.Error())
		return
	}

	var content string
	var mime string

	switch read.URI {
	case "gherkio://dsl/spec":
		mime = "text/markdown"
		content = s.buildSpecResource()

	case "gherkio://dsl/schema.json":
		mime = "application/json"
		schemaData, err := schema.GenerateAllSchemas()
		if err != nil {
			s.writeError(id, InternalError, "Failed to generate autocomplete schemas", err.Error())
			return
		}
		content = string(schemaData)

	case "gherkio://dsl/examples":
		mime = "text/yaml"
		content = s.buildExamplesResource()
	case "gherkio://dsl/matchers":
		mime = "application/json"
		content = s.buildMatchersResource()

	case "gherkio://dsl/variables":
		mime = "application/json"
		content = s.buildVariablesResource()

	case "gherkio://dsl/paths":
		mime = "application/json"
		content = s.buildPathsResource()

	case "gherkio://project/structure":
		mime = "text/markdown"
		content = s.buildProjectStructureResource()

	default:
		s.writeError(id, InvalidParams, fmt.Sprintf("Unknown resource URI '%s'", read.URI), nil)
		return
	}

	result := ReadResourceResult{
		Contents: []ResourceContent{
			{
				URI:      read.URI,
				MimeType: mime,
				Text:     content,
			},
		},
	}
	s.writeResponse(id, result, nil)
}

func (s *Server) resolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	// Try relative to .gherkio/tests/
	testPath := filepath.Join(s.projectDir, ".gherkio", "tests", p)
	if _, err := os.Stat(testPath); err == nil {
		return testPath
	}
	// Fallback to relative to s.projectDir
	return filepath.Join(s.projectDir, p)
}

func (s *Server) writeResponse(id interface{}, result interface{}, err *RPCError) {
	resp := RPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		Error:   err,
		ID:      id,
	}
	data, _ := json.Marshal(resp)
	os.Stdout.Write(data)
	os.Stdout.Write([]byte("\n"))
}

func (s *Server) writeError(id interface{}, code int, message string, data interface{}) {
	s.writeResponse(id, nil, &RPCError{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

func (s *Server) writeToolResponse(id interface{}, text string) {
	s.writeResponse(id, CallToolResult{
		Content: []Content{
			{Type: "text", Text: text},
		},
	}, nil)
}

func (s *Server) writeToolError(id interface{}, errMsg string) {
	s.writeResponse(id, CallToolResult{
		Content: []Content{
			{Type: "text", Text: errMsg},
		},
		IsError: true,
	}, nil)
}

func (s *Server) logStderr(msg string) {
	fmt.Fprintf(os.Stderr, "ℹ [MCP] %s\n", msg)
}
