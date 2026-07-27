// Package imaging provides image processing operations using pure Go.
package imaging

import (
	"context"
	"fmt"
	"image"

	"github.com/apresai/gimage/internal/progress"
	"github.com/disintegration/imaging"
)

// CropImage crops a rectangular region from an image.
//
// Parameters:
//   - ctx: context for cancellation support
//   - inputPath: path to input image
//   - outputPath: path to save cropped image
//   - x, y: top-left corner coordinates of the crop region
//   - width, height: dimensions of crop region
//
// Progress reporting can be provided via context using progress.FromContext.
//
// Returns error if:
//   - context is cancelled
//   - input file does not exist or cannot be read
//   - crop region is outside image bounds
//   - dimensions are not positive
//   - output cannot be written
func CropImage(ctx context.Context, inputPath, outputPath string, x, y, width, height int) error {
	reporter := progress.FromContext(ctx)
	reporter.Start(ctx, fmt.Sprintf("Cropping image region %dx%d at (%d,%d)", width, height, x, y))

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

	// Validate coordinates
	if x < 0 {
		err := fmt.Errorf("x coordinate must be non-negative, got %d", x)
		reporter.Error(err)
		return err
	}
	if y < 0 {
		err := fmt.Errorf("y coordinate must be non-negative, got %d", y)
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

	// Get image dimensions
	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	// Validate crop region is within image bounds
	reporter.Update(2, 4, "Validating crop region")
	if x >= imgWidth {
		err := fmt.Errorf("x coordinate %d is outside image width %d", x, imgWidth)
		reporter.Error(err)
		return err
	}
	if y >= imgHeight {
		err := fmt.Errorf("y coordinate %d is outside image height %d", y, imgHeight)
		reporter.Error(err)
		return err
	}
	if x+width > imgWidth {
		err := fmt.Errorf("crop region (x=%d + width=%d = %d) exceeds image width %d", x, width, x+width, imgWidth)
		reporter.Error(err)
		return err
	}
	if y+height > imgHeight {
		err := fmt.Errorf("crop region (y=%d + height=%d = %d) exceeds image height %d", y, height, y+height, imgHeight)
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

	// Create crop rectangle
	cropRect := image.Rect(x, y, x+width, y+height)

	// Perform the crop
	reporter.Update(3, 4, "Cropping image")
	cropped := imaging.Crop(img, cropRect)

	// Check for cancellation
	select {
	case <-ctx.Done():
		err := fmt.Errorf("operation cancelled: %w", ctx.Err())
		reporter.Error(err)
		return err
	default:
	}

	// Save the cropped image
	reporter.Update(4, 4, "Saving cropped image")
	if err := imaging.Save(cropped, outputPath); err != nil {
		err = fmt.Errorf("failed to save cropped image to %s: %w", outputPath, err)
		reporter.Error(err)
		return err
	}

	reporter.Complete(outputPath)
	return nil
}
