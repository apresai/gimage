package generate

import (
	"context"
	"strings"
	"testing"

	"github.com/apresai/gimage/pkg/models"
)

func TestNewBedrockRESTClient(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		region  string
		wantErr bool
	}{
		{
			name:    "empty API key",
			apiKey:  "",
			region:  "us-east-1",
			wantErr: true,
		},
		{
			name:    "valid API key and region",
			apiKey:  "test-api-key-ABCDEFabcdef1234567890",
			region:  "us-east-1",
			wantErr: false,
		},
		{
			name:    "valid API key with empty region defaults to us-east-1",
			apiKey:  "test-api-key-ABCDEFabcdef1234567890",
			region:  "",
			wantErr: false,
		},
		{
			name:    "valid API key with different region",
			apiKey:  "test-api-key-ABCDEFabcdef1234567890",
			region:  "us-west-2",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewBedrockRESTClient(tt.apiKey, tt.region)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBedrockRESTClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if client == nil {
					t.Error("NewBedrockRESTClient() returned nil client")
					return
				}
				if client.apiKey != tt.apiKey {
					t.Errorf("NewBedrockRESTClient() apiKey = %v, want %v", client.apiKey, tt.apiKey)
				}
				expectedRegion := tt.region
				if expectedRegion == "" {
					expectedRegion = "us-east-1"
				}
				if client.region != expectedRegion {
					t.Errorf("NewBedrockRESTClient() region = %v, want %v", client.region, expectedRegion)
				}
				// Verify baseURL format
				expectedBaseURL := "https://bedrock-runtime." + expectedRegion + ".amazonaws.com"
				if client.baseURL != expectedBaseURL {
					t.Errorf("NewBedrockRESTClient() baseURL = %v, want %v", client.baseURL, expectedBaseURL)
				}
			}
		})
	}
}

func TestBedrockRESTClient_buildRequest(t *testing.T) {
	client, err := NewBedrockRESTClient("test-api-key", "us-east-1")
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
			name:   "valid request with all options",
			prompt: "detailed test prompt",
			options: models.GenerateOptions{
				Size:           "768x768",
				Seed:           42,
				NegativePrompt: "avoid ugly, distorted",
				Style:          "premium",
			},
			wantErr: false,
		},
		{
			name:   "valid request minimum size",
			prompt: "test prompt",
			options: models.GenerateOptions{
				Size: "512x512",
			},
			wantErr: false,
		},
		{
			name:   "valid request maximum size",
			prompt: "test prompt",
			options: models.GenerateOptions{
				Size: "2048x2048",
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
				Size: "256x1024",
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
			name:   "dimensions auto-normalized - not multiple of 64 rounds up",
			prompt: "test prompt",
			options: models.GenerateOptions{
				Size: "1000x1024",
			},
			wantErr: false, // NormalizeDimensions auto-corrects to valid values
		},
		{
			name:   "seed out of range - too large",
			prompt: "test prompt",
			options: models.GenerateOptions{
				Size: "1024x1024",
				Seed: 999999999,
			},
			wantErr:     true,
			errContains: "invalid seed",
		},
		{
			name:   "seed out of range - negative",
			prompt: "test prompt",
			options: models.GenerateOptions{
				Size: "1024x1024",
				Seed: -100,
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
				// Verify dimensions
				if req.ImageGenerationConfig.Width == 0 || req.ImageGenerationConfig.Height == 0 {
					t.Error("buildRequest() dimensions not set")
				}
				// Verify quality is set
				if req.ImageGenerationConfig.Quality == "" {
					t.Error("buildRequest() quality should be set")
				}
			}
		})
	}
}

func TestBedrockRESTClient_GenerateImage_EmptyPrompt(t *testing.T) {
	client, err := NewBedrockRESTClient("test-api-key", "us-east-1")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	_, err = client.GenerateImage(ctx, "", models.GenerateOptions{Size: "1024x1024"})
	if err == nil {
		t.Error("GenerateImage() with empty prompt should return error")
	}
	if err != nil && !strings.Contains(err.Error(), "prompt cannot be empty") {
		t.Errorf("GenerateImage() error = %v, should contain 'prompt cannot be empty'", err)
	}
}

func TestBedrockRESTClient_Close(t *testing.T) {
	client, err := NewBedrockRESTClient("test-api-key", "us-east-1")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestBedrockRESTClient_URLConstruction(t *testing.T) {
	tests := []struct {
		name            string
		region          string
		expectedBaseURL string
	}{
		{
			name:            "us-east-1",
			region:          "us-east-1",
			expectedBaseURL: "https://bedrock-runtime.us-east-1.amazonaws.com",
		},
		{
			name:            "us-west-2",
			region:          "us-west-2",
			expectedBaseURL: "https://bedrock-runtime.us-west-2.amazonaws.com",
		},
		{
			name:            "eu-west-1",
			region:          "eu-west-1",
			expectedBaseURL: "https://bedrock-runtime.eu-west-1.amazonaws.com",
		},
		{
			name:            "ap-southeast-1",
			region:          "ap-southeast-1",
			expectedBaseURL: "https://bedrock-runtime.ap-southeast-1.amazonaws.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewBedrockRESTClient("test-api-key", tt.region)
			if err != nil {
				t.Fatalf("NewBedrockRESTClient() error = %v", err)
			}
			if client.baseURL != tt.expectedBaseURL {
				t.Errorf("baseURL = %v, want %v", client.baseURL, tt.expectedBaseURL)
			}
		})
	}
}

func TestBedrockRESTClient_QualityMapping(t *testing.T) {
	client, err := NewBedrockRESTClient("test-api-key", "us-east-1")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

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
			name:        "photorealistic maps to premium",
			style:       "photorealistic",
			wantQuality: "premium",
		},
		{
			name:        "artistic maps to standard",
			style:       "artistic",
			wantQuality: "standard",
		},
		{
			name:        "empty defaults to standard",
			style:       "",
			wantQuality: "standard",
		},
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

func TestBedrockRESTClient_NegativePromptHandling(t *testing.T) {
	client, err := NewBedrockRESTClient("test-api-key", "us-east-1")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tests := []struct {
		name           string
		negativePrompt string
		wantSet        bool
	}{
		{
			name:           "with negative prompt",
			negativePrompt: "avoid this",
			wantSet:        true,
		},
		{
			name:           "empty negative prompt",
			negativePrompt: "",
			wantSet:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := client.buildRequest("test prompt", models.GenerateOptions{
				Size:           "1024x1024",
				NegativePrompt: tt.negativePrompt,
			})
			if err != nil {
				t.Fatalf("buildRequest() error = %v", err)
			}
			hasNegativePrompt := req.TextToImageParams.NegativeText != ""
			if hasNegativePrompt != tt.wantSet {
				t.Errorf("NegativeText set = %v, want %v", hasNegativePrompt, tt.wantSet)
			}
			if tt.wantSet && req.TextToImageParams.NegativeText != tt.negativePrompt {
				t.Errorf("NegativeText = %v, want %v", req.TextToImageParams.NegativeText, tt.negativePrompt)
			}
		})
	}
}
