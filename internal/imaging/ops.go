// Package imaging provides image processing operations using pure Go.
package imaging

import (
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/HugoSmits86/nativewebp"
	"golang.org/x/image/bmp"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/tiff"
	_ "golang.org/x/image/webp" // Register WebP decoder for image.Decode
)

// Open loads and decodes an image from inputPath.
// Supported formats: PNG, JPEG, WebP, GIF, TIFF, BMP.
func Open(inputPath string) (image.Image, error) {
	f, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", inputPath, err)
	}
	defer func() { _ = f.Close() }()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image %s: %w", inputPath, err)
	}
	return img, nil
}

// Save encodes and writes img to outputPath, inferring the format from the file extension.
// For JPEG outputs, a default quality of 90 is used.
func Save(img image.Image, outputPath string) error {
	return SaveWithQuality(img, outputPath, 90)
}

// SaveJPEG encodes and writes img as a JPEG to outputPath with the specified quality (1-100).
func SaveJPEG(img image.Image, outputPath string, quality int) error {
	return encodeToFile(img, outputPath, "jpeg", quality)
}

// SaveWithQuality encodes and writes img to outputPath, inferring the format from the extension
// and using quality (1-100) when the target format is JPEG.
func SaveWithQuality(img image.Image, outputPath string, quality int) error {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(outputPath), "."))
	if ext == "" {
		return fmt.Errorf("unsupported image format: no file extension in %s", outputPath)
	}
	return encodeToFile(img, outputPath, ext, quality)
}

func encodeToFile(img image.Image, outputPath, format string, quality int) error {
	if img == nil {
		return fmt.Errorf("image cannot be nil")
	}
	if quality <= 0 || quality > 100 {
		quality = 90
	}

	normalized := normalizeFormat(format)
	switch normalized {
	case "png", "jpeg", "webp", "gif", "tiff", "bmp":
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	if !supportsTransparency(normalized) && hasTransparency(img) {
		img = removeTransparency(img)
	}

	if dir := filepath.Dir(outputPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}

	var encodeErr error
	switch normalized {
	case "png":
		encodeErr = png.Encode(outFile, img)
	case "jpeg":
		encodeErr = jpeg.Encode(outFile, img, &jpeg.Options{Quality: quality})
	case "webp":
		encodeErr = nativewebp.Encode(outFile, img, &nativewebp.Options{
			UseExtendedFormat: false,
		})
	case "gif":
		encodeErr = gif.Encode(outFile, img, &gif.Options{NumColors: 256})
	case "tiff":
		encodeErr = tiff.Encode(outFile, img, &tiff.Options{Compression: tiff.Deflate, Predictor: true})
	case "bmp":
		encodeErr = bmp.Encode(outFile, img)
	}

	closeErr := outFile.Close()
	if encodeErr != nil {
		_ = os.Remove(outputPath)
		return fmt.Errorf("failed to encode %s: %w", normalized, encodeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close output file %s: %w", outputPath, closeErr)
	}
	return nil
}

// Clone returns a copy of img as an *image.NRGBA with bounds (0, 0, w, h).
func Clone(img image.Image) *image.NRGBA {
	if img == nil {
		return &image.NRGBA{}
	}
	b := img.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Src)
	return dst
}

// Resize scales img to the exact target width and height using Catmull-Rom resampling.
func Resize(img image.Image, width, height int) *image.NRGBA {
	if img == nil || width <= 0 || height <= 0 {
		return &image.NRGBA{}
	}
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return &image.NRGBA{}
	}
	if b.Dx() == width && b.Dy() == height {
		return Clone(img)
	}
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	return dst
}

// Fit scales img down to fit within maxWidth x maxHeight while preserving aspect ratio.
// If img is already smaller than or equal to maxWidth x maxHeight, a clone at original size is returned.
func Fit(img image.Image, maxWidth, maxHeight int) *image.NRGBA {
	if img == nil || maxWidth <= 0 || maxHeight <= 0 {
		return &image.NRGBA{}
	}
	b := img.Bounds()
	srcW, srcH := b.Dx(), b.Dy()
	if srcW <= 0 || srcH <= 0 {
		return &image.NRGBA{}
	}
	if srcW <= maxWidth && srcH <= maxHeight {
		return Clone(img)
	}

	srcAspect := float64(srcW) / float64(srcH)
	maxAspect := float64(maxWidth) / float64(maxHeight)

	var newW, newH int
	if srcAspect > maxAspect {
		newW = maxWidth
		newH = int(math.Round(float64(newW) / srcAspect))
	} else {
		newH = maxHeight
		newW = int(math.Round(float64(newH) * srcAspect))
	}
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}
	return Resize(img, newW, newH)
}

// Fill center-crops img to match the target aspect ratio and scales it to width x height.
func Fill(img image.Image, width, height int) *image.NRGBA {
	if img == nil || width <= 0 || height <= 0 {
		return &image.NRGBA{}
	}
	b := img.Bounds()
	srcW, srcH := b.Dx(), b.Dy()
	if srcW <= 0 || srcH <= 0 {
		return &image.NRGBA{}
	}
	if srcW == width && srcH == height {
		return Clone(img)
	}

	srcAspect := float64(srcW) / float64(srcH)
	dstAspect := float64(width) / float64(height)

	var cropW, cropH int
	if srcAspect > dstAspect {
		cropH = srcH
		cropW = int(math.Round(float64(cropH) * dstAspect))
		if cropW > srcW {
			cropW = srcW
		}
		if cropW < 1 {
			cropW = 1
		}
	} else {
		cropW = srcW
		cropH = int(math.Round(float64(cropW) / dstAspect))
		if cropH > srcH {
			cropH = srcH
		}
		if cropH < 1 {
			cropH = 1
		}
	}

	minX := b.Min.X + (srcW-cropW)/2
	minY := b.Min.Y + (srcH-cropH)/2
	cropRect := image.Rect(minX, minY, minX+cropW, minY+cropH)

	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, cropRect, draw.Src, nil)
	return dst
}

// Crop extracts the sub-image specified by rect (in 0-based coordinates relative to img's origin).
func Crop(img image.Image, rect image.Rectangle) *image.NRGBA {
	if img == nil {
		return &image.NRGBA{}
	}
	b := img.Bounds()
	r := rect.Add(b.Min).Intersect(b)
	if r.Empty() {
		return &image.NRGBA{}
	}
	dst := image.NewNRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(dst, dst.Bounds(), img, r.Min, draw.Src)
	return dst
}
