package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateGeminiAPIKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty key",
			key:     "",
			wantErr: true,
			errMsg:  "empty",
		},
		{
			name:    "too short key",
			key:     "short",
			wantErr: true,
			errMsg:  "too short",
		},
		{
			name:    "valid format key",
			key:     "AIzaSyABCDEFGHIJKLMNOPQR",
			wantErr: false,
		},
		{
			name:    "key with spaces and special chars",
			key:     "key with spaces and !@#$",
			wantErr: true,
			errMsg:  "invalid characters",
		},
		{
			name:    "valid key with hyphens and underscores",
			key:     "AIzaSy_ABCDEFGH-IJKLMNOPQR",
			wantErr: false,
		},
		{
			name:    "exactly 20 characters",
			key:     "12345678901234567890",
			wantErr: false,
		},
		{
			name:    "19 characters too short",
			key:     "1234567890123456789",
			wantErr: true,
			errMsg:  "too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGeminiAPIKey(tt.key)
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

func TestValidateVertexCredentials(t *testing.T) {
	tests := []struct {
		name     string
		project  string
		location string
		credFile bool // whether to create a temp cred file
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "empty project",
			project:  "",
			location: "us-central1",
			wantErr:  true,
			errMsg:   "empty",
		},
		{
			name:     "empty location",
			project:  "my-project",
			location: "",
			wantErr:  true,
			errMsg:   "empty",
		},
		{
			name:     "invalid project format uppercase",
			project:  "INVALID",
			location: "us-central1",
			wantErr:  true,
			errMsg:   "invalid",
		},
		{
			name:     "unsupported location",
			project:  "my-project",
			location: "us-invalid",
			wantErr:  true,
			errMsg:   "unsupported",
		},
		{
			name:     "valid project and location with cred file",
			project:  "my-project",
			location: "us-central1",
			credFile: true,
			wantErr:  false,
		},
		{
			name:     "valid project no cred file env",
			project:  "my-project",
			location: "us-central1",
			credFile: false,
			wantErr:  true,
			errMsg:   "GOOGLE_APPLICATION_CREDENTIALS",
		},
		{
			name:     "project starting with number",
			project:  "123project",
			location: "us-central1",
			wantErr:  true,
			errMsg:   "invalid",
		},
		{
			name:     "single char project",
			project:  "a",
			location: "us-central1",
			wantErr:  true,
			errMsg:   "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.credFile {
				tmpDir := t.TempDir()
				credPath := filepath.Join(tmpDir, "creds.json")
				err := os.WriteFile(credPath, []byte(`{"type":"service_account"}`), 0600)
				require.NoError(t, err)
				t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credPath)
			} else {
				t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")
			}

			err := ValidateVertexCredentials(tt.project, tt.location)
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

func TestSanitizeAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "empty key",
			key:      "",
			expected: "<empty>",
		},
		{
			name:     "short key 4 chars",
			key:      "abcd",
			expected: "****",
		},
		{
			name:     "exactly 8 chars",
			key:      "abcdefgh",
			expected: "********",
		},
		{
			name:     "longer key shows first and last 4",
			key:      "abcdefghijklmnop",
			expected: "abcd********mnop",
		},
		{
			name:     "9 char key shows first and last 4",
			key:      "123456789",
			expected: "1234*6789",
		},
		{
			name:     "real-looking API key",
			key:      "AIzaSyABCDEFGHIJKLMNOPQRSTUV",
			expected: "AIza********************STUV",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeAPIKey(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// clearCredentialEnvVars sets all credential-related env vars to empty strings
// to isolate tests from the host environment.
func clearCredentialEnvVars(t *testing.T) {
	t.Helper()
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("VERTEX_API_KEY", "")
	t.Setenv("GROK_API_KEY", "")
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_BEARER_TOKEN_BEDROCK", "")
	t.Setenv("AWS_REGION", "")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")
	t.Setenv("VERTEX_PROJECT", "")
	t.Setenv("VERTEX_LOCATION", "")
	t.Setenv("GIMAGE_LOG_LEVEL", "")
}

func TestHasGeminiCredentials(t *testing.T) {
	t.Run("returns true when GEMINI_API_KEY set", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "nonexistent", "config.md"))
		t.Setenv("GEMINI_API_KEY", "test-key-present")

		assert.True(t, HasGeminiCredentials())
	})

	t.Run("returns false when not set", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "nonexistent", "config.md"))

		assert.False(t, HasGeminiCredentials())
	})

	t.Run("returns true from config file", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.md")
		t.Setenv("GIMAGE_CONFIG", configPath)

		content := "**gemini_api_key**: from-config-file\n"
		err := os.WriteFile(configPath, []byte(content), 0600)
		require.NoError(t, err)

		assert.True(t, HasGeminiCredentials())
	})
}

func TestHasVertexCredentials(t *testing.T) {
	t.Run("returns true when VERTEX_API_KEY set", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "nonexistent", "config.md"))
		t.Setenv("VERTEX_API_KEY", "vertex-key-present")

		assert.True(t, HasVertexCredentials())
	})

	t.Run("returns false when nothing set", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "nonexistent", "config.md"))

		assert.False(t, HasVertexCredentials())
	})

	t.Run("returns true when GOOGLE_APPLICATION_CREDENTIALS points to real file", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "nonexistent", "config.md"))

		credPath := filepath.Join(tmpDir, "service-account.json")
		err := os.WriteFile(credPath, []byte(`{"type":"service_account"}`), 0600)
		require.NoError(t, err)
		t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credPath)

		assert.True(t, HasVertexCredentials())
	})

	t.Run("returns false when GOOGLE_APPLICATION_CREDENTIALS points to nonexistent file", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "nonexistent", "config.md"))
		t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/nonexistent/file.json")

		assert.False(t, HasVertexCredentials())
	})
}

func TestHasGrokCredentials(t *testing.T) {
	t.Run("returns true when GROK_API_KEY set", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "nonexistent", "config.md"))
		t.Setenv("GROK_API_KEY", "xai-key-present")

		assert.True(t, HasGrokCredentials())
	})

	t.Run("returns false when not set", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "nonexistent", "config.md"))

		assert.False(t, HasGrokCredentials())
	})

	t.Run("returns true from config file", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.md")
		t.Setenv("GIMAGE_CONFIG", configPath)

		content := "**grok_api_key**: grok-from-config\n"
		err := os.WriteFile(configPath, []byte(content), 0600)
		require.NoError(t, err)

		assert.True(t, HasGrokCredentials())
	})
}

func TestHasBedrockCredentials(t *testing.T) {
	t.Run("returns true when AWS_BEARER_TOKEN_BEDROCK set", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "nonexistent", "config.md"))
		t.Setenv("AWS_BEARER_TOKEN_BEDROCK", "bearer-token")

		assert.True(t, HasBedrockCredentials())
	})

	t.Run("returns true when both AWS access keys set", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "nonexistent", "config.md"))
		t.Setenv("AWS_ACCESS_KEY_ID", "AKIATEST123")
		t.Setenv("AWS_SECRET_ACCESS_KEY", "secret123")

		assert.True(t, HasBedrockCredentials())
	})

	t.Run("returns true when AWS_PROFILE set", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "nonexistent", "config.md"))
		t.Setenv("AWS_PROFILE", "production")

		assert.True(t, HasBedrockCredentials())
	})

	t.Run("returns false with only access key ID (no secret)", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "nonexistent", "config.md"))
		// Override HOME to prevent ~/.aws/credentials from being found
		t.Setenv("HOME", tmpDir)
		t.Setenv("AWS_ACCESS_KEY_ID", "AKIATEST123")

		assert.False(t, HasBedrockCredentials())
	})
}

func TestGetGeminiAPIKey(t *testing.T) {
	t.Run("flag value returned first", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "config.md"))
		t.Setenv("GEMINI_API_KEY", "env-key-should-not-win")

		key, err := GetGeminiAPIKey("flag-key-abcdefghijklmnopqrst")
		require.NoError(t, err)
		assert.Equal(t, "flag-key-abcdefghijklmnopqrst", key)
	})

	t.Run("env var returned when no flag", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "config.md"))
		t.Setenv("GEMINI_API_KEY", "envkey-abcdefghijklmnopqrst")

		key, err := GetGeminiAPIKey("")
		require.NoError(t, err)
		assert.Equal(t, "envkey-abcdefghijklmnopqrst", key)
	})

	t.Run("error when no key available", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "nonexistent", "config.md"))

		_, err := GetGeminiAPIKey("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("invalid flag key returns error", func(t *testing.T) {
		_, err := GetGeminiAPIKey("short")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid API key")
	})
}

func TestGetGrokAPIKey(t *testing.T) {
	t.Run("flag value returned first", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "config.md"))
		t.Setenv("GROK_API_KEY", "env-grok-should-not-win")

		key, err := GetGrokAPIKey("flag-grok-key")
		require.NoError(t, err)
		assert.Equal(t, "flag-grok-key", key)
	})

	t.Run("env var returned when no flag", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "config.md"))
		t.Setenv("GROK_API_KEY", "env-grok-key-value")

		key, err := GetGrokAPIKey("")
		require.NoError(t, err)
		assert.Equal(t, "env-grok-key-value", key)
	})

	t.Run("error when no key available", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "nonexistent", "config.md"))

		_, err := GetGrokAPIKey("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("config file key returned when no flag or env", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.md")
		t.Setenv("GIMAGE_CONFIG", configPath)

		content := "**grok_api_key**: config-grok-key\n"
		err := os.WriteFile(configPath, []byte(content), 0600)
		require.NoError(t, err)

		key, err := GetGrokAPIKey("")
		require.NoError(t, err)
		assert.Equal(t, "config-grok-key", key)
	})
}

func TestGetAWSRegion(t *testing.T) {
	t.Run("flag value returned first", func(t *testing.T) {
		clearCredentialEnvVars(t)
		t.Setenv("AWS_REGION", "ap-southeast-1")

		region := GetAWSRegion("eu-west-1")
		assert.Equal(t, "eu-west-1", region)
	})

	t.Run("env var returned when no flag", func(t *testing.T) {
		clearCredentialEnvVars(t)
		t.Setenv("AWS_REGION", "ap-southeast-1")

		region := GetAWSRegion("")
		assert.Equal(t, "ap-southeast-1", region)
	})

	t.Run("default us-east-1 when nothing set", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		t.Setenv("GIMAGE_CONFIG", filepath.Join(tmpDir, "nonexistent", "config.md"))

		region := GetAWSRegion("")
		assert.Equal(t, "us-east-1", region)
	})

	t.Run("config file region used when no flag or env", func(t *testing.T) {
		clearCredentialEnvVars(t)
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.md")
		t.Setenv("GIMAGE_CONFIG", configPath)

		content := "**aws_region**: ap-northeast-1\n"
		err := os.WriteFile(configPath, []byte(content), 0600)
		require.NoError(t, err)

		region := GetAWSRegion("")
		assert.Equal(t, "ap-northeast-1", region)
	})
}
