package mcp

// RegisterAllPrompts registers all gimage prompts with the MCP server
// These prompts teach LLMs how to use gimage tools through interactive examples
func RegisterAllPrompts(server *MCPServer) {
	// 1. Quick Start - Simplest possible workflow
	server.RegisterPrompt(Prompt{
		Name:        "gimage_quick_start",
		Title:       "Get Started with gimage",
		Description: "Learn how to generate your first AI image using the default Gemini Nano Banana 2.1 provider",
		Arguments:   []PromptArgument{},
		Template: `I want to generate an AI image using gimage. Here's how to get started:

STEP 1: Check available providers
Call list_models to see all providers with pricing:
- Default: Gemini Nano Banana 2.1 ($0.0225-$0.0756/image, native 4K)
- Fast option: Gemini 3.1 Flash Lite ($0.034/image, 1K)
- Pro option: Gemini 3 Pro ($0.134-$0.24/image, native 4K)
- Other options: Vertex AI presets, Grok Imagine 2.0

STEP 2: Generate a simple image
Call generate_image with:
- prompt: "a sunset over mountains"
- output: "~/Desktop/sunset.png"
- (defaults to gemini-nano-banana-2.1)

EXAMPLE:
generate_image(
  prompt="a sunset over mountains",
  output="~/Desktop/sunset.png"
)

Result will show:
- Output path
- Provider used
- Pricing per image

That's it! The image will be saved to your Desktop.`,
	})

	// 2. Generate with Style - Teach style parameters
	server.RegisterPrompt(Prompt{
		Name:        "generate_with_style",
		Title:       "Generate Images with Artistic Styles",
		Description: "Learn how to use style parameters to control the artistic rendering of generated images",
		Arguments: []PromptArgument{
			{Name: "subject", Description: "What to generate (e.g., 'a cat', 'a landscape')", Required: true},
			{Name: "style", Description: "Style preference: photorealistic, artistic, or anime", Required: false},
		},
		Template: `I want to generate {{subject}} with a specific artistic style.

RECOMMENDED APPROACH:
generate_image(
  prompt="{{subject}}, {{style}} style, high quality, detailed",
  style="{{style}}",
  output="~/Desktop/{{subject}}_{{style}}.png"
)

STYLE OPTIONS:
- photorealistic: For realistic photos
- artistic: For artistic/painterly renders
- anime: For anime/manga style

EXAMPLE:
generate_image(
  prompt="a futuristic city, photorealistic style, high quality, detailed",
  style="photorealistic",
  output="~/Desktop/city_photorealistic.png"
)

NOTE: Gemini Nano Banana 2.1 is the default and lowest-cost Google option for quick iterations ($0.0225 at 0.5K, $0.0336 at 1K).
Use model="vertex-nano-banana" when you need Vertex credentials/billing.`,
	})

	// 3. Generate and Crop Workflow - Multi-step workflow
	server.RegisterPrompt(Prompt{
		Name:        "generate_and_crop",
		Title:       "Generate and Crop for Hero Images",
		Description: "Learn the workflow to generate an image and crop it to specific dimensions (e.g., for hero images or banners)",
		Arguments: []PromptArgument{
			{Name: "description", Description: "What to generate", Required: true},
			{Name: "crop_width", Description: "Desired crop width in pixels", Required: true},
			{Name: "crop_height", Description: "Desired crop height in pixels", Required: true},
		},
		Template: `I want to generate {{description}} and crop it to {{crop_width}}x{{crop_height}} for a specific use case (like a hero image or banner).

WORKFLOW:
1. Generate the full image first
2. Crop to desired dimensions

STEP 1: Generate
generate_image(
  prompt="{{description}}",
  size="1024x1024",
  output="~/Desktop/temp_full.png"
)

STEP 2: Crop to {{crop_width}}x{{crop_height}}
crop_image(
  input="~/Desktop/temp_full.png",
  x=0,
  y=312,
  width={{crop_width}},
  height={{crop_height}},
  output="~/Desktop/final_cropped.png"
)

EXAMPLE (Hero Image 1024x400):
1. generate_image(prompt="abstract tech art", size="1024x1024", output="~/Desktop/temp.png")
2. crop_image(input="~/Desktop/temp.png", x=0, y=312, width=1024, height=400, output="~/Desktop/hero.png")

TIP: Use get_image_info first to verify actual image dimensions before cropping.`,
	})

	// 4. High Quality Generation - When to use paid providers
	server.RegisterPrompt(Prompt{
		Name:        "high_quality_image",
		Title:       "Generate High-Quality Professional Images",
		Description: "Learn when and how to use Gemini Nano Banana 2.1, Gemini 3 Pro, or Vertex presets for professional-quality images",
		Arguments: []PromptArgument{
			{Name: "subject", Description: "What to generate", Required: true},
		},
		Template: `I want to generate a high-quality image of {{subject}} for professional use.

STEP 1: Check providers with list_models
This shows all providers with pricing and capabilities:
- gemini/nano-banana-2.1: $0.0225-$0.0756/image, native 4K (default)
- gemini/flash-3.1-lite: $0.034/image, 1K only (fast Gemini image model)
- gemini/pro-3: $0.134-$0.24/image, native 4K
- vertex/nano-banana-2.1: Gemini Nano Banana 2.1 via Vertex, $0.0225-$0.0756/image by resolution
- grok/grok-imagine-2.0: $0.015-$0.07/image (xAI Grok Imagine 2.0)

STEP 2: Use Gemini Nano Banana 2.1 or Gemini 3 Pro at 4K for highest quality
generate_image(
  prompt="{{subject}}, ultra detailed, professional quality, 8k",
  model="gemini-nano-banana-2.1",
  image_size="4K",
  output="~/Desktop/{{subject}}_hq.png"
)

PROVIDER COMPARISON:
- Gemini Nano Banana 2.1 (gemini/nano-banana-2.1): Default, native 4K, $0.0225-$0.0756/image
- Gemini Flash Lite (gemini/flash-3.1-lite): Fast Gemini image, $0.034/image, 1K only
- Gemini 3 Pro (gemini/pro-3): Native 4K, $0.134-$0.24/image, sharp text
- vertex-nano-banana (vertex/nano-banana-2.1): Gemini Nano Banana 2.1 via Vertex

WHEN TO USE EACH:
- Gemini Nano Banana 2.1: Default for most use cases, great price/performance ($0.0336 at 1K, $0.0756 at 4K)
- Gemini 3 Pro: Complex multi-element compositions with deterministic seed support ($0.134-$0.24/image)
- vertex-nano-banana / vertex-flash presets: Vertex-authenticated workflows

EXAMPLE:
generate_image(
  prompt="professional headshot, studio lighting, ultra detailed, 8k",
  model="gemini-nano-banana-2.1",
  image_size="4K",
  output="~/Desktop/headshot_hq.png"
)

NOTE: The vertex-* presets require Vertex AI setup. Run 'gimage auth vertex' first.`,
	})

	// 5. Web Optimization Workflow - Multi-tool advanced workflow
	server.RegisterPrompt(Prompt{
		Name:        "optimize_for_web",
		Title:       "Generate and Optimize Images for Web",
		Description: "Learn how to generate images and optimize them for web use (smaller file size, WebP format)",
		Arguments: []PromptArgument{
			{Name: "count", Description: "Number of images to generate", Required: false},
		},
		Template: `I want to generate images and optimize them for web use (smaller file size, WebP format).

WORKFLOW:
1. Generate images with Gemini Nano Banana 2.1 for low-cost iteration
2. Resize to web-friendly dimensions
3. Convert to WebP (30% smaller than PNG)

EXAMPLE (3 images):

# Generate 3 variations
generate_image(prompt="tech artwork 1", output="~/Desktop/raw1.png")
generate_image(prompt="tech artwork 2", output="~/Desktop/raw2.png")
generate_image(prompt="tech artwork 3", output="~/Desktop/raw3.png")

# Resize to web dimensions
resize_image(input="~/Desktop/raw1.png", width=800, height=600, output="~/Desktop/resized1.png")
resize_image(input="~/Desktop/raw2.png", width=800, height=600, output="~/Desktop/resized2.png")
resize_image(input="~/Desktop/raw3.png", width=800, height=600, output="~/Desktop/resized3.png")

# Convert to WebP for smaller file size
convert_image(input="~/Desktop/resized1.png", format="webp", output="~/Desktop/web1.webp")
convert_image(input="~/Desktop/resized2.png", format="webp", output="~/Desktop/web2.webp")
convert_image(input="~/Desktop/resized3.png", format="webp", output="~/Desktop/web3.webp")

TIP: You can also use batch_resize, batch_compress, or batch_convert for multiple files at once!`,
	})

	// 6. Troubleshooting - Help with common errors
	server.RegisterPrompt(Prompt{
		Name:        "troubleshooting",
		Title:       "Troubleshoot Common gimage Errors",
		Description: "Learn how to fix common errors when using gimage tools",
		Arguments:   []PromptArgument{},
		Template: `I encountered an error when using gimage. Here are common issues and solutions:

ERROR: "Provider/model not found"
SOLUTION: Check available providers first
1. Call list_models to see all available providers
2. Use common aliases: "gemini", "nano-banana", "vertex-nano-banana", "grok"
✅ CORRECT: model="gemini" or model="gemini-nano-banana-2.1"
✅ CORRECT: model="vertex-nano-banana" (resolves to vertex/nano-banana-2.1 provider)

ERROR: "Gemini API key not configured"
SOLUTION: Set up authentication first
Run: gimage auth gemini
Then retry your generate_image call

ERROR: "Missing credentials for Gemini Nano Banana 2.1 via Vertex"
SOLUTION: Set up Vertex AI authentication
Run: gimage auth vertex
Requires: GCP project ID and service account or API key

ERROR: "crop region (x=0 + width=1792 = 1792) exceeds image width 1024"
SOLUTION: Check provider size limits
1. Call list_models to see max dimensions per provider
2. Gemini Flash Lite: up to 1024x1024
3. Gemini Nano Banana 2.1 / Gemini 3+: native resolution via image_size (0.5K/1K/2K/4K)
4. Use get_image_info to verify actual dimensions before cropping

ERROR: "unknown flag: --width" (when using crop)
SOLUTION: crop_image uses positional arguments, not flags
✅ CORRECT: crop_image(input="file.png", x=0, y=100, width=800, height=600)
❌ WRONG: crop_image(input="file.png", --width=800, --height=600)

PROVIDER SIZE LIMITS:
- gemini/flash-3.1-lite: up to 1024x1024
- gemini/nano-banana-2.1 and vertex/nano-banana-2.1: native resolution via image_size (0.5K/1K/2K/4K)

GENERAL TIPS:
1. Call list_models first to check providers, pricing, and auth status
2. Always specify output path (e.g., ~/Desktop/image.png)
3. Start with Gemini Nano Banana 2.1 for low-cost testing
4. Use get_image_info before cropping to verify dimensions
5. Check provider capabilities with list_models`,
	})
}
