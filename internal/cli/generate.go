package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/apresai/gimage/internal/config"
	"github.com/apresai/gimage/internal/generate"
	"github.com/apresai/gimage/pkg/models"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

// sanitizeInputImagePaths trims whitespace from each path and drops empty
// entries. This keeps CLI behavior consistent with the MCP server's
// coerceStringArray, so `--input-image ""` (or whitespace-only paths) are
// silently ignored rather than passed downstream as broken paths.
func sanitizeInputImagePaths(paths []string) []string {
	if len(paths) == 0 {
		return paths
	}
	out := paths[:0]
	for _, p := range paths {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
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

// ============================================================================
// Standardized Output Functions
// ============================================================================
// These functions ensure consistent output format across all providers.
// Uses shared pricing helpers from generate.GetProviderPricing() for consistency
// with MCP server output.

// printProviderHeader prints the standard provider selection header
// Format: "Using: <Provider Name>"
// Note: Provider.Name already includes the API context (e.g., "via Gemini API")
func printProviderHeader(provider *generate.Provider) {
	printInfo("Using: %s", provider.Name)
}

// printPricingInfo prints the standard pricing line and, if the provider is
// expensive (>$0.05/image), a cost warning on stderr. Combines what used to be
// two helpers so callers only do one pricing lookup per generation.
func printPricingInfo(provider *generate.Provider, imageSize, dimensions, style string) {
	pricing := generate.GetProviderPricing(provider, imageSize, dimensions, style)
	printInfo("Pricing: %s", pricing.Display)
	if pricing.IsExpensive {
		fmt.Fprintf(os.Stderr, "⚠️  %s costs $%.4f/image\n", provider.Name, pricing.Cost)
	}
}

// printGenerationTiming prints the generation timing
// Format: "Generation completed in X.XXs"
func printGenerationTiming(elapsed time.Duration) {
	printInfo("Generation completed in %.2fs", elapsed.Seconds())
}

func defaultThinkingLevelForProvider(provider *generate.Provider) string {
	if provider == nil {
		return ""
	}
	switch provider.ID {
	case "vertex/flash-3.1-fast", "vertex/flash-3.1-lite":
		return "minimal"
	case "vertex/flash-3.1":
		return "medium"
	case "vertex/flash-3.1-ultra":
		return "high"
	default:
		return ""
	}
}

// reconcileAndWarn strips options the resolved provider does not support and
// prints a friendly warning for each one. No-ops safely on a nil provider.
func reconcileAndWarn(provider *generate.Provider, options *models.GenerateOptions) {
	for _, msg := range generate.ReconcileOptions(provider, options) {
		printWarning("%s", msg)
	}
}

// printSuccessDetails prints the standard success block for a single image
func printSuccessDetails(provider *generate.Provider, output string, imageData []byte, width, height int, imageSize, dimensions, style string) {
	printInfo("  File: %s", output)
	printInfo("  Size: %s", formatImageSize(int64(len(imageData))))
	printInfo("  Dimensions: %dx%d", width, height)

	// Cost info using shared pricing helper
	pricing := generate.GetProviderPricing(provider, imageSize, dimensions, style)
	if pricing.IsFree {
		printInfo("  Cost: %s", pricing.Display)
	} else if pricing.Cost > 0 {
		printInfo("  Cost: %s", pricing.Display)
	}
}

var generateCmd = &cobra.Command{
	Use:   "generate [prompt]",
	Short: "Generate an image from a text prompt using AI",
	Long: `Generate an image from a text prompt using Google Gemini, Vertex AI, or xAI Grok.

The prompt should describe the image you want to generate. You can optionally
specify style, size, aspect ratio, and other parameters. Options that the chosen
provider does not support are reported and ignored.

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
  gimage generate "futuristic city" --model vertex-flash

  # Generate with xAI Grok
  gimage generate "futuristic city" --model grok-imagine

  # Generate with specific style and size
  gimage generate "abstract art" --size 1024x1024 --style photorealistic

  # Override API selection (rarely needed)
  gimage generate "abstract art" --api vertex --model vertex-flash

  # Use a seed for reproducibility (Gemini API providers)
  gimage generate "forest scene" --seed 12345`,
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

	registry := generate.GetProviderRegistry()
	var modelProvider *generate.Provider
	if model != "" {
		var providerErr error
		modelProvider, providerErr = registry.ResolveProvider(model)
		if providerErr != nil {
			return fmt.Errorf("invalid model: %w\nUse --list-models to see available models", providerErr)
		}
		printVerbose("Resolved model '%s' to provider '%s' (%s)", model, modelProvider.ID, modelProvider.ModelID)
	}

	size, _ := cmd.Flags().GetString("size")
	imageSize, _ := cmd.Flags().GetString("image-size")     // For Gemini 3+ (Pro/3.1 Flash): 1K, 2K, 4K
	aspectRatio, _ := cmd.Flags().GetString("aspect-ratio") // For Gemini 3+ and Grok Imagine: 1:1, 16:9, etc.
	style, _ := cmd.Flags().GetString("style")
	seed, _ := cmd.Flags().GetInt64("seed")
	count, _ := cmd.Flags().GetInt("count") // Number of images to generate
	outputFormat, _ := cmd.Flags().GetString("output-format")
	listModels, _ := cmd.Flags().GetBool("list-models")
	listProviders, _ := cmd.Flags().GetBool("list-providers")
	resizeMode, _ := cmd.Flags().GetString("resize-mode")
	thinkingLevel, _ := cmd.Flags().GetString("thinking")       // Gemini 3+ only
	grounding, _ := cmd.Flags().GetBool("grounding")            // Gemini 3+ only
	inputImages, _ := cmd.Flags().GetStringArray("input-image") // Reference images for compositional editing
	userID, _ := cmd.Flags().GetString("user")                  // Optional end-user id (Grok/xAI abuse monitoring)
	quality, _ := cmd.Flags().GetString("quality")              // Grok Imagine 2.0 only: low|medium|auto
	inputImages = sanitizeInputImagePaths(inputImages)

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
		return runGenerateWithProvider(cmd, prompt, providerID, output, size, style, seed)
	}

	// Determine which API to use
	// Priority: 1. --api flag, 2. Auto-detect from model, 3. Auto-detect from available credentials
	selectedAPI := api
	if selectedAPI == "" {
		// Auto-detect API from model
		if modelProvider != nil {
			selectedAPI = modelProvider.API
		} else {
			// Auto-detect from available credentials
			hasGemini := config.HasGeminiCredentials()
			hasVertex := config.HasVertexCredentials()
			hasGrok := config.HasGrokCredentials()

			// Count available credentials
			availableCount := 0
			if hasGemini {
				availableCount++
			}
			if hasVertex {
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
					"  Grok:    export GROK_API_KEY=your-key")
			} else if availableCount == 1 {
				// Only one API available
				if hasGemini {
					selectedAPI = "gemini"
					printVerbose("Auto-detected Gemini API (found credentials)")
				} else if hasVertex {
					selectedAPI = "vertex"
					printVerbose("Auto-detected Vertex API (found credentials)")
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
	if modelProvider != nil && modelProvider.API != selectedAPI {
		return fmt.Errorf("model %s is not available on %s API (only on %s)", model, selectedAPI, modelProvider.API)
	}
	if modelProvider != nil {
		model = modelProvider.ModelID
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var generatedImages []*models.GeneratedImage
	var resolvedProvider *generate.Provider
	var startTime time.Time
	var err error

	// Create unified generation options
	options := models.GenerateOptions{
		Model:              model,
		Size:               size,
		ImageSize:          imageSize,
		AspectRatio:        aspectRatio,
		Style:              style,
		Seed:               seed,
		NumberOfImages:     count,
		OutputFormat:       outputFormat,
		ResizeMode:         resizeMode,
		User:               userID,
		Quality:            quality,
		ThinkingLevel:      thinkingLevel,
		WebSearchGrounding: grounding,
		InputImages:        inputImages,
	}

	// Generate based on API selection
	if selectedAPI == "gemini" {
		// Use Gemini API (REST implementation)
		// NOTE: use distinct names for inner errors so the outer `err` (declared
		// above) is correctly assigned by `generatedImages, err = client.GenerateImage(...)`
		// below. Otherwise the `err :=` here would shadow it and API errors would
		// be silently dropped when the if-block ends.
		key, keyErr := config.GetGeminiAPIKey(apiKey)
		if keyErr != nil {
			return fmt.Errorf("failed to get Gemini API key: %w\nHint: Set GEMINI_API_KEY environment variable or use --api-key flag", keyErr)
		}

		modelName := model
		if modelName == "" {
			modelName = generate.DefaultModel
		}

		// Get provider info and announce selection
		if modelProvider != nil {
			resolvedProvider = modelProvider
		} else {
			resolvedProvider, _ = registry.ResolveProvider(modelName)
		}
		if resolvedProvider != nil {
			printProviderHeader(resolvedProvider)
			printPricingInfo(resolvedProvider, imageSize, size, style)
		}

		printInfo("Generating image...")

		// Use REST client instead of SDK client
		client, clientErr := generate.NewGeminiRESTClient(key)
		if clientErr != nil {
			return fmt.Errorf("failed to create Gemini client: %w", clientErr)
		}
		defer client.Close()

		// Update model name in options if needed
		options.Model = modelName

		reconcileAndWarn(resolvedProvider, &options)

		startTime = time.Now()
		generatedImages, err = client.GenerateImage(ctx, prompt, options)
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

		// Check for Vertex API key (Express Mode). Use a distinct error name so
		// the outer `err` survives to capture the eventual GenerateImage error.
		vertexAPIKey, keyErr := config.GetVertexAPIKey(apiKey)
		if keyErr != nil {
			return fmt.Errorf("failed to get Vertex API key: %w", keyErr)
		}

		modelName := model
		if modelName == "" {
			resolvedProvider, _ = registry.Get("vertex/flash-3.1")
			if resolvedProvider != nil {
				modelName = resolvedProvider.ModelID
			} else {
				modelName = "gemini-3.1-flash-image"
			}
		}

		// Get provider info and announce selection
		if modelProvider != nil {
			resolvedProvider = modelProvider
		} else if resolvedProvider == nil {
			resolvedProvider, _ = registry.ResolveProvider(modelName)
		}
		if resolvedProvider != nil {
			printProviderHeader(resolvedProvider)
			printPricingInfo(resolvedProvider, imageSize, size, style)
		}
		if thinkingLevel == "" {
			thinkingLevel = defaultThinkingLevelForProvider(resolvedProvider)
			options.ThinkingLevel = thinkingLevel
		}
		reconcileAndWarn(resolvedProvider, &options)

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

			client, clientErr := generate.NewVertexRESTClient(vertexAPIKey, project, location)
			if clientErr != nil {
				return fmt.Errorf("failed to create Vertex AI REST client: %w", clientErr)
			}
			defer client.Close()

			startTime = time.Now()
			generatedImages, err = client.GenerateImage(ctx, prompt, options)
		} else {
			// Full Mode - Use unified SDK client with ADC/service account
			printVerbose("Using Vertex AI Full Mode (ADC/service account authentication)")

			client, clientErr := generate.NewVertexUnifiedClient(ctx, project, location)
			if clientErr != nil {
				return fmt.Errorf("failed to create Vertex AI client: %w", clientErr)
			}
			defer client.Close()

			startTime = time.Now()
			generatedImages, err = client.GenerateImage(ctx, prompt, options)
		}
	} else if selectedAPI == "grok" {
		// Use xAI Grok API. Distinct error name to avoid shadowing outer `err`.
		key, keyErr := config.GetGrokAPIKey("")
		if keyErr != nil {
			return fmt.Errorf("failed to get Grok API key: %w", keyErr)
		}

		// Use default model if not specified
		modelName := model
		if modelName == "" {
			modelName = "grok-imagine-image-2.0"
		}

		// Get provider info and announce selection
		registry := generate.GetProviderRegistry()
		resolvedProvider, _ = registry.ResolveProvider(modelName)
		if resolvedProvider != nil {
			printProviderHeader(resolvedProvider)
			printPricingInfo(resolvedProvider, imageSize, size, style)
		}

		printInfo("Generating image...")

		client, clientErr := generate.NewGrokClient(key)
		if clientErr != nil {
			return fmt.Errorf("failed to create Grok client: %w", clientErr)
		}
		defer client.Close()

		// Use the unified options with resolved model name
		options.Model = modelName

		reconcileAndWarn(resolvedProvider, &options)

		startTime = time.Now()
		generatedImages, err = client.GenerateImage(ctx, prompt, options)
	} else {
		return fmt.Errorf("invalid API: %s (must be 'gemini', 'vertex', or 'grok')", selectedAPI)
	}

	if err != nil {
		return fmt.Errorf("failed to generate image: %w", err)
	}

	// Print timing
	if !startTime.IsZero() {
		printGenerationTiming(time.Since(startTime))
	}

	// Defensive nil check (should never happen if error handling is correct)
	if len(generatedImages) == 0 {
		return fmt.Errorf("internal error: no images generated but no error was returned - please report this bug")
	}

	printSuccess("Generated %d image(s) successfully!", len(generatedImages))
	if resolvedProvider != nil {
		printInfo("  Provider: %s", resolvedProvider.Name)
	}

	// Save all generated images
	for i, img := range generatedImages {
		// Determine output path for this specific image
		var currentOutput string
		if output != "" {
			if len(generatedImages) > 1 {
				// Add suffix for multiple images (e.g., image_1.png, image_2.png)
				ext := strings.ToLower(img.Format)
				if !strings.HasPrefix(ext, ".") {
					ext = "." + ext
				}
				base := strings.TrimSuffix(output, filepath.Ext(output))
				currentOutput = fmt.Sprintf("%s_%d%s", base, i+1, ext)
			} else {
				currentOutput = output
			}
		} else {
			currentOutput = generate.GenerateOutputPath(img.Format)
		}

		printInfo("Saving image %d to: %s", i+1, currentOutput)
		if err := generate.SaveImage(img, currentOutput); err != nil {
			return fmt.Errorf("failed to save image %d: %w", i+1, err)
		}

		// Print standardized success output for this image
		if resolvedProvider != nil {
			printSuccessDetails(resolvedProvider, currentOutput, img.Data, img.Width, img.Height, imageSize, size, style)
		} else {
			printInfo("  File: %s", currentOutput)
			printInfo("  Size: %s", formatImageSize(int64(len(img.Data))))
			printInfo("  Dimensions: %dx%d", img.Width, img.Height)
		}
	}

	return nil
}

// runGenerateWithProvider handles image generation using the new provider system
func runGenerateWithProvider(cmd *cobra.Command, prompt, providerID, output, size, style string, seed int64) error {
	imageSize, _ := cmd.Flags().GetString("image-size")     // For Gemini 3+ (Pro/3.1 Flash): 1K, 2K, 4K
	aspectRatio, _ := cmd.Flags().GetString("aspect-ratio") // For Gemini 3+ and Grok Imagine: 1:1, 16:9, etc.
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

	// Show provider info using standardized output
	printProviderHeader(provider)
	printPricingInfo(provider, imageSize, size, style)

	// Create client
	client, err := registry.CreateClient(provider.ID)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Prepare options
	count, _ := cmd.Flags().GetInt("count")
	outputFormat, _ := cmd.Flags().GetString("output-format")
	thinkingLevel, _ := cmd.Flags().GetString("thinking")
	grounding, _ := cmd.Flags().GetBool("grounding")
	inputImages, _ := cmd.Flags().GetStringArray("input-image")
	inputImages = sanitizeInputImagePaths(inputImages)
	userID, _ := cmd.Flags().GetString("user")
	quality, _ := cmd.Flags().GetString("quality")
	if thinkingLevel == "" {
		thinkingLevel = defaultThinkingLevelForProvider(provider)
	}
	options := models.GenerateOptions{
		Model:              provider.ModelID,
		Size:               size,
		ImageSize:          imageSize,   // For Gemini 3+ (Pro/3.1 Flash): 1K, 2K, 4K
		AspectRatio:        aspectRatio, // For Gemini 3+ and Grok Imagine: 1:1, 16:9, etc.
		Style:              style,
		Seed:               seed,
		NumberOfImages:     count,        // Number of images to generate (Grok exact up to 10)
		OutputFormat:       outputFormat, // For all APIs: png, jpeg, webp
		User:               userID,       // Grok/xAI optional end-user id
		Quality:            quality,      // Grok Imagine 2.0 only
		ThinkingLevel:      thinkingLevel,
		WebSearchGrounding: grounding,
		InputImages:        inputImages,
	}
	reconcileAndWarn(provider, &options)

	// Generate image
	printInfo("Generating image...")
	ctx := context.Background()

	startTime := time.Now()
	generatedImages, err := client.GenerateImage(ctx, prompt, options)
	if err != nil {
		return fmt.Errorf("failed to generate image: %w", err)
	}

	printGenerationTiming(time.Since(startTime))

	if len(generatedImages) == 0 {
		return fmt.Errorf("no images generated")
	}

	printSuccess("Generated %d image(s) successfully!", len(generatedImages))
	printInfo("  Provider: %s", provider.Name)

	// Save all generated images
	for i, img := range generatedImages {
		var currentOutput string
		if output != "" {
			if len(generatedImages) > 1 {
				ext := strings.ToLower(img.Format)
				if !strings.HasPrefix(ext, ".") {
					ext = "." + ext
				}
				base := strings.TrimSuffix(output, filepath.Ext(output))
				currentOutput = fmt.Sprintf("%s_%d%s", base, i+1, ext)
			} else {
				currentOutput = output
			}
		} else {
			currentOutput = generate.GenerateOutputPath(img.Format)
		}

		printInfo("Saving image %d to: %s", i+1, currentOutput)
		if err := generate.SaveImage(img, currentOutput); err != nil {
			return fmt.Errorf("failed to save image %d: %w", i+1, err)
		}

		printSuccessDetails(provider, currentOutput, img.Data, img.Width, img.Height, imageSize, size, style)
	}

	return nil
}

// printAvailableProviders prints the list of available providers with auth status
func printAvailableProviders() error {
	registry := generate.GetProviderRegistry()

	fmt.Println("Available Providers:")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()

	// Get providers grouped by API dynamically
	grouped := registry.GroupAuthStatusByAPI()
	apis := registry.ListAPIs()

	// Print providers for each API in order
	for _, apiInfo := range apis {
		providers := grouped[apiInfo.ID]
		if len(providers) > 0 {
			fmt.Printf("%s:\n", apiInfo.DisplayName)
			for _, status := range providers {
				printProviderStatus(status)
			}
			fmt.Println()
		}
	}

	fmt.Println(strings.Repeat("─", 80))
	fmt.Println("\nUsage:")
	fmt.Println("  gimage generate \"prompt\" --provider <provider-id>")

	// Generate examples dynamically from first provider of each API
	fmt.Println("\nExamples:")
	for _, apiInfo := range apis {
		providers := grouped[apiInfo.ID]
		if len(providers) > 0 {
			// Use first provider from this API as example
			fmt.Printf("  gimage generate \"prompt\" --provider %s\n", providers[0].Provider.ID)
		}
	}

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
// Uses the provider registry dynamically so new APIs/models appear automatically.
func printAvailableModels() error {
	registry := generate.GetProviderRegistry()
	grouped := registry.GroupAuthStatusByAPI()
	apis := registry.ListAPIs()

	// Check if any auth is configured
	hasAnyAuth := false
	for _, apiInfo := range apis {
		for _, status := range grouped[apiInfo.ID] {
			if status.Configured {
				hasAnyAuth = true
				break
			}
		}
		if hasAnyAuth {
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
	printInfo("║               AI Image Generation Providers                                   ║")
	printInfo("║                                                                               ║")
	printInfo("╚═══════════════════════════════════════════════════════════════════════════════╝\n")

	if !hasAnyAuth {
		printWarning("No API credentials configured. Set up authentication to use these providers:\n")
	}

	// Track which APIs are authenticated for quick start section
	apiAuth := map[string]bool{}

	// Print providers for each API dynamically
	for _, apiInfo := range apis {
		providers := grouped[apiInfo.ID]
		if len(providers) == 0 {
			continue
		}

		// Check if this API is authenticated
		hasAuth := false
		for _, status := range providers {
			if status.Configured {
				hasAuth = true
				break
			}
		}
		apiAuth[apiInfo.ID] = hasAuth

		printInfo("┌─────────────────────────────────────────────────────────────────────────────────┐")
		if hasAuth {
			printSuccess("| AUTHENTICATED: %s", apiInfo.DisplayName)
		} else {
			printWarning("| NOT AUTHENTICATED: %s (Setup: gimage auth %s)", apiInfo.DisplayName, apiInfo.ID)
		}
		printInfo("├─────────────────────────────────────────────────────────────────────────────────┤")

		for _, status := range providers {
			p := status.Provider
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
			paddedName := padRight(p.Name, 73)
			printInfo("│ %s  %s │", authMark, paddedName)
			printInfo("│     Provider ID: %-20s  Pricing: %-35s │", p.ID, pricingDisplay)
			if viper.GetBool("verbose") {
				printVerbose("│     %s", p.Description)
				printVerbose("│     Model ID: %s", p.ModelID)
			}
		}
		printInfo("└─────────────────────────────────────────────────────────────────────────────────┘\n")
	}

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
	printInfo("║  Examples:                                                                    ║")

	// Show examples for authenticated APIs
	for _, apiInfo := range apis {
		if apiAuth[apiInfo.ID] {
			providers := grouped[apiInfo.ID]
			if len(providers) > 0 {
				// Use first provider as example
				printInfo("║    gimage generate \"prompt\" --model %s", providers[0].Provider.ID)
			}
		}
	}

	printInfo("╚═══════════════════════════════════════════════════════════════════════════════╝")

	if !hasAnyAuth {
		printInfo("\n  To get started, authenticate with at least one API:")
		printInfo("    gimage auth gemini   # Fastest, most affordable")
		printInfo("    gimage auth vertex   # Highest quality (paid)")
		printInfo("    gimage auth grok     # xAI Aurora (paid)")
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
│ For professional text-heavy assets, use Gemini 3 Pro or 3.1 Flash:              │
│   gimage generate "..." --model gemini-3-pro --image-size 4K                    │
│   gimage generate "..." --model gemini-3.1-flash --image-size 4K                │
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
│ Gemini 3+ native options (--image-size, --aspect-ratio):                        │
│   gimage generate "..." --model gemini-3-pro --image-size 4K                    │
│   gimage generate "..." --model gemini-3-pro --aspect-ratio 16:9                │
│   gimage generate "..." --model gemini-3.1-flash --image-size 2K                │
│   gimage generate "..." --model gemini-3.1-flash --aspect-ratio 16:9            │
│                                                                                 │
│ Supported aspect ratios: 1:1, 16:9, 9:16, 4:3, 3:4, 5:4, 4:5, 3:2, 2:3,         │
│   21:9, 1:4, 4:1, 1:8, 8:1 (Gemini); Grok also adds 2:1, 1:2, 19.5:9,           │
│   9:19.5, 20:9, 9:20, 5:2, auto                                                 │
│                                                                                 │
│ Gemini 3 Pro pricing (varies by size):                                          │
│   1K/2K: $0.134/image    4K: $0.24/image                                        │
│                                                                                 │
│ Gemini 3.1 Flash pricing (tiered by resolution):                                │
│   0.5K: $0.045    1K: $0.067    2K: $0.101    4K: $0.151                        │
│                                                                                 │
│ Gemini 3.1 Flash Lite pricing: $0.034/image (1K only)                           │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────┐
│ REFERENCE IMAGE EDITING (Gemini, Vertex, Grok)                                  │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│ Pass one or more reference images via --input-image and describe the change.    │
│   Caps: Grok 2.0=5, older Grok=3, Gemini 2.5=3, Pro=11, Flash/Lite=14          │
│   Formats: PNG, JPEG, WebP. Each image capped at 20 MB.                         │
│   Grok uses xAI /images/edits; multi-image prompts may use <IMAGE_0>, etc.      │
│                                                                                 │
│   gimage generate "place this on a marble counter, soft studio light" \         │
│     --model gemini-3.1-flash --input-image product.png                          │
│                                                                                 │
│   gimage generate "render this as a pencil sketch with detailed shading" \      │
│     --model grok-quality --input-image photo.png --image-size 2K                │
│                                                                                 │
│   gimage generate "this character in this scene, cinematic lighting" \          │
│     --model gemini-3-pro --input-image character.png --input-image scene.jpg    │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────┐
│ GEMINI 3+ ADVANCED FEATURES (silently ignored on other models)                  │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│ Thinking mode: dial up reasoning depth for text-heavy or complex layouts.       │
│   Levels: minimal | low (default) | medium | high                               │
│   Higher = better layout/text, slightly slower.                                 │
│                                                                                 │
│   gimage generate "Q3 infographic 'Revenue +18%'" \                             │
│     --model gemini-3-pro --image-size 4K --thinking high                        │
│                                                                                 │
│ Google Search grounding: enable for accurate renders of real-world entities.    │
│   Billed per search query in addition to the per-image cost.                    │
│                                                                                 │
│   gimage generate "official Nintendo Switch 2 console front view" \             │
│     --model gemini-3-pro --grounding                                            │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────┐
│ TIPS FOR BETTER RESULTS                                                         │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│   1. Start simple, add details incrementally                                    │
│   2. Be specific about what matters most                                        │
│   3. Use concrete nouns over abstract concepts                                  │
│   4. State exclusions directly in the prompt (e.g. "no text, no people")        │
│   5. Try different phrasings - small changes yield different results            │
│   6. Use --seed for reproducibility when iterating (Gemini API providers)       │
│                                                                                 │
│ Example excluding elements in the prompt:                                       │
│   gimage generate "serene empty forest path, no people, no buildings, no cars"  │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘

Source: https://ai.google.dev/gemini-api/docs/image-generation`)
	return nil
}

func init() {
	generateCmd.Flags().StringP("prompt", "p", "", "Text description of the image to generate (required)")
	generateCmd.Flags().StringP("output", "o", "", "Output file path (default: generated_<timestamp>.png)")
	generateCmd.Flags().String("provider", "", "Provider to use (e.g., gemini/flash-2.5, vertex/flash-3.1)")
	generateCmd.Flags().String("api", "", "API to use: gemini or vertex (deprecated, use --provider)")
	generateCmd.Flags().String("api-key", "", "Gemini API key (or use GEMINI_API_KEY env var)")
	generateCmd.Flags().String("project", "", "Vertex AI project ID (or use GIMAGE_VERTEX_PROJECT env var)")
	generateCmd.Flags().String("location", "us-central1", "Vertex AI location")
	generateCmd.Flags().String("model", "", fmt.Sprintf("Model to use (deprecated, use --provider). Default: %s", generate.DefaultModel))
	generateCmd.Flags().Bool("list-models", false, "List all available models and exit")
	generateCmd.Flags().Bool("list-providers", false, "List all available providers with pricing and auth status")
	generateCmd.Flags().Bool("prompt-howto", false, "Show tips and examples for writing effective prompts")
	generateCmd.Flags().String("size", "1024x1024", "Image size (e.g., 1024x1024, 512x512)")
	generateCmd.Flags().String("image-size", "", "Native resolution. Gemini 3 Pro / 3.1 Flash / vertex-flash*: 1K, 2K, or 4K. Gemini 3.1 Flash Lite: 1K only. Grok Imagine: 1K or 2K only.")
	generateCmd.Flags().String("aspect-ratio", "", "Aspect ratio for Gemini 3+ and Grok Imagine. Grok also accepts 2:1, 1:2, 19.5:9, 9:19.5, 20:9, 9:20, 21:9, 5:2, auto")
	generateCmd.Flags().String("style", "", "Image style: photorealistic, artistic, anime (Gemini/Vertex only)")
	generateCmd.Flags().Int64("seed", 0, "Random seed for reproducibility, 0 for random (Gemini API providers)")
	generateCmd.Flags().IntP("count", "n", 1, "Number of images to generate (Grok returns N exactly up to 10; Gemini is best-effort)")
	generateCmd.Flags().String("output-format", "", "Output format: png, jpeg, webp (default: png)")
	generateCmd.Flags().String("resize-mode", "crop", "Resize mode when dimensions don't match: crop (fill), fit (padding) (default: crop)")
	generateCmd.Flags().String("thinking", "", "Reasoning depth for Gemini 3+ (minimal|low|medium|high). Flash Lite accepts minimal|high only. Ignored for Gemini 2.5 Flash.")
	generateCmd.Flags().Bool("grounding", false, "Enable Google Search grounding for Gemini 3+ (billed per search query). Ignored by Flash Lite and Grok.")
	generateCmd.Flags().String("quality", "", "Grok Imagine 2.0 generation quality: low, medium, or auto (default auto). Ignored by other models.")
	generateCmd.Flags().StringArray("input-image", []string{}, "Reference image for editing/composition: local path (all providers) or https:// URL (Grok only). Repeatable. Caps: Grok 2.0=5, older Grok=3, Gemini 2.5 Flash=3, Gemini 3 Pro=11, Gemini 3.1 Flash/Lite=14.")
	generateCmd.Flags().String("user", "", "Optional end-user identifier for abuse monitoring (Grok/xAI only; sent as the API user field).")

	// Bind to viper for config file support
	viper.BindPFlag("generate.api", generateCmd.Flags().Lookup("api"))
	viper.BindPFlag("generate.model", generateCmd.Flags().Lookup("model"))
	viper.BindPFlag("generate.size", generateCmd.Flags().Lookup("size"))

	rootCmd.AddCommand(generateCmd)
}
