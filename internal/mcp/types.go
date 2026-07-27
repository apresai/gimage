package mcp

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      interface{}            `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      interface{}            `json:"id"`
	Result  map[string]interface{} `json:"result,omitempty"`
	Error   *JSONRPCError          `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// MCP Protocol Methods
const (
	MethodInitialize    = "initialize"
	MethodListTools     = "tools/list"
	MethodCallTool      = "tools/call"
	MethodListResources = "resources/list"
	MethodReadResource  = "resources/read"
	MethodListPrompts   = "prompts/list"
	MethodGetPrompt     = "prompts/get"
)

// MCP Protocol Version
const ProtocolVersion = "2024-11-05"

// JSON-RPC Error Codes
const (
	ErrorCodeParseError     = -32700
	ErrorCodeInvalidRequest = -32600
	ErrorCodeMethodNotFound = -32601
	ErrorCodeInvalidParams  = -32602
	ErrorCodeInternalError  = -32603
)

// ToolError represents an error from tool execution
type ToolError struct {
	Code    int
	Message string
}

func (e *ToolError) Error() string {
	return e.Message
}

// ToolAnnotations provides hints to LLMs about tool behavior (MCP spec 2025-06-18)
type ToolAnnotations struct {
	// DestructiveHint indicates if the tool makes destructive changes (deletes, overwrites)
	DestructiveHint bool `json:"destructiveHint,omitempty"`

	// IdempotentHint indicates if calling the tool multiple times with same args has same effect
	IdempotentHint bool `json:"idempotentHint,omitempty"`

	// ReadOnlyHint indicates if the tool only reads data without making changes
	ReadOnlyHint bool `json:"readOnlyHint,omitempty"`
}

// Tool represents an MCP tool
type Tool struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
	Annotations *ToolAnnotations // Optional tool annotations (MCP spec 2025-06-18)
	Handler     ToolHandler
}

// ToolHandler is a function that handles tool execution
type ToolHandler func(args map[string]interface{}) (map[string]interface{}, error)
