package integration

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/apresai/gimage/internal/config"
	"github.com/apresai/gimage/internal/imaging"
	"github.com/apresai/gimage/internal/mcp"
	"github.com/apresai/gimage/internal/mcp/tools"
)

// TestMCPToolsE2E tests all 10 MCP tools end-to-end
func TestMCPToolsE2E(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{}

	// Create test image file
	testImagePath := filepath.Join(tmpDir, "test.png")
	img := createTestImage(100, 100)
	if err := saveTestImage(img, testImagePath); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	// Initialize MCP server with all tools
	server := mcp.NewMCPServer("gimage-e2e-test", "1.0.0-test", cfg, false)
	tools.RegisterGenerateImageTool(server)
	tools.RegisterConvertImageTool(server)
	tools.RegisterResizeImageTool(server)
	tools.RegisterScaleImageTool(server)
	tools.RegisterCropImageTool(server)
	tools.RegisterCompressImageTool(server)
	tools.RegisterBatchResizeTool(server)
	tools.RegisterBatchCompressTool(server)
	tools.RegisterBatchConvertTool(server)
	tools.RegisterListModelsTool(server)

	t.Run("Tool1_ListModels", func(t *testing.T) {
		result, err := callTool(server, "list_models", map[string]interface{}{})
		if err != nil {
			t.Fatalf("list_models failed: %v", err)
		}

		// Verify result contains provider information
		t.Logf("list_models result: %+v", result)
		providers, ok := result["providers"]
		if !ok {
			t.Fatalf("No 'providers' key in result. Keys: %v", getKeys(result))
		}

		providersList, ok := providers.([]map[string]interface{})
		if !ok {
			t.Fatalf("providers is not []map[string]interface{}, got %T", providers)
		}

		if len(providersList) == 0 {
			t.Fatal("Expected non-empty providers list from list_models")
		}

		t.Logf("✓ list_models tool works (%d providers)", len(providersList))
	})

	t.Run("Tool2_ResizeImage", func(t *testing.T) {
		outputPath := filepath.Join(tmpDir, "resized.png")

		result, err := callTool(server, "resize_image", map[string]interface{}{
			"input":  testImagePath,
			"width":  50.0,
			"height": 50.0,
			"output": outputPath,
		})
		if err != nil {
			t.Fatalf("resize_image failed: %v", err)
		}

		// Verify output file exists
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatal("Resized image not created")
		}

		t.Logf("✓ resize_image tool works: %v", result)
	})

	t.Run("Tool3_ScaleImage", func(t *testing.T) {
		outputPath := filepath.Join(tmpDir, "scaled.png")

		result, err := callTool(server, "scale_image", map[string]interface{}{
			"input":  testImagePath,
			"factor": 0.5,
			"output": outputPath,
		})
		if err != nil {
			t.Fatalf("scale_image failed: %v", err)
		}

		// Verify output file exists
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatal("Scaled image not created")
		}

		t.Logf("✓ scale_image tool works: %v", result)
	})

	t.Run("Tool4_CropImage", func(t *testing.T) {
		outputPath := filepath.Join(tmpDir, "cropped.png")

		result, err := callTool(server, "crop_image", map[string]interface{}{
			"input":  testImagePath,
			"x":      10.0,
			"y":      10.0,
			"width":  50.0,
			"height": 50.0,
			"output": outputPath,
		})
		if err != nil {
			t.Fatalf("crop_image failed: %v", err)
		}

		// Verify output file exists
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatal("Cropped image not created")
		}

		t.Logf("✓ crop_image tool works: %v", result)
	})

	t.Run("Tool5_CompressImage", func(t *testing.T) {
		// Create a JPEG for compression test
		jpegPath := filepath.Join(tmpDir, "test.jpg")
		if err := saveTestImageAsJPEG(img, jpegPath); err != nil {
			t.Fatalf("Failed to create JPEG: %v", err)
		}

		outputPath := filepath.Join(tmpDir, "compressed.jpg")

		_, err := callTool(server, "compress_image", map[string]interface{}{
			"input":   jpegPath,
			"quality": 75.0,
			"output":  outputPath,
		})
		if err != nil {
			t.Fatalf("compress_image failed: %v", err)
		}

		// Verify output file exists
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatal("Compressed image not created")
		}

		t.Log("✓ compress_image tool works")
	})

	t.Run("Tool6_ConvertImage", func(t *testing.T) {
		outputPath := filepath.Join(tmpDir, "converted.webp")

		_, err := callTool(server, "convert_image", map[string]interface{}{
			"input":  testImagePath,
			"format": "webp",
			"output": outputPath,
		})
		if err != nil {
			t.Fatalf("convert_image failed: %v", err)
		}

		// Verify output file exists
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatal("Converted image not created")
		}

		t.Log("✓ convert_image tool works (PNG → WebP)")
	})

	t.Run("Tool7_BatchResize", func(t *testing.T) {
		// Create batch directory with test images
		batchDir := filepath.Join(tmpDir, "batch_resize")
		if err := os.MkdirAll(batchDir, 0755); err != nil {
			t.Fatalf("Failed to create batch dir: %v", err)
		}

		// Create test images in batch directory
		for i := 1; i <= 3; i++ {
			imgPath := filepath.Join(batchDir, fmt.Sprintf("image%d.png", i))
			if err := saveTestImage(img, imgPath); err != nil {
				t.Fatalf("Failed to create image%d: %v", i, err)
			}
		}

		outputDir := filepath.Join(tmpDir, "batch_resize_out")
		_, err := callTool(server, "batch_resize", map[string]interface{}{
			"input_dir":  batchDir,
			"output_dir": outputDir,
			"width":      40.0,
			"height":     40.0,
		})
		if err != nil {
			t.Fatalf("batch_resize failed: %v", err)
		}

		t.Log("✓ batch_resize tool works")
	})

	t.Run("Tool8_BatchCompress", func(t *testing.T) {
		// Create batch directory with JPEG images
		batchDir := filepath.Join(tmpDir, "batch_compress")
		if err := os.MkdirAll(batchDir, 0755); err != nil {
			t.Fatalf("Failed to create batch dir: %v", err)
		}

		// Create test JPEGs in batch directory
		for i := 1; i <= 3; i++ {
			imgPath := filepath.Join(batchDir, fmt.Sprintf("image%d.jpg", i))
			if err := saveTestImageAsJPEG(img, imgPath); err != nil {
				t.Fatalf("Failed to create image%d: %v", i, err)
			}
		}

		outputDir := filepath.Join(tmpDir, "batch_compress_out")
		_, err := callTool(server, "batch_compress", map[string]interface{}{
			"input_dir":  batchDir,
			"output_dir": outputDir,
			"quality":    80.0,
		})
		if err != nil {
			t.Fatalf("batch_compress failed: %v", err)
		}

		t.Log("✓ batch_compress tool works")
	})

	t.Run("Tool9_BatchConvert", func(t *testing.T) {
		// Create batch directory with PNG images
		batchDir := filepath.Join(tmpDir, "batch_convert")
		if err := os.MkdirAll(batchDir, 0755); err != nil {
			t.Fatalf("Failed to create batch dir: %v", err)
		}

		// Create test PNG in batch directory
		imgPath := filepath.Join(batchDir, "image.png")
		if err := saveTestImage(img, imgPath); err != nil {
			t.Fatalf("Failed to create image: %v", err)
		}

		outputDir := filepath.Join(tmpDir, "batch_convert_out")
		_, err := callTool(server, "batch_convert", map[string]interface{}{
			"input_dir":  batchDir,
			"output_dir": outputDir,
			"format":     "jpg",
		})
		if err != nil {
			t.Fatalf("batch_convert failed: %v", err)
		}

		t.Log("✓ batch_convert tool works")
	})

	t.Run("Tool10_GenerateImage_Validation", func(t *testing.T) {
		// Test generate_image with invalid parameters (no API key configured)
		// This should fail validation, not actual generation
		_, err := callTool(server, "generate_image", map[string]interface{}{
			"prompt": "", // Empty prompt should fail validation
		})
		if err == nil {
			t.Fatal("Expected error for empty prompt, got nil")
		}

		t.Log("✓ generate_image tool validation works")
	})

	t.Run("Tool11_GenerateImage_RealGeneration", func(t *testing.T) {
		// Only run if GEMINI_API_KEY is set (uses API quota)
		if os.Getenv("GEMINI_API_KEY") == "" {
			t.Skip("Skipping real image generation (set GEMINI_API_KEY to enable)")
		}

		outputPath := filepath.Join(tmpDir, "generated-test.png")

		result, err := callTool(server, "generate_image", map[string]interface{}{
			"prompt": "a simple red square on white background",
			"output": outputPath,
			"size":   "512x512", // Smaller size for faster test
			"model":  "gemini-nano-banana-2.1",
		})
		if err != nil {
			t.Fatalf("generate_image failed: %v", err)
		}

		// Verify output file exists
		info, err := os.Stat(outputPath)
		if os.IsNotExist(err) {
			t.Fatal("Generated image not created")
		}

		if info.Size() == 0 {
			t.Fatal("Generated image file is empty")
		}

		t.Logf("✓ generate_image tool works: %s (%d bytes)", outputPath, info.Size())
		t.Logf("  Result: %v", result)
	})
}

// TestMCPConvertWebP specifically tests WebP conversion via MCP
func TestMCPConvertWebP(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{}

	// Create test images in different formats
	pngPath := filepath.Join(tmpDir, "test.png")
	jpgPath := filepath.Join(tmpDir, "test.jpg")

	img := createTestImage(50, 50)
	if err := saveTestImage(img, pngPath); err != nil {
		t.Fatalf("Failed to create PNG: %v", err)
	}
	if err := saveTestImageAsJPEG(img, jpgPath); err != nil {
		t.Fatalf("Failed to create JPG: %v", err)
	}

	// Initialize MCP server
	server := mcp.NewMCPServer("gimage-webp-test", "1.0.0-test", cfg, false)
	tools.RegisterConvertImageTool(server)

	tests := []struct {
		name       string
		inputPath  string
		outputPath string
	}{
		{
			name:       "PNG to WebP",
			inputPath:  pngPath,
			outputPath: filepath.Join(tmpDir, "png_to_webp.webp"),
		},
		{
			name:       "JPG to WebP",
			inputPath:  jpgPath,
			outputPath: filepath.Join(tmpDir, "jpg_to_webp.webp"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := callTool(server, "convert_image", map[string]interface{}{
				"input":  tt.inputPath,
				"format": "webp",
				"output": tt.outputPath,
			})
			if err != nil {
				t.Fatalf("Conversion failed: %v", err)
			}

			// Verify output file exists
			info, err := os.Stat(tt.outputPath)
			if os.IsNotExist(err) {
				t.Fatal("WebP file not created")
			}

			if info.Size() == 0 {
				t.Fatal("WebP file is empty")
			}

			t.Logf("✓ %s successful: %s (%d bytes)", tt.name, tt.outputPath, info.Size())
			t.Logf("  Result: %v", result)
		})
	}
}

// Helper functions

func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: 128,
				A: 255,
			})
		}
	}
	return img
}

func saveTestImage(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func saveTestImageAsJPEG(img image.Image, path string) error {
	// Use imaging package to save as JPEG
	return imaging.SaveJPEG(img, path, 90)
}

func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func callTool(server *mcp.MCPServer, toolName string, args map[string]interface{}) (map[string]interface{}, error) {
	// Get the tool from the server
	tool := server.GetTool(toolName)
	if tool == nil {
		return nil, fmt.Errorf("tool not found: %s", toolName)
	}

	// Call the tool handler directly
	result, err := tool.Handler(args)
	if err != nil {
		return nil, err
	}

	return result, nil
}
