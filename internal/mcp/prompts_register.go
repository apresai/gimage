package mcp

// RegisterAllPrompts registers all gimage prompts with the MCP server
// These prompts teach LLMs how to use gimage tools through interactive examples
func RegisterAllPrompts(server *MCPServer) {
	// 1. Quick Start - Simplest possible workflow
	server.RegisterPrompt(Prompt{
		Name:        "gimage_quick_start",
		Title:       "Get Started with gimage",
		Description: "Learn how to generate your first AI image using the affordable Gemini provider",
		Arguments:   []PromptArgument{},
		Template: `I want to generate an AI image using gimage. Here's how to get started:

STEP 1: Check available providers
Call list_models to see all providers with pricing:
- Affordable option: Gemini 2.5 Flash ($0.039/image)
- Default: Gemini 3 Pro ($0.134/image, native 4K)
- Paid options: Gemini 3.1 Flash via Vertex, Nova Canvas, Grok Imagine

STEP 2: Generate a simple image
Call generate_image with:
- prompt: "a sunset over mountains"
- output: "~/Desktop/sunset.png"
- (defaults to gemini-3-pro-image)

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

NOTE: Gemini Flash is the lowest-cost Google option for quick iterations.
Use model="vertex-flash" when you need Vertex credentials/billing; it runs Gemini 3.1 Flash through Vertex with tiered image pricing.`,
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
		Description: "Learn when and how to use Gemini 3 Pro or Vertex-backed Gemini 3.1 Flash for professional-quality images",
		Arguments: []PromptArgument{
			{Name: "subject", Description: "What to generate", Required: true},
		},
		Template: `I want to generate a high-quality image of {{subject}} for professional use.

STEP 1: Check providers with list_models
This shows all providers with pricing and capabilities:
- gemini/flash-2.5: $0.039/image, up to 1024x1024
- gemini/pro-3: $0.134-$0.24/image, native 4K (default)
- vertex/flash-3.1: Gemini 3.1 Flash via Vertex, $0.045-$0.151/image by resolution
- bedrock/nova-canvas: $0.08/image, up to 1408x1408

STEP 2: Use Gemini 3 Pro for highest quality text/diagram work, or vertex-flash for Vertex billing
generate_image(
  prompt="{{subject}}, ultra detailed, professional quality, 8k",
  model="gemini-3-pro-image",
  image_size="4K",
  output="~/Desktop/{{subject}}_hq.png"
)

PROVIDER COMPARISON:
- Gemini Flash (gemini/flash-2.5): Good quality, $0.039/image, best for iterations
- Gemini 3 Pro (gemini/pro-3): Native 4K, $0.134-$0.24/image, sharp text
- vertex-flash (vertex/flash-3.1): Gemini 3.1 Flash via Vertex, $0.045-$0.151/image by resolution
- Nova Canvas (bedrock/nova-canvas): High quality, $0.08/image, AWS integration

WHEN TO USE EACH:
- Gemini Flash: Quick iterations, testing prompts, social media ($0.039/image)
- Gemini 3 Pro: Text-heavy images, diagrams, native 4K ($0.134-$0.24/image)
- vertex-flash presets: Vertex-authenticated workflows
- Nova: AWS-integrated workflows, comparable quality to Gemini 3.1 Flash via Vertex

EXAMPLE:
generate_image(
  prompt="professional headshot, studio lighting, ultra detailed, 8k",
  model="gemini-3-pro-image",
  image_size="4K",
  output="~/Desktop/headshot_hq.png"
)

NOTE: The vertex-flash presets require Vertex AI setup. Run 'gimage auth vertex' first.
Nova Canvas requires AWS Bedrock setup. Run 'gimage auth bedrock' first.`,
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
1. Generate images with Gemini 2.5 Flash for low-cost iteration
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

TIP: You can also use batch_process_images for multiple files at once!`,
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
2. Use common aliases: "gemini", "vertex-flash", "nova-canvas"
✅ CORRECT: model="gemini" or model="gemini-2.5-flash-image"
✅ CORRECT: model="vertex-flash" (resolves to vertex/flash-3.1 provider)

ERROR: "Gemini API key not configured"
SOLUTION: Set up authentication first
Run: gimage auth gemini
Then retry your generate_image call

ERROR: "Missing credentials for Gemini 3.1 Flash via Vertex"
SOLUTION: Set up Vertex AI authentication
Run: gimage auth vertex
Requires: GCP project ID and service account or API key

ERROR: "crop region (x=0 + width=1792 = 1792) exceeds image width 1024"
SOLUTION: Check provider size limits
1. Call list_models to see max dimensions per provider
2. Gemini: up to 1024x1024
3. Gemini 3.1 Flash via Vertex (vertex-flash): native resolution via image_size (1K/2K/4K)
4. Use get_image_info to verify actual dimensions before cropping

ERROR: "unknown flag: --width" (when using crop)
SOLUTION: crop_image uses positional arguments, not flags
✅ CORRECT: crop_image(input="file.png", x=0, y=100, width=800, height=600)
❌ WRONG: crop_image(input="file.png", --width=800, --height=600)

PROVIDER SIZE LIMITS:
- gemini/flash-2.5: up to 1024x1024
- vertex/flash-3.1: native resolution via image_size (1K/2K/4K)
- bedrock/nova-canvas: up to 1408x1408

GENERAL TIPS:
1. Call list_models first to check providers, pricing, and auth status
2. Always specify output path (e.g., ~/Desktop/image.png)
3. Start with Gemini 2.5 Flash for lower-cost testing
4. Use get_image_info before cropping to verify dimensions
5. Check provider capabilities with list_models`,
	})
}
