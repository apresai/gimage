package generate

import (
	"strings"
	"time"
)

const (
	maxRetries          = 3
	retryBackoffInitial = 1 * time.Second
	retryBackoffMax     = 10 * time.Second
)

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Retry on common transient errors
	retryablePatterns := []string{
		"rate limit",
		"quota exceeded",
		"timeout",
		"deadline exceeded",
		"connection",
		"unavailable",
		"503",
		"429",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// extractFormatFromMimeType extracts the image format from a MIME type
func extractFormatFromMimeType(mimeType string) string {
	switch mimeType {
	case "image/png":
		return "png"
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	default:
		return "png" // Default to PNG
	}
}
