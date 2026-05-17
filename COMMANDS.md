# Gimage Command Reference

Complete reference for all gimage commands, flags, and options.

## Table of Contents

- [Global Flags](#global-flags)
- [generate](#generate) - Generate images from text using AI
- [resize](#resize) - Resize images (strictly preserves aspect ratio)
- [scale](#scale) - Scale images by factor
- [crop](#crop) - Crop images to regions
- [compress](#compress) - Compress images
- [convert](#convert) - Convert image formats
- [auth](#auth) - Configure authentication
  - [auth setup](#auth-setup) - Interactive setup wizard
  - [auth test](#auth-test) - Test authentication
  - [auth list](#auth-list) - List all providers
  - [auth status](#auth-status) - Show auth status
- [serve](#serve) - Start MCP server (includes batch operations)
- [tui](#tui) - Launch interactive terminal UI
- [completion](#completion) - Generate shell completions

---

## Global Flags

These flags work with any command:

| Flag            | Description           | Default                   |
| --------------- | --------------------- | ------------------------- |
| `--config`      | Path to config file   | `$HOME/.gimage/config.md` |
| `--verbose`     | Enable verbose output | `false`                   |
| `-h, --help`    | Show help for command | -                         |
| `-v, --version` | Show version          | -                         |

**Examples:**

```bash
gimage generate "sunset" --verbose
gimage --config ~/custom-config.md generate "landscape"
gimage --version
```

---

## generate

Generate images from text prompts using Google Gemini, Vertex AI, AWS Bedrock, or xAI Grok.

### Usage

```bash
gimage generate [prompt] [flags]
gimage generate --prompt "your prompt" [flags]
```

**Note:** You can provide the prompt as a positional argument (most common) or use the `--prompt` flag.

### Flags

| Flag               | Type   | Description                                                                             | Default                      |
| ------------------ | ------ | --------------------------------------------------------------------------------------- | ---------------------------- |
| `-p, --prompt`     | string | Text prompt (alternative to positional arg)                                             | -                            |
| `--provider`       | string | Provider ID (e.g., `gemini/flash-2.5`, `vertex/imagen-4`)                               | Auto-detected                |
| `--api`            | string | API to use: `gemini`, `vertex`, or `bedrock` (deprecated, use `--provider`)             | Auto-detected from model     |
| `--model`          | string | Model to use (deprecated, use `--provider`)                                             | `gemini-3-pro-image-preview` |
| `--size`           | string | Image size (WxH)                                                                        | `1024x1024`                  |
| `--style`          | string | Style: `photorealistic`, `artistic`, `anime` (Bedrock: `standard`, `premium`)           | -                            |
| `--negative`       | string | Negative prompt to avoid features                                                       | -                            |
| `--seed`           | int    | Random seed for reproducibility                                                         | `0` (random)                 |
| `--cfg-scale`      | float  | CFG scale for creativity (Bedrock: 1.0-10.0, higher = more creative)                    | Model default                |
| `-n, --count`      | int    | Number of images to generate (max varies by provider)                                   | `1`                          |
| `--output-format`  | string | Output format for Vertex AI: `png`, `jpeg`, or `webp`                                   | Provider default             |
| `-o, --output`     | string | Output file path                                                                        | `generated_<timestamp>.png`  |
| `--list-models`    | bool   | List all available models with pricing                                                  | `false`                      |
| `--list-providers` | bool   | List all providers with auth status                                                     | `false`                      |
| `--prompt-howto`   | bool   | Show tips and examples for writing effective prompts                                    | `false`                      |
| `--image-size`     | string | Native resolution for Gemini 3 Pro / 3.1 Flash (`1K`, `2K`, `4K`) and Grok Imagine / Quality (`1K`, `2K` only) | -                            |
| `--aspect-ratio`   | string | Aspect ratio for Gemini 3 Pro (e.g., `1:1`, `16:9`, `9:16`, `4:3`, `3:4`, `3:2`, `2:3`) | -                            |

### Available Models

**Gemini API:**

- `gemini-3-pro-image-preview` (default) - $0.134/image (1K/2K), $0.24/image (4K), native 4K, sharp text
- `gemini-3.1-flash-image-preview` - tiered: $0.045 (0.5K), $0.067 (1K), $0.101 (2K), $0.151 (4K), improved text rendering
- `gemini-2.5-flash-image` - $0.039/image, up to 1024x1024 (also 1024x1792, 1792x1024)

**Vertex AI (Paid):**

- `imagen-4` - $0.04/image, up to 2048x2048
- `imagen-4-fast` - $0.02/image, speed-optimized
- `imagen-4-ultra` - $0.06/image, premium quality

**AWS Bedrock (Paid):**

- `amazon.nova-canvas-v1:0` - std ≤1024: $0.04, prem ≤1024: $0.06, std >1024: $0.06, prem >1024: $0.08; up to 1408x1408. Premium quality triggered by `--style photorealistic|premium|high|ultra`

**xAI Grok (Paid):**

- `grok-imagine-image` - $0.02/image, fast and affordable (default Grok model)
- `grok-imagine-image-quality` - $0.05/image, quality tier (replaces deprecated `-pro` retired by xAI 2026-05-15)

### Examples

**Basic generation:**

```bash
gimage generate "a sunset over mountains"
```

**With specific size and style:**

```bash
gimage generate "futuristic city" --size 1024x1024 --style photorealistic
```

**Using Vertex AI:**

```bash
gimage generate "abstract art" --api vertex --model imagen-4
```

**With negative prompts:**

```bash
gimage generate "forest scene" --negative "people, buildings, cars"
```

**Reproducible generation:**

```bash
gimage generate "random pattern" --seed 12345
```

**Custom output path:**

```bash
gimage generate "landscape" --output my-landscape.png
```

**Batch generation (multiple images):**

```bash
gimage generate "fantasy castle" --count 3
```

**Gemini 3 Pro with aspect ratio:**

```bash
gimage generate "wide landscape" --model gemini-3-pro --aspect-ratio 16:9
```

**Gemini 3 Pro with native 4K resolution:**

```bash
gimage generate "detailed infographic" --model gemini-3-pro --image-size 4K
```

**Vertex AI with output format:**

```bash
gimage generate "portrait" --api vertex --output-format webp
```

**Bedrock with CFG scale:**

```bash
gimage generate "abstract art" --model nova-canvas --cfg-scale 10.0
```

**Bedrock with premium quality:**

```bash
gimage generate "luxury product" --model nova-canvas --style premium
```

**Using Imagen 4 Fast (speed-optimized):**

```bash
gimage generate "product mockup" --model imagen-4-fast
```

**Using Imagen 4 Ultra (premium quality):**

```bash
gimage generate "architectural visualization" --model imagen-4-ultra
```

**Using xAI Grok Imagine (fast, $0.02/image):**

```bash
gimage generate "robot waving hello" --model grok
```

**Using Grok Imagine Quality (higher quality):**

```bash
gimage generate "detailed portrait" --model grok-imagine-quality
```

**Using Grok Imagine with aspect ratio:**

```bash
gimage generate "wide landscape" --model grok --aspect-ratio 16:9
```

**List all models:**

```bash
gimage generate --list-models
```

**Convert to WebP after generation (recommended):**

AI-generated PNGs are typically 1-3 MB. Converting to WebP reduces file size by 90%+ with no visible quality loss:

```bash
# Generate then convert to WebP
gimage generate "sunset" -o sunset.png
gimage convert --input sunset.png --format webp

# Or use cwebp for maximum control
cwebp -q 85 -mt -sharp_yuv -preset photo sunset.png -o sunset.webp
```

### Image Sizes

Common sizes (format: `WIDTHxHEIGHT`):

- `512x512` - Small, fast generation
- `1024x1024` - Default, balanced quality/speed
- `1536x1536` - High quality
- `2048x2048` - Maximum quality (Vertex AI and Bedrock at 1408x1408 max)

Custom sizes are supported by some models.

---

## resize

Resize images to specific dimensions using high-quality Lanczos resampling.

### Usage

```bash
gimage resize [input] [width] [height] [flags]
```

### Arguments

| Argument | Type   | Description             | Required |
| -------- | ------ | ----------------------- | -------- |
| `input`  | string | Input image file path   | Yes      |
| `width`  | int    | Target width in pixels  | Yes      |
| `height` | int    | Target height in pixels | Yes      |

### Flags

| Flag           | Type   | Description                | Default                 |
| -------------- | ------ | -------------------------- | ----------------------- |
| `-o, --output` | string | Output file path           | `<input>_resized.<ext>` |
| `--mode`       | string | Resize mode: `crop`, `fit` | `crop`                  |

### Examples

**Basic resize:**

```bash
gimage resize photo.jpg 800 600 --mode crop
```

**Resize with custom output:**

```bash
gimage resize photo.png 1920 1080 --output fullhd.png
```

**Resize to thumbnail:**

```bash
gimage resize image.jpg 150 150 --output thumbnail.jpg
```

### Notes

- Strictly preserves aspect ratio in all modes
- `crop` (default): Fills the target size by cropping the excess
- `fit`: Fits the entire image within the target size (adds padding)
- Uses Lanczos resampling for highest quality
- Preserves transparency for PNG images
- Output format matches input unless specified

---

## scale

Scale images by a factor (e.g., 0.5 for half size, 2.0 for double).

### Usage

```bash
gimage scale [input] [factor] [flags]
```

### Arguments

| Argument | Type   | Description                   | Required |
| -------- | ------ | ----------------------------- | -------- |
| `input`  | string | Input image file path         | Yes      |
| `factor` | float  | Scale factor (e.g., 0.5, 2.0) | Yes      |

### Flags

| Flag           | Type   | Description      | Default                |
| -------------- | ------ | ---------------- | ---------------------- |
| `-o, --output` | string | Output file path | `<input>_scaled.<ext>` |

### Examples

**Scale to 50% size:**

```bash
gimage scale photo.jpg 0.5
```

**Double the size:**

```bash
gimage scale image.png 2.0
```

**Scale with custom output:**

```bash
gimage scale photo.jpg 1.5 --output larger.jpg
```

### Notes

- Factor < 1.0 reduces size
- Factor > 1.0 increases size
- Factor = 1.0 creates a copy
- Uses Lanczos resampling for quality

---

## crop

Crop images to a specific region defined by x, y coordinates and dimensions.

### Usage

```bash
gimage crop [input] [x] [y] [width] [height] [flags]
```

### Arguments

| Argument | Type   | Description           | Required |
| -------- | ------ | --------------------- | -------- |
| `input`  | string | Input image file path | Yes      |
| `x`      | int    | Starting X coordinate | Yes      |
| `y`      | int    | Starting Y coordinate | Yes      |
| `width`  | int    | Crop width in pixels  | Yes      |
| `height` | int    | Crop height in pixels | Yes      |

### Flags

| Flag           | Type   | Description      | Default                 |
| -------------- | ------ | ---------------- | ----------------------- |
| `-o, --output` | string | Output file path | `<input>_cropped.<ext>` |

### Examples

**Basic crop:**

```bash
gimage crop photo.jpg 100 100 800 600
```

**Crop from top-left corner:**

```bash
gimage crop image.png 0 0 1920 1080
```

**Crop with custom output:**

```bash
gimage crop photo.jpg 50 50 500 500 --output square.jpg
```

### Notes

- Coordinates start at (0, 0) in top-left corner
- Region must be within image bounds
- Preserves format and transparency

---

## compress

Compress images to reduce file size while maintaining quality.

### Usage

```bash
gimage compress [input] [flags]
```

### Arguments

| Argument | Type   | Description           | Required |
| -------- | ------ | --------------------- | -------- |
| `input`  | string | Input image file path | Yes      |

### Flags

| Flag           | Type   | Description                 | Default                    |
| -------------- | ------ | --------------------------- | -------------------------- |
| `--quality`    | int    | Compression quality (1-100) | `90`                       |
| `-o, --output` | string | Output file path            | `<input>_compressed.<ext>` |

### Examples

**Basic compression:**

```bash
gimage compress photo.jpg
```

**Custom quality:**

```bash
gimage compress photo.jpg --quality 85
```

**Compress with output:**

```bash
gimage compress large.png --quality 75 --output small.png
```

### Quality Guidelines

| Quality | Use Case                 | File Size |
| ------- | ------------------------ | --------- |
| 95-100  | Professional photography | Largest   |
| 85-94   | High-quality web images  | Large     |
| 75-84   | Standard web images      | Medium    |
| 60-74   | Thumbnails, previews     | Small     |
| 1-59    | Maximum compression      | Smallest  |

### Notes

- Quality 90 is recommended default
- JPEG compression is lossy
- PNG uses lossless compression
- Higher quality = larger file size

---

## convert

Convert images to different formats.

### Usage

```bash
gimage convert [input] [format] [flags]
```

### Arguments

| Argument | Type   | Description           | Required |
| -------- | ------ | --------------------- | -------- |
| `input`  | string | Input image file path | Yes      |
| `format` | string | Target format         | Yes      |

### Flags

| Flag           | Type   | Description      | Default            |
| -------------- | ------ | ---------------- | ------------------ |
| `-o, --output` | string | Output file path | `<input>.<format>` |

### Supported Formats

| Format   | Extension       | Description                     |
| -------- | --------------- | ------------------------------- |
| PNG      | `.png`          | Lossless, supports transparency |
| JPEG/JPG | `.jpg`, `.jpeg` | Lossy, best for photos          |
| WebP     | `.webp`         | Modern, efficient               |
| GIF      | `.gif`          | Animated images                 |
| TIFF     | `.tiff`, `.tif` | Professional/archival           |
| BMP      | `.bmp`          | Uncompressed                    |

### Examples

**PNG to JPEG:**

```bash
gimage convert image.png jpg
```

**JPEG to WebP:**

```bash
gimage convert photo.jpg webp
```

**With custom output:**

```bash
gimage convert input.png webp --output optimized.webp
```

### Notes

- Transparency is preserved when converting to PNG
- Converting to JPEG from PNG with transparency adds white background
- WebP offers best compression for web use
- Format is case-insensitive

---

## Batch Operations

**Batch operations are not available as CLI commands.** They are available through:

1. **MCP Server** (for AI assistants like Claude Desktop):

   - `batch_resize` - Concurrent image resizing
   - `batch_compress` - Concurrent compression
   - `batch_convert` - Concurrent format conversion
   - See [serve](#serve) command and [MCP documentation](mcp.md)

2. **Shell Scripts** (for CLI users):

   ```bash
   # Resize all JPG files using find + xargs
   find photos/ -name "*.jpg" | xargs -P 4 -I {} gimage resize --input {} --width 800 --height 600

   # Compress all images in parallel
   find photos/ -name "*.jpg" -o -name "*.png" | xargs -P 4 -I {} gimage compress --input {} --quality 85

   # Convert all PNG to WebP
   find photos/ -name "*.png" | xargs -P 4 -I {} sh -c 'gimage convert --input "$1" --format webp --output "${1%.png}.webp"' _ {}
   ```

---

## auth

Manage authentication for all image generation providers (Gemini, Vertex AI, AWS Bedrock, xAI Grok).

### Usage

```bash
gimage auth [subcommand]
```

### Subcommands

- `setup <provider>` - Interactive setup wizard for a provider
- `test <provider>` - Test authentication for a provider
- `list` - List all providers with auth status and pricing
- `status` - Show detailed authentication status

---

### auth setup

Interactive setup wizard for configuring provider credentials.

#### Usage

```bash
gimage auth setup <provider>
```

#### Arguments

| Argument   | Description          | Examples                                                               |
| ---------- | -------------------- | ---------------------------------------------------------------------- |
| `provider` | Provider ID or alias | `gemini`, `gemini/flash-2.5`, `vertex/imagen-4`, `bedrock/nova-canvas` |

#### Examples

**Quick start with Gemini:**

```bash
gimage auth setup gemini
```

**Setup Vertex AI Imagen 4:**

```bash
gimage auth setup vertex/imagen-4
```

**Setup AWS Bedrock Nova Canvas:**

```bash
gimage auth setup bedrock/nova-canvas
```

#### What it does

1. Shows provider information (pricing, models, capabilities)
2. Guides you through required credentials
3. Shows existing values (masked for secrets)
4. Validates configuration
5. Saves to `~/.gimage/config.md` with secure permissions (0600)

---

### auth test

Test if authentication works for a provider by making real API calls.

#### Usage

```bash
gimage auth test <provider>
```

#### Arguments

| Argument   | Description          | Examples                               |
| ---------- | -------------------- | -------------------------------------- |
| `provider` | Provider ID or alias | `gemini`, `vertex/imagen-4`, `bedrock` |

#### Flags

| Flag         | Description                                  | Default |
| ------------ | -------------------------------------------- | ------- |
| `--all`      | Test all configured providers                | `false` |
| `--generate` | Actually generate a test image (costs money) | `false` |
| `--verbose`  | Show detailed test output                    | `false` |

#### Examples

**Test Gemini authentication:**

```bash
gimage auth test gemini
```

**Test with actual image generation:**

```bash
gimage auth test gemini --generate
```

**Test all configured providers:**

```bash
gimage auth test --all
```

#### What it does

1. Checks if required credentials are present
2. Validates credential format
3. Attempts to create API client
4. Optionally generates a small test image (256x256)
5. Reports success or detailed errors

---

### auth list

List all available providers with auth status, pricing, and capabilities.

#### Usage

```bash
gimage auth list [flags]
```

#### Flags

| Flag           | Description                             | Default |
| -------------- | --------------------------------------- | ------- |
| `--configured` | Show only configured providers          | `false` |
| `--missing`    | Show only providers missing credentials | `false` |
| `--detailed`   | Show detailed credential requirements   | `false` |

#### Examples

**List all providers:**

```bash
gimage auth list
```

**Show only configured providers:**

```bash
gimage auth list --configured
```

**Show detailed requirements for missing providers:**

```bash
gimage auth list --missing --detailed
```

#### Output

Shows table with:

- ✓ / ✗ Configuration status
- Provider ID (e.g., `gemini/flash-2.5`)
- Provider name
- Pricing (cost per image)
- Credential source (env/config/both)
- Missing credentials

---

### auth status

Show detailed authentication status for all providers.

#### Usage

```bash
gimage auth status
```

#### What it shows

**For each provider:**

- Configuration status (✓ configured / ✗ not configured)
- Credential sources (environment variables, config file)
- Masked preview of API keys
- Warnings about conflicting credentials

**Credential Priority Hierarchy:**

```
1. CLI Flags       (highest priority)
   ↓
2. Environment Variables
   ↓
3. Config File (~/.gimage/config.md)
   ↓
4. Defaults        (lowest priority)
```

#### Example Output

```bash
$ gimage auth status
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Authentication Status
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Credential Priority: CLI Flags > Environment Variables > Config File > Defaults

✓ Gemini API (Configured)
  • Environment: GEMINI_API_KEY = AIza***cVVI

✓ Vertex AI (Configured)
  • Config file: vertex_api_key = AIza***xYzW (Express Mode)
  • Config file: vertex_project = my-gcp-project
  • Environment: VERTEX_LOCATION = us-central1

✗ AWS Bedrock (Not Configured)
  Run: gimage auth setup bedrock
```

---

## serve

Start MCP (Model Context Protocol) server for AI assistant integration.

### Usage

```bash
gimage serve [flags]
```

### Description

Starts gimage as an MCP server that AI assistants (like Claude Desktop) can use to:

- Generate AI images
- Process images (resize, scale, crop, compress, convert)
- Perform batch operations (concurrent processing)

The server communicates over stdio using JSON-RPC protocol.

### Flags

| Flag        | Description                       | Default |
| ----------- | --------------------------------- | ------- |
| `--verbose` | Enable detailed logging to stderr | `false` |

### Claude Desktop Configuration

**macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
**Linux:** `~/.config/Claude/claude_desktop_config.json`
**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

**Homebrew installation:**

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

**npm installation:**

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

### Available MCP Tools

The MCP server exposes 10 tools:

| Tool             | Purpose                                  |
| ---------------- | ---------------------------------------- |
| `generate_image` | AI image generation from text            |
| `resize_image`   | Resize to specific dimensions            |
| `scale_image`    | Scale by factor (preserves aspect ratio) |
| `crop_image`     | Crop to specific region                  |
| `compress_image` | Reduce file size                         |
| `convert_image`  | Convert between formats                  |
| `batch_resize`   | Resize multiple images concurrently      |
| `batch_compress` | Compress multiple images concurrently    |
| `batch_convert`  | Convert multiple images concurrently     |
| `list_models`    | List available AI models with pricing    |

### Examples

**Start server:**

```bash
gimage serve
```

**Start with verbose logging:**

```bash
gimage serve --verbose
```

**Test server manually:**

```bash
# Server reads JSON-RPC from stdin and writes to stdout
echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | gimage serve
```

### Troubleshooting

**If MCP server isn't working in Claude:**

1. **Verify gimage is in PATH:**

   ```bash
   which gimage
   gimage --version
   ```

2. **Test credentials:**

   ```bash
   gimage auth status
   gimage auth test gemini
   ```

3. **Test image generation works:**

   ```bash
   gimage generate "test image"
   ```

4. **Check Claude Desktop logs:**

   - macOS: `~/Library/Logs/Claude/`
   - Linux: `~/.config/Claude/logs/`

5. **Test serve command directly:**
   ```bash
   gimage serve --verbose
   # Press Ctrl+C to stop
   ```

For more details, see [MCP documentation](mcp.md) and [docs/MCP_USAGE.md](docs/MCP_USAGE.md).

---

## tui

Launch interactive Terminal User Interface for gimage.

### Usage

```bash
gimage tui
```

### Alternative

```bash
gimage --interactive
```

### Description

Launches an 8-step interactive menu for image generation:

1. **Enter Prompt** - Describe the image you want
2. **Select Provider** - Choose AI provider and model (Gemini, Vertex AI, Bedrock, or Grok)
3. **Choose Size** - Select image dimensions
4. **Select Style** - Pick style (photorealistic, artistic, anime, or none)
5. **Advanced Options** - Configure optional parameters:
   - Negative prompt (exclude unwanted elements)
   - Seed (for reproducibility)
   - Aspect ratio (Gemini 3 Pro: Auto, 1:1, 16:9, 9:16, 4:3, 3:4, 3:2, 2:3)
   - Native resolution (Gemini 3 Pro: Default, 1K, 2K, 4K)
   - Output format (Vertex AI: Auto, PNG, JPEG, WebP)
   - CFG scale (Bedrock: 1.0-10.0 for creativity control)
   - Count (batch generation, max varies by provider)
6. **Set Output Path** - Specify where to save the image
7. **Review & Confirm** - Review all settings before generation
8. **Generate** - Create your image

The TUI provides feature parity with CLI and MCP, supporting all advanced generation options.

### Features

- Interactive prompt-based workflow
- Real-time preview of operations
- Keyboard navigation
- Context-sensitive help

### Examples

**Launch TUI:**

```bash
gimage tui
```

**Or use global flag:**

```bash
gimage --interactive
```

### Keyboard Navigation

**General Navigation:**

- Tab / ↑↓: Navigate between fields
- ←→: Change picker values
- Enter: Continue to next step / Confirm
- Esc: Go back to previous step
- Ctrl+C: Exit at any time

**Advanced Options Step:**

- Tab: Move between input fields and pickers
- ←→: Adjust picker values (aspect ratio, resolution, format, count)
- Type directly: Enter values for text fields (negative prompt, seed, CFG scale)

The TUI provides context-sensitive help throughout the workflow.

---

## completion

Generate shell completion scripts for bash, zsh, fish, or PowerShell.

### Usage

```bash
gimage completion [shell]
```

### Supported Shells

- `bash`
- `zsh`
- `fish`
- `powershell`

### Examples

**Bash:**

```bash
gimage completion bash > /etc/bash_completion.d/gimage
```

**Zsh:**

```bash
gimage completion zsh > "${fpath[1]}/_gimage"
```

**Fish:**

```bash
gimage completion fish > ~/.config/fish/completions/gimage.fish
```

**PowerShell:**

```powershell
gimage completion powershell | Out-String | Invoke-Expression
```

### Setup Instructions

**Bash (Linux):**

```bash
gimage completion bash | sudo tee /etc/bash_completion.d/gimage
source /etc/bash_completion.d/gimage
```

**Bash (macOS):**

```bash
brew install bash-completion
gimage completion bash > $(brew --prefix)/etc/bash_completion.d/gimage
```

**Zsh:**

```bash
gimage completion zsh > ~/.zsh/completions/_gimage
# Add to ~/.zshrc:
fpath=(~/.zsh/completions $fpath)
autoload -U compinit && compinit
```

**Fish:**

```bash
gimage completion fish > ~/.config/fish/completions/gimage.fish
```

---

## Environment Variables

These environment variables can be used to configure gimage:

| Variable                         | Description                          | Example             |
| -------------------------------- | ------------------------------------ | ------------------- |
| `GEMINI_API_KEY`                 | Gemini API key                       | `AIzaSy...`         |
| `VERTEX_API_KEY`                 | Vertex AI Express Mode key           | `AIzaSy...`         |
| `VERTEX_PROJECT`                 | Vertex AI project ID                 | `my-gcp-project`    |
| `VERTEX_LOCATION`                | Vertex AI location                   | `us-central1`       |
| `GOOGLE_APPLICATION_CREDENTIALS` | Service account file path            | `/path/to/key.json` |
| `GROK_API_KEY`                   | xAI Grok API key                     | `xai-5zM...`        |
| `GIMAGE_CONFIG`                  | Custom config file path              | `~/my-config.md`    |
| `GIMAGE_LOG_LEVEL`               | Log level (debug, info, warn, error) | `debug`             |

### Priority Order

1. Command-line flags (highest)
2. Environment variables
3. Config file
4. Default values (lowest)

---

## Exit Codes

| Code | Meaning                  |
| ---- | ------------------------ |
| 0    | Success                  |
| 1    | General error            |
| 2    | Command-line usage error |
| 130  | Interrupted (Ctrl+C)     |

---

## Tips & Tricks

### Reproducible Generations

Use `--seed` to get the same image every time:

```bash
gimage generate "abstract art" --seed 42
```

### Quick Thumbnails

Create thumbnails for all images using shell scripts:

```bash
# Using find + xargs for parallel processing
find photos/ -name "*.jpg" | xargs -P 4 -I {} gimage resize --input {} --width 200 --height 200

# Or use MCP server batch_resize tool with Claude
```

### Optimize for Web

Compress and convert using shell scripts:

```bash
# Using a simple loop
for file in photos/*.jpg; do
  gimage convert --input "$file" --format webp
done

for file in photos/*.webp; do
  gimage compress --input "$file" --quality 85
done

# Or use MCP server batch_convert and batch_compress tools with Claude
```

### Test Different Models

Compare model outputs:

```bash
gimage generate "sunset" --model gemini-2.5-flash-image --output gemini.png
gimage generate "sunset" --api vertex --model imagen-4 --output imagen.png
```

### Process Only JPEGs

Use shell globbing:

```bash
for file in *.jpg; do gimage compress "$file" --quality 80; done
```

### Chain Operations

Process images in pipeline:

```bash
gimage generate "landscape" --output temp.png
gimage resize temp.png 1920 1080 --output hd.png
gimage compress hd.png --quality 90 --output final.png
```

---

## Common Workflows

### Social Media Content

```bash
# Generate image
gimage generate "tech background" --size 1024x1024

# Create Instagram post (1080x1080)
gimage resize generated.png 1080 1080 --output instagram.png

# Create Twitter header (1500x500)
gimage crop generated.png 0 262 1500 500 --output twitter-header.png
```

### Batch Optimization

Batch operations are available via MCP server for AI assistants like Claude, or using shell scripts:

```bash
# Using shell scripts for CLI users:
mkdir -p hd web optimized

# Resize all photos to HD
find photos/ -name "*.jpg" | xargs -P 4 -I {} sh -c 'gimage resize --input {} --width 1920 --height 1080 --output hd/$(basename {})'

# Compress for web
for file in hd/*.jpg; do gimage compress --input "$file" --quality 85 --output "web/$(basename $file)"; done

# Convert to WebP
for file in web/*.jpg; do gimage convert --input "$file" --format webp --output "optimized/$(basename ${file%.jpg}.webp)"; done

# Or use MCP server tools (batch_resize, batch_compress, batch_convert) with Claude
```

### AI Art Generation

```bash
# Generate multiple variations
for i in {1..5}; do
  gimage generate "abstract landscape" --seed $i --output "art-$i.png"
done

# Generate different styles
gimage generate "cityscape" --style photorealistic --output photo.png
gimage generate "cityscape" --style artistic --output artistic.png
gimage generate "cityscape" --style anime --output anime.png
```

---

## Troubleshooting

### "API key not configured"

Run authentication setup:

```bash
gimage auth setup gemini
# or
gimage auth setup vertex
```

### "Failed to load config"

Check config file permissions:

```bash
ls -la ~/.gimage/config.md
chmod 600 ~/.gimage/config.md
```

### "Model not found"

List available models:

```bash
gimage generate --list-models
```

### "Image too large"

Reduce size or use scaling:

```bash
gimage scale large-image.jpg 0.5 --output smaller.jpg
```

### Verbose Mode

Enable detailed logging:

```bash
gimage generate "test" --verbose
```

---

## More Information

- **GitHub**: https://github.com/apresai/gimage
- **Issues**: https://github.com/apresai/gimage/issues
- **Discussions**: https://github.com/apresai/gimage/discussions
- **Main Documentation**: See [README.md](README.md)
- **Developer Guide**: See [CLAUDE.md](CLAUDE.md)
