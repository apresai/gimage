# Testing Documentation

Comprehensive testing guide for gimage, including unit tests, integration tests, and validation suites.

## Table of Contents

- [Overview](#overview)
- [Test Structure](#test-structure)
- [Running Tests](#running-tests)
- [Unit Tests](#unit-tests)
- [Integration Tests](#integration-tests)
- [Test Fixtures](#test-fixtures)
- [Writing New Tests](#writing-new-tests)
- [Continuous Integration](#continuous-integration)

---

## Overview

Gimage uses a multi-layered testing approach:

1. **Unit Tests** - Fast, isolated tests for individual functions and components
2. **Integration Tests** - Real API calls to validate end-to-end workflows (costs money)
3. **Validation Tests** - Comprehensive image generation validation suite

**Coverage Requirements**: Minimum 80% test coverage for all packages.

---

## Test Structure

```
gimage/
├── internal/
│   ├── imaging/           # Image processing tests
│   │   └── *_test.go
│   ├── generate/          # Generation client tests
│   │   └── *_test.go
│   ├── config/            # Configuration tests
│   │   └── *_test.go
│   └── cli/               # CLI command tests
│       └── *_test.go
├── test/
│   ├── fixtures/          # Test images (DO NOT MODIFY)
│   │   ├── test_image.jpg
│   │   ├── test_image.png
│   │   └── test_image.webp
│   └── integration/       # Integration test suite
│       └── generate_validation_test.go
└── pkg/
    └── models/            # Model tests
        └── *_test.go
```

---

## Running Tests

### Quick Start

```bash
# Run all unit tests
make test

# Run with coverage
make test-coverage

# Run unit tests with verbose output
go test -v ./...

# Run specific package tests
go test -v ./internal/imaging
go test -v ./internal/generate
```

### Integration Tests

**IMPORTANT**: Integration tests make real API calls and **cost money**. They are not run by default.

```bash
# Run ALL integration tests (WARNING: costs money)
go test -tags=integration -v ./test/integration/...

# Run specific integration test
go test -tags=integration -v -run TestGeminiSquareImage ./test/integration/...

# Skip integration tests in short mode (default for `go test -short`)
go test -short ./...
```

**Authentication Required**: Integration tests require valid API credentials. Set environment variables before running:

```bash
export GEMINI_API_KEY="your-key-here"
export VERTEX_API_KEY="your-vertex-key"  # Optional
export AWS_ACCESS_KEY_ID="your-aws-key"  # Optional
```

---

## Unit Tests

Unit tests validate individual functions and components in isolation.

### What We Test

**Request Building**:
- JSON payload structure
- Parameter validation
- Model name resolution
- Size parsing

**Response Parsing**:
- Successful responses
- Error responses
- Edge cases (empty, malformed data)

**Configuration Loading**:
- Environment variables
- Config file parsing
- Priority hierarchy
- Credential masking

**CLI Flags**:
- Flag parsing
- Required vs optional flags
- Default values
- Validation

### Example Unit Test

```go
func TestGenerateOptions_Validate(t *testing.T) {
    tests := []struct {
        name    string
        opts    models.GenerateOptions
        wantErr bool
    }{
        {
            name: "valid options",
            opts: models.GenerateOptions{
                Model: "gemini-2.5-flash-image",
                Size:  "1024x1024",
            },
            wantErr: false,
        },
        {
            name: "invalid size format",
            opts: models.GenerateOptions{
                Model: "gemini-2.5-flash-image",
                Size:  "invalid",
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.opts.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Coverage Goals

- **Minimum 80% coverage** across all packages
- 100% coverage for critical paths (API calls, file operations)
- All error paths must be tested

```bash
# Check coverage
make test-coverage

# View coverage in browser
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## Integration Tests

Integration tests validate real-world functionality with actual API calls.

### Available Test Suites

Located in `/Users/chad/dev/gimage/test/integration/generate_validation_test.go`:

#### 1. Image Dimension Validation

**TestGeminiSquareImage** - Validates that Gemini generates correct dimensions
- Tests: 1024x1024 square image generation
- Validates: Exact dimensions with 10% tolerance
- Provider: Gemini 2.5 Flash (FREE tier)
- Cost: FREE

**TestGemini3ProImageSize** - Tests native resolution options
- Tests: 1K and 2K resolution generation
- Validates: Total pixel count ranges
- Provider: Gemini 3 Pro
- Cost: $0.134 per test run (1K), $0.134 per test run (2K)

**TestBedrockNovaCanvasDimensions** - Tests Bedrock dimension constraints
- Tests: 512x512, 1024x1024, 1024x768
- Validates: Exact dimensions
- Provider: AWS Bedrock Nova Canvas
- Cost: $0.04-$0.08 per test run

#### 2. Aspect Ratio Validation

**TestGemini3ProAspectRatio** - Tests aspect ratio controls
- Tests: 1:1, 16:9, 9:16 aspect ratios
- Validates: Aspect ratio within 5% tolerance
- Provider: Gemini 3 Pro
- Cost: $0.134 per ratio test (3 tests total = $0.40)

#### 3. Advanced Features

**TestSeedReproducibility** - Tests seed-based reproducibility
- Tests: Same seed generates similar results
- Validates: Matching dimensions across two generations
- Provider: Gemini 2.5 Flash (FREE tier)
- Cost: FREE
- Note: Exact pixel-by-pixel matching not guaranteed

**TestNegativePrompt** - Tests negative prompt processing
- Tests: Negative prompts don't cause errors
- Validates: Successful generation
- Provider: Gemini 2.5 Flash (FREE tier)
- Cost: FREE

**TestBedrockCfgScale** - Tests CFG scale parameter
- Tests: CFG values 1.0, 5.0, 10.0
- Validates: No errors at different scales
- Provider: AWS Bedrock Nova Canvas
- Cost: $0.12-$0.24 (3 tests)

#### 4. File Operations

**TestSaveAndLoadImage** - Tests complete save/load cycle
- Tests: Generate → Save → Load → Validate
- Validates: File size, format, dimensions
- Provider: Gemini 2.5 Flash (FREE tier)
- Cost: FREE

#### 5. Performance Benchmarks

**BenchmarkGeneration** - Benchmarks generation speed
- Measures: Time per image generation
- Provider: Gemini 2.5 Flash (FREE tier)
- Cost: FREE (uses existing allocations)

### Running Specific Tests

```bash
# Run all Gemini tests (FREE tier mostly)
go test -tags=integration -v -run TestGemini ./test/integration/...

# Run only square image test (FREE)
go test -tags=integration -v -run TestGeminiSquareImage ./test/integration/...

# Run Bedrock tests (PAID)
go test -tags=integration -v -run TestBedrock ./test/integration/...

# Run benchmarks
go test -tags=integration -bench=. ./test/integration/...
```

### Cost Summary

**FREE Tests** (using Gemini 2.5 Flash FREE tier):
- TestGeminiSquareImage
- TestSeedReproducibility
- TestNegativePrompt
- TestSaveAndLoadImage
- BenchmarkGeneration

**PAID Tests**:
- TestGemini3ProAspectRatio: ~$0.40
- TestGemini3ProImageSize: ~$0.27
- TestBedrockNovaCanvasDimensions: ~$0.12-$0.24
- TestBedrockCfgScale: ~$0.12-$0.24

**Total Cost for Full Suite**: ~$1.00-$1.50 per run

### Validation Functions

The integration test suite includes helper functions for validation:

**ValidateImage(data, expectedWidth, expectedHeight, expectedFormat)**
- Validates image dimensions (with 10% tolerance)
- Validates image format
- Returns detailed validation results

**ValidateAspectRatio(data, expectedRatio)**
- Validates aspect ratio (with 5% tolerance)
- Calculates GCD for exact ratio comparison
- Returns validation results with detailed errors

Example usage:
```go
validation, err := ValidateImage(result.Data, 1024, 1024, "png")
if err != nil {
    t.Fatalf("Validation failed: %v", err)
}

if !validation.IsValid {
    for _, e := range validation.Errors {
        t.Errorf("Validation error: %s", e)
    }
}

t.Logf("Generated image: %dx%d, format=%s, size=%d bytes",
    validation.Width, validation.Height, validation.Format, validation.FileSizeBytes)
```

---

## Test Fixtures

Located in `/Users/chad/dev/gimage/test/fixtures/`

**IMPORTANT**: DO NOT MODIFY test fixtures. They are used for reproducible testing.

Available fixtures:
- `test_image.jpg` - JPEG test image
- `test_image.png` - PNG test image with transparency
- `test_image.webp` - WebP test image

### Using Fixtures

```go
func TestImageProcessing(t *testing.T) {
    fixturePath := filepath.Join("../../test/fixtures", "test_image.jpg")
    img, err := imaging.Open(fixturePath)
    if err != nil {
        t.Fatalf("Failed to load fixture: %v", err)
    }

    // Test processing...
}
```

---

## Writing New Tests

### Test Naming Conventions

- Unit tests: `TestFunctionName`
- Table-driven tests: `TestFunctionName_Scenario`
- Integration tests: `TestProviderFeature` (e.g., `TestGeminiSquareImage`)
- Benchmarks: `BenchmarkOperation`

### Table-Driven Test Pattern

Preferred for testing multiple scenarios:

```go
func TestParseSize(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantW   int
        wantH   int
        wantErr bool
    }{
        {"valid square", "1024x1024", 1024, 1024, false},
        {"valid landscape", "1920x1080", 1920, 1080, false},
        {"invalid format", "1024", 0, 0, true},
        {"invalid chars", "1024xABC", 0, 0, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            w, h, err := ParseSize(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseSize() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if w != tt.wantW || h != tt.wantH {
                t.Errorf("ParseSize() = %v, %v, want %v, %v", w, h, tt.wantW, tt.wantH)
            }
        })
    }
}
```

### Integration Test Template

```go
// +build integration

func TestNewFeature(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // Check for required credentials
    apiKey, err := config.GetGeminiAPIKey("")
    if err != nil {
        t.Skipf("API key not configured: %v", err)
    }

    // Create client
    client, err := generate.NewGeminiRESTClient(apiKey)
    if err != nil {
        t.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()

    // Set timeout
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()

    // Test your feature...
    result, err := client.GenerateImage(ctx, "test prompt", opts)
    if err != nil {
        t.Fatalf("Generation failed: %v", err)
    }

    // Validate results
    validation, err := ValidateImage(result.Data, expectedW, expectedH, "png")
    if err != nil {
        t.Fatalf("Validation failed: %v", err)
    }

    if !validation.IsValid {
        for _, e := range validation.Errors {
            t.Errorf("Validation error: %s", e)
        }
    }
}
```

---

## Continuous Integration

### Local Pre-commit Checks

Before committing:

```bash
# Run all unit tests
make test

# Check coverage
make test-coverage

# Run linter
make lint

# Format code
go fmt ./...
```

### CI Pipeline (GitHub Actions)

The CI pipeline runs on every push and pull request:

1. **Linting**: golangci-lint checks code quality
2. **Unit Tests**: All unit tests with coverage reporting
3. **Build**: Ensures clean build for all platforms
4. **Integration Tests**: NOT run in CI (costs money)

Integration tests are run manually before releases.

---

## Best Practices

### DO

- ✅ Write table-driven tests for multiple scenarios
- ✅ Test both success and failure cases
- ✅ Use meaningful test names that describe the scenario
- ✅ Clean up resources (use `defer` for cleanup)
- ✅ Use `t.Helper()` for helper functions
- ✅ Skip expensive tests with `testing.Short()`
- ✅ Document why tests are skipped
- ✅ Validate all error messages
- ✅ Test edge cases (empty, nil, invalid input)

### DON'T

- ❌ Modify test fixtures (they're used for reproducibility)
- ❌ Hardcode absolute paths (use relative paths)
- ❌ Skip tests without explanation
- ❌ Test multiple things in one test function
- ❌ Ignore test failures ("it works on my machine")
- ❌ Write tests that depend on execution order
- ❌ Mock cloud provider APIs (integration tests are better)
- ❌ Commit failing tests
- ❌ Write tests without assertions

### Mocking Policy

**DO NOT MOCK cloud provider APIs** (Gemini, Vertex AI, Bedrock).

Why?
- Mocks test your mock, not the real API
- API contracts change without warning
- Real integration tests catch breaking changes
- FREE tier available (Gemini 2.5 Flash)

Instead:
- Use integration tests with real APIs
- Run manually before releases
- Use FREE tier models when possible
- Document costs clearly

---

## Troubleshooting Tests

### Test Failures

**"API key not configured"**
```bash
# Set environment variable
export GEMINI_API_KEY="your-key-here"

# Or use auth setup
gimage auth setup gemini
```

**"Timeout exceeded"**
- Increase timeout in test: `context.WithTimeout(ctx, 5*time.Minute)`
- Check internet connection
- Verify API service status

**"Dimension mismatch"**
- Integration tests use 10% tolerance
- Some APIs don't guarantee exact dimensions
- Check validation logic for appropriate tolerance

**"Failed to decode image"**
- Verify API returned valid image data
- Check response format (base64, binary)
- Validate content-type header

### Coverage Issues

**Coverage below 80%**
```bash
# Find untested code
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep -v "100.0%"
```

**Missing coverage in package**
```bash
# Test specific package
go test -cover ./internal/imaging
```

---

## Resources

- **Go Testing Package**: https://pkg.go.dev/testing
- **Table-Driven Tests**: https://dave.cheney.net/2019/05/07/prefer-table-driven-tests
- **Integration Testing**: https://go.dev/doc/tutorial/add-a-test
- **Coverage Tools**: https://go.dev/blog/cover

---

## Summary

- **Unit tests**: Fast, isolated, >80% coverage required
- **Integration tests**: Real API calls, manual runs, costs money
- **Validation suite**: Comprehensive image generation testing
- **Test fixtures**: DO NOT MODIFY, used for reproducibility
- **Best practice**: Table-driven tests, meaningful names, clean assertions
- **Mocking policy**: NO MOCKS for cloud APIs - use real integration tests

Run `make test` before every commit. Run integration tests manually before releases.
