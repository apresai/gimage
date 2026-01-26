package tools

import (
	"fmt"
	"image"
	"path/filepath"
	"time"

	"github.com/apresai/gimage/internal/mcp"
	"github.com/apresai/gimage/internal/observability"
	"github.com/disintegration/imaging"
)

// RegisterResizeImageTool registers the resize_image tool
func RegisterResizeImageTool(server *mcp.MCPServer) {
	tool := mcp.Tool{
		Name:        "resize_image",
		Description: "Resize an image to specific dimensions using high-quality Lanczos resampling. Supports three modes: 'crop' (default) preserves aspect ratio by filling the target size and cropping excess, 'fit' preserves aspect ratio by fitting within target bounds (may be smaller), 'stretch' forces exact dimensions (may distort). Use 'crop' for thumbnails/avatars, 'fit' for constrained display, 'stretch' for exact sizing.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type":        "string",
					"description": "Input image file path (absolute or relative path)",
				},
				"width": map[string]interface{}{
					"type":        "integer",
					"description": "Target width in pixels (must be positive integer)",
					"minimum":     1,
				},
				"height": map[string]interface{}{
					"type":        "integer",
					"description": "Target height in pixels (must be positive integer)",
					"minimum":     1,
				},
				"mode": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"crop", "fit", "stretch"},
					"description": "Resize mode: 'crop' (default) preserves aspect ratio by filling and cropping excess from center, 'fit' preserves aspect ratio by fitting within bounds (result may be smaller than target), 'stretch' forces exact dimensions (may distort image)",
					"default":     "crop",
				},
				"output": map[string]interface{}{
					"type":        "string",
					"description": "Output file path. If not provided, generates filename like input_resized.ext",
				},
			},
			"required": []string{"input", "width", "height"},
		},
		Handler: func(args map[string]interface{}) (map[string]interface{}, error) {
			log := observability.NewVerboseLogger(observability.ComponentMCP)
			startTime := time.Now()

			log.Debug("resize_image tool invoked")

			// Validate input file path
			inputArg, err := validateString(args["input"], "input")
			if err != nil {
				return nil, err
			}
			input, err := ValidateInputPath(inputArg)
			if err != nil {
				return nil, fmt.Errorf("input validation failed: %w", err)
			}

			// Validate dimensions
			width, err := validatePositiveInt(args["width"], "width")
			if err != nil {
				return nil, err
			}

			height, err := validatePositiveInt(args["height"], "height")
			if err != nil {
				return nil, err
			}

			// Extract mode (default: crop for aspect-ratio preservation)
			mode := "crop"
			if modeArg, ok := args["mode"].(string); ok && modeArg != "" {
				switch modeArg {
				case "crop", "fit", "stretch":
					mode = modeArg
				default:
					return nil, fmt.Errorf("invalid mode: %s (must be crop, fit, or stretch)", modeArg)
				}
			}

			// Validate and fix output path
			outputArg, _ := args["output"].(string)
			defaultFilename := generateOutputPath(input, "resized")
			pathResult, pathErr := ValidateAndFixOutputPath(outputArg, defaultFilename)
			if pathErr != nil {
				return nil, fmt.Errorf("output path validation failed: %w", pathErr)
			}
			output := pathResult.Path

			// Get original dimensions
			origWidth, origHeight, err := getImageDimensions(input)
			if err != nil {
				return nil, fmt.Errorf("failed to read input image: %w", err)
			}

			// Load image
			img, err := loadImage(input)
			if err != nil {
				return nil, fmt.Errorf("failed to load image: %w", err)
			}

			// Resize image based on mode
			var resized image.Image
			switch mode {
			case "fit":
				// Fit within bounds, preserving aspect ratio (may be smaller than target)
				resized = imaging.Fit(img, width, height, imaging.Lanczos)
			case "stretch":
				// Force exact dimensions (may distort - backward compatible with old behavior)
				resized = imaging.Resize(img, width, height, imaging.Lanczos)
			default: // "crop"
				// Fill target size, preserving aspect ratio, crop excess from center
				resized = imaging.Fill(img, width, height, imaging.Center, imaging.Lanczos)
			}

			// Save resized image
			err = saveImage(resized, output)
			if err != nil {
				return nil, fmt.Errorf("failed to save resized image: %w", err)
			}

			// Get actual output dimensions (may differ from target in 'fit' mode)
			outputBounds := resized.Bounds()
			actualWidth := outputBounds.Dx()
			actualHeight := outputBounds.Dy()

			// Get absolute path for response
			absPath, _ := filepath.Abs(output)

			log.Debug("Resize complete: %dx%d -> %dx%d (mode=%s) in %s", origWidth, origHeight, actualWidth, actualHeight, mode, time.Since(startTime))

			result := map[string]interface{}{
				"success":       true,
				"output_path":   absPath,
				"original_size": fmt.Sprintf("%dx%d", origWidth, origHeight),
				"new_size":      fmt.Sprintf("%dx%d", actualWidth, actualHeight),
				"mode":          mode,
			}

			// Add warning if path was adjusted
			if pathResult.Warning != "" {
				result["warning"] = pathResult.Warning
			}

			return result, nil
		},
	}

	server.RegisterTool(tool)
}
