package tools

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/apresai/gimage/internal/mcp"
	"github.com/apresai/gimage/internal/observability"
)

// RegisterConvertImageTool registers the convert_image tool
func RegisterConvertImageTool(server *mcp.MCPServer) {
	tool := mcp.Tool{
		Name:        "convert_image",
		Description: "Convert an image between different formats (PNG, JPG/JPEG, WebP, GIF, TIFF, BMP). Useful for web optimization (converting to WebP), compatibility (PNG to JPG), or specific application requirements. Format detection is automatic based on file extension.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type":        "string",
					"description": "Input image file path (absolute or relative path)",
				},
				"format": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"png", "jpg", "jpeg", "webp", "gif", "tiff", "bmp"},
					"description": "Target image format. WebP recommended for web, PNG for lossless, JPG for photos.",
				},
				"output": map[string]interface{}{
					"type":        "string",
					"description": "Output file path. If not provided, generates filename with new extension (e.g., image.webp)",
				},
			},
			"required": []string{"input", "format"},
		},
		Handler: func(args map[string]interface{}) (map[string]interface{}, error) {
			log := observability.NewVerboseLogger(observability.ComponentMCP)
			startTime := time.Now()

			log.Debug("convert_image tool invoked")

			// Validate input
			inputArg, err := validateString(args["input"], "input")
			if err != nil {
				return nil, err
			}
			input, err := ValidateInputPath(inputArg)
			if err != nil {
				return nil, fmt.Errorf("input validation failed: %w", err)
			}

			// Validate format
			format, err := validateString(args["format"], "format")
			if err != nil {
				return nil, err
			}

			// Normalize format
			format = strings.ToLower(format)
			validFormats := map[string]bool{
				"png": true, "jpg": true, "jpeg": true,
				"webp": true, "gif": true, "tiff": true, "bmp": true,
			}
			if !validFormats[format] {
				return nil, fmt.Errorf("invalid format: %s (must be one of: png, jpg, jpeg, webp, gif, tiff, bmp)", format)
			}

			// Determine output path
			outputArg, _ := args["output"].(string)
			// Generate default filename with new extension
			ext := filepath.Ext(input)
			base := strings.TrimSuffix(input, ext)
			targetFormat := format
			if targetFormat == "jpeg" {
				targetFormat = "jpg" // Use .jpg extension for JPEG
			}
			defaultFilename := base + "." + targetFormat
			pathResult, pathErr := ValidateAndFixOutputPath(outputArg, defaultFilename)
			if pathErr != nil {
				return nil, fmt.Errorf("output path validation failed: %w", pathErr)
			}
			output := pathResult.Path

			// Get original format
			originalExt := filepath.Ext(input)
			originalFormat := strings.TrimPrefix(strings.ToLower(originalExt), ".")

			// Load image
			img, err := loadImage(input)
			if err != nil {
				return nil, fmt.Errorf("failed to load image: %w", err)
			}

			// Save in new format
			err = saveImage(img, output)
			if err != nil {
				return nil, fmt.Errorf("failed to save converted image: %w", err)
			}

			// Get file sizes
			originalSize, _ := getFileSize(input)
			newSize, _ := getFileSize(output)

			// Get absolute path for response
			absPath, _ := filepath.Abs(output)

			log.Debug("Convert complete: %s -> %s in %s", originalFormat, format, time.Since(startTime))

			result := map[string]interface{}{
				"success":         true,
				"output_path":     absPath,
				"original_format": originalFormat,
				"new_format":      format,
				"original_size":   formatBytes(originalSize),
				"new_size":        formatBytes(newSize),
			}
			if pathResult.Warning != "" {
				result["warning"] = pathResult.Warning
			}
			return result, nil
		},
	}

	server.RegisterTool(tool)
}
