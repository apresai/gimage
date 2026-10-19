package generate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetProviderRegistry(t *testing.T) {
	reg := GetProviderRegistry()
	require.NotNil(t, reg, "registry should not be nil")

	// Singleton: same instance on multiple calls
	reg2 := GetProviderRegistry()
	assert.Same(t, reg, reg2, "should return same singleton instance")
}

func TestRegistryGet(t *testing.T) {
	reg := GetProviderRegistry()

	tests := []struct {
		name        string
		id          string
		wantModelID string
		wantErr     bool
	}{
		{"gemini nano banana 2.1", "gemini/nano-banana-2.1", "gemini-nano-banana-2.1", false},
		{"gemini flash 3.1", "gemini/flash-3.1", "gemini-3.1-flash-image", false},
		{"gemini flash 3.1 lite", "gemini/flash-3.1-lite", "gemini-3.1-flash-lite-image", false},
		{"gemini pro 3", "gemini/pro-3", "gemini-3-pro-image", false},
		{"vertex nano banana 2.1", "vertex/nano-banana-2.1", "gemini-nano-banana-2.1", false},
		{"vertex nano banana 2.1 fast", "vertex/nano-banana-2.1-fast", "gemini-nano-banana-2.1", false},
		{"vertex nano banana 2.1 ultra", "vertex/nano-banana-2.1-ultra", "gemini-nano-banana-2.1", false},
		{"vertex flash 3.1", "vertex/flash-3.1", "gemini-3.1-flash-image", false},
		{"vertex flash 3.1 fast", "vertex/flash-3.1-fast", "gemini-3.1-flash-image", false},
		{"vertex flash 3.1 ultra", "vertex/flash-3.1-ultra", "gemini-3.1-flash-image", false},
		{"vertex flash 3.1 lite", "vertex/flash-3.1-lite", "gemini-3.1-flash-lite-image", false},
		{"grok imagine", "grok/grok-imagine", "grok-imagine-image", false},
		{"grok imagine 2.0", "grok/grok-imagine-2.0", "grok-imagine-image-2.0", false},
		{"retired gemini flash 2.5 not in Get", "gemini/flash-2.5", "", true},
		{"folded grok imagine quality not in Get", "grok/grok-imagine-quality", "", true},
		{"nonexistent", "nonexistent", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := reg.Get(tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, p)
			} else {
				require.NoError(t, err)
				require.NotNil(t, p)
				assert.Equal(t, tt.wantModelID, p.ModelID)
			}
		})
	}
}

func TestRegistryList(t *testing.T) {
	reg := GetProviderRegistry()
	providers := reg.List()

	// Should have at least 13 providers (4 gemini + 7 vertex + 2 grok)
	assert.GreaterOrEqual(t, len(providers), 13, "should have at least 13 providers")

	for _, p := range providers {
		assert.NotEmpty(t, p.ID, "provider ID should not be empty")
		assert.NotEmpty(t, p.Name, "provider Name should not be empty")
		assert.NotEmpty(t, p.API, "provider API should not be empty")
		assert.NotEmpty(t, p.ModelID, "provider ModelID should not be empty")
	}
}

func TestRegistryListByAPI(t *testing.T) {
	reg := GetProviderRegistry()

	tests := []struct {
		name     string
		api      string
		minCount int
	}{
		{"gemini providers", "gemini", 4},
		{"vertex providers", "vertex", 7},
		{"grok providers", "grok", 2},
		{"invalid api empty", "invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers := reg.ListByAPI(tt.api)
			assert.GreaterOrEqual(t, len(providers), tt.minCount)

			for _, p := range providers {
				assert.Equal(t, tt.api, p.API, "all providers should be for the requested API")
			}
		})
	}
}

func TestRegistryListAPIs(t *testing.T) {
	reg := GetProviderRegistry()
	apis := reg.ListAPIs()

	assert.Len(t, apis, 3, "should have 3 APIs")

	// Verify sorted by Order field
	expectedOrder := []string{"gemini", "vertex", "grok"}
	for i, api := range apis {
		assert.Equal(t, expectedOrder[i], api.ID, "API at position %d", i)
	}

	// Verify all have display names
	for _, api := range apis {
		assert.NotEmpty(t, api.DisplayName)
		assert.NotEmpty(t, api.Description)
		assert.Greater(t, api.Order, 0)
	}
}

func TestRegistryGetAPIInfo(t *testing.T) {
	reg := GetProviderRegistry()

	tests := []struct {
		name            string
		api             string
		wantFound       bool
		wantDisplayName string
	}{
		{"gemini info", "gemini", true, "Gemini"},
		{"vertex info", "vertex", true, "Vertex"},
		{"grok info", "grok", true, "Grok"},
		{"nonexistent", "nonexistent", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, found := reg.GetAPIInfo(tt.api)
			assert.Equal(t, tt.wantFound, found)
			if tt.wantFound {
				assert.Contains(t, info.DisplayName, tt.wantDisplayName)
			}
		})
	}
}

func TestResolveProvider(t *testing.T) {
	reg := GetProviderRegistry()

	tests := []struct {
		name    string
		input   string
		wantID  string
		wantErr bool
	}{
		{"gemini default to nano-banana-2.1", "gemini", "gemini/nano-banana-2.1", false},
		{"nano-banana alias", "nano-banana", "gemini/nano-banana-2.1", false},
		{"nano-banana-2 alias", "nano-banana-2", "gemini/nano-banana-2.1", false},
		{"nano-banana-2.1 alias", "nano-banana-2.1", "gemini/nano-banana-2.1", false},
		{"gemini-nano-banana-2 alias", "gemini-nano-banana-2", "gemini/nano-banana-2.1", false},
		{"gemini-nano-banana-2.1 exact", "gemini-nano-banana-2.1", "gemini/nano-banana-2.1", false},
		{"flash alias", "flash", "gemini/flash-3.1-lite", false},
		{"gemini-flash alias", "gemini-flash", "gemini/flash-3.1-lite", false},
		{"gemini-lite alias", "gemini-lite", "gemini/flash-3.1-lite", false},
		{"retired gemini-2.5-flash-image errors", "gemini-2.5-flash-image", "", true},
		{"gemini-3 alias", "gemini-3", "gemini/pro-3", false},
		{"vertex-nano-banana alias", "vertex-nano-banana", "vertex/nano-banana-2.1", false},
		{"vertex-nano-banana-fast alias", "vertex-nano-banana-fast", "vertex/nano-banana-2.1-fast", false},
		{"vertex-nano-banana-ultra alias", "vertex-nano-banana-ultra", "vertex/nano-banana-2.1-ultra", false},
		{"vertex-flash alias", "vertex-flash", "vertex/flash-3.1", false},
		{"vertex-flash-fast alias", "vertex-flash-fast", "vertex/flash-3.1-fast", false},
		{"vertex-flash-ultra alias", "vertex-flash-ultra", "vertex/flash-3.1-ultra", false},
		{"vertex-flash-lite alias", "vertex-flash-lite", "vertex/flash-3.1-lite", false},
		{"retired imagen errors", "imagen", "", true},
		{"retired imagen-4 errors", "imagen-4", "", true},
		{"retired imagen-4-fast errors", "imagen-4-fast", "", true},
		{"retired imagen-fast errors", "imagen-fast", "", true},
		{"retired imagen-4-ultra errors", "imagen-4-ultra", "", true},
		{"retired nova errors", "nova", "", true},
		{"retired nova-canvas errors", "nova-canvas", "", true},
		{"grok alias", "grok", "grok/grok-imagine-2.0", false},
		{"grok-imagine alias", "grok-imagine", "grok/grok-imagine-2.0", false},
		{"grok-imagine-quality alias", "grok-imagine-quality", "grok/grok-imagine-2.0", false},
		{"grok-quality alias", "grok-quality", "grok/grok-imagine-2.0", false},
		{"grok-fast alias", "grok-fast", "grok/grok-imagine", false},
		{"grok-imagine-image exact", "grok-imagine-image", "grok/grok-imagine", false},
		{"grok-imagine-image-quality folds to 2.0", "grok-imagine-image-quality", "grok/grok-imagine-2.0", false},
		{"grok-imagine-pro folds to 2.0", "grok-imagine-pro", "grok/grok-imagine-2.0", false},
		{"grok-imagine-image-pro folds to 2.0", "grok-imagine-image-pro", "grok/grok-imagine-2.0", false},
		{"grok/grok-imagine-quality folds to 2.0", "grok/grok-imagine-quality", "grok/grok-imagine-2.0", false},
		{"xai alias", "xai", "grok/grok-imagine-2.0", false},
		{"aurora alias", "aurora", "grok/grok-imagine-2.0", false},
		{"exact provider ID", "gemini/nano-banana-2.1", "gemini/nano-banana-2.1", false},
		{"retired provider ID gemini/flash-2.5", "gemini/flash-2.5", "", true},
		{"nonexistent", "nonexistent", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := reg.ResolveProvider(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, p)
				assert.Equal(t, tt.wantID, p.ID)
			}
		})
	}
}

func TestRetiredAliasesError(t *testing.T) {
	reg := GetProviderRegistry()

	retired := []string{
		"imagen",
		"imagen-4",
		"imagen-4-ultra",
		"gemini-3-pro-image-preview",
		"gemini-3.1-flash-image-preview",
		"gemini-2.5",
		"gemini-2.5-flash",
		"gemini-2.5-flash-image",
		"gemini/flash-2.5",
		"nova",
		"nova-canvas",
		"amazon.nova-canvas-v1:0",
		"bedrock/nova-canvas",
	}

	for _, name := range retired {
		t.Run(name, func(t *testing.T) {
			p, err := reg.ResolveProvider(name)
			require.Error(t, err, "retired alias %q must not resolve", name)
			assert.Nil(t, p)
			assert.Contains(t, err.Error(), "retired", "error for %q should mention it was retired", name)
		})
	}
}

func TestNonImageModelsError(t *testing.T) {
	reg := GetProviderRegistry()

	names := []string{
		"omni",
		"gemini-omni",
		"gemini-omni-flash",
		"gemini-omni-flash-preview",
		"gemini-omni-1.1-flash",
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			p, err := reg.ResolveProvider(name)
			require.Error(t, err, "video model %q must not resolve", name)
			assert.Nil(t, p)
			assert.Contains(t, err.Error(), "not an image model", "error for %q should say it is not an image model", name)
			assert.Contains(t, err.Error(), "video", "error for %q should mention video", name)
		})
	}
}

func TestResolveModelName(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantID string
	}{
		{"flash alias resolves", "flash", "gemini-3.1-flash-lite-image"},
		{"nano-banana-2.1 alias resolves", "nano-banana-2.1", "gemini-nano-banana-2.1"},
		{"vertex-flash alias resolves", "vertex-flash", "gemini-3.1-flash-image"},
		{"retired imagen returns original", "imagen", "imagen"},
		{"retired nova returns original", "nova", "nova"},
		{"retired gemini-2.5-flash-image returns original", "gemini-2.5-flash-image", "gemini-2.5-flash-image"},
		{"grok alias resolves", "grok", "grok-imagine-image-2.0"},
		{"grok-imagine-pro resolves to 2.0", "grok-imagine-pro", "grok-imagine-image-2.0"},
		{"grok-imagine-image-pro resolves to 2.0", "grok-imagine-image-pro", "grok-imagine-image-2.0"},
		{"unknown returns original", "unknown-model", "unknown-model"},
		{"exact model ID returns itself", "gemini-nano-banana-2.1", "gemini-nano-banana-2.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveModelName(tt.input)
			assert.Equal(t, tt.wantID, got)
		})
	}
}

func TestDetectAPIFromModel(t *testing.T) {
	tests := []struct {
		name    string
		model   string
		wantAPI string
		wantErr bool
	}{
		{"empty defaults to gemini", "", "gemini", false},
		{"gemini nano banana model", "gemini-nano-banana-2.1", "gemini", false},
		{"retired gemini 2.5 flash model errors", "gemini-2.5-flash-image", "", true},
		{"gemini alias", "gemini", "gemini", false},
		{"vertex-flash alias", "vertex-flash", "vertex", false},
		{"retired imagen-4 errors", "imagen-4", "", true},
		{"retired nova-canvas errors", "nova-canvas", "", true},
		{"omni video model errors", "gemini-omni", "", true},
		{"grok alias", "grok", "grok", false},
		{"unknown model", "totally-unknown-model-xyz", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api, err := DetectAPIFromModel(tt.model)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantAPI, api)
			}
		})
	}
}

func TestMigratedVertexProviderCapabilities(t *testing.T) {
	reg := GetProviderRegistry()
	for _, providerID := range []string{"vertex/flash-3.1", "vertex/flash-3.1-fast", "vertex/flash-3.1-ultra"} {
		t.Run(providerID, func(t *testing.T) {
			p, err := reg.Get(providerID)
			require.NoError(t, err)
			assert.Equal(t, "gemini-3.1-flash-image", p.ModelID)
			assert.False(t, p.Capabilities.SupportsSeed, "Gemini 3.1 Flash via Vertex does not support seed")
			assert.True(t, p.Capabilities.SupportsImageSize)
			assert.True(t, p.Capabilities.SupportsAspectRatio)
			assert.True(t, p.Capabilities.SupportsThinking)
			assert.True(t, p.Capabilities.SupportsGrounding)
			assert.True(t, p.Capabilities.SupportsInputImages)
			assert.True(t, p.Capabilities.SupportsOutputFormat, "Vertex full-mode honors output format")
			assert.Contains(t, p.Name, "Gemini 3.1 Flash via Vertex")
			assert.Contains(t, p.Description, "Gemini 3.1 Flash via Vertex")
			assert.Contains(t, p.Description, "generateContent")
		})
	}
}

func TestGrokProviderCapabilities(t *testing.T) {
	reg := GetProviderRegistry()
	for _, providerID := range []string{"grok/grok-imagine", "grok/grok-imagine-2.0"} {
		t.Run(providerID, func(t *testing.T) {
			p, err := reg.Get(providerID)
			require.NoError(t, err)
			// Grok DOES honor image size (1K/2K), aspect ratio, and input-image edits.
			assert.True(t, p.Capabilities.SupportsImageSize, "Grok imagine honors resolution 1K/2K")
			assert.True(t, p.Capabilities.SupportsAspectRatio, "Grok imagine honors aspect_ratio")
			assert.True(t, p.Capabilities.SupportsInputImages, "Grok imagine supports /images/edits via --input-image")
			// Grok ignores styles, seed, and output format.
			assert.False(t, p.Capabilities.SupportsStyles)
			assert.False(t, p.Capabilities.SupportsSeed)
			assert.False(t, p.Capabilities.SupportsOutputFormat)
			// Grok returns N images exactly.
			assert.True(t, p.Capabilities.SupportsMultipleImages)
			if providerID == "grok/grok-imagine-2.0" {
				assert.True(t, p.Capabilities.SupportsQuality)
			} else {
				assert.False(t, p.Capabilities.SupportsQuality)
			}
		})
	}
}

func TestGetProviderPricing(t *testing.T) {
	reg := GetProviderRegistry()

	nanoBananaProvider, _ := reg.Get("gemini/nano-banana-2.1")
	flashLiteProvider, _ := reg.Get("gemini/flash-3.1-lite")
	flash31Provider, _ := reg.Get("gemini/flash-3.1")
	pro3Provider, _ := reg.Get("gemini/pro-3")
	grokProvider, _ := reg.Get("grok/grok-imagine")
	grok20Provider, _ := reg.Get("grok/grok-imagine-2.0")

	tests := []struct {
		name        string
		provider    *Provider
		imageSize   string
		dimensions  string
		style       string
		wantCost    float64
		wantDisplay string
	}{
		{
			name:     "gemini nano banana 2.1 default pricing",
			provider: nanoBananaProvider,
			wantCost: 0.0336,
		},
		{
			name:      "gemini nano banana 2.1 2K pricing",
			provider:  nanoBananaProvider,
			imageSize: "2K",
			wantCost:  0.0504,
		},
		{
			name:      "gemini nano banana 2.1 4K pricing",
			provider:  nanoBananaProvider,
			imageSize: "4K",
			wantCost:  0.1134,
		},
		{
			name:     "gemini flash lite pricing",
			provider: flashLiteProvider,
			wantCost: 0.034,
		},
		{
			name:      "gemini 3.1 flash 0.5K pricing",
			provider:  flash31Provider,
			imageSize: "0.5K",
			wantCost:  0.045,
		},
		{
			name:      "gemini 3.1 flash 1K pricing",
			provider:  flash31Provider,
			imageSize: "1K",
			wantCost:  0.067,
		},
		{
			name:      "gemini 3.1 flash 2K pricing",
			provider:  flash31Provider,
			imageSize: "2K",
			wantCost:  0.101,
		},
		{
			name:      "gemini 3.1 flash 4K pricing",
			provider:  flash31Provider,
			imageSize: "4K",
			wantCost:  0.151,
		},
		{
			name:     "gemini 3.1 flash default pricing",
			provider: flash31Provider,
			wantCost: 0.067, // defaults to 1K tier
		},
		{
			name:      "gemini 3 pro 4K pricing",
			provider:  pro3Provider,
			imageSize: "4K",
			wantCost:  0.24,
		},
		{
			name:      "gemini 3 pro 1K pricing",
			provider:  pro3Provider,
			imageSize: "1K",
			wantCost:  0.134,
		},
		{
			name:      "gemini 3 pro default pricing",
			provider:  pro3Provider,
			imageSize: "",
			wantCost:  0.134,
		},
		{
			name:        "nil provider",
			provider:    nil,
			wantDisplay: "Unknown",
		},
		{
			name:     "grok standard pricing",
			provider: grokProvider,
			wantCost: 0.02,
		},
		{
			name:      "grok 2.0 1K pricing",
			provider:  grok20Provider,
			imageSize: "1K",
			wantCost:  0.04,
		},
		{
			name:      "grok 2.0 2K pricing",
			provider:  grok20Provider,
			imageSize: "2K",
			wantCost:  0.06,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricing := GetProviderPricing(tt.provider, tt.imageSize, tt.dimensions, tt.style)

			if tt.wantDisplay != "" {
				assert.Equal(t, tt.wantDisplay, pricing.Display)
			}

			if tt.provider != nil {
				assert.InDelta(t, tt.wantCost, pricing.Cost, 0.001, "cost mismatch")
				assert.NotEmpty(t, pricing.Display)
			}
		})
	}
}

func TestGetProviderPricingExpensiveFlag(t *testing.T) {
	reg := GetProviderRegistry()

	pro3Provider, _ := reg.Get("gemini/pro-3")
	nanoBananaProvider, _ := reg.Get("gemini/nano-banana-2.1")

	// 4K Gemini 3 Pro should be expensive ($0.24 > $0.05)
	expensive := GetProviderPricing(pro3Provider, "4K", "", "")
	assert.True(t, expensive.IsExpensive, "4K Gemini 3 Pro should be marked expensive")

	// Gemini Nano Banana 2.1 at 1K should not be expensive ($0.0336 < $0.05)
	cheap := GetProviderPricing(nanoBananaProvider, "", "", "")
	assert.False(t, cheap.IsExpensive, "Gemini Nano Banana 2.1 at 1K should not be marked expensive")
}

func TestImageSizeLabel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"4K", "4K", "4K resolution"},
		{"2K", "2K", "2K resolution"},
		{"1K", "1K", "1K resolution"},
		{"empty default", "", "1K/2K resolution"},
		{"lowercase 4k", "4k", "4K resolution"},
		{"random string", "HD", "1K/2K resolution"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ImageSizeLabel(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProviderHasCreateClient(t *testing.T) {
	reg := GetProviderRegistry()
	providers := reg.List()

	for _, p := range providers {
		assert.NotNil(t, p.CreateClient, "provider %s should have CreateClient function", p.ID)
	}
}

func TestProviderRequiredEnvVars(t *testing.T) {
	reg := GetProviderRegistry()

	tests := []struct {
		name        string
		providerID  string
		wantEnvVars []string
	}{
		{"gemini needs api key", "gemini/nano-banana-2.1", []string{"GEMINI_API_KEY"}},
		{"vertex needs project and location", "vertex/flash-3.1", []string{"VERTEX_PROJECT", "VERTEX_LOCATION"}},
		{"grok needs api key", "grok/grok-imagine", []string{"GROK_API_KEY"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := reg.Get(tt.providerID)
			require.NoError(t, err)

			envNames := make([]string, 0)
			for _, env := range p.RequiredEnvVars {
				if env.Required {
					envNames = append(envNames, env.Name)
				}
			}

			for _, want := range tt.wantEnvVars {
				assert.Contains(t, envNames, want, "should require %s", want)
			}
		})
	}
}

func TestModelPricingRegistry(t *testing.T) {
	// Every entry must populate the required audit fields.
	for modelID, entry := range ModelPricing {
		t.Run("metadata_"+modelID, func(t *testing.T) {
			assert.Equal(t, modelID, entry.ModelID, "map key must match entry.ModelID")
			assert.NotEmpty(t, entry.Summary, "Summary required for audit display")
			assert.NotZero(t, entry.Representative, "Representative price required")
			assert.NotNil(t, entry.Calculate, "Calculate function required")
			assert.NotEmpty(t, entry.Source, "Source URL required for audit")
			assert.NotEmpty(t, entry.Verified, "Verified date required")
		})
	}

	// Gemini-specific audit: assert the most recent re-verification date and the
	// Note pointing to the separate batch endpoint. This creates a maintenance
	// signal — when pricing is re-verified next, bump geminiVerifiedDate here
	// in the same commit as the pricing.go Verified field. Without this guard,
	// the Notes silently rot.
	const geminiVerifiedDate = "2026-10-19"
	geminiEntries := []string{
		"gemini-nano-banana-2.1",
		"gemini-2.5-flash-image",
		"gemini-3-pro-image",
		"gemini-3.1-flash-image",
		"gemini-3.1-flash-lite-image",
	}
	for _, modelID := range geminiEntries {
		t.Run("gemini_audit_"+modelID, func(t *testing.T) {
			entry, ok := LookupPricing(modelID)
			require.True(t, ok, "expected Gemini entry %s", modelID)
			assert.Equal(t, geminiVerifiedDate, entry.Verified,
				"Gemini entries were re-verified %s — bump this constant when re-verified again", geminiVerifiedDate)
			assert.NotEmpty(t, entry.Note,
				"Gemini entries should document the separate batch endpoint in Note")
			assert.Contains(t, entry.Note, "batchGenerateContent",
				"Note should mention the batch endpoint as future work")
		})
	}

	// Every provider whose ModelID is non-empty must have a ModelPricing entry.
	// This guard catches drift when a new provider is added without updating
	// pricing.go.
	reg := GetProviderRegistry()
	for _, p := range reg.List() {
		t.Run("coverage_"+p.ID, func(t *testing.T) {
			entry, ok := LookupPricing(p.ModelID)
			require.True(t, ok, "provider %s (model %s) has no ModelPricing entry", p.ID, p.ModelID)

			if p.Pricing.CostPerImage != nil {
				assert.InDelta(t, entry.Representative, *p.Pricing.CostPerImage, 0.0001,
					"provider.Pricing.CostPerImage must match ModelPricing[%s].Representative", p.ModelID)
			}
		})
	}

	// Known exact-price checkpoints. These are the numbers we verified against
	// authoritative sources on the Verified date. If any change, update the
	// corresponding ModelPricing entry AND document the source.
	checkpoints := []struct {
		modelID   string
		imageSize string
		dims      string
		style     string
		want      float64
	}{
		{modelID: "gemini-nano-banana-2.1", imageSize: "0.5K", want: 0.0168},
		{modelID: "gemini-nano-banana-2.1", imageSize: "1K", want: 0.0336},
		{modelID: "gemini-nano-banana-2.1", imageSize: "2K", want: 0.0504},
		{modelID: "gemini-nano-banana-2.1", imageSize: "4K", want: 0.1134},
		{modelID: "gemini-nano-banana-2.1", want: 0.0336},
		{modelID: "gemini-2.5-flash-image", want: 0.039},
		{modelID: "gemini-3-pro-image", imageSize: "1K", want: 0.134},
		{modelID: "gemini-3-pro-image", imageSize: "2K", want: 0.134},
		{modelID: "gemini-3-pro-image", imageSize: "4K", want: 0.24},
		{modelID: "gemini-3-pro-image", want: 0.134},
		{modelID: "gemini-3.1-flash-image", imageSize: "0.5K", want: 0.045},
		{modelID: "gemini-3.1-flash-image", imageSize: "1K", want: 0.067},
		{modelID: "gemini-3.1-flash-image", imageSize: "2K", want: 0.101},
		{modelID: "gemini-3.1-flash-image", imageSize: "4K", want: 0.151},
		{modelID: "gemini-3.1-flash-image", want: 0.067},
		{modelID: "gemini-3.1-flash-lite-image", want: 0.034},
		{modelID: "grok-imagine-image", want: 0.02},
		{modelID: "grok-imagine-image-2.0", want: 0.04},
		{modelID: "grok-imagine-image-2.0", imageSize: "1K", want: 0.04},
		{modelID: "grok-imagine-image-2.0", imageSize: "2K", want: 0.06},
		{modelID: "grok-imagine-image-quality", want: 0.05},
		{modelID: "grok-imagine-image-quality", imageSize: "2K", want: 0.07},
		{modelID: "gemini-3-pro-image-preview", imageSize: "4K", want: 0.24},
		{modelID: "gemini-3.1-flash-image-preview", imageSize: "4K", want: 0.151},
	}
	for _, c := range checkpoints {
		name := "checkpoint_" + c.modelID + "_" + c.imageSize + "_" + c.dims + "_" + c.style
		t.Run(name, func(t *testing.T) {
			entry, ok := LookupPricing(c.modelID)
			require.True(t, ok)
			got := entry.Calculate(c.imageSize, c.dims, c.style)
			assert.InDelta(t, c.want, got, 0.0001,
				"%s @ imageSize=%q dims=%q style=%q", c.modelID, c.imageSize, c.dims, c.style)
		})
	}
}

func TestNewProviderRegistryIsIsolated(t *testing.T) {
	// NewProviderRegistry creates a fresh instance (not the singleton)
	reg := NewProviderRegistry()
	require.NotNil(t, reg)

	// Should have the same providers as the global
	global := GetProviderRegistry()
	assert.Equal(t, len(global.List()), len(reg.List()), "new registry should have same number of providers")

	// But be a different pointer
	assert.NotSame(t, global, reg, "new registry should be a different instance")
}
