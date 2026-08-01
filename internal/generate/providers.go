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
	DefaultModel = "gemini-3-pro-image"
)

// ImageGenerator is the common interface for all image generation clients
type ImageGenerator interface {
	GenerateImage(ctx context.Context, prompt string, options models.GenerateOptions) ([]*models.GeneratedImage, error)
	Close() error
}

// Provider represents a specific way to access a model (API + Model + Auth)
type Provider struct {
	// Unique identifier: "provider/model" e.g., "gemini/flash-2.5", "vertex/flash-3.1"
	ID string

	// Display information
	Name        string // e.g., "Gemini 2.5 Flash (via Gemini API)"
	API         string // "gemini", "vertex", "grok"
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
	SupportsSeed           bool
	SupportsImageSize      bool
	SupportsAspectRatio    bool
	SupportsThinking       bool
	SupportsGrounding      bool
	SupportsInputImages    bool
	SupportsOutputFormat   bool // Only the Vertex full-mode (unified SDK) path honors OutputMIMEType
	SupportsMultipleImages bool // count>1: Grok returns N exactly; Gemini/Vertex are best-effort
	MaxPromptLength        int
}

// Helper function for creating float64 pointers
func float64Ptr(f float64) *float64 { return &f }

// APIInfo contains metadata about an API backend
type APIInfo struct {
	ID          string // "gemini", "vertex", "grok"
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
		PricingNote: "Paid: $0.045-0.151 per image (Gemini 3.1 Flash tiered)",
		Order:       2,
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
			SupportsSeed:           true,
			SupportsAspectRatio:    true,
			SupportsInputImages:    true,
			SupportsMultipleImages: true,
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

	// Gemini 3.1 Flash Image via Gemini API
	r.Register(&Provider{
		ID:          "gemini/flash-3.1",
		Name:        "Gemini 3.1 Flash (via Gemini API)",
		API:         "gemini",
		ModelID:     "gemini-3.1-flash-image",
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
		// Tiered by resolution: see ModelPricing["gemini-3.1-flash-image"].
		// CostPerImage holds the 1K-tier price for list displays only —
		// GetProviderPricing uses the full tier schedule from pricing.go.
		Pricing: PricingInfo{
			CostPerImage: float64Ptr(0.067),
			FreeTier:     false,
			Currency:     "USD",
		},
		Capabilities: ModelCapabilities{
			SupportsStyles:         true,
			SupportsSeed:           true,
			SupportsImageSize:      true,
			SupportsAspectRatio:    true,
			SupportsThinking:       true,
			SupportsGrounding:      true,
			SupportsInputImages:    true,
			SupportsMultipleImages: true,
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

	// Gemini 3 Pro Image via Gemini API
	r.Register(&Provider{
		ID:          "gemini/pro-3",
		Name:        "Gemini 3 Pro (via Gemini API)",
		API:         "gemini",
		ModelID:     "gemini-3-pro-image",
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
			SupportsSeed:           true,
			SupportsImageSize:      true,
			SupportsAspectRatio:    true,
			SupportsThinking:       true,
			SupportsGrounding:      true,
			SupportsInputImages:    true,
			SupportsMultipleImages: true,
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

	// Gemini 3.1 Flash via Vertex AI — standard preset (medium thinking)
	r.Register(&Provider{
		ID:          "vertex/flash-3.1",
		Name:        "Gemini 3.1 Flash via Vertex (medium thinking)",
		API:         "vertex",
		ModelID:     "gemini-3.1-flash-image",
		Description: "Gemini 3.1 Flash via Vertex generateContent; tiered image pricing. Default thinking: medium (--thinking overrides).",
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
			CostPerImage: float64Ptr(0.067),
			FreeTier:     false,
			Currency:     "USD",
		},
		Capabilities: ModelCapabilities{
			SupportsStyles:         true,
			SupportsSeed:           false,
			SupportsImageSize:      true,
			SupportsAspectRatio:    true,
			SupportsThinking:       true,
			SupportsGrounding:      true,
			SupportsInputImages:    true,
			SupportsOutputFormat:   true,
			SupportsMultipleImages: true,
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

	// Gemini 3.1 Flash via Vertex AI — fast preset (minimal thinking)
	r.Register(&Provider{
		ID:          "vertex/flash-3.1-fast",
		Name:        "Gemini 3.1 Flash via Vertex (fast, minimal thinking)",
		API:         "vertex",
		ModelID:     "gemini-3.1-flash-image",
		Description: "Gemini 3.1 Flash via Vertex generateContent; tiered image pricing. Default thinking: minimal for lowest latency (--thinking overrides).",
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
			CostPerImage: float64Ptr(0.067),
			FreeTier:     false,
			Currency:     "USD",
		},
		Capabilities: ModelCapabilities{
			SupportsStyles:         true,
			SupportsSeed:           false,
			SupportsImageSize:      true,
			SupportsAspectRatio:    true,
			SupportsThinking:       true,
			SupportsGrounding:      true,
			SupportsInputImages:    true,
			SupportsOutputFormat:   true,
			SupportsMultipleImages: true,
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

	// Gemini 3.1 Flash via Vertex AI — ultra preset (high thinking)
	r.Register(&Provider{
		ID:          "vertex/flash-3.1-ultra",
		Name:        "Gemini 3.1 Flash via Vertex (ultra, high thinking)",
		API:         "vertex",
		ModelID:     "gemini-3.1-flash-image",
		Description: "Gemini 3.1 Flash via Vertex generateContent; tiered image pricing. Default thinking: high for best layout/text (--thinking overrides).",
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
			CostPerImage: float64Ptr(0.067),
			FreeTier:     false,
			Currency:     "USD",
		},
		Capabilities: ModelCapabilities{
			SupportsStyles:         true,
			SupportsSeed:           false,
			SupportsImageSize:      true,
			SupportsAspectRatio:    true,
			SupportsThinking:       true,
			SupportsGrounding:      true,
			SupportsInputImages:    true,
			SupportsOutputFormat:   true,
			SupportsMultipleImages: true,
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

	// Imagen retired entirely. Vertex now serves Gemini 3.1 Flash via the vertex/flash-3.1* presets above.

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
			SupportsSeed:           false,
			SupportsImageSize:      true,
			SupportsAspectRatio:    true,
			SupportsInputImages:    true, // POST /v1/images/edits, max 3
			SupportsMultipleImages: true,
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

	// Grok Imagine Quality via xAI (replaces deprecated -pro retired 2026-05-15)
	r.Register(&Provider{
		ID:          "grok/grok-imagine-quality",
		Name:        "Grok Imagine Quality (via xAI)",
		API:         "grok",
		ModelID:     "grok-imagine-image-quality",
		Description: "xAI quality tier ($0.05/image at 1K, $0.07 at 2K); aliases: grok-quality, grok-imagine-pro",
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
			CostPerImage: float64Ptr(0.05),
			FreeTier:     false,
			Currency:     "USD",
		},
		Capabilities: ModelCapabilities{
			SupportsStyles:         false,
			SupportsSeed:           false,
			SupportsImageSize:      true,
			SupportsAspectRatio:    true,
			SupportsInputImages:    true, // POST /v1/images/edits, max 3
			SupportsMultipleImages: true,
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
	"gemini":             "gemini/pro-3",
	"gemini-3":           "gemini/pro-3",
	"gemini-3-pro":       "gemini/pro-3",
	"gemini3":            "gemini/pro-3",
	"pro-3":              "gemini/pro-3",
	"gemini-3-pro-image": "gemini/pro-3",
	// Gemini 3.1 Flash
	"gemini-3.1-flash":       "gemini/flash-3.1",
	"gemini-3.1-flash-image": "gemini/flash-3.1",
	"gemini-3.1":             "gemini/flash-3.1",
	"3.1-flash":              "gemini/flash-3.1",
	// Gemini 2.5 Flash
	"gemini-flash":           "gemini/flash-2.5",
	"flash":                  "gemini/flash-2.5",
	"gemini-2.5":             "gemini/flash-2.5",
	"gemini-2.5-flash":       "gemini/flash-2.5",
	"gemini-2.5-flash-image": "gemini/flash-2.5",
	// Gemini 3.1 Flash via Vertex AI (thinking presets)
	"vertex-flash":       "vertex/flash-3.1",
	"vertex-flash-fast":  "vertex/flash-3.1-fast",
	"vertex-flash-ultra": "vertex/flash-3.1-ultra",
	// Grok
	"grok":                       "grok/grok-imagine",
	"grok-imagine":               "grok/grok-imagine",
	"grok-imagine-image":         "grok/grok-imagine",
	"grok-quality":               "grok/grok-imagine-quality",
	"grok-imagine-quality":       "grok/grok-imagine-quality",
	"grok-imagine-image-quality": "grok/grok-imagine-quality",
	// xAI still aliases -pro → quality (verified live GET /v1/models 2026-08-01)
	"grok-imagine-pro":       "grok/grok-imagine-quality",
	"grok-imagine-image-pro": "grok/grok-imagine-quality",
	"xai":                    "grok/grok-imagine",
	"aurora":                 "grok/grok-imagine",
}

// retiredAliases are model names we used to accept but no longer run. They must
// NOT resolve (so we never silently deliver a different model than asked) and
// are never advertised. ResolveProvider returns a helpful error naming the real
// replacement. The imagen-* names never ran Imagen — they were Gemini 3.1 Flash
// via Vertex; we now say so plainly instead of advertising a model we don't run.
var retiredAliases = map[string]string{
	"imagen":                         `it ran Gemini 3.1 Flash via Vertex (never Imagen); use "vertex-flash" or "gemini-3.1-flash"`,
	"imagen-4":                       `it ran Gemini 3.1 Flash via Vertex (never Imagen); use "vertex-flash" or "gemini-3.1-flash"`,
	"imagen-fast":                    `it ran Gemini 3.1 Flash via Vertex (never Imagen); use "vertex-flash-fast"`,
	"imagen-4-fast":                  `it ran Gemini 3.1 Flash via Vertex (never Imagen); use "vertex-flash-fast"`,
	"imagen-ultra":                   `it ran Gemini 3.1 Flash via Vertex (never Imagen); use "vertex-flash-ultra"`,
	"imagen-4-ultra":                 `it ran Gemini 3.1 Flash via Vertex (never Imagen); use "vertex-flash-ultra"`,
	"imagen-4.0-generate-001":        `it ran Gemini 3.1 Flash via Vertex (never Imagen); use "vertex-flash" or "gemini-3.1-flash"`,
	"imagen-4.0-fast-generate-001":   `it ran Gemini 3.1 Flash via Vertex (never Imagen); use "vertex-flash-fast"`,
	"imagen-4.0-ultra-generate-001":  `it ran Gemini 3.1 Flash via Vertex (never Imagen); use "vertex-flash-ultra"`,
	"gemini-3-pro-image-preview":     `preview builds were discontinued; use "gemini-3-pro"`,
	"gemini-3.1-flash-image-preview": `preview builds were discontinued; use "gemini-3.1-flash"`,
	// AWS Bedrock Nova Canvas: removed (AWS end-of-life 2026-09-30); no Bedrock image model is offered.
	"nova":                    `Bedrock Nova Canvas was retired (AWS end-of-life 2026-09-30); no Bedrock image model is currently offered. Use "gemini-3-pro" or "grok"`,
	"nova-canvas":             `Bedrock Nova Canvas was retired (AWS end-of-life 2026-09-30); no Bedrock image model is currently offered. Use "gemini-3-pro" or "grok"`,
	"amazon.nova-canvas-v1:0": `Bedrock Nova Canvas was retired (AWS end-of-life 2026-09-30); no Bedrock image model is currently offered. Use "gemini-3-pro" or "grok"`,
	"bedrock/nova-canvas":     `Bedrock Nova Canvas was retired (AWS end-of-life 2026-09-30); no Bedrock image model is currently offered. Use "gemini-3-pro" or "grok"`,
	// Old internal provider IDs (used by --provider / `auth setup <id>` before the rename).
	"vertex/imagen-4":       `it ran Gemini 3.1 Flash via Vertex (never Imagen); use "vertex/flash-3.1" or "vertex-flash"`,
	"vertex/imagen-4-fast":  `it ran Gemini 3.1 Flash via Vertex (never Imagen); use "vertex/flash-3.1-fast" or "vertex-flash-fast"`,
	"vertex/imagen-4-ultra": `it ran Gemini 3.1 Flash via Vertex (never Imagen); use "vertex/flash-3.1-ultra" or "vertex-flash-ultra"`,
}

// ResolveProvider finds a provider by various identifiers
func (r *ProviderRegistry) ResolveProvider(input string) (*Provider, error) {
	if p, err := r.Get(input); err == nil {
		return p, nil
	}

	if providerID, ok := providerAliases[strings.ToLower(input)]; ok {
		return r.Get(providerID)
	}

	if guidance, ok := retiredAliases[strings.ToLower(input)]; ok {
		return nil, fmt.Errorf("model %q was retired: %s", input, guidance)
	}

	return nil, fmt.Errorf("no provider found for: %s", input)
}

// ReconcileOptions strips every generation option the resolved provider does not
// support and returns one friendly, user-facing warning per ignored option. It
// mutates *o in place. It is safe to call with a nil provider or nil options (it
// returns nil and changes nothing), which the legacy runGenerate path relies on
// when provider resolution fails.
//
// NumberOfImages is never stripped: Grok honors N exactly, and the Gemini family
// attempts it via candidateCount (best-effort). For the Gemini family we emit an
// advisory note when count > 1 rather than silently implying N images are
// guaranteed. ResizeMode is client-side post-processing and works everywhere, so
// it is never reconciled.
func ReconcileOptions(p *Provider, o *models.GenerateOptions) []string {
	if p == nil || o == nil {
		return nil
	}
	c := p.Capabilities
	var warnings []string

	if o.Style != "" && !c.SupportsStyles {
		warnings = append(warnings, fmt.Sprintf("--style is ignored by %s; describe the look in the prompt, or use a gemini/* or vertex/* provider", p.ID))
		o.Style = ""
	}
	if o.Seed != 0 && !c.SupportsSeed {
		warnings = append(warnings, fmt.Sprintf("--seed is ignored by %s; reproducible seeds need a gemini/* API provider", p.ID))
		o.Seed = 0
	}
	if o.ImageSize != "" && !c.SupportsImageSize {
		warnings = append(warnings, fmt.Sprintf("--image-size is ignored by %s; use --size here (--image-size needs Gemini 3+ or Grok)", p.ID))
		o.ImageSize = ""
	}
	if o.AspectRatio != "" && !c.SupportsAspectRatio {
		warnings = append(warnings, fmt.Sprintf("--aspect-ratio is ignored by %s", p.ID))
		o.AspectRatio = ""
	}
	if o.ThinkingLevel != "" && !c.SupportsThinking {
		warnings = append(warnings, fmt.Sprintf("--thinking is ignored by %s; it needs a Gemini 3+ provider", p.ID))
		o.ThinkingLevel = ""
	}
	if o.WebSearchGrounding && !c.SupportsGrounding {
		warnings = append(warnings, fmt.Sprintf("--grounding is ignored by %s; it needs a Gemini 3+ provider", p.ID))
		o.WebSearchGrounding = false
	}
	if len(o.InputImages) > 0 && !c.SupportsInputImages {
		warnings = append(warnings, fmt.Sprintf("--input-image is ignored by %s; reference images need a gemini/*, vertex/*, or grok/* provider", p.ID))
		o.InputImages = nil
	}
	if o.OutputFormat != "" && !c.SupportsOutputFormat {
		warnings = append(warnings, fmt.Sprintf("--output-format is ignored by %s; the provider's default format is returned (convert afterward with `gimage convert`)", p.ID))
		o.OutputFormat = ""
	}
	if o.NumberOfImages > 1 && p.API != "grok" {
		warnings = append(warnings, fmt.Sprintf("--count %d is best-effort on %s; the Gemini image API often returns a single image (use a grok/* model for an exact count)", o.NumberOfImages, p.ID))
	}

	return warnings
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

// GetProviderPricing calculates pricing for a provider based on request parameters.
// It consults ModelPricing (the single source of truth in pricing.go) first,
// falling back to provider.Pricing.CostPerImage for models not yet in the registry.
// imageSize: "0.5K"/"1K"/"2K"/"4K" for Gemini tiered models.
// dimensions: "WIDTHxHEIGHT" (retained for dimension-tiered pricing).
// style: user-supplied style string ("photorealistic", "artistic", etc.).
func GetProviderPricing(provider *Provider, imageSize, dimensions, style string) CalculatedPricing {
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

	if entry, ok := LookupPricing(provider.ModelID); ok {
		info.Cost = entry.Calculate(imageSize, dimensions, style)
		info.Display = formatPricingDisplay(entry, info.Cost, imageSize, dimensions, style)
		info.IsExpensive = info.Cost > 0.05
		return info
	}

	if provider.Pricing.CostPerImage == nil {
		info.Display = "Variable"
		return info
	}

	info.Cost = *provider.Pricing.CostPerImage
	info.Display = fmt.Sprintf("$%.4f/image", info.Cost)
	info.IsExpensive = info.Cost > 0.05
	return info
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
