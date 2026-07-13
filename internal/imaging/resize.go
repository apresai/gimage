// Package imaging provides image processing operations using pure Go.
package imaging

import (
	"context"
	"fmt"

	"github.com/apresai/gimage/internal/progress"
	"github.com/disintegration/imaging"
)

// ResizeImage resizes an image to exactly the specified dimensions by cropping/filling
// to preserve the original aspect ratio (High-quality Lanczos resampling).
//
// Parameters:
//   - ctx: context for cancellation support
//   - inputPath: path to input image
//   - outputPath: path to save resized image
//   - width: target width in pixels
//   - height: target height in pixels
//
// Progress reporting can be provided via context using progress.FromContext.
//
// Returns error if:
//   - context is cancelled
//   - input file does not exist or cannot be read
//   - dimensions are not positive
//   - output cannot be written
func ResizeImage(ctx context.Context, inputPath, outputPath string, width, height int) error {
	return ResizeFill(ctx, inputPath, outputPath, width, height)
}

// ResizeFill resizes an image to exactly the specified dimensions by filling the area
// and cropping the excess (center-aligned), preserving the aspect ratio.
func ResizeFill(ctx context.Context, inputPath, outputPath string, width, height int) error {
	reporter := progress.FromContext(ctx)
	reporter.Start(ctx, fmt.Sprintf("Resizing image (crop/fill) to %dx%d", width, height))
	// Validate dimensions
	if width <= 0 {
		err := fmt.Errorf("width must be positive, got %d", width)
		reporter.Error(err)
		return err
	}
	if height <= 0 {
		err := fmt.Errorf("height must be positive, got %d", height)
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

	// Resize using Fill (center crop) to preserve aspect ratio
	reporter.Update(2, 3, "Resizing image (preserving aspect ratio)")
	resized := imaging.Fill(img, width, height, imaging.Center, imaging.Lanczos)

	// Check for cancellation
	select {
	case <-ctx.Done():
		err := fmt.Errorf("operation cancelled: %w", ctx.Err())
		reporter.Error(err)
		return err
	default:
	}

	// Save the resized image
	reporter.Update(3, 3, "Saving resized image")
	if err := imaging.Save(resized, outputPath); err != nil {
		err = fmt.Errorf("failed to save resized image to %s: %w", outputPath, err)
		reporter.Error(err)
		return err
	}

	reporter.Complete(outputPath)
	return nil
}

// ResizeFit resizes an image to fit within specified dimensions while preserving aspect ratio.
//
// Parameters:
//   - ctx: context for cancellation support
//   - inputPath: path to input image
//   - outputPath: path to save resized image
//   - width: maximum width in pixels
//   - height: maximum height in pixels
//
// The image will be resized to fit within the specified dimensions while maintaining
// its original aspect ratio. The resulting image may be smaller than the specified
// dimensions in one or both directions.
//
// Progress reporting can be provided via context using progress.FromContext.
func ResizeFit(ctx context.Context, inputPath, outputPath string, width, height int) error {
	reporter := progress.FromContext(ctx)
	reporter.Start(ctx, fmt.Sprintf("Resizing image to fit %dx%d", width, height))
	// Validate dimensions
	if width <= 0 {
		err := fmt.Errorf("width must be positive, got %d", width)
		reporter.Error(err)
		return err
	}
	if height <= 0 {
		err := fmt.Errorf("height must be positive, got %d", height)
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

	// Resize to fit within bounds, preserving aspect ratio
	reporter.Update(2, 3, "Resizing image to fit")
	resized := imaging.Fit(img, width, height, imaging.Lanczos)

	// Check for cancellation
	select {
	case <-ctx.Done():
		err := fmt.Errorf("operation cancelled: %w", ctx.Err())
		reporter.Error(err)
		return err
	default:
	}

	// Save the resized image
	reporter.Update(3, 3, "Saving resized image")
	if err := imaging.Save(resized, outputPath); err != nil {
		err = fmt.Errorf("failed to save resized image to %s: %w", outputPath, err)
		reporter.Error(err)
		return err
	}

	reporter.Complete(outputPath)
	return nil
}
