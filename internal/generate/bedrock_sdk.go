package generate

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/apresai/gimage/pkg/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/rs/zerolog"
	"github.com/sony/gobreaker"
	"github.com/spf13/viper"
)

// BedrockSDKClient uses AWS Bedrock Runtime for image generation
type BedrockSDKClient struct {
	client         *bedrockruntime.Client
	region         string
	verbose        bool
	logger         zerolog.Logger
	circuitBreaker *gobreaker.CircuitBreaker
}

// NovaCanvasRequest represents the Nova Canvas API request format
type NovaCanvasRequest struct {
	TaskType              string                     `json:"taskType"`
	TextToImageParams     NovaCanvasTextToImageParams `json:"textToImageParams"`
	ImageGenerationConfig NovaCanvasImageConfig       `json:"imageGenerationConfig"`
}

type NovaCanvasTextToImageParams struct {
	Text         string `json:"text"`
	NegativeText string `json:"negativeText,omitempty"`
}

type NovaCanvasImageConfig struct {
	NumberOfImages int     `json:"numberOfImages"`
	Quality        string  `json:"quality"` // "standard" or "premium"
	Height         int     `json:"height"`
	Width          int     `json:"width"`
	CfgScale       float64 `json:"cfgScale,omitempty"`
	Seed           int     `json:"seed,omitempty"`
}

// NovaCanvasResponse represents the Nova Canvas API response format
type NovaCanvasResponse struct {
	Images []string `json:"images"` // Base64-encoded images
	Error  string   `json:"error,omitempty"`
}

// NewBedrockSDKClient creates a new AWS Bedrock SDK client
func NewBedrockSDKClient(ctx context.Context, region string) (*BedrockSDKClient, error) {
	// Default region if not provided
	if region == "" {
		region = os.Getenv("AWS_REGION")
		if region == "" {
			region = "us-east-1"
		}
	}

	// Load AWS SDK configuration
	// This automatically handles:
	// - Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
	// - Shared credentials file (~/.aws/credentials)
	// - IAM role credentials (EC2, ECS, Lambda)
	// - AWS SSO profiles
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Bedrock Runtime client
	client := bedrockruntime.NewFromConfig(cfg)

	// Check verbose mode
	verbose := viper.GetBool("verbose") ||
		os.Getenv("GIMAGE_VERBOSE") == "true" ||
		os.Getenv("VERBOSE") == "true"

	// Setup logger
	logger := zerolog.New(os.Stderr).With().
		Timestamp().
		Str("component", "bedrock_sdk").
		Logger()

	if !verbose {
		logger = logger.Level(zerolog.WarnLevel)
	}

	return &BedrockSDKClient{
		client:         client,
		region:         region,
		verbose:        verbose,
		logger:         logger,
		circuitBreaker: newCircuitBreaker("BedrockAPI"),
	}, nil
}

// GenerateImage generates an image using AWS Bedrock Nova Canvas
func (c *BedrockSDKClient) GenerateImage(ctx context.Context, prompt string, options models.GenerateOptions) (*models.GeneratedImage, error) {
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
			Msg("Generating image with AWS Bedrock Nova Canvas")
	}

	// Determine model ID
	modelID := "amazon.nova-canvas-v1:0"
	if options.Model != "" {
		modelID = options.Model
	}

	// Invoke model via circuit breaker
	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		// Invoke Bedrock model
		output, err := c.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
			ModelId:     aws.String(modelID),
			ContentType: aws.String("application/json"),
			Accept:      aws.String("application/json"),
			Body:        requestBody,
		})
		if err != nil {
			return nil, c.handleError(err)
		}

		// Parse response
		var novaResponse NovaCanvasResponse
		if err := json.Unmarshal(output.Body, &novaResponse); err != nil {
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

		// Decode base64 image
		imageData, err := base64.StdEncoding.DecodeString(novaResponse.Images[0])
		if err != nil {
			return nil, fmt.Errorf("failed to decode image: %w", err)
		}

		// Parse dimensions
		width, height := parseDimensions(options.Size)

		// Build response
		generatedImage := &models.GeneratedImage{
			Data:   imageData,
			Format: "png",
			Width:  width,
			Height: height,
			Metadata: map[string]string{
				"model":   modelID,
				"prompt":  prompt,
				"size":    options.Size,
				"seed":    fmt.Sprintf("%d", request.ImageGenerationConfig.Seed),
				"quality": request.ImageGenerationConfig.Quality,
			},
		}

		if c.verbose {
			c.logger.Info().
				Str("model", modelID).
				Dur("duration", time.Since(startTime)).
				Int("image_size_kb", len(imageData)/1024).
				Msg("Image generated successfully")
		}

		return generatedImage, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(*models.GeneratedImage), nil
}

// buildRequest builds the Nova Canvas request from options
func (c *BedrockSDKClient) buildRequest(prompt string, options models.GenerateOptions) (*NovaCanvasRequest, error) {
	// Validate prompt
	if prompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	// Parse dimensions
	width, height := parseDimensions(options.Size)

	// Validate dimensions (Nova Canvas supports 512-2048, multiples of 64)
	if width < 512 || width > 2048 || width%64 != 0 {
		return nil, fmt.Errorf("invalid width: %d (must be 512-2048, multiple of 64)", width)
	}
	if height < 512 || height > 2048 || height%64 != 0 {
		return nil, fmt.Errorf("invalid height: %d (must be 512-2048, multiple of 64)", height)
	}

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
			Height:         height,
			Width:          width,
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

// handleError provides user-friendly error messages for AWS errors
func (c *BedrockSDKClient) handleError(err error) error {
	if err == nil {
		return nil
	}

	// Log raw error in verbose mode
	if c.verbose {
		c.logger.Error().Err(err).Msg("AWS Bedrock error")
	}

	errMsg := err.Error()

	// Common AWS error patterns
	switch {
	case contains(errMsg, "ValidationException"):
		return fmt.Errorf("invalid request parameters: %w\n\nTip: Check image dimensions (512-2048, multiple of 64) and quality (standard/premium)", err)

	case contains(errMsg, "AccessDeniedException"):
		return fmt.Errorf("access denied: %w\n\nTip: Ensure you have enabled model access in AWS Bedrock console and have bedrock:InvokeModel permission", err)

	case contains(errMsg, "ThrottlingException"):
		return fmt.Errorf("rate limit exceeded: %w\n\nTip: Bedrock rate limits: 10 requests/second. Wait and retry.", err)

	case contains(errMsg, "ModelNotReadyException"):
		return fmt.Errorf("model not available: %w\n\nTip: Nova Canvas may not be available in region %s. Try us-east-1.", err, c.region)

	case contains(errMsg, "ServiceQuotaExceededException"):
		return fmt.Errorf("service quota exceeded: %w\n\nTip: Request a quota increase in AWS Service Quotas console", err)

	case contains(errMsg, "NoCredentialProviders"):
		return fmt.Errorf("no AWS credentials found: %w\n\nTip: Run 'gimage auth bedrock' or set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY", err)

	default:
		return fmt.Errorf("AWS Bedrock error: %w", err)
	}
}

// Close cleans up resources (no-op for SDK client, implements interface)
func (c *BedrockSDKClient) Close() error {
	return nil
}
