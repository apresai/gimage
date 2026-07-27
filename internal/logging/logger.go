// Package logging provides structured logging for gimage with DEBUG mode support.
package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel represents the logging level
type LogLevel string

const (
	DEBUG LogLevel = "DEBUG"
	INFO  LogLevel = "INFO"
	WARN  LogLevel = "WARN"
	ERROR LogLevel = "ERROR"
)

// Logger manages application logging
type Logger struct {
	enabled  bool
	logFile  *os.File
	logPath  string
	logLevel LogLevel
	mu       sync.Mutex
}

var (
	globalLogger *Logger
	once         sync.Once
)

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	once.Do(func() {
		globalLogger = NewLogger()
	})
	return globalLogger
}

// NewLogger creates a new logger instance
func NewLogger() *Logger {
	mode := os.Getenv("GIMAGE_MODE")
	enabled := mode == "DEBUG" || mode == "LOG"

	logger := &Logger{
		enabled:  enabled,
		logLevel: DEBUG,
	}

	if enabled {
		logger.initLogFile()
		logger.LogStartup()
	}

	return logger
}

// initLogFile initializes the log file
func (l *Logger) initLogFile() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get home directory: %v\n", err)
		return
	}

	gimageDir := filepath.Join(home, ".gimage")
	if err := os.MkdirAll(gimageDir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create .gimage directory: %v\n", err)
		return
	}

	logPath := filepath.Join(gimageDir, "gimage.log")
	l.logPath = logPath

	// Open log file in append mode, create if doesn't exist
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		return
	}

	l.logFile = f
}

// Log writes a log entry
func (l *Logger) Log(level LogLevel, message string, args ...interface{}) {
	if !l.enabled {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logFile == nil {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	formatted := fmt.Sprintf(message, args...)
	entry := fmt.Sprintf("[%s] %s: %s\n", timestamp, level, formatted)

	if _, err := l.logFile.WriteString(entry); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write to log: %v\n", err)
	}

	// Flush to ensure it's written
	l.logFile.Sync()
}

// LogDebug logs a debug message
func (l *Logger) LogDebug(message string, args ...interface{}) {
	l.Log(DEBUG, message, args...)
}

// LogInfo logs an info message
func (l *Logger) LogInfo(message string, args ...interface{}) {
	l.Log(INFO, message, args...)
}

// LogWarn logs a warning message
func (l *Logger) LogWarn(message string, args ...interface{}) {
	l.Log(WARN, message, args...)
}

// LogError logs an error message
func (l *Logger) LogError(message string, args ...interface{}) {
	l.Log(ERROR, message, args...)
}

// LogStartup logs application startup information
func (l *Logger) LogStartup() {
	if !l.enabled {
		return
	}

	l.Log(INFO, "==========================================")
	l.Log(INFO, "gimage started in DEBUG mode")
	l.Log(INFO, "Log file: %s", l.logPath)
	l.Log(INFO, "Time: %s", time.Now().Format("2006-01-02 15:04:05"))
	l.Log(INFO, "==========================================")
}

// LogGenerateStart logs the start of image generation
func (l *Logger) LogGenerateStart(prompt string, model string, apiName string, size string, style string, outputPath string) {
	if !l.enabled {
		return
	}

	l.Log(INFO, "GENERATE START")
	l.Log(INFO, "  Prompt: %q", prompt)
	l.Log(INFO, "  Model: %s", model)
	l.Log(INFO, "  API: %s", apiName)
	l.Log(INFO, "  Size: %s", size)
	l.Log(INFO, "  Style: %s", style)
	l.Log(INFO, "  Output: %s", outputPath)
}

// LogGenerateCommand logs the equivalent CLI command for reproducibility
func (l *Logger) LogGenerateCommand(command string) {
	if !l.enabled {
		return
	}

	l.Log(INFO, "EQUIVALENT CLI COMMAND:")
	l.Log(INFO, "  $ %s", command)
}

// LogGenerateComplete logs completion of image generation
func (l *Logger) LogGenerateComplete(success bool, outputPath string, fileSize int64, duration time.Duration, errMsg string) {
	if !l.enabled {
		return
	}

	status := "SUCCESS"
	if !success {
		status = "FAILED"
	}

	msg := fmt.Sprintf("GENERATE COMPLETE [%s]", status)
	if success && outputPath != "" {
		msg += fmt.Sprintf(" output=%q size=%d bytes", outputPath, fileSize)
	}
	if !success && errMsg != "" {
		msg += fmt.Sprintf(" error=%q", errMsg)
	}
	msg += fmt.Sprintf(" duration=%s", duration.String())

	l.Log(INFO, "%s", msg)
}

// LogAuthStatus logs authentication status for LLMs
func (l *Logger) LogAuthStatus(api string, hasAuth bool, details string) {
	if !l.enabled {
		return
	}

	status := "NOT CONFIGURED"
	if hasAuth {
		status = "CONFIGURED"
	}

	l.Log(INFO, "AUTH STATUS: %s = %s (%s)", api, status, details)
}

// LogError logs a detailed error with context
func (l *Logger) LogErrorContext(context string, err error, details map[string]string) {
	if !l.enabled {
		return
	}

	l.Log(ERROR, "ERROR CONTEXT: %s", context)
	if err != nil {
		l.Log(ERROR, "  Error Type: %T", err)
		l.Log(ERROR, "  Error Message: %v", err)
	}

	for key, value := range details {
		l.Log(ERROR, "  %s: %s", key, value)
	}
}

// Close closes the log file
func (l *Logger) Close() error {
	if l.logFile == nil {
		return nil
	}

	// Log shutdown messages BEFORE acquiring lock
	l.Log(INFO, "gimage shutdown")
	l.Log(INFO, "==========================================")

	l.mu.Lock()
	defer l.mu.Unlock()

	return l.logFile.Close()
}

// IsEnabled returns whether logging is enabled
func (l *Logger) IsEnabled() bool {
	return l.enabled
}
