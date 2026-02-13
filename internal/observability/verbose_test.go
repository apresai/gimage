package observability

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVerboseLoggerWithOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := NewVerboseLoggerWithOutput(ComponentCLI, &buf)

	assert.True(t, logger.IsEnabled())
	assert.Equal(t, ComponentCLI, logger.component)

	logger.Info("hello %s", "world")
	output := buf.String()
	assert.Contains(t, output, "[CLI]")
	assert.Contains(t, output, "hello world")
}

func TestVerboseLoggerDisabled(t *testing.T) {
	ResetVerbose()
	defer ResetVerbose()

	logger := NewVerboseLogger(ComponentCLI)
	assert.False(t, logger.IsEnabled())

	// Disabled logger should not write anything
	var buf bytes.Buffer
	logger.output = &buf
	logger.Debug("should not appear")
	assert.Empty(t, buf.String())
}

func TestWithField(t *testing.T) {
	var buf bytes.Buffer
	logger := NewVerboseLoggerWithOutput(ComponentCLI, &buf)

	logger.WithField("key", "value").Debug("test message")

	output := buf.String()
	assert.Contains(t, output, "key=value")
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "[CLI]")
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewVerboseLoggerWithOutput(ComponentCLI, &buf)

	logger.WithFields(map[string]interface{}{
		"a": 1,
		"b": 2,
	}).Info("multi-field test")

	output := buf.String()
	assert.Contains(t, output, "a=1")
	assert.Contains(t, output, "b=2")
	assert.Contains(t, output, "multi-field test")
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name   string
		logFn  func(l *VerboseLogger, format string, args ...interface{})
		msg    string
	}{
		{"Debug", (*VerboseLogger).Debug, "debug msg"},
		{"Info", (*VerboseLogger).Info, "info msg"},
		{"Warn", (*VerboseLogger).Warn, "warn msg"},
		{"Error", (*VerboseLogger).Error, "error msg"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewVerboseLoggerWithOutput(ComponentCLI, &buf)

			tc.logFn(logger, tc.msg)

			output := buf.String()
			assert.Contains(t, output, "[CLI]")
			assert.Contains(t, output, tc.msg)
		})
	}
}

func TestLogRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := NewVerboseLoggerWithOutput(ComponentGemini, &buf)

	logger.LogRequest("GET", "https://example.com/api", 100)

	output := buf.String()
	assert.Contains(t, output, "GET")
	assert.Contains(t, output, "https://example.com/api")
	assert.Contains(t, output, "100")
}

func TestLogResponse(t *testing.T) {
	var buf bytes.Buffer
	logger := NewVerboseLoggerWithOutput(ComponentGemini, &buf)

	logger.LogResponse(200, "OK", 500)

	output := buf.String()
	assert.Contains(t, output, "200")
	assert.Contains(t, output, "OK")
	assert.Contains(t, output, "500")
}

func TestLogModelSelection(t *testing.T) {
	t.Run("same model", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewVerboseLoggerWithOutput(ComponentCLI, &buf)

		logger.LogModelSelection("flash", "flash")

		output := buf.String()
		assert.Contains(t, output, "Model: flash")
		assert.NotContains(t, output, "resolved")
	})

	t.Run("different model", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewVerboseLoggerWithOutput(ComponentCLI, &buf)

		logger.LogModelSelection("flash", "gemini-2.5-flash-image")

		output := buf.String()
		assert.Contains(t, output, "resolved")
		assert.Contains(t, output, "flash")
		assert.Contains(t, output, "gemini-2.5-flash-image")
	})
}

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", "***"},
		{"short key", "short", "***"},
		{"exactly 12 chars", "abcdefghijkl", "***"},
		{"16 char key", "abcdefghijklmnop", "abcdefgh...mnop"},
		{"long key", "AIzaSyD1234567890abcdefg", "AIzaSyD1...defg"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MaskAPIKey(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMaskURL(t *testing.T) {
	t.Run("with api key", func(t *testing.T) {
		url := "https://api.example.com?key=SECRET123"
		result := MaskURL(url, "SECRET123")
		assert.Contains(t, result, "***KEY***")
		assert.NotContains(t, result, "SECRET123")
	})

	t.Run("empty api key", func(t *testing.T) {
		url := "https://api.example.com"
		result := MaskURL(url, "")
		assert.Equal(t, url, result)
	})

	t.Run("key not in url", func(t *testing.T) {
		url := "https://api.example.com?foo=bar"
		result := MaskURL(url, "NOTHERE")
		assert.Equal(t, url, result)
	})
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"long string", "this is a very long string that needs truncation", 20, "this is a very lo..."},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := truncate(tc.input, tc.maxLen)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{LogLevel(99), "UNKNOWN"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.level.String())
		})
	}
}

func TestWithFieldDoesNotMutateOriginal(t *testing.T) {
	var buf bytes.Buffer
	original := NewVerboseLoggerWithOutput(ComponentCLI, &buf)
	original.WithField("extra", "data")

	// Original should still have no fields
	assert.Empty(t, original.fields)
}

func TestWithFieldsDoesNotMutateOriginal(t *testing.T) {
	var buf bytes.Buffer
	original := NewVerboseLoggerWithOutput(ComponentCLI, &buf)
	original.WithFields(map[string]interface{}{"a": 1, "b": 2})

	assert.Empty(t, original.fields)
}

func TestFromContext(t *testing.T) {
	t.Run("with logger in context", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewVerboseLoggerWithOutput(ComponentMCP, &buf)
		ctx := WithLogger(context.Background(), logger)
		require.NotNil(t, ctx)

		retrieved := FromContext(ctx, ComponentCLI)
		assert.Equal(t, ComponentMCP, retrieved.component)
	})

	t.Run("without logger in context", func(t *testing.T) {
		ResetVerbose()
		defer ResetVerbose()

		ctx := context.Background()
		retrieved := FromContext(ctx, ComponentCLI)
		assert.Equal(t, ComponentCLI, retrieved.component)
	})
}
