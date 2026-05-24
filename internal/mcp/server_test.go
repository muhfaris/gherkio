package mcp

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
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

	// Capture all output
	outWrite.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, outRead)

	return buf.Bytes(), <-errChan
}
