package generate

import (
	"fmt"
	"strings"
)

// PricingEntry is the authoritative pricing record for a model.
// All cost calculations in the codebase MUST go through this registry
// via GetProviderPricing. To update pricing, edit ModelPricing below —
// no other file should hardcode per-image costs.
type PricingEntry struct {
	ModelID        string
	Summary        string  // audit-readable summary, e.g. "$0.134/image (1K/2K), $0.24/image (4K)"
	Representative float64 // single-number price used for list displays (flat price or default tier)

	// Calculate returns the cost for a generation request.
	// imageSize is "0.5K"/"1K"/"2K"/"4K" for Gemini tiered models (or "" for default).
	// dimensions is "WIDTHxHEIGHT" for dimension-tiered pricing (or "" for default).
	// style is the user-supplied style string (e.g. "photorealistic").
	Calculate func(imageSize, dimensions, style string) float64

	// DisplayContext returns an optional suffix added to the price display,
	// e.g. "(1K/2K resolution)" or "(≤1024px, premium)". Nil means flat display.
	DisplayContext func(imageSize, dimensions, style string) string

	Source   string // URL where the price was verified
	Verified string // ISO date (YYYY-MM-DD) of last verification
	Note     string // caveats — legacy status, unresolved dimensions, etc.
}

// ModelPricing is the single source of truth for per-image costs.
// Every registered provider in providers.go should have an entry here.
// TestModelPricingRegistry validates coverage.
//
//nolint:gochecknoglobals // This is the central pricing source of truth
var ModelPricing = map[string]PricingEntry{
	"gemini-2.5-flash-image": {
		ModelID:        "gemini-2.5-flash-image",
		Summary:        "$0.039/image",
		Representative: 0.039,
		Calculate:      flatPrice(0.039),
		Source:         "https://ai.google.dev/gemini-api/docs/pricing",
		Verified:       "2026-05-23",
		Note:           "Batch (50% off, $0.0195) and Priority (1.8x, $0.0702) available via separate :batchGenerateContent endpoint — not wired into gimage CLI.",
	},
	"gemini-3-pro-image": {
		ModelID:        "gemini-3-pro-image",
		Summary:        "$0.134/image (1K/2K), $0.24/image (4K)",
		Representative: 0.134,
		Calculate: imageSizeTieredPrice(map[string]float64{
			"1K": 0.134, "2K": 0.134, "4K": 0.24,
			"": 0.134,
		}),
		DisplayContext: imageSizeSuffix,
		Source:         "https://ai.google.dev/gemini-api/docs/pricing",
		Verified:       "2026-05-23",
		Note:           "Batch (50% off, $0.067 @1K/2K) available via separate :batchGenerateContent endpoint — not wired into gimage CLI.",
	},
	"gemini-3-pro-image-preview": {
		ModelID:        "gemini-3-pro-image-preview",
		Summary:        "$0.134/image (1K/2K), $0.24/image (4K) legacy preview",
		Representative: 0.134,
		Calculate: imageSizeTieredPrice(map[string]float64{
			"1K": 0.134, "2K": 0.134, "4K": 0.24,
			"": 0.134,
		}),
		DisplayContext: imageSizeSuffix,
		Source:         "https://ai.google.dev/gemini-api/docs/pricing",
		Verified:       "2026-05-23",
		Note:           "Legacy preview retained for cost audit only; discontinued 2026-06-25. Batch endpoint pricing matched GA before shutdown.",
	},
	"gemini-3.1-flash-image": {
		ModelID:        "gemini-3.1-flash-image",
		Summary:        "$0.045 (0.5K), $0.067 (1K), $0.101 (2K), $0.151 (4K)",
		Representative: 0.067,
		Calculate: imageSizeTieredPrice(map[string]float64{
			"0.5K": 0.045, "1K": 0.067, "2K": 0.101, "4K": 0.151,
			"": 0.067,
		}),
		DisplayContext: imageSizeSuffix,
		Source:         "https://ai.google.dev/gemini-api/docs/pricing",
		Verified:       "2026-05-23",
		Note:           "Batch (~50% off, $0.022 @0.5K, $0.034 @1K, $0.050 @2K, $0.076 @4K) available via separate :batchGenerateContent endpoint — not wired into gimage CLI.",
	},
	"gemini-3.1-flash-image-preview": {
		ModelID:        "gemini-3.1-flash-image-preview",
		Summary:        "$0.045 (0.5K), $0.067 (1K), $0.101 (2K), $0.151 (4K) legacy preview",
		Representative: 0.067,
		Calculate: imageSizeTieredPrice(map[string]float64{
			"0.5K": 0.045, "1K": 0.067, "2K": 0.101, "4K": 0.151,
			"": 0.067,
		}),
		DisplayContext: imageSizeSuffix,
		Source:         "https://ai.google.dev/gemini-api/docs/pricing",
		Verified:       "2026-05-23",
		Note:           "Legacy preview retained for cost audit only; discontinued 2026-06-25. Batch endpoint pricing matched GA before shutdown.",
	},
	// Imagen pricing entries removed: gimage no longer runs any Imagen model
	// (the former imagen-* aliases were Gemini 3.1 Flash via Vertex, priced under
	// the "gemini-3.1-flash-image" tiered entry above).
	"grok-imagine-image": {
		ModelID:        "grok-imagine-image",
		Summary:        "$0.02/image (1K/2K)",
		Representative: 0.02,
		Calculate:      flatPrice(0.02),
		Source:         "https://docs.x.ai/developers/pricing",
		Verified:       "2026-06-21",
	},
	"grok-imagine-image-quality": {
		ModelID:        "grok-imagine-image-quality",
		Summary:        "$0.05/image (1K), $0.07/image (2K)",
		Representative: 0.05,
		Calculate: imageSizeTieredPrice(map[string]float64{
			"1K": 0.05, "2K": 0.07,
			"": 0.05,
		}),
		DisplayContext: imageSizeSuffix,
		Source:         "https://docs.x.ai/developers/pricing",
		Verified:       "2026-06-21",
		Note:           "Replaces grok-imagine-image-pro retired by xAI 2026-05-15",
	},
}

// LookupPricing returns the pricing entry for a model, or false if unknown.
func LookupPricing(modelID string) (PricingEntry, bool) {
	entry, ok := ModelPricing[modelID]
	return entry, ok
}

// flatPrice returns a Calculate function for a flat per-image price.
func flatPrice(price float64) func(string, string, string) float64 {
	return func(_, _, _ string) float64 { return price }
}

// imageSizeTieredPrice returns a Calculate function that picks a price based on
// the imageSize key ("0.5K", "1K", "2K", "4K"). Empty string falls back to
// tiers[""] which must be set by the caller.
func imageSizeTieredPrice(tiers map[string]float64) func(string, string, string) float64 {
	return func(imageSize, _, _ string) float64 {
		key := strings.ToUpper(imageSize)
		if price, ok := tiers[key]; ok {
			return price
		}
		return tiers[""]
	}
}

// imageSizeSuffix returns the display suffix for image-size-tiered pricing.
func imageSizeSuffix(imageSize, _, _ string) string {
	if imageSize == "" {
		return ""
	}
	return "(" + ImageSizeLabel(imageSize) + ")"
}

// formatPricingDisplay builds the CalculatedPricing.Display string from an
// entry's computed cost and optional display context suffix.
func formatPricingDisplay(entry PricingEntry, cost float64, imageSize, dimensions, style string) string {
	if entry.DisplayContext != nil {
		if suffix := entry.DisplayContext(imageSize, dimensions, style); suffix != "" {
			return fmt.Sprintf("$%.4f/image %s", cost, suffix)
		}
	}
	return fmt.Sprintf("$%.4f/image", cost)
}
