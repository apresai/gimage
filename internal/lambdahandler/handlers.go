package lambdahandler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/apresai/gimage/internal/config"
	"github.com/apresai/gimage/internal/generate"
	gimageimaging "github.com/apresai/gimage/internal/imaging"
	"github.com/apresai/gimage/internal/observability"
	"github.com/apresai/gimage/pkg/models"
	"github.com/aws/aws-lambda-go/events"
	"github.com/disintegration/imaging"
)

// handleGenerate handles AI image generation requests
func (h *Handler) handleGenerate(ctx context.Context, body []byte) (events.APIGatewayProxyResponse, error) {
	var req GenerateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return errorResponse(400, fmt.Sprintf("Invalid request body: %v", err)), nil
	}

	// Validate prompt
	if req.Prompt == "" {
		return errorResponse(400, "Prompt is required"), nil
	}

	// Build generate options
	options := models.GenerateOptions{
		Model:          req.Model,
		Size:           req.Size,
		Style:          req.Style,
		NegativePrompt: req.NegativePrompt,
		Seed:           req.Seed,
	}

	// Set defaults
	if options.Model == "" {
		options.Model = generate.DefaultModel
	}
	if options.Size == "" {
		options.Size = "1024x1024"
	}

	log := observability.NewVerboseLogger(observability.ComponentLambda)
	log.Info("Generating image with prompt: %s, model: %s", req.Prompt, options.Model)

	// Determine API to use
	api, err := generate.DetectAPIFromModel(options.Model)
	if err != nil {
		return errorResponse(400, fmt.Sprintf("Invalid model: %v", err)), nil
	}

	var generatedImages []*models.GeneratedImage

	// Generate based on API
	if api == "gemini" {
		key, err := config.GetGeminiAPIKey("")
		if err != nil {
			return errorResponse(500, fmt.Sprintf("Gemini API key not configured: %v", err)), nil
		}

		client, err := generate.NewGeminiRESTClient(key)
		if err != nil {
			return errorResponse(500, fmt.Sprintf("Failed to create Gemini client: %v", err)), nil
		}
		defer client.Close()

		generatedImages, err = client.GenerateImage(ctx, req.Prompt, options)
		if err != nil {
			return errorResponse(500, fmt.Sprintf("Failed to generate image: %v", err)), nil
		}
	} else if api == "vertex" {
		vertexAPIKey, _ := config.GetVertexAPIKey("")

		// Load project and location from config or env
		project := os.Getenv("VERTEX_PROJECT")
		location := os.Getenv("VERTEX_LOCATION")
		if location == "" {
			location = "us-central1"
		}

		if vertexAPIKey != "" {
			// Express Mode - REST client with API key
			client, err := generate.NewVertexRESTClient(vertexAPIKey, project, location)
			if err != nil {
				return errorResponse(500, fmt.Sprintf("Failed to create Vertex REST client: %v", err)), nil
			}
			defer client.Close()

			generatedImages, err = client.GenerateImage(ctx, req.Prompt, options)
			if err != nil {
				return errorResponse(500, fmt.Sprintf("Failed to generate image: %v", err)), nil
			}
		} else {
			// Full Mode - Unified SDK client with ADC/service account
			client, err := generate.NewVertexUnifiedClient(ctx, project, location)
			if err != nil {
				return errorResponse(500, fmt.Sprintf("Failed to create Vertex unified client: %v", err)), nil
			}
			defer client.Close()

			generatedImages, err = client.GenerateImage(ctx, req.Prompt, options)
			if err != nil {
				return errorResponse(500, fmt.Sprintf("Failed to generate image: %v", err)), nil
			}
		}
	} else {
		return errorResponse(400, fmt.Sprintf("Unsupported API: %s", api)), nil
	}

	if len(generatedImages) == 0 {
		return errorResponse(500, "No images generated"), nil
	}
	generatedImage := generatedImages[0]

	// Create response
	return h.createImageResponse(ctx, generatedImage.Data, generatedImage.Format, generatedImage.Width, generatedImage.Height, req.ResponseFormat)
}

// handleResize handles image resize requests
func (h *Handler) handleResize(ctx context.Context, body []byte) (events.APIGatewayProxyResponse, error) {
	var req ResizeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return errorResponse(400, fmt.Sprintf("Invalid request body: %v", err)), nil
	}

	// Validate inputs
	if req.Image == "" {
		return errorResponse(400, "Image is required"), nil
	}
	if req.Width <= 0 || req.Height <= 0 {
		return errorResponse(400, "Width and height must be positive"), nil
	}

	// Load image
	imageData, err := LoadImageFromInput(ctx, h.s3Client, req.Image)
	if err != nil {
		return errorResponse(400, fmt.Sprintf("Failed to load image: %v", err)), nil
	}

	// Decode image
	img, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return errorResponse(400, fmt.Sprintf("Failed to decode image: %v", err)), nil
	}

	// Resize using Lanczos resampling
	resized := imaging.Resize(img, req.Width, req.Height, imaging.Lanczos)

	// Encode to bytes
	outputData, err := encodeImage(resized, format)
	if err != nil {
		return errorResponse(500, fmt.Sprintf("Failed to encode image: %v", err)), nil
	}

	return h.createImageResponse(ctx, outputData, format, req.Width, req.Height, req.ResponseFormat)
}

// handleScale handles image scaling requests
func (h *Handler) handleScale(ctx context.Context, body []byte) (events.APIGatewayProxyResponse, error) {
	var req ScaleRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return errorResponse(400, fmt.Sprintf("Invalid request body: %v", err)), nil
	}

	// Validate inputs
	if req.Image == "" {
		return errorResponse(400, "Image is required"), nil
	}
	if req.Factor <= 0 {
		return errorResponse(400, "Factor must be positive"), nil
	}

	// Load image
	imageData, err := LoadImageFromInput(ctx, h.s3Client, req.Image)
	if err != nil {
		return errorResponse(400, fmt.Sprintf("Failed to load image: %v", err)), nil
	}

	// Decode image
	img, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return errorResponse(400, fmt.Sprintf("Failed to decode image: %v", err)), nil
	}

	// Calculate new dimensions
	bounds := img.Bounds()
	newWidth := int(float64(bounds.Dx()) * req.Factor)
	newHeight := int(float64(bounds.Dy()) * req.Factor)

	// Scale
	scaled := imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)

	// Encode
	outputData, err := encodeImage(scaled, format)
	if err != nil {
		return errorResponse(500, fmt.Sprintf("Failed to encode image: %v", err)), nil
	}

	return h.createImageResponse(ctx, outputData, format, newWidth, newHeight, req.ResponseFormat)
}

// handleCrop handles image cropping requests
func (h *Handler) handleCrop(ctx context.Context, body []byte) (events.APIGatewayProxyResponse, error) {
	var req CropRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return errorResponse(400, fmt.Sprintf("Invalid request body: %v", err)), nil
	}

	// Validate inputs
	if req.Image == "" {
		return errorResponse(400, "Image is required"), nil
	}
	if req.Width <= 0 || req.Height <= 0 {
		return errorResponse(400, "Width and height must be positive"), nil
	}

	// Load image
	imageData, err := LoadImageFromInput(ctx, h.s3Client, req.Image)
	if err != nil {
		return errorResponse(400, fmt.Sprintf("Failed to load image: %v", err)), nil
	}

	// Decode image
	img, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return errorResponse(400, fmt.Sprintf("Failed to decode image: %v", err)), nil
	}

	// Validate crop region
	bounds := img.Bounds()
	if req.X < 0 || req.Y < 0 {
		return errorResponse(400, "X and Y coordinates must be non-negative"), nil
	}
	if req.X >= bounds.Dx() || req.Y >= bounds.Dy() {
		return errorResponse(400, "Crop region is outside image bounds"), nil
	}
	if req.X+req.Width > bounds.Dx() || req.Y+req.Height > bounds.Dy() {
		return errorResponse(400, "Crop region exceeds image dimensions"), nil
	}

	// Create crop rectangle
	cropRect := image.Rect(req.X, req.Y, req.X+req.Width, req.Y+req.Height)

	// Perform crop
	cropped := imaging.Crop(img, cropRect)

	// Encode
	outputData, err := encodeImage(cropped, format)
	if err != nil {
		return errorResponse(500, fmt.Sprintf("Failed to encode image: %v", err)), nil
	}

	return h.createImageResponse(ctx, outputData, format, req.Width, req.Height, req.ResponseFormat)
}

// handleCompress handles image compression requests
func (h *Handler) handleCompress(ctx context.Context, body []byte) (events.APIGatewayProxyResponse, error) {
	var req CompressRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return errorResponse(400, fmt.Sprintf("Invalid request body: %v", err)), nil
	}

	// Validate inputs
	if req.Image == "" {
		return errorResponse(400, "Image is required"), nil
	}

	// Set defaults
	if req.Quality == 0 {
		req.Quality = 85
	}
	if req.Quality < 1 || req.Quality > 100 {
		return errorResponse(400, "Quality must be between 1 and 100"), nil
	}

	// Load image
	imageData, err := LoadImageFromInput(ctx, h.s3Client, req.Image)
	if err != nil {
		return errorResponse(400, fmt.Sprintf("Failed to load image: %v", err)), nil
	}

	// Decode
	img, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return errorResponse(400, fmt.Sprintf("Failed to decode image: %v", err)), nil
	}

	// Use target format if specified
	targetFormat := format
	if req.Format != "" {
		targetFormat = req.Format
	}

	// Encode with quality (for JPEG/WebP)
	outputData, err := encodeImageWithQuality(img, targetFormat, req.Quality)
	if err != nil {
		return errorResponse(500, fmt.Sprintf("Failed to compress image: %v", err)), nil
	}

	bounds := img.Bounds()
	return h.createImageResponse(ctx, outputData, targetFormat, bounds.Dx(), bounds.Dy(), req.ResponseFormat)
}

// handleConvert handles image format conversion requests
func (h *Handler) handleConvert(ctx context.Context, body []byte) (events.APIGatewayProxyResponse, error) {
	var req ConvertRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return errorResponse(400, fmt.Sprintf("Invalid request body: %v", err)), nil
	}

	// Validate inputs
	if req.Image == "" {
		return errorResponse(400, "Image is required"), nil
	}
	if req.TargetFormat == "" {
		return errorResponse(400, "Target format is required"), nil
	}

	// Load image
	imageData, err := LoadImageFromInput(ctx, h.s3Client, req.Image)
	if err != nil {
		return errorResponse(400, fmt.Sprintf("Failed to load image: %v", err)), nil
	}

	// Convert using existing function
	convertedData, err := gimageimaging.ConvertImageData(imageData, req.TargetFormat)
	if err != nil {
		return errorResponse(400, fmt.Sprintf("Failed to convert image: %v", err)), nil
	}

	// Get dimensions
	img, _, err := image.Decode(bytes.NewReader(convertedData))
	if err != nil {
		return errorResponse(500, "Failed to decode converted image"), nil
	}

	bounds := img.Bounds()
	return h.createImageResponse(ctx, convertedData, req.TargetFormat, bounds.Dx(), bounds.Dy(), req.ResponseFormat)
}

// handleBatch handles batch processing requests
func (h *Handler) handleBatch(ctx context.Context, body []byte) (events.APIGatewayProxyResponse, error) {
	var req BatchRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return errorResponse(400, fmt.Sprintf("Invalid request body: %v", err)), nil
	}

	if len(req.Operations) == 0 {
		return errorResponse(400, "At least one operation is required"), nil
	}

	// Process operations concurrently
	results := make([]ImageResponse, len(req.Operations))
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, len(req.Operations))

	for i, op := range req.Operations {
		wg.Add(1)
		go func(idx int, operation BatchOperation) {
			defer wg.Done()

			// Process based on operation type
			var result ImageResponse
			var err error

			switch operation.Operation {
			case "resize":
				result, err = h.processBatchResize(ctx, operation)
			case "scale":
				result, err = h.processBatchScale(ctx, operation)
			case "crop":
				result, err = h.processBatchCrop(ctx, operation)
			case "compress":
				result, err = h.processBatchCompress(ctx, operation)
			case "convert":
				result, err = h.processBatchConvert(ctx, operation)
			default:
				err = fmt.Errorf("unknown operation: %s", operation.Operation)
			}

			mu.Lock()
			if err != nil {
				errors[idx] = err
			} else {
				results[idx] = result
			}
			mu.Unlock()
		}(i, op)
	}

	wg.Wait()

	// Check for errors
	for i, err := range errors {
		if err != nil {
			return errorResponse(500, fmt.Sprintf("Operation %d failed: %v", i, err)), nil
		}
	}

	// Return batch response
	batchResp := BatchResponse{
		BatchID: fmt.Sprintf("batch-%d", time.Now().Unix()),
		Status:  "completed",
		Results: results,
	}

	return successResponse(200, batchResp), nil
}

// handleHealth handles health check requests
func (h *Handler) handleHealth(ctx context.Context) (events.APIGatewayProxyResponse, error) {
	health := HealthResponse{
		Status:  "healthy",
		Version: "1.2.66",
		APIs:    make(map[string]string),
	}

	// Check Gemini API
	if config.HasGeminiCredentials() {
		health.APIs["gemini"] = "available"
	} else {
		health.APIs["gemini"] = "not_configured"
	}

	// Check Vertex API
	if config.HasVertexCredentials() {
		health.APIs["vertex"] = "available"
	} else {
		health.APIs["vertex"] = "not_configured"
	}

	return successResponse(200, health), nil
}

// Helper functions

func (h *Handler) createImageResponse(ctx context.Context, data []byte, format string, width, height int, requestedFormat string) (events.APIGatewayProxyResponse, error) {
	responseFormat := DetermineResponseFormat(int64(len(data)), requestedFormat)

	resp := ImageResponse{
		Width:     width,
		Height:    height,
		Format:    format,
		SizeBytes: int64(len(data)),
	}

	if responseFormat == "base64" {
		// Return base64 encoded
		resp.Image = EncodeImageToBase64(data)
	} else {
		// Upload to S3 and return presigned URL
		s3Key := GenerateS3Key(format)
		contentType := GetContentType(format)

		if err := h.s3Client.Upload(ctx, s3Key, data, contentType); err != nil {
			return errorResponse(500, fmt.Sprintf("Failed to upload to S3: %v", err)), nil
		}

		presignedURL, err := h.s3Client.GeneratePresignedURL(ctx, s3Key, GetPresignedURLExpiration())
		if err != nil {
			return errorResponse(500, fmt.Sprintf("Failed to generate presigned URL: %v", err)), nil
		}

		resp.S3URL = presignedURL
		resp.S3Key = s3Key
	}

	return successResponse(200, resp), nil
}

// Batch processing helpers

func (h *Handler) processBatchResize(ctx context.Context, op BatchOperation) (ImageResponse, error) {
	width, _ := op.Params["width"].(float64)
	height, _ := op.Params["height"].(float64)

	req := ResizeRequest{
		Image:          op.Image,
		Width:          int(width),
		Height:         int(height),
		ResponseFormat: "s3_url",
	}

	bodyBytes, _ := json.Marshal(req)
	resp, err := h.handleResize(ctx, bodyBytes)
	if err != nil {
		return ImageResponse{}, err
	}

	var imgResp ImageResponse
	json.Unmarshal([]byte(resp.Body), &imgResp)
	return imgResp, nil
}

func (h *Handler) processBatchScale(ctx context.Context, op BatchOperation) (ImageResponse, error) {
	factor, _ := op.Params["factor"].(float64)

	req := ScaleRequest{
		Image:          op.Image,
		Factor:         factor,
		ResponseFormat: "s3_url",
	}

	bodyBytes, _ := json.Marshal(req)
	resp, err := h.handleScale(ctx, bodyBytes)
	if err != nil {
		return ImageResponse{}, err
	}

	var imgResp ImageResponse
	json.Unmarshal([]byte(resp.Body), &imgResp)
	return imgResp, nil
}

func (h *Handler) processBatchCrop(ctx context.Context, op BatchOperation) (ImageResponse, error) {
	x, _ := op.Params["x"].(float64)
	y, _ := op.Params["y"].(float64)
	width, _ := op.Params["width"].(float64)
	height, _ := op.Params["height"].(float64)

	req := CropRequest{
		Image:          op.Image,
		X:              int(x),
		Y:              int(y),
		Width:          int(width),
		Height:         int(height),
		ResponseFormat: "s3_url",
	}

	bodyBytes, _ := json.Marshal(req)
	resp, err := h.handleCrop(ctx, bodyBytes)
	if err != nil {
		return ImageResponse{}, err
	}

	var imgResp ImageResponse
	json.Unmarshal([]byte(resp.Body), &imgResp)
	return imgResp, nil
}

func (h *Handler) processBatchCompress(ctx context.Context, op BatchOperation) (ImageResponse, error) {
	quality, _ := op.Params["quality"].(float64)
	format, _ := op.Params["format"].(string)

	req := CompressRequest{
		Image:          op.Image,
		Quality:        int(quality),
		Format:         format,
		ResponseFormat: "s3_url",
	}

	bodyBytes, _ := json.Marshal(req)
	resp, err := h.handleCompress(ctx, bodyBytes)
	if err != nil {
		return ImageResponse{}, err
	}

	var imgResp ImageResponse
	json.Unmarshal([]byte(resp.Body), &imgResp)
	return imgResp, nil
}

func (h *Handler) processBatchConvert(ctx context.Context, op BatchOperation) (ImageResponse, error) {
	targetFormat, _ := op.Params["target_format"].(string)

	req := ConvertRequest{
		Image:          op.Image,
		TargetFormat:   targetFormat,
		ResponseFormat: "s3_url",
	}

	bodyBytes, _ := json.Marshal(req)
	resp, err := h.handleConvert(ctx, bodyBytes)
	if err != nil {
		return ImageResponse{}, err
	}

	var imgResp ImageResponse
	json.Unmarshal([]byte(resp.Body), &imgResp)
	return imgResp, nil
}

// Image encoding helpers

func encodeImage(img image.Image, format string) ([]byte, error) {
	return encodeImageWithQuality(img, format, 90)
}

func encodeImageWithQuality(img image.Image, format string, quality int) ([]byte, error) {
	var buf bytes.Buffer

	switch strings.ToLower(format) {
	case "jpg", "jpeg":
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, err
		}
	case "png":
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
	case "gif":
		if err := gif.Encode(&buf, img, nil); err != nil {
			return nil, err
		}
	default:
		// Default to PNG
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}
