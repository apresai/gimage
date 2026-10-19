package imaging

import (
	"context"
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResizeImage(t *testing.T) {
	tests := []struct {
		name      string
		imgWidth  int
		imgHeight int
		newWidth  int
		newHeight int
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid resize - smaller",
			imgWidth:  800,
			imgHeight: 600,
			newWidth:  400,
			newHeight: 300,
			wantErr:   false,
		},
		{
			name:      "valid resize - larger",
			imgWidth:  400,
			imgHeight: 300,
			newWidth:  800,
			newHeight: 600,
			wantErr:   false,
		},
		{
			name:      "resize to square",
			imgWidth:  800,
			imgHeight: 600,
			newWidth:  500,
			newHeight: 500,
			wantErr:   false,
		},
		{
			name:      "resize to same dimensions",
			imgWidth:  800,
			imgHeight: 600,
			newWidth:  800,
			newHeight: 600,
			wantErr:   false,
		},
		{
			name:      "resize to very small",
			imgWidth:  800,
			imgHeight: 600,
			newWidth:  10,
			newHeight: 10,
			wantErr:   false,
		},
		{
			name:      "resize to very large",
			imgWidth:  100,
			imgHeight: 100,
			newWidth:  2000,
			newHeight: 2000,
			wantErr:   false,
		},
		{
			name:      "zero width",
			imgWidth:  800,
			imgHeight: 600,
			newWidth:  0,
			newHeight: 300,
			wantErr:   true,
			errMsg:    "width must be positive",
		},
		{
			name:      "zero height",
			imgWidth:  800,
			imgHeight: 600,
			newWidth:  400,
			newHeight: 0,
			wantErr:   true,
			errMsg:    "height must be positive",
		},
		{
			name:      "negative width",
			imgWidth:  800,
			imgHeight: 600,
			newWidth:  -400,
			newHeight: 300,
			wantErr:   true,
			errMsg:    "width must be positive",
		},
		{
			name:      "negative height",
			imgWidth:  800,
			imgHeight: 600,
			newWidth:  400,
			newHeight: -300,
			wantErr:   true,
			errMsg:    "height must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test image
			tmpDir := t.TempDir()
			inputPath := filepath.Join(tmpDir, "input.png")
			outputPath := filepath.Join(tmpDir, "output.png")

			// Create a simple test image
			img := image.NewRGBA(image.Rect(0, 0, tt.imgWidth, tt.imgHeight))
			err := Save(img, inputPath)
			require.NoError(t, err, "failed to create test image")

			// Test resize
			err = ResizeImage(context.Background(), inputPath, outputPath, tt.newWidth, tt.newHeight)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			assert.NoError(t, err)

			// Verify output file exists
			_, err = os.Stat(outputPath)
			assert.NoError(t, err, "output file should exist")

			// Verify dimensions
			resizedImg, err := Open(outputPath)
			require.NoError(t, err, "failed to open resized image")

			bounds := resizedImg.Bounds()
			actualWidth := bounds.Dx()
			actualHeight := bounds.Dy()

			assert.Equal(t, tt.newWidth, actualWidth, "width mismatch")
			assert.Equal(t, tt.newHeight, actualHeight, "height mismatch")
		})
	}
}

func TestResizeImage_InvalidInputFile(t *testing.T) {
	tmpDir := t.TempDir()
	nonexistentPath := filepath.Join(tmpDir, "nonexistent.png")
	outputPath := filepath.Join(tmpDir, "output.png")

	err := ResizeImage(context.Background(), nonexistentPath, outputPath, 100, 100)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open image")
}

func TestResizeFit(t *testing.T) {
	tests := []struct {
		name         string
		imgWidth     int
		imgHeight    int
		maxWidth     int
		maxHeight    int
		expectWidth  int
		expectHeight int
		wantErr      bool
		errMsg       string
	}{
		{
			name:         "landscape - width limited",
			imgWidth:     800,
			imgHeight:    600,
			maxWidth:     400,
			maxHeight:    400,
			expectWidth:  400,
			expectHeight: 300,
			wantErr:      false,
		},
		{
			name:         "portrait - height limited",
			imgWidth:     600,
			imgHeight:    800,
			maxWidth:     400,
			maxHeight:    400,
			expectWidth:  300,
			expectHeight: 400,
			wantErr:      false,
		},
		{
			name:         "square - fits exactly",
			imgWidth:     500,
			imgHeight:    500,
			maxWidth:     500,
			maxHeight:    500,
			expectWidth:  500,
			expectHeight: 500,
			wantErr:      false,
		},
		{
			name:         "already smaller - no resize",
			imgWidth:     200,
			imgHeight:    150,
			maxWidth:     400,
			maxHeight:    400,
			expectWidth:  200,
			expectHeight: 150,
			wantErr:      false,
		},
		{
			name:         "wide image - width limited",
			imgWidth:     1920,
			imgHeight:    1080,
			maxWidth:     800,
			maxHeight:    600,
			expectWidth:  800,
			expectHeight: 450,
			wantErr:      false,
		},
		{
			name:      "zero max width",
			imgWidth:  800,
			imgHeight: 600,
			maxWidth:  0,
			maxHeight: 300,
			wantErr:   true,
			errMsg:    "width must be positive",
		},
		{
			name:      "zero max height",
			imgWidth:  800,
			imgHeight: 600,
			maxWidth:  400,
			maxHeight: 0,
			wantErr:   true,
			errMsg:    "height must be positive",
		},
		{
			name:      "negative max width",
			imgWidth:  800,
			imgHeight: 600,
			maxWidth:  -400,
			maxHeight: 300,
			wantErr:   true,
			errMsg:    "width must be positive",
		},
		{
			name:      "negative max height",
			imgWidth:  800,
			imgHeight: 600,
			maxWidth:  400,
			maxHeight: -300,
			wantErr:   true,
			errMsg:    "height must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test image
			tmpDir := t.TempDir()
			inputPath := filepath.Join(tmpDir, "input.png")
			outputPath := filepath.Join(tmpDir, "output.png")

			// Create a simple test image
			img := image.NewRGBA(image.Rect(0, 0, tt.imgWidth, tt.imgHeight))
			err := Save(img, inputPath)
			require.NoError(t, err, "failed to create test image")

			// Test resize fit
			err = ResizeFit(context.Background(), inputPath, outputPath, tt.maxWidth, tt.maxHeight)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			assert.NoError(t, err)

			// Verify output file exists
			_, err = os.Stat(outputPath)
			assert.NoError(t, err, "output file should exist")

			// Verify dimensions
			resizedImg, err := Open(outputPath)
			require.NoError(t, err, "failed to open resized image")

			bounds := resizedImg.Bounds()
			actualWidth := bounds.Dx()
			actualHeight := bounds.Dy()

			assert.Equal(t, tt.expectWidth, actualWidth, "width mismatch")
			assert.Equal(t, tt.expectHeight, actualHeight, "height mismatch")

			// Verify it fits within bounds
			assert.LessOrEqual(t, actualWidth, tt.maxWidth, "width exceeds max")
			assert.LessOrEqual(t, actualHeight, tt.maxHeight, "height exceeds max")
		})
	}
}

func TestResizeFit_InvalidInputFile(t *testing.T) {
	tmpDir := t.TempDir()
	nonexistentPath := filepath.Join(tmpDir, "nonexistent.png")
	outputPath := filepath.Join(tmpDir, "output.png")

	err := ResizeFit(context.Background(), nonexistentPath, outputPath, 100, 100)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open image")
}

func TestResizeFit_RealFixture(t *testing.T) {
	// Use real test fixture if available
	fixturePath := filepath.Join("..", "..", "test", "fixtures", "sample.png")
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skip("Test fixture not found, skipping")
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "resized_fit.png")

	err := ResizeFit(context.Background(), fixturePath, outputPath, 400, 400)
	assert.NoError(t, err)

	// Verify output exists
	_, err = os.Stat(outputPath)
	assert.NoError(t, err)

	// Verify dimensions are within bounds
	img, err := Open(outputPath)
	require.NoError(t, err)

	bounds := img.Bounds()
	assert.LessOrEqual(t, bounds.Dx(), 400)
	assert.LessOrEqual(t, bounds.Dy(), 400)
}
