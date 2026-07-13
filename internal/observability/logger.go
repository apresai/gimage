package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// contextKey is a private type for context keys to avoid collisions
type contextKey string

const (
	// requestIDKey is the context key for request IDs
	requestIDKey contextKey = "request_id"
)

// Initialize sets up structured logging for the application
// Call this once at application startup
func Initialize(verbose bool) {
	// Configure zerolog output to stderr (MCP requirement)
	log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

	// Set log level based on verbose flag
	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Use human-friendly console output if stderr is a terminal
	if isTerminal(os.Stderr) {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
		})
	}
}

// isTerminal checks if the file descriptor is a terminal
func isTerminal(f *os.File) bool {
	fileInfo, err := f.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// GenerateRequestID creates a new unique request ID
func GenerateRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if random fails
		return time.Now().Format("20060102150405")
	}
	return hex.EncodeToString(b)
}

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID retrieves the request ID from the context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// Logger returns a logger with request ID from context
func Logger(ctx context.Context) zerolog.Logger {
	logger := log.Logger
	if requestID := GetRequestID(ctx); requestID != "" {
		logger = logger.With().Str("request_id", requestID).Logger()
	}
	return logger
}

// LoggerWithComponent returns a logger with component name and request ID
func LoggerWithComponent(ctx context.Context, component string) zerolog.Logger {
	logger := Logger(ctx)
	return logger.With().Str("component", component).Logger()
}
