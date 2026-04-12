// Package generate provides a unified provider system for managing model access
// across different APIs with clear credential requirements and pricing.
package generate

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/apresai/gimage/internal/config"
	"github.com/apresai/gimage/pkg/models"
)

// Model ID constants for backward compatibility
const (
	DefaultModel    = "gemini-3-pro-image-preview"
	ModelNovaCanvas = "amazon.nova-canvas-v1:0"
)

// ImageGenerator is the common interface for all image generation clients
type ImageGenerator interface {
	GenerateImage(ctx context.Context, prompt string, options models.GenerateOptions) ([]*models.GeneratedImage, error)
	Close() error
}

// Provider represents a specific way to access a model (API + Model + Auth)
type Provider struct {
	// Unique identifier: "provider/model" e.g., "gemini/flash-2.5", "vertex/imagen-4"
	ID string

	// Display information
	Name        string // e.g., "Gemini 2.5 Flash (via Gemini API)"
	API         string // "gemini", "vertex", "bedrock"
	ModelID     string // Actual model identifier for the API
	Description string

	// Authentication requirements
	RequiredEnvVars []EnvVar // Exactly which env vars/config keys are needed

	// Pricing (specific to this provider)
	Pricing PricingInfo

	// Capabilities
	Capabilities ModelCapabilities

	// Client factory
	CreateClient func(creds map[string]string) (ImageGenerator, error)
}

// EnvVar represents a required environment variable or config key
type EnvVar struct {
	Name        string // e.g., "GEMINI_API_KEY"
	ConfigKey   string // e.g., "gemini_api_key" in config file
	Description string // e.g., "API key from https://aistudio.google.com"
	Required    bool   // Is this absolutely required?
	Secret      bool   // Should we hide this value in output?
}

// PricingInfo represents pricing information for a provider
type PricingInfo struct {
	CostPerImage  *float64 // USD per image (nil = variable/unknown)
	FreeTier      bool     // Has free tier
	FreeTierLimit string   // Free tier description (e.g., "500 images/day")
	Currency      string   // "USD", etc.
}

// ModelCapabilities represents what features a model supports
type ModelCapabilities struct {
	SupportsStyles         bool
	SupportsNegativePrompt bool
	SupportsSeed           bool
	MaxPromptLength        int
}

// Helper function for creating float64 pointers
func float64Ptr(f float64) *float64 { return &f }

// APIInfo contains metadata about an API backend
type APIInfo struct {
	ID          string // "gemini", "vertex", "bedrock", "grok"
	DisplayName string // "Gemini API (Google AI Studio)"
	Description string // Brief description
	PricingNote string // e.g., "Free tier: 500 images/day"
	Order       int    // Display order (lower = first)
}

// apiRegistry maps API IDs to their metadata (single source of truth)
var apiRegistry = map[string]APIInfo{
	"gemini": {
		ID:          "gemini",
		DisplayName: "Gemini API (Google AI Studio)",
		Description: "Direct access via Google AI Studio",
		PricingNote: "Paid: $0.039-0.24 per image",
		Order:       1,
	},
	"vertex": {
		ID:          "vertex",
		DisplayName: "Vertex AI (Google Cloud)",
		Description: "Google Cloud's enterprise AI platform",
		PricingNote: "Paid: $0.02-0.06 per image",
		Order:       2,
	},
	"bedrock": {
		ID:          "bedrock",
		DisplayName: "AWS Bedrock",
		Description: "Amazon's managed AI service",
		PricingNote: "Paid: $0.04-0.08 per image",
		Order:       3,
	},
	"grok": {
		ID:          "grok",
		DisplayName: "xAI Grok",
		Description: "xAI's Aurora-powered image generation",
		PricingNote: "Paid: $0.02-0.07 per image",
		Order:       4,
	},
}

// ProviderRegistry manages all available providers
type ProviderRegistry struct {
	providers map[string]*Provider
}

// Global provider registry
var providerRegistry = NewProviderRegistry()

// NewProviderRegistry creates and initializes the registry
func NewProviderRegistry() *ProviderRegistry {
	r := &ProviderRegistry{
		providers: make(map[string]*Provider),
	}
	r.registerAllProviders()
	return r
}

// GetProviderRegistry returns the global registry
func GetProviderRegistry() *ProviderRegistry {
	return providerRegistry
}

func (r *ProviderRegistry) registerAllProviders() {
	// Gemini 2.5 Flash via Gemini API
	r.Register(&Provider{
		ID:          "gemini/flash-2.5",
		Name:        "Gemini 2.5 Flash (via Gemini API)",
		API:         "gemini",
		ModelID:     "gemini-2.5-flash-image",
		Description: "Direct access via Google AI Studio - Simple, affordable",
		RequiredEnvVars: []EnvVar{
			{
				Name:        "GEMINI_API_KEY",
				ConfigKey:   "gemini_api_key",
				Description: "API key from https://aistudio.google.com",
				Required:    true,
				Secret:      true,
			},
		},
		Pricing: PricingInfo{
			CostPerImage: float64Ptr(0.039),
			FreeTier:     false,
			Currency:     "USD",
		},
		Capabilities: ModelCapabilities{
			SupportsStyles:         true,
			SupportsNegativePrompt: true,
			SupportsSeed:           true,
			MaxPromptLength:        480,
		},
		CreateClient: func(creds map[string]string) (ImageGenerator, error) {
			apiKey := creds["GEMINI_API_KEY"]
			if apiKey == "" {
				return nil, fmt.Errorf("GEMINI_API_KEY is required")
			}
			return NewGeminiRESTClient(apiKey)
		},
	})

	// Gemini 3.1 Flash Image Preview via Gemini API
	r.Register(&Provider{
		ID:          "gemini/flash-3.1",
		Name:        "Gemini 3.1 Flash (via Gemini API)",
		API:         "gemini",
		ModelID:     "gemini-3.1-flash-image-preview",
		Description: "4K resolution, improved text rendering - fast and affordable",
		RequiredEnvVars: []EnvVar{
			{
				Name:        "GEMINI_API_KEY",
				ConfigKey:   "gemini_api_key",
				Description: "API key from https://aistudio.google.com",
				Required:    true,
				Secret:      true,
			},
		},
		// TODO(gemini-3.1-flash): pricing and MaxPromptLength are estimates —
		// ai.google.dev/gemini-api/docs/pricing does not list the preview model
		// as of 2026-04-12. Verify once Google publishes official pricing.
		Pricing: PricingInfo{
			CostPerImage: float64Ptr(0.05),
			FreeTier:     false,
			Currency:     "USD",
		},
		Capabilities: ModelCapabilities{
			SupportsStyles:         true,
			SupportsNegativePrompt: true,
			SupportsSeed:           true,
			MaxPromptLength:        2000,
		},
		CreateClient: func(creds map[string]string) (ImageGenerator, error) {
			apiKey := creds["GEMINI_API_KEY"]
			if apiKey == "" {
				return nil, fmt.Errorf("GEMINI_API_KEY is required")
			}
			return NewGeminiRESTClient(apiKey)
		},
	})

	// Gemini 3 Pro Image Preview via Gemini API
	r.Register(&Provider{
		ID:          "gemini/pro-3",
		Name:        "Gemini 3 Pro (via Gemini API)",
		API:         "gemini",
		ModelID:     "gemini-3-pro-image-preview",
		Description: "Native 4K, sharp text rendering, grounded generation with Google Search",
		RequiredEnvVars: []EnvVar{
			{
				Name:        "GEMINI_API_KEY",
				ConfigKey:   "gemini_api_key",
				Description: "API key from https://aistudio.google.com",
				Required:    true,
				Secret:      true,
			},
		},
		Pricing: PricingInfo{
			CostPerImage:  float64Ptr(0.134), // $0.134 for 1K/2K, $0.24 for 4K
			FreeTier:      false,
			FreeTierLimit: "",
			Currency:      "USD",
		},
		Capabilities: ModelCapabilities{
			SupportsStyles:         true,
			SupportsNegativePrompt: true,
			SupportsSeed:           true,
			MaxPromptLength:        2000, // 65k context window
		},
		CreateClient: func(creds map[string]string) (ImageGenerator, error) {
			apiKey := creds["GEMINI_API_KEY"]
			if apiKey == "" {
				return nil, fmt.Errorf("GEMINI_API_KEY is required")
			}
			return NewGeminiRESTClient(apiKey)
		},
	})

	// Imagen 4 via Vertex AI
	r.Register(&Provider{
		ID:          "vertex/imagen-4",
		Name:        "Imagen 4 (via Vertex AI)",
		API:         "vertex",
		ModelID:     "imagen-4.0-generate-001",
		Description: "Google's premium image generation model",
		RequiredEnvVars: []EnvVar{
			{
				Name:        "VERTEX_PROJECT",
				ConfigKey:   "vertex_project",
				Description: "GCP Project ID",
				Required:    true,
				Secret:      false,
			},
			{
				Name:        "VERTEX_LOCATION",
				ConfigKey:   "vertex_location",
				Description: "GCP region (e.g., us-central1)",
				Required:    true,
				Secret:      false,
			},
			{
				Name:        "VERTEX_API_KEY",
				ConfigKey:   "vertex_api_key",
				Description: "Vertex AI API key (optional)",
				Required:    false,
				Secret:      true,
			},
		},
		Pricing: PricingInfo{
			CostPerImage: float64Ptr(0.04),
			FreeTier:     false,
			Currency:     "USD",
		},
		Capabilities: ModelCapabilities{
			SupportsStyles:         true,
			SupportsNegativePrompt: true,
			SupportsSeed:           true,
			MaxPromptLength:        2000,
		},
		CreateClient: func(creds map[string]string) (ImageGenerator, error) {
			project := creds["VERTEX_PROJECT"]
			location := creds["VERTEX_LOCATION"]
			apiKey := creds["VERTEX_API_KEY"]

			if project == "" || location == "" {
				return nil, fmt.Errorf("VERTEX_PROJECT and VERTEX_LOCATION are required")
			}

			if apiKey != "" {
				return NewVertexRESTClient(apiKey, project, location)
			}
			ctx := context.Background()
			return NewVertexUnifiedClient(ctx, project, location)
		},
	})

	// Imagen 4 Fast via Vertex AI
	r.Register(&Provider{
		ID:          "vertex/imagen-4-fast",
		Name:        "Imagen 4 Fast (via Vertex AI)",
		API:         "vertex",
		ModelID:     "imagen-4.0-fast-generate-001",
		Description: "Google's Imagen 4 optimized for speed",
		RequiredEnvVars: []EnvVar{
			{
				Name:        "VERTEX_PROJECT",
				ConfigKey:   "vertex_project",
				Description: "GCP Project ID",
				Required:    true,
				Secret:      false,
			},
			{
				Name:        "VERTEX_LOCATION",
				ConfigKey:   "vertex_location",
				Description: "GCP region (e.g., us-central1)",
				Required:    true,
				Secret:      false,
			},
			{
				Name:        "VERTEX_API_KEY",
				ConfigKey:   "vertex_api_key",
				Description: "Vertex AI API key (optional)",
				Required:    false,
				Secret:      true,
			},
		},
		Pricing: PricingInfo{
			CostPerImage: float64Ptr(0.02),
			FreeTier:     false,
			Currency:     "USD",
		},
		Capabilities: ModelCapabilities{
			SupportsStyles:         true,
			SupportsNegativePrompt: true,
			SupportsSeed:           true,
			MaxPromptLength:        2000,
		},
		CreateClient: func(creds map[string]string) (ImageGenerator, error) {
			project := creds["VERTEX_PROJECT"]
			location := creds["VERTEX_LOCATION"]
			apiKey := creds["VERTEX_API_KEY"]

			if project == "" || location == "" {
				return nil, fmt.Errorf("VERTEX_PROJECT and VERTEX_LOCATION are required")
			}

			if apiKey != "" {
				return NewVertexRESTClient(apiKey, project, location)
			}
			ctx := context.Background()
			return NewVertexUnifiedClient(ctx, project, location)
		},
	})

	// Imagen 4 Ultra via Vertex AI
	r.Register(&Provider{
		ID:          "vertex/imagen-4-ultra",
		Name:        "Imagen 4 Ultra (via Vertex AI)",
		API:         "vertex",
		ModelID:     "imagen-4.0-ultra-generate-001",
		Description: "Google's Imagen 4 highest quality model",
		RequiredEnvVars: []EnvVar{
			{
				Name:        "VERTEX_PROJECT",
				ConfigKey:   "vertex_project",
				Description: "GCP Project ID",
				Required:    true,
				Secret:      false,
			},
			{
				Name:        "VERTEX_LOCATION",
				ConfigKey:   "vertex_location",
				Description: "GCP region (e.g., us-central1)",
				Required:    true,
				Secret:      false,
			},
			{
				Name:        "VERTEX_API_KEY",
				ConfigKey:   "vertex_api_key",
				Description: "Vertex AI API key (optional)",
				Required:    false,
				Secret:      true,
			},
		},
		Pricing: PricingInfo{
			CostPerImage: float64Ptr(0.06),
			FreeTier:     false,
			Currency:     "USD",
		},
		Capabilities: ModelCapabilities{
			SupportsStyles:         true,
			SupportsNegativePrompt: true,
			SupportsSeed:           true,
			MaxPromptLength:        2000,
		},
		CreateClient: func(creds map[string]string) (ImageGenerator, error) {
			project := creds["VERTEX_PROJECT"]
			location := creds["VERTEX_LOCATION"]
			apiKey := creds["VERTEX_API_KEY"]

			if project == "" || location == "" {
				return nil, fmt.Errorf("VERTEX_PROJECT and VERTEX_LOCATION are required")
			}

			if apiKey != "" {
				return NewVertexRESTClient(apiKey, project, location)
			}
			ctx := context.Background()
			return NewVertexUnifiedClient(ctx, project, location)
		},
	})

	// Imagen 3 (latest, legacy) via Vertex AI
	r.Register(&Provider{
		ID:          "vertex/imagen-3",
		Name:        "Imagen 3 (via Vertex AI)",
		API:         "vertex",
		ModelID:     "imagen-3.0-generate-002",
		Description: "Google's Imagen 3 model (legacy, prefer Imagen 4)",
		RequiredEnvVars: []EnvVar{
			{
				Name:        "VERTEX_PROJECT",
				ConfigKey:   "vertex_project",
				Description: "GCP Project ID",
				Required:    true,
				Secret:      false,
			},
			{
				Name:        "VERTEX_LOCATION",
				ConfigKey:   "vertex_location",
				Description: "GCP region (e.g., us-central1)",
				Required:    true,
				Secret:      false,
			},
			{
				Name:        "VERTEX_API_KEY",
				ConfigKey:   "vertex_api_key",
				Description: "Vertex AI API key (optional)",
				Required:    false,
				Secret:      true,
			},
		},
		Pricing: PricingInfo{
			CostPerImage: float64Ptr(0.04),
			FreeTier:     false,
			Currency:     "USD",
		},
		Capabilities: ModelCapabilities{
			SupportsStyles:         true,
			SupportsNegativePrompt: true,
			SupportsSeed:           true,
			MaxPromptLength:        2000,
		},
		CreateClient: func(creds map[string]string) (ImageGenerator, error) {
			project := creds["VERTEX_PROJECT"]
			location := creds["VERTEX_LOCATION"]
			apiKey := creds["VERTEX_API_KEY"]

			if project == "" || location == "" {
				return nil, fmt.Errorf("VERTEX_PROJECT and VERTEX_LOCATION are required")
			}

			if apiKey != "" {
				return NewVertexRESTClient(apiKey, project, location)
			}
			ctx := context.Background()
			return NewVertexUnifiedClient(ctx, project, location)
		},
	})

	// Imagen 3 (standard) via Vertex AI
	r.Register(&Provider{
		ID:          "vertex/imagen-3-standard",
		Name:        "Imagen 3 Standard (via Vertex AI)",
		API:         "vertex",
		ModelID:     "imagen-3.0-generate-001",
		Description: "Google's Imagen 3 model, standard quality (legacy)",
		RequiredEnvVars: []EnvVar{
			{
				Name:        "VERTEX_PROJECT",
				ConfigKey:   "vertex_project",
				Description: "GCP Project ID",
				Required:    true,
				Secret:      false,
			},
			{
				Name:        "VERTEX_LOCATION",
				ConfigKey:   "vertex_location",
				Description: "GCP region (e.g., us-central1)",
				Required:    true,
				Secret:      false,
			},
			{
				Name:        "VERTEX_API_KEY",
				ConfigKey:   "vertex_api_key",
				Description: "Vertex AI API key (optional)",
				Required:    false,
				Secret:      true,
			},
		},
		Pricing: PricingInfo{
			CostPerImage: float64Ptr(0.04),
			FreeTier:     false,
			Currency:     "USD",
		},
		Capabilities: ModelCapabilities{
			SupportsStyles:         true,
			SupportsNegativePrompt: true,
			SupportsSeed:           true,
			MaxPromptLength:        2000,
		},
		CreateClient: func(creds map[string]string) (ImageGenerator, error) {
			project := creds["VERTEX_PROJECT"]
			location := creds["VERTEX_LOCATION"]
			apiKey := creds["VERTEX_API_KEY"]

			if project == "" || location == "" {
				return nil, fmt.Errorf("VERTEX_PROJECT and VERTEX_LOCATION are required")
			}

			if apiKey != "" {
				return NewVertexRESTClient(apiKey, project, location)
			}
			ctx := context.Background()
			return NewVertexUnifiedClient(ctx, project, location)
		},
	})

	// Imagen 3 Fast via Vertex AI
	r.Register(&Provider{
		ID:          "vertex/imagen-3-fast",
		Name:        "Imagen 3 Fast (via Vertex AI)",
		API:         "vertex",
		ModelID:     "imagen-3.0-fast-generate-001",
		Description: "Google's Imagen 3 Fast model (legacy, prefer Imagen 4 Fast)",
		RequiredEnvVars: []EnvVar{
			{
				Name:        "VERTEX_PROJECT",
				ConfigKey:   "vertex_project",
				Description: "GCP Project ID",
				Required:    true,
				Secret:      false,
			},
			{
				Name:        "VERTEX_LOCATION",
				ConfigKey:   "vertex_location",
				Description: "GCP region (e.g., us-central1)",
				Required:    true,
				Secret:      false,
			},
			{
				Name:        "VERTEX_API_KEY",
				ConfigKey:   "vertex_api_key",
				Description: "Vertex AI API key (optional)",
				Required:    false,
				Secret:      true,
			},
		},
		Pricing: PricingInfo{
			CostPerImage: float64Ptr(0.02),
			FreeTier:     false,
			Currency:     "USD",
		},
		Capabilities: ModelCapabilities{
			SupportsStyles:         true,
			SupportsNegativePrompt: true,
			SupportsSeed:           true,
			MaxPromptLength:        2000,
		},
		CreateClient: func(creds map[string]string) (ImageGenerator, error) {
			project := creds["VERTEX_PROJECT"]
			location := creds["VERTEX_LOCATION"]
			apiKey := creds["VERTEX_API_KEY"]

			if project == "" || location == "" {
				return nil, fmt.Errorf("VERTEX_PROJECT and VERTEX_LOCATION are required")
			}

			if apiKey != "" {
				return NewVertexRESTClient(apiKey, project, location)
			}
			ctx := context.Background()
			return NewVertexUnifiedClient(ctx, project, location)
		},
	})

	// Grok Imagine via xAI (new default)
	r.Register(&Provider{
		ID:          "grok/grok-imagine",
		Name:        "Grok Imagine (via xAI)",
		API:         "grok",
		ModelID:     "grok-imagine-image",
		Description: "xAI's latest image generation model - fast and affordable",
		RequiredEnvVars: []EnvVar{
			{
				Name:        "GROK_API_KEY",
				ConfigKey:   "grok_api_key",
				Description: "API key from https://console.x.ai",
				Required:    true,
				Secret:      true,
			},
		},
		Pricing: PricingInfo{
			CostPerImage: float64Ptr(0.02),
			FreeTier:     false,
			Currency:     "USD",
		},
		Capabilities: ModelCapabilities{
			SupportsStyles:         false,
			SupportsNegativePrompt: false,
			SupportsSeed:           false,
			MaxPromptLength:        8000,
		},
		CreateClient: func(creds map[string]string) (ImageGenerator, error) {
			apiKey := creds["GROK_API_KEY"]
			if apiKey == "" {
				return nil, fmt.Errorf("GROK_API_KEY is required")
			}
			return NewGrokClient(apiKey)
		},
	})

	// Grok Imagine Pro via xAI (higher quality)
	r.Register(&Provider{
		ID:          "grok/grok-imagine-pro",
		Name:        "Grok Imagine Pro (via xAI)",
		API:         "grok",
		ModelID:     "grok-imagine-image-pro",
		Description: "xAI's premium image generation model - higher quality",
		RequiredEnvVars: []EnvVar{
			{
				Name:        "GROK_API_KEY",
				ConfigKey:   "grok_api_key",
				Description: "API key from https://console.x.ai",
				Required:    true,
				Secret:      true,
			},
		},
		Pricing: PricingInfo{
			CostPerImage: float64Ptr(0.07),
			FreeTier:     false,
			Currency:     "USD",
		},
		Capabilities: ModelCapabilities{
			SupportsStyles:         false,
			SupportsNegativePrompt: false,
			SupportsSeed:           false,
			MaxPromptLength:        8000,
		},
		CreateClient: func(creds map[string]string) (ImageGenerator, error) {
			apiKey := creds["GROK_API_KEY"]
			if apiKey == "" {
				return nil, fmt.Errorf("GROK_API_KEY is required")
			}
			return NewGrokClient(apiKey)
		},
	})

	// Grok 2 Image via xAI (legacy)
	r.Register(&Provider{
		ID:          "grok/grok-2-image",
		Name:        "Grok 2 Image (via xAI)",
		API:         "grok",
		ModelID:     "grok-2-image-1212", // Versioned model name works more reliably
		Description: "xAI's Aurora-powered image generation model (legacy, prefer Grok Imagine)",
		RequiredEnvVars: []EnvVar{
			{
				Name:        "GROK_API_KEY",
				ConfigKey:   "grok_api_key",
				Description: "API key from https://console.x.ai",
				Required:    true,
				Secret:      true,
			},
		},
		Pricing: PricingInfo{
			CostPerImage: float64Ptr(0.07),
			FreeTier:     false,
			Currency:     "USD",
		},
		Capabilities: ModelCapabilities{
			SupportsStyles:         false,
			SupportsNegativePrompt: false,
			SupportsSeed:           false,
			MaxPromptLength:        4000,
		},
		CreateClient: func(creds map[string]string) (ImageGenerator, error) {
			apiKey := creds["GROK_API_KEY"]
			if apiKey == "" {
				return nil, fmt.Errorf("GROK_API_KEY is required")
			}
			return NewGrokClient(apiKey)
		},
	})

	// Nova Canvas via Bedrock
	r.Register(&Provider{
		ID:          "bedrock/nova-canvas",
		Name:        "Nova Canvas (via AWS Bedrock)",
		API:         "bedrock",
		ModelID:     "amazon.nova-canvas-v1:0",
		Description: "Amazon's AI image generation model",
		RequiredEnvVars: []EnvVar{
			{
				Name:        "AWS_REGION",
				ConfigKey:   "aws_region",
				Description: "AWS region (e.g., us-east-1)",
				Required:    true,
				Secret:      false,
			},
			{
				Name:        "AWS_BEDROCK_API_KEY",
				ConfigKey:   "aws_bedrock_api_key",
				Description: "Bearer token for Bedrock REST API (optional)",
				Required:    false,
				Secret:      true,
			},
			{
				Name:        "AWS_ACCESS_KEY_ID",
				ConfigKey:   "aws_access_key_id",
				Description: "AWS Access Key (if not using bearer token)",
				Required:    false,
				Secret:      false,
			},
			{
				Name:        "AWS_SECRET_ACCESS_KEY",
				ConfigKey:   "aws_secret_access_key",
				Description: "AWS Secret Key (if not using bearer token)",
				Required:    false,
				Secret:      true,
			},
		},
		Pricing: PricingInfo{
			CostPerImage: float64Ptr(0.04), // $0.04 for ≤1024x1024, $0.08 for >1024x1024
			FreeTier:     false,
			Currency:     "USD",
		},
		Capabilities: ModelCapabilities{
			SupportsStyles:         true,
			SupportsNegativePrompt: true,
			SupportsSeed:           true,
			MaxPromptLength:        4096,
		},
		CreateClient: func(creds map[string]string) (ImageGenerator, error) {
			region := creds["AWS_REGION"]
			if region == "" {
				return nil, fmt.Errorf("AWS_REGION is required")
			}

			// Try bearer token first
			if bearerToken := creds["AWS_BEDROCK_API_KEY"]; bearerToken != "" {
				return NewBedrockRESTClient(bearerToken, region)
			}

			// Fall back to SDK
			ctx := context.Background()
			return NewBedrockSDKClient(ctx, region)
		},
	})
}

// Register adds a provider to the registry
func (r *ProviderRegistry) Register(p *Provider) {
	r.providers[p.ID] = p
}

// Get retrieves a provider by ID
func (r *ProviderRegistry) Get(id string) (*Provider, error) {
	if p, ok := r.providers[id]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("provider not found: %s", id)
}

// List returns all registered providers
func (r *ProviderRegistry) List() []*Provider {
	providers := make([]*Provider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	return providers
}

// ListByAPI returns providers for a specific API
func (r *ProviderRegistry) ListByAPI(api string) []*Provider {
	var providers []*Provider
	for _, p := range r.providers {
		if p.API == api {
			providers = append(providers, p)
		}
	}
	return providers
}

// GetAPIInfo returns metadata for an API (single source of truth)
func (r *ProviderRegistry) GetAPIInfo(api string) (APIInfo, bool) {
	info, ok := apiRegistry[api]
	return info, ok
}

// ListAPIs returns all unique APIs in display order
func (r *ProviderRegistry) ListAPIs() []APIInfo {
	// Collect unique APIs from registered providers
	seenAPIs := make(map[string]bool)
	for _, p := range r.providers {
		seenAPIs[p.API] = true
	}

	// Build list of API info for APIs that have providers
	apis := make([]APIInfo, 0, len(seenAPIs))
	for api := range seenAPIs {
		if info, ok := apiRegistry[api]; ok {
			apis = append(apis, info)
		}
	}

	// Sort by Order field
	for i := 0; i < len(apis)-1; i++ {
		for j := i + 1; j < len(apis); j++ {
			if apis[i].Order > apis[j].Order {
				apis[i], apis[j] = apis[j], apis[i]
			}
		}
	}

	return apis
}

// GroupAuthStatusByAPI groups auth statuses by API in display order
func (r *ProviderRegistry) GroupAuthStatusByAPI() map[string][]AuthStatus {
	statuses := r.GetAuthStatus()
	grouped := make(map[string][]AuthStatus)

	for _, status := range statuses {
		api := status.Provider.API
		grouped[api] = append(grouped[api], status)
	}

	return grouped
}

// CheckAuth checks if a provider has all required credentials
func (r *ProviderRegistry) CheckAuth(p *Provider) (bool, []string, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return false, nil, fmt.Errorf("failed to load config: %w", err)
	}

	creds := r.gatherCredentials(p, cfg)
	missing := []string{}

	for _, env := range p.RequiredEnvVars {
		if !env.Required {
			continue
		}
		if creds[env.Name] == "" {
			missing = append(missing, env.Name)
		}
	}

	// Special check for Bedrock - needs either bearer token OR AWS keys
	if p.API == "bedrock" && creds["AWS_BEDROCK_API_KEY"] == "" {
		if creds["AWS_ACCESS_KEY_ID"] == "" || creds["AWS_SECRET_ACCESS_KEY"] == "" {
			missing = append(missing, "AWS_BEDROCK_API_KEY or (AWS_ACCESS_KEY_ID + AWS_SECRET_ACCESS_KEY)")
		}
	}

	return len(missing) == 0, missing, nil
}

// gatherCredentials collects credentials from env vars and config
func (r *ProviderRegistry) gatherCredentials(p *Provider, cfg *config.Config) map[string]string {
	creds := make(map[string]string)

	for _, env := range p.RequiredEnvVars {
		// Check environment variable first
		if val := os.Getenv(env.Name); val != "" {
			creds[env.Name] = val
			continue
		}

		// Fall back to config file
		switch env.ConfigKey {
		case "gemini_api_key":
			creds[env.Name] = cfg.GeminiAPIKey
		case "vertex_project":
			creds[env.Name] = cfg.VertexProject
		case "vertex_location":
			creds[env.Name] = cfg.VertexLocation
		case "vertex_api_key":
			creds[env.Name] = cfg.VertexAPIKey
		case "aws_region":
			creds[env.Name] = cfg.AWSRegion
		case "aws_bedrock_api_key":
			creds[env.Name] = cfg.AWSBedrockAPIKey
		case "aws_access_key_id":
			creds[env.Name] = cfg.AWSAccessKeyID
		case "aws_secret_access_key":
			creds[env.Name] = cfg.AWSSecretAccessKey
		case "grok_api_key":
			creds[env.Name] = cfg.GrokAPIKey
		}
	}

	return creds
}

// CreateClient creates a client for the provider
func (r *ProviderRegistry) CreateClient(providerID string) (ImageGenerator, error) {
	p, err := r.Get(providerID)
	if err != nil {
		return nil, err
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	creds := r.gatherCredentials(p, cfg)

	// Check auth before creating client
	hasAuth, missing, _ := r.CheckAuth(p)
	if !hasAuth {
		return nil, fmt.Errorf("missing credentials for %s: %v", p.Name, missing)
	}

	return p.CreateClient(creds)
}

// AuthStatus represents the authentication status of a provider
type AuthStatus struct {
	Provider   *Provider
	Configured bool
	Missing    []string
	Source     string // "env" or "config" or "both"
}

// GetAuthStatus returns detailed auth status for all providers
func (r *ProviderRegistry) GetAuthStatus() []AuthStatus {
	cfg, _ := config.LoadConfig()
	statuses := []AuthStatus{}

	for _, p := range r.providers {
		creds := r.gatherCredentials(p, cfg)
		hasAuth, missing, _ := r.CheckAuth(p)

		// Determine source
		source := "none"
		hasEnv := false
		hasConfig := false

		for _, env := range p.RequiredEnvVars {
			if env.Required {
				if os.Getenv(env.Name) != "" {
					hasEnv = true
				}
				if creds[env.Name] != "" && os.Getenv(env.Name) == "" {
					hasConfig = true
				}
			}
		}

		if hasEnv && hasConfig {
			source = "both"
		} else if hasEnv {
			source = "env"
		} else if hasConfig {
			source = "config"
		}

		statuses = append(statuses, AuthStatus{
			Provider:   p,
			Configured: hasAuth,
			Missing:    missing,
			Source:     source,
		})
	}

	return statuses
}

// providerAliases maps user-friendly names to canonical provider IDs.
// Kept at package scope so ResolveProvider doesn't re-allocate on every call.
var providerAliases = map[string]string{
	// Default "gemini" maps to Gemini 3 Pro
	"gemini":                     "gemini/pro-3",
	"gemini-3":                   "gemini/pro-3",
	"gemini-3-pro":               "gemini/pro-3",
	"gemini3":                    "gemini/pro-3",
	"pro-3":                      "gemini/pro-3",
	"gemini-3-pro-image-preview": "gemini/pro-3",
	// Gemini 3.1 Flash
	"gemini-3.1-flash":               "gemini/flash-3.1",
	"gemini-3.1":                     "gemini/flash-3.1",
	"3.1-flash":                      "gemini/flash-3.1",
	"gemini-3.1-flash-image-preview": "gemini/flash-3.1",
	// Gemini 2.5 Flash
	"gemini-flash":           "gemini/flash-2.5",
	"flash":                  "gemini/flash-2.5",
	"gemini-2.5":             "gemini/flash-2.5",
	"gemini-2.5-flash":       "gemini/flash-2.5",
	"gemini-2.5-flash-image": "gemini/flash-2.5",
	// Vertex/Imagen
	"imagen":                        "vertex/imagen-4",
	"imagen-4":                      "vertex/imagen-4",
	"imagen-4.0-generate-001":       "vertex/imagen-4",
	"imagen-4-fast":                 "vertex/imagen-4-fast",
	"imagen-fast":                   "vertex/imagen-4-fast",
	"imagen-4.0-fast-generate-001":  "vertex/imagen-4-fast",
	"imagen-4-ultra":                "vertex/imagen-4-ultra",
	"imagen-ultra":                  "vertex/imagen-4-ultra",
	"imagen-4.0-ultra-generate-001": "vertex/imagen-4-ultra",
	// Bedrock
	"nova":                    "bedrock/nova-canvas",
	"nova-canvas":             "bedrock/nova-canvas",
	"amazon.nova-canvas-v1:0": "bedrock/nova-canvas",
	// Grok
	"grok":                   "grok/grok-imagine",
	"grok-imagine":           "grok/grok-imagine",
	"grok-imagine-image":     "grok/grok-imagine",
	"grok-imagine-pro":       "grok/grok-imagine-pro",
	"grok-imagine-image-pro": "grok/grok-imagine-pro",
	"grok-2":                 "grok/grok-2-image",
	"grok-2-image":           "grok/grok-2-image",
	"grok-2-image-1212":      "grok/grok-2-image",
	"xai":                    "grok/grok-imagine",
	"aurora":                 "grok/grok-imagine",
}

// ResolveProvider finds a provider by various identifiers
func (r *ProviderRegistry) ResolveProvider(input string) (*Provider, error) {
	if p, err := r.Get(input); err == nil {
		return p, nil
	}

	if providerID, ok := providerAliases[strings.ToLower(input)]; ok {
		return r.Get(providerID)
	}

	return nil, fmt.Errorf("no provider found for: %s", input)
}

// ResolveModelName resolves a model alias to its official name (for backward compatibility)
func ResolveModelName(name string) string {
	registry := GetProviderRegistry()
	provider, err := registry.ResolveProvider(name)
	if err != nil {
		// No match found, return original
		return name
	}
	// Return the model ID (API identifier)
	return provider.ModelID
}

// DetectAPIFromModel determines which API to use based on model name (for backward compatibility)
func DetectAPIFromModel(modelName string) (string, error) {
	if modelName == "" {
		return "gemini", nil // Default to Gemini
	}

	registry := GetProviderRegistry()
	provider, err := registry.ResolveProvider(modelName)
	if err != nil {
		return "", fmt.Errorf("unknown model: %s", modelName)
	}

	return provider.API, nil
}

// ValidateModelForAPI validates that a model is compatible with an API (for backward compatibility)
func ValidateModelForAPI(modelName, api string) error {
	if modelName == "" {
		return nil // No validation needed for default
	}

	registry := GetProviderRegistry()
	provider, err := registry.ResolveProvider(modelName)
	if err != nil {
		return fmt.Errorf("unknown model: %s", modelName)
	}

	if provider.API != api {
		return fmt.Errorf("model %s is not available on %s API (only on %s)", modelName, api, provider.API)
	}

	return nil
}

// ============================================================================
// Shared Pricing Helpers
// ============================================================================
// These functions provide consistent pricing display across CLI and MCP server.

// CalculatedPricing contains calculated pricing information for a specific request
// This differs from PricingInfo (which is static provider metadata) by calculating
// the actual cost based on request parameters like image size and dimensions.
type CalculatedPricing struct {
	Display     string  // Human-readable pricing string (e.g., "$0.1340/image (1K/2K resolution)")
	Cost        float64 // Actual cost for this request
	IsFree      bool    // Whether this is free tier
	IsExpensive bool    // Whether cost > $0.05 (triggers warning)
}

// GetProviderPricing calculates pricing info for a provider based on request parameters
// imageSize: For Gemini 3 Pro native resolution (1K, 2K, 4K)
// dimensions: For size-based pricing like Nova Canvas (e.g., "1024x1024", "2048x2048")
func GetProviderPricing(provider *Provider, imageSize, dimensions string) CalculatedPricing {
	info := CalculatedPricing{}

	if provider == nil {
		info.Display = "Unknown"
		return info
	}

	if provider.Pricing.FreeTier {
		info.IsFree = true
		info.Display = fmt.Sprintf("FREE (%s)", provider.Pricing.FreeTierLimit)
		return info
	}

	if provider.Pricing.CostPerImage == nil {
		info.Display = "Variable"
		return info
	}

	info.Cost = *provider.Pricing.CostPerImage

	// Handle variable pricing models
	if strings.Contains(provider.ModelID, "gemini-3-pro") {
		info.Cost = getGemini3ProCost(imageSize)
		info.Display = fmt.Sprintf("$%.4f/image (%s)", info.Cost, ImageSizeLabel(imageSize))
	} else if strings.Contains(provider.ModelID, "nova-canvas") {
		info.Cost = getNovaCanvasCost(dimensions)
		w, h := parseDimensionsForCost(dimensions)
		if w > 1024 || h > 1024 {
			info.Display = fmt.Sprintf("$%.4f/image (>1024px)", info.Cost)
		} else {
			info.Display = fmt.Sprintf("$%.4f/image (≤1024px)", info.Cost)
		}
	} else {
		info.Display = fmt.Sprintf("$%.4f/image", info.Cost)
	}

	info.IsExpensive = info.Cost > 0.05
	return info
}

// getGemini3ProCost returns the cost for Gemini 3 Pro based on image size
// Pricing: $0.134 for 1K/2K, $0.24 for 4K
func getGemini3ProCost(imageSize string) float64 {
	if strings.ToUpper(imageSize) == "4K" {
		return 0.24
	}
	return 0.134 // Default for 1K/2K or unspecified
}

// getNovaCanvasCost returns the cost for Nova Canvas based on image dimensions
// Pricing: $0.04 for ≤1024x1024, $0.08 for >1024x1024
func getNovaCanvasCost(size string) float64 {
	width, height := parseDimensionsForCost(size)
	if width > 1024 || height > 1024 {
		return 0.08
	}
	return 0.04
}

// parseDimensionsForCost parses "WIDTHxHEIGHT" string for cost calculation
func parseDimensionsForCost(size string) (int, int) {
	if size == "" {
		return 1024, 1024
	}
	parts := strings.Split(size, "x")
	if len(parts) != 2 {
		return 1024, 1024
	}
	var width, height int
	fmt.Sscanf(parts[0], "%d", &width)
	fmt.Sscanf(parts[1], "%d", &height)
	if width <= 0 {
		width = 1024
	}
	if height <= 0 {
		height = 1024
	}
	return width, height
}

// ImageSizeLabel returns a human-readable label for the image size
func ImageSizeLabel(imageSize string) string {
	switch strings.ToUpper(imageSize) {
	case "4K":
		return "4K resolution"
	case "2K":
		return "2K resolution"
	case "1K":
		return "1K resolution"
	default:
		return "1K/2K resolution"
	}
}
