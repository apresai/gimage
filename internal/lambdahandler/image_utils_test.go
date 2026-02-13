package lambdahandler

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeImageToBase64(t *testing.T) {
	t.Run("known bytes produce expected base64", func(t *testing.T) {
		input := []byte("hello world")
		expected := base64.StdEncoding.EncodeToString(input)
		assert.Equal(t, expected, EncodeImageToBase64(input))
	})

	t.Run("empty bytes produce empty string", func(t *testing.T) {
		assert.Equal(t, "", EncodeImageToBase64([]byte{}))
	})
}

func TestDecodeBase64ToImage(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  []byte
		wantError bool
	}{
		{
			name:     "valid base64",
			input:    base64.StdEncoding.EncodeToString([]byte("test image data")),
			expected: []byte("test image data"),
		},
		{
			name:      "invalid base64",
			input:     "not-valid-base64!!!",
			wantError: true,
		},
		{
			name:     "data URL format",
			input:    "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("png data")),
			expected: []byte("png data"),
		},
		{
			name:      "invalid data URL no comma",
			input:     "data:image/png;base64",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DecodeBase64ToImage(tt.input)
			if tt.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsBase64Input(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "data URL prefix",
			input:    "data:image/png;base64,abc123",
			expected: true,
		},
		{
			name:     "S3 key short path",
			input:    "images/test.png",
			expected: false,
		},
		{
			name:     "long base64 string",
			input:    strings.Repeat("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/", 5),
			expected: true,
		},
		{
			name:     "path with slashes is short",
			input:    "images/path/with/slashes.png",
			expected: false,
		},
		{
			name:     "short string",
			input:    "abc",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsBase64Input(tt.input))
		})
	}
}

func TestDetermineResponseFormat(t *testing.T) {
	tests := []struct {
		name      string
		size      int64
		format    string
		envKey    string
		envVal    string
		expected  string
	}{
		{
			name:     "explicit base64",
			size:     100,
			format:   "base64",
			expected: "base64",
		},
		{
			name:     "explicit s3_url",
			size:     100,
			format:   "s3_url",
			expected: "s3_url",
		},
		{
			name:     "small image auto selects base64",
			size:     100,
			format:   "",
			expected: "base64",
		},
		{
			name:     "large image auto selects s3_url",
			size:     1_000_000,
			format:   "",
			expected: "s3_url",
		},
		{
			name:     "custom max size via env",
			size:     200,
			format:   "",
			envKey:   "MAX_RESPONSE_SIZE_KB",
			envVal:   "0",
			expected: "s3_url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envKey != "" {
				t.Setenv(tt.envKey, tt.envVal)
			}
			assert.Equal(t, tt.expected, DetermineResponseFormat(tt.size, tt.format))
		})
	}
}

func TestGenerateS3Key(t *testing.T) {
	t.Run("png key format", func(t *testing.T) {
		key := GenerateS3Key("png")
		assert.Contains(t, key, "images/")
		assert.Contains(t, key, ".png")
		assert.NotEmpty(t, key)
	})

	t.Run("jpg key format", func(t *testing.T) {
		key := GenerateS3Key("jpg")
		assert.Contains(t, key, ".jpg")
	})

	t.Run("unique keys", func(t *testing.T) {
		key1 := GenerateS3Key("png")
		key2 := GenerateS3Key("png")
		assert.NotEqual(t, key1, key2)
	})
}

func TestGetContentType(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"png", "image/png"},
		{"jpg", "image/jpeg"},
		{"jpeg", "image/jpeg"},
		{"gif", "image/gif"},
		{"webp", "image/webp"},
		{"tiff", "image/tiff"},
		{"bmp", "image/bmp"},
		{".PNG", "image/png"},
		{"unknown", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetContentType(tt.format))
		})
	}
}

func TestGetPresignedURLExpiration(t *testing.T) {
	t.Run("default 60 minutes", func(t *testing.T) {
		assert.Equal(t, 60*time.Minute, GetPresignedURLExpiration())
	})

	t.Run("custom via env", func(t *testing.T) {
		t.Setenv("PRESIGNED_URL_EXPIRATION_MINUTES", "120")
		assert.Equal(t, 120*time.Minute, GetPresignedURLExpiration())
	})
}
