package generate

import (
	"bytes"
	"context"
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
		projectID = os.Getenv("VERTEX_PROJECT")
		if projectID == "" {
			return nil, fmt.Errorf("VERTEX_PROJECT is required for Vertex AI Express Mode. Set via environment variable or config file")
		}
	}

	if location == "" {
		location = "us-central1" // Default location
	}

	return &VertexRESTClient{
		apiKey:    apiKey,
		projectID: projectID,
		location:  location,
		model:     "gemini-3.1-flash-image", // Default model
		log:       observability.NewVerboseLogger(observability.ComponentVertex),
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
		circuitBreaker: newCircuitBreaker("VertexAPI"),
	}, nil
}

// GenerateImage generates an image using Vertex AI REST API
func (c *VertexRESTClient) GenerateImage(ctx context.Context, prompt string, options models.GenerateOptions) ([]*models.GeneratedImage, error) {
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
			return result.([]*models.GeneratedImage), nil
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
func (c *VertexRESTClient) generateWithRetry(ctx context.Context, modelName, prompt string, options models.GenerateOptions) ([]*models.GeneratedImage, error) {
	if strings.Contains(modelName, "gemini") {
		// Build request using the shared gemini request builder
		request, err := buildGeminiGenerateContentRequest(c.log, modelName, prompt, options)
		if err != nil {
			return nil, err
		}

		// Marshal request to JSON
		requestJSON, err := json.Marshal(request)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}

		c.log.Debug("Gemini request body: %s", string(requestJSON))

		// Build API URL for Vertex AI generateContent endpoint
		apiURL := fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:generateContent", c.location, c.projectID, c.location, modelName)

		maskedKey := c.apiKey
		if len(maskedKey) > 8 {
			maskedKey = maskedKey[:8] + "***"
		}
		c.log.Debug("API URL: %s", apiURL)
		c.log.Debug("Using API key: %s", maskedKey)

		// Create HTTP request
		req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(requestJSON))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-goog-api-key", c.apiKey)

		c.log.Debug("Sending request to Vertex AI Platform API (generateContent)...")

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

		if len(body) > 500 {
			c.log.Debug("Response body (first 500 chars): %s...", string(body[:500]))
		} else {
			c.log.Debug("Response body: %s", string(body))
		}

		// Check for HTTP errors
		if resp.StatusCode != http.StatusOK {
			return nil, c.handleHTTPError(resp.StatusCode, body)
		}

		// Parse response using Gemini's generateContent response format
		var response geminiGenerateContentResponse
		if err := json.Unmarshal(body, &response); err != nil {
			c.log.Debug("Failed to parse response: %v", err)
			return nil, fmt.Errorf("failed to parse response: %w (body: %s)", err, string(body))
		}

		// Extract images using shared helper
		return extractImagesFromGeminiGenerateContentResponse(c.log, response, modelName, prompt, options, "vertex-rest")
	}

	return nil, fmt.Errorf("unsupported model %q for Vertex AI: only Gemini models are supported", modelName)
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
