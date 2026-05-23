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
func (c *GeminiRESTClient) GenerateImage(ctx context.Context, prompt string, options models.GenerateOptions) ([]*models.GeneratedImage, error) {
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

// isGeminiAdvanced returns true if the model supports TEXT+IMAGE modalities and imageConfig (Gemini 3+ models)
func isGeminiAdvanced(modelName string) bool {
	return strings.Contains(modelName, "gemini-3")
}

// buildGenerateContentRequest assembles the full generateContent request payload
// for a given model + prompt + options. It returns the struct (not a serialized
// payload) so unit tests can introspect the assembled shape without an HTTP
// roundtrip. Returns an error early if the multi-image cap is exceeded or a
// reference image cannot be read.
//
// Gemini 3+ exclusive fields (thinkingConfig, tools[google_search]) are gated
// here via isGeminiAdvanced so callers that pass them on Gemini 2.5 Flash get
// a quietly-stripped request rather than an API-side rejection.
func (c *GeminiRESTClient) buildGenerateContentRequest(modelName, prompt string, options models.GenerateOptions) (geminiGenerateContentRequest, error) {
	// Build the prompt with options
	fullPrompt := buildPromptWithOptions(prompt, options)

	// Parse dimensions from options
	width, height := ParseSizeString(options.Size)

	c.log.Debug("Building request for model: %s", modelName)
	c.log.Debug("Full prompt: %s", fullPrompt)
	c.log.Debug("Requested dimensions: %dx%d", width, height)

	// Build generation config based on model
	genConfig := &geminiGenerationConfig{}

	// Gemini 3+ models use TEXT+IMAGE modalities and support imageConfig
	if isGeminiAdvanced(modelName) {
		genConfig.ResponseModalities = []string{"TEXT", "IMAGE"}

		// Add imageConfig for Gemini 3+ (supports 4K, aspect ratio)
		imageConfig := &geminiImageConfig{}

		// Set imageSize if specified (1K, 2K, 4K)
		if options.ImageSize != "" {
			imageConfig.ImageSize = strings.ToUpper(options.ImageSize)
			c.log.Debug("Using imageSize: %s", imageConfig.ImageSize)
		}

		if options.NegativePrompt != "" {
			imageConfig.NegativePrompt = options.NegativePrompt
			c.log.Debug("Using negativePrompt: %s", options.NegativePrompt)
		}

		// Set aspectRatio - use explicit value, infer from Size, or default to 1:1
		if options.AspectRatio != "" {
			imageConfig.AspectRatio = options.AspectRatio
			c.log.Debug("Using explicit aspectRatio: %s", imageConfig.AspectRatio)
		} else if options.Size != "" && width > 0 && height > 0 {
			imageConfig.AspectRatio = InferAspectRatio(width, height, nil)
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

		// Thinking config (Gemini 3+ only). "off" or empty omits the field entirely.
		if level := strings.ToLower(options.ThinkingLevel); level != "" && level != "off" {
			genConfig.ThinkingConfig = &geminiThinkingConfig{ThinkingLevel: level}
			c.log.Debug("Using thinkingLevel: %s", level)
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
			aspectRatio := InferAspectRatio(width, height, nil)
			imageConfig := &geminiImageConfig{AspectRatio: aspectRatio}
			genConfig.ImageConfig = imageConfig
			c.log.Debug("Inferred aspectRatio from size %dx%d: %s", width, height, aspectRatio)
		}
	}

	// Set common generation config (seed and count)
	if options.Seed != 0 {
		genConfig.Seed = &options.Seed
		c.log.Debug("Using seed: %d", options.Seed)
	}

	if options.NumberOfImages > 1 {
		count := options.NumberOfImages
		if count > 4 {
			count = 4 // Gemini API max candidates is typically 4
		}
		genConfig.CandidateCount = &count
		c.log.Debug("Using candidateCount: %d", count)
	}

	// Build parts list: text first, then any reference images for compositional editing.
	parts := []geminiPart{{Text: fullPrompt}}
	if len(options.InputImages) > 0 {
		maxImages := maxInputImagesForModel(modelName)
		if len(options.InputImages) > maxImages {
			return geminiGenerateContentRequest{}, fmt.Errorf("model %s supports at most %d input images, got %d", modelName, maxImages, len(options.InputImages))
		}
		for _, path := range options.InputImages {
			part, err := readInputImageAsInlineData(path)
			if err != nil {
				return geminiGenerateContentRequest{}, fmt.Errorf("read input image %s: %w", path, err)
			}
			parts = append(parts, part)
			c.log.Debug("Attached input image: %s (%s)", path, part.InlineData.MimeType)
		}
	}

	// Build tools list (Google Search grounding is Gemini 3+ only).
	var tools []geminiTool
	if options.WebSearchGrounding && isGeminiAdvanced(modelName) {
		tools = append(tools, geminiTool{GoogleSearch: &geminiGoogleSearch{}})
		c.log.Debug("Google Search grounding enabled")
	}

	return geminiGenerateContentRequest{
		Contents:         []geminiContent{{Parts: parts}},
		GenerationConfig: genConfig,
		Tools:            tools,
	}, nil
}

// generateWithRetry performs a single generation attempt using REST API
func (c *GeminiRESTClient) generateWithRetry(ctx context.Context, modelName, prompt string, options models.GenerateOptions) ([]*models.GeneratedImage, error) {
	request, err := c.buildGenerateContentRequest(modelName, prompt, options)
	if err != nil {
		return nil, err
	}

	// Parse requested dimensions (used to populate response metadata for
	// non-native-upscale models). buildGenerateContentRequest already used these
	// internally; we re-derive them here for the response-shaping logic below.
	width, height := ParseSizeString(options.Size)

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
	var generatedImages []*models.GeneratedImage
	c.log.Debug("Successfully generated %d potential image(s)", len(response.Candidates))

	// Loop through all candidates and extract images
	for i, candidate := range response.Candidates {
		if candidate.Content.Parts == nil || len(candidate.Content.Parts) == 0 {
			c.log.Debug("Candidate %d has no parts, skipping", i)
			continue
		}

		var candidateData []byte
		var candidateMimeType string
		foundPart := false

		for _, part := range candidate.Content.Parts {
			if part.InlineData != nil && part.InlineData.Data != "" {
				data, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
				if err != nil {
					c.log.Debug("Failed to decode base64 from candidate %d: %v", i, err)
					continue
				}
				candidateData = data
				candidateMimeType = part.InlineData.MimeType
				foundPart = true
				break
			}
		}

		if !foundPart {
			c.log.Debug("No image data found in candidate %d, skipping", i)
			continue
		}

		// Determine format for this candidate
		candFormat := extractFormatFromMimeType(candidateMimeType)

		// Determine dimensions
		finalWidth, finalHeight := width, height
		if isGeminiAdvanced(modelName) && options.ImageSize != "" {
			finalWidth, finalHeight = 0, 0
		}

		generatedImages = append(generatedImages, &models.GeneratedImage{
			Data:   candidateData,
			Format: candFormat,
			Width:  finalWidth,
			Height: finalHeight,
			Metadata: map[string]string{
				"model":       modelName,
				"prompt":      prompt,
				"style":       options.Style,
				"api":         "gemini-rest",
				"imageSize":   options.ImageSize,
				"seed":        fmt.Sprintf("%d", options.Seed),
				"candidate":   fmt.Sprintf("%d", i),
				"resize_mode": options.ResizeMode,
			},
		})
	}

	if len(generatedImages) == 0 {
		return nil, fmt.Errorf("no valid images found in response")
	}

	return generatedImages, nil
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
	Contents         []geminiContent         `json:"contents"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
	Tools            []geminiTool            `json:"tools,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inlineData,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"` // base64 encoded
}

type geminiGenerationConfig struct {
	ResponseModalities []string              `json:"responseModalities,omitempty"`
	Temperature        *float64              `json:"temperature,omitempty"`
	TopP               *float64              `json:"topP,omitempty"`
	TopK               *int                  `json:"topK,omitempty"`
	CandidateCount     *int                  `json:"candidateCount,omitempty"`
	Seed               *int64                `json:"seed,omitempty"`
	ImageConfig        *geminiImageConfig    `json:"imageConfig,omitempty"`
	ThinkingConfig     *geminiThinkingConfig `json:"thinkingConfig,omitempty"`
}

type geminiImageConfig struct {
	AspectRatio    string `json:"aspectRatio,omitempty"`
	ImageSize      string `json:"imageSize,omitempty"` // "1K", "2K", "4K" — API requires uppercase
	NegativePrompt string `json:"negativePrompt,omitempty"`
}

// geminiThinkingConfig controls reasoning depth on Gemini 3+ models.
// Values: "minimal", "low", "medium", "high". Image variants of Gemini 3
// support this knob for layout/composition planning.
type geminiThinkingConfig struct {
	ThinkingLevel string `json:"thinkingLevel,omitempty"`
}

// geminiTool wraps tool declarations. For image generation, only google_search
// grounding is meaningful here.
type geminiTool struct {
	GoogleSearch *geminiGoogleSearch `json:"google_search,omitempty"`
}

// geminiGoogleSearch is sent as an empty object to enable Google Search grounding.
type geminiGoogleSearch struct{}

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

// enhanceError provides more context for API errors
func enhanceError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Check for authentication errors
	if strings.Contains(errStr, "401") || strings.Contains(errStr, "unauthorized") {
		return fmt.Errorf("authentication failed: invalid API key. Set GEMINI_API_KEY or use --api-key flag: %w", err)
	}

	// Check for permission errors
	if strings.Contains(errStr, "403") || strings.Contains(errStr, "forbidden") {
		return fmt.Errorf("permission denied: check your API key and quota: %w", err)
	}

	// Check for rate limiting
	if strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit") {
		return fmt.Errorf("rate limit exceeded: too many requests. Try again later: %w", err)
	}

	// Check for server errors
	if strings.Contains(errStr, "500") || strings.Contains(errStr, "502") || strings.Contains(errStr, "503") {
		return fmt.Errorf("server error: Gemini API is experiencing issues. Try again later: %w", err)
	}

	return err
}

// Per-model caps for input reference images, per Gemini docs:
// - Gemini 2.5 Flash Image: works best with up to 3 input images
// - Gemini 3 Pro Image Preview: up to 6 objects + 5 characters = 11 total
// - Gemini 3.1 Flash Image Preview: up to 10 objects + 4 characters = 14 total
// The API doesn't distinguish object vs character classes in the request payload,
// so gimage enforces a single combined total.
const (
	geminiFlash25MaxRefImages = 3
	geminiPro3MaxRefImages    = 11
	geminiFlash31MaxRefImages = 14
)

// maxInputImageBytes caps each reference image at 20 MB to fail fast before
// reading + base64-encoding a huge file. Gemini's documented inline_data limits
// are ambiguous; 20 MB is generous for legitimate references and catches
// mis-targeted paths (logs, archives) before they OOM the process.
const maxInputImageBytes = 20 * 1024 * 1024

// maxInputImagesForModel returns the cap on reference input images for a model.
// Unknown models default to the most conservative cap.
func maxInputImagesForModel(modelName string) int {
	switch {
	case strings.Contains(modelName, "gemini-3.1-flash-image"):
		return geminiFlash31MaxRefImages
	case strings.Contains(modelName, "gemini-3-pro-image"):
		return geminiPro3MaxRefImages
	case strings.Contains(modelName, "gemini-2.5-flash-image"):
		return geminiFlash25MaxRefImages
	default:
		return geminiFlash25MaxRefImages
	}
}

// readInputImageAsInlineData reads a local image file and returns a geminiPart
// wrapping its base64-encoded content. MIME type is detected from the file
// header via http.DetectContentType; only image/png, image/jpeg, and image/webp
// are accepted (these are the formats Gemini documents for image input).
// The file is rejected before any I/O if its size exceeds maxInputImageBytes.
func readInputImageAsInlineData(path string) (geminiPart, error) {
	info, err := os.Stat(path)
	if err != nil {
		return geminiPart{}, err
	}
	if info.Size() > maxInputImageBytes {
		return geminiPart{}, fmt.Errorf("input image %s exceeds %d-byte limit (%d bytes)", path, maxInputImageBytes, info.Size())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return geminiPart{}, err
	}
	if len(data) == 0 {
		return geminiPart{}, fmt.Errorf("input image is empty")
	}
	mimeType := http.DetectContentType(data)
	// http.DetectContentType returns "image/jpeg" for jpg, "image/png" for png, "image/webp" for webp.
	switch mimeType {
	case "image/png", "image/jpeg", "image/webp":
		// ok
	default:
		return geminiPart{}, fmt.Errorf("unsupported MIME type %q (expected image/png, image/jpeg, or image/webp)", mimeType)
	}
	return geminiPart{
		InlineData: &geminiInlineData{
			MimeType: mimeType,
			Data:     base64.StdEncoding.EncodeToString(data),
		},
	}, nil
}
