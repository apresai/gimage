package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/apresai/gimage/internal/mcp"
	"github.com/apresai/gimage/internal/observability"
	"github.com/disintegration/imaging"
)

// RegisterBatchResizeTool registers the batch_resize tool
func RegisterBatchResizeTool(server *mcp.MCPServer) {
	tool := mcp.Tool{
		Name:        "batch_resize",
		Description: "Resize multiple images in a directory concurrently. Processes all image files (PNG, JPG, WebP, GIF, TIFF, BMP) in the input directory and saves resized versions to the output directory. Uses parallel workers for fast processing of large batches.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input_dir": map[string]interface{}{
					"type":        "string",
					"description": "Input directory containing images to resize",
				},
				"width": map[string]interface{}{
					"type":        "integer",
					"description": "Target width in pixels for all images",
					"minimum":     1,
				},
				"height": map[string]interface{}{
					"type":        "integer",
					"description": "Target height in pixels for all images",
					"minimum":     1,
				},
				"output_dir": map[string]interface{}{
					"type":        "string",
					"description": "Output directory for resized images (will be created if it doesn't exist)",
				},
				"workers": map[string]interface{}{
					"type":        "integer",
					"description": "Number of parallel workers (default: number of CPU cores, max: 16)",
					"minimum":     1,
					"maximum":     16,
					"default":     runtime.NumCPU(),
				},
			},
			"required": []string{"input_dir", "width", "height", "output_dir"},
		},
		Handler: func(args map[string]interface{}) (map[string]interface{}, error) {
			return batchProcessImages(args, "resize")
		},
	}

	server.RegisterTool(tool)
}

// RegisterBatchCompressTool registers the batch_compress tool
func RegisterBatchCompressTool(server *mcp.MCPServer) {
	tool := mcp.Tool{
		Name:        "batch_compress",
		Description: "Compress multiple images in a directory concurrently to reduce file sizes. Processes all image files with specified quality setting. Reports total space saved across all images. Uses parallel workers for efficient processing.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: true,  // Overwrites files in output directory
			IdempotentHint:  true,  // Same inputs produce same outputs
			ReadOnlyHint:    false, // Writes files to disk
		},
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input_dir": map[string]interface{}{
					"type":        "string",
					"description": "Input directory containing images to compress",
				},
				"quality": map[string]interface{}{
					"type":        "integer",
					"description": "Compression quality (1-100). Default is 85 for good balance of quality and size.",
					"minimum":     1,
					"maximum":     100,
					"default":     85,
				},
				"output_dir": map[string]interface{}{
					"type":        "string",
					"description": "Output directory for compressed images (will be created if it doesn't exist)",
				},
				"workers": map[string]interface{}{
					"type":        "integer",
					"description": "Number of parallel workers (default: number of CPU cores, max: 16)",
					"minimum":     1,
					"maximum":     16,
					"default":     runtime.NumCPU(),
				},
			},
			"required": []string{"input_dir", "output_dir"},
		},
		Handler: func(args map[string]interface{}) (map[string]interface{}, error) {
			return batchProcessImages(args, "compress")
		},
	}

	server.RegisterTool(tool)
}

// RegisterBatchConvertTool registers the batch_convert tool
func RegisterBatchConvertTool(server *mcp.MCPServer) {
	tool := mcp.Tool{
		Name:        "batch_convert",
		Description: "Convert multiple images in a directory to a different format concurrently. Useful for converting entire directories to WebP for web optimization, or to PNG for lossless archival. Maintains original filenames with new extensions.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input_dir": map[string]interface{}{
					"type":        "string",
					"description": "Input directory containing images to convert",
				},
				"format": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"png", "jpg", "jpeg", "webp", "gif", "tiff", "bmp"},
					"description": "Target image format for all images",
				},
				"output_dir": map[string]interface{}{
					"type":        "string",
					"description": "Output directory for converted images (will be created if it doesn't exist)",
				},
				"workers": map[string]interface{}{
					"type":        "integer",
					"description": "Number of parallel workers (default: number of CPU cores, max: 16)",
					"minimum":     1,
					"maximum":     16,
					"default":     runtime.NumCPU(),
				},
			},
			"required": []string{"input_dir", "format", "output_dir"},
		},
		Handler: func(args map[string]interface{}) (map[string]interface{}, error) {
			return batchProcessImages(args, "convert")
		},
	}

	server.RegisterTool(tool)
}

func batchProcessImages(args map[string]interface{}, operation string) (map[string]interface{}, error) {
	log := observability.NewVerboseLogger(observability.ComponentMCP)
	startTime := time.Now()

	log.Debug("batch_%s tool invoked", operation)

	// Validate input directory
	inputDirArg, err := validateString(args["input_dir"], "input_dir")
	if err != nil {
		return nil, err
	}
	inputDir, err := ValidateDirectoryPath(inputDirArg, false) // false = don't create if missing
	if err != nil {
		return nil, fmt.Errorf("input directory validation failed: %w", err)
	}

	// Validate output directory
	outputDirArg, err := validateString(args["output_dir"], "output_dir")
	if err != nil {
		return nil, err
	}
	outputDir, err := ValidateDirectoryPath(outputDirArg, true) // true = create if missing
	if err != nil {
		return nil, fmt.Errorf("output directory validation failed: %w", err)
	}

	// Determine number of workers
	workers := runtime.NumCPU()
	if workersVal, ok := args["workers"].(float64); ok {
		workers = int(workersVal)
		if workers < 1 {
			workers = 1
		} else if workers > 16 {
			workers = 16
		}
	}

	// Find all image files
	imageExtensions := []string{".png", ".jpg", ".jpeg", ".webp", ".gif", ".tiff", ".bmp"}
	var files []string

	err = filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		for _, validExt := range imageExtensions {
			if ext == validExt || ext == "."+validExt {
				files = append(files, path)
				break
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan input directory: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no image files found in %s", inputDir)
	}

	log.Debug("Found %d image files to process with %d workers", len(files), workers)

	// Process images concurrently
	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)
	var mu sync.Mutex
	processed := 0
	failed := 0
	var errors []string
	var totalOriginalSize int64
	var totalNewSize int64

	for _, file := range files {
		wg.Add(1)
		go func(inputPath string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Determine output path
			relPath, _ := filepath.Rel(inputDir, inputPath)
			var outputPath string

			if operation == "convert" {
				format, _ := args["format"].(string)
				ext := filepath.Ext(relPath)
				base := relPath[:len(relPath)-len(ext)]
				outputPath = filepath.Join(outputDir, base+"."+format)
			} else {
				outputPath = filepath.Join(outputDir, relPath)
			}

			// Ensure output subdirectory exists
			outputSubdir := filepath.Dir(outputPath)
			os.MkdirAll(outputSubdir, 0755)

			// Process based on operation
			var err error
			switch operation {
			case "resize":
				width, _ := validatePositiveInt(args["width"], "width")
				height, _ := validatePositiveInt(args["height"], "height")
				err = processResize(inputPath, outputPath, width, height)

			case "compress":
				quality := 85
				if qualityVal, ok := args["quality"].(float64); ok {
					quality = int(qualityVal)
				}
				err = processCompress(inputPath, outputPath, quality)
				if err == nil {
					// Track savings
					origSize, _ := getFileSize(inputPath)
					newSize, _ := getFileSize(outputPath)
					mu.Lock()
					totalOriginalSize += origSize
					totalNewSize += newSize
					mu.Unlock()
				}

			case "convert":
				err = processConvert(inputPath, outputPath)
			}

			mu.Lock()
			if err != nil {
				failed++
				errors = append(errors, fmt.Sprintf("%s: %v", filepath.Base(inputPath), err))
			} else {
				processed++
			}
			mu.Unlock()
		}(file)
	}

	wg.Wait()

	log.Debug("Batch %s complete: %d processed, %d failed in %s", operation, processed, failed, time.Since(startTime))

	result := map[string]interface{}{
		"success":    failed == 0,
		"processed":  processed,
		"failed":     failed,
		"total":      len(files),
		"output_dir": outputDir,
	}

	if len(errors) > 0 {
		result["errors"] = errors
	}

	if operation == "compress" && totalOriginalSize > 0 {
		savings := totalOriginalSize - totalNewSize
		savingsPercent := (float64(savings) / float64(totalOriginalSize)) * 100
		result["total_original_size"] = formatBytes(totalOriginalSize)
		result["total_new_size"] = formatBytes(totalNewSize)
		result["total_savings"] = formatBytes(savings)
		result["savings_percent"] = fmt.Sprintf("%.1f%%", savingsPercent)
	}

	return result, nil
}

func processResize(input, output string, width, height int) error {
	img, err := loadImage(input)
	if err != nil {
		return err
	}
	resized := imaging.Resize(img, width, height, imaging.Lanczos)
	return saveImage(resized, output)
}

func processCompress(input, output string, quality int) error {
	img, err := loadImage(input)
	if err != nil {
		return err
	}
	return imaging.Save(img, output, imaging.JPEGQuality(quality))
}

func processConvert(input, output string) error {
	img, err := loadImage(input)
	if err != nil {
		return err
	}
	return saveImage(img, output)
}
