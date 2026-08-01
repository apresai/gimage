// Package generate provides image generation clients for various AI providers.
package generate

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/apresai/gimage/pkg/models"
	"github.com/sony/gobreaker"
)

const (
	grokBaseURL         = "https://api.x.ai/v1"
	grokDefaultModel    = "grok-imagine-image"
	grokMaxInputImages  = 3
	grokGenerationsPath = "/images/generations"
	grokEditsPath       = "/images/edits"
)

// GrokClient handles interactions with the xAI Grok API for image generation
type GrokClient struct {
	apiKey         string
	httpClient     *http.Client
	circuitBreaker *gobreaker.CircuitBreaker
}

// GrokImageSource is a single input image for the edits endpoint (URL or data URI).
type GrokImageSource struct {
	URL  string `json:"url"`
	Type string `json:"type,omitempty"` // "image_url" for compatibility with xAI examples
}

// GrokImageRequest represents the request body for Grok image generation or editing.
// When Image/Images are set, the client POSTs to /v1/images/edits instead of /generations.
type GrokImageRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              int    `json:"n,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	AspectRatio    string `json:"aspect_ratio,omitempty"`
	// Resolution is the xAI native resolution: "1k" or "2k". Only honored by grok-imagine-* models.
	Resolution string `json:"resolution,omitempty"`
	// User is an optional end-user id for abuse monitoring (xAI REST field).
	User string `json:"user,omitempty"`
	// Image is used for single-image edits (mutually exclusive with Images).
	Image *GrokImageSource `json:"image,omitempty"`
	// Images is used for multi-reference edits (up to grokMaxInputImages). Reference as
	// <IMAGE_0>, <IMAGE_1>, … in the prompt when needed.
	Images []GrokImageSource `json:"images,omitempty"`
}

// GrokImageResponse represents the response from Grok image generation
type GrokImageResponse struct {
	Created int64            `json:"created"`
	Data    []GrokImageData  `json:"data"`
	Error   *GrokErrorDetail `json:"error,omitempty"`
	Usage   *GrokUsage       `json:"usage,omitempty"`
}

// GrokUsage holds cost metadata returned by the xAI images API.
type GrokUsage struct {
	CostInUSDTicks int64 `json:"cost_in_usd_ticks"`
}

// GrokImageData contains the generated image data
type GrokImageData struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
	MimeType      string `json:"mime_type,omitempty"`
}

// GrokErrorDetail contains error information from the API
type GrokErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

// NewGrokClient creates a new xAI Grok API client
func NewGrokClient(apiKey string) (*GrokClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("grok API key is required")
	}

	return &GrokClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Image generation can take time
		},
		circuitBreaker: newCircuitBreaker("GrokAPI"),
	}, nil
}

// isValidGrokAspectRatio reports whether ratio is in the official xAI set.
func isValidGrokAspectRatio(ratio string) bool {
	for _, r := range GrokAspectRatios {
		if r == ratio {
			return true
		}
	}
	return false
}

// buildGrokImageRequest assembles the xAI request body from generation options.
// Resolves user-typed aliases (e.g. "grok-imagine-pro") via the provider registry
// since the Lambda handler doesn't pre-resolve. Maps --image-size 1K/2K to xAI's
// lowercase resolution param; emits a stderr warning for unsupported values
// (4K, etc.) so users learn the request was downgraded.
func buildGrokImageRequest(enhancedPrompt string, options models.GenerateOptions) GrokImageRequest {
	numImages := 1
	if options.NumberOfImages > 0 && options.NumberOfImages <= 10 {
		numImages = options.NumberOfImages
	}

	modelID := grokDefaultModel
	if options.Model != "" {
		modelID = ResolveModelName(options.Model)
	}
	request := GrokImageRequest{
		Model:          modelID,
		Prompt:         enhancedPrompt,
		N:              numImages,
		ResponseFormat: "b64_json",
		User:           options.User,
	}

	isImagine := isGrokImagineModel(request.Model)
	if options.AspectRatio != "" && isImagine {
		if isValidGrokAspectRatio(options.AspectRatio) {
			request.AspectRatio = options.AspectRatio
		} else {
			fmt.Fprintf(os.Stderr, "warning: --aspect-ratio %q not supported on Grok; valid: %s; using xAI default\n",
				options.AspectRatio, strings.Join(GrokAspectRatios, ", "))
		}
	}
	if isImagine && options.ImageSize != "" {
		switch options.ImageSize {
		case "1K", "1k":
			request.Resolution = "1k"
		case "2K", "2k":
			request.Resolution = "2k"
		default:
			fmt.Fprintf(os.Stderr, "warning: --image-size %q not supported on Grok (only 1K/2K); using xAI default\n", options.ImageSize)
		}
	}

	return request
}

// isHTTPSImageURL reports whether s is an absolute https URL xAI can fetch.
func isHTTPSImageURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return u.Scheme == "https" && u.Host != ""
}

// attachGrokInputImages loads reference images into the request for /images/edits.
// Each entry may be a local file path (encoded as a data URI) or a public https:// URL
// (passed through to xAI without downloading). One entry uses the singular `image`
// field; two or more use `images` (max 3).
func attachGrokInputImages(request *GrokImageRequest, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	if len(paths) > grokMaxInputImages {
		return fmt.Errorf("Grok Imagine supports at most %d input images, got %d", grokMaxInputImages, len(paths))
	}

	sources := make([]GrokImageSource, 0, len(paths))
	for _, path := range paths {
		if path == "" {
			return fmt.Errorf("input image path is empty")
		}
		// Reject bare http://; xAI accepts public URLs and we require TLS.
		if strings.HasPrefix(strings.ToLower(path), "http://") {
			return fmt.Errorf("input image URL must use https:// (got %q)", path)
		}
		if isHTTPSImageURL(path) {
			sources = append(sources, GrokImageSource{
				URL:  path,
				Type: "image_url",
			})
			continue
		}
		data, mimeType, err := readInputImageData(path)
		if err != nil {
			return fmt.Errorf("read input image %s: %w", path, err)
		}
		dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(data))
		sources = append(sources, GrokImageSource{
			URL:  dataURL,
			Type: "image_url",
		})
	}

	if len(sources) == 1 {
		request.Image = &sources[0]
	} else {
		request.Images = sources
	}
	return nil
}

// GenerateImage generates or edits an image using the Grok API.
// With --input-image / InputImages set, requests go to POST /v1/images/edits;
// otherwise POST /v1/images/generations.
func (c *GrokClient) GenerateImage(ctx context.Context, prompt string, options models.GenerateOptions) ([]*models.GeneratedImage, error) {
	if prompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	request := buildGrokImageRequest(EnhancePrompt(prompt), options)
	endpoint := grokGenerationsPath
	if len(options.InputImages) > 0 {
		if err := attachGrokInputImages(&request, options.InputImages); err != nil {
			return nil, err
		}
		endpoint = grokEditsPath
	}

	// Execute through circuit breaker
	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.doRequest(ctx, endpoint, request)
	})

	if err != nil {
		if isCircuitBreakerError(err) {
			return nil, fmt.Errorf("API circuit breaker is open (too many failures): %w", err)
		}
		return nil, err
	}

	return result.([]*models.GeneratedImage), nil
}

// doRequest performs the actual HTTP request to the Grok API
func (c *GrokClient) doRequest(ctx context.Context, endpoint string, request GrokImageRequest) ([]*models.GeneratedImage, error) {
	// Marshal request body
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s%s", grokBaseURL, endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleHTTPError(resp.StatusCode, body)
	}

	// Parse response
	var grokResp GrokImageResponse
	if err := json.Unmarshal(body, &grokResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API error in response
	if grokResp.Error != nil {
		return nil, fmt.Errorf("API error: %s (type: %s)", grokResp.Error.Message, grokResp.Error.Type)
	}

	// Check we got image data
	if len(grokResp.Data) == 0 {
		return nil, fmt.Errorf("no images returned from API")
	}

	var generatedImages []*models.GeneratedImage

	for i, imageData := range grokResp.Data {
		var rawImageData []byte

		// Handle base64 response
		if imageData.B64JSON != "" {
			rawImageData, err = base64.StdEncoding.DecodeString(imageData.B64JSON)
			if err != nil {
				continue // Skip if decode fails
			}
		} else if imageData.URL != "" {
			// Download from URL if base64 not available
			rawImageData, err = downloadImage(ctx, imageData.URL)
			if err != nil {
				continue // Skip if download fails
			}
		} else {
			continue // Skip if no data
		}

		// Detect image format from data
		format := detectImageFormat(rawImageData)

		// Get actual image dimensions from the data
		width, height, err := GetImageDimensionsFromBytes(rawImageData)
		if err != nil {
			// Fallback to 0,0 which will skip dimension enforcement
			width, height = 0, 0
		}

		// Build metadata
		metadata := map[string]string{
			"model":     request.Model,
			"prompt":    request.Prompt,
			"api":       "grok",
			"generated": time.Now().UTC().Format(time.RFC3339),
			"candidate": fmt.Sprintf("%d", i),
			"endpoint":  endpoint,
		}
		if imageData.RevisedPrompt != "" {
			metadata["revised_prompt"] = imageData.RevisedPrompt
		}
		if imageData.MimeType != "" {
			metadata["mime_type"] = imageData.MimeType
		}
		if grokResp.Usage != nil && grokResp.Usage.CostInUSDTicks > 0 {
			metadata["cost_in_usd_ticks"] = fmt.Sprintf("%d", grokResp.Usage.CostInUSDTicks)
		}

		generatedImages = append(generatedImages, &models.GeneratedImage{
			Data:     rawImageData,
			Format:   format,
			Width:    width,
			Height:   height,
			Metadata: metadata,
		})
	}

	if len(generatedImages) == 0 {
		return nil, fmt.Errorf("no images returned from API")
	}

	return generatedImages, nil
}

// handleHTTPError converts HTTP errors to user-friendly messages
func (c *GrokClient) handleHTTPError(statusCode int, body []byte) error {
	// Try to parse error response
	var errResp struct {
		Error *GrokErrorDetail `json:"error"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != nil {
		switch statusCode {
		case http.StatusUnauthorized:
			return fmt.Errorf("authentication failed: %s\nHint: Check your GROK_API_KEY is valid", errResp.Error.Message)
		case http.StatusTooManyRequests:
			return fmt.Errorf("rate limit exceeded: %s\nHint: Wait a moment and try again", errResp.Error.Message)
		case http.StatusBadRequest:
			return fmt.Errorf("invalid request: %s", errResp.Error.Message)
		default:
			return fmt.Errorf("API error (status %d): %s", statusCode, errResp.Error.Message)
		}
	}

	// Generic error if we couldn't parse the response
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("authentication failed (status 401)\nHint: Check your GROK_API_KEY is valid")
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limit exceeded (status 429)\nHint: Wait a moment and try again")
	case http.StatusBadRequest:
		return fmt.Errorf("invalid request (status 400): %s", string(body))
	default:
		return fmt.Errorf("API error (status %d): %s", statusCode, string(body))
	}
}

// detectImageFormat detects the image format from the raw bytes
func detectImageFormat(data []byte) string {
	if len(data) < 8 {
		return "png" // Default
	}

	// Check magic bytes
	switch {
	case data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47:
		return "png"
	case data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return "jpg"
	case data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46:
		return "webp"
	case data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46:
		return "gif"
	default:
		return "png" // Default to PNG
	}
}

// isGrokImagineModel returns true if the model supports grok-imagine features (aspect_ratio)
func isGrokImagineModel(model string) bool {
	return strings.HasPrefix(model, "grok-imagine")
}

// Close cleans up resources (implements ImageGenerator interface)
func (c *GrokClient) Close() error {
	return nil
}
