package imaging

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/disintegration/imaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertImageFile(t *testing.T) {
	tests := []struct {
		name   string
		inExt  string
		outExt string
	}{
		{
			name:   "PNG to JPEG",
			inExt:  ".png",
			outExt: ".jpg",
		},
		{
			name:   "PNG to BMP",
			inExt:  ".png",
			outExt: ".bmp",
		},
		{
			name:   "JPEG to PNG",
			inExt:  ".jpg",
			outExt: ".png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			inputPath := filepath.Join(tmpDir, "input"+tt.inExt)
			outputPath := filepath.Join(tmpDir, "output"+tt.outExt)

			// Create a simple test image
			img := image.NewRGBA(image.Rect(0, 0, 200, 150))
			err := imaging.Save(img, inputPath)
			require.NoError(t, err, "failed to create test image")

			// Test convert
			err = ConvertImageFile(context.Background(), inputPath, outputPath)
			assert.NoError(t, err)

			// Verify output file exists and is readable
			_, err = os.Stat(outputPath)
			assert.NoError(t, err, "output file should exist")

			openedImg, err := imaging.Open(outputPath)
			assert.NoError(t, err, "output should be a valid image")
			assert.NotNil(t, openedImg)
		})
	}
}

func TestConvertImageFile_InvalidInput(t *testing.T) {
	tmpDir := t.TempDir()
	nonexistentPath := filepath.Join(tmpDir, "nonexistent.png")
	outputPath := filepath.Join(tmpDir, "output.jpg")

	err := ConvertImageFile(context.Background(), nonexistentPath, outputPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open")
}

func TestConvertImageData(t *testing.T) {
	tests := []struct {
		name         string
		createData   func(t *testing.T) []byte
		targetFormat string
		wantErr      bool
		errMsg       string
	}{
		{
			name: "PNG to JPEG",
			createData: func(t *testing.T) []byte {
				t.Helper()
				img := image.NewRGBA(image.Rect(0, 0, 100, 100))
				var buf bytes.Buffer
				err := png.Encode(&buf, img)
				require.NoError(t, err)
				return buf.Bytes()
			},
			targetFormat: "jpeg",
			wantErr:      false,
		},
		{
			name: "empty data",
			createData: func(t *testing.T) []byte {
				return []byte{}
			},
			targetFormat: "jpeg",
			wantErr:      true,
			errMsg:       "input data is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.createData(t)

			result, err := ConvertImageData(data, tt.targetFormat)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, result, "converted data should not be empty")
		})
	}
}

func TestExtractFormatFromPath(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		expect string
	}{
		{name: "png extension", path: "image.png", expect: "png"},
		{name: "jpg normalized to jpeg", path: "image.jpg", expect: "jpeg"},
		{name: "jpeg stays jpeg", path: "image.jpeg", expect: "jpeg"},
		{name: "webp extension", path: "image.webp", expect: "webp"},
		{name: "gif extension", path: "image.gif", expect: "gif"},
		{name: "no extension defaults to png", path: "noext", expect: "png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractFormatFromPath(tt.path)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestNormalizeFormat(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{name: "jpg to jpeg", input: "jpg", expect: "jpeg"},
		{name: "jpeg stays jpeg", input: "jpeg", expect: "jpeg"},
		{name: "tif to tiff", input: "tif", expect: "tiff"},
		{name: "png stays png", input: "png", expect: "png"},
		{name: "dot-uppercase PNG", input: ".PNG", expect: "png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeFormat(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestSupportsTransparency(t *testing.T) {
	tests := []struct {
		name   string
		format string
		expect bool
	}{
		{name: "png supports transparency", format: "png", expect: true},
		{name: "gif supports transparency", format: "gif", expect: true},
		{name: "webp supports transparency", format: "webp", expect: true},
		{name: "jpeg does not support transparency", format: "jpeg", expect: false},
		{name: "bmp does not support transparency", format: "bmp", expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := supportsTransparency(tt.format)
			assert.Equal(t, tt.expect, result)
		})
	}
}
