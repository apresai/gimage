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
		{"gemini flash", "gemini/flash-2.5", "gemini-2.5-flash-image", false},
		{"gemini pro 3", "gemini/pro-3", "gemini-3-pro-image-preview", false},
		{"vertex imagen 4", "vertex/imagen-4", "imagen-4.0-generate-001", false},
		{"vertex imagen 4 fast", "vertex/imagen-4-fast", "imagen-4.0-fast-generate-001", false},
		{"vertex imagen 4 ultra", "vertex/imagen-4-ultra", "imagen-4.0-ultra-generate-001", false},
		{"bedrock nova canvas", "bedrock/nova-canvas", "amazon.nova-canvas-v1:0", false},
		{"grok imagine", "grok/grok-imagine", "grok-imagine-image", false},
		{"grok imagine pro", "grok/grok-imagine-pro", "grok-imagine-image-pro", false},
		{"grok 2 image", "grok/grok-2-image", "grok-2-image-1212", false},
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

	// Should have at least 10 providers (2 gemini + 5+ vertex + 1 bedrock + 3 grok)
	assert.GreaterOrEqual(t, len(providers), 10, "should have at least 10 providers")

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
		{"gemini providers", "gemini", 2},
		{"vertex providers", "vertex", 4},
		{"bedrock providers", "bedrock", 1},
		{"grok providers", "grok", 3},
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

	assert.Len(t, apis, 4, "should have 4 APIs")

	// Verify sorted by Order field
	expectedOrder := []string{"gemini", "vertex", "bedrock", "grok"}
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
		{"bedrock info", "bedrock", true, "Bedrock"},
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
		name       string
		input      string
		wantID     string
		wantErr    bool
	}{
		{"gemini default to pro-3", "gemini", "gemini/pro-3", false},
		{"flash alias", "flash", "gemini/flash-2.5", false},
		{"gemini-flash alias", "gemini-flash", "gemini/flash-2.5", false},
		{"gemini-2.5-flash-image exact model", "gemini-2.5-flash-image", "gemini/flash-2.5", false},
		{"gemini-3 alias", "gemini-3", "gemini/pro-3", false},
		{"imagen alias", "imagen", "vertex/imagen-4", false},
		{"imagen-4 alias", "imagen-4", "vertex/imagen-4", false},
		{"imagen-4-fast alias", "imagen-4-fast", "vertex/imagen-4-fast", false},
		{"imagen-fast alias", "imagen-fast", "vertex/imagen-4-fast", false},
		{"imagen-4-ultra alias", "imagen-4-ultra", "vertex/imagen-4-ultra", false},
		{"nova alias", "nova", "bedrock/nova-canvas", false},
		{"nova-canvas alias", "nova-canvas", "bedrock/nova-canvas", false},
		{"grok alias", "grok", "grok/grok-imagine", false},
		{"grok-imagine alias", "grok-imagine", "grok/grok-imagine", false},
		{"grok-imagine-pro alias", "grok-imagine-pro", "grok/grok-imagine-pro", false},
		{"grok-2 alias", "grok-2", "grok/grok-2-image", false},
		{"xai alias", "xai", "grok/grok-imagine", false},
		{"aurora alias", "aurora", "grok/grok-imagine", false},
		{"exact provider ID", "gemini/flash-2.5", "gemini/flash-2.5", false},
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

func TestResolveModelName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantID  string
	}{
		{"flash alias resolves", "flash", "gemini-2.5-flash-image"},
		{"imagen alias resolves", "imagen", "imagen-4.0-generate-001"},
		{"nova alias resolves", "nova", "amazon.nova-canvas-v1:0"},
		{"grok alias resolves", "grok", "grok-imagine-image"},
		{"unknown returns original", "unknown-model", "unknown-model"},
		{"exact model ID returns itself", "gemini-2.5-flash-image", "gemini-2.5-flash-image"},
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
		{"gemini flash model", "gemini-2.5-flash-image", "gemini", false},
		{"gemini alias", "gemini", "gemini", false},
		{"imagen alias", "imagen-4", "vertex", false},
		{"nova-canvas alias", "nova-canvas", "bedrock", false},
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

func TestValidateModelForAPI(t *testing.T) {
	tests := []struct {
		name    string
		model   string
		api     string
		wantErr bool
	}{
		{"empty model no validation", "", "gemini", false},
		{"gemini model on gemini api", "gemini-2.5-flash-image", "gemini", false},
		{"gemini model on vertex api mismatch", "gemini-2.5-flash-image", "vertex", true},
		{"imagen on vertex api", "imagen-4", "vertex", false},
		{"nova on bedrock api", "nova-canvas", "bedrock", false},
		{"grok on grok api", "grok", "grok", false},
		{"unknown model", "totally-unknown-xyz", "gemini", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateModelForAPI(tt.model, tt.api)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetProviderPricing(t *testing.T) {
	reg := GetProviderRegistry()

	flashProvider, _ := reg.Get("gemini/flash-2.5")
	pro3Provider, _ := reg.Get("gemini/pro-3")
	novaProvider, _ := reg.Get("bedrock/nova-canvas")
	grokProvider, _ := reg.Get("grok/grok-imagine")

	tests := []struct {
		name        string
		provider    *Provider
		imageSize   string
		dimensions  string
		wantCost    float64
		wantDisplay string
	}{
		{
			name:     "gemini flash pricing",
			provider: flashProvider,
			wantCost: 0.039,
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
			name:       "nova canvas small",
			provider:   novaProvider,
			dimensions: "1024x1024",
			wantCost:   0.04,
		},
		{
			name:       "nova canvas large",
			provider:   novaProvider,
			dimensions: "2048x2048",
			wantCost:   0.08,
		},
		{
			name:       "nova canvas default dimensions",
			provider:   novaProvider,
			dimensions: "",
			wantCost:   0.04,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricing := GetProviderPricing(tt.provider, tt.imageSize, tt.dimensions)

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
	flashProvider, _ := reg.Get("gemini/flash-2.5")

	// 4K Gemini 3 Pro should be expensive ($0.24 > $0.05)
	expensive := GetProviderPricing(pro3Provider, "4K", "")
	assert.True(t, expensive.IsExpensive, "4K Gemini 3 Pro should be marked expensive")

	// Gemini Flash should not be expensive ($0.039 < $0.05)
	cheap := GetProviderPricing(flashProvider, "", "")
	assert.False(t, cheap.IsExpensive, "Gemini Flash should not be marked expensive")
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
		{"gemini needs api key", "gemini/flash-2.5", []string{"GEMINI_API_KEY"}},
		{"vertex needs project and location", "vertex/imagen-4", []string{"VERTEX_PROJECT", "VERTEX_LOCATION"}},
		{"bedrock needs region", "bedrock/nova-canvas", []string{"AWS_REGION"}},
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
