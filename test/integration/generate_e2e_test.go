// +build e2e

package integration

import (
	"context"
	"os"
	"testing"

	"github.com/apresai/gimage/internal/config"
	"github.com/apresai/gimage/internal/generate"
	"github.com/apresai/gimage/pkg/models"
)

// E2E tests for real API calls
// These tests cost money and require real credentials
// Run with: go test -tags=e2e ./test/integration/...

func TestGeminiAPIE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
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
		Size:  "512x512", // Smaller size for faster/cheaper testing
	}

	t.Log("🎨 Generating test image with Gemini API...")
	t.Log("⚠️  This will consume API quota/credits")

	result, err := client.GenerateImage(ctx, "a simple red circle on white background", options)
	if err != nil {
		t.Fatalf("Gemini image generation failed: %v", err)
	}

	if len(result.Data) == 0 {
		t.Fatal("Gemini returned empty image data")
	}

	t.Logf("✅ Gemini E2E test passed - generated %d bytes", len(result.Data))
	t.Logf("   Format: %s", result.Format)
	t.Logf("   Size: %dx%d", result.Width, result.Height)
}

func TestVertexAIE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Check for Vertex credentials
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
		Model: "imagen-4.0-fast-generate-001", // Use fast/cheap model
		Size:  "512x512",
	}

	t.Log("🎨 Generating test image with Vertex AI...")
	t.Log("⚠️  This will cost approximately $0.02")

	result, err := client.GenerateImage(ctx, "a simple blue square on white background", options)
	if err != nil {
		t.Fatalf("Vertex image generation failed: %v", err)
	}

	if len(result.Data) == 0 {
		t.Fatal("Vertex returned empty image data")
	}

	t.Logf("✅ Vertex E2E test passed - generated %d bytes", len(result.Data))
	t.Logf("   Format: %s", result.Format)
	t.Logf("   Size: %dx%d", result.Width, result.Height)
}

func TestBedrockNovaCanvasE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Check for Bedrock credentials
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
		Style: "standard", // Use standard quality
	}

	t.Log("🎨 Generating test image with AWS Bedrock Nova Canvas...")
	t.Log("⚠️  This will cost $0.04 (standard quality)")

	result, err := client.GenerateImage(ctx, "a simple green triangle on white background", options)
	if err != nil {
		t.Fatalf("Bedrock Nova Canvas image generation failed: %v", err)
	}

	if len(result.Data) == 0 {
		t.Fatal("Bedrock returned empty image data")
	}

	t.Logf("✅ Bedrock Nova Canvas E2E test passed - generated %d bytes", len(result.Data))
	t.Logf("   Format: %s", result.Format)
	t.Logf("   Size: %dx%d", result.Width, result.Height)
}

func TestGrokE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Check for Grok credentials
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
		Model: "grok-2-image",
		Size:  "512x512",
	}

	t.Log("🎨 Generating test image with xAI Grok...")
	t.Log("⚠️  This will cost approximately $0.07")

	result, err := client.GenerateImage(ctx, "a simple purple star on white background", options)
	if err != nil {
		t.Fatalf("Grok image generation failed: %v", err)
	}

	if len(result.Data) == 0 {
		t.Fatal("Grok returned empty image data")
	}

	t.Logf("✅ Grok E2E test passed - generated %d bytes", len(result.Data))
	t.Logf("   Format: %s", result.Format)
	t.Logf("   Size: %dx%d", result.Width, result.Height)
}

func TestVertexAIUnifiedSDKE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Check for Vertex ADC credentials (service account or gcloud auth)
	project := os.Getenv("VERTEX_PROJECT")
	location := os.Getenv("VERTEX_LOCATION")
	if location == "" {
		location = "us-central1"
	}

	// Skip if API key is set (use REST test instead) or no project
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
		Model: "imagen-4.0-fast-generate-001", // Use fast/cheap model
		Size:  "512x512",
	}

	t.Log("🎨 Generating test image with Vertex AI Unified SDK (ADC)...")
	t.Log("⚠️  This will cost approximately $0.02")

	result, err := client.GenerateImage(ctx, "a simple orange diamond on white background", options)
	if err != nil {
		t.Fatalf("Vertex unified SDK image generation failed: %v", err)
	}

	if len(result.Data) == 0 {
		t.Fatal("Vertex unified SDK returned empty image data")
	}

	t.Logf("✅ Vertex AI Unified SDK E2E test passed - generated %d bytes", len(result.Data))
	t.Logf("   Format: %s", result.Format)
	t.Logf("   Size: %dx%d", result.Width, result.Height)
}

// TestAllAPIsE2E runs all API tests in sequence if credentials are available
func TestAllAPIsE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	t.Run("Gemini", TestGeminiAPIE2E)
	t.Run("VertexREST", TestVertexAIE2E)
	t.Run("VertexUnifiedSDK", TestVertexAIUnifiedSDKE2E)
	t.Run("Bedrock", TestBedrockNovaCanvasE2E)
	t.Run("Grok", TestGrokE2E)
}
