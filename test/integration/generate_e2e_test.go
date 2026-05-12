//go:build e2e
// +build e2e

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apresai/gimage/internal/config"
	"github.com/apresai/gimage/internal/generate"
	"github.com/apresai/gimage/pkg/models"
)

// E2E tests for real API calls
// These tests cost money and require real credentials
// Run with: go test -tags=e2e ./test/integration/...
// Filter providers: GIMAGE_TEST_PROVIDERS=gemini,grok go test -tags=e2e ./test/integration/...

// e2ePrompt is the shared prompt used across all provider tests for visual comparison.
const e2ePrompt = "A young woman on a Southern California beach in 1970s vintage beach attire, retro color palette with warm golden tones, film grain texture, classic 70s aesthetic"

// shouldTestProvider checks if a provider should be tested based on the
// GIMAGE_TEST_PROVIDERS environment variable. If unset, all providers are tested.
func shouldTestProvider(provider string) bool {
	filter := os.Getenv("GIMAGE_TEST_PROVIDERS")
	if filter == "" {
		return true // test all by default
	}
	for _, p := range strings.Split(filter, ",") {
		if strings.TrimSpace(p) == provider {
			return true
		}
	}
	return false
}

// e2eOutputDir returns the output directory for E2E test images
func e2eOutputDir() string {
	if dir := os.Getenv("GIMAGE_E2E_OUTPUT_DIR"); dir != "" {
		return dir
	}
	absDir, err := filepath.Abs(filepath.Join("..", "..", "test", "output", "e2e"))
	if err != nil {
		return filepath.Join("test", "output", "e2e")
	}
	return absDir
}

// saveAndLogImage saves generated image data to disk and logs the full path
func saveAndLogImage(t *testing.T, img *models.GeneratedImage, provider string) {
	t.Helper()

	outDir := e2eOutputDir()
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Logf("Warning: could not create output dir %s: %v", outDir, err)
		return
	}

	ext := img.Format
	if ext == "" {
		ext = "png"
	}
	filename := fmt.Sprintf("e2e_%s.%s", provider, ext)
	fullPath := filepath.Join(outDir, filename)

	if err := os.WriteFile(fullPath, img.Data, 0644); err != nil {
		t.Logf("Warning: could not save image to %s: %v", fullPath, err)
		return
	}

	t.Logf("GENERATED_IMAGE: %s (%d bytes, %s, %dx%d)", fullPath, len(img.Data), img.Format, img.Width, img.Height)
}

func TestGeminiFlashE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}
	if !shouldTestProvider("gemini") {
		t.Skip("Skipping: gemini not in GIMAGE_TEST_PROVIDERS")
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		cfg, err := config.LoadConfig()
		if err != nil || cfg.GeminiAPIKey == "" {
			t.Skip("GEMINI_API_KEY not set, skipping Gemini E2E test")
		}
		apiKey = cfg.GeminiAPIKey
	}

	client, err := generate.NewGeminiRESTClient(apiKey)
	if err != nil {
		t.Fatalf("Failed to create Gemini client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	options := models.GenerateOptions{
		Model: "gemini-2.5-flash-image",
		Size:  "1024x1024",
	}

	t.Log("Generating test image with Gemini 2.5 Flash...")
	t.Log("This will cost approximately $0.04")

	results, err := client.GenerateImage(ctx, e2ePrompt, options)
	if err != nil {
		t.Fatalf("Gemini Flash image generation failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Gemini Flash returned no images")
	}

	for i, img := range results {
		t.Logf("Gemini Flash E2E image %d: %d bytes, format=%s, size=%dx%d", i+1, len(img.Data), img.Format, img.Width, img.Height)
		saveAndLogImage(t, img, fmt.Sprintf("gemini_flash_%d", i+1))
	}
}

func TestGemini3ProE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}
	if !shouldTestProvider("gemini") {
		t.Skip("Skipping: gemini not in GIMAGE_TEST_PROVIDERS")
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		cfg, err := config.LoadConfig()
		if err != nil || cfg.GeminiAPIKey == "" {
			t.Skip("GEMINI_API_KEY not set, skipping Gemini 3 Pro E2E test")
		}
		apiKey = cfg.GeminiAPIKey
	}

	client, err := generate.NewGeminiRESTClient(apiKey)
	if err != nil {
		t.Fatalf("Failed to create Gemini client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	options := models.GenerateOptions{
		Model: "gemini-3-pro-image-preview",
	}

	t.Log("Generating test image with Gemini 3 Pro...")
	t.Log("This will cost approximately $0.13")

	results, err := client.GenerateImage(ctx, e2ePrompt, options)
	if err != nil {
		t.Fatalf("Gemini 3 Pro image generation failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Gemini 3 Pro returned no images")
	}

	for i, img := range results {
		t.Logf("Gemini 3 Pro E2E image %d: %d bytes, format=%s, size=%dx%d", i+1, len(img.Data), img.Format, img.Width, img.Height)
		saveAndLogImage(t, img, fmt.Sprintf("gemini_3_pro_%d", i+1))
	}
}

func TestVertexAIE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}
	if !shouldTestProvider("vertex") {
		t.Skip("Skipping: vertex not in GIMAGE_TEST_PROVIDERS")
	}

	apiKey := os.Getenv("VERTEX_API_KEY")
	project := os.Getenv("VERTEX_PROJECT")

	if apiKey == "" || project == "" {
		cfg, err := config.LoadConfig()
		if err != nil || cfg.VertexAPIKey == "" || cfg.VertexProject == "" {
			t.Skip("Vertex AI credentials not set, skipping Vertex E2E test")
		}
		apiKey = cfg.VertexAPIKey
		project = cfg.VertexProject
	}

	client, err := generate.NewVertexRESTClient(apiKey, project, "us-central1")
	if err != nil {
		t.Fatalf("Failed to create Vertex client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	options := models.GenerateOptions{
		Model: "imagen-4.0-fast-generate-001",
		Size:  "512x512",
	}

	t.Log("Generating test image with Vertex AI (Imagen 4 Fast)...")
	t.Log("This will cost approximately $0.02")

	results, err := client.GenerateImage(ctx, e2ePrompt, options)
	if err != nil {
		t.Fatalf("Vertex image generation failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Vertex returned no images")
	}

	for i, img := range results {
		t.Logf("Vertex E2E image %d: %d bytes, format=%s, size=%dx%d", i+1, len(img.Data), img.Format, img.Width, img.Height)
		saveAndLogImage(t, img, fmt.Sprintf("vertex_%d", i+1))
	}
}

func TestBedrockNovaCanvasE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}
	if !shouldTestProvider("bedrock") {
		t.Skip("Skipping: bedrock not in GIMAGE_TEST_PROVIDERS")
	}

	apiKey := os.Getenv("AWS_BEARER_TOKEN_BEDROCK")
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	if apiKey == "" {
		cfg, err := config.LoadConfig()
		if err != nil || cfg.AWSBedrockAPIKey == "" {
			t.Skip("AWS Bedrock credentials not set, skipping Bedrock E2E test")
		}
		apiKey = cfg.AWSBedrockAPIKey
		if cfg.AWSRegion != "" {
			region = cfg.AWSRegion
		}
	}

	client, err := generate.NewBedrockRESTClient(apiKey, region)
	if err != nil {
		t.Fatalf("Failed to create Bedrock client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	options := models.GenerateOptions{
		Model: "amazon.nova-canvas-v1:0",
		Size:  "512x512",
		Style: "standard",
	}

	t.Log("Generating test image with AWS Bedrock Nova Canvas...")
	t.Log("This will cost $0.04 (standard quality)")

	results, err := client.GenerateImage(ctx, e2ePrompt, options)
	if err != nil {
		t.Fatalf("Bedrock Nova Canvas image generation failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Bedrock returned no images")
	}

	for i, img := range results {
		t.Logf("Bedrock E2E image %d: %d bytes, format=%s, size=%dx%d", i+1, len(img.Data), img.Format, img.Width, img.Height)
		saveAndLogImage(t, img, fmt.Sprintf("bedrock_%d", i+1))
	}
}

func TestGrokImagineE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}
	if !shouldTestProvider("grok") {
		t.Skip("Skipping: grok not in GIMAGE_TEST_PROVIDERS")
	}

	apiKey := os.Getenv("GROK_API_KEY")
	if apiKey == "" {
		cfg, err := config.LoadConfig()
		if err != nil || cfg.GrokAPIKey == "" {
			t.Skip("GROK_API_KEY not set, skipping Grok E2E test")
		}
		apiKey = cfg.GrokAPIKey
	}

	client, err := generate.NewGrokClient(apiKey)
	if err != nil {
		t.Fatalf("Failed to create Grok client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	options := models.GenerateOptions{
		Model: "grok-imagine-image",
	}

	t.Log("Generating test image with xAI Grok Imagine...")
	t.Log("This will cost approximately $0.02")

	results, err := client.GenerateImage(ctx, e2ePrompt, options)
	if err != nil {
		t.Fatalf("Grok Imagine image generation failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Grok Imagine returned no images")
	}

	for i, img := range results {
		t.Logf("Grok Imagine E2E image %d: %d bytes, format=%s, size=%dx%d", i+1, len(img.Data), img.Format, img.Width, img.Height)
		saveAndLogImage(t, img, fmt.Sprintf("grok_imagine_%d", i+1))
	}
}

func TestGrokImagineQualityE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}
	if !shouldTestProvider("grok") {
		t.Skip("Skipping: grok not in GIMAGE_TEST_PROVIDERS")
	}

	apiKey := os.Getenv("GROK_API_KEY")
	if apiKey == "" {
		cfg, err := config.LoadConfig()
		if err != nil || cfg.GrokAPIKey == "" {
			t.Skip("GROK_API_KEY not set, skipping Grok Imagine Quality E2E test")
		}
		apiKey = cfg.GrokAPIKey
	}

	client, err := generate.NewGrokClient(apiKey)
	if err != nil {
		t.Fatalf("Failed to create Grok client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	options := models.GenerateOptions{
		Model: "grok-imagine-image-quality",
	}

	t.Log("Generating test image with xAI Grok Imagine Quality...")
	t.Log("This will cost approximately $0.05")

	results, err := client.GenerateImage(ctx, e2ePrompt, options)
	if err != nil {
		t.Fatalf("Grok Imagine Quality image generation failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Grok Imagine Quality returned no images")
	}

	for i, img := range results {
		t.Logf("Grok Imagine Quality E2E image %d: %d bytes, format=%s, size=%dx%d", i+1, len(img.Data), img.Format, img.Width, img.Height)
		saveAndLogImage(t, img, fmt.Sprintf("grok_imagine_quality_%d", i+1))
	}
}

func TestVertexAIUnifiedSDKE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}
	if !shouldTestProvider("vertex") {
		t.Skip("Skipping: vertex not in GIMAGE_TEST_PROVIDERS")
	}

	project := os.Getenv("VERTEX_PROJECT")
	location := os.Getenv("VERTEX_LOCATION")
	if location == "" {
		location = "us-central1"
	}

	if os.Getenv("VERTEX_API_KEY") != "" {
		t.Skip("VERTEX_API_KEY is set, use TestVertexAIE2E for REST mode")
	}

	if project == "" {
		cfg, err := config.LoadConfig()
		if err != nil || cfg.VertexProject == "" {
			t.Skip("VERTEX_PROJECT not set, skipping Vertex unified SDK E2E test")
		}
		project = cfg.VertexProject
		if cfg.VertexLocation != "" {
			location = cfg.VertexLocation
		}
	}

	ctx := context.Background()
	client, err := generate.NewVertexUnifiedClient(ctx, project, location)
	if err != nil {
		t.Skipf("Failed to create Vertex unified client (ADC may not be configured): %v", err)
	}
	defer client.Close()

	options := models.GenerateOptions{
		Model: "imagen-4.0-fast-generate-001",
		Size:  "512x512",
	}

	t.Log("Generating test image with Vertex AI Unified SDK (ADC)...")
	t.Log("This will cost approximately $0.02")

	results, err := client.GenerateImage(ctx, e2ePrompt, options)
	if err != nil {
		t.Fatalf("Vertex unified SDK image generation failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Vertex unified SDK returned no images")
	}

	for i, img := range results {
		t.Logf("Vertex SDK E2E image %d: %d bytes, format=%s, size=%dx%d", i+1, len(img.Data), img.Format, img.Width, img.Height)
		saveAndLogImage(t, img, fmt.Sprintf("vertex_sdk_%d", i+1))
	}
}

// TestAllAPIsE2E runs all API tests in sequence if credentials are available
func TestAllAPIsE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	t.Run("GeminiFlash", TestGeminiFlashE2E)
	t.Run("Gemini3Pro", TestGemini3ProE2E)
	t.Run("VertexREST", TestVertexAIE2E)
	t.Run("VertexUnifiedSDK", TestVertexAIUnifiedSDKE2E)
	t.Run("Bedrock", TestBedrockNovaCanvasE2E)
	t.Run("GrokImagine", TestGrokImagineE2E)
	t.Run("GrokImagineQuality", TestGrokImagineQualityE2E)
}
