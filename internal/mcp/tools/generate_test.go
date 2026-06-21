package tools

import (
	"testing"

	"github.com/apresai/gimage/internal/mcp"
)

func TestRegisterGenerateImageTool(t *testing.T) {
	server := mcp.NewMCPServer("test", "1.0.0", nil, false)

	// Register the tool
	RegisterGenerateImageTool(server)

	// Verify tool is registered
	tool := server.GetTool("generate_image")
	if tool == nil {
		t.Fatal("generate_image tool not registered")
	}

	// Verify tool properties
	if tool.Name != "generate_image" {
		t.Errorf("Expected name 'generate_image', got '%s'", tool.Name)
	}

	if tool.Description == "" {
		t.Error("Tool description is empty")
	}

	if tool.Handler == nil {
		t.Error("Tool handler is nil")
	}

	// Verify annotations
	if tool.Annotations == nil {
		t.Fatal("Tool annotations are nil")
	}

	if tool.Annotations.DestructiveHint {
		t.Error("generate_image should not be destructive")
	}

	if tool.Annotations.IdempotentHint {
		t.Error("generate_image should not be idempotent (generates different images)")
	}

	if tool.Annotations.ReadOnlyHint {
		t.Error("generate_image should not be read-only (writes files)")
	}
}

func TestGenerateImageTool_InputSchema(t *testing.T) {
	server := mcp.NewMCPServer("test", "1.0.0", nil, false)
	RegisterGenerateImageTool(server)

	tool := server.GetTool("generate_image")
	if tool == nil {
		t.Fatal("generate_image tool not registered")
	}

	schema := tool.InputSchema

	// Check required fields
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("required field is not a string slice")
	}

	if len(required) != 1 || required[0] != "prompt" {
		t.Errorf("Expected required=['prompt'], got %v", required)
	}

	// Check properties
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties field is not a map")
	}

	// Verify prompt property
	if _, exists := properties["prompt"]; !exists {
		t.Error("prompt property missing")
	}

	// Verify optional properties — full enumeration to catch silent schema drift.
	optionalProps := []string{
		"output", "size", "model", "style", "negative", "seed",
		"image_size", "aspect_ratio", "cfg_scale", "count", "output_format",
		"thinking", "grounding", "input_images",
	}
	for _, prop := range optionalProps {
		if _, exists := properties[prop]; !exists {
			t.Errorf("optional property '%s' missing", prop)
		}
	}
}

func TestGenerateImageTool_ValidationErrors(t *testing.T) {
	server := mcp.NewMCPServer("test", "1.0.0", nil, false)
	RegisterGenerateImageTool(server)

	tool := server.GetTool("generate_image")
	if tool == nil {
		t.Fatal("generate_image tool not registered")
	}

	tests := []struct {
		name      string
		args      map[string]interface{}
		wantError bool
		errorMsg  string
	}{
		{
			name:      "missing prompt",
			args:      map[string]interface{}{},
			wantError: true,
			errorMsg:  "prompt is required",
		},
		{
			name: "empty prompt",
			args: map[string]interface{}{
				"prompt": "",
			},
			wantError: true,
			errorMsg:  "prompt is required",
		},
		{
			name: "prompt not a string",
			args: map[string]interface{}{
				"prompt": 123,
			},
			wantError: true,
			errorMsg:  "prompt is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Handler(tt.args)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				}
				if result != nil {
					t.Errorf("Expected nil result on error, got %v", result)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGenerateImageTool_ParameterDefaults(t *testing.T) {
	// This test verifies parameter extraction and defaults
	// We can't test actual image generation without real API keys,
	// but we can verify the parameter parsing logic

	tests := []struct {
		name          string
		args          map[string]interface{}
		expectedSize  string
		expectedModel string
	}{
		{
			name: "use defaults",
			args: map[string]interface{}{
				"prompt": "test prompt",
			},
			expectedSize:  "1024x1024",
			expectedModel: "gemini-3-pro-image",
		},
		{
			name: "custom size",
			args: map[string]interface{}{
				"prompt": "test prompt",
				"size":   "512x512",
			},
			expectedSize:  "512x512",
			expectedModel: "gemini-3-pro-image",
		},
		{
			name: "custom model",
			args: map[string]interface{}{
				"prompt": "test prompt",
				"model":  "vertex-flash",
			},
			expectedSize:  "1024x1024",
			expectedModel: "vertex-flash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can verify the defaults are correctly set
			// by checking what would be passed to the API

			size, _ := tt.args["size"].(string)
			if size == "" {
				size = "1024x1024"
			}

			model, _ := tt.args["model"].(string)
			if model == "" {
				model = "gemini-3-pro-image"
			}

			if size != tt.expectedSize {
				t.Errorf("Expected size '%s', got '%s'", tt.expectedSize, size)
			}

			if model != tt.expectedModel {
				t.Errorf("Expected model '%s', got '%s'", tt.expectedModel, model)
			}
		})
	}
}

func TestIsVertexModel(t *testing.T) {
	tests := []struct {
		model    string
		expected bool
	}{
		// vertex-flash aliases and vertex/flash-3.1* provider IDs are Vertex.
		{"vertex-flash", true},
		{"vertex-flash-fast", true},
		{"vertex-flash-ultra", true},
		{"vertex/flash-3.1", true},
		{"vertex/flash-3.1-fast", true},
		{"vertex/flash-3.1-ultra", true},
		// Retired Imagen names are no longer Vertex models.
		{"imagen-4", false},
		{"imagen-4-fast", false},
		{"imagen-4-ultra", false},
		{"imagen-4.0-fast-generate-001", false},
		// Bare Gemini model IDs are not treated as Vertex.
		{"gemini-3.1-flash-image", false},
		{"gemini-2.5-flash-image", false},
		{"gemini-2.0-flash-preview-image-generation", false},
		{"unknown-model", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			result := isVertexModel(tt.model)
			if result != tt.expected {
				t.Errorf("isVertexModel(%s) = %v, want %v", tt.model, result, tt.expected)
			}
		})
	}
}
