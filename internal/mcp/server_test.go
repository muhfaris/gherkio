package mcp

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMCPServerHandshakeAndList(t *testing.T) {
	// Setup standard input and output pipes/buffers
	inRead, inWrite, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create input pipe: %v", err)
	}
	defer inRead.Close()
	defer inWrite.Close()

	outRead, outWrite, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create output pipe: %v", err)
	}
	defer outRead.Close()
	defer outWrite.Close()

	// Redirect standard I/O for testing
	oldStdin := os.Stdin
	oldStdout := os.Stdout
	defer func() {
		os.Stdin = oldStdin
		os.Stdout = oldStdout
	}()

	os.Stdin = inRead
	os.Stdout = outWrite

	// Instantiate server
	tmpDir := t.TempDir()
	srv := NewServer(tmpDir)

	// Run Server.Start in a background goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start()
	}()

	// 1. Send JSON-RPC initialize request
	initReq := RPCRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		ID:      1,
	}
	initData, _ := json.Marshal(initReq)
	_, _ = inWrite.Write(initData)
	_, _ = inWrite.Write([]byte("\n"))

	// Read response
	var initResp RPCResponse
	dec := json.NewDecoder(outRead)
	if err := dec.Decode(&initResp); err != nil {
		t.Fatalf("failed to decode initialize response: %v", err)
	}

	if initResp.JSONRPC != "2.0" {
		t.Errorf("expected JSONRPC 2.0, got '%s'", initResp.JSONRPC)
	}
	if initResp.Error != nil {
		t.Fatalf("handshake returned error: %+v", initResp.Error)
	}

	// 2. Send tools/list request
	toolsReq := RPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      2,
	}
	toolsData, _ := json.Marshal(toolsReq)
	_, _ = inWrite.Write(toolsData)
	_, _ = inWrite.Write([]byte("\n"))

	var toolsResp RPCResponse
	if err := dec.Decode(&toolsResp); err != nil {
		t.Fatalf("failed to decode tools/list response: %v", err)
	}

	if toolsResp.Error != nil {
		t.Fatalf("tools/list returned error: %+v", toolsResp.Error)
	}

	resultJSON, err := json.Marshal(toolsResp.Result)
	if err != nil {
		t.Fatalf("failed to encode tools/list result: %v", err)
	}
	var toolsResult struct {
		Tools []struct {
			Name        string `json:"name"`
			InputSchema struct {
				AdditionalProperties *bool                      `json:"additionalProperties"`
				Properties           map[string]json.RawMessage `json:"properties"`
				Required             *[]string                  `json:"required"`
			} `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(resultJSON, &toolsResult); err != nil {
		t.Fatalf("failed to decode tools/list result: %v", err)
	}
	for _, tool := range toolsResult.Tools {
		if tool.InputSchema.AdditionalProperties == nil {
			t.Errorf("tool %q input schema is missing additionalProperties", tool.Name)
			continue
		}
		if *tool.InputSchema.AdditionalProperties {
			t.Errorf("tool %q input schema allows additional properties", tool.Name)
		}
		if tool.InputSchema.Required == nil {
			t.Errorf("tool %q input schema is missing required", tool.Name)
			continue
		}
		required := make(map[string]bool, len(*tool.InputSchema.Required))
		for _, name := range *tool.InputSchema.Required {
			required[name] = true
		}
		for name := range tool.InputSchema.Properties {
			if !required[name] {
				t.Errorf("tool %q required does not include property %q", tool.Name, name)
			}
		}
		if tool.Name == "init_project" {
			var pathSchema struct {
				Type []string `json:"type"`
			}
			if err := json.Unmarshal(tool.InputSchema.Properties["path"], &pathSchema); err != nil {
				t.Fatalf("init_project optional path is not nullable: %v", err)
			}
			if len(pathSchema.Type) != 2 || pathSchema.Type[0] != "string" || pathSchema.Type[1] != "null" {
				t.Errorf("init_project path type = %v, want [string null]", pathSchema.Type)
			}
		}
	}

	// Terminate stdio loop gracefully by closing stdin pipe
	inWrite.Close()

	// Ensure server terminates cleanly
	serverErr := <-errChan
	if serverErr != nil && serverErr != io.EOF {
		t.Errorf("server terminated with unexpected error: %v", serverErr)
	}
}

type RawResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
	ID      interface{}     `json:"id"`
}
type RawRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

func mockServerIO(inData []byte, projectDir string) ([]byte, error) {
	// Create pipes
	inRead, inWrite, _ := os.Pipe()
	outRead, outWrite, _ := os.Pipe()

	oldStdin := os.Stdin
	oldStdout := os.Stdout
	defer func() {
		os.Stdin = oldStdin
		os.Stdout = oldStdout
		inRead.Close()
		outRead.Close()
	}()

	os.Stdin = inRead
	os.Stdout = outWrite

	srv := NewServer(projectDir)
	go func() {
		_, _ = inWrite.Write(inData)
		_, _ = inWrite.Write([]byte("\n"))
		inWrite.Close()
	}()

	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start()
	}()

	// Wait for server to finish processing and exit
	serverErr := <-errChan

	// Close write end of the pipe to unblock io.Copy
	outWrite.Close()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, outRead)

	return buf.Bytes(), serverErr
}

func TestMCPInitProject(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Prepare JSON-RPC tools/call request for init_project
	callReq := RawRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name": "init_project"}`),
		ID:      1,
	}
	reqData, _ := json.Marshal(callReq)

	// Since we pass s.projectDir as tmpDir, calling init_project should initialize it inside tmpDir
	outBytes, err := mockServerIO(reqData, tmpDir)
	if err != nil && err != io.EOF {
		t.Fatalf("server execution failed: %v", err)
	}

	var resp RawResponse
	if err := json.Unmarshal(outBytes, &resp); err != nil {
		t.Fatalf("failed to decode response: %v, raw output: %s", err, string(outBytes))
	}

	if resp.Error != nil {
		t.Fatalf("init_project returned error: %+v", resp.Error)
	}

	var toolResult CallToolResult
	if err := json.Unmarshal(resp.Result, &toolResult); err != nil {
		t.Fatalf("failed to decode tool result: %v", err)
	}

	if toolResult.IsError {
		t.Fatalf("tool execution returned error content: %+v", toolResult.Content)
	}

	// Verify that the .gherkio/config.yaml is successfully created
	configPath := filepath.Join(tmpDir, ".gherkio", "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("expected config.yaml to exist at %s, but got error: %v", configPath, err)
	}
}

func TestMCPNewTools(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup a Gherkio project manually
	gDir := filepath.Join(tmpDir, ".gherkio")
	testsDir := filepath.Join(gDir, "tests")
	schemasDir := filepath.Join(gDir, "schemas")
	_ = os.MkdirAll(testsDir, 0755)
	_ = os.MkdirAll(schemasDir, 0755)
	_ = os.WriteFile(filepath.Join(gDir, "config.yaml"), []byte("environments:\n  default: local\n"), 0644)

	// Write a simple test file to testsDir
	testYaml := `scenario: User Login
steps:
  - request:
      method: POST
      url: https://api.example.com/login
`
	_ = os.WriteFile(filepath.Join(testsDir, "login.yaml"), []byte(testYaml), 0644)

	// 1. Test convert_curl_to_yaml
	curlReq := RawRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name": "convert_curl_to_yaml", "arguments": {"curl": "curl -X POST https://api.example.com/login"}}`),
		ID:      1,
	}
	reqData, _ := json.Marshal(curlReq)
	outBytes, err := mockServerIO(reqData, tmpDir)
	if err != nil && err != io.EOF {
		t.Fatalf("server execution failed for convert_curl_to_yaml: %v", err)
	}

	var resp RawResponse
	if err := json.Unmarshal(outBytes, &resp); err != nil {
		t.Fatalf("failed to decode response: %v, raw output: %s", err, string(outBytes))
	}
	if resp.Error != nil {
		t.Fatalf("convert_curl_to_yaml returned error: %+v", resp.Error)
	}

	// 2. Test convert_yaml_to_curl
	yamlToCurlReq := RawRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name": "convert_yaml_to_curl", "arguments": {"path": "login.yaml"}}`),
		ID:      2,
	}
	reqData2, _ := json.Marshal(yamlToCurlReq)
	outBytes2, err := mockServerIO(reqData2, tmpDir)
	if err != nil && err != io.EOF {
		t.Fatalf("server execution failed for convert_yaml_to_curl: %v", err)
	}
	var resp2 RawResponse
	if err := json.Unmarshal(outBytes2, &resp2); err != nil {
		t.Fatalf("failed to decode response: %v, raw output: %s", err, string(outBytes2))
	}
	if resp2.Error != nil {
		t.Fatalf("convert_yaml_to_curl returned error: %+v", resp2.Error)
	}

	// 3. Test validate_workspace
	validateReq := RawRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name": "validate_workspace"}`),
		ID:      3,
	}
	reqData3, _ := json.Marshal(validateReq)
	outBytes3, err := mockServerIO(reqData3, tmpDir)
	if err != nil && err != io.EOF {
		t.Fatalf("server execution failed for validate_workspace: %v", err)
	}
	var resp3 RawResponse
	if err := json.Unmarshal(outBytes3, &resp3); err != nil {
		t.Fatalf("failed to decode response: %v, raw output: %s", err, string(outBytes3))
	}
	if resp3.Error != nil {
		t.Fatalf("validate_workspace returned error: %+v", resp3.Error)
	}

	// 4. Test convert_curl_to_yaml (negative case: invalid curl)
	invalidCurlReq := RawRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name": "convert_curl_to_yaml", "arguments": {"curl": "curl -X"}}`),
		ID:      4,
	}
	reqData4, _ := json.Marshal(invalidCurlReq)
	outBytes4, err := mockServerIO(reqData4, tmpDir)
	if err != nil && err != io.EOF {
		t.Fatalf("server execution failed: %v", err)
	}
	var resp4 RawResponse
	if err := json.Unmarshal(outBytes4, &resp4); err != nil {
		t.Fatalf("failed to decode response: %v, raw output: %s", err, string(outBytes4))
	}
	var toolResult4 CallToolResult
	_ = json.Unmarshal(resp4.Result, &toolResult4)
	if !toolResult4.IsError {
		t.Error("expected convert_curl_to_yaml to return IsError: true for invalid command")
	}

	// 5. Test convert_yaml_to_curl (negative case: missing file)
	invalidYamlToCurlReq := RawRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name": "convert_yaml_to_curl", "arguments": {"path": "missing.yaml"}}`),
		ID:      5,
	}
	reqData5, _ := json.Marshal(invalidYamlToCurlReq)
	outBytes5, err := mockServerIO(reqData5, tmpDir)
	if err != nil && err != io.EOF {
		t.Fatalf("server execution failed: %v", err)
	}
	var resp5 RawResponse
	if err := json.Unmarshal(outBytes5, &resp5); err != nil {
		t.Fatalf("failed to decode response: %v, raw output: %s", err, string(outBytes5))
	}
	var toolResult5 CallToolResult
	_ = json.Unmarshal(resp5.Result, &toolResult5)
	if !toolResult5.IsError {
		t.Error("expected convert_yaml_to_curl to return IsError: true for non-existent file")
	}

	// 6. Test convert_yaml_to_curl (negative case: out of bounds step)
	invalidStepYamlToCurlReq := RawRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name": "convert_yaml_to_curl", "arguments": {"path": "login.yaml", "step": 99}}`),
		ID:      6,
	}
	reqData6, _ := json.Marshal(invalidStepYamlToCurlReq)
	outBytes6, err := mockServerIO(reqData6, tmpDir)
	if err != nil && err != io.EOF {
		t.Fatalf("server execution failed: %v", err)
	}
	var resp6 RawResponse
	if err := json.Unmarshal(outBytes6, &resp6); err != nil {
		t.Fatalf("failed to decode response: %v, raw output: %s", err, string(outBytes6))
	}
	var toolResult6 CallToolResult
	_ = json.Unmarshal(resp6.Result, &toolResult6)
	if !toolResult6.IsError {
		t.Error("expected convert_yaml_to_curl to return IsError: true for out of bounds step index")
	}
}

func TestMCPRunTestStepZeroIsolation(t *testing.T) {
	tmpDir := t.TempDir()

	gDir := filepath.Join(tmpDir, ".gherkio")
	testsDir := filepath.Join(gDir, "tests")
	envDir := filepath.Join(gDir, "environments")
	_ = os.MkdirAll(testsDir, 0755)
	_ = os.MkdirAll(envDir, 0755)
	_ = os.WriteFile(filepath.Join(gDir, "config.yaml"), []byte("environments:\n  default: local\n"), 0644)
	_ = os.WriteFile(filepath.Join(envDir, "local.yaml"), []byte("baseUrl: https://api.example.com\n"), 0644)

	// Two steps so we can verify step: 0 isolates only the first (0-indexed).
	testYaml := `scenario: User Login
steps:
  - request:
      method: POST
      url: https://api.example.com/login
  - request:
      method: GET
      url: https://api.example.com/me
`
	_ = os.WriteFile(filepath.Join(testsDir, "login.yaml"), []byte(testYaml), 0644)

	runReq := RawRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name": "run_test", "arguments": {"path": "login.yaml", "step": 0, "dryRun": true}}`),
		ID:      1,
	}
	reqData, _ := json.Marshal(runReq)
	outBytes, err := mockServerIO(reqData, tmpDir)
	if err != nil && err != io.EOF {
		t.Fatalf("server execution failed: %v", err)
	}

	var resp RawResponse
	if err := json.Unmarshal(outBytes, &resp); err != nil {
		t.Fatalf("failed to decode response: %v, raw output: %s", err, string(outBytes))
	}
	if resp.Error != nil {
		t.Fatalf("run_test returned error: %+v", resp.Error)
	}

	var toolResult CallToolResult
	if err := json.Unmarshal(resp.Result, &toolResult); err != nil {
		t.Fatalf("failed to decode tool result: %v", err)
	}
	if toolResult.IsError {
		t.Fatalf("run_test reported error content: %+v", toolResult.Content)
	}
	if len(toolResult.Content) == 0 {
		t.Fatal("run_test returned no content")
	}

	// The text payload is the marshaled runner.RunResult — assert only one step ran.
	var runResult struct {
		Steps []json.RawMessage `json:"steps"`
	}
	if err := json.Unmarshal([]byte(toolResult.Content[0].Text), &runResult); err != nil {
		t.Fatalf("failed to decode run result: %v, payload: %s", err, toolResult.Content[0].Text)
	}
	if len(runResult.Steps) != 1 {
		t.Errorf("expected exactly 1 step for step: 0, got %d", len(runResult.Steps))
	}
}

func TestMCPPlanScenarioPromptGuidesPayloadVariants(t *testing.T) {
	promptReq := RawRequest{
		JSONRPC: "2.0",
		Method:  "prompts/get",
		Params:  json.RawMessage(`{"name": "plan-scenario", "arguments": {"endpoint": "/orders"}}`),
		ID:      1,
	}
	reqData, _ := json.Marshal(promptReq)
	outBytes, err := mockServerIO(reqData, t.TempDir())
	if err != nil && err != io.EOF {
		t.Fatalf("server execution failed: %v", err)
	}

	var resp RawResponse
	if err := json.Unmarshal(outBytes, &resp); err != nil {
		t.Fatalf("failed to decode response: %v, raw output: %s", err, string(outBytes))
	}
	if resp.Error != nil {
		t.Fatalf("prompts/get returned error: %+v", resp.Error)
	}

	var result GetPromptResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("failed to decode prompt result: %v", err)
	}

	// Concatenate all message text for keyword assertions.
	var all string
	for _, m := range result.Messages {
		all += m.Content.Text
	}

	for _, kw := range []string{"PAYLOAD VARIANTS", "create_test", "NEVER invent", "confirm", "CRUD"} {
		if !strings.Contains(all, kw) {
			t.Errorf("plan-scenario prompt missing expected keyword %q", kw)
		}
	}

	// validate_flow prompt must include review guidance keywords.
	vfReq := RawRequest{
		JSONRPC: "2.0",
		Method:  "prompts/get",
		Params:  json.RawMessage(`{"name": "validate_flow"}`),
		ID:      3,
	}
	vfData, _ := json.Marshal(vfReq)
	vfOut, err := mockServerIO(vfData, t.TempDir())
	if err != nil && err != io.EOF {
		t.Fatalf("server execution failed: %v", err)
	}
	var vfResp RawResponse
	if err := json.Unmarshal(vfOut, &vfResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if vfResp.Error != nil {
		t.Fatalf("validate_flow returned error: %+v", vfResp.Error)
	}
	var vfResult GetPromptResult
	if err := json.Unmarshal(vfResp.Result, &vfResult); err != nil {
		t.Fatalf("failed to decode validate_flow result: %v", err)
	}
	var vfAll string
	for _, m := range vfResult.Messages {
		vfAll += m.Content.Text
	}
	for _, kw := range []string{"save", "teardown", "assertion", "create_test", "confirm"} {
		if !strings.Contains(vfAll, kw) {
			t.Errorf("validate_flow prompt missing expected keyword %q", kw)
		}
	}

	// Unknown prompt must return an error.
	badReq := RawRequest{
		JSONRPC: "2.0",
		Method:  "prompts/get",
		Params:  json.RawMessage(`{"name": "does-not-exist"}`),
		ID:      2,
	}
	badData, _ := json.Marshal(badReq)
	badOut, err := mockServerIO(badData, t.TempDir())
	if err != nil && err != io.EOF {
		t.Fatalf("server execution failed: %v", err)
	}
	var badResp RawResponse
	if err := json.Unmarshal(badOut, &badResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if badResp.Error == nil {
		t.Error("expected unknown prompt to return a JSON-RPC error")
	}
}
