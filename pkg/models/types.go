package models

// GenerateOptions contains configuration for AI image generation
type GenerateOptions struct {
	Model          string
	Size           string
	AspectRatio    string
	Style          string
	NegativePrompt string
	Seed           int64
	ImageSize      string // For Gemini 3 Pro: "1K", "2K", "4K" (native upscaling)
}

// GeneratedImage represents the result of an AI image generation request
type GeneratedImage struct {
	Data     []byte
	Format   string
	Width    int
	Height   int
	Metadata map[string]string
}
