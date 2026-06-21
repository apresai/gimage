package tools

import (
	"fmt"

	"github.com/apresai/gimage/internal/generate"
	"github.com/apresai/gimage/internal/mcp"
)

// RegisterListModelsTool registers the list_models tool
func RegisterListModelsTool(server *mcp.MCPServer) {
	tool := mcp.Tool{
		Name:        "list_models",
		Description: "List all available AI image generation providers with pricing, capabilities, and authentication status. Use this FIRST to discover: which providers are configured and ready to use, current per-image pricing, native image-size support, and which credentials are missing. Returns provider IDs (e.g., 'gemini/pro-3', 'vertex/flash-3.1'), pricing, capability flags, and availability status. RECOMMENDED: Call this before generate_image to choose the best provider for your needs.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: func(args map[string]interface{}) (map[string]interface{}, error) {
			// Use the Provider system (single source of truth)
			registry := generate.GetProviderRegistry()
			statuses := registry.GetAuthStatus()

			// Build provider list with availability
			providers := []map[string]interface{}{}
			for _, status := range statuses {
				p := status.Provider

				// Convert pricing info to map
				pricingMap := map[string]interface{}{
					"currency":  p.Pricing.Currency,
					"free_tier": p.Pricing.FreeTier,
				}

				if p.Pricing.CostPerImage != nil {
					pricingMap["cost_per_image"] = *p.Pricing.CostPerImage
				}
				if p.Pricing.FreeTierLimit != "" {
					pricingMap["free_tier_limit"] = p.Pricing.FreeTierLimit
				}

				// Format pricing summary
				pricingSummary := "Variable"
				if p.Pricing.FreeTier {
					pricingSummary = fmt.Sprintf("FREE (%s)", p.Pricing.FreeTierLimit)
				} else if p.Pricing.CostPerImage != nil {
					pricingSummary = fmt.Sprintf("$%.4f/image", *p.Pricing.CostPerImage)
				}

				providerData := map[string]interface{}{
					"provider_id":         p.ID,
					"name":                p.Name,
					"api":                 p.API,
					"model_id":            p.ModelID,
					"description":         p.Description,
					"available":           status.Configured,
					"missing_credentials": status.Missing,

					// Pricing information
					"pricing":         pricingMap,
					"pricing_summary": pricingSummary,

					// Capabilities
					"supports_styles":          p.Capabilities.SupportsStyles,
					"supports_negative_prompt": p.Capabilities.SupportsNegativePrompt,
					"supports_seed":            p.Capabilities.SupportsSeed,
					"supports_image_size":      p.Capabilities.SupportsImageSize,
					"supports_aspect_ratio":    p.Capabilities.SupportsAspectRatio,
					"supports_thinking":        p.Capabilities.SupportsThinking,
					"supports_grounding":       p.Capabilities.SupportsGrounding,
					"supports_input_images":    p.Capabilities.SupportsInputImages,
					"max_prompt_length":        p.Capabilities.MaxPromptLength,
				}

				providers = append(providers, providerData)
			}

			// Resolve the actual default provider (generate.DefaultModel), not the
			// first/last configured one. Report its real identity, registry pricing,
			// and whether the caller currently has credentials for it.
			var defaultProviderID string
			var defaultProviderName string
			var defaultProviderPricing string
			var defaultProviderConfigured bool
			if defP, defErr := registry.ResolveProvider(generate.DefaultModel); defErr == nil && defP != nil {
				defaultProviderID = defP.ID
				defaultProviderName = defP.Name
				defaultProviderPricing = generate.GetProviderPricing(defP, "", "", "").Display
				for _, status := range statuses {
					if status.Provider != nil && status.Provider.ID == defP.ID {
						defaultProviderConfigured = status.Configured
						break
					}
				}
			}

			// Count configured providers
			configuredCount := 0
			for _, status := range statuses {
				if status.Configured {
					configuredCount++
				}
			}

			return map[string]interface{}{
				"providers":  providers,
				"total":      len(providers),
				"configured": configuredCount,
				"default_provider": map[string]interface{}{
					"provider_id":     defaultProviderID,
					"name":            defaultProviderName,
					"pricing_summary": defaultProviderPricing,
					"configured":      defaultProviderConfigured,
				},
				"pricing_note": "Costs shown are in USD. The vertex-flash* providers (Gemini 3.1 Flash via Vertex) use tiered pricing by resolution via generateContent.",
				"recommendations": map[string]interface{}{
					"budget_users": "gemini/flash-2.5 ($0.039/image via Gemini API, most affordable)",
					"paid_users":   "vertex-flash (Gemini 3.1 Flash via Vertex, $0.045-$0.151/image by resolution)",
				},
			}, nil
		},
	}

	server.RegisterTool(tool)
}
