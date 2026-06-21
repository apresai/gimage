package test

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apresai/gimage/internal/config"
	"github.com/apresai/gimage/internal/mcp"
	"github.com/apresai/gimage/internal/mcp/tools"
)

// TestMCPServerIntegration tests the full MCP server flow from request to response
func TestMCPServerIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{}

	// Create a test image
	testImagePath := filepath.Join(tmpDir, "test.png")
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.RGBA{100, 150, 200, 255})
		}
	}
	file, err := os.Create(testImagePath)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}
	png.Encode(file, img)
	file.Close()

	// Create and setup server
	server := mcp.NewMCPServer("test-server", "1.0.0", cfg, false)

	// Register all tools
	tools.RegisterResizeImageTool(server)
	tools.RegisterScaleImageTool(server)
	tools.RegisterCompressImageTool(server)
	tools.RegisterListModelsTool(server)

	// Test 1: Initialize
	t.Run("initialize", func(t *testing.T) {
		initRequest := mcp.JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      "init-1",
			Method:  mcp.MethodInitialize,
			Params:  map[string]interface{}{},
		}

		response := server.HandleRequest(context.Background(), &initRequest)

		if response.Error != nil {
			t.Fatalf("Initialize failed: %v", response.Error)
		}

		result := response.Result
		serverInfo := result["serverInfo"].(map[string]interface{})
		if serverInfo["name"] != "test-server" {
			t.Errorf("Expected server name 'test-server', got %s", serverInfo["name"])
		}

		// Verify capabilities include listChanged
		capabilities := result["capabilities"].(map[string]interface{})
		toolsCaps := capabilities["tools"].(map[string]interface{})
		if listChanged, ok := toolsCaps["listChanged"].(bool); !ok || !listChanged {
			t.Error("Expected listChanged capability to be true")
		}
	})

	// Test 2: List Tools
	t.Run("list_tools", func(t *testing.T) {
		listRequest := mcp.JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      "list-1",
			Method:  mcp.MethodListTools,
			Params:  map[string]interface{}{},
		}

		response := server.HandleRequest(context.Background(), &listRequest)

		if response.Error != nil {
			t.Fatalf("List tools failed: %v", response.Error)
		}

		result := response.Result

		// Tools are returned as []map[string]interface{} directly (not marshaled)
		var toolsList []map[string]interface{}

		// The result["tools"] could be either []interface{} or []map[string]interface{}
		// depending on whether it's been through JSON marshaling
		switch v := result["tools"].(type) {
		case []map[string]interface{}:
			toolsList = v
		case []interface{}:
			for _, item := range v {
				toolsList = append(toolsList, item.(map[string]interface{}))
			}
		default:
			t.Fatalf("Unexpected tools type: %T", result["tools"])
		}

		if len(toolsList) != 4 {
			t.Errorf("Expected 4 tools, got %d", len(toolsList))
		}

		// Verify tool names and annotations
		expectedTools := map[string]bool{
			"resize_image":   false,
			"scale_image":    false,
			"compress_image": false,
			"list_models":    false,
		}

		for _, tool := range toolsList {
			name := tool["name"].(string)
			if _, exists := expectedTools[name]; exists {
				expectedTools[name] = true
			}

			// Verify tool has expected fields
			if _, hasDesc := tool["description"]; !hasDesc {
				t.Errorf("Tool %s missing description", name)
			}
			if _, hasSchema := tool["inputSchema"]; !hasSchema {
				t.Errorf("Tool %s missing inputSchema", name)
			}

			// Note: annotations are optional, so we don't require them
		}

		for name, found := range expectedTools {
			if !found {
				t.Errorf("Expected tool %s not found in list", name)
			}
		}
	})

	// Test 3: Call Tool - Resize
	t.Run("call_resize_tool", func(t *testing.T) {
		callRequest := mcp.JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      "call-1",
			Method:  mcp.MethodCallTool,
			Params: map[string]interface{}{
				"name": "resize_image",
				"arguments": map[string]interface{}{
					"input":  testImagePath,
					"width":  float64(100),
					"height": float64(100),
				},
			},
		}

		response := server.HandleRequest(context.Background(), &callRequest)

		if response.Error != nil {
			t.Fatalf("Resize tool call failed: %v", response.Error)
		}

		result := response.Result

		// Content is returned as []map[string]interface{} directly
		var contentList []map[string]interface{}
		switch v := result["content"].(type) {
		case []map[string]interface{}:
			contentList = v
		case []interface{}:
			for _, item := range v {
				contentList = append(contentList, item.(map[string]interface{}))
			}
		default:
			t.Fatalf("Unexpected content type: %T", result["content"])
		}

		if len(contentList) == 0 {
			t.Fatal("Expected content in response")
		}

		text := contentList[0]["text"].(string)

		// Verify response contains success info
		if !strings.Contains(text, "success") {
			t.Error("Expected success in response text")
		}
	})

	// Test 4: Call Tool - List Models
	t.Run("call_list_models_tool", func(t *testing.T) {
		callRequest := mcp.JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      "call-2",
			Method:  mcp.MethodCallTool,
			Params: map[string]interface{}{
				"name":      "list_models",
				"arguments": map[string]interface{}{},
			},
		}

		response := server.HandleRequest(context.Background(), &callRequest)

		if response.Error != nil {
			t.Fatalf("List models tool call failed: %v", response.Error)
		}

		result := response.Result

		// Content is returned as []map[string]interface{} directly
		var contentList []map[string]interface{}
		switch v := result["content"].(type) {
		case []map[string]interface{}:
			contentList = v
		case []interface{}:
			for _, item := range v {
				contentList = append(contentList, item.(map[string]interface{}))
			}
		default:
			t.Fatalf("Unexpected content type: %T", result["content"])
		}

		text := contentList[0]["text"].(string)

		// Verify response contains provider info
		if !strings.Contains(text, "providers") {
			t.Error("Expected providers in response text")
		}
	})
}

// TestMCPServerErrorHandling tests error scenarios
func TestMCPServerErrorHandling(t *testing.T) {
	cfg := &config.Config{}
	server := mcp.NewMCPServer("test", "1.0.0", cfg, false)
	tools.RegisterResizeImageTool(server)

	// Test 1: Invalid tool name
	t.Run("invalid_tool_name", func(t *testing.T) {
		request := mcp.JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      "err-1",
			Method:  mcp.MethodCallTool,
			Params: map[string]interface{}{
				"name":      "nonexistent_tool",
				"arguments": map[string]interface{}{},
			},
		}

		response := server.HandleRequest(context.Background(), &request)

		if response.Error == nil {
			t.Error("Expected error for nonexistent tool")
		}

		if response.Error.Code != mcp.ErrorCodeMethodNotFound {
			t.Errorf("Expected error code %d, got %d", mcp.ErrorCodeMethodNotFound, response.Error.Code)
		}
	})

	// Test 2: Invalid parameters
	t.Run("invalid_parameters", func(t *testing.T) {
		request := mcp.JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      "err-2",
			Method:  mcp.MethodCallTool,
			Params: map[string]interface{}{
				"name": "resize_image",
				"arguments": map[string]interface{}{
					"input": "test.png",
					// Missing width and height
				},
			},
		}

		response := server.HandleRequest(context.Background(), &request)

		if response.Error == nil {
			t.Error("Expected error for invalid parameters")
		}
	})

	// Test 3: Invalid method
	t.Run("invalid_method", func(t *testing.T) {
		request := mcp.JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      "err-3",
			Method:  "invalid.method",
			Params:  map[string]interface{}{},
		}

		response := server.HandleRequest(context.Background(), &request)

		if response.Error == nil {
			t.Error("Expected error for invalid method")
		}

		if response.Error.Code != mcp.ErrorCodeMethodNotFound {
			t.Errorf("Expected error code %d, got %d", mcp.ErrorCodeMethodNotFound, response.Error.Code)
		}
	})
}

// TestToolAnnotations verifies that tools properly expose annotations
func TestToolAnnotations(t *testing.T) {
	cfg := &config.Config{}
	server := mcp.NewMCPServer("test", "1.0.0", cfg, false)

	// Register tools that have annotations
	tools.RegisterGenerateImageTool(server)
	tools.RegisterBatchCompressTool(server)

	request := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "annot-1",
		Method:  mcp.MethodListTools,
		Params:  map[string]interface{}{},
	}

	response := server.HandleRequest(context.Background(), &request)

	if response.Error != nil {
		t.Fatalf("List tools failed: %v", response.Error)
	}

	result := response.Result

	// Tools are returned as []map[string]interface{} directly (not marshaled)
	var toolsList []map[string]interface{}

	// The result["tools"] could be either []interface{} or []map[string]interface{}
	switch v := result["tools"].(type) {
	case []map[string]interface{}:
		toolsList = v
	case []interface{}:
		for _, item := range v {
			toolsList = append(toolsList, item.(map[string]interface{}))
		}
	default:
		t.Fatalf("Unexpected tools type: %T", result["tools"])
	}

	// Find generate_image tool and verify annotations
	for _, tool := range toolsList {
		name := tool["name"].(string)

		if name == "generate_image" {
			annotations, hasAnnot := tool["annotations"]
			if !hasAnnot {
				t.Error("generate_image tool missing annotations")
				continue
			}

			// Annotations can be either *mcp.ToolAnnotations or map[string]interface{}
			var annot *mcp.ToolAnnotations
			switch v := annotations.(type) {
			case *mcp.ToolAnnotations:
				annot = v
			case map[string]interface{}:
				annot = &mcp.ToolAnnotations{
					DestructiveHint: v["destructiveHint"].(bool),
					IdempotentHint:  v["idempotentHint"].(bool),
					ReadOnlyHint:    v["readOnlyHint"].(bool),
				}
			default:
				t.Errorf("Unexpected annotations type: %T", annotations)
				continue
			}

			if annot.DestructiveHint != false {
				t.Error("generate_image destructiveHint should be false")
			}
			if annot.IdempotentHint != false {
				t.Error("generate_image idempotentHint should be false")
			}
			if annot.ReadOnlyHint != false {
				t.Error("generate_image readOnlyHint should be false")
			}
		}

		if name == "batch_compress" {
			annotations, hasAnnot := tool["annotations"]
			if !hasAnnot {
				t.Error("batch_compress tool missing annotations")
				continue
			}

			// Annotations can be either *mcp.ToolAnnotations or map[string]interface{}
			var annot *mcp.ToolAnnotations
			switch v := annotations.(type) {
			case *mcp.ToolAnnotations:
				annot = v
			case map[string]interface{}:
				annot = &mcp.ToolAnnotations{
					DestructiveHint: v["destructiveHint"].(bool),
					IdempotentHint:  v["idempotentHint"].(bool),
					ReadOnlyHint:    v["readOnlyHint"].(bool),
				}
			default:
				t.Errorf("Unexpected annotations type: %T", annotations)
				continue
			}

			if annot.DestructiveHint != true {
				t.Error("batch_compress destructiveHint should be true")
			}
			if annot.IdempotentHint != true {
				t.Error("batch_compress idempotentHint should be true")
			}
			if annot.ReadOnlyHint != false {
				t.Error("batch_compress readOnlyHint should be false")
			}
		}
	}
}
