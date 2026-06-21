package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestLogger creates a logger with a temp file for testing.
func newTestLogger(t *testing.T) (*Logger, string) {
	t.Helper()
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	require.NoError(t, err)
	return &Logger{
		enabled:  true,
		logFile:  f,
		logPath:  logPath,
		logLevel: DEBUG,
	}, logPath
}

func readLog(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func TestNewLogger_Disabled(t *testing.T) {
	t.Setenv("GIMAGE_MODE", "")
	logger := NewLogger()
	defer logger.Close()
	assert.False(t, logger.IsEnabled())
}

func TestNewLogger_Enabled_DEBUG(t *testing.T) {
	t.Setenv("GIMAGE_MODE", "DEBUG")
	logger := NewLogger()
	defer logger.Close()
	assert.True(t, logger.IsEnabled())
}

func TestNewLogger_Enabled_LOG(t *testing.T) {
	t.Setenv("GIMAGE_MODE", "LOG")
	logger := NewLogger()
	defer logger.Close()
	assert.True(t, logger.IsEnabled())
}

func TestLogger_Log_Disabled(t *testing.T) {
	logger := &Logger{enabled: false}
	// Should not panic even with nil logFile
	logger.Log(INFO, "should be a no-op")
}

func TestLogger_Log_Enabled(t *testing.T) {
	logger, logPath := newTestLogger(t)
	defer logger.Close()

	logger.Log(INFO, "test message %s", "arg")

	content := readLog(t, logPath)
	assert.Contains(t, content, "INFO: test message arg")
	// Verify timestamp bracket format
	assert.Contains(t, content, "[")
}

func TestLogger_LogLevels(t *testing.T) {
	logger, logPath := newTestLogger(t)
	defer logger.Close()

	logger.LogDebug("debug msg")
	logger.LogInfo("info msg")
	logger.LogWarn("warn msg")
	logger.LogError("error msg")

	content := readLog(t, logPath)
	assert.Contains(t, content, "DEBUG: debug msg")
	assert.Contains(t, content, "INFO: info msg")
	assert.Contains(t, content, "WARN: warn msg")
	assert.Contains(t, content, "ERROR: error msg")
}

func TestLogger_LogCommandStart(t *testing.T) {
	logger, logPath := newTestLogger(t)
	defer logger.Close()

	logger.LogCommandStart("generate", []string{"--model", "flash"})

	content := readLog(t, logPath)
	assert.Contains(t, content, "COMMAND START: gimage generate")
	assert.Contains(t, content, "--model")
}

func TestLogger_LogCommandComplete_Success(t *testing.T) {
	logger, logPath := newTestLogger(t)
	defer logger.Close()

	logger.LogCommandComplete("generate", true, 5*time.Second, "")

	content := readLog(t, logPath)
	assert.Contains(t, content, "COMMAND COMPLETE")
	assert.Contains(t, content, "SUCCESS")
	assert.Contains(t, content, "5s")
}

func TestLogger_LogCommandComplete_Failure(t *testing.T) {
	logger, logPath := newTestLogger(t)
	defer logger.Close()

	logger.LogCommandComplete("generate", false, 1*time.Second, "api error")

	content := readLog(t, logPath)
	assert.Contains(t, content, "FAILED")
	assert.Contains(t, content, "api error")
}

func TestLogger_LogGenerateStart(t *testing.T) {
	logger, logPath := newTestLogger(t)
	defer logger.Close()

	logger.LogGenerateStart("sunset", "flash", "gemini", "1024x1024", "photo", "out.png")

	content := readLog(t, logPath)
	assert.Contains(t, content, "GENERATE START")
	assert.Contains(t, content, "sunset")
	assert.Contains(t, content, "flash")
	assert.Contains(t, content, "gemini")
}

func TestLogger_LogGenerateComplete_Success(t *testing.T) {
	logger, logPath := newTestLogger(t)
	defer logger.Close()

	logger.LogGenerateComplete(true, "out.png", 12345, 2*time.Second, "")

	content := readLog(t, logPath)
	assert.Contains(t, content, "GENERATE COMPLETE")
	assert.Contains(t, content, "SUCCESS")
	assert.Contains(t, content, "out.png")
	assert.Contains(t, content, "12345")
}

func TestLogger_LogGenerateComplete_Failure(t *testing.T) {
	logger, logPath := newTestLogger(t)
	defer logger.Close()

	logger.LogGenerateComplete(false, "", 0, 1*time.Second, "timeout")

	content := readLog(t, logPath)
	assert.Contains(t, content, "FAILED")
	assert.Contains(t, content, "timeout")
}

func TestLogger_LogAuthStatus(t *testing.T) {
	logger, logPath := newTestLogger(t)
	defer logger.Close()

	logger.LogAuthStatus("gemini", true, "env var")
	logger.LogAuthStatus("vertex", false, "not set")

	content := readLog(t, logPath)
	assert.Contains(t, content, "AUTH STATUS: gemini = CONFIGURED")
	assert.Contains(t, content, "AUTH STATUS: vertex = NOT CONFIGURED")
}

func TestLogger_LogAPICall(t *testing.T) {
	logger, logPath := newTestLogger(t)
	defer logger.Close()

	logger.LogAPICall("gemini", "flash", "https://api.example.com", 200, "")

	content := readLog(t, logPath)
	assert.Contains(t, content, "API CALL: gemini")
	assert.Contains(t, content, "status=200")
}

func TestLogger_LogAPICall_WithError(t *testing.T) {
	logger, logPath := newTestLogger(t)
	defer logger.Close()

	logger.LogAPICall("grok", "grok-imagine-image", "https://api.example.com", 500, "internal error")

	content := readLog(t, logPath)
	assert.Contains(t, content, "API CALL: grok")
	assert.Contains(t, content, "internal error")
}

func TestLogger_LogErrorContext(t *testing.T) {
	logger, logPath := newTestLogger(t)
	defer logger.Close()

	logger.LogErrorContext("generation failed", nil, map[string]string{
		"model": "flash",
		"api":   "gemini",
	})

	content := readLog(t, logPath)
	assert.Contains(t, content, "ERROR CONTEXT: generation failed")
	// Map iteration order is not deterministic, just check both keys exist
	assert.Contains(t, content, "model")
	assert.Contains(t, content, "flash")
}

func TestLogger_Close(t *testing.T) {
	logger, _ := newTestLogger(t)
	err := logger.Close()
	assert.NoError(t, err)
}

func TestLogger_Close_NilLogFile(t *testing.T) {
	logger := &Logger{enabled: false, logFile: nil}
	err := logger.Close()
	assert.NoError(t, err)
}

func TestLogger_GetLogPath(t *testing.T) {
	logger, logPath := newTestLogger(t)
	defer logger.Close()
	assert.Equal(t, logPath, logger.GetLogPath())
}

func TestLogger_IsEnabled(t *testing.T) {
	assert.True(t, (&Logger{enabled: true}).IsEnabled())
	assert.False(t, (&Logger{enabled: false}).IsEnabled())
}

func TestLogger_LogStartup(t *testing.T) {
	logger, logPath := newTestLogger(t)
	defer logger.Close()

	logger.LogStartup()

	content := readLog(t, logPath)
	assert.Contains(t, content, "gimage started in DEBUG mode")
	assert.True(t, strings.Count(content, "==========") >= 2)
}

func TestLogger_LogGenerateCommand(t *testing.T) {
	logger, logPath := newTestLogger(t)
	defer logger.Close()

	logger.LogGenerateCommand("gimage generate --model flash \"sunset\"")

	content := readLog(t, logPath)
	assert.Contains(t, content, "EQUIVALENT CLI COMMAND")
	assert.Contains(t, content, "gimage generate --model flash")
}

func TestLogger_Log_WithFormatArgs(t *testing.T) {
	logger, logPath := newTestLogger(t)
	defer logger.Close()

	logger.Log(INFO, "count=%d name=%s ok=%t", 42, "test", true)

	content := readLog(t, logPath)
	assert.Contains(t, content, "count=42 name=test ok=true")
}

func TestLogger_Disabled_Methods_NoOp(t *testing.T) {
	// All methods on a disabled logger should be no-ops (no panics)
	logger := &Logger{enabled: false}

	logger.LogDebug("msg")
	logger.LogInfo("msg")
	logger.LogWarn("msg")
	logger.LogError("msg")
	logger.LogStartup()
	logger.LogCommandStart("cmd", nil)
	logger.LogCommandComplete("cmd", true, time.Second, "")
	logger.LogGenerateStart("p", "m", "a", "s", "st", "o")
	logger.LogGenerateComplete(true, "o", 0, time.Second, "")
	logger.LogGenerateCommand("cmd")
	logger.LogAuthStatus("api", true, "d")
	logger.LogAPICall("api", "m", "e", 200, "")
	logger.LogErrorContext("ctx", nil, nil)
}
