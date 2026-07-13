// Package imaging provides image processing operations using pure Go.
package imaging

import (
	"context"
	"fmt"

	"github.com/apresai/gimage/internal/progress"
	"github.com/disintegration/imaging"
)

// ScaleImage scales an image by a factor using high-quality Lanczos resampling.
//
// Parameters:
//   - ctx: context for cancellation support
//   - inputPath: path to input image
//   - outputPath: path to save scaled image
//   - factor: scaling factor (e.g., 0.5 for half size, 2.0 for double size)
//
// The image dimensions will be multiplied by the factor. For example:
//   - factor 0.5 = 50% size (half)
//   - factor 1.0 = same size (no change)
//   - factor 2.0 = 200% size (double)
//
// Progress reporting can be provided via context using progress.FromContext.
//
// Returns error if:
//   - context is cancelled
//   - input file does not exist or cannot be read
//   - factor is not positive
//   - output cannot be written
func ScaleImage(ctx context.Context, inputPath, outputPath string, factor float64) error {
	reporter := progress.FromContext(ctx)
	reporter.Start(ctx, fmt.Sprintf("Scaling image by factor %.2f", factor))

	// Validate factor
	if factor <= 0 {
		err := fmt.Errorf("factor must be positive, got %f", factor)
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
	reporter.Update(1, 4, "Loading input image")
	img, err := imaging.Open(inputPath)
	if err != nil {
		err = fmt.Errorf("failed to open image %s: %w", inputPath, err)
		reporter.Error(err)
		return err
	}

	// Get current dimensions
	bounds := img.Bounds()
	currentWidth := bounds.Dx()
	currentHeight := bounds.Dy()

	// Calculate new dimensions
	reporter.Update(2, 4, "Calculating new dimensions")
	newWidth := int(float64(currentWidth) * factor)
	newHeight := int(float64(currentHeight) * factor)

	// Ensure dimensions are at least 1
	if newWidth < 1 {
		newWidth = 1
	}
	if newHeight < 1 {
		newHeight = 1
	}

	// Check for cancellation
	select {
	case <-ctx.Done():
		err := fmt.Errorf("operation cancelled: %w", ctx.Err())
		reporter.Error(err)
		return err
	default:
	}

	// Scale using Lanczos resampling (high quality)
	reporter.Update(3, 4, fmt.Sprintf("Scaling to %dx%d", newWidth, newHeight))
	scaled := imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)

	// Check for cancellation
	select {
	case <-ctx.Done():
		err := fmt.Errorf("operation cancelled: %w", ctx.Err())
		reporter.Error(err)
		return err
	default:
	}

	// Save the scaled image
	reporter.Update(4, 4, "Saving scaled image")
	if err := imaging.Save(scaled, outputPath); err != nil {
		err = fmt.Errorf("failed to save scaled image to %s: %w", outputPath, err)
		reporter.Error(err)
		return err
	}

	reporter.Complete(outputPath)
	return nil
}
