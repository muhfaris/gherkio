package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/muhfaris/gherkio/internal/core/envstore"
	"github.com/muhfaris/gherkio/internal/core/credentialstore"
	"github.com/muhfaris/gherkio/internal/core/project"
	"github.com/muhfaris/gherkio/internal/core/schemastore"
	"github.com/muhfaris/gherkio/internal/core/teststore"
	"github.com/muhfaris/gherkio/internal/model"
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
				Name:        "get_project_info",
				Description: "Retrieve active Gherkio project metadata, workspace root, and resolved directory structures.",
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
							"description": "Raw YAML string of the Gherkio test scenario.",
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
							"description": "Updated Gherkio test scenario YAML content.",
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
						"description": "Step index to execute in isolation (0-indexed). Defaults to -1 (run entire scenario or section if section is set).",
					},
					"section": map[string]interface{}{
						"type":        "string",
						"description": "Section to run (setup, steps, teardown). When set without step, runs ALL steps in that section only.",
					},
"dryRun": map[string]interface{}{
						"type":        "boolean",
						"description": "Preview test execution without making HTTP requests (dry-run mode).",
					},
					"verbose": map[string]interface{}{
						"type":        "boolean",
						"description": "Show full request/response payloads and resolved variables. Defaults to true.",
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

	if s.projectDir == "" && call.Name != "get_project_info" {
		// MCP capability check: let LLM know we are outside of a Gherkio workspace
		s.writeToolError(id, "Gherkio project not initialized in this directory. Call 'gherkio init' or configure workspace.")
		return
	}

	switch call.Name {
	case "get_project_info":
		meta, err := project.GetMeta(s.projectDir)
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Failed to get project info: %v", err))
			return
		}
		data, _ := json.MarshalIndent(meta, "", "  ")
		s.writeToolResponse(id, string(data))

	case "list_tests":
		tests, err := teststore.ListTests(s.projectDir)
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

		res, err := teststore.Validate(&tf, s.projectDir)
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

		err := teststore.CreateTest(s.projectDir, path, &tf)
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

		var tf model.TestFile
		if err := yaml.Unmarshal([]byte(scenarioYaml), &tf); err != nil {
			s.writeToolError(id, fmt.Sprintf("Invalid YAML structure: %v", err))
			return
		}

		fullPath := s.resolvePath(path)
		err := teststore.UpdateTest(fullPath, &tf, s.projectDir)
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

		// Set default environment name
		if envName == "" {
			envName = "local"
			cfg, err := project.LoadConfig(s.projectDir)
			if err == nil && cfg.Environments.Default != "" {
				envName = cfg.Environments.Default
			}
		}

		// Expose run setup similar to cmd/run.go
		var credentialVars map[string]interface{}
		var maskFields []string
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
				maskFields = append(maskFields, runner.GetSensitiveFieldsFromCredentials(acc)...)
			}
		}

		cfg := runner.RunConfig{
			TestPath:       fullPath,
			EnvName:        envName,
			ProjectDir:     s.projectDir,
Verbose:        verbose,
			MaskFields:     maskFields,
			AccountName:    accountName,
			CredentialVars: credentialVars,
			AllAccounts:    allAccountsMap,
			StepIndex:      stepIndex,
			StepSection:    sectionArg,
			DryRun:         dryRun,
		}

		result, err := runner.Run(cfg)
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Execution execution failed: %v", err))
			return
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
		content = `## Gherkio DSL Grammar Spec

### Structural Keys
- **scenario**: (String, Required) Human readable name of the scenario.
- **setup**: (List of Steps, Optional) Scenario pre-conditions.
- **steps**: (List of Steps, Required) Execution block steps.
- **teardown**: (List of Steps, Optional) Post-execution cleanup steps.

### Step Block
- **use**: (String, Conditional) Path to compose/execute another scenario. Mutually exclusive with request.
- **request**: (Request object, Conditional) HTTP Request config. Mutually exclusive with use.
- **expect**: (Expect object, Optional) Response assertions.
- **save**: (Map of name:path, Optional) Extract dynamic values to context variables. Paths support variable interpolation (e.g. 'body.data[$randomInt(0,9)].id').
- **timing**: (TimingConfig, Optional) Execution latency check.

### Request Config
- **service**: (String, Optional) Named service override matching environments.
- **method**: (String, Required) HTTP Method (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS).
- **url**: (String, Required) Target endpoint url (appends to baseUrl). Supports variable interpolation.
- **headers**: (Map of string:string, Optional) Custom HTTP headers. Supports variable interpolation in values.
- **body**: (Free-form object/string, Optional) Request body content. Supports variable interpolation in string values.

### Variable Interpolation
All string values in request fields support variable substitution:
- **\$var** — Simple variable reference (e.g. \$username, \$token)
- **\${var}** — Explicit braces syntax (e.g. \${accessToken})
- **\${var:default}** — With default fallback (e.g. \${role:user})
- **\$accounts.<name>.<field>** — Access any account's credentials directly from .gherkio/credentials/<env>.yaml without needing --account flag (e.g. \$accounts.eka.username)
- **\${func(arg1,arg2)}** — Parametrized built-in generator with arguments (e.g. \${randomInt(1,100)})

Variables are sourced from:
1. **Built-in generators** — Pre-populated variables available in every test run:
   - **\$uuid** — UUID v4 string (e.g. a1b2c3d4-e5f6-4789-abcd-ef1234567890)
   - **\$ulid** — ULID string (e.g. 01ARZ3NDEKTSV4RRFFQ69G5FAV)
   - **\$randomInt** — Random integer between 0 and 999999 (e.g. 74291). Use **\${randomInt(min,max)}** for custom range (e.g. \${randomInt(1,100)})
   - **\$randomEmail** — Random email at @example.com (e.g. user_123456@example.com)
   - **\$randomPhone** — Random Indonesian-format phone number (e.g. +6281234567890)
2. **Credentials** — Account fields from .gherkio/credentials/<env>.yaml (injected automatically when --account is used, or via \$accounts.<name>.<field>)
3. **Step saves** — Values extracted from previous step responses via save: blocks
4. **Saved vars override credentials** — When a step saves a variable with the same name as a credential

Built-in variables can be overridden by credentials or step saves with the same name.

### Assertions (Expect)
- **status**: (Integer) Expected HTTP status (e.g. 'status: 200').
- **body.<path>**: Assert on JSON body fields using a matcher or literal value (e.g. 'body.id: exists', 'body.name: Emily').
- **headers.<name>**: Assert on response header values (e.g. 'headers.content-type: contains application/json').
- **jwt.<claim>**: Assert on decoded JWT claims (e.g. 'jwt.role: admin').
- **schema**: Validate full body against a YAML schema file in .gherkio/schemas/ (e.g. 'schema: user-profile').
  Negative form: 'schema: not <name>' asserts the response does NOT match the schema.

**Available Matchers:**
- 'exists' / 'not exists' — Field present / absent
- 'uuid', 'email', 'datetime', 'uri' — Format validators
- 'string', 'number', 'boolean', 'array', 'object', 'null', 'true', 'false' — Type checkers
- 'empty' — String, array, or object is empty
- 'contains <substring>', 'startsWith <prefix>', 'endsWith <suffix>' — String matchers
- 'regex <pattern>' — Regex match
- 'gt <N>', 'gte <N>', 'lt <N>', 'lte <N>' — Numeric comparisons
- 'ipv4', 'ipv6', 'base64', 'mac' — Format validators

**Collection Matchers (for arrays):**
- 'count(<path>): <N>' — Array has exactly N items (e.g. 'count(body.items): 3')
- 'count(<path>).gte: <N>' — Array has >= N items (e.g. 'count(body.items).gte: 1' means "has data")
- 'count(<path>).gt: <N>' — Array has > N items
- 'count(<path>).lte: <N>' — Array has <= N items
- 'count(<path>).lt: <N>' — Array has < N items
- 'all(<path>): <matcher>' — Every element matches (e.g. 'all(body.items.status): active')
- 'all(<path>.<field>): <matcher>' — Every element's field matches (e.g. 'all(body.items.id): uuid')

**Examples:**

    expect:
      status: 200
      body.data: exists
      body.token: uuid
      body.items: array
      body.email: email
      body.role: admin          # literal equality
      body.count: gt 10         # numeric > 10
      body.name: contains John
      count(body.items): 5      # exactly 5 items
      count(body.items).gte: 1  # at least 1 item (has data)
      schema: user-profile
      schema: not error-payload
`
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
		content = `# Basic example: login with inline credentials
scenario: login and fetch profile

steps:
  - request:
      method: POST
      url: /auth/login
      body:
        username: emilys
        password: emilyspass
    expect:
      status: 200
      body.accessToken: exists
    save:
      authToken: body.accessToken

  - request:
      method: GET
      url: /auth/me
      headers:
        Authorization: Bearer $authToken
    expect:
      status: 200
      body.username: emilys

---
# Multi-account example: access any account without --account flag
# Uses $accounts.<name>.<field> from .gherkio/credentials/local.yaml
scenario: login as specific account via $accounts

steps:
  - request:
      method: POST
      url: /auth/login
      body:
        username: $accounts.eka.username
        password: $accounts.eka.password
        expiresInMins: 30
    expect:
      status: 200
      body.accessToken: exists
    save:
      accessToken: body.accessToken

  - request:
      method: GET
      url: /auth/me
      headers:
        Authorization: Bearer $accessToken
    expect:
      status: 200
      body.role: $accounts.default.role

---
# Built-in generators example: $uuid, $ulid, $randomInt
# These variables are available in every test with no setup needed
scenario: using built-in generators

steps:
  - request:
      method: POST
      url: /auth/login
      body:
        username: $accounts.default.username
        password: $accounts.default.password
        idempotencyKey: $uuid
        requestId: $ulid
        otpCode: $randomInt
    expect:
      status: 200
      body.accessToken: exists
    save:
      accessToken: body.accessToken

  - request:
      method: GET
      url: /auth/me
      headers:
        Authorization: Bearer $accessToken
        X-Idempotency: $uuid
    expect:
      status: 200
      body.username: $accounts.default.username

---
# Parametrized randomInt example: custom range with ${randomInt(min,max)}
# Also demonstrates count().gte for checking array has data
scenario: parametrized randomInt and array length check

steps:
  - request:
      method: POST
      url: /products
      body:
        name: "Product ${randomInt(1000,9999)}"
        price: ${randomInt(1000,500000)}
        quantity: ${randomInt(1,100)}
    expect:
      status: 201
      count(body.items).gte: 1
    save:
      productId: body.id
      sku: body.sku
`
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
