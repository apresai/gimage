package generate

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/apresai/gimage/internal/observability"
	"github.com/apresai/gimage/pkg/models"
	"github.com/sony/gobreaker"
)

// BedrockRESTClient uses AWS Bedrock REST API with bearer token for image generation
type BedrockRESTClient struct {
	apiKey         string
	region         string
	baseURL        string
	httpClient     *http.Client
	log            *observability.VerboseLogger
	circuitBreaker *gobreaker.CircuitBreaker
}

// NewBedrockRESTClient creates a new AWS Bedrock REST client with bearer token
func NewBedrockRESTClient(apiKey, region string) (*BedrockRESTClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("AWS Bedrock API key is required")
	}

	// Default region if not provided
	if region == "" {
		region = os.Getenv("AWS_REGION")
		if region == "" {
			region = "us-east-1"
		}
	}

	// Build base URL for Bedrock Runtime
	baseURL := fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com", region)

	return &BedrockRESTClient{
		apiKey:  apiKey,
		region:  region,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
		log:            observability.NewVerboseLogger(observability.ComponentBedrock),
		circuitBreaker: newCircuitBreaker("BedrockRESTAPI"),
	}, nil
}

// GenerateImage generates an image using AWS Bedrock Nova Canvas via REST API
func (c *BedrockRESTClient) GenerateImage(ctx context.Context, prompt string, options models.GenerateOptions) ([]*models.GeneratedImage, error) {
	startTime := time.Now()

	// Build request payload
	request, err := c.buildRequest(prompt, options)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	// Marshal request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	c.log.Debug("Generating image with AWS Bedrock Nova Canvas via REST API")
	c.log.Debug("Prompt: %s, Model: %s, Size: %s, Quality: %s",
		prompt, options.Model, options.Size, request.ImageGenerationConfig.Quality)

	// Determine model ID
	modelID := "amazon.nova-canvas-v1:0"
	if options.Model != "" {
		modelID = options.Model
	}

	// Build API endpoint
	endpoint := fmt.Sprintf("%s/model/%s/invoke", c.baseURL, modelID)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers - Bearer token authentication
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// Execute request via circuit breaker
	var response []*models.GeneratedImage
	_, err = c.circuitBreaker.Execute(func() (interface{}, error) {
		// Make HTTP request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("HTTP request failed: %w", err)
		}
		defer resp.Body.Close()

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		// Check for HTTP errors
		if resp.StatusCode != http.StatusOK {
			return nil, c.handleHTTPError(resp.StatusCode, body)
		}

		// Parse response
		var novaResponse NovaCanvasResponse
		if err := json.Unmarshal(body, &novaResponse); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		// Check for API error
		if novaResponse.Error != "" {
			return nil, fmt.Errorf("API error: %s", novaResponse.Error)
		}

		// Check for images
		if len(novaResponse.Images) == 0 {
			return nil, fmt.Errorf("no images generated")
		}

		// Parse dimensions
		requestedWidth, requestedHeight := ParseSizeString(options.Size)

		// Build response
		var generatedImages []*models.GeneratedImage
		for i, imgStr := range novaResponse.Images {
			imageData, err := base64.StdEncoding.DecodeString(imgStr)
			if err != nil {
				c.log.Debug("Failed to decode image %d, skipping: %v", i, err)
				continue
			}

			generatedImages = append(generatedImages, &models.GeneratedImage{
				Data:   imageData,
				Format: "png",
				Width:  requestedWidth,
				Height: requestedHeight,
				Metadata: map[string]string{
					"model":       modelID,
					"prompt":      prompt,
					"size":        options.Size,
					"seed":        fmt.Sprintf("%d", request.ImageGenerationConfig.Seed),
					"quality":     request.ImageGenerationConfig.Quality,
					"candidate":   fmt.Sprintf("%d", i),
					"resize_mode": options.ResizeMode,
				},
			})
		}

		if len(generatedImages) == 0 {
			return nil, fmt.Errorf("no valid images generated")
		}

		c.log.Debug("Images generated successfully via REST API: model=%s, count=%d, duration=%s",
			modelID, len(generatedImages), time.Since(startTime))

		response = generatedImages
		return response, nil
	})

	if err != nil {
		return nil, err
	}

	return response, nil
}

// buildRequest builds the Nova Canvas request from options (delegates to shared function)
func (c *BedrockRESTClient) buildRequest(prompt string, options models.GenerateOptions) (*NovaCanvasRequest, error) {
	return BuildNovaCanvasRequest(prompt, options, c.log)
}

// handleHTTPError provides user-friendly error messages for HTTP errors
func (c *BedrockRESTClient) handleHTTPError(statusCode int, body []byte) error {
	// Try to parse error message from response
	var errorMsg string
	var errorResponse map[string]interface{}
	if err := json.Unmarshal(body, &errorResponse); err == nil {
		if msg, ok := errorResponse["message"].(string); ok {
			errorMsg = msg
		}
	}

	if errorMsg == "" {
		errorMsg = string(body)
	}

	c.log.Error("AWS Bedrock REST API error: status=%d, message=%s", statusCode, errorMsg)

	// Provide user-friendly messages based on status code
	switch statusCode {
	case http.StatusBadRequest:
		return fmt.Errorf("invalid request (400): %s\n\nTip: Check image dimensions (512-2048, multiple of 64) and quality (standard/premium)", errorMsg)

	case http.StatusUnauthorized:
		return fmt.Errorf("authentication failed (401): %s\n\nTip: Check your AWS Bedrock API key. Run 'gimage auth bedrock' to update credentials", errorMsg)

	case http.StatusForbidden:
		return fmt.Errorf("access denied (403): %s\n\nTip: Ensure you have enabled model access in AWS Bedrock console and your API key has the correct permissions", errorMsg)

	case http.StatusNotFound:
		return fmt.Errorf("model not found (404): %s\n\nTip: Nova Canvas may not be available in region %s. Try us-east-1", errorMsg, c.region)

	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limit exceeded (429): %s\n\nTip: Bedrock rate limits: 10 requests/second. Wait and retry.", errorMsg)

	case http.StatusInternalServerError:
		return fmt.Errorf("server error (500): %s\n\nTip: AWS Bedrock service issue. Try again later.", errorMsg)

	case http.StatusServiceUnavailable:
		return fmt.Errorf("service unavailable (503): %s\n\nTip: AWS Bedrock is temporarily unavailable. Retry in a few moments.", errorMsg)

	default:
		return fmt.Errorf("HTTP error %d: %s", statusCode, errorMsg)
	}
}

// Close cleans up resources (no-op for REST client, implements interface)
func (c *BedrockRESTClient) Close() error {
	return nil
}
