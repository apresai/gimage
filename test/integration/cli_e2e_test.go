// +build e2e

package integration

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/apresai/gimage/internal/imaging"
)

// TestCLIResizeE2E tests the gimage resize command end-to-end
func TestCLIResizeE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Ensure binary is built
	binaryPath := ensureBinaryBuilt(t)

	tmpDir := t.TempDir()

	// Create test image
	testImagePath := filepath.Join(tmpDir, "test.png")
	createCLITestImage(t, testImagePath, 800, 600)

	tests := []struct {
		name           string
		width          int
		height         int
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "resize to 400x300",
			width:          400,
			height:         300,
			expectedWidth:  400,
			expectedHeight: 300,
		},
		{
			name:           "resize to 200x150",
			width:          200,
			height:         150,
			expectedWidth:  200,
			expectedHeight: 150,
		},
		{
			name:           "resize to 100x100 (square)",
			width:          100,
			height:         100,
			expectedWidth:  100,
			expectedHeight: 100,
		},
		{
			name:           "resize to 1600x1200 (upscale)",
			width:          1600,
			height:         1200,
			expectedWidth:  1600,
			expectedHeight: 1200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create explicit output path
			outputPath := filepath.Join(tmpDir, fmt.Sprintf("resized_%dx%d.png", tt.width, tt.height))

			// Run CLI command with explicit output
			cmd := exec.Command(binaryPath, "resize",
				"--input", testImagePath,
				"--width", fmt.Sprintf("%d", tt.width),
				"--height", fmt.Sprintf("%d", tt.height),
				"--output", outputPath)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("CLI command failed: %v\nOutput: %s", err, string(output))
			}

			t.Logf("CLI output: %s", string(output))

			// Verify output file exists
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				t.Fatalf("Output file not created: %s", outputPath)
			}

			// Verify image dimensions
			img, err := imaging.Open(outputPath)
			if err != nil {
				t.Fatalf("Failed to open output image: %v", err)
			}

			bounds := img.Bounds()
			width := bounds.Dx()
			height := bounds.Dy()

			if width != tt.expectedWidth {
				t.Errorf("Expected width %d, got %d", tt.expectedWidth, width)
			}
			if height != tt.expectedHeight {
				t.Errorf("Expected height %d, got %d", tt.expectedHeight, height)
			}

			t.Logf("✅ Resized image created successfully: %dx%d", width, height)
		})
	}
}

// TestCLIScaleE2E tests the gimage scale command end-to-end
func TestCLIScaleE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	binaryPath := ensureBinaryBuilt(t)
	tmpDir := t.TempDir()

	// Create test image
	testImagePath := filepath.Join(tmpDir, "test.png")
	createCLITestImage(t, testImagePath, 800, 600)

	tests := []struct {
		name           string
		factor         float64
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "scale down by 0.5",
			factor:         0.5,
			expectedWidth:  400,
			expectedHeight: 300,
		},
		{
			name:           "scale down by 0.25",
			factor:         0.25,
			expectedWidth:  200,
			expectedHeight: 150,
		},
		{
			name:           "scale up by 2.0",
			factor:         2.0,
			expectedWidth:  1600,
			expectedHeight: 1200,
		},
		{
			name:           "scale by 0.75",
			factor:         0.75,
			expectedWidth:  600,
			expectedHeight: 450,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create explicit output path
			outputPath := filepath.Join(tmpDir, fmt.Sprintf("scaled_%.2fx.png", tt.factor))

			// Run CLI command with explicit output
			cmd := exec.Command(binaryPath, "scale",
				"--input", testImagePath,
				"--factor", fmt.Sprintf("%.2f", tt.factor),
				"--output", outputPath)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("CLI command failed: %v\nOutput: %s", err, string(output))
			}

			t.Logf("CLI output: %s", string(output))

			// Verify output file exists
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				t.Fatalf("Output file not created: %s", outputPath)
			}

			// Verify image dimensions
			img, err := imaging.Open(outputPath)
			if err != nil {
				t.Fatalf("Failed to open output image: %v", err)
			}

			bounds := img.Bounds()
			width := bounds.Dx()
			height := bounds.Dy()

			if width != tt.expectedWidth {
				t.Errorf("Expected width %d, got %d", tt.expectedWidth, width)
			}
			if height != tt.expectedHeight {
				t.Errorf("Expected height %d, got %d", tt.expectedHeight, height)
			}

			t.Logf("✅ Scaled image created successfully: %dx%d (%.2fx scale)", width, height, tt.factor)
		})
	}
}

// TestCLICropE2E tests the gimage crop command end-to-end
func TestCLICropE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	binaryPath := ensureBinaryBuilt(t)
	tmpDir := t.TempDir()

	// Create test image with colored quadrants for visual verification
	testImagePath := filepath.Join(tmpDir, "test.png")
	createCLITestImageWithQuadrants(t, testImagePath, 800, 600)

	tests := []struct {
		name           string
		x              int
		y              int
		width          int
		height         int
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "crop 400x300 from top-left (0,0)",
			x:              0,
			y:              0,
			width:          400,
			height:         300,
			expectedWidth:  400,
			expectedHeight: 300,
		},
		{
			name:           "crop 200x150 from center (300,225)",
			x:              300,
			y:              225,
			width:          200,
			height:         150,
			expectedWidth:  200,
			expectedHeight: 150,
		},
		{
			name:           "crop 400x300 from bottom-right (400,300)",
			x:              400,
			y:              300,
			width:          400,
			height:         300,
			expectedWidth:  400,
			expectedHeight: 300,
		},
		{
			name:           "crop 600x400 from (100,100)",
			x:              100,
			y:              100,
			width:          600,
			height:         400,
			expectedWidth:  600,
			expectedHeight: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create explicit output path
			outputPath := filepath.Join(tmpDir, fmt.Sprintf("cropped_%d_%d_%dx%d.png", tt.x, tt.y, tt.width, tt.height))

			// Run CLI command with explicit output
			cmd := exec.Command(binaryPath, "crop",
				"--input", testImagePath,
				"--x", fmt.Sprintf("%d", tt.x),
				"--y", fmt.Sprintf("%d", tt.y),
				"--width", fmt.Sprintf("%d", tt.width),
				"--height", fmt.Sprintf("%d", tt.height),
				"--output", outputPath)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("CLI command failed: %v\nOutput: %s", err, string(output))
			}

			t.Logf("CLI output: %s", string(output))

			// Verify output file exists
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				t.Fatalf("Output file not created: %s", outputPath)
			}

			// Verify image dimensions
			img, err := imaging.Open(outputPath)
			if err != nil {
				t.Fatalf("Failed to open output image: %v", err)
			}

			bounds := img.Bounds()
			width := bounds.Dx()
			height := bounds.Dy()

			if width != tt.expectedWidth {
				t.Errorf("Expected width %d, got %d", tt.expectedWidth, width)
			}
			if height != tt.expectedHeight {
				t.Errorf("Expected height %d, got %d", tt.expectedHeight, height)
			}

			t.Logf("✅ Cropped image created successfully: %dx%d", width, height)
		})
	}
}

// TestCLICropCenterE2E tests center crop by calculating the center position manually
func TestCLICropCenterE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	binaryPath := ensureBinaryBuilt(t)
	tmpDir := t.TempDir()

	// Create test image
	testImagePath := filepath.Join(tmpDir, "test.png")
	createCLITestImage(t, testImagePath, 800, 600)

	tests := []struct {
		name           string
		imageW         int
		imageH         int
		cropW          int
		cropH          int
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "center crop 400x400 from 800x600",
			imageW:         800,
			imageH:         600,
			cropW:          400,
			cropH:          400,
			expectedWidth:  400,
			expectedHeight: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate center position: x = (imageW - cropW) / 2, y = (imageH - cropH) / 2
			x := (tt.imageW - tt.cropW) / 2
			y := (tt.imageH - tt.cropH) / 2

			outputPath := filepath.Join(tmpDir, "center_cropped.png")
			cmd := exec.Command(binaryPath, "crop",
				"--input", testImagePath,
				"--x", fmt.Sprintf("%d", x),
				"--y", fmt.Sprintf("%d", y),
				"--width", fmt.Sprintf("%d", tt.cropW),
				"--height", fmt.Sprintf("%d", tt.cropH),
				"--output", outputPath)

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("CLI command failed: %v\nOutput: %s", err, string(output))
			}

			t.Logf("CLI output: %s", string(output))

			// Verify output exists
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				t.Fatalf("Output file not created: %s", outputPath)
			}

			// Verify dimensions
			img, err := imaging.Open(outputPath)
			if err != nil {
				t.Fatalf("Failed to open output image: %v", err)
			}

			bounds := img.Bounds()
			if bounds.Dx() != tt.expectedWidth || bounds.Dy() != tt.expectedHeight {
				t.Errorf("Expected %dx%d, got %dx%d", tt.expectedWidth, tt.expectedHeight, bounds.Dx(), bounds.Dy())
			}

			t.Logf("✅ Center cropped image: %dx%d (from position %d,%d)", bounds.Dx(), bounds.Dy(), x, y)
		})
	}
}

// TestCLIErrorHandling tests CLI error cases
func TestCLIErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	binaryPath := ensureBinaryBuilt(t)
	tmpDir := t.TempDir()

	testImagePath := filepath.Join(tmpDir, "test.png")
	createCLITestImage(t, testImagePath, 800, 600)

	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "resize with missing input flag",
			args:        []string{"resize", "--width", "300", "--height", "300"},
			expectError: true,
			errorMsg:    "required flag",
		},
		{
			name:        "resize with missing width",
			args:        []string{"resize", "--input", testImagePath, "--height", "300"},
			expectError: true,
			errorMsg:    "required flag",
		},
		{
			name:        "resize with invalid width",
			args:        []string{"resize", "--input", testImagePath, "--width", "0", "--height", "300"},
			expectError: true,
			errorMsg:    "must be a positive",
		},
		{
			name:        "scale with missing factor",
			args:        []string{"scale", "--input", testImagePath},
			expectError: true,
			errorMsg:    "required flag",
		},
		{
			name:        "scale with invalid factor",
			args:        []string{"scale", "--input", testImagePath, "--factor", "0"},
			expectError: true,
			errorMsg:    "must be a positive",
		},
		{
			name:        "crop with missing width",
			args:        []string{"crop", "--input", testImagePath, "--x", "0", "--y", "0", "--height", "100"},
			expectError: true,
			errorMsg:    "required flag",
		},
		{
			name:        "crop exceeding image bounds",
			args:        []string{"crop", "--input", testImagePath, "--x", "0", "--y", "0", "--width", "10000", "--height", "10000"},
			expectError: true,
			errorMsg:    "",
		},
		{
			name:        "non-existent input file",
			args:        []string{"resize", "--input", "/nonexistent/file.png", "--width", "100", "--height", "100"},
			expectError: true,
			errorMsg:    "does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, tt.args...)
			output, err := cmd.CombinedOutput()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, but command succeeded. Output: %s", string(output))
				} else {
					t.Logf("✅ Command failed as expected: %v", err)
					if tt.errorMsg != "" && !strings.Contains(string(output), tt.errorMsg) {
						t.Logf("Warning: Expected error message to contain '%s', got: %s", tt.errorMsg, string(output))
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected success, but got error: %v\nOutput: %s", err, string(output))
				}
			}
		})
	}
}

// Helper functions

var buildBinaryOnce sync.Once

// ensureBinaryBuilt builds the gimage binary once per test run and returns its absolute path
func ensureBinaryBuilt(t *testing.T) string {
	t.Helper()

	// Get absolute path to binary
	binaryPath := filepath.Join("..", "..", "bin", "gimage")
	absPath, err := filepath.Abs(binaryPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	buildBinaryOnce.Do(func() {
		t.Log("Building gimage binary for E2E tests...")
		cmd := exec.Command("make", "build")
		repoRoot := filepath.Join("..", "..")
		absRepoRoot, _ := filepath.Abs(repoRoot)
		cmd.Dir = absRepoRoot
		output, buildErr := cmd.CombinedOutput()
		if buildErr != nil {
			t.Fatalf("Failed to build binary: %v\nOutput: %s", buildErr, string(output))
		}
		t.Logf("Built binary: %s", absPath)
	})

	// Verify binary exists
	if _, err := os.Stat(absPath); err != nil {
		t.Fatalf("Binary not found after build: %s", absPath)
	}

	return absPath
}

// createCLITestImage creates a simple gradient test image
func createCLITestImage(t *testing.T, path string, width, height int) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: 128,
				A: 255,
			})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("Failed to encode test image: %v", err)
	}

	t.Logf("Created test image: %s (%dx%d)", path, width, height)
}

// createCLITestImageWithQuadrants creates an image with 4 colored quadrants
func createCLITestImageWithQuadrants(t *testing.T, path string, width, height int) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with different colors in quadrants
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var c color.RGBA
			if x < width/2 && y < height/2 {
				c = color.RGBA{255, 0, 0, 255} // Red top-left
			} else if x >= width/2 && y < height/2 {
				c = color.RGBA{0, 255, 0, 255} // Green top-right
			} else if x < width/2 && y >= height/2 {
				c = color.RGBA{0, 0, 255, 255} // Blue bottom-left
			} else {
				c = color.RGBA{255, 255, 0, 255} // Yellow bottom-right
			}
			img.Set(x, y, c)
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("Failed to encode test image: %v", err)
	}

	t.Logf("Created test image with quadrants: %s (%dx%d)", path, width, height)
}

// TestCLIAllCommandsIntegration runs all CLI commands in sequence
func TestCLIAllCommandsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	binaryPath := ensureBinaryBuilt(t)
	tmpDir := t.TempDir()

	// Create original test image
	originalPath := filepath.Join(tmpDir, "original.png")
	createCLITestImageWithQuadrants(t, originalPath, 800, 600)

	// Step 1: Resize
	resizedPath := filepath.Join(tmpDir, "step1_resized.png")
	cmd := exec.Command(binaryPath, "resize",
		"--input", originalPath,
		"--width", "400",
		"--height", "300",
		"--output", resizedPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Resize failed: %v\nOutput: %s", err, string(output))
	}
	t.Log("✅ Step 1: Resized 800x600 → 400x300")

	// Step 2: Scale the resized image
	scaledPath := filepath.Join(tmpDir, "step2_scaled.png")
	cmd = exec.Command(binaryPath, "scale",
		"--input", resizedPath,
		"--factor", "0.5",
		"--output", scaledPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Scale failed: %v\nOutput: %s", err, string(output))
	}
	t.Log("✅ Step 2: Scaled 400x300 → 200x150 (0.5x)")

	// Step 3: Crop the scaled image
	croppedPath := filepath.Join(tmpDir, "step3_cropped.png")
	cmd = exec.Command(binaryPath, "crop",
		"--input", scaledPath,
		"--x", "50",
		"--y", "50",
		"--width", "100",
		"--height", "50",
		"--output", croppedPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Crop failed: %v\nOutput: %s", err, string(output))
	}
	t.Log("✅ Step 3: Cropped 200x150 → 100x50")

	// Verify final image
	finalImg, err := imaging.Open(croppedPath)
	if err != nil {
		t.Fatalf("Failed to open final image: %v", err)
	}

	bounds := finalImg.Bounds()
	if bounds.Dx() != 100 || bounds.Dy() != 50 {
		t.Errorf("Expected final dimensions 100x50, got %dx%d", bounds.Dx(), bounds.Dy())
	}

	t.Log("✅ Integration test complete: original → resize → scale → crop")
	t.Logf("   Original: 800x600")
	t.Logf("   Resized:  400x300")
	t.Logf("   Scaled:   200x150")
	t.Logf("   Cropped:  100x50")
}
