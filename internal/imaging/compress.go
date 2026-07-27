// Package imaging provides image processing operations using pure Go.
package imaging

import (
	"context"
	"fmt"

	"github.com/apresai/gimage/internal/progress"
	"github.com/disintegration/imaging"
)

// CompressImage compresses an image by adjusting its quality setting.
//
// Parameters:
//   - ctx: context for cancellation support
//   - inputPath: path to input image
//   - outputPath: path to save compressed image
//   - quality: compression quality (1-100, where 100 is highest quality)
//
// The quality parameter affects file size vs image quality:
//   - quality 90-100: excellent quality, larger files (recommended for archival)
//   - quality 80-89: very good quality, moderate files (recommended for web)
//   - quality 70-79: good quality, smaller files (acceptable for most uses)
//   - quality 60-69: acceptable quality, small files (good for thumbnails)
//   - quality 1-59: lower quality, very small files (use with caution)
//
// Note: Compression is most effective for JPEG images. PNG images will be saved
// with the same visual quality regardless of the quality parameter.
//
// Progress reporting can be provided via context using progress.FromContext.
//
// Returns error if:
//   - context is cancelled
//   - input file does not exist or cannot be read
//   - quality is not in range 1-100
//   - output cannot be written
func CompressImage(ctx context.Context, inputPath, outputPath string, quality int) error {
	reporter := progress.FromContext(ctx)
	reporter.Start(ctx, fmt.Sprintf("Compressing image (quality %d)", quality))

	// Validate quality
	if quality < 1 || quality > 100 {
		err := fmt.Errorf("quality must be between 1 and 100, got %d", quality)
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

	// Compress and save
	reporter.Update(2, 3, "Compressing and saving image")

	// Use the format from the output path
	outputFormat := ExtractFormatFromPath(outputPath)

	// For JPEG, use explicit quality
	if outputFormat == "jpeg" || outputFormat == "jpg" {
		if err := imaging.Save(img, outputPath, imaging.JPEGQuality(quality)); err != nil {
			err = fmt.Errorf("failed to save compressed image to %s: %w", outputPath, err)
			reporter.Error(err)
			return err
		}
	} else if outputFormat == "png" {
		// For PNG, use PNG compression (lossless)
		// PNG doesn't support quality parameter in the same way
		// Use DefaultCompression level (which is -1)
		if err := imaging.Save(img, outputPath, imaging.PNGCompressionLevel(-1)); err != nil {
			err = fmt.Errorf("failed to save compressed image to %s: %w", outputPath, err)
			reporter.Error(err)
			return err
		}
	} else {
		// For other formats, save with default settings
		if err := imaging.Save(img, outputPath); err != nil {
			err = fmt.Errorf("failed to save compressed image to %s: %w", outputPath, err)
			reporter.Error(err)
			return err
		}
	}

	// Check for cancellation
	select {
	case <-ctx.Done():
		err := fmt.Errorf("operation cancelled: %w", ctx.Err())
		reporter.Error(err)
		return err
	default:
	}

	reporter.Update(3, 3, "Compression complete")
	reporter.Complete(outputPath)
	return nil
}
