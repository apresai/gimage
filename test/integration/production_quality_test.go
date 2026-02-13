//go:build e2e
// +build e2e

package integration

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/apresai/gimage/internal/config"
	"github.com/apresai/gimage/internal/generate"
	"github.com/apresai/gimage/internal/imaging"
	"github.com/apresai/gimage/pkg/models"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
)

// TestSizeEnforcement verifies that generated images are automatically resized
// to match requested dimensions if the model returns a different size
func TestSizeEnforcement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		cfg, err := config.LoadConfig()
		if err != nil || cfg.GeminiAPIKey == "" {
			t.Skip("GEMINI_API_KEY not set, skipping size enforcement test")
		}
		apiKey = cfg.GeminiAPIKey
	}

	client, err := generate.NewGeminiRESTClient(apiKey)
	if err != nil {
		t.Fatalf("Failed to create Gemini client: %v", err)
	}
	defer client.Close()

	testCases := []struct {
		name           string
		requestedSize  string
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "Square 512x512",
			requestedSize:  "512x512",
			expectedWidth:  512,
			expectedHeight: 512,
		},
		{
			name:           "Square 1024x1024",
			requestedSize:  "1024x1024",
			expectedWidth:  1024,
			expectedHeight: 1024,
		},
		{
			name:           "Portrait 512x768",
			requestedSize:  "512x768",
			expectedWidth:  512,
			expectedHeight: 768,
		},
		{
			name:           "Landscape 768x512",
			requestedSize:  "768x512",
			expectedWidth:  768,
			expectedHeight: 512,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			options := models.GenerateOptions{
				Model: "gemini-2.5-flash-image",
				Size:  tc.requestedSize,
			}

			t.Logf("Generating image with requested size: %s", tc.requestedSize)

			results, err := client.GenerateImage(ctx, "a simple test image", options)
			if err != nil {
				t.Fatalf("Image generation failed: %v", err)
			}
			if len(results) == 0 {
				t.Fatal("No images returned")
			}

			// Save to E2E output dir for visibility
			outDir := e2eOutputDir()
			os.MkdirAll(outDir, 0755)
			outputPath := filepath.Join(outDir, fmt.Sprintf("e2e_size_%s.png", tc.requestedSize))

			// Save the image (which should enforce size)
			if err := generate.SaveImage(results[0], outputPath); err != nil {
				t.Fatalf("Failed to save image: %v", err)
			}

			t.Logf("GENERATED_IMAGE: %s", outputPath)

			// Verify the file was created
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				t.Fatal("Output file was not created")
			}

			// Read actual dimensions
			actualWidth, actualHeight, err := getImageDimensions(outputPath)
			if err != nil {
				t.Fatalf("Failed to read image dimensions: %v", err)
			}

			t.Logf("Requested: %dx%d, Got: %dx%d",
				tc.expectedWidth, tc.expectedHeight, actualWidth, actualHeight)

			// Verify dimensions match exactly
			if actualWidth != tc.expectedWidth || actualHeight != tc.expectedHeight {
				t.Errorf("Size enforcement failed: expected %dx%d, got %dx%d",
					tc.expectedWidth, tc.expectedHeight, actualWidth, actualHeight)
			}
		})
	}
}

// TestFormatConversion verifies that images are automatically converted
// to the requested output format regardless of what the API returns
func TestFormatConversion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		cfg, err := config.LoadConfig()
		if err != nil || cfg.GeminiAPIKey == "" {
			t.Skip("GEMINI_API_KEY not set, skipping format conversion test")
		}
		apiKey = cfg.GeminiAPIKey
	}

	client, err := generate.NewGeminiRESTClient(apiKey)
	if err != nil {
		t.Fatalf("Failed to create Gemini client: %v", err)
	}
	defer client.Close()

	testCases := []struct {
		name           string
		outputFormat   string
		expectedFormat string
	}{
		{
			name:           "Convert to WebP",
			outputFormat:   "webp",
			expectedFormat: "webp",
		},
		{
			name:           "Convert to JPEG",
			outputFormat:   "jpg",
			expectedFormat: "jpeg",
		},
		{
			name:           "Convert to PNG",
			outputFormat:   "png",
			expectedFormat: "png",
		},
		{
			name:           "Convert to GIF",
			outputFormat:   "gif",
			expectedFormat: "gif",
		},
		{
			name:           "Convert to BMP",
			outputFormat:   "bmp",
			expectedFormat: "bmp",
		},
		{
			name:           "Convert to TIFF",
			outputFormat:   "tiff",
			expectedFormat: "tiff",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			options := models.GenerateOptions{
				Model: "gemini-2.5-flash-image",
				Size:  "512x512",
			}

			t.Logf("Generating image and converting to %s", tc.outputFormat)

			results, err := client.GenerateImage(ctx, "a simple test image", options)
			if err != nil {
				t.Fatalf("Image generation failed: %v", err)
			}
			if len(results) == 0 {
				t.Fatal("No images returned")
			}

			// Save to E2E output dir for visibility
			outDir := e2eOutputDir()
			os.MkdirAll(outDir, 0755)
			outputPath := filepath.Join(outDir, fmt.Sprintf("e2e_format_test.%s", tc.outputFormat))

			// Save the image (which should convert format)
			if err := generate.SaveImage(results[0], outputPath); err != nil {
				t.Fatalf("Failed to save image: %v", err)
			}

			t.Logf("GENERATED_IMAGE: %s", outputPath)

			// Verify the file was created
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				t.Fatal("Output file was not created")
			}

			// Read and verify format
			actualFormat, err := getImageFormat(outputPath)
			if err != nil {
				t.Fatalf("Failed to detect image format: %v", err)
			}

			t.Logf("Requested format: %s, Detected format: %s", tc.expectedFormat, actualFormat)

			// Verify format matches (normalize jpg/jpeg)
			if normalizeFormat(actualFormat) != normalizeFormat(tc.expectedFormat) {
				t.Errorf("Format conversion failed: expected %s, got %s",
					tc.expectedFormat, actualFormat)
			}

			// Verify the file is readable as that format
			if err := verifyImageReadable(outputPath); err != nil {
				t.Errorf("Generated image is not readable: %v", err)
			}
		})
	}
}

// TestOutputPathHandling verifies that all commands correctly handle
// various output path formats
func TestOutputPathHandling(t *testing.T) {
	// Get a test fixture image
	fixtureDir := "../../test/fixtures"
	fixturePath := filepath.Join(fixtureDir, "test_image_512x512.png")

	// Check if fixture exists, if not create a simple test image
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skipf("Test fixture not found: %s", fixturePath)
	}

	testCases := []struct {
		name        string
		pathType    string
		pathBuilder func(tempDir string) string
	}{
		{
			name:     "Absolute path",
			pathType: "absolute",
			pathBuilder: func(tempDir string) string {
				return filepath.Join(tempDir, "output_absolute.png")
			},
		},
		{
			name:     "Relative path in current dir",
			pathType: "relative",
			pathBuilder: func(tempDir string) string {
				// Create subdirectory in temp
				subdir := filepath.Join(tempDir, "subdir")
				os.MkdirAll(subdir, 0755)
				return filepath.Join(subdir, "output_relative.png")
			},
		},
		{
			name:     "Path with spaces",
			pathType: "spaces",
			pathBuilder: func(tempDir string) string {
				spacedDir := filepath.Join(tempDir, "dir with spaces")
				os.MkdirAll(spacedDir, 0755)
				return filepath.Join(spacedDir, "output image.png")
			},
		},
		{
			name:     "Nested directory creation",
			pathType: "nested",
			pathBuilder: func(tempDir string) string {
				return filepath.Join(tempDir, "level1", "level2", "level3", "output.png")
			},
		},
	}

	// Test resize command
	t.Run("Resize", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tempDir := t.TempDir()
				outputPath := tc.pathBuilder(tempDir)

				// Ensure parent directory exists for nested case
				if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
					t.Fatalf("Failed to create parent directory: %v", err)
				}

				ctx := context.Background()
				err := imaging.ResizeImage(ctx, fixturePath, outputPath, 256, 256)
				if err != nil {
					t.Errorf("Resize failed with %s path: %v", tc.pathType, err)
					return
				}

				// Verify file exists
				if _, err := os.Stat(outputPath); os.IsNotExist(err) {
					t.Errorf("Output file not created at %s", outputPath)
				}

				// Verify dimensions
				width, height, err := getImageDimensions(outputPath)
				if err != nil {
					t.Errorf("Failed to read output dimensions: %v", err)
					return
				}

				if width != 256 || height != 256 {
					t.Errorf("Expected 256x256, got %dx%d", width, height)
				}
			})
		}
	})

	// Test scale command
	t.Run("Scale", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tempDir := t.TempDir()
				outputPath := tc.pathBuilder(tempDir)

				// Ensure parent directory exists for nested case
				if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
					t.Fatalf("Failed to create parent directory: %v", err)
				}

				ctx := context.Background()
				err := imaging.ScaleImage(ctx, fixturePath, outputPath, 0.5)
				if err != nil {
					t.Errorf("Scale failed with %s path: %v", tc.pathType, err)
					return
				}

				// Verify file exists
				if _, err := os.Stat(outputPath); os.IsNotExist(err) {
					t.Errorf("Output file not created at %s", outputPath)
				}
			})
		}
	})

	// Test convert command
	t.Run("Convert", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tempDir := t.TempDir()
				// Change extension to test conversion
				basePath := tc.pathBuilder(tempDir)
				outputPath := basePath[:len(basePath)-4] + ".jpg" // Change .png to .jpg

				// Ensure parent directory exists for nested case
				if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
					t.Fatalf("Failed to create parent directory: %v", err)
				}

				ctx := context.Background()
				err := imaging.ConvertImageFile(ctx, fixturePath, outputPath)
				if err != nil {
					t.Errorf("Convert failed with %s path: %v", tc.pathType, err)
					return
				}

				// Verify file exists
				if _, err := os.Stat(outputPath); os.IsNotExist(err) {
					t.Errorf("Output file not created at %s", outputPath)
				}
			})
		}
	})

	// Test crop command
	t.Run("Crop", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tempDir := t.TempDir()
				outputPath := tc.pathBuilder(tempDir)

				// Ensure parent directory exists for nested case
				if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
					t.Fatalf("Failed to create parent directory: %v", err)
				}

				ctx := context.Background()
				err := imaging.CropImage(ctx, fixturePath, outputPath, 0, 0, 128, 128)
				if err != nil {
					t.Errorf("Crop failed with %s path: %v", tc.pathType, err)
					return
				}

				// Verify file exists
				if _, err := os.Stat(outputPath); os.IsNotExist(err) {
					t.Errorf("Output file not created at %s", outputPath)
				}

				// Verify dimensions
				width, height, err := getImageDimensions(outputPath)
				if err != nil {
					t.Errorf("Failed to read output dimensions: %v", err)
					return
				}

				if width != 128 || height != 128 {
					t.Errorf("Expected 128x128, got %dx%d", width, height)
				}
			})
		}
	})
}

// Helper functions

func getImageDimensions(path string) (width, height int, err error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode image config: %w", err)
	}

	return config.Width, config.Height, nil
}

func getImageFormat(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	_, format, err := image.DecodeConfig(file)
	if err != nil {
		return "", fmt.Errorf("failed to decode image config: %w", err)
	}

	return format, nil
}

func verifyImageReadable(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	_, _, err = image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	return nil
}

func normalizeFormat(format string) string {
	switch format {
	case "jpg", "jpeg":
		return "jpeg"
	case "tif", "tiff":
		return "tiff"
	default:
		return format
	}
}
