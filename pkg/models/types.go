package models

// GenerateOptions contains configuration for AI image generation
type GenerateOptions struct {
	Model          string
	Size           string
	AspectRatio    string
	Style          string
	Seed           int64
	ImageSize      string // For Gemini 3 Pro / Grok: "1K", "2K", "4K" (native upscaling; Grok 1K/2K only)
	NumberOfImages int    // Number of images to generate (Grok exact; Gemini best-effort)
	OutputFormat   string // Output format: "png", "jpeg", "webp" (default varies by API)
	ResizeMode     string // "stretch", "fit", "crop" (default "crop")
	// User is an optional end-user identifier for abuse monitoring (Grok/xAI only).
	User string

	// Gemini 3+ exclusive features (ignored by other providers / older Gemini models)
	ThinkingLevel      string   // "minimal", "low", "medium", "high" — controls reasoning depth on Gemini 3+
	WebSearchGrounding bool     // Enables Google Search grounding via tools field on Gemini 3+
	// InputImages are reference images for compositional editing. Local paths (all
	// providers) or https:// URLs (Grok edits only; passed through to xAI).
	InputImages []string
}

// GeneratedImage represents the result of an AI image generation request
type GeneratedImage struct {
	Data     []byte
	Format   string
	Width    int
	Height   int
	Metadata map[string]string
}
