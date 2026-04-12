package gimage_test

import (
	"context"
	"fmt"
	"log"
	"net/http"

	gimage "github.com/apresai/gimage/sdk/go"
)

// Example demonstrates how to use the Gimage Go SDK
func Example() {
	// Create a client with your API Gateway endpoint
	baseURL := "https://cf3xrk9w63.execute-api.us-east-1.amazonaws.com/production"

	client, err := gimage.NewClient(baseURL)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Create context
	ctx := context.Background()

	// Generate an image
	resp, err := client.GenerateImage(ctx, gimage.GenerateImageJSONRequestBody{
		Prompt: "sunset over mountains",
		Model:  modelPtr(gimage.Gemini25FlashImage),
		Size:   stringPtr("1024x1024"),
	})
	if err != nil {
		log.Fatalf("Failed to generate image: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode)
}

// Example_withAPIKey demonstrates authentication with API Gateway API key
func Example_withAPIKey() {
	baseURL := "https://cf3xrk9w63.execute-api.us-east-1.amazonaws.com/production"
	apiKey := "your-api-key-here"

	// Create client with API key authentication
	client, err := gimage.NewClient(baseURL, gimage.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("x-api-key", apiKey)
		return nil
	}))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Generate image with custom options
	resp, err := client.GenerateImage(ctx, gimage.GenerateImageJSONRequestBody{
		Prompt:         "futuristic city with flying cars",
		Model:          modelPtr(gimage.Gemini25FlashImage),
		Size:           stringPtr("1024x1024"),
		Style:          stylePtr(gimage.Photorealistic),
		NegativePrompt: stringPtr("people, text"),
		Seed:           int64Ptr(42),
		ResponseFormat: respFmtPtr(gimage.GenerateRequestResponseFormatBase64),
	})
	if err != nil {
		log.Fatalf("Failed to generate image: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode)
}

// Example_healthCheck demonstrates checking API health
func Example_healthCheck() {
	baseURL := "https://cf3xrk9w63.execute-api.us-east-1.amazonaws.com/production"
	apiKey := "your-api-key-here"

	client, err := gimage.NewClient(baseURL, gimage.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("x-api-key", apiKey)
		return nil
	}))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Check health
	resp, err := client.HealthCheck(ctx)
	if err != nil {
		log.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Health Status: %d\n", resp.StatusCode)
}

// Helper functions
func stringPtr(s string) *string { return &s }
func int64Ptr(i int64) *int64    { return &i }

func modelPtr(m gimage.GenerateRequestModel) *gimage.GenerateRequestModel {
	return &m
}

func stylePtr(s gimage.GenerateRequestStyle) *gimage.GenerateRequestStyle {
	return &s
}

func respFmtPtr(r gimage.GenerateRequestResponseFormat) *gimage.GenerateRequestResponseFormat {
	return &r
}
