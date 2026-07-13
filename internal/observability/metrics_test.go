package observability

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMetrics() *ToolMetrics {
	return &ToolMetrics{
		toolInvocations: make(map[string]*ToolStats),
	}
}

func TestGetMetrics(t *testing.T) {
	m := GetMetrics()
	require.NotNil(t, m)
	assert.NotNil(t, m.toolInvocations)
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
