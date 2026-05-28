package mcp

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
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
