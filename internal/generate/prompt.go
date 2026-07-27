package generate

import (
	"fmt"
	"strings"
)

// QualityEnhancements are generic quality-improving phrases
var QualityEnhancements = []string{
	"high quality",
	"detailed",
	"professional",
	"sharp focus",
	"well composed",
}

// EnhancePrompt takes a user prompt and enhances it for better AI image generation results
// It applies quality enhancements and structural improvements
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
