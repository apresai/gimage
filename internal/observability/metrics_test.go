package observability

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureLogs redirects the global zerolog logger into a buffer for the
// duration of the test, since Logger(ctx) derives from it.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	original := log.Logger
	log.Logger = zerolog.New(&buf)
	t.Cleanup(func() { log.Logger = original })
	return &buf
}

func newTestMetrics() *ToolMetrics {
	return &ToolMetrics{
		toolInvocations: make(map[string]*ToolStats),
	}
}

func TestGetMetrics(t *testing.T) {
	m := GetMetrics()
	require.NotNil(t, m)
	assert.NotNil(t, m.toolInvocations)

	// The MCP handler records against GetMetrics() and serve.go reports from
	// GetMetrics(). If this stopped returning one shared instance, the shutdown
	// summary would silently report an empty second instance.
	assert.Same(t, m, GetMetrics(), "GetMetrics must return the same instance")
}

func TestRecordToolInvocation(t *testing.T) {
	m := newTestMetrics()
	ctx := context.Background()

	m.RecordToolInvocation(ctx, "test_tool", 100*time.Millisecond, true)

	m.mu.RLock()
	stats := m.toolInvocations["test_tool"]
	m.mu.RUnlock()

	require.NotNil(t, stats)
	assert.Equal(t, int64(1), stats.Invocations)
	assert.Equal(t, int64(1), stats.Successes)
	assert.Equal(t, int64(0), stats.Failures)
	assert.Equal(t, "test_tool", stats.Name)
	assert.Equal(t, 100*time.Millisecond, stats.MinLatency)
	assert.Equal(t, 100*time.Millisecond, stats.MaxLatency)
	assert.Equal(t, int64(1), m.totalInvocations)
	assert.Equal(t, int64(1), m.totalSuccesses)

	m.RecordToolInvocation(ctx, "test_tool", 200*time.Millisecond, false)

	m.mu.RLock()
	stats = m.toolInvocations["test_tool"]
	m.mu.RUnlock()

	require.NotNil(t, stats)
	assert.Equal(t, int64(2), stats.Invocations)
	assert.Equal(t, int64(1), stats.Successes)
	assert.Equal(t, int64(1), stats.Failures)
	assert.Equal(t, 100*time.Millisecond, stats.MinLatency)
	assert.Equal(t, 200*time.Millisecond, stats.MaxLatency)
	assert.Equal(t, 150*time.Millisecond, stats.AvgLatency)
	assert.Equal(t, int64(2), m.totalInvocations)
	assert.Equal(t, int64(1), m.totalFailures)
}

func TestRecordToolInvocationConcurrent(t *testing.T) {
	m := newTestMetrics()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.RecordToolInvocation(ctx, "concurrent_tool", 10*time.Millisecond, true)
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(50), m.totalInvocations)
	assert.Equal(t, int64(50), m.totalSuccesses)
}

func TestLogSummary(t *testing.T) {
	m := newTestMetrics()
	ctx := context.Background()

	m.RecordToolInvocation(ctx, "resize_image", 100*time.Millisecond, true)
	m.RecordToolInvocation(ctx, "resize_image", 300*time.Millisecond, false)
	m.RecordToolInvocation(ctx, "generate_image", 50*time.Millisecond, true)

	buf := captureLogs(t)
	m.LogSummary(ctx)
	out := buf.String()

	// Aggregate tier: every counter RecordToolInvocation writes must be read.
	assert.Contains(t, out, "Metrics summary")
	assert.Contains(t, out, `"total_invocations":3`)
	assert.Contains(t, out, `"total_successes":2`)
	assert.Contains(t, out, `"total_failures":1`)
	assert.Contains(t, out, `"avg_latency_ms":150`)
	assert.Contains(t, out, `"tools_count":2`)

	// Per-tool tier: the toolInvocations map and its ToolStats fields.
	assert.Contains(t, out, "Tool metrics")
	assert.Contains(t, out, `"tool":"resize_image"`)
	assert.Contains(t, out, `"tool":"generate_image"`)
	assert.Contains(t, out, `"min_latency_ms":100`)
	assert.Contains(t, out, `"max_latency_ms":300`)
	assert.Contains(t, out, `"total_latency_ms":400`)
	assert.Contains(t, out, "last_invoked")
}

func TestLogSummaryNoInvocations(t *testing.T) {
	m := newTestMetrics()

	buf := captureLogs(t)
	m.LogSummary(context.Background())

	assert.Empty(t, buf.String(), "no invocations should log nothing, not a zero-divide")
}
