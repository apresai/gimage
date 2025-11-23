# Gimage MCP Server Implementation Plan

**Version**: 1.0.0
**Target Runtime**: Go 1.22+
**MCP Protocol**: Model Context Protocol (stdio transport)
**Deployment**: npm package + direct binary distribution

---

## 🎯 Project Overview

This document outlines the complete plan to wrap the gimage CLI tool as a Model Context Protocol (MCP) server, enabling AI assistants (Claude, ChatGPT, etc.) to perform AI-powered image generation and processing operations.

### Goals

1. **Wrap existing gimage CLI** - Reuse all existing functionality without duplication
2. **Follow MCP conventions** - Implement standard MCP server protocol
3. **Easy installation** - npm package + binary downloads from GitHub
4. **Production-ready** - Comprehensive error handling, logging, and testing
5. **LLM-optimized** - Clear tool descriptions, examples, and error messages

---

## 📋 Table of Contents

1. [Architecture](#architecture)
2. [MCP Tools](#mcp-tools)
3. [Implementation Plan](#implementation-plan)
4. [Project Structure](#project-structure)
5. [Installation Methods](#installation-methods)
6. [Configuration](#configuration)
7. [Testing Strategy](#testing-strategy)
8. [Documentation](#documentation)
9. [Deployment](#deployment)
10. [Timeline](#timeline)

---

## 🏗️ Architecture

### High-Level Design

```
┌─────────────────────────────────────────────────────────┐
│              AI Assistant (Claude/ChatGPT)              │
└────────────────────┬────────────────────────────────────┘
                     │ MCP Protocol (JSON-RPC over stdio)
                     ▼
┌─────────────────────────────────────────────────────────┐
│          Gimage MCP Server (Go Implementation)          │
│  ┌───────────────────────────────────────────────────┐  │
│  │   MCP Protocol Handler (stdio transport)         │  │
│  └───────────────────────┬───────────────────────────┘  │
│                          │                              │
│  ┌───────────────────────▼───────────────────────────┐  │
│  │        Tool Handlers (10 MCP tools)              │  │
│  │  • generate_image     • batch_resize             │  │
│  │  • resize_image       • batch_compress           │  │
│  │  • scale_image        • batch_convert            │  │
│  │  • crop_image         • get_image_info           │  │
│  │  • compress_image     • list_models              │  │
│  │  • convert_image                                 │  │
│  └───────────────────────┬───────────────────────────┘  │
│                          │                              │
│  ┌───────────────────────▼───────────────────────────┐  │
│  │      Internal Go Implementation                   │  │
│  │   (Reuse existing internal/ packages)            │  │
│  │   • internal/generate   • internal/imaging       │  │
│  │   • internal/config     • pkg/models             │  │
│  └───────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

### Why Go for MCP Server?

While most MCP servers use TypeScript/Node.js, implementing in Go provides:

1. **Native integration** - Direct access to all gimage internals
2. **Single binary** - No Node.js dependency, easier deployment
3. **Type safety** - Strong typing for all operations
4. **Performance** - Fast image processing without IPC overhead
5. **Consistency** - Same language as CLI tool

### MCP Protocol Implementation

The MCP server will use **stdio transport** (standard input/output) for communication:

- **Input**: JSON-RPC 2.0 messages via stdin
- **Output**: JSON-RPC 2.0 responses via stdout
- **Logging**: All logs go to stderr (never stdout)
- **Protocol**: Model Context Protocol specification

---

## 🛠️ MCP Tools

The MCP server will expose 10 tools covering all gimage operations:

### 1. generate_image

**Description**: Generate an AI image from a text prompt using Gemini or Vertex AI.

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "prompt": {
      "type": "string",
      "description": "Text description of the image to generate"
    },
    "output": {
      "type": "string",
      "description": "Output file path (default: auto-generated with timestamp)"
    },
    "size": {
      "type": "string",
      "enum": ["256x256", "512x512", "1024x1024", "1024x1792", "1792x1024", "2048x2048"],
      "description": "Image dimensions (default: 1024x1024)"
    },
    "model": {
      "type": "string",
      "enum": [
        "gemini-2.5-flash-image",
        "gemini-2.0-flash-preview-image-generation",
        "imagen-3.0-generate-002",
        "imagen-4"
      ],
      "description": "AI model to use (default: gemini-3-pro-image-preview)"
    },
    "style": {
      "type": "string",
      "enum": ["photorealistic", "artistic", "anime"],
      "description": "Image style (optional)"
    },
    "negative": {
      "type": "string",
      "description": "Negative prompt - things to avoid in the image"
    },
    "seed": {
      "type": "integer",
      "description": "Random seed for reproducible generation"
    }
  },
  "required": ["prompt"]
}
```

**Output Schema**:
```json
{
  "type": "object",
  "properties": {
    "success": { "type": "boolean" },
    "output_path": { "type": "string" },
    "size": { "type": "string" },
    "model": { "type": "string" },
    "error": { "type": "string" }
  }
}
```

**Example**:
```json
{
  "prompt": "a sunset over mountains with vibrant colors",
  "size": "1024x1024",
  "style": "photorealistic"
}
```

---

### 2. resize_image

**Description**: Resize an image to specific dimensions.

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "input": {
      "type": "string",
      "description": "Input image file path"
    },
    "width": {
      "type": "integer",
      "description": "Target width in pixels",
      "minimum": 1
    },
    "height": {
      "type": "integer",
      "description": "Target height in pixels",
      "minimum": 1
    },
    "output": {
      "type": "string",
      "description": "Output file path (default: auto-generated)"
    }
  },
  "required": ["input", "width", "height"]
}
```

**Output Schema**:
```json
{
  "type": "object",
  "properties": {
    "success": { "type": "boolean" },
    "output_path": { "type": "string" },
    "original_size": { "type": "string" },
    "new_size": { "type": "string" },
    "error": { "type": "string" }
  }
}
```

---

### 3. scale_image

**Description**: Scale an image by a factor (e.g., 0.5 for half size, 2.0 for double).

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "input": {
      "type": "string",
      "description": "Input image file path"
    },
    "factor": {
      "type": "number",
      "description": "Scale factor (0.1 to 10.0)",
      "minimum": 0.1,
      "maximum": 10.0
    },
    "output": {
      "type": "string",
      "description": "Output file path (default: auto-generated)"
    }
  },
  "required": ["input", "factor"]
}
```

**Output Schema**:
```json
{
  "type": "object",
  "properties": {
    "success": { "type": "boolean" },
    "output_path": { "type": "string" },
    "scale_factor": { "type": "number" },
    "original_size": { "type": "string" },
    "new_size": { "type": "string" },
    "error": { "type": "string" }
  }
}
```

---

### 4. crop_image

**Description**: Crop an image to a specific region.

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "input": {
      "type": "string",
      "description": "Input image file path"
    },
    "x": {
      "type": "integer",
      "description": "X coordinate of top-left corner",
      "minimum": 0
    },
    "y": {
      "type": "integer",
      "description": "Y coordinate of top-left corner",
      "minimum": 0
    },
    "width": {
      "type": "integer",
      "description": "Width of crop region",
      "minimum": 1
    },
    "height": {
      "type": "integer",
      "description": "Height of crop region",
      "minimum": 1
    },
    "output": {
      "type": "string",
      "description": "Output file path (default: auto-generated)"
    }
  },
  "required": ["input", "x", "y", "width", "height"]
}
```

**Output Schema**:
```json
{
  "type": "object",
  "properties": {
    "success": { "type": "boolean" },
    "output_path": { "type": "string" },
    "crop_region": { "type": "string" },
    "error": { "type": "string" }
  }
}
```

---

### 5. compress_image

**Description**: Compress an image to reduce file size.

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "input": {
      "type": "string",
      "description": "Input image file path"
    },
    "quality": {
      "type": "integer",
      "description": "Compression quality (1-100, default: 90)",
      "minimum": 1,
      "maximum": 100
    },
    "output": {
      "type": "string",
      "description": "Output file path (default: auto-generated)"
    }
  },
  "required": ["input"]
}
```

**Output Schema**:
```json
{
  "type": "object",
  "properties": {
    "success": { "type": "boolean" },
    "output_path": { "type": "string" },
    "original_size_bytes": { "type": "integer" },
    "compressed_size_bytes": { "type": "integer" },
    "compression_ratio": { "type": "number" },
    "error": { "type": "string" }
  }
}
```

---

### 6. convert_image

**Description**: Convert an image between formats (PNG, JPG, WebP, GIF, TIFF, BMP).

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "input": {
      "type": "string",
      "description": "Input image file path"
    },
    "format": {
      "type": "string",
      "enum": ["png", "jpg", "jpeg", "webp", "gif", "tiff", "bmp"],
      "description": "Target image format"
    },
    "output": {
      "type": "string",
      "description": "Output file path (default: auto-generated)"
    }
  },
  "required": ["input", "format"]
}
```

**Output Schema**:
```json
{
  "type": "object",
  "properties": {
    "success": { "type": "boolean" },
    "output_path": { "type": "string" },
    "original_format": { "type": "string" },
    "new_format": { "type": "string" },
    "error": { "type": "string" }
  }
}
```

---

### 7. batch_resize

**Description**: Resize multiple images concurrently.

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "input_dir": {
      "type": "string",
      "description": "Input directory containing images"
    },
    "width": {
      "type": "integer",
      "description": "Target width in pixels",
      "minimum": 1
    },
    "height": {
      "type": "integer",
      "description": "Target height in pixels",
      "minimum": 1
    },
    "output_dir": {
      "type": "string",
      "description": "Output directory for resized images"
    },
    "workers": {
      "type": "integer",
      "description": "Number of parallel workers (default: 4)",
      "minimum": 1,
      "maximum": 16
    }
  },
  "required": ["input_dir", "width", "height", "output_dir"]
}
```

**Output Schema**:
```json
{
  "type": "object",
  "properties": {
    "success": { "type": "boolean" },
    "processed": { "type": "integer" },
    "failed": { "type": "integer" },
    "output_dir": { "type": "string" },
    "errors": { "type": "array", "items": { "type": "string" } }
  }
}
```

---

### 8. batch_compress

**Description**: Compress multiple images concurrently.

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "input_dir": {
      "type": "string",
      "description": "Input directory containing images"
    },
    "quality": {
      "type": "integer",
      "description": "Compression quality (1-100)",
      "minimum": 1,
      "maximum": 100
    },
    "output_dir": {
      "type": "string",
      "description": "Output directory for compressed images"
    },
    "workers": {
      "type": "integer",
      "description": "Number of parallel workers (default: 4)",
      "minimum": 1,
      "maximum": 16
    }
  },
  "required": ["input_dir", "output_dir"]
}
```

**Output Schema**:
```json
{
  "type": "object",
  "properties": {
    "success": { "type": "boolean" },
    "processed": { "type": "integer" },
    "failed": { "type": "integer" },
    "total_savings_bytes": { "type": "integer" },
    "output_dir": { "type": "string" },
    "errors": { "type": "array", "items": { "type": "string" } }
  }
}
```

---

### 9. batch_convert

**Description**: Convert multiple images to a different format concurrently.

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "input_dir": {
      "type": "string",
      "description": "Input directory containing images"
    },
    "format": {
      "type": "string",
      "enum": ["png", "jpg", "jpeg", "webp", "gif", "tiff", "bmp"],
      "description": "Target image format"
    },
    "output_dir": {
      "type": "string",
      "description": "Output directory for converted images"
    },
    "workers": {
      "type": "integer",
      "description": "Number of parallel workers (default: 4)",
      "minimum": 1,
      "maximum": 16
    }
  },
  "required": ["input_dir", "format", "output_dir"]
}
```

**Output Schema**:
```json
{
  "type": "object",
  "properties": {
    "success": { "type": "boolean" },
    "processed": { "type": "integer" },
    "failed": { "type": "integer" },
    "output_dir": { "type": "string" },
    "errors": { "type": "array", "items": { "type": "string" } }
  }
}
```

---

### 10. list_models

**Description**: List all available AI image generation models.

**Input Schema**:
```json
{
  "type": "object",
  "properties": {}
}
```

**Output Schema**:
```json
{
  "type": "object",
  "properties": {
    "models": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": { "type": "string" },
          "provider": { "type": "string" },
          "description": { "type": "string" },
          "max_resolution": { "type": "string" },
          "requires_api_key": { "type": "boolean" }
        }
      }
    }
  }
}
```

---

## 🔧 Implementation Plan

### Phase 1: Core MCP Server Infrastructure (Week 1)

#### 1.1 Create MCP Server Package

**File**: `internal/mcp/server.go`

```go
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/apresai/gimage/internal/config"
)

// MCPServer implements the Model Context Protocol server
type MCPServer struct {
	name    string
	version string
	config  *config.Config
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
	tools   map[string]Tool
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(name, version string, cfg *config.Config) *MCPServer {
	return &MCPServer{
		name:    name,
		version: version,
		config:  cfg,
		stdin:   os.Stdin,
		stdout:  os.Stdout,
		stderr:  os.Stderr,
		tools:   make(map[string]Tool),
	}
}

// Tool represents an MCP tool
type Tool struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
	Handler     func(context.Context, map[string]interface{}) (map[string]interface{}, error)
}

// RegisterTool adds a tool to the server
func (s *MCPServer) RegisterTool(tool Tool) {
	s.tools[tool.Name] = tool
}

// Start begins listening for MCP protocol messages
func (s *MCPServer) Start(ctx context.Context) error {
	scanner := bufio.NewScanner(s.stdin)

	for scanner.Scan() {
		line := scanner.Bytes()

		var request JSONRPCRequest
		if err := json.Unmarshal(line, &request); err != nil {
			s.logError("Failed to parse request: %v", err)
			continue
		}

		response := s.handleRequest(ctx, &request)

		responseBytes, err := json.Marshal(response)
		if err != nil {
			s.logError("Failed to marshal response: %v", err)
			continue
		}

		fmt.Fprintln(s.stdout, string(responseBytes))
	}

	return scanner.Err()
}

func (s *MCPServer) logError(format string, args ...interface{}) {
	fmt.Fprintf(s.stderr, "[gimage-mcp] "+format+"\n", args...)
}
```

#### 1.2 JSON-RPC Message Types

**File**: `internal/mcp/types.go`

```go
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
	MethodInitialize       = "initialize"
	MethodListTools        = "tools/list"
	MethodCallTool         = "tools/call"
	MethodListResources    = "resources/list"
	MethodReadResource     = "resources/read"
	MethodListPrompts      = "prompts/list"
	MethodGetPrompt        = "prompts/get"
)
```

#### 1.3 Request Handler

**File**: `internal/mcp/handler.go`

```go
package mcp

import (
	"context"
)

func (s *MCPServer) handleRequest(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case MethodInitialize:
		return s.handleInitialize(req)
	case MethodListTools:
		return s.handleListTools(req)
	case MethodCallTool:
		return s.handleCallTool(ctx, req)
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32601,
				Message: "Method not found",
			},
		}
	}
}

func (s *MCPServer) handleInitialize(req *JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]interface{}{
				"name":    s.name,
				"version": s.version,
			},
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
		},
	}
}

func (s *MCPServer) handleListTools(req *JSONRPCRequest) *JSONRPCResponse {
	tools := make([]map[string]interface{}, 0, len(s.tools))

	for _, tool := range s.tools {
		tools = append(tools, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		})
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

func (s *MCPServer) handleCallTool(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	// Extract tool name and arguments
	name, ok := req.Params["name"].(string)
	if !ok {
		return s.errorResponse(req.ID, -32602, "Invalid params: missing tool name")
	}

	arguments, ok := req.Params["arguments"].(map[string]interface{})
	if !ok {
		arguments = make(map[string]interface{})
	}

	// Find tool
	tool, exists := s.tools[name]
	if !exists {
		return s.errorResponse(req.ID, -32602, fmt.Sprintf("Tool not found: %s", name))
	}

	// Execute tool
	result, err := tool.Handler(ctx, arguments)
	if err != nil {
		return s.errorResponse(req.ID, -32603, err.Error())
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": formatToolResult(result),
				},
			},
		},
	}
}

func (s *MCPServer) errorResponse(id interface{}, code int, message string) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
	}
}

func formatToolResult(result map[string]interface{}) string {
	// Format result as readable text for LLM
	data, _ := json.MarshalIndent(result, "", "  ")
	return string(data)
}
```

---

### Phase 2: Tool Implementations (Week 2-3)

#### 2.1 Image Generation Tool

**File**: `internal/mcp/tools/generate.go`

```go
package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/apresai/gimage/internal/generate"
	"github.com/apresai/gimage/internal/mcp"
	"github.com/apresai/gimage/pkg/models"
)

// RegisterGenerateImageTool registers the generate_image tool
func RegisterGenerateImageTool(server *mcp.MCPServer, cfg *config.Config) {
	tool := mcp.Tool{
		Name:        "generate_image",
		Description: "Generate an AI image from a text prompt using Gemini or Vertex AI. Supports multiple models, sizes, and styles.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"prompt": map[string]interface{}{
					"type":        "string",
					"description": "Text description of the image to generate. Be specific and descriptive.",
				},
				"output": map[string]interface{}{
					"type":        "string",
					"description": "Output file path. If not provided, auto-generates with timestamp.",
				},
				"size": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"256x256", "512x512", "1024x1024", "1024x1792", "1792x1024", "2048x2048"},
					"description": "Image dimensions (default: 1024x1024)",
				},
				"model": map[string]interface{}{
					"type": "string",
					"enum": []string{
						"gemini-2.5-flash-image",
						"gemini-2.0-flash-preview-image-generation",
						"imagen-3.0-generate-002",
						"imagen-4",
					},
					"description": "AI model to use (default: gemini-3-pro-image-preview)",
				},
				"style": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"photorealistic", "artistic", "anime"},
					"description": "Image style (optional)",
				},
				"negative": map[string]interface{}{
					"type":        "string",
					"description": "Negative prompt - describe what you DON'T want in the image",
				},
				"seed": map[string]interface{}{
					"type":        "integer",
					"description": "Random seed for reproducible generation (optional)",
				},
			},
			"required": []string{"prompt"},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			// Extract parameters
			prompt, _ := args["prompt"].(string)
			if prompt == "" {
				return nil, fmt.Errorf("prompt is required")
			}

			output, _ := args["output"].(string)
			if output == "" {
				output = fmt.Sprintf("generated_%d.png", time.Now().Unix())
			}

			size, _ := args["size"].(string)
			if size == "" {
				size = "1024x1024"
			}

			modelName, _ := args["model"].(string)
			if modelName == "" {
				modelName = "gemini-2.5-flash-image"
			}

			style, _ := args["style"].(string)
			negative, _ := args["negative"].(string)
			seed, _ := args["seed"].(float64) // JSON numbers are float64

			// Create generate options
			opts := models.GenerateOptions{
				Prompt:   prompt,
				Output:   output,
				Size:     size,
				Model:    modelName,
				Style:    style,
				Negative: negative,
			}
			if seed != 0 {
				seedInt := int64(seed)
				opts.Seed = &seedInt
			}

			// Determine which API to use
			var client generate.ImageGenerator
			var err error

			if isVertexModel(modelName) {
				client, err = generate.NewVertexClient(cfg)
			} else {
				client, err = generate.NewGeminiClient(cfg)
			}

			if err != nil {
				return nil, fmt.Errorf("failed to create client: %w", err)
			}

			// Generate image
			result, err := client.GenerateImage(prompt, opts)
			if err != nil {
				return nil, fmt.Errorf("failed to generate image: %w", err)
			}

			// Return result
			absPath, _ := filepath.Abs(output)
			return map[string]interface{}{
				"success":     true,
				"output_path": absPath,
				"size":        size,
				"model":       modelName,
				"prompt":      prompt,
			}, nil
		},
	}

	server.RegisterTool(tool)
}

func isVertexModel(model string) bool {
	return model == "imagen-3.0-generate-002" || model == "imagen-4"
}
```

#### 2.2 Resize Tool

**File**: `internal/mcp/tools/resize.go`

```go
package tools

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/apresai/gimage/internal/imaging"
	"github.com/apresai/gimage/internal/mcp"
)

// RegisterResizeImageTool registers the resize_image tool
func RegisterResizeImageTool(server *mcp.MCPServer) {
	tool := mcp.Tool{
		Name:        "resize_image",
		Description: "Resize an image to specific dimensions using high-quality Lanczos resampling. Aspect ratio is not preserved unless dimensions match original ratio.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type":        "string",
					"description": "Input image file path (absolute or relative)",
				},
				"width": map[string]interface{}{
					"type":        "integer",
					"description": "Target width in pixels",
					"minimum":     1,
				},
				"height": map[string]interface{}{
					"type":        "integer",
					"description": "Target height in pixels",
					"minimum":     1,
				},
				"output": map[string]interface{}{
					"type":        "string",
					"description": "Output file path (optional, auto-generated if not provided)",
				},
			},
			"required": []string{"input", "width", "height"},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			input, _ := args["input"].(string)
			if input == "" {
				return nil, fmt.Errorf("input is required")
			}

			width, ok := args["width"].(float64)
			if !ok || width < 1 {
				return nil, fmt.Errorf("width must be a positive integer")
			}

			height, ok := args["height"].(float64)
			if !ok || height < 1 {
				return nil, fmt.Errorf("height must be a positive integer")
			}

			output, _ := args["output"].(string)
			if output == "" {
				output = generateOutputPath(input, "resized")
			}

			// Get original dimensions
			origWidth, origHeight, err := imaging.GetImageDimensions(input)
			if err != nil {
				return nil, fmt.Errorf("failed to read input image: %w", err)
			}

			// Resize image
			err = imaging.ResizeImage(input, output, int(width), int(height))
			if err != nil {
				return nil, fmt.Errorf("failed to resize image: %w", err)
			}

			absPath, _ := filepath.Abs(output)
			return map[string]interface{}{
				"success":       true,
				"output_path":   absPath,
				"original_size": fmt.Sprintf("%dx%d", origWidth, origHeight),
				"new_size":      fmt.Sprintf("%dx%d", int(width), int(height)),
			}, nil
		},
	}

	server.RegisterTool(tool)
}
```

#### 2.3 Additional Tool Files

Create similar tool files for:
- `internal/mcp/tools/scale.go` - scale_image tool
- `internal/mcp/tools/crop.go` - crop_image tool
- `internal/mcp/tools/compress.go` - compress_image tool
- `internal/mcp/tools/convert.go` - convert_image tool
- `internal/mcp/tools/batch.go` - batch_* tools
- `internal/mcp/tools/info.go` - get_image_info tool
- `internal/mcp/tools/models.go` - list_models tool

---

### Phase 3: CLI Integration (Week 3)

#### 3.1 Add Serve Command

**File**: `internal/cli/serve.go`

```go
package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/apresai/gimage/internal/config"
	"github.com/apresai/gimage/internal/mcp"
	"github.com/apresai/gimage/internal/mcp/tools"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server for AI assistant integration",
	Long: `Start the gimage MCP (Model Context Protocol) server.

This allows AI assistants like Claude to use gimage for image generation
and processing operations. The server communicates over stdio using the
MCP protocol.

USAGE WITH CLAUDE DESKTOP:

Add this to your Claude Desktop MCP configuration:

  {
    "mcpServers": {
      "gimage": {
        "command": "gimage",
        "args": ["serve"]
      }
    }
  }

Configuration file location:
  • macOS: ~/Library/Application Support/Claude/claude_desktop_config.json
  • Linux: ~/.config/Claude/claude_desktop_config.json

ENVIRONMENT VARIABLES:

The serve command respects the same environment variables as the CLI:
  • GEMINI_API_KEY - Gemini API key for image generation
  • VERTEX_API_KEY - Vertex AI API key (Express Mode)
  • VERTEX_PROJECT - GCP project ID for Vertex AI
  • VERTEX_LOCATION - Vertex AI location (default: us-central1)
  • GOOGLE_APPLICATION_CREDENTIALS - Path to service account JSON
  • AWS_ACCESS_KEY_ID - AWS Bedrock access key
  • AWS_SECRET_ACCESS_KEY - AWS Bedrock secret key
  • AWS_REGION - AWS region (default: us-east-1)
  • AWS_PROFILE - AWS profile name
  • AWS_BEARER_TOKEN_BEDROCK - AWS Bedrock bearer token (REST mode)

FEATURES:

The MCP server exposes 10 tools to AI assistants:
  • generate_image    - AI image generation with Gemini/Vertex
  • resize_image      - Resize to specific dimensions
  • scale_image       - Scale by factor
  • crop_image        - Crop to region
  • compress_image    - Compress to reduce file size
  • convert_image     - Convert between formats
  • batch_resize      - Batch resize operations
  • batch_compress    - Batch compression
  • batch_convert     - Batch format conversion
  • list_models       - List available AI models

EXAMPLES:

  # Start MCP server (usually called by AI assistant)
  $ gimage serve

  # Test with verbose logging (logs go to stderr)
  $ gimage serve --verbose

  # Use custom config file
  $ gimage serve --config ~/.gimage/custom-config.yaml

TROUBLESHOOTING:

If the MCP server isn't working in Claude:
  1. Check that gimage is in your PATH: which gimage
  2. Check authentication status: gimage auth status
  3. Verify credentials work: gimage auth test
  4. Look at Claude's logs for error messages
  5. Test image generation works: gimage generate "test image"

For more information: https://github.com/apresai/gimage`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Create MCP server
		server := mcp.NewMCPServer("gimage", version, cfg)

		// Register all tools
		tools.RegisterGenerateImageTool(server, cfg)
		tools.RegisterResizeImageTool(server)
		tools.RegisterScaleImageTool(server)
		tools.RegisterCropImageTool(server)
		tools.RegisterCompressImageTool(server)
		tools.RegisterConvertImageTool(server)
		tools.RegisterBatchResizeTool(server)
		tools.RegisterBatchCompressTool(server)
		tools.RegisterBatchConvertTool(server)
		tools.RegisterListModelsTool(server)

		// Setup signal handling for graceful shutdown
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		go func() {
			<-sigChan
			fmt.Fprintln(os.Stderr, "\n[gimage-mcp] Shutting down gracefully...")
			cancel()
		}()

		// Log startup (to stderr, not stdout)
		if verbose {
			fmt.Fprintln(os.Stderr, "[gimage-mcp] Starting MCP server")
			fmt.Fprintln(os.Stderr, "[gimage-mcp] Protocol: Model Context Protocol")
			fmt.Fprintln(os.Stderr, "[gimage-mcp] Transport: stdio")
			fmt.Fprintln(os.Stderr, "[gimage-mcp] Tools: 10 registered")
		}

		// Start server
		return server.Start(ctx)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
```

---

### Phase 4: npm Package Wrapper (Week 4)

#### 4.1 Package Structure

```
gimage-mcp/
├── package.json
├── index.js
├── bin/
│   └── gimage-mcp.js
├── scripts/
│   ├── install.js
│   └── download-binary.js
└── README.md
```

#### 4.2 package.json

**File**: `package.json`

```json
{
  "name": "@apresai/gimage-mcp",
  "version": "0.1.0",
  "description": "MCP server for AI-powered image generation and processing with gimage",
  "keywords": [
    "mcp",
    "model-context-protocol",
    "claude",
    "ai",
    "image-generation",
    "image-processing",
    "gemini",
    "vertex-ai"
  ],
  "author": "Chad Neal <chad@apresai.com>",
  "license": "MIT",
  "repository": {
    "type": "git",
    "url": "https://github.com/apresai/gimage.git"
  },
  "homepage": "https://github.com/apresai/gimage#readme",
  "bugs": {
    "url": "https://github.com/apresai/gimage/issues"
  },
  "bin": {
    "gimage-mcp": "./bin/gimage-mcp.js"
  },
  "scripts": {
    "postinstall": "node scripts/install.js",
    "test": "node test/test-mcp.js"
  },
  "engines": {
    "node": ">=18.0.0"
  },
  "os": [
    "darwin",
    "linux",
    "win32"
  ],
  "cpu": [
    "x64",
    "arm64"
  ],
  "files": [
    "bin/",
    "scripts/",
    "index.js",
    "README.md"
  ],
  "dependencies": {
    "tar": "^7.0.0"
  },
  "devDependencies": {}
}
```

#### 4.3 Binary Installer

**File**: `scripts/install.js`

```javascript
#!/usr/bin/env node

const https = require('https');
const fs = require('fs');
const path = require('path');
const tar = require('tar');
const { execSync } = require('child_process');

const GITHUB_REPO = 'apresai/gimage';
const VERSION = process.env.npm_package_version || '0.1.0';

function getPlatformInfo() {
  const platform = process.platform;
  const arch = process.arch;

  const platformMap = {
    darwin: 'darwin',
    linux: 'linux',
    win32: 'windows'
  };

  const archMap = {
    x64: 'amd64',
    arm64: 'arm64'
  };

  return {
    platform: platformMap[platform],
    arch: archMap[arch],
    ext: platform === 'win32' ? '.exe' : ''
  };
}

async function downloadBinary() {
  const { platform, arch, ext } = getPlatformInfo();
  const binaryName = `gimage${ext}`;
  const tarballName = `gimage-${platform}-${arch}.tar.gz`;
  const url = `https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/${tarballName}`;

  const binDir = path.join(__dirname, '..', 'bin');
  const binaryPath = path.join(binDir, binaryName);

  // Create bin directory if it doesn't exist
  if (!fs.existsSync(binDir)) {
    fs.mkdirSync(binDir, { recursive: true });
  }

  console.log(`Downloading gimage binary for ${platform}-${arch}...`);
  console.log(`URL: ${url}`);

  return new Promise((resolve, reject) => {
    https.get(url, (response) => {
      if (response.statusCode === 302 || response.statusCode === 301) {
        // Follow redirect
        https.get(response.headers.location, (redirectResponse) => {
          if (redirectResponse.statusCode !== 200) {
            reject(new Error(`Download failed: ${redirectResponse.statusCode}`));
            return;
          }

          const tarPath = path.join(binDir, tarballName);
          const file = fs.createWriteStream(tarPath);

          redirectResponse.pipe(file);

          file.on('finish', async () => {
            file.close();

            // Extract tarball
            await tar.x({
              file: tarPath,
              cwd: binDir
            });

            // Remove tarball
            fs.unlinkSync(tarPath);

            // Make binary executable
            fs.chmodSync(binaryPath, 0o755);

            console.log('✓ gimage binary installed successfully');
            resolve();
          });
        }).on('error', reject);
      } else if (response.statusCode === 200) {
        const tarPath = path.join(binDir, tarballName);
        const file = fs.createWriteStream(tarPath);

        response.pipe(file);

        file.on('finish', async () => {
          file.close();

          // Extract tarball
          await tar.x({
            file: tarPath,
            cwd: binDir
          });

          // Remove tarball
          fs.unlinkSync(tarPath);

          // Make binary executable
          fs.chmodSync(binaryPath, 0o755);

          console.log('✓ gimage binary installed successfully');
          resolve();
        });
      } else {
        reject(new Error(`Download failed: ${response.statusCode}`));
      }
    }).on('error', reject);
  });
}

async function main() {
  try {
    await downloadBinary();
    console.log('\n✓ Installation complete!');
    console.log('\nTo use with Claude Desktop, add this to your MCP configuration:');
    console.log('\n{');
    console.log('  "mcpServers": {');
    console.log('    "gimage": {');
    console.log('      "command": "npx",');
    console.log('      "args": ["-y", "@apresai/gimage-mcp"]');
    console.log('    }');
    console.log('  }');
    console.log('}');
  } catch (error) {
    console.error('Installation failed:', error.message);
    console.error('\nFallback: You can install gimage manually from:');
    console.error(`https://github.com/${GITHUB_REPO}/releases`);
    process.exit(1);
  }
}

main();
```

#### 4.4 MCP Wrapper Script

**File**: `bin/gimage-mcp.js`

```javascript
#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

function getBinaryPath() {
  const platform = process.platform;
  const ext = platform === 'win32' ? '.exe' : '';
  const binaryName = `gimage${ext}`;

  // Try local binary first
  const localPath = path.join(__dirname, '..', 'bin', binaryName);
  if (fs.existsSync(localPath)) {
    return localPath;
  }

  // Fall back to PATH
  return 'gimage';
}

function main() {
  const binaryPath = getBinaryPath();

  // Spawn gimage serve command
  const child = spawn(binaryPath, ['serve'], {
    stdio: 'inherit',
    env: process.env
  });

  child.on('error', (error) => {
    console.error('Failed to start gimage MCP server:', error.message);
    console.error('\nPlease ensure gimage is installed:');
    console.error('  npm install -g @apresai/gimage-mcp');
    console.error('  or');
    console.error('  brew install gimage');
    process.exit(1);
  });

  child.on('exit', (code) => {
    process.exit(code || 0);
  });

  // Handle signals
  process.on('SIGINT', () => {
    child.kill('SIGINT');
  });

  process.on('SIGTERM', () => {
    child.kill('SIGTERM');
  });
}

main();
```

---

## 📦 Project Structure

```
gimage/
├── cmd/
│   ├── gimage/
│   │   └── main.go              # CLI entrypoint
│   └── lambda/
│       └── main.go              # Lambda entrypoint
├── internal/
│   ├── mcp/                     # NEW: MCP server implementation
│   │   ├── server.go            # Core MCP server
│   │   ├── types.go             # JSON-RPC types
│   │   ├── handler.go           # Request handlers
│   │   └── tools/               # Tool implementations
│   │       ├── generate.go      # generate_image tool
│   │       ├── resize.go        # resize_image tool
│   │       ├── scale.go         # scale_image tool
│   │       ├── crop.go          # crop_image tool
│   │       ├── compress.go      # compress_image tool
│   │       ├── convert.go       # convert_image tool
│   │       ├── batch.go         # batch_* tools
│   │       ├── info.go          # get_image_info tool
│   │       ├── models.go        # list_models tool
│   │       └── helpers.go       # Shared helpers
│   ├── cli/
│   │   ├── serve.go             # NEW: MCP serve command
│   │   └── ...                  # Existing CLI commands
│   ├── generate/                # Existing: AI generation
│   ├── imaging/                 # Existing: Image processing
│   └── config/                  # Existing: Configuration
├── pkg/
│   └── models/                  # Existing: Shared types
├── npm/                         # NEW: npm package wrapper
│   ├── package.json
│   ├── index.js
│   ├── bin/
│   │   └── gimage-mcp.js
│   ├── scripts/
│   │   └── install.js
│   └── README.md
├── docs/                        # NEW: MCP documentation
│   ├── MCP_USAGE.md
│   ├── MCP_TOOLS.md
│   └── MCP_EXAMPLES.md
└── test/
    └── mcp/                     # NEW: MCP tests
        ├── server_test.go
        └── tools_test.go
```

---

## 🔧 Installation Methods

### Method 1: npm Package (Recommended for Claude Desktop)

```bash
# Install globally
npm install -g @apresai/gimage-mcp

# Or use with npx (no installation required)
npx @apresai/gimage-mcp
```

**Claude Desktop Configuration**:
```json
{
  "mcpServers": {
    "gimage": {
      "command": "npx",
      "args": ["-y", "@apresai/gimage-mcp"]
    }
  }
}
```

### Method 2: Direct Binary (Recommended for Advanced Users)

```bash
# Install gimage CLI
brew install apresai/tap/gimage

# Or download from GitHub releases
curl -L https://github.com/apresai/gimage/releases/latest/download/gimage-darwin-arm64 -o gimage
chmod +x gimage
sudo mv gimage /usr/local/bin/
```

**Claude Desktop Configuration**:
```json
{
  "mcpServers": {
    "gimage": {
      "command": "gimage",
      "args": ["serve"]
    }
  }
}
```

### Method 3: Build from Source

```bash
# Clone repository
git clone https://github.com/apresai/gimage.git
cd gimage

# Build binary
make build

# Install
sudo cp bin/gimage /usr/local/bin/
```

---

## ⚙️ Configuration

### Authentication Setup

Before using the MCP server, configure API credentials:

```bash
# Option 1: Gemini API (simplest)
gimage auth gemini

# Option 2: Vertex AI
gimage auth vertex
```

### Environment Variables

The MCP server respects these environment variables:

```bash
# Gemini API
export GEMINI_API_KEY="your-gemini-key"

# Vertex AI Express Mode
export VERTEX_API_KEY="your-vertex-key"
export VERTEX_PROJECT="your-gcp-project"
export VERTEX_LOCATION="us-central1"

# Vertex AI Full Mode
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account.json"
export VERTEX_PROJECT="your-gcp-project"
export VERTEX_LOCATION="us-central1"
```

### Config File

Configuration is stored in `~/.gimage/config.md`:

```markdown
# Gimage Configuration

**gemini_api_key**: your-key
**vertex_project**: your-project
**vertex_location**: us-central1
**default_api**: gemini
**default_model**: gemini-3-pro-image-preview
**log_level**: info
```

---

## 🧪 Testing Strategy

### Unit Tests

```go
// test/mcp/server_test.go
package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/apresai/gimage/internal/mcp"
	"github.com/stretchr/testify/assert"
)

func TestMCPServerInitialize(t *testing.T) {
	server := mcp.NewMCPServer("test", "1.0.0", nil)

	request := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}

	response := server.HandleRequest(context.Background(), &request)

	assert.Equal(t, "2.0", response.JSONRPC)
	assert.Equal(t, 1, response.ID)
	assert.Nil(t, response.Error)
	assert.NotNil(t, response.Result)
}

func TestMCPServerListTools(t *testing.T) {
	server := mcp.NewMCPServer("test", "1.0.0", nil)

	// Register test tool
	server.RegisterTool(mcp.Tool{
		Name:        "test_tool",
		Description: "Test tool",
		InputSchema: map[string]interface{}{},
		Handler:     func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{"result": "ok"}, nil
		},
	})

	request := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}

	response := server.HandleRequest(context.Background(), &request)

	assert.Equal(t, "2.0", response.JSONRPC)
	assert.Nil(t, response.Error)

	tools := response.Result["tools"].([]map[string]interface{})
	assert.Len(t, tools, 1)
	assert.Equal(t, "test_tool", tools[0]["name"])
}
```

### Integration Tests

```go
// test/mcp/integration_test.go
package mcp_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/apresai/gimage/internal/config"
	"github.com/apresai/gimage/internal/mcp"
	"github.com/apresai/gimage/internal/mcp/tools"
	"github.com/stretchr/testify/assert"
)

func TestGenerateImageTool(t *testing.T) {
	if os.Getenv("GEMINI_API_KEY") == "" {
		t.Skip("GEMINI_API_KEY not set")
	}

	cfg, err := config.Load()
	assert.NoError(t, err)

	server := mcp.NewMCPServer("test", "1.0.0", cfg)
	tools.RegisterGenerateImageTool(server, cfg)

	request := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "generate_image",
			"arguments": map[string]interface{}{
				"prompt": "test image for automated testing",
				"size":   "256x256",
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response := server.HandleRequest(ctx, &request)

	assert.Nil(t, response.Error)
	assert.NotNil(t, response.Result)

	// Cleanup generated file
	content := response.Result["content"].([]map[string]interface{})
	resultText := content[0]["text"].(string)

	var result map[string]interface{}
	json.Unmarshal([]byte(resultText), &result)

	outputPath := result["output_path"].(string)
	defer os.Remove(outputPath)

	// Verify file exists
	_, err = os.Stat(outputPath)
	assert.NoError(t, err)
}
```

### E2E Tests with Claude

```bash
# test/mcp/e2e_test.sh

#!/bin/bash

# Test MCP server with real Claude Desktop integration

set -e

echo "Testing gimage MCP server..."

# Start MCP server in background
gimage serve &
SERVER_PID=$!

# Give server time to start
sleep 2

# Send test request
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"test","version":"1.0.0"}}}' | \
  gimage serve

# Cleanup
kill $SERVER_PID

echo "✓ MCP server test passed"
```

---

## 📚 Documentation

### User Documentation

#### MCP_USAGE.md

```markdown
# Using Gimage with AI Assistants

## Quick Start

### 1. Install gimage MCP server

```bash
npm install -g @apresai/gimage-mcp
```

### 2. Configure API credentials

```bash
# Setup Gemini API (free tier available)
gimage auth gemini
```

Get your API key from: https://aistudio.google.com/app/apikey

### 3. Add to Claude Desktop

Edit your Claude Desktop configuration:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Linux**: `~/.config/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "gimage": {
      "command": "npx",
      "args": ["-y", "@apresai/gimage-mcp"]
    }
  }
}
```

### 4. Restart Claude Desktop

Quit and reopen Claude Desktop to load the MCP server.

### 5. Start Using!

Try these prompts in Claude:

**Image Generation**:
- "Generate an image of a sunset over mountains"
- "Create a photorealistic portrait of a wise old wizard"
- "Generate an anime-style image of cherry blossoms"

**Image Processing**:
- "Resize photo.jpg to 800x600"
- "Compress all images in the photos directory"
- "Convert image.png to WebP format"

## Available Operations

### AI Image Generation
- Multiple AI models (Gemini 2.5 Flash, Imagen 4)
- Custom sizes (up to 2048x2048)
- Style control (photorealistic, artistic, anime)
- Negative prompts
- Reproducible generation with seeds

### Image Processing
- Resize to specific dimensions
- Scale by factor
- Crop regions
- Compress with quality control
- Convert between formats (PNG, JPG, WebP, GIF, TIFF, BMP)

### Batch Operations
- Process multiple images concurrently
- Configurable worker pools
- Progress reporting

## Troubleshooting

### Server not connecting

1. Verify gimage is installed:
   ```bash
   which gimage
   ```

2. Test manually:
   ```bash
   gimage serve
   ```

3. Check Claude Desktop logs

### Image generation fails

1. Verify API key:
   ```bash
   gimage auth gemini
   ```

2. Test generation:
   ```bash
   gimage generate "test image"
   ```

### Permission errors

Ensure gimage has write permissions to the current directory.

## Examples

See MCP_EXAMPLES.md for comprehensive examples.
```

#### MCP_TOOLS.md

```markdown
# Gimage MCP Tools Reference

Complete reference for all 10 MCP tools available in gimage.

## Tool List

1. [generate_image](#generate_image) - AI image generation
2. [resize_image](#resize_image) - Resize to dimensions
3. [scale_image](#scale_image) - Scale by factor
4. [crop_image](#crop_image) - Crop to region
5. [compress_image](#compress_image) - Compress file size
6. [convert_image](#convert_image) - Convert formats
7. [batch_resize](#batch_resize) - Batch resize
8. [batch_compress](#batch_compress) - Batch compress
9. [batch_convert](#batch_convert) - Batch convert
10. [list_models](#list_models) - List AI models

---

## generate_image

Generate an AI image from a text prompt.

**Input**:
- `prompt` (string, required): Description of the image
- `output` (string): Output file path
- `size` (string): Image dimensions (e.g., "1024x1024")
- `model` (string): AI model name
- `style` (string): Style (photorealistic, artistic, anime)
- `negative` (string): Negative prompt
- `seed` (integer): Random seed

**Output**:
- `success` (boolean)
- `output_path` (string)
- `size` (string)
- `model` (string)

**Example**:
```json
{
  "prompt": "a sunset over mountains",
  "size": "1024x1024",
  "style": "photorealistic"
}
```

---

## resize_image

Resize an image to specific dimensions.

**Input**:
- `input` (string, required): Input file path
- `width` (integer, required): Target width
- `height` (integer, required): Target height
- `output` (string): Output file path

**Output**:
- `success` (boolean)
- `output_path` (string)
- `original_size` (string)
- `new_size` (string)

**Example**:
```json
{
  "input": "photo.jpg",
  "width": 800,
  "height": 600
}
```

[... continue for all 10 tools ...]
```

#### MCP_EXAMPLES.md

```markdown
# Gimage MCP Examples

Real-world examples of using gimage with AI assistants.

## Image Generation

### Basic Generation

**Prompt**: "Generate an image of a sunset over mountains"

**Claude's Actions**:
1. Calls `generate_image` tool
2. Returns path to generated image
3. Can display image if in appropriate context

### Advanced Generation with Customization

**Prompt**: "Create a 2048x2048 photorealistic image of a wise old wizard using Imagen 4. Avoid modern clothing and technology."

**Claude's Actions**:
1. Calls `generate_image`:
   ```json
   {
     "prompt": "wise old wizard with ancient robes and magical staff",
     "size": "2048x2048",
     "model": "imagen-4",
     "style": "photorealistic",
     "negative": "modern clothing, technology, smartphones, computers"
   }
   ```

### Reproducible Generation

**Prompt**: "Generate the same wizard image twice to compare"

**Claude's Actions**:
1. First generation with seed: `{"prompt": "wizard", "seed": 42}`
2. Second generation with same seed: `{"prompt": "wizard", "seed": 42}`
3. Both images will be identical

## Image Processing

### Web Optimization Workflow

**Prompt**: "Optimize all photos in my-photos directory for web: resize to 1920x1080, compress to 85% quality, and convert to WebP"

**Claude's Actions**:
1. `batch_resize`: Resize all images
2. `batch_compress`: Compress with quality=85
3. `batch_convert`: Convert to WebP format
4. Reports total space saved

### Social Media Preparation

**Prompt**: "Prepare photo.jpg for Instagram: square crop from center, resize to 1080x1080, compress"

**Claude's Actions**:
1. `get_image_info`: Get original dimensions
2. `crop_image`: Square crop from center
3. `resize_image`: Resize to 1080x1080
4. `compress_image`: Compress with quality=90

## Complex Workflows

### E-commerce Product Images

**Prompt**: "Process product.jpg for my store: create large (1200x1200), medium (600x600), and thumbnail (200x200) versions, all in WebP"

**Claude's Actions**:
1. `resize_image`: Create large version
2. `resize_image`: Create medium version
3. `resize_image`: Create thumbnail
4. `convert_image`: Convert each to WebP
5. Returns all three file paths

### Batch Processing with Reporting

**Prompt**: "Compress all images in photos/ to save disk space, show me how much space we save"

**Claude's Actions**:
1. `batch_compress`: Process all images with quality=85
2. Parses compression ratios
3. Calculates total savings
4. Presents summary:
   - Original total size
   - Compressed total size
   - Space saved (GB)
   - Compression percentage

## Creative Workflows

### Generate and Process

**Prompt**: "Generate a fantasy landscape, then create mobile and desktop versions"

**Claude's Actions**:
1. `generate_image`: Create 2048x2048 landscape
2. `resize_image`: Desktop version (1920x1080)
3. `resize_image`: Mobile version (750x1334)
4. `compress_image`: Compress each for web

### Style Comparison

**Prompt**: "Generate the same scene in photorealistic, artistic, and anime styles"

**Claude's Actions**:
1. `generate_image`: Photorealistic version
2. `generate_image`: Artistic version
3. `generate_image`: Anime version
4. Presents all three for comparison

## Troubleshooting Examples

### Handle Missing Files

**User**: "Resize missing.jpg"

**Claude**: Attempts resize, receives error, explains file not found and suggests checking path

### Handle Invalid Parameters

**User**: "Resize photo.jpg to 0x0"

**Claude**: Attempts resize, receives validation error, explains minimum dimensions requirement

### API Key Issues

**User**: "Generate an image"

**Claude**: Attempts generation, receives auth error, guides user through `gimage auth gemini` setup
```

### LLM-Optimized Documentation

**File**: `docs/LLM_GUIDE.md`

This document is specifically formatted for LLMs to understand gimage capabilities:

```markdown
# Gimage MCP Server - LLM Guide

## System Prompt Addition

When using gimage MCP server, you have access to powerful image generation and processing capabilities.

## Capabilities Summary

You can:
1. Generate AI images from text prompts using Gemini or Vertex AI
2. Resize, scale, crop, compress, and convert images
3. Process multiple images in batches efficiently
4. List available AI models
5. Get detailed image information

## Tool Usage Guidelines

### When to use generate_image
- User requests image creation from text
- User describes a scene, object, or concept visually
- User wants AI-generated artwork

### When to use processing tools
- User has existing images to modify
- User needs web optimization
- User needs specific dimensions or formats
- User wants to reduce file sizes

### When to use batch tools
- User has multiple images to process
- User mentions a directory of images
- User wants consistent processing across many files

## Best Practices

1. **Always confirm file paths** - Ask user to verify paths before processing
2. **Explain API key requirements** - Guide users through auth setup if needed
3. **Provide clear error messages** - Parse tool errors and explain in user-friendly terms
4. **Suggest optimizations** - Recommend appropriate sizes, quality settings
5. **Report results clearly** - Summarize what was done and where files are located

## Common Workflows

### Image Generation
1. Understand user's vision
2. Craft detailed prompt
3. Select appropriate model and size
4. Generate image
5. Offer to make adjustments

### Web Optimization
1. Understand target platform
2. Resize to appropriate dimensions
3. Compress with suitable quality
4. Convert to modern format (WebP)
5. Report savings

### Batch Processing
1. Confirm source directory
2. Create output directory
3. Process all images
4. Report success/failure counts
5. Show total results

## Error Handling

When tools fail:
1. Parse error message
2. Identify root cause
3. Suggest solution
4. Offer to try again with corrections

Common issues:
- Missing API keys → Guide through `gimage auth`
- File not found → Check path, suggest `ls` command
- Invalid dimensions → Explain constraints
- Insufficient permissions → Suggest chmod/sudo

## Example Interactions

See MCP_EXAMPLES.md for detailed scenarios.
```

---

## 🚀 Deployment

### GitHub Releases

#### Release Process

1. **Update Version**
   ```bash
   # Update version in all files
   VERSION=0.1.0

   # Update CLI
   sed -i '' "s/version = .*/version = \"$VERSION\"/" internal/cli/root.go

   # Update npm package
   sed -i '' "s/\"version\": .*/\"version\": \"$VERSION\",/" npm/package.json
   ```

2. **Create Git Tag**
   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0: MCP server support"
   git push origin v0.1.0
   ```

3. **GitHub Actions Build**

**File**: `.github/workflows/release.yml`

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    name: Build Binaries
    runs-on: ubuntu-latest
    strategy:
      matrix:
        goos: [linux, darwin, windows]
        goarch: [amd64, arm64]
        exclude:
          - goos: windows
            goarch: arm64

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Build
        run: |
          GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} \
          go build -o gimage-${{ matrix.goos }}-${{ matrix.goarch }}${{ matrix.goos == 'windows' && '.exe' || '' }} \
          -ldflags="-s -w" \
          ./cmd/gimage

      - name: Create tarball
        run: |
          tar czf gimage-${{ matrix.goos }}-${{ matrix.goarch }}.tar.gz \
          gimage-${{ matrix.goos }}-${{ matrix.goarch }}${{ matrix.goos == 'windows' && '.exe' || '' }}

      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        with:
          name: gimage-${{ matrix.goos }}-${{ matrix.goarch }}
          path: gimage-${{ matrix.goos }}-${{ matrix.goarch }}.tar.gz

  release:
    name: Create Release
    needs: build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/download-artifact@v4

      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: |
            gimage-*/gimage-*.tar.gz
          body: |
            # Gimage ${{ github.ref_name }}

            ## Installation

            ### npm (Recommended for Claude Desktop)
            ```bash
            npm install -g @apresai/gimage-mcp
            ```

            ### Direct Binary
            Download the appropriate binary for your platform:

            - **macOS Intel**: gimage-darwin-amd64.tar.gz
            - **macOS Apple Silicon**: gimage-darwin-arm64.tar.gz
            - **Linux x86_64**: gimage-linux-amd64.tar.gz
            - **Linux ARM64**: gimage-linux-arm64.tar.gz
            - **Windows**: gimage-windows-amd64.tar.gz

            ### Usage with Claude Desktop

            See [MCP_USAGE.md](docs/MCP_USAGE.md)
          draft: false
          prerelease: false
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### npm Publishing

**File**: `.github/workflows/publish-npm.yml`

```yaml
name: Publish npm Package

on:
  release:
    types: [created]

jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          registry-url: 'https://registry.npmjs.org'

      - name: Publish to npm
        run: |
          cd npm
          npm publish --access public
        env:
          NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}
```

### Homebrew Formula Update

After release, update Homebrew formula:

```ruby
# homebrew-tap/Formula/gimage.rb
class Gimage < Formula
  desc "AI-powered image generation and processing CLI with MCP server"
  homepage "https://github.com/apresai/gimage"
  version "0.1.0"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/apresai/gimage/releases/download/v0.1.0/gimage-darwin-amd64.tar.gz"
      sha256 "..." # Calculate with: shasum -a 256 gimage-darwin-amd64.tar.gz
    else
      url "https://github.com/apresai/gimage/releases/download/v0.1.0/gimage-darwin-arm64.tar.gz"
      sha256 "..." # Calculate with: shasum -a 256 gimage-darwin-arm64.tar.gz
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/apresai/gimage/releases/download/v0.1.0/gimage-linux-amd64.tar.gz"
      sha256 "..."
    else
      url "https://github.com/apresai/gimage/releases/download/v0.1.0/gimage-linux-arm64.tar.gz"
      sha256 "..."
    end
  end

  def install
    bin.install "gimage"
  end

  test do
    assert_match "gimage version 0.1.0", shell_output("#{bin}/gimage --version")
  end
end
```

---

## 📅 Timeline

### Week 1: Core Infrastructure
- Day 1-2: MCP server scaffolding (`internal/mcp/`)
- Day 3-4: JSON-RPC protocol implementation
- Day 5: Initialize/ListTools/CallTool handlers
- Day 6-7: Testing framework setup

### Week 2: Tool Implementation (Part 1)
- Day 1: `generate_image` tool
- Day 2: `resize_image` tool
- Day 3: `scale_image` tool
- Day 4: `crop_image` tool
- Day 5: `compress_image` tool
- Day 6-7: Unit tests for tools

### Week 3: Tool Implementation (Part 2) + CLI
- Day 1: `convert_image` tool
- Day 2: Batch tools (`batch_resize`, `batch_compress`, `batch_convert`)
- Day 3: `get_image_info` and `list_models` tools
- Day 4: `serve` CLI command
- Day 5: Integration testing
- Day 6-7: Bug fixes and polish

### Week 4: npm Package + Documentation
- Day 1-2: npm package wrapper
- Day 3: Binary download scripts
- Day 4: Documentation (MCP_USAGE.md, MCP_TOOLS.md, MCP_EXAMPLES.md)
- Day 5: E2E testing with Claude Desktop
- Day 6: README updates
- Day 7: Release preparation

### Week 5: Release + Marketing
- Day 1: GitHub release
- Day 2: npm publish
- Day 3: Homebrew formula update
- Day 4: Documentation site updates
- Day 5: Blog post / announcement
- Day 6-7: Community support

---

## 🎯 Success Criteria

### Technical
- [ ] All 10 MCP tools implemented and tested
- [ ] MCP protocol fully compliant
- [ ] npm package installable and working
- [ ] Binary downloads available for all platforms
- [ ] Integration with Claude Desktop verified
- [ ] All tests passing (unit, integration, E2E)
- [ ] Documentation complete

### User Experience
- [ ] Installation takes < 5 minutes
- [ ] Setup takes < 2 minutes
- [ ] Clear error messages for all failure modes
- [ ] Examples work out of the box
- [ ] Claude can use all tools successfully

### Distribution
- [ ] npm package published
- [ ] GitHub releases created
- [ ] Homebrew formula updated
- [ ] Documentation site live
- [ ] Community examples shared

---

## 📞 Support & Resources

### Documentation
- **MCP_USAGE.md** - User guide for Claude Desktop integration
- **MCP_TOOLS.md** - Complete tool reference
- **MCP_EXAMPLES.md** - Real-world usage examples
- **LLM_GUIDE.md** - Guidelines for LLMs using gimage

### Community
- **GitHub Issues**: https://github.com/apresai/gimage/issues
- **Discussions**: https://github.com/apresai/gimage/discussions
- **Discord** (optional): TBD

### References
- **Model Context Protocol**: https://modelcontextprotocol.io
- **MCP TypeScript SDK**: https://github.com/modelcontextprotocol/typescript-sdk
- **Claude Desktop**: https://claude.ai/desktop

---

## 🚧 Future Enhancements

### Phase 2 Features (Post-Launch)
1. **Resources** - Expose saved images as MCP resources
2. **Prompts** - Pre-configured prompt templates
3. **Sampling** - LLM-assisted prompt enhancement
4. **Streaming** - Progress updates for long operations
5. **Caching** - Cache generated images
6. **Webhooks** - Async notifications

### Advanced Features
1. **Image editing** - In-place editing operations
2. **Style transfer** - Apply artistic styles
3. **Background removal** - Automatic background removal
4. **Face detection** - Detect and crop faces
5. **OCR** - Extract text from images
6. **Video processing** - Extend to video operations

---

**Plan Complete**: This document provides a comprehensive roadmap for implementing a production-ready MCP server for gimage, with clear phases, detailed implementation plans, and extensive documentation for both developers and users.
