package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/apresai/gimage/internal/config"
	"github.com/apresai/gimage/internal/generate"
	"github.com/apresai/gimage/internal/mcp"
	"github.com/apresai/gimage/internal/observability"
	"github.com/apresai/gimage/pkg/models"
)

func defaultThinkingLevelForProvider(provider *generate.Provider) string {
	if provider == nil {
		return ""
	}
	switch provider.ID {
	case "vertex/flash-3.1-fast":
		return "minimal"
	case "vertex/flash-3.1":
		return "medium"
	case "vertex/flash-3.1-ultra":
		return "high"
	default:
		return ""
	}
}

// RegisterGenerateImageTool registers the generate_image tool
func RegisterGenerateImageTool(server *mcp.MCPServer) {
	tool := mcp.Tool{
		Name:        "generate_image",
		Description: "Generate AI images using multiple providers (Gemini, Vertex AI, xAI Grok). Call list_models first to see available providers with pricing and capability flags. Quick start: generate_image(prompt='sunset', output='~/Desktop/sunset.png') uses the default Gemini provider. The 'count' parameter requests multiple images (Grok returns N exactly; the Gemini family is best-effort), saved as image_1.png, image_2.png, etc. Styles, seeds, image_size, and other options depend on provider capabilities; options the chosen provider does not support are reported in the response 'warning' field and ignored. IMPORTANT: Always specify output path (e.g., ~/Desktop/image.png).",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: false, // Creates new files but doesn't modify existing ones
			IdempotentHint:  false, // Each call generates a different image
			ReadOnlyHint:    false, // Writes files to disk
		},
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"prompt": map[string]interface{}{
					"type":        "string",
					"description": "Text description of the image to generate. Be specific and descriptive for best results.",
				},
				"output": map[string]interface{}{
					"type":        "string",
					"description": "Output file path. RECOMMENDED: Always specify a path like ~/Desktop/image.png or ~/Documents/image.png. If not provided, will try current directory first, then fall back to home directory. Supports tilde (~) expansion for home directory.",
				},
				"size": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"512x512", "1024x1024", "1024x1792", "1792x1024"},
					"description": "Image dimensions (WIDTHxHEIGHT). Default: 1024x1024. Provider limits: gemini/flash-2.5 up to 1024x1024. Gemini 3+ and the vertex-flash* presets (Gemini 3.1 Flash via Vertex) set native resolution via the image_size param (1K/2K/4K), not this field. Examples: '1024x1024' (square), '1792x1024' (16:9 landscape), '1024x1792' (9:16 portrait).",
					"default":     "1024x1024",
				},
				"model": map[string]interface{}{
					"type": "string",
					"enum": []string{
						"gemini-2.5-flash-image",
						"gemini-3-pro-image",
						"gemini-3",
						"gemini-3-pro",
						"gemini-3.1-flash-image",
						"gemini-3.1-flash",
						"gemini-3.1",
						"3.1-flash",
						"vertex-flash",
						"vertex-flash-fast",
						"vertex-flash-ultra",
						"gemini",
						"flash",
						"gemini-flash",
						"grok",
						"grok-imagine-image",
						"grok-imagine-image-quality",
						"grok-imagine",
						"grok-imagine-quality",
						"grok-quality",
						"xai",
						"aurora",
					},
					"description": "Provider/model to use. Call list_models to see all options with pricing. Common choices: 'gemini-flash' ($0.039/image, up to 1024x1024), 'gemini-3.1-flash' ($0.045-$0.151/image by resolution), 'gemini-3' or 'gemini-3-pro-image' ($0.134-$0.24/image, native 4K with sharp text), 'vertex-flash' (Gemini 3.1 Flash via Vertex, $0.045-$0.151/image by resolution, medium thinking default), 'vertex-flash-fast' (same backend, minimal thinking default), 'vertex-flash-ultra' (same backend, high thinking default), 'grok' ($0.02/image, xAI Grok Imagine), 'grok-imagine-quality' ($0.05 1K, $0.07 2K, replaces -pro retired 2026-05-15). Aliases automatically resolve to correct provider. Defaults to gemini-3-pro-image when no model is supplied; an explicitly invalid model returns an error.",
					"default":     "gemini-3-pro-image",
				},
				"image_size": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"1K", "2K", "4K"},
					"description": "Native image resolution for Gemini 3+ models (gemini-3-pro-image, gemini-3.1-flash-image). Supports '1K', '2K', or '4K' for native upscaling. Produces sharper text and diagrams at higher resolutions.",
				},
				"style": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"photorealistic", "artistic", "anime"},
					"description": "Image style. Affects rendering approach. Optional.",
				},
				"seed": map[string]interface{}{
					"type":        "integer",
					"description": "Random seed for reproducible generation (Gemini API providers). Use the same seed to get the same image.",
				},
				"aspect_ratio": map[string]interface{}{
					"type": "string",
					"enum": []string{
						"1:1", "16:9", "9:16", "4:3", "3:4", "3:2", "2:3", "5:4", "4:5",
						"2:1", "1:2", "19.5:9", "9:19.5", "20:9", "9:20", "auto",
					},
					"description": "Aspect ratio. Gemini 3+: 1:1, 16:9, 9:16, 4:3, 3:4, 3:2, 2:3, 5:4, 4:5. Grok Imagine also supports 2:1, 1:2, 19.5:9, 9:19.5, 20:9, 9:20, and auto. Overrides size.",
				},
				"count": map[string]interface{}{
					"type":        "integer",
					"description": "Number of images to generate (1-10). Grok returns N exactly; the Gemini family is best-effort and often returns one. Images are saved with _1, _2 suffixes.",
					"minimum":     1,
					"maximum":     10,
					"default":     1,
				},
				"output_format": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"png", "jpeg", "webp"},
					"description": "Output format for Vertex AI requests. Default: png.",
				},
				"thinking": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"minimal", "low", "medium", "high"},
					"description": "Reasoning depth for Gemini 3+ models. Higher = more planning before generation (slower, may improve layout/text). Ignored for Gemini 2.5 Flash and non-Gemini providers.",
				},
				"grounding": map[string]interface{}{
					"type":        "boolean",
					"description": "Enable Google Search grounding for Gemini 3+ models. Lets the model pull real-time web context (logos, current events) before generating. Billed per search query in addition to the per-image cost.",
					"default":     false,
				},
				"input_images": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Optional local file paths to reference images for compositional editing. Accepted formats: PNG, JPEG, WebP. Caps: Grok=3, Gemini 2.5 Flash=3, Gemini 3 Pro=11, Gemini 3.1 Flash=14. Grok uses POST /v1/images/edits; for multi-image Grok prompts you may reference <IMAGE_0>, <IMAGE_1>, etc.",
					"maxItems":    14,
				},
			},
			"required": []string{"prompt"},
		},
		Handler: func(args map[string]interface{}) (map[string]interface{}, error) {
			log := observability.NewVerboseLogger(observability.ComponentMCP)
			startTime := time.Now()

			// Extract and validate prompt
			prompt, ok := args["prompt"].(string)
			if !ok || prompt == "" {
				return nil, fmt.Errorf("prompt is required and must be a non-empty string")
			}

			log.Debug("generate_image tool invoked")

			// Extract optional parameters
			outputArg, _ := args["output"].(string)

			// Validate and fix output path BEFORE generating image
			// This avoids wasting API calls if the path is not writable
			defaultFilename := fmt.Sprintf("generated_%d.png", time.Now().Unix())
			pathResult, pathErr := ValidateAndFixOutputPath(outputArg, defaultFilename)
			if pathErr != nil {
				return nil, fmt.Errorf("output path validation failed: %w\n\nTIP: Try specifying an explicit output path like ~/Desktop/image.png or ~/Documents/image.png", pathErr)
			}
			output := pathResult.Path

			// Include warning in response if we had to fall back to a different location
			var pathWarning string
			if pathResult.Warning != "" {
				pathWarning = pathResult.Warning
			}

			size, _ := args["size"].(string)
			if size == "" {
				size = "1024x1024"
			}

			modelName, _ := args["model"].(string)
			userSpecifiedModel := strings.TrimSpace(modelName) != ""
			if !userSpecifiedModel {
				modelName = generate.DefaultModel
			}

			registry := generate.GetProviderRegistry()
			selectedProvider, providerErr := registry.ResolveProvider(modelName)
			if providerErr != nil {
				if userSpecifiedModel {
					// The caller explicitly named a model that failed to resolve.
					// Surface the resolve error (which carries guidance for retired
					// aliases) instead of silently substituting the default, so an
					// invalid model is a hard error rather than a surprise
					// generation with a different model.
					return nil, providerErr
				}
				// No model supplied: fall back to the default model.
				log.Debug("No model supplied, falling back to default %s", generate.DefaultModel)
				selectedProvider, providerErr = registry.ResolveProvider(generate.DefaultModel)
				if providerErr != nil {
					return nil, fmt.Errorf("default provider not found: %w", providerErr)
				}
			}
			modelName = selectedProvider.ModelID

			style, _ := args["style"].(string)
			imageSize, _ := args["image_size"].(string)     // For Gemini 3+ (Pro/3.1 Flash): 1K, 2K, 4K
			aspectRatio, _ := args["aspect_ratio"].(string) // For Gemini 3+ and Grok Imagine
			outputFormat, _ := args["output_format"].(string)
			thinkingLevel, _ := args["thinking"].(string)
			grounding, _ := args["grounding"].(bool)
			inputImages := coerceStringArray(args["input_images"])
			if thinkingLevel == "" {
				thinkingLevel = defaultThinkingLevelForProvider(selectedProvider)
			}

			// Parse optional numeric parameters with type coercion (accepts string or number)
			var seed int64
			if args["seed"] != nil {
				if seedVal, sErr := coerceToInt(args["seed"], "seed"); sErr == nil {
					seed = int64(seedVal)
				}
			}

			var count int
			if args["count"] != nil {
				if countVal, cErr := coerceToInt(args["count"], "count"); cErr == nil {
					count = countVal
				}
			}

			// Create generate options
			opts := models.GenerateOptions{
				Model:              modelName,
				Size:               size,
				Style:              style,
				Seed:               seed,
				ImageSize:          imageSize,   // Native resolution for Gemini 3+ (Pro/3.1 Flash)
				AspectRatio:        aspectRatio, // For Gemini 3+ and Grok Imagine
				NumberOfImages:     count,
				OutputFormat:       outputFormat,
				ThinkingLevel:      thinkingLevel,
				WebSearchGrounding: grounding,
				InputImages:        inputImages,
			}

			// Strip options the resolved provider does not support and collect a
			// friendly warning for each. Surfaced to the caller via result["warning"].
			featureWarning := strings.Join(generate.ReconcileOptions(selectedProvider, &opts), "; ")

			log.LogGenerationStart(prompt, map[string]interface{}{
				"model":              modelName,
				"size":               size,
				"style":              opts.Style,
				"image_size":         opts.ImageSize,
				"aspect_ratio":       opts.AspectRatio,
				"seed":               opts.Seed,
				"thinking":           opts.ThinkingLevel,
				"grounding":          opts.WebSearchGrounding,
				"input_images_count": len(opts.InputImages),
			})

			// Determine backend from the resolved provider. This preserves aliases
			// like vertex-flash that target a Gemini model through Vertex.
			selectedAPI := selectedProvider.API
			log.Debug("Selected API: %s", selectedAPI)

			// Create context for API calls
			ctx := context.Background()

			// Generate based on backend
			var generatedImages []*models.GeneratedImage

			if selectedAPI == "gemini" {
				// Use Gemini REST client
				apiKey, apiErr := config.GetGeminiAPIKey("")
				if apiErr != nil {
					return nil, fmt.Errorf("Gemini API key not configured: %w\nPlease run: gimage auth gemini", apiErr)
				}

				client, err := generate.NewGeminiRESTClient(apiKey)
				if err != nil {
					return nil, fmt.Errorf("failed to create Gemini client: %w", err)
				}
				defer client.Close()

				generatedImages, err = client.GenerateImage(ctx, prompt, opts)
				if err != nil {
					return nil, fmt.Errorf("image generation failed: %w", err)
				}
			} else if selectedAPI == "vertex" {
				// Use Vertex AI
				// Load config to get project and location
				cfg, err := config.LoadConfig()
				if err != nil {
					return nil, fmt.Errorf("failed to load config: %w", err)
				}

				project := cfg.VertexProject
				location := cfg.VertexLocation
				if location == "" {
					location = "us-central1"
				}

				// Check for Express Mode (API key) first
				vertexAPIKey, _ := config.GetVertexAPIKey("")

				if vertexAPIKey != "" {
					// Express Mode - Use REST client
					client, err := generate.NewVertexRESTClient(vertexAPIKey, project, location)
					if err != nil {
						return nil, fmt.Errorf("failed to create Vertex AI REST client: %w", err)
					}
					defer client.Close()

					generatedImages, err = client.GenerateImage(ctx, prompt, opts)
					if err != nil {
						return nil, fmt.Errorf("image generation failed: %w", err)
					}
				} else {
					// Full Mode - Use unified SDK client
					client, err := generate.NewVertexUnifiedClient(ctx, project, location)
					if err != nil {
						return nil, fmt.Errorf("failed to create Vertex AI unified client: %w\nPlease run: gimage auth vertex", err)
					}
					defer client.Close()

					generatedImages, err = client.GenerateImage(ctx, prompt, opts)
					if err != nil {
						return nil, fmt.Errorf("image generation failed: %w", err)
					}
				}
			} else if selectedAPI == "grok" {
				// Use xAI Grok
				cfg, _ := config.LoadConfig()
				apiKey := ""
				if cfg != nil {
					apiKey = cfg.GrokAPIKey
				}
				// Also check environment variable directly
				if apiKey == "" {
					apiKey = os.Getenv("GROK_API_KEY")
				}
				if apiKey == "" {
					return nil, fmt.Errorf("Grok API key not configured\nPlease set GROK_API_KEY environment variable or run: gimage auth grok")
				}

				client, err := generate.NewGrokClient(apiKey)
				if err != nil {
					return nil, fmt.Errorf("failed to create Grok client: %w", err)
				}
				defer client.Close()

				generatedImages, err = client.GenerateImage(ctx, prompt, opts)
				if err != nil {
					return nil, fmt.Errorf("image generation failed: %w", err)
				}
			} else {
				return nil, fmt.Errorf("unsupported API: %s", selectedAPI)
			}

			// Save the generated image(s)
			var savedPaths []string
			for i, img := range generatedImages {
				// Determine output path for this specific image
				var currentOutput string
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

				if err := generate.SaveImage(img, currentOutput); err != nil {
					return nil, fmt.Errorf("failed to save image %d: %w", i+1, err)
				}

				// Get absolute output path
				absOutput, err := filepath.Abs(currentOutput)
				if err != nil {
					absOutput = currentOutput
				}
				savedPaths = append(savedPaths, absOutput)
			}

			// Get provider info for pricing display using shared helper
			var modelDisplayName string
			var pricingInfo string

			if selectedProvider != nil {
				modelDisplayName = selectedProvider.Name
				// Use shared pricing helper for consistent pricing display
				pricing := generate.GetProviderPricing(selectedProvider, imageSize, size, style)
				pricingInfo = pricing.Display
			} else {
				modelDisplayName = modelName
				pricingInfo = "Unknown"
			}

			// Build result with comprehensive information
			result := map[string]interface{}{
				"success":       true,
				"output_path":   savedPaths[0],
				"saved_paths":   savedPaths,
				"count":         len(generatedImages),
				"size":          size,
				"model":         modelName,
				"model_display": modelDisplayName,
				"api":           selectedAPI,
				"pricing":       pricingInfo,
				"prompt":        prompt,
			}

			// Create user-friendly message
			msg := fmt.Sprintf("Generated %d image(s) using %s (%s)", len(generatedImages), modelDisplayName, pricingInfo)
			result["message"] = msg

			// Add warning if we had to fall back to a different location
			if pathWarning != "" {
				result["warning"] = pathWarning
			}
			if featureWarning != "" {
				if existing, ok := result["warning"].(string); ok && existing != "" {
					result["warning"] = existing + "; " + featureWarning
				} else {
					result["warning"] = featureWarning
				}
			}

			log.Debug("Generation complete in %s: %d images saved", time.Since(startTime), len(generatedImages))

			return result, nil
		},
	}

	server.RegisterTool(tool)
}
