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

// setupTestImage creates a test image for testing
func setupTestImage(t *testing.T, width, height int) string {
	t.Helper()

	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.png")

	// Create a simple colored image
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Save it
	err := imaging.Save(img, testFile)
	require.NoError(t, err, "failed to create test image")

	return testFile
}

func TestCropImage(t *testing.T) {
	tests := []struct {
		name      string
		imgWidth  int
		imgHeight int
		x         int
		y         int
		width     int
		height    int
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid crop from top-left",
			imgWidth:  800,
			imgHeight: 600,
			x:         0,
			y:         0,
			width:     400,
			height:    300,
			wantErr:   false,
		},
		{
			name:      "valid crop from middle",
			imgWidth:  800,
			imgHeight: 600,
			x:         200,
			y:         150,
			width:     400,
			height:    300,
			wantErr:   false,
		},
		{
			name:      "crop at edge - exact fit",
			imgWidth:  800,
			imgHeight: 600,
			x:         400,
			y:         300,
			width:     400,
			height:    300,
			wantErr:   false,
		},
		{
			name:      "small crop region",
			imgWidth:  800,
			imgHeight: 600,
			x:         100,
			y:         100,
			width:     10,
			height:    10,
			wantErr:   false,
		},
		{
			name:      "negative x coordinate",
			imgWidth:  800,
			imgHeight: 600,
			x:         -10,
			y:         0,
			width:     100,
			height:    100,
			wantErr:   true,
			errMsg:    "x coordinate must be non-negative",
		},
		{
			name:      "negative y coordinate",
			imgWidth:  800,
			imgHeight: 600,
			x:         0,
			y:         -10,
			width:     100,
			height:    100,
			wantErr:   true,
			errMsg:    "y coordinate must be non-negative",
		},
		{
			name:      "zero width",
			imgWidth:  800,
			imgHeight: 600,
			x:         0,
			y:         0,
			width:     0,
			height:    100,
			wantErr:   true,
			errMsg:    "width must be positive",
		},
		{
			name:      "zero height",
			imgWidth:  800,
			imgHeight: 600,
			x:         0,
			y:         0,
			width:     100,
			height:    0,
			wantErr:   true,
			errMsg:    "height must be positive",
		},
		{
			name:      "negative width",
			imgWidth:  800,
			imgHeight: 600,
			x:         0,
			y:         0,
			width:     -100,
			height:    100,
			wantErr:   true,
			errMsg:    "width must be positive",
		},
		{
			name:      "negative height",
			imgWidth:  800,
			imgHeight: 600,
			x:         0,
			y:         0,
			width:     100,
			height:    -100,
			wantErr:   true,
			errMsg:    "height must be positive",
		},
		{
			name:      "x coordinate outside image",
			imgWidth:  800,
			imgHeight: 600,
			x:         800,
			y:         0,
			width:     100,
			height:    100,
			wantErr:   true,
			errMsg:    "x coordinate 800 is outside image width",
		},
		{
			name:      "y coordinate outside image",
			imgWidth:  800,
			imgHeight: 600,
			x:         0,
			y:         600,
			width:     100,
			height:    100,
			wantErr:   true,
			errMsg:    "y coordinate 600 is outside image height",
		},
		{
			name:      "crop width exceeds image bounds",
			imgWidth:  800,
			imgHeight: 600,
			x:         500,
			y:         0,
			width:     400,
			height:    100,
			wantErr:   true,
			errMsg:    "crop region (x=500 + width=400 = 900) exceeds image width",
		},
		{
			name:      "crop height exceeds image bounds",
			imgWidth:  800,
			imgHeight: 600,
			x:         0,
			y:         400,
			width:     100,
			height:    300,
			wantErr:   true,
			errMsg:    "crop region (y=400 + height=300 = 700) exceeds image height",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test image
			inputPath := setupTestImage(t, tt.imgWidth, tt.imgHeight)
			outputPath := filepath.Join(t.TempDir(), "output.png")

			// Perform crop
			err := CropImage(context.Background(), inputPath, outputPath, tt.x, tt.y, tt.width, tt.height)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			// Should succeed
			require.NoError(t, err)

			// Verify output file exists
			_, err = os.Stat(outputPath)
			assert.NoError(t, err, "output file should exist")

			// Load and verify cropped image dimensions
			croppedImg, err := imaging.Open(outputPath)
			require.NoError(t, err)

			bounds := croppedImg.Bounds()
			assert.Equal(t, tt.width, bounds.Dx(), "cropped width should match")
			assert.Equal(t, tt.height, bounds.Dy(), "cropped height should match")
		})
	}
}

func TestCropImage_InvalidInputFile(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "output.png")
	err := CropImage(context.Background(), "nonexistent.png", outputPath, 0, 0, 100, 100)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open image")
}

// TestCropImage_RealFixture tests with actual fixture images if they exist
func TestCropImage_RealFixture(t *testing.T) {
	fixtureDir := "../../test/fixtures"
	fixturePath := filepath.Join(fixtureDir, "test_image.png")

	// Skip if fixture doesn't exist
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skip("Test fixture not found, skipping")
	}

	outputPath := filepath.Join(t.TempDir(), "cropped_fixture.png")

	// Crop a region from the fixture
	err := CropImage(context.Background(), fixturePath, outputPath, 100, 100, 200, 150)
	require.NoError(t, err)

	// Verify output
	croppedImg, err := imaging.Open(outputPath)
	require.NoError(t, err)

	bounds := croppedImg.Bounds()
	assert.Equal(t, 200, bounds.Dx())
	assert.Equal(t, 150, bounds.Dy())
}
