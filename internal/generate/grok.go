// Package generate provides image generation clients for various AI providers.
package generate

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/apresai/gimage/pkg/models"
	"github.com/sony/gobreaker"
)

const (
	grokBaseURL      = "https://api.x.ai/v1"
	grokDefaultModel = "grok-2-image-1212" // Versioned model name works more reliably
)

// GrokClient handles interactions with the xAI Grok API for image generation
type GrokClient struct {
	apiKey         string
	httpClient     *http.Client
	circuitBreaker *gobreaker.CircuitBreaker
}

// GrokImageRequest represents the request body for Grok image generation
type GrokImageRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              int    `json:"n,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
}

// GrokImageResponse represents the response from Grok image generation
type GrokImageResponse struct {
	Created int64            `json:"created"`
	Data    []GrokImageData  `json:"data"`
	Error   *GrokErrorDetail `json:"error,omitempty"`
}

// GrokImageData contains the generated image data
type GrokImageData struct {
	URL     string `json:"url,omitempty"`
	B64JSON string `json:"b64_json,omitempty"`
}

// GrokErrorDetail contains error information from the API
type GrokErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

// NewGrokClient creates a new xAI Grok API client
func NewGrokClient(apiKey string) (*GrokClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("grok API key is required")
	}

	return &GrokClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Image generation can take time
		},
		circuitBreaker: newCircuitBreaker("GrokAPI"),
	}, nil
}

// GenerateImage generates an image from a text prompt using the Grok API
func (c *GrokClient) GenerateImage(ctx context.Context, prompt string, options models.GenerateOptions) ([]*models.GeneratedImage, error) {
	if prompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	// Enhance the prompt for better results
	enhancedPrompt := EnhancePrompt(prompt)

	// Determine number of images (Grok supports 1-10)
	numImages := 1
	if options.NumberOfImages > 0 && options.NumberOfImages <= 10 {
		numImages = options.NumberOfImages
	}

	// Build request - Grok doesn't support size/quality/style parameters
	request := GrokImageRequest{
		Model:          grokDefaultModel,
		Prompt:         enhancedPrompt,
		N:              numImages,
		ResponseFormat: "b64_json", // Get base64 data directly
	}

	// Use custom model if specified
	if options.Model != "" && options.Model != "grok" && options.Model != "grok-2-image" {
		request.Model = options.Model
	}

	// Execute through circuit breaker
	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.doRequest(ctx, request)
	})

	if err != nil {
		if isCircuitBreakerError(err) {
			return nil, fmt.Errorf("API circuit breaker is open (too many failures): %w", err)
		}
		return nil, err
	}

	if err != nil {
		if isCircuitBreakerError(err) {
			return nil, fmt.Errorf("API circuit breaker is open (too many failures): %w", err)
		}
		return nil, err
	}

	return result.([]*models.GeneratedImage), nil
}

// doRequest performs the actual HTTP request to the Grok API
func (c *GrokClient) doRequest(ctx context.Context, request GrokImageRequest) ([]*models.GeneratedImage, error) {
	// Marshal request body
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/images/generations", grokBaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
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
	var grokResp GrokImageResponse
	if err := json.Unmarshal(body, &grokResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API error in response
	if grokResp.Error != nil {
		return nil, fmt.Errorf("API error: %s (type: %s)", grokResp.Error.Message, grokResp.Error.Type)
	}

	// Check we got image data
	if len(grokResp.Data) == 0 {
		return nil, fmt.Errorf("no images returned from API")
	}

	var generatedImages []*models.GeneratedImage

	for i, imageData := range grokResp.Data {
		var rawImageData []byte

		// Handle base64 response
		if imageData.B64JSON != "" {
			rawImageData, err = base64.StdEncoding.DecodeString(imageData.B64JSON)
			if err != nil {
				continue // Skip if decode fails
			}
		} else if imageData.URL != "" {
			// Download from URL if base64 not available
			rawImageData, err = downloadImage(ctx, imageData.URL)
			if err != nil {
				continue // Skip if download fails
			}
		} else {
			continue // Skip if no data
		}

		// Detect image format from data
		format := detectImageFormat(rawImageData)

		// Get actual image dimensions from the data
		width, height, err := GetImageDimensionsFromBytes(rawImageData)
		if err != nil {
			// Fallback to 0,0 which will skip dimension enforcement
			width, height = 0, 0
		}

		// Build metadata
		metadata := map[string]string{
			"model":     request.Model,
			"prompt":    request.Prompt,
			"api":       "grok",
			"generated": time.Now().UTC().Format(time.RFC3339),
			"candidate": fmt.Sprintf("%d", i),
		}

		generatedImages = append(generatedImages, &models.GeneratedImage{
			Data:     rawImageData,
			Format:   format,
			Width:    width,
			Height:   height,
			Metadata: metadata,
		})
	}

	if len(generatedImages) == 0 {
		return nil, fmt.Errorf("no images returned from API")
	}

	return generatedImages, nil
}

// handleHTTPError converts HTTP errors to user-friendly messages
func (c *GrokClient) handleHTTPError(statusCode int, body []byte) error {
	// Try to parse error response
	var errResp struct {
		Error *GrokErrorDetail `json:"error"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != nil {
		switch statusCode {
		case http.StatusUnauthorized:
			return fmt.Errorf("authentication failed: %s\nHint: Check your GROK_API_KEY is valid", errResp.Error.Message)
		case http.StatusTooManyRequests:
			return fmt.Errorf("rate limit exceeded: %s\nHint: Wait a moment and try again", errResp.Error.Message)
		case http.StatusBadRequest:
			return fmt.Errorf("invalid request: %s", errResp.Error.Message)
		default:
			return fmt.Errorf("API error (status %d): %s", statusCode, errResp.Error.Message)
		}
	}

	// Generic error if we couldn't parse the response
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("authentication failed (status 401)\nHint: Check your GROK_API_KEY is valid")
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limit exceeded (status 429)\nHint: Wait a moment and try again")
	case http.StatusBadRequest:
		return fmt.Errorf("invalid request (status 400): %s", string(body))
	default:
		return fmt.Errorf("API error (status %d): %s", statusCode, string(body))
	}
}

// detectImageFormat detects the image format from the raw bytes
func detectImageFormat(data []byte) string {
	if len(data) < 8 {
		return "png" // Default
	}

	// Check magic bytes
	switch {
	case data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47:
		return "png"
	case data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return "jpg"
	case data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46:
		return "webp"
	case data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46:
		return "gif"
	default:
		return "png" // Default to PNG
	}
}

// Close cleans up resources (implements ImageGenerator interface)
func (c *GrokClient) Close() error {
	return nil
}
