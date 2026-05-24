package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/muhfaris/gherkio/internal/core/envstore"
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
							"description": "Optional account name from environments credentials to use for dynamic variable injection.",
						},
						"step": map[string]interface{}{
							"type":        "integer",
							"description": "Step index to execute in isolation (0-indexed). Defaults to -1 (run entire scenario).",
						},
					},
					Required: []string{"path"},
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

		if path == "" {
			s.writeToolError(id, "Missing required argument 'path'")
			return
		}

		fullPath := s.resolvePath(path)
		stepIndex := -1
		if stepVal != 0 {
			stepIndex = int(stepVal)
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

		creds, _ := runner.LoadCredentials(s.projectDir, envName)
		if creds != nil {
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
			Verbose:        true,
			MaskFields:     maskFields,
			AccountName:    accountName,
			CredentialVars: credentialVars,
			StepIndex:      stepIndex,
		}

		result, err := runner.Run(cfg)
		if err != nil {
			s.writeToolError(id, fmt.Sprintf("Execution execution failed: %v", err))
			return
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		s.writeToolResponse(id, string(data))

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
- **save**: (Map of name:path, Optional) Extract dynamic values to context variables.
- **timing**: (TimingConfig, Optional) Execution latency check.

### Request Config
- **service**: (String, Optional) Named service override matching environments.
- **method**: (String, Required) HTTP Method (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS).
- **url**: (String, Required) Target endpoint url (appends to baseUrl).
- **headers**: (Map of string:string, Optional) Custom HTTP headers.
- **body**: (Free-form object/string, Optional) Request body content.

### Assertions (Expect)
- **status**: (Integer) Expected HTTP status.
- **body.<path>**: Assert matches on JSON body values (e.g. body.id: exists).
- **headers.<name>**: Assert matches on response header keys.
- **jwt.<claim>**: Assert on decoded JWT values.
- **schema**: Name of a custom validator schema file in .gherkio/schemas/.
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
		content = `scenario: login and fetch profile

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
`
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
