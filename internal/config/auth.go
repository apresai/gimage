package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
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
		"  3. Config file: gimage config set gemini_api_key YOUR_KEY\n" +
		"Get your API key at: https://ai.google.dev/")
}

// ValidateVertexCredentials validates Vertex AI credentials and configuration
func ValidateVertexCredentials(project, location string) error {
	if project == "" {
		return fmt.Errorf("Vertex AI project ID is empty")
	}

	if location == "" {
		return fmt.Errorf("Vertex AI location is empty")
	}

	// Validate project ID format (lowercase letters, numbers, hyphens)
	projectPattern := regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)
	if !projectPattern.MatchString(project) {
		return fmt.Errorf("invalid Vertex AI project ID format: %s (must start with lowercase letter, contain only lowercase letters, numbers, and hyphens)", project)
	}

	// Validate location format (e.g., us-central1, europe-west1)
	validLocations := map[string]bool{
		"us-central1":     true,
		"us-east1":        true,
		"us-west1":        true,
		"europe-west1":    true,
		"europe-west4":    true,
		"asia-southeast1": true,
		"asia-northeast1": true,
	}

	if !validLocations[location] {
		return fmt.Errorf("unsupported Vertex AI location: %s (supported: us-central1, us-east1, us-west1, europe-west1, europe-west4, asia-southeast1, asia-northeast1)", location)
	}

	// Check for GOOGLE_APPLICATION_CREDENTIALS environment variable
	credPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credPath == "" {
		return fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS environment variable not set.\n" +
			"Please set it to the path of your service account JSON file:\n" +
			"  export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json\n" +
			"Learn more: https://cloud.google.com/docs/authentication/getting-started")
	}

	// Verify the credentials file exists
	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		return fmt.Errorf("credentials file not found at %s", credPath)
	}

	// Verify the file is readable
	file, err := os.Open(credPath)
	if err != nil {
		return fmt.Errorf("cannot read credentials file at %s: %w", credPath, err)
	}
	file.Close()

	return nil
}

// GetVertexCredentials retrieves Vertex AI credentials from multiple sources
// Priority order: flag parameters > environment variables > config file
func GetVertexCredentials(flagProject, flagLocation string) (project, location string, err error) {
	// 1. Check command-line flags (highest priority)
	if flagProject != "" {
		project = flagProject
	}
	if flagLocation != "" {
		location = flagLocation
	}

	// 2. Check environment variables
	if project == "" {
		if envProject := os.Getenv("VERTEX_PROJECT"); envProject != "" {
			project = envProject
		}
	}
	if location == "" {
		if envLocation := os.Getenv("VERTEX_LOCATION"); envLocation != "" {
			location = envLocation
		}
	}

	// 3. Load from config file
	if project == "" || location == "" {
		cfg, loadErr := LoadConfig()
		if loadErr != nil {
			return "", "", fmt.Errorf("failed to load config: %w", loadErr)
		}

		if project == "" && cfg.VertexProject != "" {
			project = cfg.VertexProject
		}
		if location == "" && cfg.VertexLocation != "" {
			location = cfg.VertexLocation
		}
	}

	// Check if we have both required values
	if project == "" {
		return "", "", fmt.Errorf("Vertex AI project ID not found. Please set it via:\n" +
			"  1. Command flag: --project YOUR_PROJECT_ID\n" +
			"  2. Environment variable: export VERTEX_PROJECT=YOUR_PROJECT_ID\n" +
			"  3. Config file: gimage config set vertex_project YOUR_PROJECT_ID")
	}

	if location == "" {
		// Use default if not specified
		location = "us-central1"
	}

	// Validate credentials
	if err := ValidateVertexCredentials(project, location); err != nil {
		return "", "", err
	}

	return project, location, nil
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
		"  2. Config file: gimage config set grok_api_key YOUR_KEY\n" +
		"Get your API key at: https://console.x.ai")
}

// SanitizeAPIKey returns a sanitized version of an API key for safe logging
// Shows only the first 4 and last 4 characters
func SanitizeAPIKey(key string) string {
	if key == "" {
		return "<empty>"
	}

	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}

	// Calculate the number of asterisks needed (total length - 8 chars shown)
	numAsterisks := len(key) - 8
	return key[:4] + strings.Repeat("*", numAsterisks) + key[len(key)-4:]
}
