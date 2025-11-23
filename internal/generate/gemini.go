package generate

import (
	"context"
	"fmt"
	"time"

	"github.com/apresai/gimage/pkg/models"
	"google.golang.org/genai"
)

const (
	defaultModel        = "gemini-3-pro-image-preview"
	defaultSize         = "1024x1024"
	maxRetries          = 3
	retryBackoffInitial = 1 * time.Second
	retryBackoffMax     = 10 * time.Second
)

// GeminiClient handles interactions with the Gemini API for image generation
type GeminiClient struct {
	apiKey string
	model  string
	client *genai.Client
}

// NewGeminiClient creates a new Gemini API client
// Returns an error if the API key is empty or client initialization fails
func NewGeminiClient(apiKey string) (*GeminiClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("gemini API key is required")
	}

	// Initialize the Gemini client
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	return &GeminiClient{
		apiKey: apiKey,
		model:  defaultModel,
		client: client,
	}, nil
}

// SetModel updates the model to use for generation
func (c *GeminiClient) SetModel(model string) {
	if model != "" {
		c.model = model
	}
}

// GenerateImage generates an image from a text prompt using the Gemini API
// It implements retry logic with exponential backoff for transient failures
func (c *GeminiClient) GenerateImage(ctx context.Context, prompt string, options models.GenerateOptions) (*models.GeneratedImage, error) {
	if prompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	// Apply default options
	if options.Model == "" {
		options.Model = c.model
	}
	if options.Size == "" {
		options.Size = defaultSize
	}

	// Enhance the prompt for better results
	enhancedPrompt := EnhancePrompt(prompt)

	var lastErr error
	backoff := retryBackoffInitial

	// Retry loop with exponential backoff
	for attempt := 1; attempt <= maxRetries; attempt++ {
		result, err := c.generateWithRetry(ctx, enhancedPrompt, options)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Don't retry on certain errors (invalid prompts, auth failures, etc.)
		if !isRetryableError(err) {
			return nil, fmt.Errorf("non-retryable error: %w", err)
		}

		// Wait before retrying (unless it's the last attempt)
		if attempt < maxRetries {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled during retry backoff: %w", ctx.Err())
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

// generateWithRetry performs a single generation attempt
func (c *GeminiClient) generateWithRetry(ctx context.Context, prompt string, options models.GenerateOptions) (*models.GeneratedImage, error) {
	// Build the generation config
	config := &genai.GenerateImagesConfig{
		NegativePrompt: options.NegativePrompt,
		NumberOfImages: 1,
	}

	// Parse aspect ratio or size
	if options.AspectRatio != "" {
		config.AspectRatio = options.AspectRatio
	}

	// Add person generation control if needed
	config.PersonGeneration = genai.PersonGenerationDontAllow

	// Generate the image using the Models API
	resp, err := c.client.Models.GenerateImages(ctx, options.Model, prompt, config)
	if err != nil {
		return nil, fmt.Errorf("generation request failed: %w", err)
	}

	// Extract image data from response
	if len(resp.GeneratedImages) == 0 {
		return nil, fmt.Errorf("no images returned from API")
	}

	generatedImg := resp.GeneratedImages[0]

	// Get the image data
	var imageData []byte
	var imageFormat string

	if generatedImg.Image != nil && generatedImg.Image.ImageBytes != nil {
		imageData = generatedImg.Image.ImageBytes
		imageFormat = extractFormatFromMimeType(generatedImg.Image.MIMEType)
	} else {
		return nil, fmt.Errorf("no image data found in response")
	}

	// Parse dimensions from size option (e.g., "1024x1024")
	width, height := parseSizeString(options.Size)

	// Build metadata
	metadata := map[string]string{
		"model":      options.Model,
		"prompt":     prompt,
		"size":       options.Size,
		"style":      options.Style,
		"generated":  time.Now().UTC().Format(time.RFC3339),
		"api":        "gemini",
	}

	if options.Seed != 0 {
		metadata["seed"] = fmt.Sprintf("%d", options.Seed)
	}

	if options.NegativePrompt != "" {
		metadata["negative_prompt"] = options.NegativePrompt
	}

	return &models.GeneratedImage{
		Data:     imageData,
		Format:   imageFormat,
		Width:    width,
		Height:   height,
		Metadata: metadata,
	}, nil
}

// ValidateCredentials checks if the API credentials are valid
func (c *GeminiClient) ValidateCredentials() error {
	if c.apiKey == "" {
		return fmt.Errorf("API key is not set")
	}

	// Perform a simple API call to validate credentials
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to list models as a lightweight validation
	_, err := c.client.Models.List(ctx, nil)
	if err != nil {
		return fmt.Errorf("credential validation failed: %w", err)
	}

	return nil
}

// Close closes the Gemini client and releases resources
func (c *GeminiClient) Close() error {
	// The genai.Client doesn't have a Close method in the current SDK version
	// Resources will be cleaned up by the garbage collector
	return nil
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Retry on common transient errors
	retryablePatterns := []string{
		"rate limit",
		"quota exceeded",
		"timeout",
		"deadline exceeded",
		"connection",
		"unavailable",
		"503",
		"429",
	}

	for _, pattern := range retryablePatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// parseSizeString parses a size string like "1024x1024" into width and height
func parseSizeString(size string) (width, height int) {
	// Default to 1024x1024 if parsing fails
	width, height = 1024, 1024

	var w, h int
	_, err := fmt.Sscanf(size, "%dx%d", &w, &h)
	if err == nil && w > 0 && h > 0 {
		width, height = w, h
	}

	return width, height
}

// extractFormatFromMimeType extracts the image format from a MIME type
func extractFormatFromMimeType(mimeType string) string {
	switch mimeType {
	case "image/png":
		return "png"
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	default:
		return "png" // Default to PNG
	}
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
		 len(s) > len(substr) &&
		 (s[:len(substr)] == substr ||
		  s[len(s)-len(substr):] == substr ||
		  containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
