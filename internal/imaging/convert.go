package imaging

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/HugoSmits86/nativewebp"
	"github.com/apresai/gimage/internal/progress"
	"github.com/disintegration/imaging"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

// ConvertImageData converts image data from one format to another.
//
// Parameters:
//   - data: input image data (any supported format)
//   - targetFormat: desired output format (png, jpg, jpeg, webp, gif, tiff, bmp)
//
// Returns converted image data and error if conversion fails.
func ConvertImageData(data []byte, targetFormat string) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("input data is empty")
	}

	// Normalize target format
	targetFormat = strings.ToLower(targetFormat)
	targetFormat = strings.TrimPrefix(targetFormat, ".")

	// Decode the input image
	img, srcFormat, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Normalize source format for comparison
	normalizedSrcFormat := normalizeFormat(srcFormat)
	normalizedTargetFormat := normalizeFormat(targetFormat)

	// If formats match, return original data
	if normalizedSrcFormat == normalizedTargetFormat {
		return data, nil
	}

	// Convert format
	var buf bytes.Buffer
	if err := encodeImage(&buf, img, targetFormat); err != nil {
		return nil, fmt.Errorf("failed to encode image as %s: %w", targetFormat, err)
	}

	return buf.Bytes(), nil
}

// ConvertImageFile converts an image file from one format to another.
//
// Parameters:
//   - ctx: context for cancellation support
//   - inputPath: path to input image
//   - outputPath: path to save converted image
//
// The output format is determined by the file extension of outputPath.
// Progress reporting can be provided via context using progress.FromContext.
func ConvertImageFile(ctx context.Context, inputPath, outputPath string) error {
	reporter := progress.FromContext(ctx)
	targetFormat := ExtractFormatFromPath(outputPath)
	reporter.Start(ctx, fmt.Sprintf("Converting image to %s", targetFormat))

	// Check for cancellation
	select {
	case <-ctx.Done():
		err := fmt.Errorf("operation cancelled: %w", ctx.Err())
		reporter.Error(err)
		return err
	default:
	}

	// Load the input image
	reporter.Update(1, 3, "Loading input image")
	img, err := imaging.Open(inputPath)
	if err != nil {
		err = fmt.Errorf("failed to open image %s: %w", inputPath, err)
		reporter.Error(err)
		return err
	}

	// Check for cancellation
	select {
	case <-ctx.Done():
		err := fmt.Errorf("operation cancelled: %w", ctx.Err())
		reporter.Error(err)
		return err
	default:
	}

	// Save with the target format
	reporter.Update(2, 3, fmt.Sprintf("Converting to %s format", targetFormat))
	if err := SaveImageWithFormat(img, outputPath, targetFormat); err != nil {
		reporter.Error(err)
		return err
	}

	reporter.Update(3, 3, "Conversion complete")
	reporter.Complete(outputPath)
	return nil
}

// SaveImageWithFormat saves an image to a file with explicit format specification.
func SaveImageWithFormat(img image.Image, path string, format string) error {
	format = strings.ToLower(format)
	format = strings.TrimPrefix(format, ".")

	// Handle transparency for formats that don't support it
	if !supportsTransparency(format) && hasTransparency(img) {
		img = removeTransparency(img)
	}

	// Handle WebP encoding with nativewebp
	if format == "webp" {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}

		dir := filepath.Dir(absPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		outFile, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		defer outFile.Close()

		// nativewebp provides lossless VP8L encoding
		return nativewebp.Encode(outFile, img, &nativewebp.Options{
			UseExtendedFormat: false, // Basic VP8L format, no metadata
		})
	}

	// Use imaging package's Save which handles format detection
	if format == "jpg" || format == "jpeg" {
		// For JPEG, use explicit quality setting
		return imaging.Save(img, path, imaging.JPEGQuality(90))
	}

	return imaging.Save(img, path)
}

// ExtractFormatFromPath extracts the image format from a file path.
func ExtractFormatFromPath(path string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return "png" // Default
	}

	// Remove leading dot and convert to lowercase
	format := strings.ToLower(strings.TrimPrefix(ext, "."))

	// Normalize variations
	return normalizeFormat(format)
}

// encodeImage encodes an image to a specific format
func encodeImage(w io.Writer, img image.Image, format string) error {
	format = strings.ToLower(format)

	// Handle transparency for formats that don't support it
	if !supportsTransparency(format) && hasTransparency(img) {
		img = removeTransparency(img)
	}

	switch format {
	case "png":
		return png.Encode(w, img)

	case "jpg", "jpeg":
		// Use 90% quality for JPEG
		return jpeg.Encode(w, img, &jpeg.Options{Quality: 90})

	case "gif":
		return gif.Encode(w, img, &gif.Options{NumColors: 256})

	case "webp":
		// Use nativewebp for pure Go lossless WebP encoding (VP8L)
		return nativewebp.Encode(w, img, &nativewebp.Options{
			UseExtendedFormat: false, // Basic VP8L format
		})

	case "tiff", "tif":
		return tiff.Encode(w, img, &tiff.Options{Compression: tiff.Deflate})

	case "bmp":
		return bmp.Encode(w, img)

	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// normalizeFormat normalizes format strings for comparison
func normalizeFormat(format string) string {
	format = strings.ToLower(format)
	format = strings.TrimPrefix(format, ".")

	// Normalize variations
	switch format {
	case "jpg", "jpeg":
		return "jpeg"
	case "tif":
		return "tiff"
	default:
		return format
	}
}

// supportsTransparency returns true if the format supports transparency
func supportsTransparency(format string) bool {
	format = normalizeFormat(format)
	switch format {
	case "png", "gif", "webp":
		return true
	default:
		return false
	}
}

// hasTransparency checks if an image has transparent pixels
func hasTransparency(img image.Image) bool {
	bounds := img.Bounds()

	// Check a sample of pixels for transparency
	// Full scan would be too expensive, so we sample
	for y := bounds.Min.Y; y < bounds.Max.Y; y += 10 {
		for x := bounds.Min.X; x < bounds.Max.X; x += 10 {
			_, _, _, a := img.At(x, y).RGBA()
			// RGBA values are in range 0-65535, fully opaque is 65535
			if a < 65535 {
				return true
			}
		}
	}

	return false
}

// removeTransparency converts transparent areas to white background
func removeTransparency(img image.Image) image.Image {
	bounds := img.Bounds()
	newImg := image.NewRGBA(bounds)

	// Fill with white background
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			newImg.Set(x, y, white)
		}
	}

	// Composite original image over white background
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			r, g, b, a := c.RGBA()

			// If fully opaque, just set the color
			if a == 65535 {
				newImg.Set(x, y, c)
				continue
			}

			// Alpha blend with white background
			if a > 0 {
				// Convert to 8-bit values
				alpha := float64(a) / 65535.0
				r8 := uint8((float64(r>>8) * alpha) + (255.0 * (1.0 - alpha)))
				g8 := uint8((float64(g>>8) * alpha) + (255.0 * (1.0 - alpha)))
				b8 := uint8((float64(b>>8) * alpha) + (255.0 * (1.0 - alpha)))

				newImg.Set(x, y, color.RGBA{R: r8, G: g8, B: b8, A: 255})
			}
		}
	}

	return newImg
}
