package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apresai/gimage/pkg/models"
)

func TestGenerateOutputPath(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{
			name:   "png format",
			format: "png",
		},
		{
			name:   "jpg format",
			format: "jpg",
		},
		{
			name:   "webp format",
			format: "webp",
		},
		{
			name:   "format with dot",
			format: ".png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateOutputPath(tt.format)
			if got == "" {
				t.Error("GenerateOutputPath() returned empty string")
			}

			// Check that the path contains the expected format
			expectedFormat := normalizeFormat(tt.format)
			if !filepath.IsAbs(got) {
				// Relative path should contain the prefix
				if !strings.Contains(got, defaultOutputPrefix) {
					t.Errorf("GenerateOutputPath() = %v, doesn't contain prefix %v", got, defaultOutputPrefix)
				}
			}

			// Check extension
			ext := filepath.Ext(got)
			if ext != "."+expectedFormat {
				t.Errorf("GenerateOutputPath() extension = %v, want %v", ext, "."+expectedFormat)
			}
		})
	}
}

func TestNormalizeFormat(t *testing.T) {
	tests := []struct {
		name   string
		format string
		want   string
	}{
		{
			name:   "with dot",
			format: ".png",
			want:   "png",
		},
		{
			name:   "without dot",
			format: "jpg",
			want:   "jpeg",
		},
		{
			name:   "uppercase",
			format: "PNG",
			want:   "png",
		},
		{
			name:   "empty",
			format: "",
			want:   "png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeFormat(tt.format)
			if got != tt.want {
				t.Errorf("normalizeFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSaveImage(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		image      *models.GeneratedImage
		outputPath string
		wantErr    bool
	}{
		{
			name: "valid image",
			image: &models.GeneratedImage{
				Data:   []byte("fake image data"),
				Format: "png",
				Width:  1024,
				Height: 1024,
			},
			outputPath: filepath.Join(tmpDir, "test1.png"),
			wantErr:    false,
		},
		{
			name:       "nil image",
			image:      nil,
			outputPath: filepath.Join(tmpDir, "test2.png"),
			wantErr:    true,
		},
		{
			name: "empty output path",
			image: &models.GeneratedImage{
				Data:   []byte("fake image data"),
				Format: "png",
			},
			outputPath: "",
			wantErr:    true,
		},
		{
			name: "empty image data",
			image: &models.GeneratedImage{
				Data:   []byte{},
				Format: "png",
			},
			outputPath: filepath.Join(tmpDir, "test3.png"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SaveImage(tt.image, tt.outputPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveImage() error = %v, wantErr %v", err, tt.wantErr)
			}

			// If no error expected, check that file was created
			if !tt.wantErr && tt.outputPath != "" {
				if _, err := os.Stat(tt.outputPath); os.IsNotExist(err) {
					t.Errorf("SaveImage() did not create file at %v", tt.outputPath)
				}
			}
		})
	}
}
