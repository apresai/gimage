package generate

import (
	"context"
	"strings"
	"testing"

	"github.com/apresai/gimage/internal/observability"
	"github.com/apresai/gimage/pkg/models"
)

func newTestLogger() *observability.VerboseLogger {
	return observability.NewVerboseLogger(observability.ComponentBedrock)
}

func TestNewBedrockSDKClient(t *testing.T) {
	tests := []struct {
		name    string
		region  string
		wantErr bool
	}{
		{
			name:    "empty region defaults to us-east-1",
			region:  "",
			wantErr: false,
		},
		{
			name:    "valid region",
			region:  "us-west-2",
			wantErr: false,
		},
		{
			name:    "another valid region",
			region:  "eu-west-1",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client, err := NewBedrockSDKClient(ctx, tt.region)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBedrockSDKClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewBedrockSDKClient() returned nil client")
			}
			if client != nil {
				expectedRegion := tt.region
				if expectedRegion == "" {
					expectedRegion = "us-east-1"
				}
				if client.region != expectedRegion {
					t.Errorf("NewBedrockSDKClient() region = %v, want %v", client.region, expectedRegion)
				}
			}
		})
	}
}

func TestBedrockSDKClient_buildRequest(t *testing.T) {
	ctx := context.Background()
	client, err := NewBedrockSDKClient(ctx, "us-east-1")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tests := []struct {
		name        string
		prompt      string
		options     models.GenerateOptions
		wantErr     bool
		errContains string
	}{
		{
			name:   "valid request with defaults",
			prompt: "test prompt",
			options: models.GenerateOptions{
				Size: "1024x1024",
			},
			wantErr: false,
		},
		{
			name:   "valid request with seed",
			prompt: "test prompt with seed",
			options: models.GenerateOptions{
				Size: "512x512",
				Seed: 12345,
			},
			wantErr: false,
		},
		{
			name:   "valid request with negative prompt",
			prompt: "test prompt",
			options: models.GenerateOptions{
				Size:           "768x768",
				NegativePrompt: "avoid this",
			},
			wantErr: false,
		},
		{
			name:   "valid request with style (maps to quality)",
			prompt: "test prompt",
			options: models.GenerateOptions{
				Size:  "1024x1024",
				Style: "premium",
			},
			wantErr: false,
		},
		{
			name:   "empty prompt",
			prompt: "",
			options: models.GenerateOptions{
				Size: "1024x1024",
			},
			wantErr:     true,
			errContains: "prompt cannot be empty",
		},
		{
			name:   "dimensions auto-normalized - width too small becomes 512",
			prompt: "test prompt",
			options: models.GenerateOptions{
				Size: "256x512",
			},
			wantErr: false, // NormalizeDimensions auto-corrects to valid values
		},
		{
			name:   "dimensions auto-normalized - width too large becomes 2048",
			prompt: "test prompt",
			options: models.GenerateOptions{
				Size: "4096x1024",
			},
			wantErr: false, // NormalizeDimensions auto-corrects to valid values
		},
		{
			name:   "dimensions auto-normalized - width not multiple of 64 rounds up",
			prompt: "test prompt",
			options: models.GenerateOptions{
				Size: "1000x1024",
			},
			wantErr: false, // NormalizeDimensions auto-corrects to valid values
		},
		{
			name:   "dimensions auto-normalized - height too small becomes 512",
			prompt: "test prompt",
			options: models.GenerateOptions{
				Size: "512x256",
			},
			wantErr: false, // NormalizeDimensions auto-corrects to valid values
		},
		{
			name:   "dimensions auto-normalized - height too large becomes 2048",
			prompt: "test prompt",
			options: models.GenerateOptions{
				Size: "1024x4096",
			},
			wantErr: false, // NormalizeDimensions auto-corrects to valid values
		},
		{
			name:   "dimensions auto-normalized - height not multiple of 64 rounds up",
			prompt: "test prompt",
			options: models.GenerateOptions{
				Size: "1024x1000",
			},
			wantErr: false, // NormalizeDimensions auto-corrects to valid values
		},
		{
			name:   "seed too large",
			prompt: "test prompt",
			options: models.GenerateOptions{
				Size: "1024x1024",
				Seed: 999999999,
			},
			wantErr:     true,
			errContains: "invalid seed",
		},
		{
			name:   "seed negative",
			prompt: "test prompt",
			options: models.GenerateOptions{
				Size: "1024x1024",
				Seed: -1,
			},
			wantErr:     true,
			errContains: "invalid seed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := client.buildRequest(tt.prompt, tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("buildRequest() error = %v, should contain %q", err, tt.errContains)
				}
				return
			}
			if !tt.wantErr {
				if req == nil {
					t.Error("buildRequest() returned nil request")
					return
				}
				if req.TaskType != "TEXT_IMAGE" {
					t.Errorf("buildRequest() taskType = %v, want TEXT_IMAGE", req.TaskType)
				}
				if req.TextToImageParams.Text != tt.prompt {
					t.Errorf("buildRequest() prompt = %v, want %v", req.TextToImageParams.Text, tt.prompt)
				}
				if req.ImageGenerationConfig.Quality == "" {
					t.Error("buildRequest() quality should be set")
				}
			}
		})
	}
}

func TestBedrockSDKClient_GenerateImage_EmptyPrompt(t *testing.T) {
	ctx := context.Background()
	client, err := NewBedrockSDKClient(ctx, "us-east-1")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	_, err = client.GenerateImage(ctx, "", models.GenerateOptions{Size: "1024x1024"})
	if err == nil {
		t.Error("GenerateImage() with empty prompt should return error")
	}
	if err != nil && !strings.Contains(err.Error(), "prompt cannot be empty") {
		t.Errorf("GenerateImage() error = %v, should contain 'prompt cannot be empty'", err)
	}
}

func TestBedrockSDKClient_Close(t *testing.T) {
	ctx := context.Background()
	client, err := NewBedrockSDKClient(ctx, "us-east-1")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestBuildNovaCanvasRequest(t *testing.T) {
	log := newTestLogger()

	tests := []struct {
		name        string
		prompt      string
		options     models.GenerateOptions
		wantErr     bool
		errContains string
		validate    func(t *testing.T, req *NovaCanvasRequest)
	}{
		{
			name:   "defaults",
			prompt: "test prompt",
			options: models.GenerateOptions{
				Size: "1024x1024",
			},
			validate: func(t *testing.T, req *NovaCanvasRequest) {
				if req.ImageGenerationConfig.Quality != "standard" {
					t.Errorf("Quality = %q, want standard", req.ImageGenerationConfig.Quality)
				}
				if req.ImageGenerationConfig.CfgScale != 7.0 {
					t.Errorf("CfgScale = %f, want 7.0", req.ImageGenerationConfig.CfgScale)
				}
				if req.ImageGenerationConfig.NumberOfImages != 1 {
					t.Errorf("NumberOfImages = %d, want 1", req.ImageGenerationConfig.NumberOfImages)
				}
			},
		},
		{
			name:    "empty prompt",
			prompt:  "",
			options: models.GenerateOptions{Size: "1024x1024"},
			wantErr: true, errContains: "prompt cannot be empty",
		},
		{
			name:   "cfg scale clamped low",
			prompt: "test",
			options: models.GenerateOptions{
				Size:     "1024x1024",
				CfgScale: 0.5,
			},
			validate: func(t *testing.T, req *NovaCanvasRequest) {
				if req.ImageGenerationConfig.CfgScale != 1.0 {
					t.Errorf("CfgScale = %f, want 1.0 (clamped)", req.ImageGenerationConfig.CfgScale)
				}
			},
		},
		{
			name:   "cfg scale clamped high",
			prompt: "test",
			options: models.GenerateOptions{
				Size:     "1024x1024",
				CfgScale: 15.0,
			},
			validate: func(t *testing.T, req *NovaCanvasRequest) {
				if req.ImageGenerationConfig.CfgScale != 10.0 {
					t.Errorf("CfgScale = %f, want 10.0 (clamped)", req.ImageGenerationConfig.CfgScale)
				}
			},
		},
		{
			name:   "number of images clamped to 5",
			prompt: "test",
			options: models.GenerateOptions{
				Size:           "1024x1024",
				NumberOfImages: 10,
			},
			validate: func(t *testing.T, req *NovaCanvasRequest) {
				if req.ImageGenerationConfig.NumberOfImages != 5 {
					t.Errorf("NumberOfImages = %d, want 5 (clamped)", req.ImageGenerationConfig.NumberOfImages)
				}
			},
		},
		{
			name:   "style high maps to premium",
			prompt: "test",
			options: models.GenerateOptions{
				Size:  "1024x1024",
				Style: "high",
			},
			validate: func(t *testing.T, req *NovaCanvasRequest) {
				if req.ImageGenerationConfig.Quality != "premium" {
					t.Errorf("Quality = %q, want premium", req.ImageGenerationConfig.Quality)
				}
			},
		},
		{
			name:   "style ultra maps to premium",
			prompt: "test",
			options: models.GenerateOptions{
				Size:  "1024x1024",
				Style: "ultra",
			},
			validate: func(t *testing.T, req *NovaCanvasRequest) {
				if req.ImageGenerationConfig.Quality != "premium" {
					t.Errorf("Quality = %q, want premium", req.ImageGenerationConfig.Quality)
				}
			},
		},
		{
			name:   "negative prompt set",
			prompt: "test",
			options: models.GenerateOptions{
				Size:           "1024x1024",
				NegativePrompt: "no people",
			},
			validate: func(t *testing.T, req *NovaCanvasRequest) {
				if req.TextToImageParams.NegativeText != "no people" {
					t.Errorf("NegativeText = %q, want 'no people'", req.TextToImageParams.NegativeText)
				}
			},
		},
		{
			name:   "seed too large",
			prompt: "test",
			options: models.GenerateOptions{
				Size: "1024x1024",
				Seed: 999999999,
			},
			wantErr: true, errContains: "invalid seed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := BuildNovaCanvasRequest(tt.prompt, tt.options, log)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildNovaCanvasRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, should contain %q", err, tt.errContains)
				}
				return
			}
			if tt.validate != nil && req != nil {
				tt.validate(t, req)
			}
		})
	}
}

func TestBedrockQualityMapping(t *testing.T) {
	tests := []struct {
		name        string
		style       string
		wantQuality string
	}{
		{
			name:        "premium style",
			style:       "premium",
			wantQuality: "premium",
		},
		{
			name:        "standard style",
			style:       "standard",
			wantQuality: "standard",
		},
		{
			name:        "photorealistic style defaults to premium",
			style:       "photorealistic",
			wantQuality: "premium",
		},
		{
			name:        "artistic style defaults to standard",
			style:       "artistic",
			wantQuality: "standard",
		},
		{
			name:        "empty style defaults to standard",
			style:       "",
			wantQuality: "standard",
		},
	}

	ctx := context.Background()
	client, err := NewBedrockSDKClient(ctx, "us-east-1")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := client.buildRequest("test prompt", models.GenerateOptions{
				Size:  "1024x1024",
				Style: tt.style,
			})
			if err != nil {
				t.Fatalf("buildRequest() error = %v", err)
			}
			if req.ImageGenerationConfig.Quality != tt.wantQuality {
				t.Errorf("Quality = %v, want %v", req.ImageGenerationConfig.Quality, tt.wantQuality)
			}
		})
	}
}
