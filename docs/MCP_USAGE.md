# Using Gimage with AI Assistants

Complete guide to using the gimage MCP server with Claude and other AI assistants.

## Quick Start

### 1. Install gimage MCP server

**Option A: npm (Recommended)**

```bash
npm install -g @apresai/gimage-mcp
```

**Option B: Homebrew**

```bash
brew install apresai/tap/gimage
```

**Option C: Manual Download**

Download from [GitHub Releases](https://github.com/apresai/gimage/releases)

### 2. Configure API credentials

Before using the MCP server, you need to set up authentication for AI image generation.

**Recommended: Use environment variables** for better security:

```bash
export GEMINI_API_KEY="your-api-key-here"
```

**Alternative: Interactive setup**:

```bash
# Check current authentication status
gimage auth status

# Interactive setup wizard
gimage auth setup

# Test your credentials
gimage auth test
```

**Get your API key**: https://aistudio.google.com/app/apikey

Gemini API pricing:

- Gemini 2.5 Flash: $0.039/image (most affordable)
- Gemini 3 Pro: $0.134/image (1K/2K), $0.24/image (4K)

**For advanced users**:

- **Vertex AI**: 3 authentication modes (Express/Service Account/ADC)
- **AWS Bedrock**: 4 authentication modes (Bearer Token/Access Keys/Profile/IAM Role)
- **xAI Grok**: `GROK_API_KEY` environment variable (Grok Imagine $0.02/image, Quality $0.05-$0.07/image)

See [Authentication Guide](../README.md#configuration) for complete details.

### 3. Add to Claude Desktop

Edit your Claude Desktop MCP configuration file:

**Configuration File Locations:**

- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Linux**: `~/.config/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

**Add this configuration:**

```json
{
  "mcpServers": {
    "gimage": {
      "command": "npx",
      "args": ["-y", "@apresai/gimage-mcp"]
    }
  }
}
```

**If using Homebrew installation:**

```json
{
  "mcpServers": {
    "gimage": {
      "command": "gimage",
      "args": ["serve"]
    }
  }
}
```

### 4. Restart Claude Desktop

Quit Claude Desktop completely and reopen it. The MCP server will start automatically.

### 5. Start Using!

Try these prompts in Claude:

**Image Generation:**

- "Generate an image of a sunset over mountains"
- "Create a photorealistic portrait of a wise old wizard"
- "Generate an anime-style image of cherry blossoms in spring"

**Image Processing:**

- "Resize photo.jpg to 800x600 pixels"
- "Compress all images in my photos folder to save space"
- "Convert image.png to WebP format for web use"

---

## Available Operations

### AI Image Generation

Generate stunning images from text descriptions using state-of-the-art AI models.

**Capabilities:**

- **Multiple AI models**: Gemini 3 Pro (default), Gemini 3.1 Flash, Gemini 2.5 Flash ($0.039/image), Gemini 3.1 Flash via Vertex (vertex-flash), Nova Canvas, Grok Imagine/Quality
- **Size options**: Standard sizes from 512x512 to 1792x1024; native 1K/2K/4K with Gemini 3+ models via `image_size`; up to 1408x1408 with Nova Canvas
- **Style controls**: Photorealistic, artistic, anime
- **Negative prompts**: Exclude unwanted elements
- **Reproducible results**: Use seeds for consistent generation
- **Advanced controls**:
  - Aspect ratio (Gemini 3 Pro: 1:1, 16:9, 9:16, 4:3, 3:4, 5:4, 4:5, 3:2, 2:3)
  - Native resolution (Gemini 3 Pro / 3.1 Flash: 1K/2K/4K; Grok Imagine / Quality: 1K/2K only)
  - Output format (Vertex AI: PNG, JPEG, WebP)
  - CFG scale (Bedrock: 1.0-10.0 for creativity control)
  - Batch generation (max count varies by provider)
- **Gemini 3+ exclusives** (ignored by other providers):
  - `thinking` (`minimal|low|medium|high`): reasoning depth before generation
  - `grounding` (bool): enables Google Search grounding (billed per search query)
  - `input_images` (array of file paths): reference images for compositional editing — Nano Banana style. Caps: 2.5 Flash=3, 3 Pro=11, 3.1 Flash=14

**Example Prompts:**

```
"Generate a 1024x1024 photorealistic image of a medieval castle on a hill"
"Create an artistic interpretation of a futuristic city at night"
"Generate an anime-style character with blue hair and a sword"
"Make me an image of a sunset, but avoid showing any people or buildings"
"Generate a 16:9 wide landscape using Gemini 3 Pro"
"Create 3 variations of a fantasy dragon in flight"
"Generate an abstract art image with maximum creativity (CFG scale 10)"
"Generate a portrait in WebP format for web optimization"
"Drop this product on a marble counter under soft studio light" (with input_images=["product.png"])
"Place this character in this environment, cinematic lighting" (with input_images=["character.png","scene.jpg"])
"Generate an infographic showing Q3 revenue with sharp text" (with thinking="high", image_size="4K")
"Generate the official Nintendo Switch 2 console" (with grounding=true)
```

### 🖼️ Image Processing

#### Resize

Change image dimensions with flexible aspect ratio handling. Three modes available:
- **crop** (default): Fills target size and crops excess, preserving aspect ratio
- **fit**: Fits within target bounds, may be smaller than target
- **stretch**: Forces exact dimensions, may distort

**Example Prompts:**

```
"Resize landscape.jpg to 1920x1080" (uses crop mode by default)
"Resize photo.png to 800x600 using fit mode to preserve aspect ratio"
"Resize avatar.jpg to exactly 100x100 using stretch mode"
"Create thumbnails at 200x200 using crop mode for consistent sizes"
```

#### Scale

Resize image proportionally by a factor.

**Example Prompts:**

```
"Scale photo.jpg to half its size"
"Make image.png twice as large"
"Reduce all dimensions by 75% (use factor 0.25)"
```

#### Crop

Extract a specific region from an image.

**Example Prompts:**

```
"Crop photo.jpg starting at position (100, 100) with width 800 and height 600"
"Extract a 500x500 square from the center of the image"
```

#### Compress

Reduce file size while maintaining quality.

**Example Prompts:**

```
"Compress photo.jpg to 85% quality to save space"
"Reduce the file size of all images in this directory"
```

#### Convert

Change image format (PNG, JPG, WebP, GIF, TIFF, BMP).

**Example Prompts:**

```
"Convert photo.png to JPEG format"
"Change all images to WebP for better web performance"
```

### ⚡ Batch Operations

Process multiple images concurrently for efficient workflows.

**Example Prompts:**

```
"Resize all images in the vacation-photos folder to 1920x1080" (uses crop mode by default)
"Resize all images in gallery/ to 800x600 using fit mode to preserve aspect ratios"
"Compress every image in my-photos directory to 85% quality"
"Convert all PNG files in this directory to WebP"
```

**Features:**

- Concurrent processing with multiple CPU cores
- Progress reporting
- Error handling (continues even if some files fail)
- Preserves directory structure
- Resize mode support (crop, fit, stretch)

---

## Common Workflows

### Web Image Optimization

**Scenario**: You have a folder of photos that need to be optimized for web use.

**Prompt**:

```
"I have photos in the 'website-images' folder. Please:
1. Resize them all to a maximum of 1920x1080
2. Compress them to 85% quality
3. Convert them to WebP format
Save the results in 'optimized-images' folder"
```

Claude will execute these operations in sequence.

### Social Media Preparation

**Scenario**: Prepare an image for Instagram.

**Prompt**:

```
"Take photo.jpg and prepare it for Instagram:
- Crop it to a square (1080x1080) from the center
- Compress to 90% quality
Save it as instagram.jpg"
```

### E-commerce Product Images

**Scenario**: Generate multiple sizes of a product image.

**Prompt**:

```
"I have product.jpg. Create three versions:
1. Large: 1200x1200 (save as product-large.jpg)
2. Medium: 600x600 (save as product-medium.jpg)
3. Thumbnail: 200x200 (save as product-thumb.jpg)
All should be compressed to 90% quality"
```

### AI Content Generation

**Scenario**: Generate multiple variations of an image.

**Prompt**:

```
"Generate 3 different versions of a fantasy landscape:
1. A photorealistic mountain scene with a lake
2. An artistic interpretation with vibrant colors
3. An anime-style version with dramatic clouds
Use 1024x1024 size for all"
```

### Generate and Optimize for Web

**Scenario**: Generate an AI image and convert to WebP for web use (90%+ file size reduction).

**Prompt**:

```
"Generate a hero image of a futuristic city skyline at sunset.
Then convert it to WebP format for my website."
```

Claude will use `generate_image` followed by `convert_image` to produce an optimized WebP file. AI-generated PNGs are typically 1-3 MB; WebP at quality 85 reduces this to 100-300 KB with no visible quality loss.

---

## Troubleshooting

### MCP Server Not Connecting

**Symptoms**: Claude shows no gimage tools available

**Solutions**:

1. **Verify gimage is installed**

   ```bash
   which gimage
   ```

   Should show the path to gimage binary.

2. **Test manual startup**

   ```bash
   gimage serve
   ```

   Should start without errors. Press Ctrl+C to stop.

3. **Check Claude Desktop configuration**

   - Verify JSON syntax is correct
   - Ensure file path matches your system
   - Check for typos in command/args

4. **Restart Claude Desktop**

   - Quit completely (not just close window)
   - Reopen application

5. **Check Claude logs**
   - macOS: `~/Library/Logs/Claude/`
   - Look for error messages related to gimage

### Image Generation Fails

**Symptoms**: Error messages when trying to generate images

**Solutions**:

1. **Check authentication status**

   ```bash
   gimage auth status
   ```

   This shows which credentials are configured and their sources.

2. **Verify API key is configured**

   ```bash
   # Use interactive setup
   gimage auth setup

   # OR set environment variable (recommended)
   export GEMINI_API_KEY="your-api-key-here"
   ```

3. **Test credentials**

   ```bash
   gimage auth test
   ```

   This makes real API calls to verify your credentials work.

4. **Test generation manually**

   ```bash
   gimage generate "test image"
   ```

   If this works, MCP server should work too.

5. **Check API key validity**

   - Ensure key hasn't expired
   - Verify it's correctly copied (no extra spaces)
   - Get a new key if needed: https://aistudio.google.com/app/apikey

6. **Verify internet connection**
   - Gemini API requires internet access
   - Check firewall settings

### Permission Errors

**Symptoms**: "Permission denied" when processing images

**Solutions**:

1. **Check file permissions**

   ```bash
   ls -la /path/to/image.jpg
   ```

2. **Ensure write access to output directory**

   ```bash
   mkdir -p output-folder
   chmod 755 output-folder
   ```

3. **Use absolute paths**
   Instead of: `photo.jpg`
   Use: `/Users/yourname/photos/photo.jpg`

### Batch Operations Slow

**Symptoms**: Batch processing takes too long

**Solutions**:

1. **Increase workers**
   "Process all images in photos/ using 8 workers for faster processing"

2. **Process in smaller batches**
   Split large directories into smaller chunks

3. **Check disk space**
   Ensure sufficient space for output files

---

## Advanced Usage

### Environment Variables

You can override configuration with environment variables:

```bash
# Gemini API
export GEMINI_API_KEY="your-key-here"

# Vertex AI Express Mode
export VERTEX_API_KEY="your-vertex-key"
export VERTEX_PROJECT="your-gcp-project"
export VERTEX_LOCATION="us-central1"

# Vertex AI Full Mode (Service Account)
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account.json"
export VERTEX_PROJECT="your-gcp-project"
```

### Using Different AI Models

**Gemini Models**:

- `gemini-3-pro-image` (default, native 4K, sharp text, $0.134-$0.24/image)
- `gemini-3.1-flash-image` (native 4K at flash speed, tiered pricing: $0.045-$0.151/image)
- `gemini-2.5-flash-image` ($0.039/image, up to 1024x1024)

**Vertex AI Models** (Gemini 3.1 Flash via Vertex):

- `vertex-flash` (`vertex/flash-3.1`, medium thinking default)
- `vertex-flash-fast` (`vertex/flash-3.1-fast`, minimal thinking default)
- `vertex-flash-ultra` (`vertex/flash-3.1-ultra`, high thinking default)
- Pricing follows Gemini 3.1 Flash tiers: $0.045 (0.5K), $0.067 (1K), $0.101 (2K), $0.151 (4K)
- Negative prompts and seeds are not supported by Gemini 3.1 Flash via Vertex.

**AWS Bedrock Models**:

- `amazon.nova-canvas-v1:0` ($0.04-$0.08/image, up to 1408x1408)

**xAI Grok Models**:

- `grok-imagine-image` (fast, $0.02/image, supports aspect ratio and resolution)
- `grok-imagine-image-quality` (quality tier, $0.05 at 1K, $0.07 at 2K; replaces deprecated `-pro` retired by xAI 2026-05-15)

**Examples**:

```
"Generate an image using vertex-flash-ultra with image_size 4K showing a hyper-realistic dragon"
"Generate a 4K infographic using Gemini 3 Pro with image_size 4K"
"Generate an image using gemini-2.5-flash-image for affordable generation"
"Generate a 16:9 landscape using Gemini 3 Pro aspect ratio control"
"Generate an image in WebP format using Vertex AI for smaller file size"
"Generate an abstract image with CFG scale 10 using Nova Canvas for maximum creativity"
```

### Reproducible Generation

Use seeds for consistent results:

```
"Generate an image with seed 12345 of a mountain landscape"
"Generate the same image again with seed 12345 to verify it's identical"
```

### Complex Multi-Step Workflows

Claude can chain multiple operations:

```
"Please help me with this workflow:
1. Generate an AI image of a fantasy castle (1024x1024)
2. Create 3 different sizes: 1536x1536, 1024x1024, 512x512
3. Compress all of them to 90% quality
4. Convert to WebP format
5. Save in a folder called 'castle-variants'"
```

---

## Tips for Best Results

### Image Generation

1. **Be specific** - Detailed prompts produce better results

   - ✅ "A photorealistic sunset over mountain peaks with orange and purple sky, reflection in a calm lake"
   - ❌ "Sunset"

2. **Use style keywords** - Specify the artistic style

   - "photorealistic", "artistic", "anime", "painting", "sketch", etc.

3. **Use negative prompts** - Exclude unwanted elements

   - "Generate a forest scene, but avoid showing any people, buildings, or modern objects"

4. **Try different models** - Each has strengths

   - Gemini 2.5 Flash: $0.039/image, fast, great for quick iterations
   - Gemini 3 Pro: Best for text, diagrams, native 4K, aspect ratio control
   - vertex-flash: Gemini 3.1 Flash via Vertex, $0.045-$0.151/image by resolution
   - vertex-flash-fast: same Vertex-backed Gemini model with minimal thinking default
   - vertex-flash-ultra: same Vertex-backed Gemini model with high thinking default
   - Nova Canvas: AWS integration, CFG scale for creativity control
   - Grok Imagine: Fast, affordable generation via xAI ($0.02/image)
   - Grok Imagine Quality: Higher quality xAI generation ($0.05 at 1K, $0.07 at 2K; replaces deprecated `-pro`)

5. **Use provider-specific features**:

   - Gemini 3 Pro: Aspect ratio (16:9 for landscapes) and native resolution (4K for detail)
   - Vertex AI: Output format (WebP for web optimization)
   - Bedrock: CFG scale (higher values for more creative/abstract results)

6. **Batch generation** - Use count parameter for variations
   - "Generate 3 variations of..." instead of calling multiple times
   - Faster and more efficient than sequential generation

### Image Processing

1. **Know your target** - Understand final use case

   - Web: 1920x1080 or smaller, WebP format, 85% quality
   - Print: Larger sizes, PNG/TIFF, 95-100% quality
   - Mobile: 750x1334, 80% quality

2. **Batch similar operations** - More efficient

   - Process all images in a folder at once
   - Use consistent settings across related images

3. **Preserve originals** - Always work on copies

   - Use explicit output paths
   - Create separate directories for processed images

4. **Check results** - Verify quality meets needs
   - View processed images before deleting originals
   - Adjust quality settings if needed

---

## Next Steps

- **Explore all tools**: Try each operation to understand capabilities
- **Create workflows**: Combine operations for your specific needs
- **Share examples**: Help others by sharing successful prompts
- **Report issues**: Help improve the tool by reporting bugs

---

## Support

- **Documentation**: [Complete Tools Reference](MCP_TOOLS.md)
- **Examples**: [Real-World Examples](MCP_EXAMPLES.md)
- **GitHub**: https://github.com/apresai/gimage
- **Issues**: https://github.com/apresai/gimage/issues
