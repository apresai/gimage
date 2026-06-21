package generate

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/apresai/gimage/internal/observability"
	"github.com/apresai/gimage/pkg/models"
	"github.com/sony/gobreaker"
	"google.golang.org/genai"
)

// VertexUnifiedClient uses the unified google.golang.org/genai SDK for Vertex AI
// This replaces the deprecated cloud.google.com/go/vertexai/genai SDK
type VertexUnifiedClient struct {
	client         *genai.Client
	project        string
	location       string
	log            *observability.VerboseLogger
	circuitBreaker *gobreaker.CircuitBreaker
}

// NewVertexUnifiedClient creates a new Vertex AI client using the unified SDK
// Authentication uses Application Default Credentials (ADC):
// - GOOGLE_APPLICATION_CREDENTIALS environment variable (service account JSON)
// - gcloud CLI auth (gcloud auth application-default login)
// - Workload Identity on GCP (GKE, Cloud Run, etc.)
func NewVertexUnifiedClient(ctx context.Context, project, location string) (*VertexUnifiedClient, error) {
	log := observability.NewVerboseLogger(observability.ComponentVertex)

	// Get project from parameter or environment
	if project == "" {
		project = os.Getenv("VERTEX_PROJECT")
		if project == "" {
			return nil, fmt.Errorf("project ID required: set VERTEX_PROJECT environment variable or use --project flag")
		}
	}

	// Default location
	if location == "" {
		location = os.Getenv("VERTEX_LOCATION")
		if location == "" {
			location = "us-central1"
		}
	}

	log.Debug("Creating Vertex AI client with unified SDK")
	log.Debug("Project: %s, Location: %s", project, location)

	// Check for service account credentials (optional - ADC handles other methods)
	credsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credsPath != "" {
		log.Debug("Using credentials from: %s", credsPath)
		if _, err := os.Stat(credsPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("credentials file not found: %s", credsPath)
		}
	} else {
		log.Debug("Using Application Default Credentials (ADC)")
	}

	// Create unified SDK client with Vertex AI backend
	// The unified SDK automatically handles ADC, service accounts, and workload identity
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Vertex AI client: %w\nHint: Check that credentials are configured and have 'Vertex AI User' role", err)
	}

	return &VertexUnifiedClient{
		client:         client,
		project:        project,
		location:       location,
		log:            log,
		circuitBreaker: newCircuitBreaker("VertexUnified"),
	}, nil
}

func buildVertexGeminiGenerateContentRequest(modelName, prompt string, options models.GenerateOptions) ([]*genai.Content, *genai.GenerateContentConfig, int, int, error) {
	width, height := ParseSizeString(options.Size)

	parts := []*genai.Part{{Text: prompt}}
	if len(options.InputImages) > 0 {
		maxImages := maxInputImagesForModel(modelName)
		if len(options.InputImages) > maxImages {
			return nil, nil, 0, 0, fmt.Errorf("model %s supports at most %d input images, got %d", modelName, maxImages, len(options.InputImages))
		}
		for _, imgPath := range options.InputImages {
			data, mimeType, err := readInputImageData(imgPath)
			if err != nil {
				return nil, nil, 0, 0, fmt.Errorf("read input image %s: %w", imgPath, err)
			}
			parts = append(parts, &genai.Part{
				InlineData: &genai.Blob{
					Data:     data,
					MIMEType: mimeType,
				},
			})
		}
	}

	config := &genai.GenerateContentConfig{
		ResponseModalities: []string{"TEXT", "IMAGE"},
	}

	if options.NumberOfImages > 1 {
		count := options.NumberOfImages
		if count > 4 {
			count = 4
		}
		config.CandidateCount = int32(count)
	}

	imageConfig := &genai.ImageConfig{}
	if options.AspectRatio != "" {
		imageConfig.AspectRatio = options.AspectRatio
	} else if options.Size != "" && width > 0 && height > 0 {
		imageConfig.AspectRatio = InferAspectRatio(width, height, nil)
	} else if options.ImageSize != "" {
		imageConfig.AspectRatio = "1:1"
	}
	if options.ImageSize != "" {
		imageConfig.ImageSize = strings.ToUpper(options.ImageSize)
	}
	if options.OutputFormat != "" {
		switch strings.ToLower(options.OutputFormat) {
		case "png":
			imageConfig.OutputMIMEType = "image/png"
		case "jpeg", "jpg":
			imageConfig.OutputMIMEType = "image/jpeg"
		case "webp":
			imageConfig.OutputMIMEType = "image/webp"
		}
	}
	if imageConfig.AspectRatio != "" || imageConfig.ImageSize != "" || imageConfig.OutputMIMEType != "" {
		config.ImageConfig = imageConfig
	}

	if options.Seed != 0 {
		config.Seed = genai.Ptr[int32](int32(options.Seed))
	}
	if level := strings.ToLower(options.ThinkingLevel); level != "" && level != "off" {
		config.ThinkingConfig = &genai.ThinkingConfig{ThinkingLevel: toGenAIThinkingLevel(level)}
	}
	if options.WebSearchGrounding && isGeminiAdvanced(modelName) {
		config.Tools = []*genai.Tool{{GoogleSearch: &genai.GoogleSearch{}}}
	}

	return []*genai.Content{{Parts: parts}}, config, width, height, nil
}

func toGenAIThinkingLevel(level string) genai.ThinkingLevel {
	switch strings.ToLower(level) {
	case "minimal":
		return genai.ThinkingLevelMinimal
	case "low":
		return genai.ThinkingLevelLow
	case "medium":
		return genai.ThinkingLevelMedium
	case "high":
		return genai.ThinkingLevelHigh
	default:
		return genai.ThinkingLevel(strings.ToUpper(level))
	}
}

// GenerateImage generates an image using Vertex AI unified SDK
func (c *VertexUnifiedClient) GenerateImage(ctx context.Context, prompt string, options models.GenerateOptions) ([]*models.GeneratedImage, error) {
	// Validate prompt
	if err := ValidatePrompt(prompt); err != nil {
		return nil, err
	}

	// Enhance prompt for better results, including style
	enhancedPrompt := buildPromptWithOptions(EnhancePrompt(prompt), options)

	// Use custom model if provided, otherwise default to Gemini 3.1 Flash Image
	modelName := options.Model
	if modelName == "" {
		modelName = "gemini-3.1-flash-image"
	}

	if strings.Contains(modelName, "gemini") {
		contents, config, width, height, err := buildVertexGeminiGenerateContentRequest(modelName, enhancedPrompt, options)
		if err != nil {
			return nil, err
		}

		c.log.Debug("Sending request to Vertex AI (GenerateContent)...")
		result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
			return c.client.Models.GenerateContent(ctx, modelName, contents, config)
		})
		if err != nil {
			// Check if circuit breaker is open
			if isCircuitBreakerError(err) {
				c.log.Debug("Circuit breaker is open, failing fast")
				return nil, fmt.Errorf("API circuit breaker is open (too many failures): %w", err)
			}
			c.log.Debug("Generation failed: %v", err)
			return nil, fmt.Errorf("failed to generate image: %w", err)
		}

		resp := result.(*genai.GenerateContentResponse)
		if len(resp.Candidates) == 0 {
			c.log.Debug("No candidates in response")
			return nil, fmt.Errorf("no image generated from prompt")
		}

		c.log.Debug("Got %d candidates", len(resp.Candidates))

		var generatedImages []*models.GeneratedImage
		for i, candidate := range resp.Candidates {
			if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
				c.log.Debug("Candidate %d has no parts, skipping", i)
				continue
			}

			var candidateData []byte
			var format string = "png"
			foundPart := false

			for _, part := range candidate.Content.Parts {
				if part.InlineData != nil && len(part.InlineData.Data) > 0 {
					candidateData = part.InlineData.Data
					format = extractFormatFromMimeType(part.InlineData.MIMEType)
					foundPart = true
					break
				}
			}

			if !foundPart {
				c.log.Debug("No image data found in candidate %d, skipping", i)
				continue
			}

			finalWidth, finalHeight := width, height
			if isGeminiAdvanced(modelName) && options.ImageSize != "" {
				finalWidth, finalHeight = 0, 0
			}

			generatedImages = append(generatedImages, &models.GeneratedImage{
				Data:   candidateData,
				Format: format,
				Width:  finalWidth,
				Height: finalHeight,
				Metadata: map[string]string{
					"model":       modelName,
					"prompt":      prompt,
					"style":       options.Style,
					"api":         "vertex-unified-content",
					"imageSize":   options.ImageSize,
					"project":     c.project,
					"location":    c.location,
					"generated":   time.Now().UTC().Format(time.RFC3339),
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

	c.log.Debug("Generating image with model: %s", modelName)
	c.log.Debug("Prompt: %s", enhancedPrompt)

	// Build the generation config
	numberOfImages := 1
	if options.NumberOfImages > 0 && options.NumberOfImages <= 4 {
		numberOfImages = options.NumberOfImages
	}

	config := &genai.GenerateImagesConfig{
		NegativePrompt: options.NegativePrompt,
		NumberOfImages: int32(numberOfImages),
	}

	// Determine aspect ratio: explicit flag > inferred from Size
	if options.AspectRatio != "" {
		config.AspectRatio = options.AspectRatio
		c.log.Debug("Using explicit aspect ratio: %s", options.AspectRatio)
	} else if options.Size != "" {
		w, h := ParseSizeString(options.Size)
		if w > 0 && h > 0 {
			config.AspectRatio = InferAspectRatio(w, h, nil)
			c.log.Debug("Inferred aspect ratio from size %dx%d: %s", w, h, config.AspectRatio)
		}
	}

	// Set output format if supported
	if options.OutputFormat != "" {
		switch options.OutputFormat {
		case "png":
			config.OutputMIMEType = "image/png"
		case "jpeg", "jpg":
			config.OutputMIMEType = "image/jpeg"
		case "webp":
			config.OutputMIMEType = "image/webp"
		}
	}

	c.log.Debug("Sending request to Vertex AI...")

	// Generate image through circuit breaker
	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.client.Models.GenerateImages(ctx, modelName, enhancedPrompt, config)
	})

	if err != nil {
		// Check if circuit breaker is open
		if isCircuitBreakerError(err) {
			c.log.Debug("Circuit breaker is open, failing fast")
			return nil, fmt.Errorf("API circuit breaker is open (too many failures): %w", err)
		}
		c.log.Debug("Generation failed: %v", err)
		return nil, fmt.Errorf("failed to generate image: %w\nHint: Check that billing is enabled and you have Vertex AI User role", err)
	}

	resp := result.(*genai.GenerateImagesResponse)

	// Check if we got images
	if len(resp.GeneratedImages) == 0 {
		c.log.Debug("No images in response")
		return nil, fmt.Errorf("no image generated from prompt")
	}

	c.log.Debug("Got %d generated images", len(resp.GeneratedImages))

	// Parse dimensions from options
	width, height := ParseSizeString(options.Size)

	var generatedImages []*models.GeneratedImage
	for i, generatedImg := range resp.GeneratedImages {
		var imageData []byte
		var format string = "png"

		if generatedImg.Image != nil && generatedImg.Image.ImageBytes != nil {
			imageData = generatedImg.Image.ImageBytes
			format = extractFormatFromMimeType(generatedImg.Image.MIMEType)
			c.log.Debug("Image data for image %d: %d bytes, format=%s", i, len(imageData), format)
		} else {
			c.log.Debug("No image data found for image %d in response, skipping", i)
			continue
		}

		generatedImages = append(generatedImages, &models.GeneratedImage{
			Data:   imageData,
			Format: format,
			Width:  width,
			Height: height,
			Metadata: map[string]string{
				"model":       modelName,
				"prompt":      prompt,
				"style":       options.Style,
				"api":         "vertex-unified",
				"project":     c.project,
				"location":    c.location,
				"generated":   time.Now().UTC().Format(time.RFC3339),
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

// ValidateCredentials checks if the Vertex AI credentials are valid
func (c *VertexUnifiedClient) ValidateCredentials() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to list models as a lightweight validation
	_, err := c.client.Models.List(ctx, nil)
	if err != nil {
		return fmt.Errorf("credential validation failed: %w", err)
	}

	return nil
}

// Close closes the Vertex AI client
func (c *VertexUnifiedClient) Close() error {
	// The unified genai.Client doesn't have a Close method
	// Resources will be cleaned up by the garbage collector
	return nil
}
