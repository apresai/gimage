package lambdahandler

// GenerateRequest represents a request to generate an image from a text prompt
type GenerateRequest struct {
	Prompt         string `json:"prompt"`
	Model          string `json:"model,omitempty"`
	Size           string `json:"size,omitempty"`
	Style          string `json:"style,omitempty"`
	Seed           int64  `json:"seed,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"` // "base64" or "s3_url"
}

// ResizeRequest represents a request to resize an image
type ResizeRequest struct {
	Image          string `json:"image"` // base64 encoded image or S3 key
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	ResponseFormat string `json:"response_format,omitempty"`
}

// ScaleRequest represents a request to scale an image by a factor
type ScaleRequest struct {
	Image          string  `json:"image"` // base64 encoded image or S3 key
	Factor         float64 `json:"factor"`
	ResponseFormat string  `json:"response_format,omitempty"`
}

// CropRequest represents a request to crop an image
type CropRequest struct {
	Image          string `json:"image"` // base64 encoded image or S3 key
	X              int    `json:"x"`
	Y              int    `json:"y"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	ResponseFormat string `json:"response_format,omitempty"`
}

// CompressRequest represents a request to compress an image
type CompressRequest struct {
	Image          string `json:"image"` // base64 encoded image or S3 key
	Quality        int    `json:"quality,omitempty"`
	Format         string `json:"format,omitempty"` // jpg, png, webp
	ResponseFormat string `json:"response_format,omitempty"`
}

// ConvertRequest represents a request to convert an image format
type ConvertRequest struct {
	Image          string `json:"image"` // base64 encoded image or S3 key
	TargetFormat   string `json:"target_format"`
	ResponseFormat string `json:"response_format,omitempty"`
}

// BatchOperation represents a single operation in a batch request
type BatchOperation struct {
	Operation string                 `json:"operation"` // "resize", "scale", "crop", "compress", "convert"
	Image     string                 `json:"image"`     // base64 encoded image or S3 key
	Params    map[string]interface{} `json:"params"`
}

// BatchRequest represents a request to process multiple images
type BatchRequest struct {
	Operations  []BatchOperation `json:"operations"`
	CallbackURL string           `json:"callback_url,omitempty"`
}

// ImageResponse represents the response for image operations
type ImageResponse struct {
	Image     string            `json:"image,omitempty"`  // base64 encoded (for small images)
	S3URL     string            `json:"s3_url,omitempty"` // presigned URL (for large images)
	S3Key     string            `json:"s3_key,omitempty"` // S3 key for chaining operations
	Width     int               `json:"width"`
	Height    int               `json:"height"`
	Format    string            `json:"format"`
	SizeBytes int64             `json:"size_bytes"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// BatchResponse represents the response for batch operations
type BatchResponse struct {
	BatchID   string          `json:"batch_id"`
	Status    string          `json:"status"` // "processing", "completed", "failed"
	StatusURL string          `json:"status_url,omitempty"`
	Results   []ImageResponse `json:"results,omitempty"`
}

// HealthResponse represents the response for health check
type HealthResponse struct {
	Status  string            `json:"status"`
	Version string            `json:"version"`
	APIs    map[string]string `json:"apis"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}
