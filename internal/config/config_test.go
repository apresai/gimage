package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with all defaults",
			cfg: &Config{
				DefaultAPI:  "gemini",
				DefaultSize: "1024x1024",
				LogLevel:    "info",
			},
			wantErr: false,
		},
		{
			name: "invalid default_api",
			cfg: &Config{
				DefaultAPI:  "invalid",
				DefaultSize: "1024x1024",
				LogLevel:    "info",
			},
			wantErr: true,
			errMsg:  "must be",
		},
		{
			name: "invalid default_size",
			cfg: &Config{
				DefaultAPI:  "gemini",
				DefaultSize: "abc",
				LogLevel:    "info",
			},
			wantErr: true,
			errMsg:  "invalid default_size",
		},
		{
			name: "invalid log_level",
			cfg: &Config{
				DefaultAPI:  "gemini",
				DefaultSize: "1024x1024",
				LogLevel:    "verbose",
			},
			wantErr: true,
			errMsg:  "invalid log_level",
		},
		{
			name: "invalid vertex_project uppercase",
			cfg: &Config{
				DefaultAPI:    "gemini",
				DefaultSize:   "1024x1024",
				LogLevel:      "info",
				VertexProject: "INVALID",
			},
			wantErr: true,
			errMsg:  "invalid vertex_project",
		},
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
			errMsg:  "config cannot be nil",
		},
		{
			name:    "empty config - all empty strings are valid",
			cfg:     &Config{},
			wantErr: false,
		},
		{
			name: "valid config with vertex api",
			cfg: &Config{
				DefaultAPI:  "vertex",
				DefaultSize: "512x768",
				LogLevel:    "debug",
			},
			wantErr: false,
		},
		{
			name: "valid config with bedrock api",
			cfg: &Config{
				DefaultAPI:  "bedrock",
				DefaultSize: "1024x1024",
				LogLevel:    "warn",
			},
			wantErr: false,
		},
		{
			name: "valid config with grok api",
			cfg: &Config{
				DefaultAPI:  "grok",
				DefaultSize: "1024x1024",
				LogLevel:    "error",
			},
			wantErr: false,
		},
		{
			name: "valid vertex_project format",
			cfg: &Config{
				VertexProject: "my-cool-project-123",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSize(t *testing.T) {
	tests := []struct {
		name    string
		size    string
		wantErr bool
	}{
		{name: "valid 1024x1024", size: "1024x1024", wantErr: false},
		{name: "valid 512x768", size: "512x768", wantErr: false},
		{name: "valid 0x0", size: "0x0", wantErr: false},
		{name: "invalid abc", size: "abc", wantErr: true},
		{name: "invalid single number", size: "1024", wantErr: true},
		{name: "empty string", size: "", wantErr: true},
		{name: "invalid with spaces", size: "1024 x 1024", wantErr: true},
		{name: "invalid negative", size: "-1x-1", wantErr: true},
		{name: "valid large", size: "4096x4096", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSize(tt.size)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err, "need home dir for tests")

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "tilde with path",
			path:     "~/path",
			expected: filepath.Join(home, "path"),
		},
		{
			name:     "tilde only",
			path:     "~",
			expected: home,
		},
		{
			name:     "absolute path unchanged",
			path:     "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "empty string",
			path:     "",
			expected: "",
		},
		{
			name:     "relative path unchanged",
			path:     "relative/path",
			expected: "relative/path",
		},
		{
			name:     "tilde with nested path",
			path:     "~/a/b/c",
			expected: filepath.Join(home, "a/b/c"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandHome(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadConfig(t *testing.T) {
	t.Run("env var GEMINI_API_KEY loaded", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "config.md"))
		t.Setenv("GEMINI_API_KEY", "test-gemini-key-12345678")

		cfg, err := LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, "test-gemini-key-12345678", cfg.GeminiAPIKey)
	})

	t.Run("env var GROK_API_KEY loaded", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "config.md"))
		t.Setenv("GROK_API_KEY", "xai-test-grok-key-123")

		cfg, err := LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, "xai-test-grok-key-123", cfg.GrokAPIKey)
	})

	t.Run("env var AWS_REGION overrides default", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "config.md"))
		t.Setenv("AWS_REGION", "eu-west-1")

		cfg, err := LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, "eu-west-1", cfg.AWSRegion)
	})

	t.Run("defaults when no env vars set", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "config.md"))
		// Clear all relevant env vars
		t.Setenv("GEMINI_API_KEY", "")
		t.Setenv("VERTEX_API_KEY", "")
		t.Setenv("GROK_API_KEY", "")
		t.Setenv("AWS_REGION", "")
		t.Setenv("AWS_ACCESS_KEY_ID", "")
		t.Setenv("AWS_SECRET_ACCESS_KEY", "")
		t.Setenv("AWS_PROFILE", "")
		t.Setenv("AWS_BEARER_TOKEN_BEDROCK", "")
		t.Setenv("VERTEX_PROJECT", "")
		t.Setenv("VERTEX_LOCATION", "")
		t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")
		t.Setenv("GIMAGE_LOG_LEVEL", "")

		cfg, err := LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, "gemini", cfg.DefaultAPI)
		assert.Equal(t, "1024x1024", cfg.DefaultSize)
		assert.Equal(t, "us-east-1", cfg.AWSRegion)
		assert.Equal(t, "us-central1", cfg.VertexLocation)
		assert.Equal(t, "info", cfg.LogLevel)
		assert.Equal(t, "gemini-3-pro-image", cfg.DefaultModel)
	})

	t.Run("config file is loaded", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.md")
		t.Setenv("GIMAGE_CONFIG", configPath)

		// Clear env vars so config file values are visible
		t.Setenv("GEMINI_API_KEY", "")
		t.Setenv("GROK_API_KEY", "")

		content := "# Test Config\n\n**gemini_api_key**: file-gemini-key-abc123\n**grok_api_key**: file-grok-key-xyz789\n"
		err := os.WriteFile(configPath, []byte(content), 0600)
		require.NoError(t, err)

		cfg, err := LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, "file-gemini-key-abc123", cfg.GeminiAPIKey)
		assert.Equal(t, "file-grok-key-xyz789", cfg.GrokAPIKey)
	})

	t.Run("env var overrides config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.md")
		t.Setenv("GIMAGE_CONFIG", configPath)
		t.Setenv("GEMINI_API_KEY", "env-gemini-key-overrides")

		content := "**gemini_api_key**: file-gemini-key-should-lose\n"
		err := os.WriteFile(configPath, []byte(content), 0600)
		require.NoError(t, err)

		cfg, err := LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, "env-gemini-key-overrides", cfg.GeminiAPIKey)
	})

	t.Run("all env vars loaded", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "config.md"))
		t.Setenv("GEMINI_API_KEY", "gem-key")
		t.Setenv("VERTEX_API_KEY", "vtx-key")
		t.Setenv("VERTEX_PROJECT", "my-project")
		t.Setenv("VERTEX_LOCATION", "europe-west1")
		t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/tmp/creds.json")
		t.Setenv("AWS_ACCESS_KEY_ID", "AKIATEST")
		t.Setenv("AWS_SECRET_ACCESS_KEY", "secret123")
		t.Setenv("AWS_REGION", "ap-southeast-1")
		t.Setenv("AWS_PROFILE", "myprofile")
		t.Setenv("AWS_BEARER_TOKEN_BEDROCK", "bearer-tok")
		t.Setenv("GROK_API_KEY", "grok-key")
		t.Setenv("GIMAGE_LOG_LEVEL", "debug")

		cfg, err := LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, "gem-key", cfg.GeminiAPIKey)
		assert.Equal(t, "vtx-key", cfg.VertexAPIKey)
		assert.Equal(t, "my-project", cfg.VertexProject)
		assert.Equal(t, "europe-west1", cfg.VertexLocation)
		assert.Equal(t, "/tmp/creds.json", cfg.VertexCredentialsPath)
		assert.Equal(t, "AKIATEST", cfg.AWSAccessKeyID)
		assert.Equal(t, "secret123", cfg.AWSSecretAccessKey)
		assert.Equal(t, "ap-southeast-1", cfg.AWSRegion)
		assert.Equal(t, "myprofile", cfg.AWSProfile)
		assert.Equal(t, "bearer-tok", cfg.AWSBedrockAPIKey)
		assert.Equal(t, "grok-key", cfg.GrokAPIKey)
		assert.Equal(t, "debug", cfg.LogLevel)
	})
}

func TestParseMarkdownConfig(t *testing.T) {
	t.Run("valid markdown with keys", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.md")

		content := `# Gimage Configuration

**gemini_api_key**: test-key-123
**vertex_api_key**: vertex-key-456
**vertex_project**: my-project
**vertex_location**: us-central1
**aws_region**: eu-west-1
**grok_api_key**: xai-grok-key
**default_api**: vertex
**default_model**: imagen-4
**default_size**: 512x512
**log_level**: debug
`
		err := os.WriteFile(configPath, []byte(content), 0600)
		require.NoError(t, err)

		cfg := &Config{}
		err = parseMarkdownConfig(configPath, cfg)
		require.NoError(t, err)

		assert.Equal(t, "test-key-123", cfg.GeminiAPIKey)
		assert.Equal(t, "vertex-key-456", cfg.VertexAPIKey)
		assert.Equal(t, "my-project", cfg.VertexProject)
		assert.Equal(t, "us-central1", cfg.VertexLocation)
		assert.Equal(t, "eu-west-1", cfg.AWSRegion)
		assert.Equal(t, "xai-grok-key", cfg.GrokAPIKey)
		assert.Equal(t, "vertex", cfg.DefaultAPI)
		assert.Equal(t, "imagen-4", cfg.DefaultModel)
		assert.Equal(t, "512x512", cfg.DefaultSize)
		assert.Equal(t, "debug", cfg.LogLevel)
	})

	t.Run("comments and empty lines skipped", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.md")

		content := `# This is a comment
## Another comment

# More comments

**gemini_api_key**: only-this-should-parse

Some random text that doesn't match the pattern
`
		err := os.WriteFile(configPath, []byte(content), 0600)
		require.NoError(t, err)

		cfg := &Config{}
		err = parseMarkdownConfig(configPath, cfg)
		require.NoError(t, err)

		assert.Equal(t, "only-this-should-parse", cfg.GeminiAPIKey)
	})

	t.Run("invalid format lines skipped", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.md")

		content := `key: value without bold markers
gemini_api_key: no-bold
**invalid key with spaces**: should-not-parse
**gemini_api_key**: valid-key-here
`
		err := os.WriteFile(configPath, []byte(content), 0600)
		require.NoError(t, err)

		cfg := &Config{}
		err = parseMarkdownConfig(configPath, cfg)
		require.NoError(t, err)

		assert.Equal(t, "valid-key-here", cfg.GeminiAPIKey)
	})

	t.Run("all config keys parsed correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.md")

		content := `**gemini_api_key**: gem-key
**vertex_api_key**: vtx-key
**vertex_project**: my-proj
**vertex_location**: us-east1
**vertex_credentials_path**: /path/to/creds.json
**aws_access_key_id**: AKIAEXAMPLE
**aws_secret_access_key**: secret-key
**aws_region**: us-west-2
**aws_profile**: prod
**aws_bedrock_api_key**: bedrock-bearer
**grok_api_key**: grok-xai
**default_api**: bedrock
**default_model**: nova-canvas
**default_size**: 768x768
**cache_dir**: /tmp/cache
**log_level**: warn
`
		err := os.WriteFile(configPath, []byte(content), 0600)
		require.NoError(t, err)

		cfg := &Config{}
		err = parseMarkdownConfig(configPath, cfg)
		require.NoError(t, err)

		assert.Equal(t, "gem-key", cfg.GeminiAPIKey)
		assert.Equal(t, "vtx-key", cfg.VertexAPIKey)
		assert.Equal(t, "my-proj", cfg.VertexProject)
		assert.Equal(t, "us-east1", cfg.VertexLocation)
		assert.Equal(t, "/path/to/creds.json", cfg.VertexCredentialsPath)
		assert.Equal(t, "AKIAEXAMPLE", cfg.AWSAccessKeyID)
		assert.Equal(t, "secret-key", cfg.AWSSecretAccessKey)
		assert.Equal(t, "us-west-2", cfg.AWSRegion)
		assert.Equal(t, "prod", cfg.AWSProfile)
		assert.Equal(t, "bedrock-bearer", cfg.AWSBedrockAPIKey)
		assert.Equal(t, "grok-xai", cfg.GrokAPIKey)
		assert.Equal(t, "bedrock", cfg.DefaultAPI)
		assert.Equal(t, "nova-canvas", cfg.DefaultModel)
		assert.Equal(t, "768x768", cfg.DefaultSize)
		assert.Equal(t, "/tmp/cache", cfg.CacheDir)
		assert.Equal(t, "warn", cfg.LogLevel)
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		cfg := &Config{}
		err := parseMarkdownConfig("/nonexistent/path/config.md", cfg)
		assert.Error(t, err)
	})
}

func TestSaveConfig(t *testing.T) {
	t.Run("valid config saves and is readable", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "subdir", "config.md")
		t.Setenv("GIMAGE_CONFIG", configPath)

		cfg := &Config{
			GeminiAPIKey: "save-test-gemini-key",
			GrokAPIKey:   "save-test-grok-key",
			DefaultAPI:   "gemini",
			DefaultSize:  "1024x1024",
			AWSRegion:    "us-east-1",
			LogLevel:     "info",
		}

		err := SaveConfig(cfg)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(configPath)
		require.NoError(t, err)

		// Read back the file content
		data, err := os.ReadFile(configPath)
		require.NoError(t, err)
		content := string(data)

		assert.Contains(t, content, "**gemini_api_key**: save-test-gemini-key")
		assert.Contains(t, content, "**grok_api_key**: save-test-grok-key")
		assert.Contains(t, content, "**default_api**: gemini")
		assert.Contains(t, content, "**default_size**: 1024x1024")
		assert.Contains(t, content, "SECURITY WARNING")
	})

	t.Run("nil config returns error", func(t *testing.T) {
		err := SaveConfig(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config cannot be nil")
	})

	t.Run("invalid config returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "config.md"))

		cfg := &Config{
			DefaultAPI: "invalid-api",
		}

		err := SaveConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid config")
	})

	t.Run("round-trip save and load", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.md")
		t.Setenv("GIMAGE_CONFIG", configPath)

		// Clear env vars so config file values are read back
		t.Setenv("GEMINI_API_KEY", "")
		t.Setenv("GROK_API_KEY", "")
		t.Setenv("AWS_REGION", "")
		t.Setenv("AWS_ACCESS_KEY_ID", "")
		t.Setenv("AWS_SECRET_ACCESS_KEY", "")
		t.Setenv("AWS_PROFILE", "")
		t.Setenv("AWS_BEARER_TOKEN_BEDROCK", "")
		t.Setenv("VERTEX_API_KEY", "")
		t.Setenv("VERTEX_PROJECT", "")
		t.Setenv("VERTEX_LOCATION", "")
		t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")
		t.Setenv("GIMAGE_LOG_LEVEL", "")

		original := &Config{
			GeminiAPIKey: "roundtrip-gem-key",
			GrokAPIKey:   "roundtrip-grok-key",
			DefaultAPI:   "grok",
			DefaultSize:  "512x512",
			AWSRegion:    "eu-west-1",
			LogLevel:     "debug",
		}

		err := SaveConfig(original)
		require.NoError(t, err)

		loaded, err := LoadConfig()
		require.NoError(t, err)

		assert.Equal(t, original.GeminiAPIKey, loaded.GeminiAPIKey)
		assert.Equal(t, original.GrokAPIKey, loaded.GrokAPIKey)
		assert.Equal(t, original.DefaultAPI, loaded.DefaultAPI)
		assert.Equal(t, original.DefaultSize, loaded.DefaultSize)
		assert.Equal(t, original.AWSRegion, loaded.AWSRegion)
		assert.Equal(t, original.LogLevel, loaded.LogLevel)
	})
}

func TestGetConfigPath(t *testing.T) {
	t.Run("default path contains .gimage/config.md", func(t *testing.T) {
		t.Setenv("GIMAGE_CONFIG", "")
		path := GetConfigPath()
		assert.Contains(t, path, ".gimage")
		assert.Contains(t, path, "config.md")
	})

	t.Run("custom path from GIMAGE_CONFIG env", func(t *testing.T) {
		t.Setenv("GIMAGE_CONFIG", "/custom/path/config.md")
		path := GetConfigPath()
		assert.Equal(t, "/custom/path/config.md", path)
	})

	t.Run("tilde expansion in GIMAGE_CONFIG", func(t *testing.T) {
		home, err := os.UserHomeDir()
		require.NoError(t, err)

		t.Setenv("GIMAGE_CONFIG", "~/myconfig.md")
		path := GetConfigPath()
		assert.Equal(t, filepath.Join(home, "myconfig.md"), path)
	})
}
