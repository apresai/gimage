package tools

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/apresai/gimage/internal/config"
	"github.com/apresai/gimage/internal/mcp"
)

func TestResizeImageTool(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{}

	// Create a test image (200x200)
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

	// Create server and register tool
	server := mcp.NewMCPServer("test", "1.0.0", cfg, false)
	RegisterResizeImageTool(server)

	tests := []struct {
		name       string
		args       map[string]interface{}
		wantError  bool
		checkWidth int
	}{
		{
			name: "valid resize",
			args: map[string]interface{}{
				"input":  testImagePath,
				"width":  100.0,
				"height": 100.0,
			},
			wantError:  false,
			checkWidth: 100,
		},
		{
			name: "resize with output path",
			args: map[string]interface{}{
				"input":  testImagePath,
				"width":  150.0,
				"height": 150.0,
				"output": filepath.Join(tmpDir, "resized_custom.png"),
			},
			wantError:  false,
			checkWidth: 150,
		},
		{
			name: "missing width",
			args: map[string]interface{}{
				"input":  testImagePath,
				"height": 100.0,
			},
			wantError: true,
		},
		{
			name: "missing height",
			args: map[string]interface{}{
				"input": testImagePath,
				"width": 100.0,
			},
			wantError: true,
		},
		{
			name: "zero width",
			args: map[string]interface{}{
				"input":  testImagePath,
				"width":  0.0,
				"height": 100.0,
			},
			wantError: true,
		},
		{
			name: "negative dimensions",
			args: map[string]interface{}{
				"input":  testImagePath,
				"width":  -100.0,
				"height": 100.0,
			},
			wantError: true,
		},
		{
			name: "nonexistent input file",
			args: map[string]interface{}{
				"input":  filepath.Join(tmpDir, "nonexistent.png"),
				"width":  100.0,
				"height": 100.0,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := server.GetTool("resize_image")
			if tool == nil {
				t.Fatal("resize_image tool not registered")
			}

			result, err := tool.Handler(tt.args)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify result structure
			success, ok := result["success"].(bool)
			if !ok || !success {
				t.Error("Expected success: true in result")
			}

			outputPath, ok := result["output_path"].(string)
			if !ok {
				t.Error("Expected output_path in result")
				return
			}

			// Verify output file exists
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				t.Errorf("Output file not created: %s", outputPath)
				return
			}

			// Verify dimensions
			width, _, err := getImageDimensions(outputPath)
			if err != nil {
				t.Errorf("Failed to get dimensions: %v", err)
				return
			}

			if width != tt.checkWidth {
				t.Errorf("Expected width %d, got %d", tt.checkWidth, width)
			}

			// Verify new_size in result
			newSize, ok := result["new_size"].(string)
			if !ok {
				t.Error("Expected new_size in result")
			} else {
				expectedSize := formatDimensions(tt.checkWidth, int(tt.args["height"].(float64)))
				if newSize != expectedSize {
					t.Errorf("Expected new_size %s, got %s", expectedSize, newSize)
				}
			}

			// Verify original_size in result
			originalSize, ok := result["original_size"].(string)
			if !ok {
				t.Error("Expected original_size in result")
			} else if originalSize != "200x200" {
				t.Errorf("Expected original_size 200x200, got %s", originalSize)
			}
		})
	}
}

func TestResizeImageToolSchema(t *testing.T) {
	cfg := &config.Config{}
	server := mcp.NewMCPServer("test", "1.0.0", cfg, false)
	RegisterResizeImageTool(server)

	tool := server.GetTool("resize_image")
	if tool == nil {
		t.Fatal("resize_image tool not registered")
	}

	// Verify schema structure
	schema := tool.InputSchema
	if schema["type"] != "object" {
		t.Error("Expected type 'object' in schema")
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties in schema")
	}

	// Check required fields
	requiredFields := []string{"input", "width", "height"}
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("Expected required array in schema")
	}

	for _, field := range requiredFields {
		found := false
		for _, req := range required {
			if req == field {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Required field '%s' not in schema", field)
		}

		if _, exists := properties[field]; !exists {
			t.Errorf("Property '%s' not defined in schema", field)
		}
	}

	// Verify optional output field exists
	if _, exists := properties["output"]; !exists {
		t.Error("Optional 'output' property not defined in schema")
	}
}

func formatDimensions(width, height int) string {
	return fmt.Sprintf("%dx%d", width, height)
}

func TestResizeImageToolModes(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{}

	// Create a test image (200x100, 2:1 aspect ratio)
	testImagePath := filepath.Join(tmpDir, "landscape.png")
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
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

	// Create server and register tool
	server := mcp.NewMCPServer("test", "1.0.0", cfg, false)
	RegisterResizeImageTool(server)

	tool := server.GetTool("resize_image")
	if tool == nil {
		t.Fatal("resize_image tool not registered")
	}

	tests := []struct {
		name           string
		mode           string
		targetWidth    int
		targetHeight   int
		expectedWidth  int
		expectedHeight int
		description    string
	}{
		{
			name:           "crop mode - preserves ratio, fills target, crops excess",
			mode:           "crop",
			targetWidth:    100,
			targetHeight:   100,
			expectedWidth:  100, // exact target because it crops
			expectedHeight: 100,
			description:    "200x100 (2:1) -> 100x100 square: scales up height, crops width",
		},
		{
			name:           "fit mode - preserves ratio, fits within bounds",
			mode:           "fit",
			targetWidth:    100,
			targetHeight:   100,
			expectedWidth:  100, // fits within 100x100, preserving 2:1 ratio
			expectedHeight: 50,  // 100/2 = 50 to maintain 2:1 ratio
			description:    "200x100 (2:1) -> 100x50 within 100x100 bounds",
		},
		{
			name:           "stretch mode - forces exact dimensions, may distort",
			mode:           "stretch",
			targetWidth:    100,
			targetHeight:   100,
			expectedWidth:  100, // exact target
			expectedHeight: 100, // exact target (distorted from 2:1 to 1:1)
			description:    "200x100 (2:1) -> 100x100 forced (distorted)",
		},
		{
			name:           "default mode is crop",
			mode:           "", // empty mode should default to crop
			targetWidth:    100,
			targetHeight:   100,
			expectedWidth:  100,
			expectedHeight: 100,
			description:    "empty mode defaults to crop behavior",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := filepath.Join(tmpDir, fmt.Sprintf("resized_%s.png", tt.name))

			args := map[string]interface{}{
				"input":  testImagePath,
				"width":  float64(tt.targetWidth),
				"height": float64(tt.targetHeight),
				"output": outputPath,
			}

			if tt.mode != "" {
				args["mode"] = tt.mode
			}

			result, err := tool.Handler(args)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify success
			success, ok := result["success"].(bool)
			if !ok || !success {
				t.Error("Expected success: true in result")
			}

			// Verify output dimensions
			width, height, err := getImageDimensions(outputPath)
			if err != nil {
				t.Fatalf("Failed to get dimensions: %v", err)
			}

			if width != tt.expectedWidth || height != tt.expectedHeight {
				t.Errorf("%s: expected %dx%d, got %dx%d",
					tt.description, tt.expectedWidth, tt.expectedHeight, width, height)
			}

			// Verify mode is returned in result (except for empty mode which defaults)
			if tt.mode != "" {
				resultMode, ok := result["mode"].(string)
				if !ok {
					t.Error("Expected mode in result")
				} else if resultMode != tt.mode {
					t.Errorf("Expected mode %s in result, got %s", tt.mode, resultMode)
				}
			}
		})
	}
}

func TestResizeImageToolModeSchema(t *testing.T) {
	cfg := &config.Config{}
	server := mcp.NewMCPServer("test", "1.0.0", cfg, false)
	RegisterResizeImageTool(server)

	tool := server.GetTool("resize_image")
	if tool == nil {
		t.Fatal("resize_image tool not registered")
	}

	// Verify mode is in schema
	properties, ok := tool.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties in schema")
	}

	modeSchema, exists := properties["mode"].(map[string]interface{})
	if !exists {
		t.Fatal("Expected 'mode' property in schema")
	}

	// Verify mode type
	if modeSchema["type"] != "string" {
		t.Error("Expected mode type 'string'")
	}

	// Verify mode enum values
	enumValues, ok := modeSchema["enum"].([]string)
	if !ok {
		t.Fatal("Expected enum array for mode")
	}

	expectedModes := []string{"crop", "fit", "stretch"}
	for _, expected := range expectedModes {
		found := false
		for _, actual := range enumValues {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected mode '%s' in enum", expected)
		}
	}

	// Verify default is crop
	if modeSchema["default"] != "crop" {
		t.Error("Expected mode default to be 'crop'")
	}
}

func TestResizeImageToolStringCoercion(t *testing.T) {
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

	// Create server and register tool
	server := mcp.NewMCPServer("test", "1.0.0", cfg, false)
	RegisterResizeImageTool(server)

	tool := server.GetTool("resize_image")
	if tool == nil {
		t.Fatal("resize_image tool not registered")
	}

	// Test with string numbers (LLM common behavior)
	args := map[string]interface{}{
		"input":  testImagePath,
		"width":  "100", // string instead of float64
		"height": "100", // string instead of float64
		"output": filepath.Join(tmpDir, "resized_string.png"),
	}

	result, err := tool.Handler(args)
	if err != nil {
		t.Fatalf("String coercion failed: %v", err)
	}

	success, ok := result["success"].(bool)
	if !ok || !success {
		t.Error("Expected success: true in result")
	}

	// Verify output dimensions
	width, height, err := getImageDimensions(result["output_path"].(string))
	if err != nil {
		t.Fatalf("Failed to get dimensions: %v", err)
	}

	if width != 100 || height != 100 {
		t.Errorf("Expected 100x100, got %dx%d", width, height)
	}
}
