package tools

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/apresai/gimage/internal/generate"
	gimaging "github.com/apresai/gimage/internal/imaging"
	"github.com/disintegration/imaging"
)

// generateOutputPath creates an output path based on input and suffix
func generateOutputPath(input, suffix string) string {
	ext := filepath.Ext(input)
	base := strings.TrimSuffix(filepath.Base(input), ext)
	return fmt.Sprintf("%s_%s%s", base, suffix, ext)
}

// generateTimestampedPath creates a path with timestamp
func generateTimestampedPath(prefix, ext string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s_%d.%s", prefix, timestamp, ext)
}

// getImageDimensions returns the width and height of an image
func getImageDimensions(path string) (int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	img, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode image config: %w", err)
	}

	return img.Width, img.Height, nil
}

// getFileSize returns the size of a file in bytes
func getFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("failed to stat file: %w", err)
	}
	return info.Size(), nil
}

// coerceToInt converts various numeric types to int.
// Accepts: float64 (JSON default), string, int, int64.
// This allows LLMs to pass numbers as strings (e.g., "20" instead of 20).
func coerceToInt(value interface{}, name string) (int, error) {
	if value == nil {
		return 0, fmt.Errorf("%s is required", name)
	}

	switch v := value.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case string:
		// Try to parse as integer first, then as float
		if i, err := strconv.Atoi(v); err == nil {
			return i, nil
		}
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return int(f), nil
		}
		return 0, fmt.Errorf("%s must be a number, got string: %q", name, v)
	default:
		return 0, fmt.Errorf("%s must be a number, got %T", name, value)
	}
}

// coerceToFloat converts various numeric types to float64.
// Accepts: float64 (JSON default), string, int, int64.
// This allows LLMs to pass numbers as strings (e.g., "0.5" instead of 0.5).
func coerceToFloat(value interface{}, name string) (float64, error) {
	if value == nil {
		return 0, fmt.Errorf("%s is required", name)
	}

	switch v := value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, nil
		}
		return 0, fmt.Errorf("%s must be a number, got string: %q", name, v)
	default:
		return 0, fmt.Errorf("%s must be a number, got %T", name, value)
	}
}

// coerceStringArray extracts and sanitizes a []string from a JSON-RPC arg that
// arrives as []interface{}. Non-string entries are dropped; empty and
// whitespace-only strings are filtered. Returns nil for unknown / wrong types
// (callers treat nil and empty slice the same way).
func coerceStringArray(value interface{}) []string {
	list, ok := value.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, v := range list {
		s, ok := v.(string)
		if !ok {
			continue
		}
		if trimmed := strings.TrimSpace(s); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// validatePositiveInt validates that a value is a positive integer.
// Uses coerceToInt to accept various numeric types including strings.
func validatePositiveInt(value interface{}, name string) (int, error) {
	num, err := coerceToInt(value, name)
	if err != nil {
		return 0, err
	}
	if num < 1 {
		return 0, fmt.Errorf("%s must be positive", name)
	}
	return num, nil
}

// validateNonNegativeInt validates that a value is a non-negative integer (0 or greater).
// Used for coordinates like x, y which can be 0.
func validateNonNegativeInt(value interface{}, name string) (int, error) {
	num, err := coerceToInt(value, name)
	if err != nil {
		return 0, err
	}
	if num < 0 {
		return 0, fmt.Errorf("%s must be non-negative", name)
	}
	return num, nil
}

// validateFloatInRange validates that a value is a float within a specified range.
// Used for scale factors, cfg_scale, etc.
func validateFloatInRange(value interface{}, name string, min, max float64) (float64, error) {
	num, err := coerceToFloat(value, name)
	if err != nil {
		return 0, err
	}
	if num < min || num > max {
		return 0, fmt.Errorf("%s must be between %.1f and %.1f", name, min, max)
	}
	return num, nil
}

// validateString validates that a value is a non-empty string
func validateString(value interface{}, name string) (string, error) {
	str, ok := value.(string)
	if !ok || str == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return str, nil
}

// loadImage loads an image from a file
func loadImage(path string) (image.Image, error) {
	return imaging.Open(path)
}

// saveImage saves an image to a file
func saveImage(img image.Image, path string) error {
	// Use internal/imaging package which now supports WebP via nativewebp
	format := filepath.Ext(path)
	format = strings.ToLower(strings.TrimPrefix(format, "."))

	// For WebP, use our custom implementation
	if format == "webp" {
		return gimaging.SaveImageWithFormat(img, path, format)
	}

	// For other formats, use imaging.Save
	return imaging.Save(img, path)
}

// isVertexModel checks if a model is a Vertex AI model
func isVertexModel(model string) bool {
	// Match the user-facing vertex-flash aliases and the vertex/flash-3.1*
	// provider IDs (Gemini 3.1 Flash via Vertex). Bare Gemini model IDs are
	// intentionally not treated as Vertex here because the same ID can be used
	// through the Gemini API; callers should prefer provider resolution when
	// backend identity matters.
	vertexModels := []string{
		"vertex-flash",
		"vertex-flash-fast",
		"vertex-flash-ultra",
		"vertex/flash-3.1",
		"vertex/flash-3.1-fast",
		"vertex/flash-3.1-ultra",
	}
	for _, vm := range vertexModels {
		if model == vm {
			return true
		}
	}
	return false
}

// isGrokModel checks if a model is an xAI Grok model. Delegates to the
// provider registry so this list never drifts from providerAliases.
func isGrokModel(model string) bool {
	api, err := generate.DetectAPIFromModel(model)
	return err == nil && api == "grok"
}

// formatBytes formats bytes as human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
