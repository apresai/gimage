package generate

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/apresai/gimage/internal/observability"
	"github.com/apresai/gimage/pkg/models"
	"github.com/sony/gobreaker"
)

// Gemini REST API endpoint
const geminiAPIEndpoint = "https://generativelanguage.googleapis.com/v1beta/models"

// GeminiRESTClient uses Gemini REST API for image generation
type GeminiRESTClient struct {
	apiKey         string
	model          string
	httpClient     *http.Client
	log            *observability.VerboseLogger
	circuitBreaker *gobreaker.CircuitBreaker
}

// NewGeminiRESTClient creates a new Gemini REST API client
func NewGeminiRESTClient(apiKey string) (*GeminiRESTClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	return &GeminiRESTClient{
		apiKey: apiKey,
		model:  DefaultModel,
		log:    observability.NewVerboseLogger(observability.ComponentGemini),
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
		circuitBreaker: newCircuitBreaker("GeminiAPI"),
	}, nil
}

// GenerateImage generates an image using Gemini REST API
func (c *GeminiRESTClient) GenerateImage(ctx context.Context, prompt string, options models.GenerateOptions) (*models.GeneratedImage, error) {
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

// isGemini3Pro returns true if the model is Gemini 3 Pro
func isGemini3Pro(modelName string) bool {
	return strings.Contains(modelName, "gemini-3") || strings.Contains(modelName, "pro-image-preview")
}

// generateWithRetry performs a single generation attempt using REST API
func (c *GeminiRESTClient) generateWithRetry(ctx context.Context, modelName, prompt string, options models.GenerateOptions) (*models.GeneratedImage, error) {
	// Build the prompt with options
	fullPrompt := buildPromptWithOptions(prompt, options)

	// Parse dimensions from options
	width, height := parseDimensions(options.Size)

	c.log.Debug("Building request for model: %s", modelName)
	c.log.Debug("Full prompt: %s", fullPrompt)
	c.log.Debug("Requested dimensions: %dx%d", width, height)

	// Build generation config based on model
	genConfig := &geminiGenerationConfig{}

	// Gemini 3 Pro uses TEXT+IMAGE modalities and supports imageConfig
	if isGemini3Pro(modelName) {
		genConfig.ResponseModalities = []string{"TEXT", "IMAGE"}

		// Add imageConfig for Gemini 3 Pro (supports 4K, aspect ratio)
		imageConfig := &geminiImageConfig{}

		// Set imageSize if specified (1K, 2K, 4K)
		if options.ImageSize != "" {
			imageConfig.ImageSize = strings.ToUpper(options.ImageSize)
			c.log.Debug("Using imageSize: %s", imageConfig.ImageSize)
		}

		// Set aspectRatio - use explicit value, infer from Size, or default to 1:1
		if options.AspectRatio != "" {
			imageConfig.AspectRatio = options.AspectRatio
			c.log.Debug("Using explicit aspectRatio: %s", imageConfig.AspectRatio)
		} else if options.Size != "" && width > 0 && height > 0 {
			// Infer aspect ratio from Size dimensions for Gemini 3 Pro
			imageConfig.AspectRatio = inferAspectRatio(width, height)
			c.log.Debug("Inferred aspectRatio from size %dx%d: %s", width, height, imageConfig.AspectRatio)
		} else if imageConfig.ImageSize != "" {
			// Default to 1:1 when imageSize is specified but no aspectRatio
			// This prevents the API from defaulting to 16:9
			imageConfig.AspectRatio = "1:1"
			c.log.Debug("Using default aspectRatio 1:1 for imageSize %s", imageConfig.ImageSize)
		}

		// Only add imageConfig if we have settings
		if imageConfig.ImageSize != "" || imageConfig.AspectRatio != "" {
			genConfig.ImageConfig = imageConfig
		}
	} else {
		// Standard Gemini 2.5 Flash: IMAGE only, supports aspectRatio but not imageSize
		genConfig.ResponseModalities = []string{"IMAGE"}

		// Gemini 2.5 Flash also supports aspectRatio
		if options.AspectRatio != "" {
			imageConfig := &geminiImageConfig{AspectRatio: options.AspectRatio}
			genConfig.ImageConfig = imageConfig
			c.log.Debug("Using aspectRatio for Gemini Flash: %s", options.AspectRatio)
		} else if options.Size != "" && width > 0 && height > 0 {
			// Infer aspect ratio from Size dimensions
			aspectRatio := inferAspectRatio(width, height)
			imageConfig := &geminiImageConfig{AspectRatio: aspectRatio}
			genConfig.ImageConfig = imageConfig
			c.log.Debug("Inferred aspectRatio from size %dx%d: %s", width, height, aspectRatio)
		}
	}

	// Build request payload using Gemini's generateContent API format
	request := geminiGenerateContentRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{
						Text: fullPrompt,
					},
				},
			},
		},
		GenerationConfig: genConfig,
	}

	// Marshal request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	c.log.Debug("Request body: %s", string(requestBody))

	// Build API URL - use generateContent endpoint
	apiURL := fmt.Sprintf("%s/%s:generateContent?key=%s", geminiAPIEndpoint, modelName, c.apiKey)

	c.log.Debug("API URL: %s", strings.Replace(apiURL, c.apiKey, "***KEY***", -1))

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	c.log.Debug("Sending request to Gemini API...")

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

	// Parse response using Gemini's generateContent response format
	var response geminiGenerateContentResponse
	if err := json.Unmarshal(body, &response); err != nil {
		c.log.Debug("Failed to parse response: %v", err)
		return nil, fmt.Errorf("failed to parse response: %w (body: %s)", err, string(body))
	}

	// Validate response structure
	if len(response.Candidates) == 0 {
		c.log.Debug("No candidates in response")
		return nil, fmt.Errorf("no image generated from prompt")
	}

	candidate := response.Candidates[0]
	if candidate.Content.Parts == nil || len(candidate.Content.Parts) == 0 {
		c.log.Debug("No parts in candidate content")
		return nil, fmt.Errorf("no content parts in response")
	}

	// Find the image part (should have inline_data)
	var imageData []byte
	var mimeType string
	found := false

	for i, part := range candidate.Content.Parts {
		c.log.Debug("Part %d: has InlineData=%v", i, part.InlineData != nil)
		if part.InlineData != nil && part.InlineData.Data != "" {
			// Decode base64 image data
			imageData, err = base64.StdEncoding.DecodeString(part.InlineData.Data)
			if err != nil {
				c.log.Debug("Failed to decode base64 from part %d: %v", i, err)
				return nil, fmt.Errorf("failed to decode base64 image data: %w", err)
			}
			mimeType = part.InlineData.MimeType
			found = true
			c.log.Debug("Found image data in part %d: %d bytes, mime=%s", i, len(imageData), mimeType)
			break
		}
	}

	if !found {
		c.log.Debug("No inline_data found in any parts")
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

	c.log.Debug("Successfully generated image: %d bytes, format=%s", len(imageData), format)

	// For Gemini 3 Pro with native imageSize (1K, 2K, 4K), preserve the native resolution
	// by setting width/height to 0, which skips dimension enforcement in SaveImage
	finalWidth, finalHeight := width, height
	if isGemini3Pro(modelName) && options.ImageSize != "" {
		finalWidth, finalHeight = 0, 0
		c.log.Debug("Preserving native %s resolution from Gemini 3 Pro", options.ImageSize)
	}

	return &models.GeneratedImage{
		Data:   imageData,
		Format: format,
		Width:  finalWidth,
		Height: finalHeight,
		Metadata: map[string]string{
			"model":     modelName,
			"prompt":    prompt,
			"style":     options.Style,
			"api":       "gemini-rest",
			"imageSize": options.ImageSize,
		},
	}, nil
}

// handleHTTPError handles HTTP error responses from the API
func (c *GeminiRESTClient) handleHTTPError(statusCode int, body []byte) error {
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
		return fmt.Errorf("authentication failed (401): invalid API key. Please check your GEMINI_API_KEY")
	case 403:
		return fmt.Errorf("permission denied (403): API key may not have access to image generation")
	case 429:
		return fmt.Errorf("rate limit exceeded (429): too many requests, please try again later")
	case 500, 502, 503:
		return fmt.Errorf("server error (%d): the API is temporarily unavailable, please retry", statusCode)
	default:
		return fmt.Errorf("HTTP error %d: %s", statusCode, string(body))
	}
}

// Close closes the client connection
func (c *GeminiRESTClient) Close() error {
	// HTTP client doesn't need explicit closing
	return nil
}

// Request/Response structs for Gemini REST API (generateContent format)
type geminiGenerateContentRequest struct {
	Contents         []geminiContent          `json:"contents"`
	GenerationConfig *geminiGenerationConfig  `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type geminiPart struct {
	Text       string           `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inlineData,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"` // base64 encoded
}

type geminiGenerationConfig struct {
	ResponseModalities []string          `json:"responseModalities,omitempty"`
	Temperature        *float64          `json:"temperature,omitempty"`
	TopP               *float64          `json:"topP,omitempty"`
	TopK               *int              `json:"topK,omitempty"`
	ImageConfig        *geminiImageConfig `json:"imageConfig,omitempty"` // For Gemini 3 Pro
}

// geminiImageConfig configures image generation for Gemini 3 Pro
type geminiImageConfig struct {
	AspectRatio string `json:"aspectRatio,omitempty"` // e.g., "16:9", "1:1", "4:3"
	ImageSize   string `json:"imageSize,omitempty"`   // "1K", "2K", "4K" (uppercase required)
}

type geminiGenerateContentResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
}

type geminiCandidate struct {
	Content       geminiContent `json:"content"`
	FinishReason  string        `json:"finishReason,omitempty"`
	SafetyRatings []interface{} `json:"safetyRatings,omitempty"`
}

// buildPromptWithOptions enhances the prompt with style options
func buildPromptWithOptions(prompt string, options models.GenerateOptions) string {
	enhanced := prompt
	if options.Style != "" {
		enhanced = fmt.Sprintf("%s, %s style", enhanced, options.Style)
	}
	return enhanced
}

// parseDimensions parses "WIDTHxHEIGHT" string
func parseDimensions(size string) (int, int) {
	if size == "" {
		return 1024, 1024
	}

	parts := strings.Split(size, "x")
	if len(parts) != 2 {
		return 1024, 1024
	}

	width, err := strconv.Atoi(parts[0])
	if err != nil || width <= 0 {
		return 1024, 1024
	}

	height, err := strconv.Atoi(parts[1])
	if err != nil || height <= 0 {
		return 1024, 1024
	}

	return width, height
}

// inferAspectRatio determines the best matching aspect ratio for given dimensions
// Supported ratios: 1:1, 16:9, 9:16, 4:3, 3:4, 3:2, 2:3
func inferAspectRatio(width, height int) string {
	if width <= 0 || height <= 0 {
		return "1:1"
	}

	ratio := float64(width) / float64(height)

	// Define aspect ratios with their numeric values
	type aspectRatio struct {
		name  string
		value float64
	}

	ratios := []aspectRatio{
		{"1:1", 1.0},
		{"16:9", 16.0 / 9.0},   // 1.778
		{"9:16", 9.0 / 16.0},   // 0.5625
		{"4:3", 4.0 / 3.0},     // 1.333
		{"3:4", 3.0 / 4.0},     // 0.75
		{"3:2", 3.0 / 2.0},     // 1.5
		{"2:3", 2.0 / 3.0},     // 0.667
		{"5:4", 5.0 / 4.0},     // 1.25
		{"4:5", 4.0 / 5.0},     // 0.8
	}

	// Find closest match
	closestRatio := ratios[0]
	smallestDiff := abs(ratio - closestRatio.value)

	for _, r := range ratios[1:] {
		diff := abs(ratio - r.value)
		if diff < smallestDiff {
			smallestDiff = diff
			closestRatio = r
		}
	}

	return closestRatio.name
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// enhanceError provides more context for API errors
func enhanceError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Check for authentication errors
	if contains(errStr, "401") || contains(errStr, "unauthorized") {
		return fmt.Errorf("authentication failed: invalid API key. Set GEMINI_API_KEY or use --api-key flag: %w", err)
	}

	// Check for permission errors
	if contains(errStr, "403") || contains(errStr, "forbidden") {
		return fmt.Errorf("permission denied: check your API key and quota: %w", err)
	}

	// Check for rate limiting
	if contains(errStr, "429") || contains(errStr, "rate limit") {
		return fmt.Errorf("rate limit exceeded: too many requests. Try again later: %w", err)
	}

	// Check for server errors
	if contains(errStr, "500") || contains(errStr, "502") || contains(errStr, "503") {
		return fmt.Errorf("server error: Gemini API is experiencing issues. Try again later: %w", err)
	}

	return err
}
