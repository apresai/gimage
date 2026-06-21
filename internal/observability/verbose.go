// Package observability provides unified logging for gimage.
// This file implements standardized verbose logging across CLI, MCP, and Lambda.
package observability

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// Component represents a logging component/module
type Component string

const (
	ComponentGemini  Component = "GEMINI"
	ComponentVertex  Component = "VERTEX"
	ComponentMCP     Component = "MCP"
	ComponentLambda  Component = "LAMBDA"
	ComponentCLI     Component = "CLI"
	ComponentImaging Component = "IMAGING"
)

// LogLevel for verbose logging
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// VerboseLogger provides standardized verbose logging
type VerboseLogger struct {
	component Component
	enabled   bool
	output    io.Writer
	mu        sync.Mutex
	fields    map[string]interface{}
}

// Global verbose mode check
var (
	globalVerbose     bool
	globalVerboseOnce sync.Once
)

// IsVerbose returns true if verbose mode is enabled globally
func IsVerbose() bool {
	globalVerboseOnce.Do(func() {
		globalVerbose = viper.GetBool("verbose") ||
			os.Getenv("GIMAGE_VERBOSE") == "true" ||
			os.Getenv("VERBOSE") == "true"
	})
	return globalVerbose
}

// ResetVerbose resets the verbose check (useful for testing)
func ResetVerbose() {
	globalVerboseOnce = sync.Once{}
	globalVerbose = false
}

// NewVerboseLogger creates a new verbose logger for a component
func NewVerboseLogger(component Component) *VerboseLogger {
	return &VerboseLogger{
		component: component,
		enabled:   IsVerbose(),
		output:    os.Stderr,
		fields:    make(map[string]interface{}),
	}
}

// NewVerboseLoggerWithOutput creates a logger with custom output (for testing)
func NewVerboseLoggerWithOutput(component Component, output io.Writer) *VerboseLogger {
	return &VerboseLogger{
		component: component,
		enabled:   true,
		output:    output,
		fields:    make(map[string]interface{}),
	}
}

// WithField returns a new logger with an additional field
func (v *VerboseLogger) WithField(key string, value interface{}) *VerboseLogger {
	newLogger := &VerboseLogger{
		component: v.component,
		enabled:   v.enabled,
		output:    v.output,
		fields:    make(map[string]interface{}),
	}
	// Copy existing fields
	for k, val := range v.fields {
		newLogger.fields[k] = val
	}
	newLogger.fields[key] = value
	return newLogger
}

// WithFields returns a new logger with additional fields
func (v *VerboseLogger) WithFields(fields map[string]interface{}) *VerboseLogger {
	newLogger := &VerboseLogger{
		component: v.component,
		enabled:   v.enabled,
		output:    v.output,
		fields:    make(map[string]interface{}),
	}
	// Copy existing fields
	for k, val := range v.fields {
		newLogger.fields[k] = val
	}
	// Add new fields
	for k, val := range fields {
		newLogger.fields[k] = val
	}
	return newLogger
}

// IsEnabled returns whether verbose logging is enabled
func (v *VerboseLogger) IsEnabled() bool {
	return v.enabled
}

// formatMessage formats a log message with component prefix and fields
func (v *VerboseLogger) formatMessage(level LogLevel, format string, args ...interface{}) string {
	msg := fmt.Sprintf(format, args...)

	// Build field string if we have fields
	var fieldStr string
	if len(v.fields) > 0 {
		parts := make([]string, 0, len(v.fields))
		for k, val := range v.fields {
			parts = append(parts, fmt.Sprintf("%s=%v", k, val))
		}
		fieldStr = " " + strings.Join(parts, " ")
	}

	// Format: [COMPONENT] message fields
	return fmt.Sprintf("[%s] %s%s", v.component, msg, fieldStr)
}

// log writes a log message
func (v *VerboseLogger) log(level LogLevel, format string, args ...interface{}) {
	if !v.enabled {
		return
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	msg := v.formatMessage(level, format, args...)
	fmt.Fprintln(v.output, msg)
}

// Debug logs a debug message
func (v *VerboseLogger) Debug(format string, args ...interface{}) {
	v.log(LevelDebug, format, args...)
}

// Info logs an info message
func (v *VerboseLogger) Info(format string, args ...interface{}) {
	v.log(LevelInfo, format, args...)
}

// Warn logs a warning message
func (v *VerboseLogger) Warn(format string, args ...interface{}) {
	v.log(LevelWarn, format, args...)
}

// Error logs an error message
func (v *VerboseLogger) Error(format string, args ...interface{}) {
	v.log(LevelError, format, args...)
}

// --- Structured logging methods for common operations ---

// LogRequest logs an API request
func (v *VerboseLogger) LogRequest(method, url string, bodySize int) {
	v.Debug("Request: %s %s (body: %d bytes)", method, url, bodySize)
}

// LogResponse logs an API response
func (v *VerboseLogger) LogResponse(statusCode int, status string, bodySize int) {
	v.Debug("Response: %d %s (body: %d bytes)", statusCode, status, bodySize)
}

// LogModelSelection logs model selection
func (v *VerboseLogger) LogModelSelection(model, resolvedModel string) {
	if model != resolvedModel {
		v.Debug("Model resolved: %s -> %s", model, resolvedModel)
	} else {
		v.Debug("Model: %s", model)
	}
}

// LogGenerationStart logs the start of image generation
func (v *VerboseLogger) LogGenerationStart(prompt string, options map[string]interface{}) {
	v.Debug("Generation starting...")
	v.Debug("Prompt: %s", truncate(prompt, 100))
	for k, val := range options {
		if val != nil && val != "" && val != 0 {
			v.Debug("Option %s: %v", k, val)
		}
	}
}

// LogGenerationComplete logs successful generation
func (v *VerboseLogger) LogGenerationComplete(size int, format string, duration time.Duration) {
	v.Debug("Generation complete: %d bytes, format=%s, duration=%s", size, format, duration)
}

// LogGenerationError logs generation failure
func (v *VerboseLogger) LogGenerationError(err error) {
	v.Error("Generation failed: %v", err)
}

// LogImageProcessing logs image processing operations
func (v *VerboseLogger) LogImageProcessing(operation string, input, output string, details map[string]interface{}) {
	v.Debug("Processing: %s", operation)
	v.Debug("Input: %s", input)
	v.Debug("Output: %s", output)
	for k, val := range details {
		v.Debug("  %s: %v", k, val)
	}
}

// LogCircuitBreaker logs circuit breaker state changes
func (v *VerboseLogger) LogCircuitBreaker(state string) {
	v.Warn("Circuit breaker: %s", state)
}

// LogRetry logs retry attempts
func (v *VerboseLogger) LogRetry(attempt, maxAttempts int, err error, backoff time.Duration) {
	v.Debug("Retry %d/%d after %s: %v", attempt, maxAttempts, backoff, err)
}

// --- Context-aware logging for Lambda/MCP ---

type loggerContextKey struct{}

// WithLogger adds a logger to context
func WithLogger(ctx context.Context, logger *VerboseLogger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// FromContext retrieves logger from context, or creates a default one
func FromContext(ctx context.Context, defaultComponent Component) *VerboseLogger {
	if logger, ok := ctx.Value(loggerContextKey{}).(*VerboseLogger); ok {
		return logger
	}
	return NewVerboseLogger(defaultComponent)
}

// --- Helper functions ---

// truncate truncates a string to maxLen with ellipsis
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// MaskAPIKey masks an API key for logging (shows first 8 and last 4 chars)
func MaskAPIKey(key string) string {
	if len(key) <= 12 {
		return "***"
	}
	return key[:8] + "..." + key[len(key)-4:]
}

// MaskURL masks API key in URL for logging
func MaskURL(url, apiKey string) string {
	if apiKey == "" {
		return url
	}
	return strings.Replace(url, apiKey, "***KEY***", -1)
}
