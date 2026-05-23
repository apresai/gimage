# Real-World Example

This example shows how to use the Gimage Go SDK with your deployed Lambda API.

## Setup

```bash
go get github.com/apresai/gimage/sdk/go
```

## Complete Example

```go
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	gimage "github.com/apresai/gimage/sdk/go"
)

func main() {
	// Your deployed Lambda endpoint
	baseURL := "https://cf3xrk9w63.execute-api.us-east-1.amazonaws.com/production"
	apiKey := os.Getenv("GIMAGE_API_KEY") // or hardcode for testing

	if apiKey == "" {
		log.Fatal("Please set GIMAGE_API_KEY environment variable")
	}

	// Create authenticated client
	client, err := gimage.NewClient(
		baseURL,
		gimage.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			req.Header.Set("x-api-key", apiKey)
			return nil
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Example 1: Health check
	fmt.Println("1. Checking API health...")
	healthResp, err := client.HealthCheck(ctx)
	if err != nil {
		log.Fatalf("Health check failed: %v", err)
	}
	defer healthResp.Body.Close()

	if healthResp.StatusCode == 200 {
		body, _ := io.ReadAll(healthResp.Body)
		fmt.Printf("✓ API is healthy: %s\n\n", body)
	}

	// Example 2: Generate image with Gemini
	fmt.Println("2. Generating image with Gemini...")
	genResp, err := client.GenerateImage(ctx, gimage.GenerateImageJSONRequestBody{
		Prompt:         "a sunset over mountains with vibrant colors",
		Model:          stringPtr("gemini-2.5-flash-image"),
		Size:           stringPtr("1024x1024"),
		ResponseFormat: (*gimage.ResponseFormat)(stringPtr("base64")),
	})
	if err != nil {
		log.Fatalf("Image generation failed: %v", err)
	}
	defer genResp.Body.Close()

	if genResp.StatusCode == 200 {
		var result struct {
			Image      string `json:"image"`
			Width      int    `json:"width"`
			Height     int    `json:"height"`
			Format     string `json:"format"`
			SizeBytes  int    `json:"size_bytes"`
		}

		body, _ := io.ReadAll(genResp.Body)
		if err := json.Unmarshal(body, &result); err != nil {
			log.Fatalf("Failed to parse response: %v", err)
		}

		fmt.Printf("✓ Image generated!\n")
		fmt.Printf("  Size: %dx%d\n", result.Width, result.Height)
		fmt.Printf("  Format: %s\n", result.Format)
		fmt.Printf("  Size: %d bytes\n", result.SizeBytes)

		// Save to file
		imageData, err := base64.StdEncoding.DecodeString(result.Image)
		if err != nil {
			log.Fatalf("Failed to decode image: %v", err)
		}

		filename := "generated-sunset.png"
		if err := os.WriteFile(filename, imageData, 0644); err != nil {
			log.Fatalf("Failed to save image: %v", err)
		}

		fmt.Printf("  Saved to: %s\n\n", filename)
	}

	// Example 3: Generate with S3 URL response (for large images)
	fmt.Println("3. Generating large image with S3 URL...")
	s3Resp, err := client.GenerateImage(ctx, gimage.GenerateImageJSONRequestBody{
		Prompt:         "futuristic city with flying cars, ultra detailed",
		Model:          stringPtr("gemini-2.5-flash-image"),
		Size:           stringPtr("1024x1024"),
		Style:          (*gimage.ImageStyle)(stringPtr("photorealistic")),
		ResponseFormat: (*gimage.ResponseFormat)(stringPtr("s3_url")),
	})
	if err != nil {
		log.Fatalf("Image generation failed: %v", err)
	}
	defer s3Resp.Body.Close()

	if s3Resp.StatusCode == 200 {
		var result struct {
			S3URL        string `json:"s3_url"`
			S3Key        string `json:"s3_key"`
			Width        int    `json:"width"`
			Height       int    `json:"height"`
			Format       string `json:"format"`
			ExpiresIn    int    `json:"expires_in"`
		}

		body, _ := io.ReadAll(s3Resp.Body)
		if err := json.Unmarshal(body, &result); err != nil {
			log.Fatalf("Failed to parse response: %v", err)
		}

		fmt.Printf("✓ Image uploaded to S3!\n")
		fmt.Printf("  URL: %s\n", result.S3URL)
		fmt.Printf("  Expires in: %d seconds\n", result.ExpiresIn)
		fmt.Printf("  Size: %dx%d\n\n", result.Width, result.Height)
	}

	// Example 4: Gemini 3+ compositional editing with reference images + thinking
	fmt.Println("4. Compositional editing with Gemini 3.1 Flash...")
	thinkingLevel := gimage.GenerateRequestThinkingLevelLow
	composeResp, err := client.GenerateImage(ctx, gimage.GenerateImageJSONRequestBody{
		Prompt:        "place this character on a beach at golden hour, cinematic lighting",
		Model:         stringPtr("gemini-3.1-flash-image-preview"),
		ImageSize:     stringPtr("2K"),
		AspectRatio:   stringPtr("16:9"),
		// InputImages on the wire are local paths the server reads; for the
		// HTTP API surface this field is expected as a string slice — see
		// the OpenAPI spec for the deployment-specific convention.
		InputImages:   &[]string{"/path/to/character.png"},
		ThinkingLevel: &thinkingLevel,
		Grounding:     boolPtr(false), // enable for current/real-world references
	})
	if err != nil {
		log.Fatalf("Compositional edit failed: %v", err)
	}
	defer composeResp.Body.Close()
	if composeResp.StatusCode == 200 {
		fmt.Println("✓ Compositional edit complete (Gemini 3+ exclusive — silently ignored on other providers)")
	}

	fmt.Println("✓ All examples completed successfully!")
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
```

## Running the Example

```bash
# Set your API key
export GIMAGE_API_KEY="your-api-key-here"

# Run the example
go run example.go
```

## Expected Output

```
1. Checking API health...
✓ API is healthy: {"status":"healthy","version":"0.1.1","apis":{"gemini":"not_configured","vertex":"not_configured"}}

2. Generating image with Gemini...
✓ Image generated!
  Size: 1024x1024
  Format: png
  Size: 524288 bytes
  Saved to: generated-sunset.png

3. Generating large image with S3 URL...
✓ Image uploaded to S3!
  URL: https://gimage-storage-production.s3.amazonaws.com/images/1234567890-abc.png?X-Amz-...
  Expires in: 3600 seconds
  Size: 1024x1024

✓ All examples completed successfully!
```

## Environment Variables

You can configure API keys using environment variables instead of hardcoding:

```bash
export GIMAGE_API_KEY="your-api-key"
export GEMINI_API_KEY="your-gemini-key"  # For AI generation
export VERTEX_API_KEY="your-vertex-key"  # For Vertex AI
```

## Error Handling

```go
resp, err := client.GenerateImage(ctx, request)
if err != nil {
	log.Fatalf("Request failed: %v", err)
}
defer resp.Body.Close()

switch resp.StatusCode {
case 200:
	// Success
	var result ImageResponse
	json.NewDecoder(resp.Body).Decode(&result)
case 400:
	// Bad request
	body, _ := io.ReadAll(resp.Body)
	log.Printf("Invalid request: %s", body)
case 401:
	// Unauthorized
	log.Fatal("Invalid API key")
case 403:
	// Forbidden
	log.Fatal("Access denied - check API key permissions")
case 500:
	// Server error
	body, _ := io.ReadAll(resp.Body)
	log.Printf("Server error: %s", body)
default:
	body, _ := io.ReadAll(resp.Body)
	log.Printf("Unexpected status %d: %s", resp.StatusCode, body)
}
```

## Production Tips

1. **Always use environment variables** for API keys
2. **Set reasonable timeouts** on the HTTP client (30-60 seconds)
3. **Handle rate limits** - implement retry with exponential backoff
4. **Log request IDs** for debugging
5. **Use S3 URLs** for large images to reduce bandwidth
6. **Cache responses** when appropriate
7. **Monitor API health** periodically
