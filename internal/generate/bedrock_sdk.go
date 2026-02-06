package generate

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/apresai/gimage/internal/observability"
	"github.com/apresai/gimage/pkg/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/sony/gobreaker"
)

// BedrockSDKClient uses AWS Bedrock Runtime for image generation
type BedrockSDKClient struct {
	client         *bedrockruntime.Client
	region         string
	log            *observability.VerboseLogger
	circuitBreaker *gobreaker.CircuitBreaker
}

// NovaCanvasRequest represents the Nova Canvas API request format
type NovaCanvasRequest struct {
	TaskType              string                      `json:"taskType"`
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

// BuildNovaCanvasRequest builds a Nova Canvas request from prompt and options.
// Shared by both BedrockSDKClient and BedrockRESTClient.
func BuildNovaCanvasRequest(prompt string, options models.GenerateOptions, log *observability.VerboseLogger) (*NovaCanvasRequest, error) {
	if prompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	// Parse and normalize dimensions (Nova Canvas supports 512-2048, multiples of 64)
	requestedWidth, requestedHeight := ParseSizeString(options.Size)
	apiWidth, apiHeight := NormalizeDimensions(requestedWidth, requestedHeight, BedrockConstraints)
	log.Debug("Normalized dimensions: %dx%d -> %dx%d",
		requestedWidth, requestedHeight, apiWidth, apiHeight)

	// Validate seed (Nova Canvas supports 0-858993459)
	if options.Seed < 0 || options.Seed > 858993459 {
		return nil, fmt.Errorf("invalid seed: %d (must be 0-858993459)", options.Seed)
	}

	// Map style keywords to Nova Canvas quality levels
	quality := "standard"
	if options.Style != "" {
		lowerStyle := strings.ToLower(options.Style)
		if lowerStyle == "premium" || lowerStyle == "high" || lowerStyle == "ultra" || lowerStyle == "photorealistic" {
			quality = "premium"
		}
	}

	// CFG scale (default 7.0, clamped to 1.0-10.0)
	cfgScale := 7.0
	if options.CfgScale > 0 {
		cfgScale = options.CfgScale
		if cfgScale < 1.0 {
			cfgScale = 1.0
		} else if cfgScale > 10.0 {
			cfgScale = 10.0
		}
	}

	// Number of images (default 1, max 5)
	numberOfImages := 1
	if options.NumberOfImages > 0 {
		numberOfImages = options.NumberOfImages
		if numberOfImages > 5 {
			numberOfImages = 5
		}
	}

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

	if options.NegativePrompt != "" {
		request.TextToImageParams.NegativeText = options.NegativePrompt
	}

	if options.Seed != 0 {
		request.ImageGenerationConfig.Seed = int(options.Seed)
	}

	return request, nil
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

	log := observability.NewVerboseLogger(observability.ComponentBedrock)

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

	return &BedrockSDKClient{
		client:         client,
		region:         region,
		log:            log,
		circuitBreaker: newCircuitBreaker("BedrockAPI"),
	}, nil
}

// GenerateImage generates an image using AWS Bedrock Nova Canvas
func (c *BedrockSDKClient) GenerateImage(ctx context.Context, prompt string, options models.GenerateOptions) ([]*models.GeneratedImage, error) {
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

	c.log.Debug("Generating image with AWS Bedrock Nova Canvas")
	c.log.Debug("Prompt: %s, Model: %s, Size: %s, Quality: %s",
		prompt, options.Model, options.Size, request.ImageGenerationConfig.Quality)

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

		c.log.Debug("Images generated successfully: model=%s, count=%d, duration=%s",
			modelID, len(generatedImages), time.Since(startTime))

		return generatedImages, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]*models.GeneratedImage), nil
}

// buildRequest builds the Nova Canvas request from options (delegates to shared function)
func (c *BedrockSDKClient) buildRequest(prompt string, options models.GenerateOptions) (*NovaCanvasRequest, error) {
	return BuildNovaCanvasRequest(prompt, options, c.log)
}

// handleError provides user-friendly error messages for AWS errors
func (c *BedrockSDKClient) handleError(err error) error {
	if err == nil {
		return nil
	}

	c.log.Error("AWS Bedrock error: %v", err)

	errMsg := err.Error()

	// Common AWS error patterns
	switch {
	case strings.Contains(errMsg, "ValidationException"):
		return fmt.Errorf("invalid request parameters: %w\n\nTip: Check image dimensions (512-2048, multiple of 64) and quality (standard/premium)", err)

	case strings.Contains(errMsg, "AccessDeniedException"):
		return fmt.Errorf("access denied: %w\n\nTip: Ensure you have enabled model access in AWS Bedrock console and have bedrock:InvokeModel permission", err)

	case strings.Contains(errMsg, "ThrottlingException"):
		return fmt.Errorf("rate limit exceeded: %w\n\nTip: Bedrock rate limits: 10 requests/second. Wait and retry.", err)

	case strings.Contains(errMsg, "ModelNotReadyException"):
		return fmt.Errorf("model not available: %w\n\nTip: Nova Canvas may not be available in region %s. Try us-east-1.", err, c.region)

	case strings.Contains(errMsg, "ServiceQuotaExceededException"):
		return fmt.Errorf("service quota exceeded: %w\n\nTip: Request a quota increase in AWS Service Quotas console", err)

	case strings.Contains(errMsg, "NoCredentialProviders"):
		return fmt.Errorf("no AWS credentials found: %w\n\nTip: Run 'gimage auth bedrock' or set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY", err)

	default:
		return fmt.Errorf("AWS Bedrock error: %w", err)
	}
}

// Close cleans up resources (no-op for SDK client, implements interface)
func (c *BedrockSDKClient) Close() error {
	return nil
}
