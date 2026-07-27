// Package observability provides unified logging for gimage.
// This file implements standardized verbose logging across CLI, MCP, and Lambda.
package observability

import (
	"fmt"
	"io"
	"os"
	"sync"

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
)

// VerboseLogger provides standardized verbose logging
type VerboseLogger struct {
	component Component
	enabled   bool
	output    io.Writer
	mu        sync.Mutex
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

// NewVerboseLogger creates a new verbose logger for a component
func NewVerboseLogger(component Component) *VerboseLogger {
	return &VerboseLogger{
		component: component,
		enabled:   IsVerbose(),
		output:    os.Stderr,
	}
}

// IsEnabled returns whether verbose logging is enabled
func (v *VerboseLogger) IsEnabled() bool {
	return v.enabled
}

// formatMessage formats a log message with component prefix
func (v *VerboseLogger) formatMessage(format string, args ...interface{}) string {
	msg := fmt.Sprintf(format, args...)
	return fmt.Sprintf("[%s] %s", v.component, msg)
}

// log writes a log message
func (v *VerboseLogger) log(format string, args ...interface{}) {
	if !v.enabled {
		return
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	msg := v.formatMessage(format, args...)
	fmt.Fprintln(v.output, msg)
}

// Debug logs a debug message
func (v *VerboseLogger) Debug(format string, args ...interface{}) {
	v.log(format, args...)
}

// Info logs an info message
func (v *VerboseLogger) Info(format string, args ...interface{}) {
	v.log(format, args...)
}

// Warn logs a warning message
func (v *VerboseLogger) Warn(format string, args ...interface{}) {
	v.log(format, args...)
}

// Error logs an error message
func (v *VerboseLogger) Error(format string, args ...interface{}) {
	v.log(format, args...)
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

// truncate truncates a string to maxLen with ellipsis
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
