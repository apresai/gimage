package tools

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateOutputPath(t *testing.T) {
	tests := []struct {
		name       string
		inputPath  string
		suffix     string
		wantSuffix string
	}{
		{
			name:       "simple filename",
			inputPath:  "photo.jpg",
			suffix:     "resized",
			wantSuffix: "_resized.jpg",
		},
		{
			name:       "path with directory",
			inputPath:  "/path/to/photo.png",
			suffix:     "cropped",
			wantSuffix: "_cropped.png",
		},
		{
			name:       "no extension",
			inputPath:  "image",
			suffix:     "scaled",
			wantSuffix: "_scaled",
		},
		{
			name:       "multiple dots",
			inputPath:  "my.photo.jpeg",
			suffix:     "compressed",
			wantSuffix: "_compressed.jpeg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateOutputPath(tt.inputPath, tt.suffix)

			resultBase := filepath.Base(result)
			if !contains(resultBase, tt.wantSuffix) {
				t.Errorf("Expected path to contain suffix %s, got: %s", tt.wantSuffix, result)
			}
		})
	}
}

func TestCoerceToInt(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		wantValue int
		wantError bool
	}{
		// float64 (JSON default)
		{name: "float64 positive", value: 100.0, wantValue: 100, wantError: false},
		{name: "float64 with decimal", value: 100.7, wantValue: 100, wantError: false},
		{name: "float64 zero", value: 0.0, wantValue: 0, wantError: false},
		{name: "float64 negative", value: -50.0, wantValue: -50, wantError: false},

		// string (LLM common behavior)
		{name: "string integer", value: "100", wantValue: 100, wantError: false},
		{name: "string zero", value: "0", wantValue: 0, wantError: false},
		{name: "string negative", value: "-50", wantValue: -50, wantError: false},
		{name: "string float", value: "100.7", wantValue: 100, wantError: false},
		{name: "string invalid", value: "abc", wantValue: 0, wantError: true},
		{name: "string empty", value: "", wantValue: 0, wantError: true},

		// int
		{name: "int positive", value: int(100), wantValue: 100, wantError: false},
		{name: "int zero", value: int(0), wantValue: 0, wantError: false},

		// int64
		{name: "int64 positive", value: int64(100), wantValue: 100, wantError: false},

		// nil
		{name: "nil value", value: nil, wantValue: 0, wantError: true},

		// invalid types
		{name: "bool type", value: true, wantValue: 0, wantError: true},
		{name: "slice type", value: []int{1, 2, 3}, wantValue: 0, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := coerceToInt(tt.value, "test")

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.wantValue {
					t.Errorf("Expected %d, got %d", tt.wantValue, result)
				}
			}
		})
	}
}

func TestCoerceToFloat(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		wantValue float64
		wantError bool
	}{
		// float64 (JSON default)
		{name: "float64 positive", value: 0.5, wantValue: 0.5, wantError: false},
		{name: "float64 integer", value: 2.0, wantValue: 2.0, wantError: false},
		{name: "float64 zero", value: 0.0, wantValue: 0.0, wantError: false},

		// string (LLM common behavior)
		{name: "string float", value: "0.5", wantValue: 0.5, wantError: false},
		{name: "string integer", value: "2", wantValue: 2.0, wantError: false},
		{name: "string zero", value: "0", wantValue: 0.0, wantError: false},
		{name: "string invalid", value: "abc", wantValue: 0, wantError: true},
		{name: "string empty", value: "", wantValue: 0, wantError: true},

		// int
		{name: "int positive", value: int(2), wantValue: 2.0, wantError: false},

		// int64
		{name: "int64 positive", value: int64(2), wantValue: 2.0, wantError: false},

		// nil
		{name: "nil value", value: nil, wantValue: 0, wantError: true},

		// invalid types
		{name: "bool type", value: true, wantValue: 0, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := coerceToFloat(tt.value, "test")

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.wantValue {
					t.Errorf("Expected %f, got %f", tt.wantValue, result)
				}
			}
		})
	}
}

func TestValidateNonNegativeInt(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		wantValue int
		wantError bool
	}{
		// Valid values
		{name: "zero", value: 0.0, wantValue: 0, wantError: false},
		{name: "positive", value: 100.0, wantValue: 100, wantError: false},
		{name: "string zero", value: "0", wantValue: 0, wantError: false},
		{name: "string positive", value: "100", wantValue: 100, wantError: false},

		// Invalid values
		{name: "negative", value: -1.0, wantValue: 0, wantError: true},
		{name: "string negative", value: "-1", wantValue: 0, wantError: true},
		{name: "nil", value: nil, wantValue: 0, wantError: true},
		{name: "invalid string", value: "abc", wantValue: 0, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validateNonNegativeInt(tt.value, "test")

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.wantValue {
					t.Errorf("Expected %d, got %d", tt.wantValue, result)
				}
			}
		})
	}
}

func TestValidateFloatInRange(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		min       float64
		max       float64
		wantValue float64
		wantError bool
	}{
		// Valid values in range [0.1, 10.0]
		{name: "middle of range", value: 5.0, min: 0.1, max: 10.0, wantValue: 5.0, wantError: false},
		{name: "at min", value: 0.1, min: 0.1, max: 10.0, wantValue: 0.1, wantError: false},
		{name: "at max", value: 10.0, min: 0.1, max: 10.0, wantValue: 10.0, wantError: false},
		{name: "string value", value: "0.5", min: 0.1, max: 10.0, wantValue: 0.5, wantError: false},

		// Out of range
		{name: "below min", value: 0.05, min: 0.1, max: 10.0, wantValue: 0, wantError: true},
		{name: "above max", value: 15.0, min: 0.1, max: 10.0, wantValue: 0, wantError: true},
		{name: "string below min", value: "0.05", min: 0.1, max: 10.0, wantValue: 0, wantError: true},

		// Invalid values
		{name: "nil", value: nil, min: 0.1, max: 10.0, wantValue: 0, wantError: true},
		{name: "invalid string", value: "abc", min: 0.1, max: 10.0, wantValue: 0, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validateFloatInRange(tt.value, "test", tt.min, tt.max)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.wantValue {
					t.Errorf("Expected %f, got %f", tt.wantValue, result)
				}
			}
		})
	}
}

func TestValidatePositiveInt(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		key       string
		wantValue int
		wantError bool
	}{
		{
			name:      "valid int",
			args:      map[string]interface{}{"width": 800.0},
			key:       "width",
			wantValue: 800,
			wantError: false,
		},
		{
			name:      "valid float64",
			args:      map[string]interface{}{"height": 600.0},
			key:       "height",
			wantValue: 600,
			wantError: false,
		},
		{
			name:      "zero value",
			args:      map[string]interface{}{"size": 0.0},
			key:       "size",
			wantValue: 0,
			wantError: true,
		},
		{
			name:      "negative value",
			args:      map[string]interface{}{"dimension": -100.0},
			key:       "dimension",
			wantValue: 0,
			wantError: true,
		},
		{
			name:      "missing key",
			args:      map[string]interface{}{},
			key:       "missing",
			wantValue: 0,
			wantError: true,
		},
		{
			name:      "invalid type",
			args:      map[string]interface{}{"value": "not a number"},
			key:       "value",
			wantValue: 0,
			wantError: true,
		},
		// NEW: String type coercion tests
		{
			name:      "string integer",
			args:      map[string]interface{}{"width": "800"},
			key:       "width",
			wantValue: 800,
			wantError: false,
		},
		{
			name:      "string float",
			args:      map[string]interface{}{"height": "600.5"},
			key:       "height",
			wantValue: 600,
			wantError: false,
		},
		{
			name:      "string zero is not positive",
			args:      map[string]interface{}{"size": "0"},
			key:       "size",
			wantValue: 0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validatePositiveInt(tt.args[tt.key], tt.key)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.wantValue {
					t.Errorf("Expected %d, got %d", tt.wantValue, result)
				}
			}
		})
	}
}

func TestLoadImage(t *testing.T) {
	// Create a temporary test image
	tmpDir := t.TempDir()
	testImagePath := filepath.Join(tmpDir, "test.png")

	// Create a simple 100x100 white image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.White)
		}
	}

	file, err := os.Create(testImagePath)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		t.Fatalf("Failed to encode test image: %v", err)
	}

	tests := []struct {
		name      string
		path      string
		wantError bool
	}{
		{
			name:      "valid image",
			path:      testImagePath,
			wantError: false,
		},
		{
			name:      "nonexistent file",
			path:      filepath.Join(tmpDir, "nonexistent.png"),
			wantError: true,
		},
		{
			name:      "empty path",
			path:      "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := loadImage(tt.path)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Expected image, got nil")
				}
				bounds := result.Bounds()
				if bounds.Dx() != 100 || bounds.Dy() != 100 {
					t.Errorf("Expected 100x100 image, got %dx%d", bounds.Dx(), bounds.Dy())
				}
			}
		})
	}
}

func TestSaveImage(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test image
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255}) // Red
		}
	}

	tests := []struct {
		name      string
		filename  string
		wantError bool
	}{
		{
			name:      "save PNG",
			filename:  "output.png",
			wantError: false,
		},
		{
			name:      "save JPEG",
			filename:  "output.jpg",
			wantError: false,
		},
		{
			name:      "save WebP",
			filename:  "output.webp",
			wantError: false, // WebP now supported via nativewebp
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := filepath.Join(tmpDir, tt.filename)
			err := saveImage(img, outputPath)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// Verify file exists
				if _, err := os.Stat(outputPath); os.IsNotExist(err) {
					t.Error("Output file was not created")
				}

				// Verify file is readable as image
				_, err := loadImage(outputPath)
				if err != nil {
					t.Errorf("Saved image is not readable: %v", err)
				}
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "bytes",
			bytes:    500,
			expected: "500 B",
		},
		{
			name:     "kilobytes",
			bytes:    1024,
			expected: "1.0 KB",
		},
		{
			name:     "megabytes",
			bytes:    1024 * 1024,
			expected: "1.0 MB",
		},
		{
			name:     "gigabytes",
			bytes:    1024 * 1024 * 1024,
			expected: "1.0 GB",
		},
		{
			name:     "fractional KB",
			bytes:    1536,
			expected: "1.5 KB",
		},
		{
			name:     "fractional MB",
			bytes:    2621440, // 2.5 MB
			expected: "2.5 MB",
		},
		{
			name:     "zero bytes",
			bytes:    0,
			expected: "0 B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetImageDimensions(t *testing.T) {
	// Create test images with known dimensions
	tmpDir := t.TempDir()

	testCases := []struct {
		name   string
		width  int
		height int
	}{
		{"small", 100, 100},
		{"landscape", 1920, 1080},
		{"portrait", 1080, 1920},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create image
			img := image.NewRGBA(image.Rect(0, 0, tc.width, tc.height))
			path := filepath.Join(tmpDir, tc.name+".png")

			file, err := os.Create(path)
			if err != nil {
				t.Fatalf("Failed to create test image: %v", err)
			}
			png.Encode(file, img)
			file.Close()

			// Test dimension retrieval
			width, height, err := getImageDimensions(path)
			if err != nil {
				t.Fatalf("Failed to get dimensions: %v", err)
			}

			if width != tc.width || height != tc.height {
				t.Errorf("Expected %dx%d, got %dx%d", tc.width, tc.height, width, height)
			}
		})
	}

	// Test with nonexistent file
	_, _, err := getImageDimensions(filepath.Join(tmpDir, "nonexistent.png"))
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		 findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
