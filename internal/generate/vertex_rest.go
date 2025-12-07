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
	"strings"
	"time"

	"github.com/apresai/gimage/internal/observability"
	"github.com/apresai/gimage/pkg/models"
	"github.com/sony/gobreaker"
)

// Vertex AI Platform API endpoint (supports API key authentication)
// Format: https://{location}-aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/publishers/google/models/{model}:predict
const vertexAIPlatformEndpoint = "https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:predict"

// VertexRESTClient uses Vertex AI REST API for image generation
type VertexRESTClient struct {
	apiKey         string
	projectID      string
	location       string
	model          string
	httpClient     *http.Client
	log            *observability.VerboseLogger
	circuitBreaker *gobreaker.CircuitBreaker
}

// NewVertexRESTClient creates a new Vertex REST API client
func NewVertexRESTClient(apiKey, projectID, location string) (*VertexRESTClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if projectID == "" {
		// Try to get from environment
		projectID = os.Getenv("VERTEX_PROJECT")
		if projectID == "" {
			projectID = "gen-lang-client-0241846458" // Default project
		}
	}

	if location == "" {
		location = "us-central1" // Default location
	}

	return &VertexRESTClient{
		apiKey:    apiKey,
		projectID: projectID,
		location:  location,
		model:     "imagen-4.0-generate-001", // Default model (AI Studio API)
		log:       observability.NewVerboseLogger(observability.ComponentVertex),
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
		circuitBreaker: newCircuitBreaker("VertexAPI"),
	}, nil
}

// GenerateImage generates an image using Vertex AI REST API
func (c *VertexRESTClient) GenerateImage(ctx context.Context, prompt string, options models.GenerateOptions) (*models.GeneratedImage, error) {
	// Validate prompt
	if err := ValidatePrompt(prompt); err != nil {
		return nil, err
	}

	// Enhance prompt for better results
	enhancedPrompt := EnhancePrompt(prompt)

	// Use custom model if provided
	modelName := c.model
	if options.Model != "" {
		modelName = options.Model
	}

	// Generate image with circuit breaker and retry logic
	var lastErr error
	backoff := retryBackoffInitial

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Execute through circuit breaker
		result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
			return c.generateWithRetry(ctx, modelName, enhancedPrompt, options)
		})

		if err == nil {
			return result.(*models.GeneratedImage), nil
		}

		lastErr = err

		// Check if circuit breaker is open
		if isCircuitBreakerError(err) {
			c.log.Debug("Circuit breaker is open, failing fast")
			return nil, fmt.Errorf("API circuit breaker is open (too many failures): %w", err)
		}

		// Check if error is retryable
		if !isRetryableError(err) {
			return nil, err
		}

		// Don't sleep after the last attempt
		if attempt < maxRetries {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			case <-time.After(backoff):
				// Exponential backoff with cap
				backoff *= 2
				if backoff > retryBackoffMax {
					backoff = retryBackoffMax
				}
			}
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// generateWithRetry performs a single generation attempt using REST API
func (c *VertexRESTClient) generateWithRetry(ctx context.Context, modelName, prompt string, options models.GenerateOptions) (*models.GeneratedImage, error) {
	// Build the prompt with options
	fullPrompt := buildPromptWithOptions(prompt, options)

	c.log.Debug("Building request for model: %s", modelName)
	c.log.Debug("Project: %s, Location: %s", c.projectID, c.location)
	c.log.Debug("Full prompt: %s", fullPrompt)

	// Determine aspect ratio from size
	aspectRatio := "1:1" // Default
	if options.Size != "" {
		width, height := parseDimensions(options.Size)
		c.log.Debug("Requested dimensions: %dx%d", width, height)

		// Calculate aspect ratio
		if width == height {
			aspectRatio = "1:1"
		} else if width > height {
			ratio := float64(width) / float64(height)
			if ratio >= 1.7 && ratio <= 1.9 {
				aspectRatio = "16:9"
			} else if ratio >= 1.4 && ratio <= 1.6 {
				aspectRatio = "3:2"
			} else {
				aspectRatio = "4:3"
			}
		} else {
			ratio := float64(height) / float64(width)
			if ratio >= 1.7 && ratio <= 1.9 {
				aspectRatio = "9:16"
			} else if ratio >= 1.4 && ratio <= 1.6 {
				aspectRatio = "2:3"
			} else {
				aspectRatio = "3:4"
			}
		}
	}

	c.log.Debug("Using aspect ratio: %s", aspectRatio)

	// Determine number of images (Vertex AI Imagen supports 1-8)
	sampleCount := 1
	if options.NumberOfImages > 0 && options.NumberOfImages <= 8 {
		sampleCount = options.NumberOfImages
	}

	// Build request payload for Vertex AI Imagen
	// Format according to Vertex AI Imagen API:
	// https://cloud.google.com/vertex-ai/docs/generative-ai/image/generate-images
	requestBody := map[string]interface{}{
		"instances": []map[string]interface{}{
			{
				"prompt": fullPrompt,
			},
		},
		"parameters": map[string]interface{}{
			"sampleCount":  sampleCount,
			"aspectRatio":  aspectRatio,
		},
	}

	// Add negative prompt if provided
	if options.NegativePrompt != "" {
		requestBody["parameters"].(map[string]interface{})["negativePrompt"] = options.NegativePrompt
	}

	// Add seed if provided
	if options.Seed != 0 {
		requestBody["parameters"].(map[string]interface{})["seed"] = options.Seed
	}

	// Add output format if provided (jpeg, png, webp)
	if options.OutputFormat != "" {
		mimeType := "image/png" // default
		switch strings.ToLower(options.OutputFormat) {
		case "jpeg", "jpg":
			mimeType = "image/jpeg"
		case "webp":
			mimeType = "image/webp"
		case "png":
			mimeType = "image/png"
		}
		requestBody["parameters"].(map[string]interface{})["outputOptions"] = map[string]string{
			"mimeType": mimeType,
		}
		c.log.Debug("Using output format: %s", mimeType)
	}

	// Marshal request to JSON
	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	c.log.Debug("Request body: %s", string(requestJSON))

	// Build API URL using Vertex AI Platform endpoint
	// Format: https://{location}-aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/publishers/google/models/{model}:predict
	apiURL := fmt.Sprintf(vertexAIPlatformEndpoint, c.location, c.projectID, c.location, modelName)

	// Mask the API key in logs
	maskedKey := c.apiKey
	if len(maskedKey) > 8 {
		maskedKey = maskedKey[:8] + "***"
	}
	c.log.Debug("API URL: %s", apiURL)
	c.log.Debug("Using API key: %s", maskedKey)

	// Create HTTP request with API key header
	// Vertex AI Platform uses x-goog-api-key header
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(requestJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)

	c.log.Debug("Sending request to Vertex AI Platform API...")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Debug("Request failed: %v", err)
		return nil, enhanceError(err)
	}
	defer resp.Body.Close()

	c.log.Debug("Response status: %d %s", resp.StatusCode, resp.Status)

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log response (truncated)
	if len(body) > 500 {
		c.log.Debug("Response body (first 500 chars): %s...", string(body[:500]))
	} else {
		c.log.Debug("Response body: %s", string(body))
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleHTTPError(resp.StatusCode, body)
	}

	// Parse response
	var response vertexPredictResponse
	if err := json.Unmarshal(body, &response); err != nil {
		c.log.Debug("Failed to parse response: %v", err)
		return nil, fmt.Errorf("failed to parse response: %w (body: %s)", err, string(body))
	}

	// Validate response structure
	if len(response.Predictions) == 0 {
		c.log.Debug("No predictions in response")
		return nil, fmt.Errorf("no image generated from prompt")
	}

	prediction := response.Predictions[0]

	// Check for base64 encoded image
	var imageData []byte
	var mimeType string

	if prediction.BytesBase64Encoded != "" {
		// Decode base64 image data
		imageData, err = base64.StdEncoding.DecodeString(prediction.BytesBase64Encoded)
		if err != nil {
			c.log.Debug("Failed to decode base64: %v", err)
			return nil, fmt.Errorf("failed to decode base64 image data: %w", err)
		}
		mimeType = prediction.MimeType
		c.log.Debug("Successfully decoded image: %d bytes, mime=%s", len(imageData), mimeType)
	} else {
		c.log.Debug("No image data found in prediction")
		return nil, fmt.Errorf("no image data found in response")
	}

	// Determine format from MIME type
	format := "png"
	if mimeType != "" {
		switch mimeType {
		case "image/jpeg":
			format = "jpg"
		case "image/png":
			format = "png"
		case "image/webp":
			format = "webp"
		}
	}

	// Parse dimensions from options
	width, height := parseDimensions(options.Size)

	c.log.Debug("Successfully generated image: %d bytes, format=%s", len(imageData), format)

	return &models.GeneratedImage{
		Data:   imageData,
		Format: format,
		Width:  width,
		Height: height,
		Metadata: map[string]string{
			"model":    modelName,
			"prompt":   prompt,
			"style":    options.Style,
			"api":      "imagen-genai",
			"project":  c.projectID,
			"location": c.location,
		},
	}, nil
}

// handleHTTPError handles HTTP error responses from the API
func (c *VertexRESTClient) handleHTTPError(statusCode int, body []byte) error {
	// Try to parse error response
	var errorResp struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
		return fmt.Errorf("API error %d: %s", statusCode, errorResp.Error.Message)
	}

	// Generic error messages based on status code
	switch statusCode {
	case 401:
		return fmt.Errorf("authentication failed (401): invalid API key. Please check your VERTEX_API_KEY")
	case 403:
		return fmt.Errorf("permission denied (403): API key may not have access to Vertex AI or project")
	case 404:
		return fmt.Errorf("not found (404): check project ID (%s) and model name", c.projectID)
	case 429:
		return fmt.Errorf("rate limit exceeded (429): too many requests, please try again later")
	case 500, 502, 503:
		return fmt.Errorf("server error (%d): the API is temporarily unavailable, please retry", statusCode)
	default:
		return fmt.Errorf("HTTP error %d: %s", statusCode, string(body))
	}
}

// Close closes the client connection
func (c *VertexRESTClient) Close() error {
	// HTTP client doesn't need explicit closing
	return nil
}

// Request/Response structs for Vertex AI REST API
type vertexPredictResponse struct {
	Predictions []vertexPrediction `json:"predictions"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type vertexPrediction struct {
	BytesBase64Encoded string `json:"bytesBase64Encoded"`
	MimeType           string `json:"mimeType"`
}
