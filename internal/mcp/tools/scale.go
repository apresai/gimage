package tools

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/apresai/gimage/internal/mcp"
	"github.com/apresai/gimage/internal/observability"
	"github.com/disintegration/imaging"
)

// RegisterScaleImageTool registers the scale_image tool
func RegisterScaleImageTool(server *mcp.MCPServer) {
	tool := mcp.Tool{
		Name:        "scale_image",
		Description: "Scale an image by a factor while preserving aspect ratio. Use this when you want to make an image larger or smaller proportionally. For example, factor 0.5 makes image half size, factor 2.0 makes it double size. Uses high-quality Lanczos resampling.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type":        "string",
					"description": "Input image file path (absolute or relative path)",
				},
				"factor": map[string]interface{}{
					"type":        "number",
					"description": "Scale factor (0.1 to 10.0). Examples: 0.5 = half size, 2.0 = double size, 0.25 = quarter size",
					"minimum":     0.1,
					"maximum":     10.0,
				},
				"output": map[string]interface{}{
					"type":        "string",
					"description": "Output file path. If not provided, generates filename like input_scaled.ext",
				},
			},
			"required": []string{"input", "factor"},
		},
		Handler: func(args map[string]interface{}) (map[string]interface{}, error) {
			log := observability.NewVerboseLogger(observability.ComponentMCP)
			startTime := time.Now()

			log.Debug("scale_image tool invoked")

			// Validate input file path
			inputArg, err := validateString(args["input"], "input")
			if err != nil {
				return nil, err
			}
			input, err := ValidateInputPath(inputArg)
			if err != nil {
				return nil, fmt.Errorf("input validation failed: %w", err)
			}

			// Validate factor (accepts string or number, must be 0.1-10.0)
			factorVal, err := validateFloatInRange(args["factor"], "factor", 0.1, 10.0)
			if err != nil {
				return nil, err
			}

			// Validate and fix output path
			outputArg, _ := args["output"].(string)
			defaultFilename := generateOutputPath(input, "scaled")
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

			// Calculate new dimensions
			newWidth := int(float64(origWidth) * factorVal)
			newHeight := int(float64(origHeight) * factorVal)

			if newWidth < 1 || newHeight < 1 {
				return nil, fmt.Errorf("resulting dimensions would be too small (less than 1 pixel)")
			}

			// Load image
			img, err := loadImage(input)
			if err != nil {
				return nil, fmt.Errorf("failed to load image: %w", err)
			}

			// Resize image using Lanczos resampling
			scaled := imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)

			// Save scaled image
			err = saveImage(scaled, output)
			if err != nil {
				return nil, fmt.Errorf("failed to save scaled image: %w", err)
			}

			// Get absolute path for response
			absPath, _ := filepath.Abs(output)

			log.Debug("Scale complete: %dx%d -> %dx%d (%.1fx) in %s", origWidth, origHeight, newWidth, newHeight, factorVal, time.Since(startTime))

			result := map[string]interface{}{
				"success":       true,
				"output_path":   absPath,
				"scale_factor":  factorVal,
				"original_size": fmt.Sprintf("%dx%d", origWidth, origHeight),
				"new_size":      fmt.Sprintf("%dx%d", newWidth, newHeight),
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
