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

	"github.com/apresai/gimage/pkg/models"
	"github.com/rs/zerolog"
	"github.com/sony/gobreaker"
	"github.com/spf13/viper"
)

// BedrockRESTClient uses AWS Bedrock REST API with bearer token for image generation
type BedrockRESTClient struct {
	apiKey         string
	region         string
	baseURL        string
	httpClient     *http.Client
	verbose        bool
	logger         zerolog.Logger
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

	// Check verbose mode
	verbose := viper.GetBool("verbose") ||
		os.Getenv("GIMAGE_VERBOSE") == "true" ||
		os.Getenv("VERBOSE") == "true"

	// Setup logger
	logger := zerolog.New(os.Stderr).With().
		Timestamp().
		Str("component", "bedrock_rest").
		Logger()

	if !verbose {
		logger = logger.Level(zerolog.WarnLevel)
	}

	return &BedrockRESTClient{
		apiKey:  apiKey,
		region:  region,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
		verbose:        verbose,
		logger:         logger,
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

	if c.verbose {
		c.logger.Debug().
			Str("prompt", prompt).
			Str("model", options.Model).
			Str("size", options.Size).
			Str("quality", request.ImageGenerationConfig.Quality).
			Int("seed", request.ImageGenerationConfig.Seed).
			Msg("Generating image with AWS Bedrock Nova Canvas via REST API")
	}

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
				c.logger.Debug().Int("index", i).Err(err).Msg("Failed to decode image, skipping")
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

		if c.verbose {
			c.logger.Info().
				Str("model", modelID).
				Dur("duration", time.Since(startTime)).
				Int("count", len(generatedImages)).
				Msg("Images generated successfully via REST API")
		}

		response = generatedImages
		return response, nil
	})

	if err != nil {
		return nil, err
	}

	return response, nil
}

// buildRequest builds the Nova Canvas request from options
func (c *BedrockRESTClient) buildRequest(prompt string, options models.GenerateOptions) (*NovaCanvasRequest, error) {
	// Validate prompt
	if prompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	// Parse and normalize dimensions (Nova Canvas supports 512-2048, multiples of 64)
	requestedWidth, requestedHeight := ParseSizeString(options.Size)
	apiWidth, apiHeight := NormalizeDimensions(requestedWidth, requestedHeight, BedrockConstraints)
	c.logger.Debug().
		Int("requested_w", requestedWidth).Int("requested_h", requestedHeight).
		Int("api_w", apiWidth).Int("api_h", apiHeight).
		Msg("Normalized dimensions for Bedrock")

	// Validate seed if provided (Nova Canvas supports 0-858993459)
	if options.Seed < 0 || options.Seed > 858993459 {
		return nil, fmt.Errorf("invalid seed: %d (must be 0-858993459)", options.Seed)
	}

	// Determine quality from style (Nova Canvas uses standard/premium, not style)
	// Map common quality/style keywords to Nova Canvas quality levels
	quality := "standard"
	if options.Style != "" {
		lowerStyle := strings.ToLower(options.Style)
		if lowerStyle == "premium" || lowerStyle == "high" || lowerStyle == "ultra" || lowerStyle == "photorealistic" {
			quality = "premium"
		}
	}

	// Determine CFG scale (use option or default to 7.0)
	cfgScale := 7.0
	if options.CfgScale > 0 {
		// Clamp to valid range (1.0-10.0)
		cfgScale = options.CfgScale
		if cfgScale < 1.0 {
			cfgScale = 1.0
		} else if cfgScale > 10.0 {
			cfgScale = 10.0
		}
	}

	// Determine number of images (use option or default to 1)
	numberOfImages := 1
	if options.NumberOfImages > 0 {
		numberOfImages = options.NumberOfImages
		if numberOfImages > 5 {
			numberOfImages = 5 // Nova Canvas max is 5
		}
	}

	// Build request
	request := &NovaCanvasRequest{
		TaskType: "TEXT_IMAGE",
		TextToImageParams: NovaCanvasTextToImageParams{
			Text: prompt,
		},
		ImageGenerationConfig: NovaCanvasImageConfig{
			NumberOfImages: numberOfImages,
			Quality:        quality,
			Height:         apiHeight,
			Width:          apiWidth,
			CfgScale:       cfgScale,
		},
	}

	// Add negative prompt if provided
	if options.NegativePrompt != "" {
		request.TextToImageParams.NegativeText = options.NegativePrompt
	}

	// Add seed if provided
	if options.Seed != 0 {
		request.ImageGenerationConfig.Seed = int(options.Seed)
	}

	return request, nil
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

	// Log raw error in verbose mode
	if c.verbose {
		c.logger.Error().
			Int("status_code", statusCode).
			Str("error_message", errorMsg).
			Msg("AWS Bedrock REST API error")
	}

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
