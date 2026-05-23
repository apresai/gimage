# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

(empty - ready for next release)

## [1.2.131] - 2026-05-23

### Added
- Documentation for Gemini 3 thinking levels, Google Search grounding, and compositional editing with reference images in `docs/MCP_EXAMPLES.md`, `docs/PROMPT_GUIDE.md`, and `sdk/go/EXAMPLE.md`.
- CLI help text for `--thinking`, `--grounding`, and `--input-image` flags on the `generate` command.

### Changed
- Updated `mcp.md` to reflect new Gemini 3 capabilities.


## [1.2.129] - 2026-05-23

### Added
- Gemini 3+ thinking control via `--thinking` flag (`minimal|low|medium|high`) to tune reasoning depth before generation.
- Google Search grounding for Gemini 3+ via `--grounding` flag, billed per search query.
- Multi-image compositional editing via repeatable `--input-image` flag for Nano Banana-style reference inputs (caps: 3 for Gemini 2.5 Flash, 11 for Gemini 3 Pro, 14 for Gemini 3.1 Flash).
- MCP `generate_image` tool support for `thinking`, `grounding`, and `input_images` parameters.
- OpenAPI spec and SDK types updated to surface the new Gemini parameters.

### Changed
- Gemini REST client now constructs requests with `thinkingConfig`, `tools`, and inline image parts when the new options are supplied.


## [1.2.127] - 2026-05-17

### Changed
- Updated provider test counts to reflect Imagen 3 removal


## [1.2.125] - 2026-05-17

### Added
- `--image-size` flag support for Grok Imagine models (1K, 2K) via xAI's `resolution` parameter

### Changed
- Renamed Grok `grok-imagine-image-pro` to `grok-imagine-image-quality` (old name retained as alias)

### Removed
- Legacy Imagen 3 providers and all related references
- Grok-2 Image model (retired by xAI)


## [1.2.122] - 2026-04-13

### Added
- `gemini-3.1-flash-image-preview` listed in MCP tools documentation with tiered pricing ($0.045/0.5K → $0.151/4K)

### Changed
- `--image-size` flag now supports Gemini 3.1 Flash in addition to Gemini 3 Pro (updated help text and CLI descriptions)
- `--aspect-ratio` flag documentation updated to reflect support for all Gemini 3+ models
- MCP `aspect_ratio` tool description updated to include `gemini-3.1-flash-image-preview`
- Prompt guide and help output updated with Gemini 3.1 Flash examples for native 4K generation


## [1.2.120] - 2026-04-13

### Changed
- 


## [1.2.119] - 2026-04-12

### Added
- Gemini 3.1 Flash (`gemini-3.1-flash-image-preview`) model with tiered resolution pricing ($0.045/0.5K, $0.067/1K, $0.101/2K, $0.151/4K)
- Centralized pricing system (`internal/generate/pricing.go`) as single source of truth for all provider/model cost data
- Full Nova Canvas quality × size pricing matrix (standard/premium × ≤1024/>1024)

### Changed
- Updated Go dependencies (`go.mod`, `go.sum`)

### Fixed
- Pricing display bug in provider listings


## [1.2.115] - 2026-02-13

### Added
- Grok Imagine and Grok Imagine Pro model support with aliases (grok, xai, aurora)
- 265 unit tests across 13 new test files (coverage: 15.6% → 30%)
- Coverage report with embedded E2E images and prompt display
- Path traversal protection in image download paths

### Changed
- Updated provider registry with accurate pricing for all models
- Gemini Flash marked as paid ($0.039/image), no longer listed as free
- Bedrock REST client timeout reduced from 5 to 2 minutes
- Updated all dependencies to latest versions

### Fixed
- CLI/TUI/MCP model name mismatches for Grok providers


## [1.2.113] - 2026-02-05

### Added
- Imagen 4 Fast and Imagen 4 Ultra model support with automatic selection
- Bedrock SDK retry logic with exponential backoff (3 attempts, 2s/4s/8s delays)
- Credential validation in TUI settings page with real API test calls
- `list_models` tool now includes Imagen 4 Fast/Ultra variants

### Changed
- Unified Bedrock SDK implementation with consistent request/response handling
- Simplified CLI generate command to use provider registry (removed 200+ lines)
- Enhanced TUI generate flow with comprehensive model information and auto-selection
- Model resolution now automatically selects best model for each provider (Gemini 3 Pro, Imagen 4, etc.)
- Improved error handling across all generation backends with detailed context

### Fixed
- CLI/TUI/MCP model naming mismatches now consistent across all interfaces
- Bedrock REST client now properly normalizes model IDs (removes `.` suffixes)
- Bedrock SDK test coverage increased to match REST implementation
- Base64 image handling in download utilities now consistent

### Removed
- Deprecated `gemini.go` and `gemini_test.go` (consolidated into `gemini_rest.go`)
- Redundant Bedrock SDK code paths in favor of unified implementation


## [1.2.111] - 2026-01-25

### Changed
- Updated all Go dependencies to latest versions


## [1.2.109] - 2026-01-25

### Fixed
- Bedrock test cases to match auto-normalization behavior for image dimensions


## [1.2.107] - 2026-01-25

### Added
- Unified image size handling with aspect-ratio-preserving resize mode across all components
- Type coercion for image dimensions in MCP tools (auto-converts strings to integers)
- Shared pricing helper for consistent display of variable pricing across providers
- Comprehensive test suite for MCP resize tool (234 test cases, 100% coverage)
- Helper functions for MCP tool dimension parsing and validation

### Changed
- Standardized CLI output format across all image generation providers (Gemini, Vertex, Bedrock, Grok)
- Updated resize functionality in TUI, CLI, MCP tools, and Lambda handler to support aspect-ratio preservation
- Improved MCP tool documentation with clearer examples and type information
- Updated dependencies to latest versions
- Consolidated documentation by streamlining mcp.md content

### Fixed
- Resize mode parameter handling in MCP tools now properly validates allowed values
- Image dimension validation now correctly handles both integer and string inputs


## [1.2.99] - 2025-12-28

### Added
- GitHub Actions workflow for automated releases (GoReleaser, npm publishing via OIDC)
- `make release` command for automated version tagging and release process
- Token-free npm publishing using GitHub Actions OIDC trusted publishers

### Changed
- Release process now fully automated through GitHub Actions
- Version calculation now uses git commit count: `1.2.$(git rev-list --count HEAD)`
- Updated documentation with streamlined release instructions and OIDC setup guide


## [1.2.97] - 2025-12-27

### Changed
- Build number incremented to 1.2.97 (automatic versioning from git commit count)


## [1.2.95] - 2025-12-27

### Changed
- Improved CI/CD documentation with npm token setup requirements and granular access token configuration
- Updated MCP server documentation references and links


## [1.2.93] - 2025-12-27

### Changed
- Configured npm granular access token for CI/CD publishing
- Updated `.npmrc` to use `NPM_TOKEN` environment variable
- Verified full `make release` pipeline (GitHub + Homebrew + npm)


## [1.2.92] - 2025-12-27

### Security
- Fixed 3 vulnerabilities in `golang.org/x/crypto` (SSH DoS issues)
  - GO-2025-4135: Malformed constraint denial of service in ssh/agent
  - GO-2025-4134: Unbounded memory consumption in ssh
  - GO-2025-4116: Potential denial of service in ssh/agent
- Updated `golang.org/x/crypto` from v0.39.0 to v0.46.0

### Changed
- Updated all dependencies to latest versions:
  - `google.golang.org/genai`: v1.37.0 → v1.40.0 (Google AI SDK)
  - `aws-lambda-go`: v1.50.0 → v1.51.1
  - `aws-sdk-go-v2`: v1.40.1 → v1.41.0 (and all sub-packages)
  - `aws-sdk-go-v2/service/s3`: v1.93.0 → v1.95.0
  - `spf13/cobra`: v1.10.1 → v1.10.2
  - `golang.org/x/image`: v0.24.0 → v0.34.0
  - `golang.org/x/term`: v0.36.0 → v0.38.0
  - `golang.org/x/net`: v0.41.0 → v0.48.0
  - `golang.org/x/text`: v0.28.0 → v0.32.0
  - `google.golang.org/grpc`: v1.73.0 → v1.78.0
  - `google.golang.org/protobuf`: v1.36.6 → v1.36.11
  - OpenTelemetry: v1.36.0 → v1.39.0
  - Charmbracelet packages updated to latest

## [1.2.91] - 2025-12-07

### Changed
- Build number incremented to 1.2.91 (automatic versioning from git commit count)


## [1.2.89] - 2025-12-07

### Changed
- 


## [1.2.88] - 2025-12-07

### Changed
- Build number incremented to 1.2.88 (automatic versioning from git commit count)


## [1.2.86] - 2025-12-07

### Changed
- Build number incremented to 1.2.86 (automatic versioning from git commit count)


## [1.2.84] - 2025-12-07

### Changed
- 


## [1.2.83] - 2025-11-23

### Changed
- Build number incremented to 1.2.83 (automatic versioning from git commit count)


## [1.2.82] - 2025-11-23

### Changed
- Build number incremented to 1.2.82 (automatic versioning from git commit count)


## [1.2.81] - 2025-11-23

### Changed
- 


## [1.2.80] - 2025-11-23

### Changed
- Build number incremented to 1.2.80 (automatic versioning from git commit count)


## [1.2.79] - 2025-11-23

### Changed
- Build number incremented to 1.2.79 (automatic versioning from git commit count)


## [1.2.78] - 2025-11-23

### Changed
- Build number incremented to 1.2.78 (automatic versioning from git commit count)


## [1.2.76] - 2025-11-23

### Added
- **Go SDK**: Auto-generated type-safe SDK published to separate repository (`github.com/apresai/gimage-go-sdk`) with complete documentation and examples
- **gimage-deploy**: Complete Lambda deployment management tool with AWS SDK v2, API key management, CloudWatch monitoring, and interactive TUI
- **`--aspect-ratio` flag**: Support for aspect ratio constraints in image generation (e.g., `16:9`, `9:16`)
- **`--prompt-howto` guide**: Interactive prompt engineering guide in generate command
- **Prompt engineering documentation**: New `docs/PROMPT_GUIDE.md` with 159 lines of best practices
- **Verbose logging**: Comprehensive `--verbose` flag support across all commands with structured output
- **Homebrew installation**: Added `gimage-deploy` to Homebrew tap for easy installation

### Changed
- **Release process**: Replaced GitHub Actions with `make release` for all CI/CD (local-only releases)
- **Pricing accuracy**: Fixed pricing display for all providers (Gemini 3 Pro, Gemini Flash, Imagen 4, Nova Canvas)
- **Documentation**: Updated CLAUDE.md with Git Security Hooks guidelines, make release automation, and gimage-deploy integration
- **GoReleaser**: Enhanced config for multi-module builds supporting both gimage and gimage-deploy
- **OpenAPI spec**: Updated with aspect ratio support and improved model documentation

### Fixed
- **Multi-module builds**: Fixed GoReleaser configuration for proper handling of gimage-deploy alongside main CLI
- **Provider pricing**: Corrected cost estimates across Gemini, Vertex AI, and AWS Bedrock
- **Documentation links**: Updated references to point to separate Go SDK repository

### Removed
- **GitHub Actions workflows**: Removed `.github/workflows/` directory in favor of local `make release`


## [1.2.63] - 2025-11-06

### Changed

- **Generate command**: Now accepts positional prompt argument (`gimage generate "sunset"`) in addition to flag-based prompt (`--prompt`) for improved UX
- **Auth commands**: Refactored into modular structure with new dedicated `auth status` command providing detailed credential information
- **CLI verbose output**: Improved formatting and consistency across all image processing commands (resize, crop, scale, compress, convert)
- **Configuration**: Streamlined config management, removing legacy config command in favor of auth commands
- **Documentation**: Comprehensive overhaul of README.md, COMMANDS.md, CLAUDE.md, and MCP_USAGE.md with clearer examples and better organization
- **Integration tests**: Updated CLI E2E tests to use new positional prompt syntax

### Removed

- **Batch CLI commands**: Removed `gimage batch` command (batch operations now exclusively through MCP server)
- **TUI batch menu**: Removed batch processing UI from TUI (use MCP server batch tools instead)
- **Batch history tracking**: Removed history persistence system (`internal/batch/history.go` and tests)
- **Documentation files**: Removed 12 planning/implementation docs (DESIGN.md, IMPLEMENTATION_SUMMARY.md, PRODUCTION_QUALITY_FIXES.md, TUI_FEATURE_TOUR.md, TUI_IMPLEMENTATION_SUMMARY.md, TESTING.md, lambda.md, tui.md, npm/README.md, test/integration/README.md, internal/generate/README.md, INTEGRATION_GUIDE.md) - ~14,000 lines total
- **Config command**: Removed legacy `gimage config` subcommand


## [1.2.61] - 2025-11-05

### Changed
- 


## [1.2.60] - 2025-11-05

### Added
- Imagen 3 models support (`imagen-3.0-generate-001`, `imagen-3.0-generate-002`, `imagen-3.0-fast-generate-001`)
- Model alias `imagen-3` for latest Imagen 3 model
- Enhanced MCP end-to-end test coverage with model metadata validation

### Changed
- Updated model registry with comprehensive Imagen 3 and Imagen 4 model definitions
- Improved provider auto-detection logic to handle both Imagen 3 and Imagen 4 models
- Streamlined `models_test.go` using table-driven tests (197 → 151 lines)

### Fixed
- Resolved logger deadlock in auth commands by deferring stdout restoration
- Fixed test suite reliability issues in model registry tests


## [1.2.58] - 2025-11-05

### Added
- **Provider System**: New architecture for AI backends with clean abstraction layer for Gemini, Vertex AI, and AWS Bedrock
- **Model Registry**: Centralized system for model management and auto-detection
- **Enhanced Auth Commands**: 
  - `auth list` - Display all configured credentials and their status
  - `auth setup` - Interactive setup wizard for configuring providers
  - `auth test` - Validate credentials and test API connectivity
- **Design Documentation**: Added `DESIGN.md` documenting provider architecture and patterns

### Changed
- **Major Refactor**: Migrated from monolithic `models.go` (643 lines) to modular `providers.go` (565 lines) with cleaner separation
- **Auth Command Structure**: Reorganized authentication commands with new subcommands for better UX
- **Generate Command**: Enhanced with improved provider selection logic and better error handling
- **MCP Server**: Updated tools and prompts to work with new provider system
- **Logging**: Improved context and verbosity handling across components

### Removed
- **Legacy Code**: Removed old model management system (`internal/generate/models.go` and tests)
- **Test Files**: Cleaned up automated TUI tests (`automated_test.go`, `debug_test.go`)


## [1.1.54] - 2025-11-05

### Fixed
- TUI image generation now works correctly by switching from SDK to REST client for API calls
- Enhanced TUI generation flow with improved error handling and progress feedback

### Added
- Comprehensive logging system for debugging TUI operations
- Automated testing suite for TUI image generation workflows
- Debug mode support with detailed operation logging
- Progress indicators and status messages during image generation

### Changed
- Refactored TUI generation flow to use REST client instead of SDK for better reliability
- Improved TUI styles and visual feedback during operations


## [1.1.52] - 2025-11-05

### Added
- macOS keyboard shortcuts for improved navigation
- Settings navigation menu for easier configuration access
- Editable API keys in settings interface

### Changed
- Enhanced error handling in generate flow
- Improved settings menu UI and functionality


## [1.1.50] - 2025-11-05

### Changed
- Updated Go dependencies to resolve test compatibility issues


## [1.1.48] - 2025-11-05

### Fixed

- Add context.Context parameter to test function calls for proper context handling in image processing tests


## [1.1.46] - 2025-11-05

```markdown
## [Unreleased]

### Added
- Interactive TUI (Terminal User Interface) with main menu, batch processing, generation flow, and settings management
- Batch operation history tracking with persistent storage
- Progress reporter for real-time operation feedback
- Production quality test suite with comprehensive integration tests
- Image compression operation with quality control
- TUI documentation and feature tour
- Test fixtures (small_test.png, test_image.png, test_image_512x512.png)

### Changed
- Simplified CLI command outputs for better TUI integration
- Improved image processing operations (resize, scale, crop, convert) with enhanced error handling
- Streamlined documentation: consolidated guides into concise references
- Reduced project documentation by 56% (removed planning and implementation tracking docs)
- Updated lambda.md from 1,385 to 272 lines (removed CDK code, kept deployment guide)
- Updated INTEGRATION_GUIDE.md to focus on crisp examples only

### Removed
- Project planning documents (RELEASING.md, roadmap.md, HOMEBREW.md)
- Implementation tracking docs (DEPLOYMENT_CHECKLIST.md, LAMBDA_STATUS.md)
- Research/analysis docs (MCP_LLM_LEARNING_ANALYSIS.md, AI_TOOL_CALLING_IMPROVEMENTS.md, AWS_BEDROCK_SDK_GUIDE.md, etc.)
- Redundant documentation (API_REFERENCE.md, SWAGGER_SETUP.md, RELEASE_NOTES.md, etc.)
```


## [1.1.43] - 2025-11-02

### Added

- MCP Prompts feature: New prompt templates for image generation, batch processing, and common workflows
- MCP server now exposes 13 prompt templates via the prompts/list capability
- Comprehensive documentation for MCP Prompts design and implementation (MCP_PROMPTS_DESIGN.md, MCP_PROMPTS_IMPLEMENTATION.md)
- Analysis documentation for LLM learning patterns with MCP (MCP_LLM_LEARNING_ANALYSIS.md)

### Changed

- Enhanced MCP tool descriptions with more actionable guidance for LLM clients
- Improved MCP handler with prompt list and get capabilities
- Updated MCP server to register prompt templates on initialization


## [1.1.41] - 2025-11-02

### Changed
- 


## [1.1.40] - 2025-11-02

### Changed
- Build number incremented to 1.1.40 (automatic versioning from git commit count)


## [1.1.38] - 2025-11-02

### Changed
- Upgraded GoReleaser configuration to v2 format for improved build and release automation


## [1.1.36] - 2025-11-02

### Changed
- Build number incremented to 1.1.36 (automatic versioning from git commit count)


## [1.1.34] - 2025-11-02

### Changed
- 


## [1.1.33] - 2025-11-02

### Changed
- Updated .gitignore patterns for improved exclusion rules


## [1.1.32] - 2025-11-02

### Added

- AWS Bedrock Nova Canvas integration with dual implementation modes (REST and SDK)
- AWS Bedrock authentication setup via `gimage auth bedrock` command
- Nova Canvas model support (`amazon.nova-canvas-v1:0`) with quality presets (standard/premium)
- Advanced generation controls: negative prompts, CFG scale, seed, and quality settings
- Comprehensive AWS Bedrock documentation (SDK guide, quickstart, onboarding guide)
- Testing infrastructure with coverage reporting tools (`cmd/coverage-report`, `cmd/test-report`, `cmd/test-summary`)
- Extensive test suites for Bedrock REST and SDK clients (382+ and 305+ test cases respectively)
- MCP tools test coverage (batch, convert, generate operations)
- End-to-end integration tests for CLI and generation workflows
- Testing best practices documentation (TESTING.md)
- Model onboarding guide (MODEL_ONBOARDING.md) for adding new backends
- Documentation index (DOCUMENTATION_INDEX.md) for centralized reference
- Coverage report scripts with detailed HTML output

### Changed

- Updated CLAUDE.md with multi-backend architecture guidance and AWS Bedrock sections
- Enhanced `gimage generate` command with AWS-specific flags (quality, seed, CFG scale, negative prompts)
- Expanded configuration system to support AWS credentials and region settings
- Updated README.md with AWS Bedrock usage examples
- Improved MCP_TOOLS.md with Bedrock integration examples
- Enhanced Makefile with test coverage and reporting targets
- Refactored generate models to support backend-specific options
- Updated auth.go with Bedrock credential management (REST and SDK modes)

### Fixed

- Image scaling operations with improved precision
- Crop and scale CLI commands with better error handling


## [1.1.29] - 2025-11-02

### Changed
- 


## [1.1.28] - 2025-11-02

### Changed
- 


## [1.1.27] - 2025-11-02

### Changed
- 


## [1.1.26] - 2025-11-02

### Removed
- Removed MCP tool tests (convert_test.go, generate_test.go, models_test.go) that were incompatible with current implementation


## [1.1.23] - 2025-11-02

### Added
- Comprehensive model pricing and announcement system with cost tracking and latest model information
- Unit tests for generate command with coverage for both Gemini and Vertex AI backends
- Unit tests for convert operation with format conversion validation
- Unit tests for resize operation with comprehensive dimension and format testing
- Unit tests for crop operation with boundary and validation testing
- Automated changelog update script for release process

### Changed
- Enhanced generate command with model pricing display and cost estimation
- Improved MCP server with model information and pricing details
- Updated RELEASING.md with streamlined release workflow and automation improvements
- Refactored Makefile with improved test coverage reporting and build targets


## [1.1.19] - 2025-11-02

### Changed
- Build number incremented to 1.1.19 (automatic versioning from git commit count)


## [1.1.18] - 2025-11-01

### Changed
- Build number incremented to 1.1.18 (automatic versioning from git commit count)


## [1.1.9] - 2025-11-01

### Added
- **Automated version synchronization** between CLI and npm package
- **Build number versioning** using git commit count (format: 1.1.[build])
- WebP support via nativewebp library (pure Go, zero C dependencies)
- CLI `convert` command for format conversion
- Comprehensive integration tests for WebP
- End-to-end tests for all 10 MCP tools
- Help text displayed when running `gimage` with no arguments
- Complete release automation with GoReleaser
- GitHub Actions workflows for CI and releases
- npm package for MCP server distribution
- Homebrew tap for macOS/Linux distribution
- Comprehensive RELEASING.md guide
- `make version` and `make sync-version` commands

### Changed
- **Version numbering scheme** to 1.1.[commit_count] for automatic sync
- Root command now shows help instead of crashing when run without arguments
- All MCP tools now support WebP output format
- Homebrew tap repository renamed to `homebrew-tap` (conventional naming)
- Documentation updated for new distribution methods

### Fixed
- Root command exit behavior
- WebP encoding in all contexts (CLI, MCP server, programmatic usage)
- Version synchronization between CLI binary and npm package

## [0.1.1] - 2025-11-01

### Added
- Automatic format conversion based on output file extension
  - Specify `-o output.jpg` to save as JPEG
  - Specify `-o output.png` to save as PNG
  - Specify `-o output.gif` to save as GIF
  - Specify `-o output.bmp` to save as BMP
  - Specify `-o output.tiff` to save as TIFF
- Intelligent transparency handling (converts transparent areas to white when saving to formats that don't support transparency)
- Format normalization (automatically handles .jpg vs .jpeg, .tif vs .tiff)

### Changed
- SaveImage now automatically converts image format based on file extension

## [0.1.0] - 2025-11-01

### Added
- Initial release of gimage CLI tool
- AI-powered image generation using Google Gemini 2.5 Flash Image
- AI-powered image generation using Vertex AI Imagen 4
- Image processing operations:
  - Resize: Change image dimensions
  - Scale: Scale by percentage
  - Crop: Extract specific regions
  - Compress: Reduce file size
- Batch processing with concurrent operations
- MCP server for Claude integration
- Support for multiple image formats: PNG, JPG, WebP, GIF, TIFF, BMP
- Pure Go implementation with zero C dependencies
- Cross-platform support (Linux, macOS, Windows, ARM)
- Interactive authentication setup:
  - `gimage auth gemini` - Gemini API key setup
  - `gimage auth vertex` - Vertex AI setup (Express Mode or Full Mode)
- Smart credential detection - auto-selects API based on available credentials
- Configuration system with markdown-based config file (~/.gimage/config.md)
- Comprehensive CLI with Cobra framework

### Features
- Text-to-image generation with customizable prompts
- Multiple generation styles: photorealistic, artistic, anime
- Configurable image sizes and aspect ratios
- Negative prompts for image generation
- Seed support for reproducible results
- Verbose mode for debugging
- Model listing and auto-detection
- Express Mode for Vertex AI (API key authentication)
- Full Mode for Vertex AI (service account authentication)

### Technical
- Built with Go 1.22+
- Uses disintegration/imaging library for image processing
- Gemini API integration via REST
- Vertex AI integration via REST (Express Mode) and SDK (Full Mode)
- Concurrent batch processing with worker pools
- Comprehensive error handling and validation
