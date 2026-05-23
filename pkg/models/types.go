package models

// GenerateOptions contains configuration for AI image generation
type GenerateOptions struct {
	Model          string
	Size           string
	AspectRatio    string
	Style          string
	NegativePrompt string
	Seed           int64
	ImageSize      string  // For Gemini 3 Pro: "1K", "2K", "4K" (native upscaling)
	CfgScale       float64 // Guidance scale for Bedrock Nova Canvas (1.0-10.0, default 7.0)
	NumberOfImages int     // Number of images to generate (1-4, default 1)
	OutputFormat   string  // Output format: "png", "jpeg", "webp" (default varies by API)
	ResizeMode     string  // "stretch", "fit", "crop" (default "crop")

	// Gemini 3+ exclusive features (ignored by other providers / older Gemini models)
	ThinkingLevel      string   // "minimal", "low", "medium", "high" — controls reasoning depth on Gemini 3+
	WebSearchGrounding bool     // Enables Google Search grounding via tools field on Gemini 3+
	InputImages        []string // Local file paths to reference images for compositional editing (Nano Banana)
}

// GeneratedImage represents the result of an AI image generation request
type GeneratedImage struct {
	Data     []byte
	Format   string
	Width    int
	Height   int
	Metadata map[string]string
}

// GeneratedImages represents multiple generated images from a single request
type GeneratedImages struct {
	Images   []*GeneratedImage
	Metadata map[string]string
}
