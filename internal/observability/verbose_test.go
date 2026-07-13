package observability

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewVerboseLogger(t *testing.T) {
	logger := NewVerboseLogger(ComponentCLI)
	assert.Equal(t, ComponentCLI, logger.component)
	assert.NotNil(t, logger.output)
}

func TestVerboseLoggerDisabled(t *testing.T) {
	// Without verbose env, logger should be disabled by default
	t.Setenv("GIMAGE_VERBOSE", "")
	t.Setenv("VERBOSE", "")

	// Force re-check by creating after clearing env; IsVerbose uses sync.Once
	// so we test the Debug no-op path by forcing enabled=false.
	logger := &VerboseLogger{
		component: ComponentCLI,
		enabled:   false,
		output:    &bytes.Buffer{},
	}

	logger.Debug("should not appear")
	assert.Empty(t, logger.output.(*bytes.Buffer).String())
}

func TestVerboseLoggerDebugInfoWarnError(t *testing.T) {
	var buf bytes.Buffer
	logger := &VerboseLogger{
		component: ComponentGemini,
		enabled:   true,
		output:    &buf,
	}

	logger.Debug("debug %s", "msg")
	logger.Info("info %s", "msg")
	logger.Warn("warn %s", "msg")
	logger.Error("error %s", "msg")

	output := buf.String()
	assert.Contains(t, output, "[GEMINI] debug msg")
	assert.Contains(t, output, "[GEMINI] info msg")
	assert.Contains(t, output, "[GEMINI] warn msg")
	assert.Contains(t, output, "[GEMINI] error msg")
}

func TestLogGenerationStart(t *testing.T) {
	var buf bytes.Buffer
	logger := &VerboseLogger{
		component: ComponentMCP,
		enabled:   true,
		output:    &buf,
	}

	logger.LogGenerationStart("a beautiful sunset over mountains", map[string]interface{}{
		"model": "gemini-3-pro-image",
		"size":  "1024x1024",
		"empty": "",
		"zero":  0,
	})

	output := buf.String()
	assert.Contains(t, output, "Generation starting")
	assert.Contains(t, output, "Prompt:")
	assert.Contains(t, output, "Option model: gemini-3-pro-image")
	assert.Contains(t, output, "Option size: 1024x1024")
	assert.NotContains(t, output, "Option empty")
	assert.NotContains(t, output, "Option zero")
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "short", truncate("short", 100))
	assert.Equal(t, "hello...", truncate("hello world this is long", 8))
}

func TestIsVerboseEnv(t *testing.T) {
	// IsVerbose uses sync.Once — only verify it returns a bool without panic.
	_ = os.Getenv("GIMAGE_VERBOSE")
	_ = IsVerbose()
}
