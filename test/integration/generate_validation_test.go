//go:build integration
// +build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/apresai/gimage/internal/config"
	"github.com/apresai/gimage/internal/generate"
	"github.com/apresai/gimage/pkg/models"
)

// TestSuite provides comprehensive validation of image generation
// Run with: go test -tags=integration -v ./test/integration/...

// ImageValidationResult contains the results of image validation
type ImageValidationResult struct {
	Width         int
	Height        int
	AspectRatio   string
	Format        string
	FileSizeBytes int64
	IsValid       bool
	Errors        []string
}

// ValidateImage checks that a generated image meets the expected specifications
func ValidateImage(data []byte, expectedWidth, expectedHeight int, expectedFormat string) (*ImageValidationResult, error) {
	result := &ImageValidationResult{
		Errors: []string{},
	}

	// Decode the image to get actual dimensions
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	result.Width = bounds.Dx()
	result.Height = bounds.Dy()
	result.Format = format
	result.FileSizeBytes = int64(len(data))

	// Calculate aspect ratio
	gcd := gcdFunc(result.Width, result.Height)
	result.AspectRatio = fmt.Sprintf("%d:%d", result.Width/gcd, result.Height/gcd)

	// Validate dimensions
	// Allow some tolerance for APIs that don't guarantee exact dimensions
	widthTolerance := int(float64(expectedWidth) * 0.1) // 10% tolerance
	heightTolerance := int(float64(expectedHeight) * 0.1)

	if expectedWidth > 0 {
		if abs(result.Width-expectedWidth) > widthTolerance {
			result.Errors = append(result.Errors, fmt.Sprintf("width mismatch: expected ~%d, got %d", expectedWidth, result.Width))
		}
	}

	if expectedHeight > 0 {
		if abs(result.Height-expectedHeight) > heightTolerance {
			result.Errors = append(result.Errors, fmt.Sprintf("height mismatch: expected ~%d, got %d", expectedHeight, result.Height))
		}
	}

	// Validate format
	if expectedFormat != "" {
		normalizedExpected := normalizeFormat(expectedFormat)
		normalizedActual := normalizeFormat(format)
		if normalizedExpected != normalizedActual {
			result.Errors = append(result.Errors, fmt.Sprintf("format mismatch: expected %s, got %s", expectedFormat, format))
		}
	}

	result.IsValid = len(result.Errors) == 0
	return result, nil
}

// ValidateAspectRatio checks if the image matches the expected aspect ratio
func ValidateAspectRatio(data []byte, expectedRatio string) (*ImageValidationResult, error) {
	result := &ImageValidationResult{
		Errors: []string{},
	}

	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	result.Width = bounds.Dx()
	result.Height = bounds.Dy()
	result.Format = format
	result.FileSizeBytes = int64(len(data))

	// Calculate actual aspect ratio
	gcd := gcdFunc(result.Width, result.Height)
	result.AspectRatio = fmt.Sprintf("%d:%d", result.Width/gcd, result.Height/gcd)

	// Parse expected ratio
	expectedRatioValue := parseAspectRatio(expectedRatio)
	actualRatioValue := float64(result.Width) / float64(result.Height)

	// Allow 5% tolerance for aspect ratio matching
	tolerance := 0.05
	diff := abs64(expectedRatioValue - actualRatioValue)
	if diff > expectedRatioValue*tolerance {
		result.Errors = append(result.Errors, fmt.Sprintf("aspect ratio mismatch: expected %s (%.4f), got %s (%.4f)",
			expectedRatio, expectedRatioValue, result.AspectRatio, actualRatioValue))
	}

	result.IsValid = len(result.Errors) == 0
	return result, nil
}

// Helper functions
func gcdFunc(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func abs64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func normalizeFormat(format string) string {
	format = strings.ToLower(format)
	switch format {
	case "jpg", "jpeg":
		return "jpeg"
	default:
		return format
	}
}

func parseAspectRatio(ratio string) float64 {
	parts := strings.Split(ratio, ":")
	if len(parts) != 2 {
		return 1.0
	}
	var w, h float64
	fmt.Sscanf(parts[0], "%f", &w)
	fmt.Sscanf(parts[1], "%f", &h)
	if h == 0 {
		return 1.0
	}
	return w / h
}

// =============================================================================
// Integration Tests
// =============================================================================

// TestGeminiSquareImage tests that Gemini generates a square image when requested
func TestGeminiSquareImage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiKey, err := config.GetGeminiAPIKey("")
	if err != nil {
		t.Skipf("Gemini API key not configured: %v", err)
	}

	client, err := generate.NewGeminiRESTClient(apiKey)
	if err != nil {
		t.Fatalf("Failed to create Gemini client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	opts := models.GenerateOptions{
		Model:       "gemini-2.5-flash-image",
		Size:        "1024x1024",
		AspectRatio: "1:1",
	}

	result, err := client.GenerateImage(ctx, "a simple red circle on white background", opts)
	if err != nil {
		t.Fatalf("Generation failed: %v", err)
	}

	validation, err := ValidateImage(result.Data, 1024, 1024, "png")
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	t.Logf("Generated image: %dx%d, format=%s, size=%d bytes",
		validation.Width, validation.Height, validation.Format, validation.FileSizeBytes)

	if !validation.IsValid {
		for _, e := range validation.Errors {
			t.Errorf("Validation error: %s", e)
		}
	}
}

// TestGemini3ProAspectRatio tests that Gemini 3 Pro respects aspect ratio settings
func TestGemini3ProAspectRatio(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiKey, err := config.GetGeminiAPIKey("")
	if err != nil {
		t.Skipf("Gemini API key not configured: %v", err)
	}

	client, err := generate.NewGeminiRESTClient(apiKey)
	if err != nil {
		t.Fatalf("Failed to create Gemini client: %v", err)
	}
	defer client.Close()

	testCases := []struct {
		name          string
		aspectRatio   string
		imageSize     string
		expectedRatio float64
	}{
		{"Square 1:1", "1:1", "1K", 1.0},
		{"Landscape 16:9", "16:9", "1K", 16.0 / 9.0},
		{"Portrait 9:16", "9:16", "1K", 9.0 / 16.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			opts := models.GenerateOptions{
				Model:       "gemini-3-pro-image",
				AspectRatio: tc.aspectRatio,
				ImageSize:   tc.imageSize,
			}

			result, err := client.GenerateImage(ctx, "a simple blue square", opts)
			if err != nil {
				t.Fatalf("Generation failed: %v", err)
			}

			validation, err := ValidateAspectRatio(result.Data, tc.aspectRatio)
			if err != nil {
				t.Fatalf("Validation failed: %v", err)
			}

			t.Logf("Generated image: %dx%d (ratio=%s), expected ratio=%s",
				validation.Width, validation.Height, validation.AspectRatio, tc.aspectRatio)

			if !validation.IsValid {
				for _, e := range validation.Errors {
					t.Errorf("Validation error: %s", e)
				}
			}
		})
	}
}

// TestGemini3ProImageSize tests native resolution options
func TestGemini3ProImageSize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiKey, err := config.GetGeminiAPIKey("")
	if err != nil {
		t.Skipf("Gemini API key not configured: %v", err)
	}

	client, err := generate.NewGeminiRESTClient(apiKey)
	if err != nil {
		t.Fatalf("Failed to create Gemini client: %v", err)
	}
	defer client.Close()

	// Test that 1K produces smaller images than 2K
	testCases := []struct {
		name      string
		imageSize string
		minPixels int // Minimum total pixels expected
		maxPixels int // Maximum total pixels expected
	}{
		{"1K resolution", "1K", 500000, 1500000},  // ~1MP
		{"2K resolution", "2K", 2000000, 5000000}, // ~4MP
		// Skip 4K in tests as it's expensive ($0.24/image)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			opts := models.GenerateOptions{
				Model:       "gemini-3-pro-image",
				ImageSize:   tc.imageSize,
				AspectRatio: "1:1", // Use square for easier pixel counting
			}

			result, err := client.GenerateImage(ctx, "a simple green triangle", opts)
			if err != nil {
				t.Fatalf("Generation failed: %v", err)
			}

			validation, err := ValidateImage(result.Data, 0, 0, "")
			if err != nil {
				t.Fatalf("Validation failed: %v", err)
			}

			totalPixels := validation.Width * validation.Height
			t.Logf("Generated %s image: %dx%d = %d pixels",
				tc.imageSize, validation.Width, validation.Height, totalPixels)

			if totalPixels < tc.minPixels {
				t.Errorf("Image too small: got %d pixels, expected at least %d", totalPixels, tc.minPixels)
			}
			if totalPixels > tc.maxPixels {
				t.Errorf("Image too large: got %d pixels, expected at most %d", totalPixels, tc.maxPixels)
			}
		})
	}
}

// TestSeedReproducibility tests that the same seed produces similar results
// Note: Not all APIs guarantee exact reproducibility, but results should be similar
func TestSeedReproducibility(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiKey, err := config.GetGeminiAPIKey("")
	if err != nil {
		t.Skipf("Gemini API key not configured: %v", err)
	}

	client, err := generate.NewGeminiRESTClient(apiKey)
	if err != nil {
		t.Fatalf("Failed to create Gemini client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	seed := int64(12345)
	prompt := "a minimalist sunset with orange sky"

	opts := models.GenerateOptions{
		Model:       "gemini-2.5-flash-image",
		Size:        "512x512",
		AspectRatio: "1:1",
		Seed:        seed,
	}

	// Generate first image
	result1, err := client.GenerateImage(ctx, prompt, opts)
	if err != nil {
		t.Fatalf("First generation failed: %v", err)
	}

	// Generate second image with same seed
	result2, err := client.GenerateImage(ctx, prompt, opts)
	if err != nil {
		t.Fatalf("Second generation failed: %v", err)
	}

	// Both images should exist and be valid
	validation1, _ := ValidateImage(result1.Data, 0, 0, "")
	validation2, _ := ValidateImage(result2.Data, 0, 0, "")

	t.Logf("Image 1: %dx%d, %d bytes", validation1.Width, validation1.Height, validation1.FileSizeBytes)
	t.Logf("Image 2: %dx%d, %d bytes", validation2.Width, validation2.Height, validation2.FileSizeBytes)

	// Dimensions should match
	if validation1.Width != validation2.Width || validation1.Height != validation2.Height {
		t.Errorf("Dimensions don't match with same seed: %dx%d vs %dx%d",
			validation1.Width, validation1.Height, validation2.Width, validation2.Height)
	}

	// Note: We don't check exact pixel-by-pixel match as that's not guaranteed by most APIs
}

// TestBedrockNovaCanvasDimensions tests Bedrock dimension constraints
func TestBedrockNovaCanvasDimensions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check for Bedrock credentials
	if !config.HasBedrockCredentials() {
		t.Skip("AWS Bedrock credentials not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	client, err := generate.NewBedrockSDKClient(ctx, "")
	if err != nil {
		t.Fatalf("Failed to create Bedrock client: %v", err)
	}
	defer client.Close()

	testCases := []struct {
		name           string
		size           string
		expectedWidth  int
		expectedHeight int
	}{
		{"512x512", "512x512", 512, 512},
		{"1024x1024", "1024x1024", 1024, 1024},
		{"1024x768", "1024x768", 1024, 768},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := models.GenerateOptions{
				Model:    "amazon.nova-canvas-v1:0",
				Size:     tc.size,
				CfgScale: 7.0,
			}

			result, err := client.GenerateImage(ctx, "simple geometric shapes", opts)
			if err != nil {
				t.Fatalf("Generation failed: %v", err)
			}

			validation, err := ValidateImage(result.Data, tc.expectedWidth, tc.expectedHeight, "png")
			if err != nil {
				t.Fatalf("Validation failed: %v", err)
			}

			t.Logf("Generated image: %dx%d (expected %dx%d)",
				validation.Width, validation.Height, tc.expectedWidth, tc.expectedHeight)

			if !validation.IsValid {
				for _, e := range validation.Errors {
					t.Errorf("Validation error: %s", e)
				}
			}
		})
	}
}

// TestBedrockCfgScale tests that CFG scale affects generation
func TestBedrockCfgScale(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if !config.HasBedrockCredentials() {
		t.Skip("AWS Bedrock credentials not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	client, err := generate.NewBedrockSDKClient(ctx, "")
	if err != nil {
		t.Fatalf("Failed to create Bedrock client: %v", err)
	}
	defer client.Close()

	// Test that different CFG scales don't cause errors
	cfgScales := []float64{1.0, 5.0, 10.0}

	for _, cfg := range cfgScales {
		t.Run(fmt.Sprintf("CFG_%.1f", cfg), func(t *testing.T) {
			opts := models.GenerateOptions{
				Model:    "amazon.nova-canvas-v1:0",
				Size:     "512x512",
				CfgScale: cfg,
			}

			result, err := client.GenerateImage(ctx, "abstract art", opts)
			if err != nil {
				t.Fatalf("Generation with CFG %.1f failed: %v", cfg, err)
			}

			validation, err := ValidateImage(result.Data, 512, 512, "")
			if err != nil {
				t.Fatalf("Validation failed: %v", err)
			}

			t.Logf("CFG %.1f: Generated %dx%d image, %d bytes",
				cfg, validation.Width, validation.Height, validation.FileSizeBytes)
		})
	}
}

// TestNegativePrompt tests that negative prompts are processed without error
func TestNegativePrompt(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiKey, err := config.GetGeminiAPIKey("")
	if err != nil {
		t.Skipf("Gemini API key not configured: %v", err)
	}

	client, err := generate.NewGeminiRESTClient(apiKey)
	if err != nil {
		t.Fatalf("Failed to create Gemini client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	opts := models.GenerateOptions{
		Model:          "gemini-2.5-flash-image",
		Size:           "512x512",
		AspectRatio:    "1:1",
		NegativePrompt: "text, watermarks, logos, blurry, low quality",
	}

	result, err := client.GenerateImage(ctx, "a serene mountain landscape at dawn", opts)
	if err != nil {
		t.Fatalf("Generation with negative prompt failed: %v", err)
	}

	validation, err := ValidateImage(result.Data, 0, 0, "")
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	t.Logf("Generated image with negative prompt: %dx%d, %d bytes",
		validation.Width, validation.Height, validation.FileSizeBytes)

	if !validation.IsValid {
		for _, e := range validation.Errors {
			t.Errorf("Validation error: %s", e)
		}
	}
}

// TestSaveAndLoadImage tests the full save/load cycle
func TestSaveAndLoadImage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiKey, err := config.GetGeminiAPIKey("")
	if err != nil {
		t.Skipf("Gemini API key not configured: %v", err)
	}

	client, err := generate.NewGeminiRESTClient(apiKey)
	if err != nil {
		t.Fatalf("Failed to create Gemini client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	opts := models.GenerateOptions{
		Model:       "gemini-2.5-flash-image",
		Size:        "512x512",
		AspectRatio: "1:1",
	}

	result, err := client.GenerateImage(ctx, "a simple yellow star", opts)
	if err != nil {
		t.Fatalf("Generation failed: %v", err)
	}

	// Create temp file
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_output.png")

	// Save image
	err = generate.SaveImage(result, outputPath)
	if err != nil {
		t.Fatalf("Failed to save image: %v", err)
	}

	// Verify file exists
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Output file not found: %v", err)
	}

	t.Logf("Saved image to %s (%d bytes)", outputPath, info.Size())

	// Read back and validate
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read saved image: %v", err)
	}

	validation, err := ValidateImage(data, 0, 0, "png")
	if err != nil {
		t.Fatalf("Failed to validate saved image: %v", err)
	}

	// File size should match
	if int64(len(data)) != info.Size() {
		t.Errorf("File size mismatch: read %d bytes, stat says %d", len(data), info.Size())
	}

	t.Logf("Validated saved image: %dx%d, format=%s",
		validation.Width, validation.Height, validation.Format)
}

// BenchmarkGeneration benchmarks image generation time
func BenchmarkGeneration(b *testing.B) {
	apiKey, err := config.GetGeminiAPIKey("")
	if err != nil {
		b.Skipf("Gemini API key not configured: %v", err)
	}

	client, err := generate.NewGeminiRESTClient(apiKey)
	if err != nil {
		b.Fatalf("Failed to create Gemini client: %v", err)
	}
	defer client.Close()

	opts := models.GenerateOptions{
		Model:       "gemini-2.5-flash-image",
		Size:        "512x512",
		AspectRatio: "1:1",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.GenerateImage(ctx, "simple test image", opts)
		if err != nil {
			b.Fatalf("Generation failed: %v", err)
		}
	}
}
