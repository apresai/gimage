package generate

import (
	"fmt"
	"strings"
)

// StyleTemplates maps style names to prompt enhancement templates
var StyleTemplates = map[string]string{
	"photorealistic": "highly detailed photorealistic image, professional photography, 8k resolution, realistic lighting",
	"artistic":       "artistic interpretation, creative composition, painterly style, expressive colors",
	"anime":          "anime style, manga art, vibrant colors, detailed character design",
	"cinematic":      "cinematic composition, dramatic lighting, movie still quality, depth of field",
	"digital-art":    "digital art, concept art quality, detailed illustration, modern aesthetic",
	"oil-painting":   "oil painting style, classical art, brushstroke texture, rich colors",
	"watercolor":     "watercolor painting, soft colors, fluid brushstrokes, delicate details",
	"3d-render":      "3D rendered image, high quality CGI, photorealistic materials, professional lighting",
}

// QualityEnhancements are generic quality-improving phrases
var QualityEnhancements = []string{
	"high quality",
	"detailed",
	"professional",
	"sharp focus",
	"well composed",
}

// EnhancePrompt takes a user prompt and enhances it for better AI image generation results
// It applies style templates, quality enhancements, and structural improvements
func EnhancePrompt(userPrompt string) string {
	if userPrompt == "" {
		return ""
	}

	// Trim and normalize whitespace
	prompt := strings.TrimSpace(userPrompt)
	prompt = normalizeWhitespace(prompt)

	// If the prompt is already detailed (>100 chars), return as-is
	if len(prompt) > 100 {
		return prompt
	}

	// Build enhanced prompt
	var parts []string
	parts = append(parts, prompt)

	// Add quality enhancements for short prompts
	if len(prompt) < 50 {
		parts = append(parts, strings.Join(QualityEnhancements, ", "))
	}

	return strings.Join(parts, ", ")
}

// EnhancePromptWithStyle enhances a prompt with a specific style template
func EnhancePromptWithStyle(userPrompt, style string) string {
	if userPrompt == "" {
		return ""
	}

	prompt := strings.TrimSpace(userPrompt)
	prompt = normalizeWhitespace(prompt)

	// Look up style template
	styleTemplate, exists := StyleTemplates[strings.ToLower(style)]
	if !exists {
		// If style not found, just return enhanced prompt without style
		return EnhancePrompt(userPrompt)
	}

	// Combine user prompt with style template
	return fmt.Sprintf("%s, %s", prompt, styleTemplate)
}

// ExtractKeywords extracts potential keywords from a prompt for metadata
func ExtractKeywords(prompt string) []string {
	// Simple keyword extraction: split on common delimiters and take first 10 words
	words := strings.FieldsFunc(prompt, func(r rune) bool {
		return r == ',' || r == '.' || r == ';' || r == ':' || r == ' '
	})

	// Filter out common stop words and short words
	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "as": true, "is": true, "was": true,
		"are": true, "were": true, "be": true, "been": true, "being": true,
	}

	var keywords []string
	for _, word := range words {
		word = strings.ToLower(strings.TrimSpace(word))
		if len(word) > 2 && !stopWords[word] {
			keywords = append(keywords, word)
			if len(keywords) >= 10 {
				break
			}
		}
	}

	return keywords
}

// ValidatePrompt checks if a prompt is suitable for image generation
func ValidatePrompt(prompt string) error {
	prompt = strings.TrimSpace(prompt)

	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}

	if len(prompt) < 3 {
		return fmt.Errorf("prompt is too short (minimum 3 characters)")
	}

	if len(prompt) > 2000 {
		return fmt.Errorf("prompt is too long (maximum 2000 characters)")
	}

	// Check for potentially problematic content markers
	prohibitedPatterns := []string{
		// These would typically be more comprehensive in production
		// For now, just basic validation
	}

	lowerPrompt := strings.ToLower(prompt)
	for _, pattern := range prohibitedPatterns {
		if strings.Contains(lowerPrompt, pattern) {
			return fmt.Errorf("prompt contains prohibited content: %s", pattern)
		}
	}

	return nil
}

// normalizeWhitespace removes extra whitespace and normalizes line breaks
func normalizeWhitespace(s string) string {
	// Replace multiple spaces with single space
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}

	// Normalize line breaks
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	// Remove leading/trailing whitespace from each line
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}

	return strings.Join(lines, "\n")
}

// TruncatePrompt truncates a prompt to a maximum length, preserving word boundaries
func TruncatePrompt(prompt string, maxLength int) string {
	if len(prompt) <= maxLength {
		return prompt
	}

	// Find the last space before maxLength
	truncated := prompt[:maxLength]
	lastSpace := strings.LastIndex(truncated, " ")

	if lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}

// FormatPromptForDisplay formats a prompt for user-friendly display
func FormatPromptForDisplay(prompt string, maxLength int) string {
	prompt = normalizeWhitespace(prompt)

	if maxLength > 0 && len(prompt) > maxLength {
		return TruncatePrompt(prompt, maxLength)
	}

	return prompt
}
