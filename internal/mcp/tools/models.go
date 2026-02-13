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
		Description: "List all available AI image generation providers with pricing, capabilities, and authentication status. Use this FIRST to discover: which providers are configured and ready to use, pricing for each provider (FREE vs paid), size limits (1024x1024 vs 2048x2048), and which credentials are missing. Returns provider IDs (e.g., 'gemini/flash-2.5', 'vertex/imagen-4'), pricing (FREE 500/day, $0.04/image, etc.), and availability status. RECOMMENDED: Call this before generate_image to choose the best provider for your needs.",
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
					"currency":   p.Pricing.Currency,
					"free_tier":  p.Pricing.FreeTier,
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
					"provider_id":     p.ID,
					"name":            p.Name,
					"api":             p.API,
					"model_id":        p.ModelID,
					"description":     p.Description,
					"available":       status.Configured,
					"missing_credentials": status.Missing,

					// Pricing information
					"pricing":         pricingMap,
					"pricing_summary": pricingSummary,

					// Capabilities
					"supports_styles":          p.Capabilities.SupportsStyles,
					"supports_negative_prompt": p.Capabilities.SupportsNegativePrompt,
					"supports_seed":            p.Capabilities.SupportsSeed,
					"max_prompt_length":        p.Capabilities.MaxPromptLength,
				}

				providers = append(providers, providerData)
			}

			// Get default provider (first configured one, preferring free tier)
			var defaultProviderID string
			var defaultProviderName string
			var defaultProviderPricing string
			for _, status := range statuses {
				if status.Configured {
					defaultProviderID = status.Provider.ID
					defaultProviderName = status.Provider.Name
					if status.Provider.Pricing.FreeTier {
						defaultProviderPricing = fmt.Sprintf("FREE (%s)", status.Provider.Pricing.FreeTierLimit)
					} else if status.Provider.Pricing.CostPerImage != nil {
						defaultProviderPricing = fmt.Sprintf("$%.4f/image", *status.Provider.Pricing.CostPerImage)
					}
					// Prefer free tier, so break on first free provider
					if status.Provider.Pricing.FreeTier {
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
				"providers": providers,
				"total":     len(providers),
				"configured": configuredCount,
				"default_provider": map[string]interface{}{
					"provider_id":     defaultProviderID,
					"name":            defaultProviderName,
					"pricing_summary": defaultProviderPricing,
				},
				"pricing_note": "Costs shown are in USD. Each provider offers specific models optimized for different use cases (gemini/flash-2.5 for affordable iteration, vertex/imagen-4 for highest quality, bedrock/nova-canvas for AWS integration).",
				"recommendations": map[string]interface{}{
					"budget_users": "gemini/flash-2.5 ($0.039/image via Gemini API, most affordable)",
					"paid_users":   "vertex/imagen-4 ($0.04/image, highest quality)",
					"aws_users":    "bedrock/nova-canvas ($0.08/image, AWS integration)",
				},
			}, nil
		},
	}

	server.RegisterTool(tool)
}
