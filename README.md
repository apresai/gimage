# Gimage - AI-Powered Image Generation and Processing

**Gimage** is a powerful tool for generating AI images and processing them with ease. Built with pure Go for maximum portability.

## 🚀 Three Ways to Use Gimage

### 1. Command-Line Tool (CLI)

A single binary with zero dependencies for local image operations on your computer.

### 2. MCP Server for AI Assistants

Run as an MCP (Model Context Protocol) server for seamless integration with Claude Desktop and other AI assistants.

### 3. Cloud API (AWS Lambda)

Production-ready serverless REST API for web applications and remote processing.

## What Can You Do with Gimage?

### 🎨 AI Image Generation

- Generate stunning images from text prompts using Google Gemini, Vertex AI, AWS Bedrock, or xAI Grok
- Multiple AI models: Gemini 3 Pro (default, native 4K), Gemini 2.5 Flash ($0.039/image), Imagen 4/Fast/Ultra, Nova Canvas, Grok Imagine ($0.02-0.07/image)
- Control size, style, quality, aspect ratio, and use negative prompts
- Reproducible results with seed values
- Provider-specific features: CFG scale (Bedrock), output format (Vertex AI), native resolution (Gemini 3 Pro)
- Batch generation: create multiple images at once

### 🛠️ Image Processing

- **Resize** - Change image dimensions while preserving aspect ratio (crop or fit)
- **Scale** - Scale images by factor (2x, 0.5x, etc.)
- **Crop** - Extract specific regions from images
- **Compress** - Reduce file size while maintaining quality
- **Convert** - Transform between formats (PNG, JPG, WebP, GIF, TIFF, BMP)

### ⚡ Batch Processing (MCP Server Only)

- Process multiple images concurrently via MCP server
- Optimized for AI assistants (Claude Desktop)
- CLI users: use shell scripts or `find` + `xargs`

### 🔌 Integration Options

- **Claude Desktop**: Run as MCP server
- **Web Applications**: REST API via AWS Lambda
- **Serverless**: AWS Lambda on ARM64/Graviton2

## Quick Start - CLI

### 1. Installation

#### Homebrew (macOS/Linux) - Recommended

```bash
# Install gimage (tap is added automatically)
brew install apresai/tap/gimage
```

#### Upgrade via Homebrew

```bash
brew upgrade apresai/tap/gimage
```

#### Manual Installation

Download the latest release for your platform:

```bash
# macOS (Intel)
curl -L https://github.com/apresai/gimage/releases/latest/download/gimage-darwin-amd64 -o gimage
chmod +x gimage
sudo mv gimage /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/apresai/gimage/releases/latest/download/gimage-darwin-arm64 -o gimage
chmod +x gimage
sudo mv gimage /usr/local/bin/

# Linux
curl -L https://github.com/apresai/gimage/releases/latest/download/gimage-linux-amd64 -o gimage
chmod +x gimage
sudo mv gimage /usr/local/bin/
```

### 2. Setup Authentication

**Quick Start** (recommended for most users):

```bash
# Get a Gemini API key from https://aistudio.google.com/app/apikey
# Then run the interactive setup:
gimage auth setup gemini

# Or use environment variable (more secure):
export GEMINI_API_KEY="your-api-key-here"
gimage generate "test sunset"
```

**Check your configuration:**

```bash
# Check current authentication status
gimage auth status

# List all configured providers with pricing
gimage auth list

# Test your credentials
gimage auth test gemini
```

**Get API Keys:**

- **Gemini API**: https://aistudio.google.com/app/apikey (Gemini Flash $0.039/image, Gemini 3 Pro $0.134/image)
- **Vertex AI**: https://cloud.google.com/vertex-ai (3 auth modes: Express Mode/Service Account/Application Default Credentials)
- **AWS Bedrock**: https://console.aws.amazon.com/bedrock (4 auth modes: Bearer Token/Access Keys/Profile/IAM Role)
- **xAI Grok**: https://console.x.ai (Grok Imagine $0.02/image, Grok Imagine Pro $0.07/image)

### 3. Generate Your First Image

```bash
gimage generate "a sunset over mountains with vibrant colors"
```

That's it! Your image will be saved as `generated_<timestamp>.png`

## Examples

### Image Generation

```bash
# Basic generation (positional prompt - most common)
gimage generate "futuristic city at night"

# Specify size and style
gimage generate "abstract art" --size 1024x1024 --style photorealistic

# Use Vertex AI Imagen 4 (auto-detects vertex API)
gimage generate "beautiful landscape" --model imagen-4

# Use AWS Bedrock Nova Canvas with premium quality (via style flag)
gimage generate "futuristic robot" --model nova-canvas --style premium

# Use negative prompts to avoid unwanted elements
gimage generate "forest scene" --negative "people, buildings"

# Reproducible results with seed
gimage generate "random pattern" --seed 12345

# Control creativity with CFG scale (Bedrock Nova Canvas, 1.0-10.0)
gimage generate "abstract art" --model nova-canvas --cfg-scale 10

# Generate multiple images at once
gimage generate "fantasy landscape" --count 3

# Use aspect ratio with Gemini 3 Pro (1:1, 16:9, 9:16, 4:3, 3:4, 5:4, 4:5, 3:2, 2:3)
gimage generate "wide landscape" --model gemini-3-pro --aspect-ratio 16:9

# Control output format with Vertex AI (png, jpeg, webp)
gimage generate "portrait" --api vertex --output-format webp

# List all available models with pricing
gimage generate --list-models

# Or use explicit --prompt flag if preferred
gimage generate --prompt "your prompt here"

# Pro tip: Convert generated images to WebP for 90%+ size reduction
gimage generate "sunset over mountains" -o sunset.png
gimage convert --input sunset.png --format webp
# Or use cwebp directly for maximum control:
# cwebp -q 85 -mt -sharp_yuv -preset photo sunset.png -o sunset.webp
```

> **Tip: Always convert to WebP** — AI-generated PNGs are typically 1-3 MB. Converting to WebP at quality 85 reduces file size by 90%+ with no visible quality loss. Use `gimage convert --format webp` or `cwebp -q 85 -mt -sharp_yuv -preset photo` for best results.

### Image Processing

All image processing commands use explicit flags for clarity:

```bash
# Resize to specific dimensions (strictly preserves aspect ratio)
# --mode crop (default): fill target size, crop excess
# --mode fit: fit within target, may have borders
gimage resize --input photo.jpg --width 800 --height 600
gimage resize --input photo.jpg --width 800 --height 600 --mode fit

# Scale to 50% size
gimage scale --input photo.jpg --factor 0.5

# Crop a region
gimage crop --input photo.jpg --x 100 --y 100 --width 800 --height 600

# Compress with custom quality (supports JPG and WebP)
gimage compress --input photo.jpg --quality 85

# Convert format
gimage convert --input photo.png --format jpg

# Use --output to specify custom output path
gimage resize --input photo.jpg --width 800 --height 600 --output resized.jpg

# Add --verbose for detailed progress
gimage convert --input photo.png --format webp --verbose
```

### Batch Processing

**Batch operations are available only through the MCP server** for use with AI assistants like Claude.

For CLI users who need batch processing:

- Use shell scripts with loops
- Use `find` + `xargs` for parallel processing
- Example: `find photos/ -name "*.jpg" | xargs -P 4 -I {} gimage resize --input {} --width 800 --height 600`

MCP server batch tools (for AI assistants):

- `batch_resize` - Concurrent image resizing
- `batch_compress` - Concurrent compression
- `batch_convert` - Concurrent format conversion

See [MCP documentation](#mcp-server-for-ai-assistants) for details.

## Configuration

### Authentication Priority (Highest to Lowest)

1. **Command-line flags** (highest priority)
2. **Environment variables** (`GEMINI_API_KEY`, `VERTEX_API_KEY`, etc.)
3. **Config file** (`~/.gimage/config.md`)
4. **Default values** (lowest priority)

**Recommended**: Use environment variables for sensitive keys. Config file stores keys in **plaintext**.

### Config File

Location: `~/.gimage/config.md` (created automatically by `gimage auth` commands)

**Security Notes**:

- File permissions: 0600 (only you can read/write)
- Contains PLAINTEXT API keys
- **NEVER commit to version control**
- **PREFER environment variables** over config file
- Use `gimage auth status` to check for conflicting credentials

Format (markdown with `**key**: value`):

```markdown
# Gimage Configuration

⚠️ SECURITY WARNING ⚠️
This file contains SENSITIVE API KEYS stored in PLAINTEXT.

**gemini_api_key**: your-gemini-key
**vertex_api_key**: your-vertex-key
**vertex_project**: your-gcp-project
**vertex_location**: us-central1
**vertex_credentials_path**: /path/to/service-account.json
**aws_access_key_id**: AKIA...
**aws_secret_access_key**: wJalr...
**aws_region**: us-east-1
**aws_profile**: default
**aws_bedrock_api_key**: bearer-token
**grok_api_key**: xai-5zM...
**default_api**: gemini
**default_model**: gemini-3-pro-image-preview
**default_size**: 1024x1024
**log_level**: info
```

### Environment Variables (Recommended)

**Gemini API**:

```bash
export GEMINI_API_KEY="your-key"
```

**Vertex AI** (3 authentication modes):

```bash
# Option 1: Express Mode (REST) - Simplest
export VERTEX_API_KEY="your-key"
export VERTEX_PROJECT="your-project-id"
export VERTEX_LOCATION="us-central1"

# Option 2: Service Account - Production
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account.json"
export VERTEX_PROJECT="your-project-id"

# Option 3: Application Default Credentials - gcloud SDK
# No env vars needed - uses gcloud auth
```

**AWS Bedrock** (4 authentication modes):

```bash
# Option 1: REST API with Bearer Token
export AWS_BEARER_TOKEN_BEDROCK="your-bearer-token"
export AWS_REGION="us-east-1"

# Option 2: SDK with Access Keys (supports both long-term and short-term credentials)
export AWS_ACCESS_KEY_ID="AKIA..."           # Long-term access key ID
export AWS_SECRET_ACCESS_KEY="wJalr..."     # Long-term secret access key
export AWS_SESSION_TOKEN="FwoGZXIvYXd..."   # Optional: for short-term credentials
export AWS_REGION="us-east-1"

# Option 3: SDK with Named Profile
export AWS_PROFILE="your-profile-name"
export AWS_REGION="us-east-1"

# Option 4: SDK with IAM Role (Lambda/EC2/ECS - no credentials needed)
# Instance/task role automatically provides credentials
export AWS_REGION="us-east-1"  # Optional, defaults to instance region
```

**xAI Grok**:

```bash
export GROK_API_KEY="xai-your-key"
```

**Check credential conflicts**:

```bash
gimage auth status  # Shows which credentials are active and their sources
```

## Available Models

### Gemini API (Google AI Studio)

- **`gemini-3-pro-image-preview`** (default)

  - Pricing: $0.134/image (1K/2K), $0.24/image (4K)
  - Native 4K resolution, sharp text rendering
  - Best for: Professional work, text-heavy images, diagrams

- **`gemini-3.1-flash-image-preview`** (4K flash)

  - Pricing tiered by resolution: $0.045 (0.5K), $0.067 (1K), $0.101 (2K), $0.151 (4K)
  - Improved text rendering vs Flash 2.5
  - Best for: High-quality images at flash speed

- **`gemini-2.5-flash-image`** (affordable, recommended for beginners)

  - Pricing: $0.039/image
  - Resolution: up to 1024x1024 (also supports 1024x1792, 1792x1024)
  - Fast generation (~2-3 seconds)
  - Best for: Quick iterations, development, testing

### Vertex AI (Google Cloud) - Paid

- **`imagen-4`** (default Vertex model, highest quality)

  - Pricing: $0.04/image
  - Resolution: up to 2048x2048
  - Best for: Professional work, final production images

- **`imagen-4-fast`** (optimized for speed)

  - Pricing: $0.02/image
  - Faster generation with good quality
  - Best for: Quick iterations, batch processing

- **`imagen-4-ultra`** (premium quality)

  - Pricing: $0.06/image
  - Highest quality Vertex AI model
  - Best for: Print-ready, marketing materials

- **`imagen-3.0-generate-002`** (Imagen 3, legacy)

  - Pricing: $0.04/image
  - Resolution: up to 1536x1536
  - Note: Prefer Imagen 4 for new projects

- **`imagen-3.0-fast-generate-001`** (Imagen 3 Fast, legacy)
  - Pricing: $0.02/image
  - Note: Prefer Imagen 4 Fast for new projects

### AWS Bedrock - Paid

- **`amazon.nova-canvas-v1:0`** (Nova Canvas)
  - Resolution: up to 1408x1408
  - Pricing: $0.04/image (up to 1024x1024), $0.08/image (larger)
  - Best for: AWS-integrated applications

### xAI Grok - Paid

- **`grok-imagine-image`** (default Grok model, fast and affordable)

  - Pricing: $0.02/image
  - Supports aspect ratio (13 ratios: 1:1, 16:9, 9:16, 4:3, 3:4, etc.)
  - Best for: Quick iterations, affordable image generation

- **`grok-imagine-image-pro`** (higher quality)

  - Pricing: $0.07/image
  - Supports aspect ratio
  - Best for: Higher quality creative images

- **`grok-2-image`** (legacy, Aurora-powered)
  - Pricing: $0.07/image
  - Note: Does not support size, style, negative prompt, or aspect ratio parameters
  - Prefer Grok Imagine for new projects

**View all models with live pricing:**

```bash
gimage generate --list-models
# or
gimage auth list  # Shows configured providers
```

## Image Formats Supported

**Input & Output**: PNG, JPEG, WebP, GIF, TIFF, BMP

All processing operations preserve format by default, or you can specify output format with `--output` flag or use the `convert` command.

## Advanced Features

### Styles (Generation)

- `photorealistic` - Realistic photos
- `artistic` - Artistic interpretations
- `anime` - Anime/manga style

### Resize Modes

When resizing images, aspect ratio is always preserved:

- `crop` (default) - Fill target dimensions, crop any excess
- `fit` - Fit entire image within target, may have borders

```bash
# Crop mode: 800x600 output, excess cropped
gimage resize --input photo.jpg --width 800 --height 600 --mode crop

# Fit mode: image fits within 800x600, preserving full image
gimage resize --input photo.jpg --width 800 --height 600 --mode fit
```

### Quality Settings

- Compression quality: 1-100 (default: 90)
- Resampling: Lanczos algorithm (highest quality)
- Transparency: Preserved for PNG, white background for JPEG

### Batch Processing (MCP Server Only)

Batch operations are available exclusively through the MCP server for AI assistants:

- `batch_resize` - Concurrent image resizing
- `batch_compress` - Concurrent compression
- `batch_convert` - Concurrent format conversion

CLI users can use shell scripts with `find` + `xargs` for parallel processing.

## Use Cases

### For Designers & Artists

- Rapid prototyping of visual concepts
- Generate variations of ideas
- Quick image editing and optimization
- Batch resize for different platforms

### For Developers

- Generate placeholder images for apps
- Automate image processing pipelines
- Integrate AI generation into workflows
- Test with reproducible images (seeds)

### For Content Creators

- Create social media graphics
- Generate blog post illustrations
- Batch optimize images for web
- Convert formats for different platforms

### For AI/ML Engineers

- Generate training data
- Create synthetic datasets
- Automate image augmentation
- Integrate with Claude via MCP server

## MCP Server for AI Assistants

Gimage can run as an MCP (Model Context Protocol) server, enabling AI assistants like Claude to generate and process images directly. The server communicates over stdio using the MCP protocol and exposes 10 tools for image generation and processing.

### Installation Methods

There are two ways to install gimage for use with Claude Desktop. Choose based on your use case:

#### Method 1: Homebrew (Recommended - macOS/Linux only)

**Use this if:** You want both CLI access AND MCP server functionality from a single installation.

**Step 1: Install gimage via Homebrew**

```bash
brew install apresai/tap/gimage
gimage --version  # Verify installation
```

**Step 2: Configure Claude Desktop MCP**

Edit `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `~/.config/Claude/claude_desktop_config.json` (Linux):

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

**Why this method?**

- ✅ Single installation serves both CLI and MCP
- ✅ Easy updates: `brew upgrade apresai/tap/gimage`
- ✅ Directly calls the `gimage` binary in your PATH
- ✅ Smaller total footprint (one binary)

#### Method 2: npm Package (Cross-platform alternative)

**Use this if:** You're on Windows, don't use Homebrew, or only want MCP functionality (no CLI needed).

**Step 1: Install via npm**

```bash
npm install -g @apresai/gimage-mcp
```

**Step 2: Configure Claude Desktop MCP**

Edit your Claude Desktop config file:

- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Linux**: `~/.config/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

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

**Why this method?**

- ✅ Works on Windows (Homebrew method doesn't)
- ✅ npm-based workflow (familiar to Node.js users)
- ✅ `npx` automatically downloads/runs the correct version
- ⚠️ Note: npm installs gimage ONLY for MCP use (hidden in npm global directory)
- ⚠️ If you want CLI access too, you'll need to install via Homebrew separately

**Configuration difference explained:**

- **Homebrew**: Uses `"command": "gimage"` - directly calls the binary in your PATH
- **npm**: Uses `"command": "npx"` - npx finds and runs the npm-installed package

#### Method 3: Claude Code

**Use this if:** You're using Claude Code CLI and want to add gimage as an MCP server.

**Prerequisites:** Install gimage via Homebrew first (see Method 1, Step 1).

**One-command setup:**

```bash
claude mcp add-json gimage '{"command": "gimage", "args": ["serve"]}'
```

That's it! Claude Code will now have access to all gimage MCP tools.

### Setup Authentication

Before using the MCP server, configure your API credentials:

```bash
# Quick start with Gemini (most affordable)
gimage auth setup gemini

# Or use environment variable (more secure):
export GEMINI_API_KEY="your-api-key-here"

# Verify configuration
gimage auth status
gimage auth test gemini
```

The MCP server automatically uses credentials from:

1. **Environment variables** - `GEMINI_API_KEY`, `VERTEX_API_KEY`, `AWS_ACCESS_KEY_ID`, `GROK_API_KEY`, etc. (RECOMMENDED for security)
2. **Config file** - `~/.gimage/config.md` (created by `gimage auth setup` commands)

**Best Practice**: Use environment variables for production - they're more secure than storing credentials in config files.

### Start Using with Claude

After installation and authentication, restart Claude Desktop. You can then use natural language:

```
"Generate an image of a sunset over mountains"
"Resize photo.jpg to 800x600"
"Compress all images in the photos directory to 85% quality"
"Create a 2048x2048 photorealistic image of a wise old wizard"
```

### Available MCP Tools

The MCP server exposes 10 tools for AI assistants:

| Tool             | Purpose                                 |
| ---------------- | --------------------------------------- |
| `generate_image` | AI image generation from text           |
| `resize_image`   | Resize to specific dimensions           |
| `scale_image`    | Scale by factor (maintain aspect ratio) |
| `crop_image`     | Crop to specific region                 |
| `compress_image` | Reduce file size with quality control   |
| `convert_image`  | Convert between formats                 |
| `batch_resize`   | Resize multiple images concurrently     |
| `batch_compress` | Compress multiple images                |
| `batch_convert`  | Convert multiple images to new format   |
| `list_models`    | List available AI models                |

### Troubleshooting MCP Server

If the MCP server isn't working in Claude Desktop:

1. **Verify gimage is installed and in PATH:**

   ```bash
   which gimage
   gimage --version
   ```

2. **Test the serve command directly:**

   ```bash
   gimage serve --verbose
   ```

   This will show detailed logging to stderr. Press Ctrl+C to stop.

3. **Verify API credentials are configured:**

   ```bash
   gimage auth status  # Check all configured credentials
   ```

4. **Test image generation works outside MCP:**

   ```bash
   gimage generate "test image"
   ```

5. **Check Claude Desktop logs** for error messages:

   - **macOS**: `~/Library/Logs/Claude/`
   - **Linux**: `~/.config/Claude/logs/`

6. **Ensure config file exists and has correct permissions:**
   ```bash
   ls -la ~/.gimage/config.md
   ```
   Should be readable (permissions: `-rw-------` or `-rw-r--r--`)

### Environment Variables

The MCP server respects the same environment variables as the CLI. See [Configuration](#configuration) for complete list of supported variables for all authentication modes (Gemini, Vertex AI, AWS Bedrock).

### MCP Documentation

For comprehensive guides and examples:

- **Usage Guide**: [docs/MCP_USAGE.md](docs/MCP_USAGE.md) - Complete setup and usage
- **Tool Reference**: [docs/MCP_TOOLS.md](docs/MCP_TOOLS.md) - Detailed tool documentation
- **Examples**: [docs/MCP_EXAMPLES.md](docs/MCP_EXAMPLES.md) - Real-world workflows
- **Implementation Plan**: [mcp.md](mcp.md) - Complete MCP architecture and implementation details

## Command Reference

See [COMMANDS.md](COMMANDS.md) for complete command reference and detailed usage examples.

**Available commands**:

- `generate` - AI image generation from text prompts
- `resize` - Resize images to specific dimensions (with crop/fit modes)
- `scale` - Scale images by a factor
- `crop` - Crop images to specific regions
- `compress` - Compress images to reduce file size (JPG, WebP)
- `convert` - Convert images between formats
- `auth` - Configure and manage API credentials (setup, status, list, test)
- `serve` - Start MCP server for Claude Desktop (includes batch operations)
- `tui` - Launch interactive terminal UI (supports all CLI options)

**Removed commands** (use alternatives):

- `batch` - Use MCP server or shell scripts (see [Batch Processing](#batch-processing))
- `config` - Use `auth` commands for configuration

Run `gimage [command] --help` for detailed usage of any command.

### Interactive TUI

Launch the interactive Terminal User Interface for a guided image generation experience:

```bash
gimage tui
```

**TUI Features:**

- Step-by-step image generation workflow (8 steps)
- Provider selection with pricing and auth status
- All CLI options available: size, style, negative prompt, seed, aspect ratio, image size, output format, resize mode, CFG scale, count
- Command preview before generation
- Progress tracking and error handling

**Workflow Steps:**

1. Enter your prompt
2. Select provider (Gemini, Vertex AI, Bedrock, Grok)
3. Choose image size
4. Select style (optional)
5. Configure advanced options
6. Set output path
7. Review command
8. Generate and view results

---

## Go SDK

A type-safe Go SDK for programmatic API access.

**Repository**: [github.com/apresai/gimage-go-sdk](https://github.com/apresai/gimage-go-sdk)

### Installation

```bash
go get github.com/apresai/gimage-go-sdk@latest
```

### Quick Start

```go
import gimage "github.com/apresai/gimage-go-sdk"

// Create client with API key authentication
client, _ := gimage.NewClient(
    "https://your-api.execute-api.us-east-1.amazonaws.com/production",
    gimage.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
        req.Header.Set("x-api-key", "your-api-key")
        return nil
    }),
)

// Generate image
resp, _ := client.GenerateImage(ctx, gimage.GenerateImageJSONRequestBody{
    Prompt: "sunset over mountains",
    Model:  stringPtr("gemini-2.5-flash-image"),
    Size:   stringPtr("1024x1024"),
})
```

### Features

- ✅ **Type-safe**: All types generated from OpenAPI spec
- ✅ **Auto-complete**: Full IDE support with godoc
- ✅ **API Gateway ready**: Built-in API key authentication support
- ✅ **Standard Go modules**: Independent versioning with semantic versioning

### Documentation

Complete documentation, examples, and API reference:

- **SDK Repository**: [github.com/apresai/gimage-go-sdk](https://github.com/apresai/gimage-go-sdk)
- **GoDoc**: [pkg.go.dev/github.com/apresai/gimage-go-sdk](https://pkg.go.dev/github.com/apresai/gimage-go-sdk)
- **Examples**: See the SDK repository for working examples

---

## Lambda API Distribution

Deploy Gimage as a serverless REST API on AWS Lambda for web application integration.

### Features

- **Serverless Architecture**: AWS Lambda on ARM64/Graviton2 (provided.al2023 runtime)
- **Auto-Scaling**: Handles 0 to thousands of requests automatically
- **Cost-Effective**: Pay only for what you use (~$0.25/month for 10K requests)
- **S3 Storage**: Automatic S3 bucket for image storage with presigned URLs
- **Production-Ready**: Full CORS, error handling, monitoring
- **API Gateway Integration**: Managed API keys, usage plans, rate limiting

### Quick Deploy

**Option 1: Using gimage-deploy tool (Recommended)**

```bash
# Build Lambda function
make build-lambda
make package-lambda

# Deploy using gimage-deploy (from sibling directory)
cd ../gimage-deploy
./bin/gimage-deploy deploy \
  --id production \
  --stage production \
  --region us-east-1 \
  --lambda-code ../gimage/bin/lambda.zip \
  --memory 512 \
  --timeout 30

# Create API key
./bin/gimage-deploy keys create --name prod-key --deployment production
```

**Option 2: Manual deployment**

See the `gimage-deploy` directory for manual deployment with AWS CLI.

### Deployment Tool

The `gimage-deploy` CLI tool (separate project) provides complete deployment management:

- One-command deployment to AWS Lambda
- API Gateway configuration with API keys
- Monitoring (logs, metrics, health checks)
- Interactive TUI for management
- No hardcoded account IDs - works in any AWS account

See the `gimage-deploy` directory for the deployment management tool.

### API Endpoints

All operations available via REST API:

| Endpoint        | Method | Purpose                                  |
| --------------- | ------ | ---------------------------------------- |
| `/generate`     | POST   | AI image generation                      |
| `/resize`       | POST   | Resize to dimensions                     |
| `/scale`        | POST   | Scale by factor                          |
| `/crop`         | POST   | Crop region                              |
| `/compress`     | POST   | Compress with quality                    |
| `/convert`      | POST   | Convert format                           |
| `/batch`        | POST   | Process multiple images                  |
| `/health`       | GET    | Health check                             |
| `/docs`         | GET    | **Interactive Swagger UI documentation** |
| `/openapi.yaml` | GET    | OpenAPI specification                    |

**Try the API**: After deployment, visit `https://your-api-url/prod/docs` for interactive documentation!

### Integration Examples

**TypeScript/JavaScript:**

```typescript
const response = await fetch("https://your-api-url/generate", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    prompt: "a sunset over mountains",
    size: "1024x1024",
    response_format: "s3_url",
  }),
});

const result = await response.json();
console.log("Image URL:", result.s3_url);
```

**Python:**

```python
import requests

response = requests.post('https://your-api-url/resize', json={
    'image': base64_image,
    'width': 800,
    'height': 600
})

result = response.json()
print(f"Resized: {result['width']}x{result['height']}")
```

**Go:**

```go
client := gimage.NewClient("https://your-api-url", apiKey)
result, err := client.GenerateImage(gimage.GenerateRequest{
    Prompt: "beautiful landscape",
    Size:   "1024x1024",
})
```

### Documentation

- **OpenAPI Spec**: [openapi.yaml](openapi.yaml) - Complete API reference
- **Go SDK**: [github.com/apresai/gimage-go-sdk](https://github.com/apresai/gimage-go-sdk) - Type-safe Go client

### Architecture

```
Client Request → API Gateway → Lambda (Go/ARM64) → {S3, Gemini, Vertex AI}
                                ↓
                         Small: base64 response
                         Large: S3 presigned URL
```

**Runtime**: AWS Lambda provided.al2023 (Amazon Linux 2023)
**Architecture**: ARM64 (Graviton2 processors)
**Package Size**: 17MB compressed, 42MB uncompressed
**Memory**: 2GB (configurable)
**Timeout**: 5 minutes max

---

## Building from Source

### CLI Binary

```bash
git clone https://github.com/apresai/gimage.git
cd gimage
make build
./bin/gimage --version
```

### Lambda Function

```bash
# Build for AWS Lambda ARM64
make build-lambda

# Package as deployment zip
make package-lambda

# Output: bin/lambda.zip (17MB)
```

## Requirements

### CLI

- **None!** Single static binary with zero dependencies
- Works on: macOS, Linux, Windows (x86_64, ARM64)
- No Python, no Node.js, no system libraries required

### Lambda API

- AWS Account
- AWS CLI configured
- Node.js 20+ (for CDK deployment)
- Go 1.22+ (for building)

## Support & Documentation

### CLI

- **Full Command Reference**: [COMMANDS.md](COMMANDS.md)
- **Configuration Guide**: See `gimage auth --help`

### Lambda API

- **OpenAPI Specification**: [openapi.yaml](openapi.yaml)

### Community

- **GitHub Issues**: https://github.com/apresai/gimage/issues
- **Discussions**: https://github.com/apresai/gimage/discussions

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Credits

Built with:

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Imaging](https://github.com/disintegration/imaging) - Image processing
- [Google Gen AI SDK](https://github.com/googleapis/go-genai) - Gemini API
- [Vertex AI SDK](https://cloud.google.com/go/vertexai) - Vertex AI
- [AWS SDK for Go](https://github.com/aws/aws-sdk-go-v2) - AWS Bedrock

---

**Made with ❤️ for developers, designers, and AI enthusiasts**
