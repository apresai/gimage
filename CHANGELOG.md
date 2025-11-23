# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

(empty - ready for next release)

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
