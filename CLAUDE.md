# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Project Overview

`gimage` - A Go-based CLI tool for AI-powered image generation and processing.

**Core Capabilities**:
- Generate images using Google Gemini 3.1 Flash Lite, Gemini 3.1 Flash, Gemini 3 Pro (native 4K), Gemini 2.5 Flash (legacy), Vertex presets, or xAI Grok Imagine 2.0
- Process images: resize, scale, crop, compress, convert (PNG, JPG, WebP, GIF, TIFF, BMP)
- Batch processing via MCP server (batch_resize, batch_compress, batch_convert)
- MCP server for Claude Desktop integration
- AWS Lambda API deployment

**Technology Stack**:
- Pure Go 1.26+ (zero C dependencies for portability)
- Image processing: `github.com/disintegration/imaging`
- CLI: Cobra + Viper
- APIs: Gemini API, Vertex AI, xAI Grok

## Build Commands

```bash
make build          # Build CLI binary
make build-all      # Build for all platforms
make install        # Install locally
make test           # Run tests
make test-coverage  # Run tests with coverage
make lint           # Run linter
make scan           # Scan dependencies for known CVEs
make clean          # Clean artifacts
make benchmark      # Run benchmarks

# SDK Generation
make install-sdk-tools  # Install oapi-codegen (one-time)
make generate-sdk       # Generate Go SDK from openapi.yaml
make clean-sdk          # Remove generated SDK files
```

## Project Structure

```
gimage/
├── cmd/gimage/              # CLI entrypoint
├── cmd/lambda/              # Lambda entrypoint
├── internal/
│   ├── imaging/             # Image processing operations
│   ├── generate/            # AI image generation (Gemini, Vertex, Grok)
│   ├── config/              # Configuration & authentication
│   ├── cli/                 # CLI commands
│   ├── mcp/                 # MCP server implementation
│   └── lambdahandler/       # Lambda HTTP handler
├── pkg/models/              # Shared types
├── sdk/go/                  # Generated Go SDK (from openapi.yaml)
│   ├── client.gen.go        # HTTP client with all methods
│   ├── types.gen.go         # Type-safe request/response structs
│   ├── spec.gen.go          # Embedded OpenAPI spec
│   ├── README.md            # SDK documentation
│   └── example_test.go      # Usage examples
├── test/
│   ├── fixtures/            # Test images (DO NOT MODIFY)
│   └── integration/         # Integration tests
├── docs/                    # Documentation
└── openapi.yaml             # OpenAPI 3.0 spec (source of truth)
```

## Architecture Patterns

### Pure Go Philosophy
This project uses **pure Go with zero C dependencies**:
- Single binary distribution, no system dependencies
- Cross-compilation to any platform
- Uses `disintegration/imaging` (not bimg/libvips)
- **Never add C library dependencies**

### Configuration Hierarchy (Priority Order)
1. Command-line flags (highest)
2. Environment variables (`GEMINI_API_KEY`, `VERTEX_API_KEY`, `GROK_API_KEY`, etc.)
3. Config file (`~/.gimage/config.md`)
4. Default values (lowest)

### API Client Pattern
All backends (Gemini, Vertex, Grok) implement common interface:
```go
type ImageGenerator interface {
    GenerateImage(ctx context.Context, prompt string, options GenerateOptions) (*GeneratedImage, error)
    Close() error
}
```

### Provider Registry (Single Source of Truth)
The `ProviderRegistry` in `internal/generate/providers.go` is the central system for managing providers:
- `apiRegistry` map holds API metadata (display names, descriptions, pricing, display order)
- `registerAllProviders()` registers all provider/model combinations with aliases
- CLI and TUI dynamically derive their displays from the registry
- Adding a new API only requires updating `apiRegistry` - UI auto-updates

```go
// Example: apiRegistry entry
"grok": {
    ID:          "grok",
    DisplayName: "xAI Grok",
    Description: "xAI's Aurora-powered image generation",
    PricingNote: "Paid: ~$0.07 per image",
    Order:       4,
},
```

### Error Handling
- Return errors with context using `fmt.Errorf` with `%w`
- Provide actionable error messages
- Never panic in production code
- Validate inputs early

## Multi-Backend Architecture

**Supported Backends**:
- **Gemini API** (REST) - Paid, fastest setup
- **Vertex AI** - Express Mode (REST) or Full Mode (SDK)
- **xAI Grok** (REST) - Aurora-powered image generation

### Backend Selection Logic

Model name implies backend (auto-detect):
- `gemini-3.1-flash-lite-image` → gemini ($0.034/image, 1K only)
- `gemini-3.1-flash-image` → gemini (tiered: $0.045/0.5K, $0.067/1K, $0.101/2K, $0.151/4K)
- `gemini-3-pro-image` → gemini (native 4K, $0.134/image)
- `gemini-2.5-flash-image` → gemini ($0.039/image, legacy)
- `vertex-flash` → vertex (`vertex/flash-3.1`, Gemini 3.1 Flash via Vertex, medium thinking default)
- `vertex-flash-fast` → vertex (`vertex/flash-3.1-fast`, Gemini 3.1 Flash via Vertex, minimal thinking default)
- `vertex-flash-ultra` → vertex (`vertex/flash-3.1-ultra`, Gemini 3.1 Flash via Vertex, high thinking default)
- `vertex-flash-lite` → vertex (`vertex/flash-3.1-lite`, Gemini 3.1 Flash Lite via Vertex, minimal thinking default)
- `grok-imagine-image-2.0` → grok ($0.04/image; optional `--quality low|medium|auto`)
- `grok-imagine-image` → grok ($0.02/image, speed tier)
- `grok-imagine-image-quality` → grok ($0.05/image at 1K, $0.07/image at 2K; xAI `-pro` aliases still resolve here)

Optional `--api` flag overrides auto-detection.

### Model Name Resolution

Map informal names to exact model IDs:

| User Input | Exact Model ID | API | Features |
|-----------|---------------|-----|----------|
| "gemini", "gemini-3", "gemini-3-pro" | `gemini-3-pro-image` | gemini | Native 4K, sharp text, $0.134/image (default) |
| "gemini-3.1-flash", "gemini-3.1", "3.1-flash" | `gemini-3.1-flash-image` | gemini | Tiered by resolution: $0.045 (0.5K), $0.067 (1K), $0.101 (2K), $0.151 (4K) |
| "flash", "gemini-flash", "gemini-lite", "gemini-3.1-flash-lite" | `gemini-3.1-flash-lite-image` | gemini | $0.034/image, 1K only; thinking `minimal`/`high`; no Search grounding |
| "gemini-2.5", "gemini-2.5-flash", "gemini-2.5-flash-image" | `gemini-2.5-flash-image` | gemini | Legacy Nano Banana, $0.039/image, 1024x1024 max |
| "vertex-flash" | `gemini-3.1-flash-image` | vertex | Gemini 3.1 Flash via Vertex, medium thinking default; tiered $0.045-$0.151/image |
| "vertex-flash-fast" | `gemini-3.1-flash-image` | vertex | Gemini 3.1 Flash via Vertex, minimal thinking default; tiered $0.045-$0.151/image |
| "vertex-flash-ultra" | `gemini-3.1-flash-image` | vertex | Gemini 3.1 Flash via Vertex, high thinking default; tiered $0.045-$0.151/image |
| "vertex-flash-lite" | `gemini-3.1-flash-lite-image` | vertex | Gemini 3.1 Flash Lite via Vertex, minimal thinking default; $0.034/image, 1K only |
| "grok", "grok-imagine", "grok-2", "xai", "aurora", "grok-quality", "grok-imagine-quality" | `grok-imagine-image-2.0` | grok | Latest Quality Mode, $0.04/image; optional `--quality low\|medium\|auto`; up to 5 refs |
| "grok-fast", "grok-imagine-image" | `grok-imagine-image` | grok | Speed tier, $0.02/image |
| "grok-imagine-image-quality", "grok-imagine-pro" (xAI alias), "grok-imagine-image-pro" (xAI alias) | `grok-imagine-image-quality` | grok | Previous quality tier, $0.05/image at 1K, $0.07/image at 2K. xAI still aliases `-pro` names here. |

**Gemini 3 Pro** and **Gemini 3.1 Flash** support native upscaling via `--image-size` flag: `1K`, `2K`, or `4K`. **Gemini 3.1 Flash Lite** accepts `1K` only. **Grok Imagine** models accept `--image-size 1K` or `2K` (mapped to xAI's `resolution` param).

**Grok Imagine** supports `aspect_ratio` with 16 values: `1:1`, `16:9`, `9:16`, `4:3`, `3:4`, `3:2`, `2:3`, `2:1`, `1:2`, `19.5:9`, `9:19.5`, `20:9`, `9:20`, `21:9`, `5:2`, `auto`.

**Reference image editing** via repeatable `--input-image` (PNG/JPEG/WebP, local paths only):
- **Grok** (max 5 on Imagine 2.0, max 3 on older Grok): routes to `POST /v1/images/edits`. Each `--input-image` may be a local path or a public `https://` URL (URLs are passed through to xAI; `http://` is rejected). Multi-image prompts may reference `<IMAGE_0>`, `<IMAGE_1>`, etc. Optional `--user` sends xAI's end-user abuse-monitoring field. Optional `--quality` (`low|medium|auto`) is Imagine 2.0 only.
- **Gemini / Vertex**: compositional editing (Nano Banana style). Caps — Gemini 2.5 Flash: 3, Gemini 3 Pro: 11 (docs: 6 objects + 5 characters), Gemini 3.1 Flash / Flash Lite: 14. The API doesn't distinguish object vs character images in the payload, so gimage enforces a single combined total per model.

**Gemini 3+ exclusive options** (silently ignored by Gemini 2.5 Flash and non-Gemini providers):
- `--thinking` (`minimal|low|medium|high`): controls reasoning depth before generation. Higher = better layouts and text, slightly slower. Flash Lite accepts `minimal` and `high` only (`low`/`medium` are coerced).
- `--grounding` (bool): enables Google Search grounding via `tools: [{"google_search":{}}]`. Not supported by Flash Lite. Billed per search query in addition to per-image cost. Useful when the prompt references real-world current entities (products, news, logos).

**Batch pricing tier (Gemini, ~50% off)** is available via Google's separate `:batchGenerateContent` async endpoint with a 24-hour SLA. This is **not** wired into the gimage CLI — adding it requires a job-submission/polling subsystem, tracked as future work.

**Always use exact model IDs from the mapping table.**

**RETIRED names (now error with guidance)**: `imagen`, `imagen-4`, `imagen-4-fast`, `imagen-4-ultra`, `imagen-fast`, `imagen-ultra`, `imagen-4.0-generate-001`, `imagen-4.0-fast-generate-001`, `imagen-4.0-ultra-generate-001`, `gemini-3-pro-image-preview`, `gemini-3.1-flash-image-preview`, `nova`, `nova-canvas`, `amazon.nova-canvas-v1:0`, `bedrock/nova-canvas` (Bedrock Nova Canvas retired, AWS end-of-life 2026-09-30). Passing any of these to `--model` or `--provider` returns an error with guidance. Note: `grok-imagine-pro` / `grok-imagine-image-pro` are **not** retired errors; they resolve to `grok-imagine-image-quality` (matching xAI's live aliases).

**VIDEO / NON-IMAGE names (error, not supported)**: Gemini Omni (`omni`, `gemini-omni`, `gemini-omni-flash`, `gemini-omni-flash-preview`, `gemini-omni-1.1-flash`) is a video model on the Interactions API (3–10s, 720p). gimage is still-image only; do not register Omni as an image provider.

### Post-Generation: WebP Conversion

AI-generated PNGs are typically 1-3 MB. Always convert to WebP for 90%+ size reduction:

```bash
# Using gimage convert
gimage convert --input generated.png --format webp

# Using cwebp for maximum control (preferred for photos)
cwebp -q 85 -mt -sharp_yuv -preset photo generated.png -o generated.webp
```

## Development Workflow

### Adding a New CLI Command
1. Create command file in `internal/cli/`
2. Implement using Cobra patterns
3. Add flags with Viper binding
4. Wire up to root command
5. Add unit tests
6. Update `COMMANDS.md`

### Adding Image Processing Operations
1. Create operation file in `internal/imaging/`
2. Use `disintegration/imaging` library exclusively
3. Handle all supported formats (PNG, JPG, WebP, GIF, TIFF, BMP)
4. Add comprehensive error handling
5. Create unit tests with fixtures from `test/fixtures/` (DO NOT MODIFY)
6. Benchmark critical operations

### Adding a New Provider/Backend

When adding a new AI provider (like Grok, Vertex, etc.), update these locations:

**Core Implementation** (required):
1. `internal/generate/<provider>.go` - Client implementation (implement `ImageGenerator` interface)
2. `internal/generate/providers.go` - Two changes:
   - Add entry to `apiRegistry` map (display name, description, pricing, order)
   - Register provider in `registerAllProviders()` with aliases

**Config System** (required):
3. `internal/config/config.go` - Add fields for new credentials (e.g., `GrokAPIKey`)
4. `internal/config/auth.go` - Add `Has<Provider>Credentials()` function
5. `internal/generate/providers.go` - Add credential gathering in `gatherCredentials()`

**Dynamic UI** (automatic - no changes needed!):
- CLI `--list-providers` - Uses `registry.ListAPIs()` and `GroupAuthStatusByAPI()`
- TUI settings menu - Uses same registry methods
- TUI about page - Uses same registry methods
- TUI help text - Uses same registry methods

**MCP Server** (usually automatic):
6. `internal/mcp/tools/generate.go` - Usually works automatically via provider registry
7. `internal/mcp/tools/models.go` - May need update if `list_models` has special logic

**Documentation** (update all):
8. `CLAUDE.md` - Model name resolution table, authentication section
9. `README.md` - Supported backends list
10. `COMMANDS.md` - Generate command examples with new provider
11. `docs/MCP_TOOLS.md` - MCP tool documentation
12. `docs/MCP_USAGE.md` - Usage examples

**Tests**:
13. `test/integration/generate_e2e_test.go` - Add E2E test for new provider
14. Unit tests for client implementation

**Architecture Note**: The `apiRegistry` in `providers.go` is the single source of truth for API metadata. The CLI and TUI dynamically derive their displays from this registry, preventing configuration drift. When adding a new API, just add it to `apiRegistry` and the UI will automatically include it.

## CLI Standards

### Command Interface Pattern

**Image processing commands use explicit flags**:
- Consistent, explicit, self-documenting
- Composable with shell scripts
- Clear in logs and command history

**Generation command supports both positional and flag-based prompts**:
```bash
# Positional prompt (most common, recommended for quick use)
gimage generate "sunset over mountains"

# Flag-based prompt (explicit, useful in scripts)
gimage generate --prompt "sunset over mountains"
```

**Standard Flags**:
- `--input, -i`: Input file path (required for most image processing commands)
- `--output, -o`: Output file path (optional, auto-generated if omitted)
- `--verbose, -v`: Enable verbose output (available on all commands)

**Examples**:
```bash
# Image processing commands (flags-only)
gimage resize --input photo.jpg --width 800 --height 600 --output resized.jpg
gimage crop --input photo.jpg --x 100 --y 100 --width 400 --height 300
gimage scale --input photo.jpg --factor 0.5
gimage convert --input photo.jpg --format webp
gimage compress --input photo.jpg --quality 85

# Generation command (supports both styles)
gimage generate "sunset over mountains" --size 1024x1024
gimage generate --prompt "sunset over mountains" --output sunset.png

# Auth commands (positional provider argument)
gimage auth status
gimage auth setup gemini
gimage auth list
gimage auth test gemini
```

**Available CLI Commands**:
- `generate` - Generate images from text prompts
- `resize` - Resize images to specific dimensions
- `scale` - Scale images by a factor
- `crop` - Crop images to specific regions
- `compress` - Compress images to reduce file size (JPG, WebP)
- `convert` - Convert images between formats
- `auth` - Configure and manage API credentials
- `serve` - Start MCP server (includes batch operations)
- `tui` - Launch interactive terminal UI

**Removed Commands** (no longer available):
- `batch` - Use MCP server tools instead (batch_resize, batch_compress, batch_convert)
- `config` - Use `auth` commands for configuration

### Verbose Logging

All commands support `--verbose` flag for detailed output:
```bash
gimage resize --input photo.jpg --width 800 --height 600 --verbose
# Outputs:
# ℹ Resizing photo.jpg to 800x600...
# • Input: photo.jpg
# • Output: photo_resized_800x600.jpg
# • Dimensions: 800x600
# ✓ Resized successfully!
```

### Output Path Generation

If `--output` is omitted, commands auto-generate descriptive output paths:
- `resize`: `input_resized_WxH.ext`
- `crop`: `input_cropped_WxH.ext`
- `scale`: `input_scaled_FACTORx.ext`
- `convert`: `input_converted.FORMAT`
- `compress`: `input_compressed.ext`

### Testing Strategy

**Unit Tests (>80% coverage required)**:
- Test request building logic (validate JSON structure)
- Test response parsing with real example responses
- Test input validation (dimensions, prompts, parameters)
- Test configuration loading
- Test CLI flag parsing

**Integration Tests (manual, costs money)**:
- Real API calls to Gemini/Vertex/Grok
- Run manually: `go test -tags=integration`
- **DO NOT MOCK cloud provider APIs** - mocks provide zero value

**Table-driven tests** for multiple scenarios.

### MCP Server

MCP server runs via `gimage serve` and exposes 10 tools for AI assistants:
- **Single operations**: `generate_image`, `resize_image`, `scale_image`, `crop_image`, `compress_image`, `convert_image`
- **Batch operations**: `batch_resize`, `batch_compress`, `batch_convert` (concurrent processing)
- **Utilities**: `list_models`

**Important**: Batch operations are ONLY available through MCP server, not CLI.
- CLI users should wrap `gimage` in shell scripts for batch processing
- MCP server provides optimized concurrent batch operations for AI assistants

Config: `~/.gimage/config.md` (markdown format using `**key**: value`)

## Authentication

### Auth Commands

Modern auth command structure:

```bash
gimage auth status    # Show authentication status for all providers
gimage auth list      # List all configured providers with sources
gimage auth test      # Test credentials by making real API calls
gimage auth setup     # Interactive setup wizard for providers
```

### Authentication Precedence (Highest to Lowest)

**All Providers**:
1. Command-line flags (e.g., `--gemini-api-key`)
2. Environment variables
3. Config file (`~/.gimage/config.md`)
4. Default values

**Gemini API**:
- Single credential: `GEMINI_API_KEY`
- Simple REST client with API key

**Vertex AI** (3 authentication modes):
1. **Express Mode (REST)**: `VERTEX_API_KEY` → Fast, simple, REST-based
2. **Service Account**: `GOOGLE_APPLICATION_CREDENTIALS` → JSON key file path
3. **Application Default Credentials (ADC)**: Automatic → gcloud SDK, workload identity

**xAI Grok**:
- Single credential: `GROK_API_KEY`
- REST client with Bearer token authentication
- Get API key at: https://console.x.ai

### Config File Format

Location: `~/.gimage/config.md` (markdown format, 0600 permissions)

```markdown
# Gimage Configuration

⚠️  SECURITY WARNING ⚠️
This file contains SENSITIVE API KEYS stored in PLAINTEXT.

**gemini_api_key**: AIzaSy...
**vertex_api_key**: AIzaSy...
**vertex_project**: your-project-id
**vertex_location**: global
**vertex_credentials_path**: /path/to/service-account.json
**grok_api_key**: xai-5zM...
**default_api**: gemini
**default_model**: gemini-3-pro-image
**log_level**: info
```

## Security & Best Practices

### Credential Security

**Config File Security**:
- Config file (`~/.gimage/config.md`) stores API keys in **PLAINTEXT**
- File created with 0600 permissions (only owner can read/write)
- Includes prominent security warnings at the top
- **NEVER commit config file to version control**
- **NEVER share config file or its contents**

**Best Practices**:
- **PREFER environment variables** over config file for sensitive keys
- Use `gimage auth status` to see where credentials are coming from
- Rotate API keys regularly (every 90 days recommended)
- Use separate keys for dev/staging/production environments
- For CI/CD pipelines, always use environment variables
- For Lambda/EC2/ECS, prefer IAM roles over static credentials

**Environment Variable Priority**:
- Environment variables override config file (by design)
- Set `GEMINI_API_KEY`, `VERTEX_API_KEY`, `GROK_API_KEY`, etc.
- Use `gimage auth status` to check for conflicts

**Warning About Conflicts**:
- If both config file AND environment variable are set, env var wins
- `gimage auth status` will warn you about conflicting credentials
- Clean up unused credentials to avoid confusion

### Documentation and Dates
- **ALWAYS use `date +%Y-%m-%d`** command for current date
- Never hardcode dates in documentation
- Use dynamic date retrieval for CHANGELOG.md and docs

### Code Quality
- Follow Go idioms and conventions
- Keep functions small and focused
- Use golangci-lint
- Document all public APIs with godoc
- Never log API keys or sensitive data

### Git Security Hooks

**NEVER bypass git-secrets or other security hooks.**

**Forbidden practices**:
- `--no-verify` flag on git commit
- `SKIP=git-secrets` environment variable
- Disabling pre-commit hooks to "work around" issues

**When git-secrets fails**:
1. **Audit the error** - Determine if it's a real secret or a false positive
2. **Fix the root cause**:
   - If real secret: Remove the secret from the code immediately
   - If invalid regex pattern: Fix the pattern in `.git/config` or run `git config --unset`
   - If false positive: Add to allowed patterns with `git secrets --add --allowed 'pattern'`
3. **Never bypass** - The hook exists to protect against credential leaks

**Fixing broken git-secrets patterns**:
```bash
# List current patterns
git secrets --list

# Remove a broken pattern
git config --local --unset secrets.patterns 'broken-pattern'

# Add a properly escaped pattern
git secrets --add 'AKIA[0-9A-Z]{16}'

# Add an allowed pattern for examples
git secrets --add --allowed 'your-api-key-here'
```

**Why this matters**: A single leaked API key can result in:
- Unauthorized charges (potentially thousands of dollars)
- Service abuse and rate limit exhaustion
- Security breaches and data exposure
- Revoked credentials requiring rotation across all systems

## Common Patterns

### Loading and Saving Images
```go
img, err := imaging.Open(inputPath)
if err != nil {
    return fmt.Errorf("failed to open image: %w", err)
}

result := imaging.Resize(img, width, height, imaging.Lanczos)
err = imaging.Save(result, outputPath)
```

### Concurrent Processing
```go
workers := runtime.NumCPU()
sem := make(chan struct{}, workers)
var wg sync.WaitGroup

for _, file := range files {
    wg.Add(1)
    go func(f string) {
        defer wg.Done()
        sem <- struct{}{}
        defer func() { <-sem }()
        // Process file
    }(file)
}
wg.Wait()
```

## Git Usage Policy

**IMPORTANT**: Do NOT use git commands unless user explicitly asks.

**Do NOT**:
- Auto-commit after creating/modifying files
- Auto-commit after completing features
- Auto-commit when you think "this should be committed"

**DO**:
- Only when user says "commit this"
- Only when user says "push to GitHub"
- Only when user explicitly requests git operations

**Why**: User controls when and how code is committed. Automatic commits interrupt workflow and create unwanted history.

## Release Process

**Use `make release` for all releases.** It tags and pushes, then GitHub Actions handles the rest:

```bash
make release
```

**What it does:**
1. Updates CHANGELOG.md with current date
2. Syncs version to `package.json` and `npm/package.json`
3. Commits and pushes changes to main
4. Creates and pushes git tag `v$(VERSION)`
5. **GitHub Actions automatically:**
   - Runs GoReleaser (builds binaries, creates GitHub release, updates Homebrew tap)
   - Publishes npm package via OIDC (no token needed!)

**Version calculation:**
- Auto-calculated: `1.2.$(git rev-list --count HEAD)`
- Override with: `VERSION=1.3.0 make release`

**Required secrets (in GitHub repo settings):**
- `GITHUB_TOKEN` - Automatic (provided by GitHub Actions)
- `HOMEBREW_TAP_TOKEN` - Required for Homebrew tap updates

**npm Publishing - Token-Free via OIDC:**
npm uses GitHub Actions OIDC trusted publishing - no npm token required!

One-time setup on npmjs.com:
1. Go to https://www.npmjs.com/package/@apresai/gimage-mcp/access
2. Click "GitHub Actions" under Trusted Publishers
3. Add: owner=`apresai`, repo=`gimage`, workflow=`release.yml`

**Alternative: Local release (if GitHub Actions unavailable):**
```bash
make release-local  # Requires GITHUB_TOKEN, HOMEBREW_TAP_TOKEN, NPM_TOKEN
```

## Lambda Deployment

Deploy as serverless REST API on AWS Lambda:

```bash
make build-lambda      # Build for ARM64/Graviton2
make package-lambda    # Create deployment zip
```

See the `gimage-deploy` sibling directory for deployment management.

## Documentation Structure

- **README.md** - Main project overview
- **COMMANDS.md** - Full CLI command reference
- **TESTING.md** - Testing documentation
- **mcp.md** - MCP server overview
- **docs/MCP_TOOLS.md** - Complete MCP tools reference (for LLMs)
- **docs/MCP_USAGE.md** - Primary MCP user guide (for LLMs)
- **docs/MCP_EXAMPLES.md** - Real-world MCP examples (for LLMs)

## Implementation Priorities

Core development phases:
1. Project initialization
2. Image processing core
3. AI API integrations (Gemini → Vertex → Grok)
4. CLI commands
5. Configuration system
6. Testing suite
7. Documentation
8. MCP server
9. Lambda deployment
10. Distribution (Homebrew, npm)

## Go SDK

The Go SDK is published as a separate repository for proper Go module management.

### SDK Repository

- **Repository**: [github.com/apresai/gimage-go-sdk](https://github.com/apresai/gimage-go-sdk)
- **Import path**: `github.com/apresai/gimage-go-sdk`
- **Versioning**: Independent semantic versioning (v1.0.0, v1.1.0, etc.)
- **Installation**: `go get github.com/apresai/gimage-go-sdk@latest`

### Why Separate Repository?

Go best practices recommend separate repositories for libraries:
1. **Independent versioning** - SDK can have different release cadence than CLI
2. **Smaller dependencies** - Users only pull SDK code, not entire project
3. **Standard module path** - Simpler imports without subdirectories
4. **Better discoverability** - Listed separately on pkg.go.dev

### SDK Generation (for maintainers)

The SDK is auto-generated from `openapi.yaml` and published to the separate repository:

```bash
# In main gimage repo:
# 1. Generate SDK locally
make generate-sdk

# 2. Copy to SDK repo
cp -r sdk/go/* /path/to/gimage-go-sdk/

# 3. In SDK repo: commit, tag, and push
cd /path/to/gimage-go-sdk
git add .
git commit -m "Update SDK from openapi.yaml vX.X.X"
git tag v1.x.x
git push origin main --tags
```

### SDK Usage

**Installation**:
```bash
go get github.com/apresai/gimage-go-sdk@latest
```

**Usage**:
```go
import gimage "github.com/apresai/gimage-go-sdk"

// Create client with API key
client, _ := gimage.NewClient(
    "https://your-api.execute-api.us-east-1.amazonaws.com/production",
    gimage.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
        req.Header.Set("x-api-key", apiKey)
        return nil
    }),
)

// Generate image
resp, _ := client.GenerateImage(ctx, gimage.GenerateImageJSONRequestBody{
    Prompt: "sunset over mountains",
    Model:  stringPtr("gemini-2.5-flash-image"),
})
```

### SDK Documentation

Complete documentation in the SDK repository:
- **README**: Installation, quick start, API reference
- **EXAMPLE.md**: Real-world usage examples
- **GoDoc**: [pkg.go.dev/github.com/apresai/gimage-go-sdk](https://pkg.go.dev/github.com/apresai/gimage-go-sdk)

## Lambda Deployment Tool (gimage-deploy)

The `gimage-deploy` tool in the sibling directory manages Lambda deployments and API keys.

### Project Location

```
gimage/                  # Main gimage CLI
gimage-deploy/           # Deployment management tool (separate repo)
```

### Overview

**gimage-deploy** is a complete deployment manager for gimage Lambda functions:
- Deploy Lambda to AWS with one command
- Manage API Gateway API keys (CRUD operations)
- Monitor deployments (logs, metrics, health)
- Interactive TUI for visual management
- No CDK/Terraform required - uses AWS SDK directly

### Architecture

**Technology Stack**:
- Pure Go 1.26+
- AWS SDK v2 (Lambda, S3, IAM, API Gateway, CloudWatch, STS)
- Cobra for CLI framework
- Bubbletea for TUI
- AES-256-GCM for API key encryption

**Core Components**:
- `internal/aws/` - AWS service clients (Lambda, S3, IAM, etc.)
- `internal/deploy/` - Deployment orchestration
- `internal/apikeys/` - API key management with encryption
- `internal/storage/` - Local state management
- `internal/tui/` - Bubbletea interactive UI
- `pkg/utils/` - Crypto and validation utilities

### Deployment Manager Features

**Full Lifecycle Management**:
1. Creates S3 bucket for Lambda storage
2. Creates IAM role with Lambda execution policies
3. Deploys Lambda function (ARM64/Graviton2)
4. Creates API Gateway with proxy integration
5. Adds Lambda invoke permissions
6. Associates API keys with usage plans
7. Saves deployment metadata locally

**Resource Naming** (auto-generated, no hardcoding):
- S3 Bucket: `gimage-storage-{deployment-id}`
- Lambda: `gimage-processor-{deployment-id}`
- IAM Role: `gimage-lambda-role-{deployment-id}`
- API Gateway: `gimage-api-{deployment-id}`

### Available Commands

**Deployment Operations**:
```bash
gimage-deploy deploy --id prod --stage production --region us-east-1 --lambda-code lambda.zip
gimage-deploy list                    # List all deployments
gimage-deploy status <deployment-id>  # Show deployment details
gimage-deploy update <deployment-id>  # Update configuration
gimage-deploy destroy <deployment-id> # Delete deployment (with confirmation)
```

**API Key Management**:
```bash
gimage-deploy keys create --name prod-key --deployment prod
gimage-deploy keys list <deployment-id>
gimage-deploy keys delete <key-id>
gimage-deploy keys update <key-id> --enabled false
```

**Monitoring**:
```bash
gimage-deploy logs <deployment-id> --follow          # Tail CloudWatch logs
gimage-deploy metrics <deployment-id> --period 24h   # Show metrics
gimage-deploy health <deployment-id>                 # HTTP health check
```

**Interactive TUI**:
```bash
gimage-deploy tui  # Launch interactive terminal UI
```

### Security Features

**API Key Encryption**:
- Keys encrypted with AES-256-GCM before storage
- Machine-specific encryption key (hostname + username)
- Files stored with 0600 permissions
- Keys masked in UI/logs (shows first 12 + last 4 chars)

**AWS Account ID Resolution**:
- Uses STS GetCallerIdentity to get account ID dynamically
- **Never hardcodes account IDs** (safe for public repos)
- Works in any AWS account with valid credentials
- Respects AWS credential chain (profiles, env vars, IAM roles)

**IAM Permissions**:
- Lambda basic execution (CloudWatch Logs)
- S3 access for image storage
- Principle of least privilege

### Storage and State

**Local State Management**:
- Storage directory: `~/.gimage-deploy/`
- `deployments.json` - Deployment metadata
- `api_keys.encrypted.json` - Encrypted API keys
- `config.json` - User configuration
- All files have 0600 permissions

**Deployment Metadata**:
```go
type Deployment struct {
    ID              string
    Stage           string
    Region          string
    FunctionName    string
    FunctionARN     string
    APIGatewayID    string
    APIGatewayURL   string
    S3Bucket        string
    IAMRoleARN      string
    Status          DeploymentStatus
    Health          HealthStatus
    Configuration   LambdaConfiguration
    EnvironmentVars map[string]string
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

### TUI Features

**Interactive Screens**:
1. **Main Menu** - Navigate between sections
2. **Deployment List** - View all deployments with status
3. **API Key List** - Manage API keys with masked values

**Keyboard Navigation**:
- `↑/↓` or `j/k` - Navigate items
- `Enter` - Select item
- `r` - Refresh view
- `ESC` - Go back
- `q` - Quit

**Visual Indicators**:
- Color-coded status (green=active, red=failed)
- Real-time metrics
- Health scores (0-100)
- Masked API key values

### Deployment Workflow

**1. Build Lambda package**:
```bash
cd gimage
make build-lambda      # Build for ARM64
make package-lambda    # Create lambda.zip
```

**2. Deploy to AWS**:
```bash
cd ../gimage-deploy
./bin/gimage-deploy deploy \
  --id production \
  --stage production \
  --region us-east-1 \
  --lambda-code ../gimage/bin/lambda.zip \
  --memory 512 \
  --timeout 30 \
  --concurrency 10 \
  --architecture arm64
```

**3. Create API key**:
```bash
./bin/gimage-deploy keys create \
  --name prod-key \
  --deployment production \
  --rate-limit 1000 \
  --burst-limit 2000 \
  --quota-limit 100000
```

**4. Test deployment**:
```bash
curl https://{api-id}.execute-api.us-east-1.amazonaws.com/production/health \
  -H "x-api-key: {your-api-key}"
```

### Important Deployment Notes

**API Key Header**:
- Must use lowercase: `x-api-key` (NOT `X-API-Key`)
- API Gateway is case-sensitive for this header

**Environment Variables**:
- Set `S3_BUCKET` environment variable on Lambda
- Use `--env S3_BUCKET=bucket-name` when deploying
- Required for gimage Lambda to work

**Resource Cleanup**:
- `destroy` command removes ALL resources
- Prompts for confirmation unless `--yes` flag
- Cleans up in reverse order: API Gateway → Lambda → S3 → IAM

**Usage Plan Association**:
- API keys must be associated with API Gateway stage
- Handled automatically by `keys create` command
- Required for API key authentication to work

### Testing the Tool

**Unit Tests** (80.3% coverage):
```bash
cd gimage-deploy
make test
```

**Test Files**:
- `pkg/utils/crypto_test.go` - Encryption/decryption
- `pkg/utils/validation_test.go` - Input validation
- `internal/storage/config_test.go` - Config management
- `internal/storage/deployments_test.go` - Deployment CRUD

### Development Guidelines

**When working on gimage-deploy**:

1. **Never hardcode AWS account IDs** - use STS GetCallerIdentity
2. **Never hardcode regions** - accept as parameter or use config
3. **Always encrypt sensitive data** - API keys, credentials
4. **Use proper file permissions** - 0600 for config files
5. **Validate all inputs** - deployment IDs, stages, resource names
6. **Clean up on errors** - partial deployments should be removed
7. **Wait for IAM propagation** - 10 second delay after role creation
8. **Handle API Gateway quirks** - lowercase headers, stage association

**Architecture Decisions**:
- Uses AWS SDK v2 (not CDK) for direct control
- Stores state locally (not DynamoDB) for simplicity
- Encrypts keys locally (not KMS) to avoid AWS costs
- Uses file-based config (not database) for portability

### Related Projects

This is part of the gimage ecosystem:
- **gimage** (main) - CLI tool and Lambda function
- **gimage-deploy** (sibling) - Deployment management
- **Generated SDK** - Type-safe Go client (in gimage repo)

All three work together:
1. Build Lambda with `gimage`
2. Deploy with `gimage-deploy`
3. Use deployed API with generated SDK
