package imaging

import (
	"context"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/disintegration/imaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestJPEG creates a test JPEG image with non-uniform pixels for meaningful compression.
func createTestJPEG(t *testing.T, path string, width, height int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with a gradient so compression differences are meaningful
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x % 256),
				G: uint8(y % 256),
				B: uint8((x + y) % 256),
				A: 255,
			})
		}
	}
	err := imaging.Save(img, path)
	require.NoError(t, err, "failed to create test JPEG")
}

func TestCompressImage(t *testing.T) {
	tests := []struct {
		name    string
		quality int
		outExt  string
	}{
		{
			name:    "JPEG quality 85",
			quality: 85,
			outExt:  ".jpg",
		},
		{
			name:    "JPEG quality 100",
			quality: 100,
			outExt:  ".jpg",
		},
		{
			name:    "JPEG quality 50",
			quality: 50,
			outExt:  ".jpg",
		},
		{
			name:    "PNG output - quality ignored",
			quality: 85,
			outExt:  ".png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			inputPath := filepath.Join(tmpDir, "input.jpg")
			outputPath := filepath.Join(tmpDir, "output"+tt.outExt)

			createTestJPEG(t, inputPath, 800, 600)

			err := CompressImage(context.Background(), inputPath, outputPath, tt.quality)
			assert.NoError(t, err)

			// Verify output file exists
			info, err := os.Stat(outputPath)
			assert.NoError(t, err, "output file should exist")
			assert.Greater(t, info.Size(), int64(0), "output file should not be empty")
		})
	}
}

func TestCompressImage_QualityAffectsSize(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "input.jpg")
	outputHigh := filepath.Join(tmpDir, "high.jpg")
	outputLow := filepath.Join(tmpDir, "low.jpg")

	createTestJPEG(t, inputPath, 800, 600)

	err := CompressImage(context.Background(), inputPath, outputHigh, 100)
	require.NoError(t, err)

	err = CompressImage(context.Background(), inputPath, outputLow, 50)
	require.NoError(t, err)

	highInfo, err := os.Stat(outputHigh)
	require.NoError(t, err)
	lowInfo, err := os.Stat(outputLow)
	require.NoError(t, err)

	assert.Greater(t, highInfo.Size(), lowInfo.Size(), "quality 100 should produce larger file than quality 50")
}

func TestCompressImage_InvalidQuality(t *testing.T) {
	tests := []struct {
		name    string
		quality int
		errMsg  string
	}{
		{
			name:    "quality 0",
			quality: 0,
			errMsg:  "must be between 1 and 100",
		},
		{
			name:    "quality -1",
			quality: -1,
			errMsg:  "must be between 1 and 100",
		},
		{
			name:    "quality 101",
			quality: 101,
			errMsg:  "must be between 1 and 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			inputPath := filepath.Join(tmpDir, "input.jpg")
			outputPath := filepath.Join(tmpDir, "output.jpg")

			createTestJPEG(t, inputPath, 100, 100)

			err := CompressImage(context.Background(), inputPath, outputPath, tt.quality)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestCompressImage_InvalidInput(t *testing.T) {
	tmpDir := t.TempDir()
	nonexistentPath := filepath.Join(tmpDir, "nonexistent.jpg")
	outputPath := filepath.Join(tmpDir, "output.jpg")

	err := CompressImage(context.Background(), nonexistentPath, outputPath, 85)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open")
}
