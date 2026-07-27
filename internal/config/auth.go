package config

import (
	"fmt"
	"os"
	"regexp"
)

// ValidateGeminiAPIKey validates the format of a Gemini API key
// Gemini API keys typically start with certain prefixes and have specific formats
func ValidateGeminiAPIKey(key string) error {
	if key == "" {
		return fmt.Errorf("Gemini API key is empty")
	}

	// Basic validation: check length and format
	// Gemini API keys are typically alphanumeric with hyphens or underscores
	if len(key) < 20 {
		return fmt.Errorf("Gemini API key appears to be too short (expected at least 20 characters)")
	}

	// Check for valid characters (alphanumeric, hyphens, underscores)
	validKeyPattern := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	if !validKeyPattern.MatchString(key) {
		return fmt.Errorf("Gemini API key contains invalid characters (only alphanumeric, hyphens, and underscores allowed)")
	}

	return nil
}

// GetGeminiAPIKey retrieves the Gemini API key from multiple sources
// Priority order: flagKey parameter > GEMINI_API_KEY env var > config file
func GetGeminiAPIKey(flagKey string) (string, error) {
	// 1. Check command-line flag (highest priority)
	if flagKey != "" {
		if err := ValidateGeminiAPIKey(flagKey); err != nil {
			return "", fmt.Errorf("invalid API key provided via flag: %w", err)
		}
		return flagKey, nil
	}

	// 2. Check environment variable
	if envKey := os.Getenv("GEMINI_API_KEY"); envKey != "" {
		if err := ValidateGeminiAPIKey(envKey); err != nil {
			return "", fmt.Errorf("invalid API key in GEMINI_API_KEY environment variable: %w", err)
		}
		return envKey, nil
	}

	// 3. Load from config file
	cfg, err := LoadConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.GeminiAPIKey != "" {
		if err := ValidateGeminiAPIKey(cfg.GeminiAPIKey); err != nil {
			return "", fmt.Errorf("invalid API key in config file: %w", err)
		}
		return cfg.GeminiAPIKey, nil
	}

	// No API key found
	return "", fmt.Errorf("Gemini API key not found. Please set it via:\n" +
		"  1. Command flag: --api-key YOUR_KEY\n" +
		"  2. Environment variable: export GEMINI_API_KEY=YOUR_KEY\n" +
		"  3. Auth setup: gimage auth setup gemini\n" +
		"Get your API key at: https://ai.google.dev/")
}

// GetVertexAPIKey retrieves the Vertex AI API key from multiple sources
// Priority order: flagKey parameter > VERTEX_API_KEY env var > config file
// Returns empty string if no API key is found (Express Mode not configured)
func GetVertexAPIKey(flagKey string) (string, error) {
	// 1. Check command-line flag (highest priority)
	if flagKey != "" {
		return flagKey, nil
	}

	// 2. Check environment variable
	if envKey := os.Getenv("VERTEX_API_KEY"); envKey != "" {
		return envKey, nil
	}

	// 3. Load from config file
	cfg, err := LoadConfig()
	if err != nil {
		// If config doesn't exist, return empty (not an error - user might be using service account)
		return "", nil
	}

	// Return the API key if found (might be empty string - that's ok)
	return cfg.VertexAPIKey, nil
}

// HasGeminiCredentials checks if Gemini API credentials are available
func HasGeminiCredentials() bool {
	// Check environment variable
	if os.Getenv("GEMINI_API_KEY") != "" {
		return true
	}

	// Check config file
	cfg, err := LoadConfig()
	if err == nil && cfg.GeminiAPIKey != "" {
		return true
	}

	return false
}

// HasVertexCredentials checks if Vertex AI credentials are available
// Returns true if either Express Mode (API key) or Full Mode (service account) credentials exist
func HasVertexCredentials() bool {
	// Check for Express Mode (API key)
	if os.Getenv("VERTEX_API_KEY") != "" {
		return true
	}

	cfg, err := LoadConfig()
	if err == nil && cfg.VertexAPIKey != "" {
		return true
	}

	// Check for Full Mode (service account)
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		credPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
		if _, err := os.Stat(credPath); err == nil {
			return true
		}
	}

	return false
}

// HasGrokCredentials checks if xAI Grok API credentials are available
func HasGrokCredentials() bool {
	// Check environment variable
	if os.Getenv("GROK_API_KEY") != "" {
		return true
	}

	// Check config file
	cfg, err := LoadConfig()
	if err == nil && cfg.GrokAPIKey != "" {
		return true
	}

	return false
}

// GetGrokAPIKey retrieves the Grok API key from multiple sources
// Priority order: flagKey parameter > GROK_API_KEY env var > config file
func GetGrokAPIKey(flagKey string) (string, error) {
	// 1. Check command-line flag (highest priority)
	if flagKey != "" {
		return flagKey, nil
	}

	// 2. Check environment variable
	if envKey := os.Getenv("GROK_API_KEY"); envKey != "" {
		return envKey, nil
	}

	// 3. Load from config file
	cfg, err := LoadConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.GrokAPIKey != "" {
		return cfg.GrokAPIKey, nil
	}

	// No API key found
	return "", fmt.Errorf("Grok API key not found. Please set it via:\n" +
		"  1. Environment variable: export GROK_API_KEY=YOUR_KEY\n" +
		"  2. Auth setup: gimage auth setup grok\n" +
		"Get your API key at: https://console.x.ai")
}
