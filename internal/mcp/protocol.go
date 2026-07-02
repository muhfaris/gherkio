package mcp

import "encoding/json"

// Standard JSON-RPC 2.0 error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// RPCRequest represents an incoming JSON-RPC request frame.
type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"` // Can be string, number, or null
}

// RPCResponse represents an outgoing JSON-RPC response frame.
type RPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// RPCError represents a JSON-RPC error details payload.
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// InitializeResult is returned on successful MCP handshake.
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
}

// ServerCapabilities defines supported MCP endpoints.
type ServerCapabilities struct {
	Resources struct{} `json:"resources"`
	Tools     struct{} `json:"tools"`
	Prompts   struct{} `json:"prompts"`
}

// Implementation identifies the server software name and version.
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ListToolsResult lists all supported server tools.
type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

// Tool describes a single executable LLM tool.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema defines the JSON schema arguments of a tool.
type InputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Required   []string               `json:"required,omitempty"`
}

// CallToolRequest represents the parameters for calling a tool.
type CallToolRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// CallToolResult represents the outcome of executing a tool.
type CallToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content holds a single content return block.
type Content struct {
	Type string `json:"type"` // e.g. "text"
	Text string `json:"text"`
}

// ListResourcesResult lists all discoverable resources.
type ListResourcesResult struct {
	Resources []Resource `json:"resources"`
}

// Resource represents a read-only data uri.
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ReadResourceResult returns the text content of a resource.
type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents"`
}

// ResourceContent details single resource content bytes.
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text"`
}

// Prompt describes a reusable prompt template.
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument describes a parameter for a prompt template.
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required,omitempty"`
}

// ListPromptsResult is returned on prompts/list.
type ListPromptsResult struct {
	Prompts []Prompt `json:"prompts"`
}

// GetPromptRequest represents parameters for fetching a prompt.
type GetPromptRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// GetPromptResult is returned on prompts/get.
type GetPromptResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// PromptMessage is a single message in a prompt template.
type PromptMessage struct {
	Role    string        `json:"role"`
	Content PromptContent `json:"content"`
}

// PromptContent holds the content of a prompt message.
type PromptContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
