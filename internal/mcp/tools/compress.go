package tools

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/apresai/gimage/internal/mcp"
	"github.com/apresai/gimage/internal/observability"
	"github.com/disintegration/imaging"
)

// RegisterCompressImageTool registers the compress_image tool
func RegisterCompressImageTool(server *mcp.MCPServer) {
	tool := mcp.Tool{
		Name:        "compress_image",
		Description: "Compress an image to reduce file size while maintaining visual quality. Quality ranges from 1 (lowest quality, smallest file) to 100 (highest quality, largest file). Default is 90 which provides excellent quality with good compression. Most effective on JPEG images. PNG images are compressed losslessly.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type":        "string",
					"description": "Input image file path (absolute or relative path)",
				},
				"quality": map[string]interface{}{
					"type":        "integer",
					"description": "Compression quality (1-100). 90 is recommended for web, 85 for mobile, 75 for thumbnails. Default is 90.",
					"minimum":     1,
					"maximum":     100,
					"default":     90,
				},
				"output": map[string]interface{}{
					"type":        "string",
					"description": "Output file path. If not provided, generates filename like input_compressed.ext",
				},
			},
			"required": []string{"input"},
		},
		Handler: func(args map[string]interface{}) (map[string]interface{}, error) {
			log := observability.NewVerboseLogger(observability.ComponentMCP)
			startTime := time.Now()

			log.Debug("compress_image tool invoked")

			// Validate input
			inputArg, err := validateString(args["input"], "input")
			if err != nil {
				return nil, err
			}
			input, err := ValidateInputPath(inputArg)
			if err != nil {
				return nil, fmt.Errorf("input validation failed: %w", err)
			}

			// Validate quality (default 90)
			quality := 90
			if qualityVal, ok := args["quality"].(float64); ok {
				quality = int(qualityVal)
				if quality < 1 || quality > 100 {
					return nil, fmt.Errorf("quality must be between 1 and 100")
				}
			}

			// Determine output path
			outputArg, _ := args["output"].(string)
			defaultFilename := generateOutputPath(input, "compressed")
			pathResult, pathErr := ValidateAndFixOutputPath(outputArg, defaultFilename)
			if pathErr != nil {
				return nil, fmt.Errorf("output path validation failed: %w", pathErr)
			}
			output := pathResult.Path

			// Get original file size
			originalSize, err := getFileSize(input)
			if err != nil {
				return nil, fmt.Errorf("failed to get original file size: %w", err)
			}

			// Load image
			img, err := loadImage(input)
			if err != nil {
				return nil, fmt.Errorf("failed to load image: %w", err)
			}

			// Save with quality setting
			err = imaging.Save(img, output, imaging.JPEGQuality(quality))
			if err != nil {
				return nil, fmt.Errorf("failed to save compressed image: %w", err)
			}

			// Get new file size
			newSize, err := getFileSize(output)
			if err != nil {
				return nil, fmt.Errorf("failed to get compressed file size: %w", err)
			}

			// Calculate compression ratio
			ratio := float64(newSize) / float64(originalSize)
			savings := originalSize - newSize
			savingsPercent := (1.0 - ratio) * 100

			// Get absolute path for response
			absPath, _ := filepath.Abs(output)

			log.Debug("Compress complete: %.1f%% reduction in %s", savingsPercent, time.Since(startTime))

			result := map[string]interface{}{
				"success":               true,
				"output_path":           absPath,
				"quality":               quality,
				"original_size_bytes":   originalSize,
				"compressed_size_bytes": newSize,
				"compression_ratio":     fmt.Sprintf("%.2f", ratio),
				"savings_bytes":         savings,
				"savings_percent":       fmt.Sprintf("%.1f%%", savingsPercent),
				"original_size_human":   formatBytes(originalSize),
				"compressed_size_human": formatBytes(newSize),
			}
			if pathResult.Warning != "" {
				result["warning"] = pathResult.Warning
			}
			return result, nil
		},
	}

	server.RegisterTool(tool)
}
