package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/apresai/gimage/internal/config"
	"github.com/apresai/gimage/internal/generate"
	"github.com/apresai/gimage/pkg/models"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
)

// Helper functions for output
func printVerbose(format string, args ...interface{}) {
	if viper.GetBool("verbose") {
		fmt.Fprintf(os.Stderr, "[VERBOSE] "+format+"\n", args...)
	}
}

func printInfo(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func printSuccess(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "✓ "+format+"\n", args...)
}

func printWarning(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "⚠ "+format+"\n", args...)
}

func greenYes() string {
	return "✅"
}

func redNo() string {
	return "❌"
}

// padRight pads a string to the right, accounting for ANSI color codes
// which don't contribute to visible width
func padRight(s string, width int) string {
	// Count ANSI escape sequences (they don't contribute to visible width)
	visibleLen := 0
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
		} else if inEscape && r == 'm' {
			inEscape = false
		} else if !inEscape {
			visibleLen++
		}
	}

	padding := width - visibleLen
	if padding <= 0 {
		return s
	}

	return s + strings.Repeat(" ", padding)
}

func formatImageSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// getGemini3ProCost returns the cost for Gemini 3 Pro based on image size
// Pricing: $0.134 for 1K/2K, $0.24 for 4K
func getGemini3ProCost(imageSize string) float64 {
	if strings.ToUpper(imageSize) == "4K" {
		return 0.24
	}
	return 0.134 // Default for 1K/2K or unspecified
}

// getNovaCanvasCost returns the cost for Nova Canvas based on image dimensions
// Pricing: $0.04 for ≤1024x1024, $0.08 for >1024x1024
func getNovaCanvasCost(size string) float64 {
	width, height := parseDimensionsForCost(size)
	if width > 1024 || height > 1024 {
		return 0.08
	}
	return 0.04
}

// parseDimensionsForCost parses "WIDTHxHEIGHT" string for cost calculation
func parseDimensionsForCost(size string) (int, int) {
	if size == "" {
		return 1024, 1024
	}
	parts := strings.Split(size, "x")
	if len(parts) != 2 {
		return 1024, 1024
	}
	var width, height int
	fmt.Sscanf(parts[0], "%d", &width)
	fmt.Sscanf(parts[1], "%d", &height)
	if width <= 0 {
		width = 1024
	}
	if height <= 0 {
		height = 1024
	}
	return width, height
}

var generateCmd = &cobra.Command{
	Use:   "generate [prompt]",
	Short: "Generate an image from a text prompt using AI",
	Long: `Generate an image from a text prompt using Google Gemini, Vertex AI, or AWS Bedrock.

The prompt should describe the image you want to generate. You can optionally
specify style, size, negative prompts, and other parameters.

You can provide the prompt as a positional argument (recommended for quick use)
or use the --prompt flag for explicit clarity.

Examples:
  # List all available models
  gimage generate --list-models

  # Generate with default settings - positional prompt (most common)
  gimage generate "a sunset over mountains"

  # Or use --prompt flag explicitly
  gimage generate --prompt "a sunset over mountains"

  # Generate with specific model (auto-detects API)
  gimage generate "futuristic city" --model imagen-4

  # Generate with AWS Bedrock Nova Canvas
  gimage generate "futuristic city" --model nova-canvas

  # Generate with specific style and size
  gimage generate "abstract art" --size 1024x1024 --style photorealistic

  # Override API selection (rarely needed)
  gimage generate "abstract art" --api vertex --model imagen-4

  # Use negative prompts and seed for reproducibility
  gimage generate "forest scene" --negative "people, buildings" --seed 12345`,
	Args: cobra.ArbitraryArgs,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Check if --list-models flag is set
		listModels, _ := cmd.Flags().GetBool("list-models")
		if listModels {
			return nil // Skip validation for list-models
		}

		// Check if --list-providers flag is set
		listProviders, _ := cmd.Flags().GetBool("list-providers")
		if listProviders {
			return nil // Skip validation for list-providers
		}

		// Check if --prompt-howto flag is set
		promptHowto, _ := cmd.Flags().GetBool("prompt-howto")
		if promptHowto {
			return nil // Skip validation for prompt-howto
		}

		// Require either positional prompt or --prompt flag
		prompt, _ := cmd.Flags().GetString("prompt")
		if len(args) == 0 && prompt == "" {
			return fmt.Errorf("prompt is required (provide as argument or use --prompt flag)\nExamples:\n  gimage generate \"your prompt here\"\n  gimage generate --prompt \"your prompt here\"\n  gimage generate --list-models")
		}

		// Validate flags
		size, _ := cmd.Flags().GetString("size")
		if size != "" {
			parts := strings.Split(size, "x")
			if len(parts) != 2 {
				return fmt.Errorf("invalid size format: %s (expected format: WIDTHxHEIGHT, e.g., 1024x1024)", size)
			}
		}
		return nil
	},
	RunE: runGenerate,
}

func runGenerate(cmd *cobra.Command, args []string) error {
	// Get flags
	output, _ := cmd.Flags().GetString("output")
	providerID, _ := cmd.Flags().GetString("provider")
	api, _ := cmd.Flags().GetString("api")
	apiKey, _ := cmd.Flags().GetString("api-key")
	// project, _ := cmd.Flags().GetString("project") // TODO: Enable when Vertex is implemented
	// location, _ := cmd.Flags().GetString("location") // TODO: Enable when Vertex is implemented
	model, _ := cmd.Flags().GetString("model")

	// Resolve model aliases to exact names (e.g., "gemini" -> "gemini-2.5-flash-image")
	originalModel := model
	if model != "" {
		model = generate.ResolveModelName(model)
		if originalModel != model {
			printVerbose("Resolved model '%s' to '%s'", originalModel, model)
		}
	}

	size, _ := cmd.Flags().GetString("size")
	imageSize, _ := cmd.Flags().GetString("image-size")     // For Gemini 3 Pro: 1K, 2K, 4K
	aspectRatio, _ := cmd.Flags().GetString("aspect-ratio") // For Gemini 3 Pro: 1:1, 16:9, etc.
	style, _ := cmd.Flags().GetString("style")
	negative, _ := cmd.Flags().GetString("negative")
	seed, _ := cmd.Flags().GetInt64("seed")
	cfgScale, _ := cmd.Flags().GetFloat64("cfg-scale")      // For Bedrock Nova Canvas
	count, _ := cmd.Flags().GetInt("count")                  // Number of images to generate
	outputFormat, _ := cmd.Flags().GetString("output-format")
	listModels, _ := cmd.Flags().GetBool("list-models")
	listProviders, _ := cmd.Flags().GetBool("list-providers")

	// Handle --list-providers flag
	if listProviders {
		return printAvailableProviders()
	}

	// Handle --list-models flag (legacy)
	if listModels {
		return printAvailableModels()
	}

	// Handle --prompt-howto flag
	promptHowto, _ := cmd.Flags().GetBool("prompt-howto")
	if promptHowto {
		return printPromptHowto()
	}

	// Get prompt - prefer positional argument, fall back to --prompt flag
	var prompt string
	if len(args) > 0 {
		prompt = strings.Join(args, " ")
	} else {
		prompt, _ = cmd.Flags().GetString("prompt")
	}

	printVerbose("Generating image with prompt: %s", prompt)

	// Handle new provider system if --provider is specified
	if providerID != "" {
		return runGenerateWithProvider(cmd, prompt, providerID, output, size, style, negative, seed)
	}

	// Build generate options
	options := models.GenerateOptions{
		Model:          model,
		Size:           size,
		ImageSize:      imageSize,   // For Gemini 3 Pro: 1K, 2K, 4K
		AspectRatio:    aspectRatio, // For Gemini 3 Pro: 1:1, 16:9, etc.
		Style:          style,
		NegativePrompt: negative,
		Seed:           seed,
		CfgScale:       cfgScale,       // For Bedrock Nova Canvas (1.0-10.0)
		NumberOfImages: count,          // Number of images to generate (1-5)
		OutputFormat:   outputFormat,   // For all APIs: png, jpeg, webp
	}

	// Determine which API to use
	// Priority: 1. --api flag, 2. Auto-detect from model, 3. Auto-detect from available credentials
	selectedAPI := api
	if selectedAPI == "" {
		// Auto-detect API from model
		if model != "" {
			detectedAPI, err := generate.DetectAPIFromModel(model)
			if err != nil {
				return fmt.Errorf("invalid model: %w\nUse --list-models to see available models", err)
			}
			selectedAPI = detectedAPI
		} else {
			// Auto-detect from available credentials
			hasGemini := config.HasGeminiCredentials()
			hasVertex := config.HasVertexCredentials()
			hasBedrock := config.HasBedrockCredentials()
			hasGrok := config.HasGrokCredentials()

			// Count available credentials
			availableCount := 0
			if hasGemini {
				availableCount++
			}
			if hasVertex {
				availableCount++
			}
			if hasBedrock {
				availableCount++
			}
			if hasGrok {
				availableCount++
			}

			if availableCount == 0 {
				// No credentials found
				return fmt.Errorf("no API credentials found. Please set up credentials using:\n" +
					"  Gemini:  gimage auth gemini\n" +
					"  Vertex:  gimage auth vertex\n" +
					"  Bedrock: gimage auth bedrock\n" +
					"  Grok:    export GROK_API_KEY=your-key")
			} else if availableCount == 1 {
				// Only one API available
				if hasGemini {
					selectedAPI = "gemini"
					printVerbose("Auto-detected Gemini API (found credentials)")
				} else if hasVertex {
					selectedAPI = "vertex"
					printVerbose("Auto-detected Vertex API (found credentials)")
				} else if hasBedrock {
					selectedAPI = "bedrock"
					printVerbose("Auto-detected AWS Bedrock API (found credentials)")
				} else {
					selectedAPI = "grok"
					printVerbose("Auto-detected xAI Grok API (found credentials)")
				}
			} else {
				// Multiple APIs available - check default_api in config, or default to gemini
				cfg, err := config.LoadConfig()
				if err == nil && cfg.DefaultAPI != "" {
					selectedAPI = cfg.DefaultAPI
					printVerbose("Using default API from config: %s", selectedAPI)
				} else {
					selectedAPI = "gemini"
					printVerbose("Multiple APIs available, defaulting to Gemini")
				}
			}
		}
	}

	// Validate model is compatible with selected API
	if model != "" {
		if err := generate.ValidateModelForAPI(model, selectedAPI); err != nil {
			return err
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var generatedImage *models.GeneratedImage
	var err error

	// Generate based on API selection
	if selectedAPI == "gemini" {
		// Use Gemini API (REST implementation)
		key, err := config.GetGeminiAPIKey(apiKey)
		if err != nil {
			return fmt.Errorf("failed to get Gemini API key: %w\nHint: Set GEMINI_API_KEY environment variable or use --api-key flag", err)
		}

		modelName := model
		if modelName == "" {
			modelName = generate.DefaultModel
		}

		// Get provider info and announce selection
		registry := generate.GetProviderRegistry()
		provider, _ := registry.ResolveProvider(modelName)
		if provider != nil {
			printInfo("Using: %s (%s API)", provider.Name, provider.API)
			pricingDisplay := "Variable"
			if provider.Pricing.FreeTier {
				pricingDisplay = fmt.Sprintf("FREE (%s)", provider.Pricing.FreeTierLimit)
			} else if provider.Pricing.CostPerImage != nil {
				// Gemini 3 Pro has variable pricing based on image size
				if strings.Contains(provider.ModelID, "gemini-3") || strings.Contains(provider.ModelID, "pro-image-preview") {
					cost := getGemini3ProCost(imageSize)
					pricingDisplay = fmt.Sprintf("$%.4f/image (%s)", cost, imageSizeLabel(imageSize))
				} else {
					pricingDisplay = fmt.Sprintf("$%.4f/image", *provider.Pricing.CostPerImage)
				}
			}
			printInfo("Pricing: %s", pricingDisplay)

			// Warn if expensive (cost > $0.05)
			if provider.Pricing.CostPerImage != nil {
				cost := *provider.Pricing.CostPerImage
				// Use actual cost for Gemini 3 Pro
				if strings.Contains(provider.ModelID, "gemini-3") || strings.Contains(provider.ModelID, "pro-image-preview") {
					cost = getGemini3ProCost(imageSize)
				}
				if cost > 0.05 {
					fmt.Fprintf(os.Stderr, "⚠️  %s costs $%.4f/image\n", provider.Name, cost)
				}
			}
		}

		printInfo("Generating image...")

		// Use REST client instead of SDK client
		client, err := generate.NewGeminiRESTClient(key)
		if err != nil {
			return fmt.Errorf("failed to create Gemini client: %w", err)
		}
		defer client.Close()

		generatedImage, err = client.GenerateImage(ctx, prompt, options)
	} else if selectedAPI == "vertex" {
		// Use Vertex AI - check for Express Mode (API key) or Full Mode (service account)

		// Get project and location from flags or environment
		project, _ := cmd.Flags().GetString("project")
		if project == "" {
			project = os.Getenv("VERTEX_PROJECT")
		}
		location, _ := cmd.Flags().GetString("location")
		if location == "" {
			location = os.Getenv("VERTEX_LOCATION")
			if location == "" {
				location = "us-central1"
			}
		}

		// Check for Vertex API key (Express Mode)
		vertexAPIKey, err := config.GetVertexAPIKey(apiKey)
		if err != nil {
			return fmt.Errorf("failed to get Vertex API key: %w", err)
		}

		modelName := model
		if modelName == "" {
			modelName = "imagen-4.0-generate-001" // Default Imagen model
		}

		// Get provider info and announce selection
		registry := generate.GetProviderRegistry()
		provider, _ := registry.ResolveProvider(modelName)
		if provider != nil {
			printInfo("Using: %s (%s API)", provider.Name, provider.API)
			pricingDisplay := "Variable"
			if provider.Pricing.FreeTier {
				pricingDisplay = fmt.Sprintf("FREE (%s)", provider.Pricing.FreeTierLimit)
			} else if provider.Pricing.CostPerImage != nil {
				pricingDisplay = fmt.Sprintf("$%.4f/image", *provider.Pricing.CostPerImage)
			}
			printInfo("Pricing: %s", pricingDisplay)

			// Warn if expensive (cost > $0.05)
			if provider.Pricing.CostPerImage != nil && *provider.Pricing.CostPerImage > 0.05 {
				fmt.Fprintf(os.Stderr, "⚠️  %s costs $%.4f/image\n", provider.Name, *provider.Pricing.CostPerImage)
			}
		}

		printInfo("Generating image...")

		// Choose client based on authentication mode
		if vertexAPIKey != "" {
			// Express Mode - Use REST client with API key
			printVerbose("Using Vertex AI Express Mode (API key authentication)")

			// Load project and location from config if not provided
			if project == "" {
				cfg, err := config.LoadConfig()
				if err == nil && cfg.VertexProject != "" {
					project = cfg.VertexProject
				}
			}
			if location == "" {
				cfg, err := config.LoadConfig()
				if err == nil && cfg.VertexLocation != "" {
					location = cfg.VertexLocation
				} else {
					location = "us-central1"
				}
			}

			client, err := generate.NewVertexRESTClient(vertexAPIKey, project, location)
			if err != nil {
				return fmt.Errorf("failed to create Vertex AI REST client: %w", err)
			}
			defer client.Close()

			generatedImage, err = client.GenerateImage(ctx, prompt, options)
		} else {
			// Full Mode - Use unified SDK client with ADC/service account
			printVerbose("Using Vertex AI Full Mode (ADC/service account authentication)")

			client, err := generate.NewVertexUnifiedClient(ctx, project, location)
			if err != nil {
				return fmt.Errorf("failed to create Vertex AI client: %w", err)
			}
			defer client.Close()

			generatedImage, err = client.GenerateImage(ctx, prompt, options)
		}
	} else if selectedAPI == "bedrock" {
		// Use AWS Bedrock API - choose between REST (bearer token) or SDK (IAM/keys)
		region := config.GetAWSRegion("")
		printVerbose("Using AWS Bedrock region: %s", region)

		modelName := model
		if modelName == "" {
			modelName = generate.ModelNovaCanvas
		}

		printVerbose("Using model: %s", modelName)

		// Resolve provider alias and show info
		registry := generate.GetProviderRegistry()
		provider, err := registry.ResolveProvider(modelName)
		if err != nil {
			return fmt.Errorf("unknown provider/model: %s", modelName)
		}

		// Use the resolved full model ID
		resolvedModelID := provider.ModelID
		printVerbose("Resolved model ID: %s (Provider: %s)", resolvedModelID, provider.ID)

		printVerbose("Provider: %s", provider.Name)
		printVerbose("Model ID: %s", provider.ModelID)

		// Show pricing info
		if !provider.Pricing.FreeTier && provider.Pricing.CostPerImage != nil {
			cost := *provider.Pricing.CostPerImage
			// Nova Canvas has variable pricing based on size
			if strings.Contains(provider.ModelID, "nova-canvas") {
				cost = getNovaCanvasCost(size)
				w, h := parseDimensionsForCost(size)
				if w > 1024 || h > 1024 {
					printVerbose("Cost: $%.4f/image (>1024px)", cost)
				} else {
					printVerbose("Cost: $%.4f/image (≤1024px)", cost)
				}
			} else {
				printVerbose("Cost: $%.4f/image", cost)
			}

			// Warn if expensive (cost > $0.05)
			if cost > 0.05 {
				fmt.Fprintf(os.Stderr, "⚠️  %s costs $%.4f/image\n", provider.Name, cost)
			}
		}

		// Update options to use resolved model ID (not alias)
		bedrockOptions := options
		bedrockOptions.Model = resolvedModelID

		// Determine which authentication method to use
		// Priority: Bearer token (REST) > AWS SDK (keys/profile/IAM)
		cfg, _ := config.LoadConfig()
		bearerToken := os.Getenv("AWS_BEARER_TOKEN_BEDROCK")
		if bearerToken == "" && cfg != nil {
			bearerToken = cfg.AWSBedrockAPIKey
		}

		if bearerToken != "" {
			// Use REST client with bearer token
			printVerbose("Using Bedrock REST API with bearer token authentication")
			printInfo("Generating image with AWS Bedrock (REST API)...")

			client, err := generate.NewBedrockRESTClient(bearerToken, region)
			if err != nil {
				return fmt.Errorf("failed to create AWS Bedrock REST client: %w", err)
			}
			defer client.Close()

			generatedImage, err = client.GenerateImage(ctx, prompt, bedrockOptions)
		} else {
			// Use SDK client with IAM/keys/profile
			printVerbose("Using Bedrock SDK with IAM/keys/profile authentication")
			printInfo("Generating image with AWS Bedrock (SDK)...")

			client, err := generate.NewBedrockSDKClient(ctx, region)
			if err != nil {
				return fmt.Errorf("failed to create AWS Bedrock SDK client: %w", err)
			}
			defer client.Close()

			generatedImage, err = client.GenerateImage(ctx, prompt, bedrockOptions)
		}
	} else if selectedAPI == "grok" {
		// Use xAI Grok API
		key, err := config.GetGrokAPIKey("")
		if err != nil {
			return fmt.Errorf("failed to get Grok API key: %w", err)
		}

		// Use default model if not specified
		modelName := model
		if modelName == "" {
			modelName = "grok-2-image"
		}

		// Get provider info and announce selection
		registry := generate.GetProviderRegistry()
		provider, _ := registry.ResolveProvider(modelName)
		if provider != nil {
			printInfo("Using: %s (%s API)", provider.Name, provider.API)
			pricingDisplay := "Variable"
			if provider.Pricing.CostPerImage != nil {
				pricingDisplay = fmt.Sprintf("$%.4f/image", *provider.Pricing.CostPerImage)
			}
			printInfo("Pricing: %s", pricingDisplay)

			// Warn if expensive
			if provider.Pricing.CostPerImage != nil && *provider.Pricing.CostPerImage > 0.05 {
				fmt.Fprintf(os.Stderr, "⚠️  %s costs $%.4f/image\n", provider.Name, *provider.Pricing.CostPerImage)
			}
		}

		printInfo("Generating image with xAI Grok...")

		client, err := generate.NewGrokClient(key)
		if err != nil {
			return fmt.Errorf("failed to create Grok client: %w", err)
		}
		defer client.Close()

		// Build options - Grok doesn't support size/style/negative prompts
		grokOptions := models.GenerateOptions{
			Model: modelName,
		}

		generatedImage, err = client.GenerateImage(ctx, prompt, grokOptions)
	} else {
		return fmt.Errorf("invalid API: %s (must be 'gemini', 'vertex', 'bedrock', or 'grok')", selectedAPI)
	}

	if err != nil {
		return fmt.Errorf("failed to generate image: %w", err)
	}

	// Defensive nil check (should never happen if error handling is correct)
	if generatedImage == nil {
		return fmt.Errorf("internal error: generated image is nil but no error was returned - please report this bug")
	}

	// Determine output path
	if output == "" {
		output = generate.GenerateOutputPath(generatedImage.Format)
	}

	// Save image
	printInfo("Saving image to: %s", output)
	if err := generate.SaveImage(generatedImage, output); err != nil {
		return fmt.Errorf("failed to save image: %w", err)
	}

	// Print success with cost tracking
	printSuccess("Image generated successfully!")
	printInfo("  File: %s", output)
	printInfo("  Size: %s", formatImageSize(int64(len(generatedImage.Data))))
	printInfo("  Dimensions: %dx%d", generatedImage.Width, generatedImage.Height)

	// Log cost and token usage
	modelNameUsed := model
	if modelNameUsed == "" {
		if selectedAPI == "gemini" {
			modelNameUsed = generate.DefaultModel
		} else {
			modelNameUsed = "imagen-4.0-generate-001"
		}
	}

	registry := generate.GetProviderRegistry()
	if provider, err := registry.ResolveProvider(modelNameUsed); err == nil {
		if provider.Pricing.FreeTier {
			printInfo("  Cost: FREE (within %s)", provider.Pricing.FreeTierLimit)
		} else if provider.Pricing.CostPerImage != nil {
			// Gemini 3 Pro has variable pricing based on image size
			if strings.Contains(provider.ModelID, "gemini-3") || strings.Contains(provider.ModelID, "pro-image-preview") {
				cost := getGemini3ProCost(imageSize)
				printInfo("  Cost: $%.4f (%s)", cost, imageSizeLabel(imageSize))
			} else if strings.Contains(provider.ModelID, "nova-canvas") {
				// Nova Canvas has variable pricing based on dimensions
				cost := getNovaCanvasCost(size)
				w, h := parseDimensionsForCost(size)
				if w > 1024 || h > 1024 {
					printInfo("  Cost: $%.4f (>1024px)", cost)
				} else {
					printInfo("  Cost: $%.4f (≤1024px)", cost)
				}
			} else {
				printInfo("  Cost: $%.4f", *provider.Pricing.CostPerImage)
			}
		}
	}

	return nil
}

// imageSizeLabel returns a human-readable label for the image size
func imageSizeLabel(imageSize string) string {
	switch strings.ToUpper(imageSize) {
	case "4K":
		return "4K resolution"
	case "2K":
		return "2K resolution"
	case "1K":
		return "1K resolution"
	default:
		return "1K/2K resolution"
	}
}

// runGenerateWithProvider handles image generation using the new provider system
func runGenerateWithProvider(cmd *cobra.Command, prompt, providerID, output, size, style, negative string, seed int64) error {
	imageSize, _ := cmd.Flags().GetString("image-size")     // For Gemini 3 Pro: 1K, 2K, 4K
	aspectRatio, _ := cmd.Flags().GetString("aspect-ratio") // For Gemini 3 Pro: 1:1, 16:9, etc.
	registry := generate.GetProviderRegistry()

	// Resolve provider
	provider, err := registry.ResolveProvider(providerID)
	if err != nil {
		// Show available providers on error
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		printAvailableProviders()
		return fmt.Errorf("unknown provider: %s", providerID)
	}

	// Check authentication
	hasAuth, missing, err := registry.CheckAuth(provider)
	if err != nil {
		return fmt.Errorf("failed to check authentication: %w", err)
	}

	if !hasAuth {
		fmt.Fprintf(os.Stderr, "Provider '%s' is not configured.\n", provider.ID)
		fmt.Fprintf(os.Stderr, "Missing credentials: %v\n\n", missing)
		fmt.Fprintf(os.Stderr, "To set up authentication:\n")
		fmt.Fprintf(os.Stderr, "  gimage auth setup %s\n", provider.ID)
		return fmt.Errorf("authentication required")
	}

	// Show provider info
	printInfo("Using provider: %s", provider.Name)
	if provider.Pricing.FreeTier {
		printInfo("Pricing: FREE (%s)", provider.Pricing.FreeTierLimit)
	} else if provider.Pricing.CostPerImage != nil {
		cost := *provider.Pricing.CostPerImage
		printInfo("Pricing: $%.4f per image", cost)

		// Warn if expensive
		if cost > 0.05 {
			fmt.Fprintf(os.Stderr, "⚠️  This will cost $%.4f per image\n", cost)
		}
	}

	// Create client
	printInfo("Creating client for %s...", provider.API)
	client, err := registry.CreateClient(provider.ID)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Prepare options
	cfgScale, _ := cmd.Flags().GetFloat64("cfg-scale")
	count, _ := cmd.Flags().GetInt("count")
	outputFormat, _ := cmd.Flags().GetString("output-format")
	options := models.GenerateOptions{
		Model:          provider.ModelID,
		Size:           size,
		ImageSize:      imageSize,   // For Gemini 3 Pro: 1K, 2K, 4K
		AspectRatio:    aspectRatio, // For Gemini 3 Pro: 1:1, 16:9, etc.
		Style:          style,
		NegativePrompt: negative,
		Seed:           seed,
		CfgScale:       cfgScale,       // For Bedrock Nova Canvas (1.0-10.0)
		NumberOfImages: count,          // Number of images to generate (1-5)
		OutputFormat:   outputFormat,   // For all APIs: png, jpeg, webp
	}

	// Generate image
	printInfo("Generating image...")
	ctx := context.Background()

	startTime := time.Now()
	generatedImage, err := client.GenerateImage(ctx, prompt, options)
	if err != nil {
		return fmt.Errorf("failed to generate image: %w", err)
	}

	elapsed := time.Since(startTime)
	printInfo("Generation completed in %.2fs", elapsed.Seconds())

	// Determine output path
	if output == "" {
		output = generate.GenerateOutputPath(generatedImage.Format)
	}

	// Save image
	printInfo("Saving image to: %s", output)
	if err := generate.SaveImage(generatedImage, output); err != nil {
		return fmt.Errorf("failed to save image: %w", err)
	}

	// Print success with details
	printSuccess("Image generated successfully!")
	printInfo("  Provider: %s", provider.Name)
	printInfo("  File: %s", output)
	printInfo("  Size: %s", formatImageSize(int64(len(generatedImage.Data))))
	printInfo("  Dimensions: %dx%d", generatedImage.Width, generatedImage.Height)

	// Show cost info
	if provider.Pricing.FreeTier {
		printInfo("  Cost: FREE (within daily limit)")
	} else if provider.Pricing.CostPerImage != nil {
		// Gemini 3 Pro has variable pricing based on image size
		if strings.Contains(provider.ModelID, "gemini-3") || strings.Contains(provider.ModelID, "pro-image-preview") {
			cost := getGemini3ProCost(imageSize)
			printInfo("  Cost: $%.4f (%s)", cost, imageSizeLabel(imageSize))
		} else if strings.Contains(provider.ModelID, "nova-canvas") {
			// Nova Canvas has variable pricing based on dimensions
			cost := getNovaCanvasCost(size)
			w, h := parseDimensionsForCost(size)
			if w > 1024 || h > 1024 {
				printInfo("  Cost: $%.4f (>1024px)", cost)
			} else {
				printInfo("  Cost: $%.4f (≤1024px)", cost)
			}
		} else {
			printInfo("  Cost: $%.4f", *provider.Pricing.CostPerImage)
		}
	}

	return nil
}

// printAvailableProviders prints the list of available providers with auth status
func printAvailableProviders() error {
	registry := generate.GetProviderRegistry()
	statuses := registry.GetAuthStatus()

	fmt.Println("Available Providers:")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	// Group by API
	geminiProviders := []generate.AuthStatus{}
	vertexProviders := []generate.AuthStatus{}
	bedrockProviders := []generate.AuthStatus{}

	for _, status := range statuses {
		switch status.Provider.API {
		case "gemini":
			geminiProviders = append(geminiProviders, status)
		case "vertex":
			vertexProviders = append(vertexProviders, status)
		case "bedrock":
			bedrockProviders = append(bedrockProviders, status)
		}
	}

	// Print Gemini providers
	if len(geminiProviders) > 0 {
		fmt.Println("Gemini API (Google AI Studio):")
		for _, status := range geminiProviders {
			printProviderStatus(status)
		}
		fmt.Println()
	}

	// Print Vertex providers
	if len(vertexProviders) > 0 {
		fmt.Println("Vertex AI (Google Cloud):")
		for _, status := range vertexProviders {
			printProviderStatus(status)
		}
		fmt.Println()
	}

	// Print Bedrock providers
	if len(bedrockProviders) > 0 {
		fmt.Println("AWS Bedrock:")
		for _, status := range bedrockProviders {
			printProviderStatus(status)
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("─", 80))
	fmt.Println("\nUsage:")
	fmt.Println("  gimage generate \"prompt\" --provider <provider-id>")
	fmt.Println("\nExamples:")
	fmt.Println("  gimage generate \"sunset\" --provider gemini/flash-2.5")
	fmt.Println("  gimage generate \"portrait\" --provider vertex/imagen-4")
	fmt.Println("\nTo configure authentication:")
	fmt.Println("  gimage auth setup <provider-id>")

	return nil
}

func printProviderStatus(status generate.AuthStatus) {
	p := status.Provider

	// Status icon
	statusIcon := "✗"
	statusText := "Not configured"
	if status.Configured {
		statusIcon = "✓"
		statusText = "Ready"
	}

	// Pricing
	pricing := "Variable"
	if p.Pricing.FreeTier {
		pricing = fmt.Sprintf("FREE (%s)", p.Pricing.FreeTierLimit)
	} else if p.Pricing.CostPerImage != nil {
		pricing = fmt.Sprintf("$%.4f/image", *p.Pricing.CostPerImage)
	}

	fmt.Printf("  %s %-20s - %-30s [%s] %s\n",
		statusIcon,
		p.ID,
		p.Name,
		pricing,
		statusText,
	)
}

// printAvailableModels displays all available providers in a formatted table
func printAvailableModels() error {
	// Get provider registry and auth status
	registry := generate.GetProviderRegistry()
	statuses := registry.GetAuthStatus()

	// Group providers by API
	geminiProviders := []generate.AuthStatus{}
	vertexProviders := []generate.AuthStatus{}
	bedrockProviders := []generate.AuthStatus{}

	for _, status := range statuses {
		switch status.Provider.API {
		case "gemini":
			geminiProviders = append(geminiProviders, status)
		case "vertex":
			vertexProviders = append(vertexProviders, status)
		case "bedrock":
			bedrockProviders = append(bedrockProviders, status)
		}
	}

	// Check if any auth is configured
	hasAnyAuth := false
	for _, status := range statuses {
		if status.Configured {
			hasAnyAuth = true
			break
		}
	}

	// ASCII art header
	printInfo("\n╔═══════════════════════════════════════════════════════════════════════════════╗")
	printInfo("║                                                                               ║")
	printInfo("║     █████╗ ██╗   ██╗ █████╗ ██╗██╗      █████╗ ██████╗ ██╗     ███████╗     ║")
	printInfo("║    ██╔══██╗██║   ██║██╔══██╗██║██║     ██╔══██╗██╔══██╗██║     ██╔════╝     ║")
	printInfo("║    ███████║██║   ██║███████║██║██║     ███████║██████╔╝██║     █████╗       ║")
	printInfo("║    ██╔══██║╚██╗ ██╔╝██╔══██║██║██║     ██╔══██║██╔══██╗██║     ██╔══╝       ║")
	printInfo("║    ██║  ██║ ╚████╔╝ ██║  ██║██║███████╗██║  ██║██████╔╝███████╗███████╗     ║")
	printInfo("║    ╚═╝  ╚═╝  ╚═══╝  ╚═╝  ╚═╝╚═╝╚══════╝╚═╝  ╚═╝╚═════╝ ╚══════╝╚══════╝     ║")
	printInfo("║                                                                               ║")
	printInfo("║               🎨  AI Image Generation Providers  🎨                           ║")
	printInfo("║                                                                               ║")
	printInfo("╚═══════════════════════════════════════════════════════════════════════════════╝\n")

	if !hasAnyAuth {
		printWarning("⚠️  No API credentials configured. Set up authentication to use these providers:\n")
	}

	// Print Gemini providers - ALWAYS show, indicate auth status
	hasGemini := false
	for _, status := range geminiProviders {
		if status.Configured {
			hasGemini = true
			break
		}
	}
	printInfo("┌─────────────────────────────────────────────────────────────────────────────────┐")
	if hasGemini {
		printSuccess("│ ✓ Gemini API (AUTHENTICATED - Free Tier Available)                             │")
	} else {
		printWarning("│ ○ Gemini API (NOT AUTHENTICATED - Setup: gimage auth gemini)                   │")
	}
	printInfo("├─────────────────────────────────────────────────────────────────────────────────┤")
	printInfo("│ Providers:                                                                      │")
	printInfo("├─────────────────────────────────────────────────────────────────────────────────┤")
	for _, status := range geminiProviders {
		p := status.Provider
		// Format pricing
		pricingDisplay := "Variable"
		if p.Pricing.FreeTier {
			pricingDisplay = fmt.Sprintf("FREE (%s)", p.Pricing.FreeTierLimit)
		} else if p.Pricing.CostPerImage != nil {
			pricingDisplay = fmt.Sprintf("$%.4f/image", *p.Pricing.CostPerImage)
		}

		authMark := greenYes()
		if !status.Configured {
			authMark = redNo()
		}
		// Display provider name (73 chars for name)
		paddedName := padRight(p.Name, 73)
		printInfo("│ %s  %s │", authMark, paddedName)
		// Print details on second line (indented)
		printInfo("│     Provider ID: %-20s  Pricing: %-35s │", p.ID, pricingDisplay)
		if viper.GetBool("verbose") {
			printVerbose("│     %s", p.Description)
			printVerbose("│     Model ID: %s", p.ModelID)
		}
	}
	printInfo("└─────────────────────────────────────────────────────────────────────────────────┘\n")

	// Print Vertex providers - ALWAYS show, indicate auth status
	hasVertex := false
	for _, status := range vertexProviders {
		if status.Configured {
			hasVertex = true
			break
		}
	}
	printInfo("┌─────────────────────────────────────────────────────────────────────────────────┐")
	if hasVertex {
		printSuccess("│ ✓ Vertex AI (AUTHENTICATED - Paid, Requires GCP)                               │")
	} else {
		printWarning("│ ○ Vertex AI (NOT AUTHENTICATED - Setup: gimage auth vertex)                    │")
	}
	printInfo("├─────────────────────────────────────────────────────────────────────────────────┤")
	printInfo("│ Providers:                                                                      │")
	printInfo("├─────────────────────────────────────────────────────────────────────────────────┤")
	for _, status := range vertexProviders {
		p := status.Provider
		// Format pricing
		pricingDisplay := "Variable"
		if p.Pricing.CostPerImage != nil {
			pricingDisplay = fmt.Sprintf("$%.4f/image", *p.Pricing.CostPerImage)
		}

		authMark := greenYes()
		if !status.Configured {
			authMark = redNo()
		}
		// Display provider name (73 chars for name)
		paddedName := padRight(p.Name, 73)
		printInfo("│ %s  %s │", authMark, paddedName)
		// Print details on second line (indented)
		printInfo("│     Provider ID: %-20s  Pricing: %-35s │", p.ID, pricingDisplay)
		if viper.GetBool("verbose") {
			printVerbose("│     %s", p.Description)
			printVerbose("│     Model ID: %s", p.ModelID)
		}
	}
	printInfo("└─────────────────────────────────────────────────────────────────────────────────┘\n")

	// Print Bedrock providers - ALWAYS show, indicate auth status
	hasBedrock := false
	for _, status := range bedrockProviders {
		if status.Configured {
			hasBedrock = true
			break
		}
	}
	printInfo("┌─────────────────────────────────────────────────────────────────────────────────┐")
	if hasBedrock {
		printSuccess("│ ✓ AWS Bedrock (AUTHENTICATED - Paid, Requires AWS)                             │")
	} else {
		printWarning("│ ○ AWS Bedrock (NOT AUTHENTICATED - Setup: gimage auth bedrock)                 │")
	}
	printInfo("├─────────────────────────────────────────────────────────────────────────────────┤")
	printInfo("│ Providers:                                                                      │")
	printInfo("├─────────────────────────────────────────────────────────────────────────────────┤")
	for _, status := range bedrockProviders {
		p := status.Provider
		// Format pricing
		pricingDisplay := "Variable"
		if p.Pricing.CostPerImage != nil {
			pricingDisplay = fmt.Sprintf("$%.4f/image", *p.Pricing.CostPerImage)
		}

		authMark := greenYes()
		if !status.Configured {
			authMark = redNo()
		}
		// Display provider name (73 chars for name)
		paddedName := padRight(p.Name, 73)
		printInfo("│ %s  %s │", authMark, paddedName)
		// Print details on second line (indented)
		printInfo("│     Provider ID: %-20s  Pricing: %-35s │", p.ID, pricingDisplay)
		if viper.GetBool("verbose") {
			printVerbose("│     %s", p.Description)
			printVerbose("│     Model ID: %s", p.ModelID)
		}
	}
	printInfo("└─────────────────────────────────────────────────────────────────────────────────┘\n")

	printInfo("╔═══════════════════════════════════════════════════════════════════════════════╗")
	printInfo("║                                   LEGEND                                      ║")
	printInfo("╠═══════════════════════════════════════════════════════════════════════════════╣")
	printInfo("║  %s  = Authenticated (ready to use)                                          ║", greenYes())
	printInfo("║  %s  = Not authenticated (run setup command)                                 ║", redNo())
	printInfo("╚═══════════════════════════════════════════════════════════════════════════════╝")

	printInfo("\n╔═══════════════════════════════════════════════════════════════════════════════╗")
	printInfo("║                                 QUICK START                                   ║")
	printInfo("╠═══════════════════════════════════════════════════════════════════════════════╣")
	printInfo("║  Usage: gimage generate \"your prompt\" --model <model-name>                   ║")
	printInfo("╠═══════════════════════════════════════════════════════════════════════════════╣")
	printInfo("║  Recommended Providers:                                                       ║")
	if hasGemini {
		printInfo("║    ✓ Free users:  gemini (500/day FREE)                                       ║")
	} else {
		printInfo("║    ○ Free users:  gemini (setup: gimage auth gemini)                          ║")
	}
	if hasVertex {
		printInfo("║    ✓ Paid users:  imagen-4 ($0.04/image, highest quality)                    ║")
	} else {
		printInfo("║    ○ Paid users:  imagen-4 (setup: gimage auth vertex)                       ║")
	}
	if hasBedrock {
		printInfo("║    ✓ AWS users:   nova-canvas ($0.08/image)                                  ║")
	} else {
		printInfo("║    ○ AWS users:   nova-canvas (setup: gimage auth bedrock)                   ║")
	}
	printInfo("╠═══════════════════════════════════════════════════════════════════════════════╣")
	printInfo("║  Examples:                                                                    ║")
	if hasGemini {
		printInfo("║    gimage generate \"sunset\" --model gemini                                   ║")
	}
	if hasVertex {
		printInfo("║    gimage generate \"abstract art\" --model imagen-4                           ║")
	}
	if hasBedrock {
		printInfo("║    gimage generate \"landscape\" --api bedrock                                 ║")
		printInfo("║    gimage generate \"portrait\" --model nova-canvas                            ║")
	}
	printInfo("╚═══════════════════════════════════════════════════════════════════════════════╝")

	if !hasAnyAuth {
		printInfo("\n╔═══════════════════════════════════════════════════════════════════════════════╗")
		printInfo("║                            GET STARTED                                        ║")
		printInfo("╠═══════════════════════════════════════════════════════════════════════════════╣")
		printInfo("║  To get started, authenticate with at least one API:                         ║")
		printInfo("║                                                                               ║")
		printInfo("║    gimage auth gemini   # Fastest, has free tier                             ║")
		printInfo("║    gimage auth vertex   # Highest quality (paid)                             ║")
		printInfo("║    gimage auth bedrock  # AWS integration (paid)                             ║")
		printInfo("╚═══════════════════════════════════════════════════════════════════════════════╝")
	}

	return nil
}

// printPromptHowto displays tips and examples for writing effective prompts
func printPromptHowto() error {
	fmt.Println(`
╔═══════════════════════════════════════════════════════════════════════════════╗
║                        PROMPT WRITING GUIDE                                   ║
╚═══════════════════════════════════════════════════════════════════════════════╝

┌─────────────────────────────────────────────────────────────────────────────────┐
│ CORE PRINCIPLE                                                                  │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│   Describe the scene, don't just list keywords.                                 │
│                                                                                 │
│   The model excels at language comprehension. Descriptive paragraphs            │
│   consistently outperform disconnected word lists.                              │
│                                                                                 │
│   ✗ BAD:  "coffee mug, black, modern, studio"                                   │
│   ✓ GOOD: "a minimalist ceramic coffee mug in matte black, resting on           │
│            polished concrete, illuminated by soft three-point studio lighting"  │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────┐
│ PHOTOREALISTIC IMAGES                                                           │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│ Use photography terminology:                                                    │
│   • Camera/Lens: "shot with 85mm lens", "wide-angle perspective"                │
│   • Lighting: "golden hour light", "soft diffused lighting"                     │
│   • Mood: "serene atmosphere", "moody and cinematic"                            │
│   • Details: "fine texture visible", "shallow depth of field"                   │
│                                                                                 │
│ Example:                                                                        │
│   gimage generate "portrait of an elderly craftsman in his woodworking shop,    │
│   shot with 85mm lens, golden hour light streaming through dusty windows,       │
│   shallow depth of field, warm tones"                                           │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────┐
│ STYLIZED ILLUSTRATIONS                                                          │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│ Be explicit about artistic style:                                               │
│   • Art style: kawaii, cel-shading, minimalist, watercolor, vector art          │
│   • Line work: bold outlines, thin linework, no outlines                        │
│   • Colors: pastel palette, vibrant colors, monochromatic                       │
│   • Background: transparent, gradient, solid color                              │
│                                                                                 │
│ Example:                                                                        │
│   gimage generate "kawaii style cat sticker with bold black outlines,           │
│   pastel pink and white color palette, chibi proportions, simple background"    │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────┐
│ TEXT IN IMAGES (Gemini excels at this!)                                         │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│   • State the exact text to include in quotes                                   │
│   • Describe font style: "bold sans-serif", "elegant script"                    │
│   • Specify context: logo, poster, menu, sign                                   │
│                                                                                 │
│ Example:                                                                        │
│   gimage generate "vintage coffee shop chalkboard menu with hand-lettered       │
│   text reading 'Fresh Roasted Daily', weathered wooden frame, warm lighting"    │
│                                                                                 │
│ For professional text-heavy assets, use Gemini 3 Pro:                           │
│   gimage generate "..." --model gemini-3-pro --image-size 4K                    │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────┐
│ PRODUCT PHOTOGRAPHY                                                             │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│ Structure prompts with studio details:                                          │
│   1. Product description: materials, colors, size context                       │
│   2. Studio setup: lighting type, background surface                            │
│   3. Camera angle: overhead, eye-level, 45-degree                               │
│   4. Focus: what should be sharp vs soft                                        │
│                                                                                 │
│ Example:                                                                        │
│   gimage generate "luxury wristwatch with rose gold case and black leather      │
│   strap, positioned at 45-degree angle on white marble surface, three-point     │
│   softbox lighting, macro detail on watch face, clean white background"         │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────┐
│ QUICK TRANSFORMATIONS                                                           │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│ Weak → Strong:                                                                  │
│                                                                                 │
│   "sunset" →                                                                    │
│   "dramatic sunset over ocean waves, vibrant orange and purple sky,             │
│    silhouetted palm trees, shot from beach level"                               │
│                                                                                 │
│   "cat" →                                                                       │
│   "fluffy orange tabby cat curled up on vintage velvet armchair,                │
│    soft window light, cozy living room atmosphere"                              │
│                                                                                 │
│   "logo" →                                                                      │
│   "minimalist mountain logo in deep blue, clean geometric lines,                │
│    text 'SUMMIT' in bold sans-serif below"                                      │
│                                                                                 │
│   "food" →                                                                      │
│   "artisan sourdough bread loaf on rustic wooden cutting board,                 │
│    steam rising, golden crust with flour dusting, warm kitchen background"      │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────┐
│ STYLE MODIFIERS                                                                 │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│ Add these to shift the aesthetic:                                               │
│                                                                                 │
│   Realism:  "photorealistic", "hyperrealistic", "documentary style"             │
│   Art:      "oil painting style", "digital art", "pencil sketch"                │
│   Mood:     "cinematic", "ethereal", "gritty", "whimsical"                       │
│   Era:      "1970s film grain", "art deco", "cyberpunk", "vintage"              │
│   Quality:  "highly detailed", "4K resolution", "studio quality"                │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────┐
│ LIGHTING KEYWORDS                                                               │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│   Soft:     diffused, overcast, ambient, fill light                             │
│   Hard:     direct sunlight, spotlight, harsh shadows                           │
│   Dramatic: rim lighting, backlit, chiaroscuro, silhouette                      │
│   Natural:  golden hour, blue hour, midday sun, overcast sky                    │
│   Studio:   softbox, ring light, three-point setup, product lighting            │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────┐
│ SIZE & ASPECT RATIO                                                             │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│ Standard sizes (--size):                                                        │
│   Square (1024x1024):     social media posts, icons, avatars                    │
│   Landscape (1792x1024):  banners, scenes, desktop wallpapers                   │
│   Portrait (1024x1792):   phone wallpapers, portraits, stories                  │
│                                                                                 │
│   gimage generate "..." --size 1024x1024    # Square                            │
│   gimage generate "..." --size 1792x1024    # Landscape                         │
│   gimage generate "..." --size 1024x1792    # Portrait                          │
│                                                                                 │
│ Gemini 3 Pro native options (--image-size, --aspect-ratio):                     │
│   gimage generate "..." --model gemini-3-pro --image-size 4K                    │
│   gimage generate "..." --model gemini-3-pro --aspect-ratio 16:9                │
│   gimage generate "..." --model gemini-3-pro --image-size 2K --aspect-ratio 5:4 │
│                                                                                 │
│ Supported aspect ratios: 1:1, 16:9, 9:16, 4:3, 3:4, 5:4, 4:5, 3:2, 2:3          │
│                                                                                 │
│ Gemini 3 Pro pricing (varies by size):                                          │
│   1K/2K: $0.134/image    4K: $0.24/image                                        │
│                                                                                 │
│ Nova Canvas pricing (varies by dimensions):                                     │
│   ≤1024x1024: $0.04/image    >1024x1024: $0.08/image                            │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────┐
│ TIPS FOR BETTER RESULTS                                                         │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│   1. Start simple, add details incrementally                                    │
│   2. Be specific about what matters most                                        │
│   3. Use concrete nouns over abstract concepts                                  │
│   4. Specify what you don't want with --negative flag                           │
│   5. Try different phrasings - small changes yield different results            │
│   6. Use --seed for reproducibility when iterating                              │
│                                                                                 │
│ Example with negative prompt:                                                   │
│   gimage generate "serene forest path" --negative "people, buildings, cars"     │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘

Source: https://ai.google.dev/gemini-api/docs/image-generation`)
	return nil
}

func init() {
	generateCmd.Flags().StringP("prompt", "p", "", "Text description of the image to generate (required)")
	generateCmd.Flags().StringP("output", "o", "", "Output file path (default: generated_<timestamp>.png)")
	generateCmd.Flags().String("provider", "", "Provider to use (e.g., gemini/flash-2.5, vertex/imagen-4)")
	generateCmd.Flags().String("api", "", "API to use: gemini or vertex (deprecated, use --provider)")
	generateCmd.Flags().String("api-key", "", "Gemini API key (or use GEMINI_API_KEY env var)")
	generateCmd.Flags().String("project", "", "Vertex AI project ID (or use GIMAGE_VERTEX_PROJECT env var)")
	generateCmd.Flags().String("location", "us-central1", "Vertex AI location")
	generateCmd.Flags().String("model", "", fmt.Sprintf("Model to use (deprecated, use --provider). Default: %s", generate.DefaultModel))
	generateCmd.Flags().Bool("list-models", false, "List all available models and exit")
	generateCmd.Flags().Bool("list-providers", false, "List all available providers with pricing and auth status")
	generateCmd.Flags().Bool("prompt-howto", false, "Show tips and examples for writing effective prompts")
	generateCmd.Flags().String("size", "1024x1024", "Image size (e.g., 1024x1024, 512x512)")
	generateCmd.Flags().String("image-size", "", "Native resolution for Gemini 3 Pro: 1K, 2K, or 4K (sharp text/diagrams)")
	generateCmd.Flags().String("aspect-ratio", "", "Aspect ratio for Gemini 3 Pro (e.g., 1:1, 16:9, 4:3, 3:4, 9:16)")
	generateCmd.Flags().String("style", "", "Image style: photorealistic, artistic, anime")
	generateCmd.Flags().String("negative", "", "Negative prompt to avoid certain features")
	generateCmd.Flags().Int64("seed", 0, "Random seed for reproducibility (0 for random)")
	generateCmd.Flags().Float64("cfg-scale", 0, "Guidance scale for Nova Canvas (1.0-10.0, default 7.0)")
	generateCmd.Flags().IntP("count", "n", 1, "Number of images to generate (1-5, default 1)")
	generateCmd.Flags().String("output-format", "", "Output format: png, jpeg, webp (default: png)")

	// Bind to viper for config file support
	viper.BindPFlag("generate.api", generateCmd.Flags().Lookup("api"))
	viper.BindPFlag("generate.model", generateCmd.Flags().Lookup("model"))
	viper.BindPFlag("generate.size", generateCmd.Flags().Lookup("size"))

	rootCmd.AddCommand(generateCmd)
}
