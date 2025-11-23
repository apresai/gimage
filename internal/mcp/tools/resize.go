package tools

import (
	"fmt"
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
		Description: "Resize an image to specific dimensions using high-quality Lanczos resampling. This changes both width and height to exact pixel values. Note: Aspect ratio is NOT preserved unless dimensions match original ratio. Use scale_image if you want to maintain aspect ratio.",
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

			// Resize image using Lanczos resampling
			resized := imaging.Resize(img, width, height, imaging.Lanczos)

			// Save resized image
			err = saveImage(resized, output)
			if err != nil {
				return nil, fmt.Errorf("failed to save resized image: %w", err)
			}

			// Get absolute path for response
			absPath, _ := filepath.Abs(output)

			log.Debug("Resize complete: %dx%d -> %dx%d in %s", origWidth, origHeight, width, height, time.Since(startTime))

			result := map[string]interface{}{
				"success":       true,
				"output_path":   absPath,
				"original_size": fmt.Sprintf("%dx%d", origWidth, origHeight),
				"new_size":      fmt.Sprintf("%dx%d", width, height),
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
