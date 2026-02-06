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

func TestGenerateOutputPathWithPrefix(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		format string
	}{
		{
			name:   "custom prefix",
			prefix: "myimage",
			format: "png",
		},
		{
			name:   "empty prefix uses default",
			prefix: "",
			format: "jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateOutputPathWithPrefix(tt.prefix, tt.format)
			if got == "" {
				t.Error("GenerateOutputPathWithPrefix() returned empty string")
			}

			expectedPrefix := tt.prefix
			if expectedPrefix == "" {
				expectedPrefix = defaultOutputPrefix
			}

			if !strings.Contains(got, expectedPrefix) {
				t.Errorf("GenerateOutputPathWithPrefix() = %v, doesn't contain prefix %v", got, expectedPrefix)
			}
		})
	}
}

func TestGenerateOutputPathInDir(t *testing.T) {
	tests := []struct {
		name   string
		dir    string
		format string
	}{
		{
			name:   "custom directory",
			dir:    "/tmp/images",
			format: "png",
		},
		{
			name:   "empty directory uses default",
			dir:    "",
			format: "jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateOutputPathInDir(tt.dir, tt.format)
			if got == "" {
				t.Error("GenerateOutputPathInDir() returned empty string")
			}

			expectedDir := tt.dir
			if expectedDir == "" {
				expectedDir = defaultOutputDir
			}

			dir := filepath.Dir(got)
			if expectedDir != defaultOutputDir && dir != expectedDir {
				t.Errorf("GenerateOutputPathInDir() dir = %v, want %v", dir, expectedDir)
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

func TestSaveImageWithMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	img := &models.GeneratedImage{
		Data:   []byte("fake image data"),
		Format: "png",
		Width:  1024,
		Height: 1024,
		Metadata: map[string]string{
			"model":  "gemini-2.5-flash-image",
			"prompt": "test prompt",
		},
	}

	outputPath := filepath.Join(tmpDir, "test_with_metadata.png")

	err := SaveImageWithMetadata(img, outputPath)
	if err != nil {
		t.Errorf("SaveImageWithMetadata() error = %v", err)
	}

	// Check that both files exist
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("SaveImageWithMetadata() did not create image file")
	}

	metadataPath := outputPath + ".json"
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		t.Errorf("SaveImageWithMetadata() did not create metadata file")
	}
}

func TestValidateOutputPath(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid new file",
			path:    filepath.Join(tmpDir, "new_file.png"),
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOutputPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOutputPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	existingFile := filepath.Join(tmpDir, "existing.txt")
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "existing file",
			path: existingFile,
			want: true,
		},
		{
			name: "non-existing file",
			path: filepath.Join(tmpDir, "nonexistent.txt"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FileExists(tt.path)
			if got != tt.want {
				t.Errorf("FileExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateUniqueOutputPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to tmp dir temporarily
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// First call should generate a unique path
	path1 := GenerateUniqueOutputPath("png")
	if path1 == "" {
		t.Error("GenerateUniqueOutputPath() returned empty string")
	}

	// Create the file to simulate conflict
	if err := os.WriteFile(path1, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Second call should generate a different path
	path2 := GenerateUniqueOutputPath("png")
	if path2 == path1 {
		t.Error("GenerateUniqueOutputPath() returned same path twice")
	}
}

func TestEnsureOutputDir(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		dir     string
		wantErr bool
	}{
		{
			name:    "create new directory",
			dir:     filepath.Join(tmpDir, "newdir"),
			wantErr: false,
		},
		{
			name:    "current directory",
			dir:     ".",
			wantErr: false,
		},
		{
			name:    "empty directory",
			dir:     "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnsureOutputDir(tt.dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnsureOutputDir() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Check directory exists if not special cases
			if tt.dir != "" && tt.dir != "." && tt.dir != "/" && !tt.wantErr {
				if _, err := os.Stat(tt.dir); os.IsNotExist(err) {
					t.Errorf("EnsureOutputDir() did not create directory %v", tt.dir)
				}
			}
		})
	}
}
