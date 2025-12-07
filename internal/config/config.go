package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Config represents the gimage configuration
type Config struct {
	GeminiAPIKey          string
	VertexAPIKey          string // For Vertex AI Express Mode
	VertexProject         string
	VertexLocation        string
	VertexCredentialsPath string
	AWSAccessKeyID        string // For AWS Bedrock SDK
	AWSSecretAccessKey    string // For AWS Bedrock SDK
	AWSRegion             string // AWS region (default: us-east-1)
	AWSProfile            string // AWS profile name
	AWSBedrockAPIKey      string // For AWS Bedrock REST API (bearer token)
	GrokAPIKey            string // For xAI Grok API
	DefaultAPI            string
	DefaultModel          string
	DefaultSize           string
	CacheDir              string
	LogLevel              string
}

// LoadConfig loads the configuration from file, environment variables, and defaults
// Priority order: environment variables > config file > defaults
func LoadConfig() (*Config, error) {
	cfg := &Config{
		// Set defaults
		DefaultAPI:     "gemini",
		DefaultModel:   "gemini-3-pro-image-preview",
		DefaultSize:    "1024x1024",
		VertexLocation: "us-central1",
		AWSRegion:      "us-east-1",
		LogLevel:       "info",
	}

	// Get config file path
	configPath := GetConfigPath()

	// Try to read config file if it exists
	if _, err := os.Stat(configPath); err == nil {
		if err := parseMarkdownConfig(configPath, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Override with environment variables if set (highest priority)
	if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
		cfg.GeminiAPIKey = apiKey
	}
	if apiKey := os.Getenv("VERTEX_API_KEY"); apiKey != "" {
		cfg.VertexAPIKey = apiKey
	}
	if project := os.Getenv("VERTEX_PROJECT"); project != "" {
		cfg.VertexProject = project
	}
	if location := os.Getenv("VERTEX_LOCATION"); location != "" {
		cfg.VertexLocation = location
	}
	if credsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); credsPath != "" {
		cfg.VertexCredentialsPath = credsPath
	}
	if accessKey := os.Getenv("AWS_ACCESS_KEY_ID"); accessKey != "" {
		cfg.AWSAccessKeyID = accessKey
	}
	if secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY"); secretKey != "" {
		cfg.AWSSecretAccessKey = secretKey
	}
	if region := os.Getenv("AWS_REGION"); region != "" {
		cfg.AWSRegion = region
	}
	if profile := os.Getenv("AWS_PROFILE"); profile != "" {
		cfg.AWSProfile = profile
	}
	if bedrockKey := os.Getenv("AWS_BEARER_TOKEN_BEDROCK"); bedrockKey != "" {
		cfg.AWSBedrockAPIKey = bedrockKey
	}
	if grokKey := os.Getenv("GROK_API_KEY"); grokKey != "" {
		cfg.GrokAPIKey = grokKey
	}
	if logLevel := os.Getenv("GIMAGE_LOG_LEVEL"); logLevel != "" {
		cfg.LogLevel = logLevel
	}

	// Apply defaults if values are still empty
	if cfg.DefaultAPI == "" {
		cfg.DefaultAPI = "gemini"
	}
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = "gemini-3-pro-image-preview"
	}
	if cfg.DefaultSize == "" {
		cfg.DefaultSize = "1024x1024"
	}
	if cfg.VertexLocation == "" {
		cfg.VertexLocation = "us-central1"
	}
	if cfg.AWSRegion == "" {
		cfg.AWSRegion = "us-east-1"
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	// Expand home directory in paths if present
	if cfg.CacheDir != "" {
		cfg.CacheDir = expandHome(cfg.CacheDir)
	}
	if cfg.VertexCredentialsPath != "" {
		cfg.VertexCredentialsPath = expandHome(cfg.VertexCredentialsPath)
	}

	return cfg, nil
}

// parseMarkdownConfig parses a markdown config file into a Config struct
// Format: **key**: value
func parseMarkdownConfig(path string, cfg *Config) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Pattern to match: **key**: value
	pattern := regexp.MustCompile(`^\*\*([a-z_]+)\*\*:\s*(.+)$`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Try to match the pattern
		matches := pattern.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}

		key := matches[1]
		value := strings.TrimSpace(matches[2])

		// Map key to config field
		switch key {
		case "gemini_api_key":
			cfg.GeminiAPIKey = value
		case "vertex_api_key":
			cfg.VertexAPIKey = value
		case "vertex_project":
			cfg.VertexProject = value
		case "vertex_location":
			cfg.VertexLocation = value
		case "vertex_credentials_path":
			cfg.VertexCredentialsPath = value
		case "aws_access_key_id":
			cfg.AWSAccessKeyID = value
		case "aws_secret_access_key":
			cfg.AWSSecretAccessKey = value
		case "aws_region":
			cfg.AWSRegion = value
		case "aws_profile":
			cfg.AWSProfile = value
		case "aws_bedrock_api_key":
			cfg.AWSBedrockAPIKey = value
		case "grok_api_key":
			cfg.GrokAPIKey = value
		case "default_api":
			cfg.DefaultAPI = value
		case "default_model":
			cfg.DefaultModel = value
		case "default_size":
			cfg.DefaultSize = value
		case "cache_dir":
			cfg.CacheDir = value
		case "log_level":
			cfg.LogLevel = value
		}
	}

	return scanner.Err()
}

// SaveConfig saves the configuration to file with secure permissions (0600)
func SaveConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Validate config before saving
	if err := ValidateConfig(cfg); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Get config file path
	configPath := GetConfigPath()

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Build markdown content with security warnings
	var content strings.Builder
	content.WriteString("# Gimage Configuration\n\n")
	content.WriteString("⚠️  SECURITY WARNING ⚠️\n")
	content.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	content.WriteString("This file contains SENSITIVE API KEYS stored in PLAINTEXT.\n\n")
	content.WriteString("Security Best Practices:\n")
	content.WriteString("  • File permissions: 0600 (only you can read/write)\n")
	content.WriteString("  • NEVER commit this file to version control\n")
	content.WriteString("  • NEVER share this file or its contents\n")
	content.WriteString("  • PREFER environment variables over this file\n")
	content.WriteString("  • Rotate your API keys regularly (every 90 days)\n")
	content.WriteString("  • Use separate keys for dev/staging/production\n\n")
	content.WriteString("Environment variables take precedence over this file:\n")
	content.WriteString("  export GEMINI_API_KEY=\"your-key\"\n")
	content.WriteString("  export VERTEX_API_KEY=\"your-key\"\n")
	content.WriteString("  export AWS_ACCESS_KEY_ID=\"your-key\"\n\n")
	content.WriteString("For more security info: https://github.com/apresai/gimage#security\n\n")
	content.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	content.WriteString("## Configuration\n\n")

	// Write each field if it has a value
	if cfg.GeminiAPIKey != "" {
		content.WriteString(fmt.Sprintf("**gemini_api_key**: %s\n", cfg.GeminiAPIKey))
	}
	if cfg.VertexAPIKey != "" {
		content.WriteString(fmt.Sprintf("**vertex_api_key**: %s\n", cfg.VertexAPIKey))
	}
	if cfg.VertexProject != "" {
		content.WriteString(fmt.Sprintf("**vertex_project**: %s\n", cfg.VertexProject))
	}
	if cfg.VertexLocation != "" {
		content.WriteString(fmt.Sprintf("**vertex_location**: %s\n", cfg.VertexLocation))
	}
	if cfg.VertexCredentialsPath != "" {
		content.WriteString(fmt.Sprintf("**vertex_credentials_path**: %s\n", cfg.VertexCredentialsPath))
	}
	if cfg.AWSAccessKeyID != "" {
		content.WriteString(fmt.Sprintf("**aws_access_key_id**: %s\n", cfg.AWSAccessKeyID))
	}
	if cfg.AWSSecretAccessKey != "" {
		content.WriteString(fmt.Sprintf("**aws_secret_access_key**: %s\n", cfg.AWSSecretAccessKey))
	}
	if cfg.AWSRegion != "" {
		content.WriteString(fmt.Sprintf("**aws_region**: %s\n", cfg.AWSRegion))
	}
	if cfg.AWSProfile != "" {
		content.WriteString(fmt.Sprintf("**aws_profile**: %s\n", cfg.AWSProfile))
	}
	if cfg.AWSBedrockAPIKey != "" {
		content.WriteString(fmt.Sprintf("**aws_bedrock_api_key**: %s\n", cfg.AWSBedrockAPIKey))
	}
	if cfg.GrokAPIKey != "" {
		content.WriteString(fmt.Sprintf("**grok_api_key**: %s\n", cfg.GrokAPIKey))
	}
	if cfg.DefaultAPI != "" {
		content.WriteString(fmt.Sprintf("**default_api**: %s\n", cfg.DefaultAPI))
	}
	if cfg.DefaultModel != "" {
		content.WriteString(fmt.Sprintf("**default_model**: %s\n", cfg.DefaultModel))
	}
	if cfg.DefaultSize != "" {
		content.WriteString(fmt.Sprintf("**default_size**: %s\n", cfg.DefaultSize))
	}
	if cfg.CacheDir != "" {
		content.WriteString(fmt.Sprintf("**cache_dir**: %s\n", cfg.CacheDir))
	}
	if cfg.LogLevel != "" {
		content.WriteString(fmt.Sprintf("**log_level**: %s\n", cfg.LogLevel))
	}

	// Write config file with secure permissions (0600)
	if err := os.WriteFile(configPath, []byte(content.String()), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfigPath returns the path to the config file
// Checks GIMAGE_CONFIG environment variable first, then defaults to ~/.gimage/config.md
func GetConfigPath() string {
	// Check for custom config path from environment
	if configPath := os.Getenv("GIMAGE_CONFIG"); configPath != "" {
		return expandHome(configPath)
	}

	// Default to ~/.gimage/config.md
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if we can't get home
		return ".gimage/config.md"
	}

	return filepath.Join(home, ".gimage", "config.md")
}

// ValidateConfig validates the configuration values
func ValidateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Validate default_api if set
	if cfg.DefaultAPI != "" {
		validAPIs := map[string]bool{
			"gemini":  true,
			"vertex":  true,
			"bedrock": true,
			"grok":    true,
		}
		if !validAPIs[cfg.DefaultAPI] {
			return fmt.Errorf("default_api must be 'gemini', 'vertex', 'bedrock', or 'grok', got: %s", cfg.DefaultAPI)
		}
	}

	// Validate default_size if set
	if cfg.DefaultSize != "" {
		if err := validateSize(cfg.DefaultSize); err != nil {
			return fmt.Errorf("invalid default_size: %w", err)
		}
	}

	// Validate log_level if set
	if cfg.LogLevel != "" {
		validLevels := map[string]bool{
			"debug": true,
			"info":  true,
			"warn":  true,
			"error": true,
		}
		if !validLevels[cfg.LogLevel] {
			return fmt.Errorf("invalid log_level: %s (must be debug, info, warn, or error)", cfg.LogLevel)
		}
	}

	// Validate Vertex project ID format if set
	if cfg.VertexProject != "" {
		projectPattern := regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$`)
		if !projectPattern.MatchString(cfg.VertexProject) {
			return fmt.Errorf("invalid vertex_project format: %s (must start with lowercase letter, contain only lowercase letters, numbers, and hyphens)", cfg.VertexProject)
		}
	}

	return nil
}

// validateSize validates image size format (e.g., "1024x1024", "512x512")
func validateSize(size string) error {
	sizePattern := regexp.MustCompile(`^(\d+)x(\d+)$`)
	if !sizePattern.MatchString(size) {
		return fmt.Errorf("size must be in format WIDTHxHEIGHT (e.g., 1024x1024), got: %s", size)
	}
	return nil
}

// expandHome expands the tilde (~) in a path to the user's home directory
func expandHome(path string) string {
	if path == "" {
		return path
	}

	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}

	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}

	return path
}
