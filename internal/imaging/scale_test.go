package imaging

import (
	"context"
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/disintegration/imaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScaleImage(t *testing.T) {
	tests := []struct {
		name         string
		imgWidth     int
		imgHeight    int
		factor       float64
		expectWidth  int
		expectHeight int
	}{
		{
			name:         "scale down by half",
			imgWidth:     800,
			imgHeight:    600,
			factor:       0.5,
			expectWidth:  400,
			expectHeight: 300,
		},
		{
			name:         "scale up by double",
			imgWidth:     400,
			imgHeight:    300,
			factor:       2.0,
			expectWidth:  800,
			expectHeight: 600,
		},
		{
			name:         "scale by 1.0 - no change",
			imgWidth:     800,
			imgHeight:    600,
			factor:       1.0,
			expectWidth:  800,
			expectHeight: 600,
		},
		{
			name:         "scale down to very small",
			imgWidth:     800,
			imgHeight:    600,
			factor:       0.1,
			expectWidth:  80,
			expectHeight: 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			inputPath := filepath.Join(tmpDir, "input.png")
			outputPath := filepath.Join(tmpDir, "output.png")

			// Create a simple test image
			img := image.NewRGBA(image.Rect(0, 0, tt.imgWidth, tt.imgHeight))
			err := imaging.Save(img, inputPath)
			require.NoError(t, err, "failed to create test image")

			// Test scale
			err = ScaleImage(context.Background(), inputPath, outputPath, tt.factor)
			assert.NoError(t, err)

			// Verify output file exists
			_, err = os.Stat(outputPath)
			assert.NoError(t, err, "output file should exist")

			// Verify dimensions
			scaledImg, err := imaging.Open(outputPath)
			require.NoError(t, err, "failed to open scaled image")

			bounds := scaledImg.Bounds()
			assert.Equal(t, tt.expectWidth, bounds.Dx(), "width mismatch")
			assert.Equal(t, tt.expectHeight, bounds.Dy(), "height mismatch")
		})
	}
}

func TestScaleImage_InvalidFactor(t *testing.T) {
	tests := []struct {
		name   string
		factor float64
		errMsg string
	}{
		{
			name:   "zero factor",
			factor: 0,
			errMsg: "factor must be positive",
		},
		{
			name:   "negative factor",
			factor: -1.0,
			errMsg: "factor must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			inputPath := filepath.Join(tmpDir, "input.png")
			outputPath := filepath.Join(tmpDir, "output.png")

			// Create a simple test image
			img := image.NewRGBA(image.Rect(0, 0, 100, 100))
			err := imaging.Save(img, inputPath)
			require.NoError(t, err, "failed to create test image")

			err = ScaleImage(context.Background(), inputPath, outputPath, tt.factor)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestScaleImage_InvalidInput(t *testing.T) {
	tmpDir := t.TempDir()
	nonexistentPath := filepath.Join(tmpDir, "nonexistent.png")
	outputPath := filepath.Join(tmpDir, "output.png")

	err := ScaleImage(context.Background(), nonexistentPath, outputPath, 1.0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open")
}
