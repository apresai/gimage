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

func TestNewToolMetrics(t *testing.T) {
	m := newTestMetrics()

	assert.NotNil(t, m.toolInvocations)
	assert.Empty(t, m.toolInvocations)
	assert.Equal(t, int64(0), m.totalInvocations)
	assert.Equal(t, int64(0), m.totalSuccesses)
	assert.Equal(t, int64(0), m.totalFailures)
	assert.Equal(t, int64(0), m.totalLatencyMs)
}

func TestRecordToolInvocation(t *testing.T) {
	m := newTestMetrics()
	ctx := context.Background()

	// Record a success with 100ms duration
	m.RecordToolInvocation(ctx, "test_tool", 100*time.Millisecond, true)

	stats := m.GetToolStats("test_tool")
	require.NotNil(t, stats)
	assert.Equal(t, int64(1), stats.Invocations)
	assert.Equal(t, int64(1), stats.Successes)
	assert.Equal(t, int64(0), stats.Failures)
	assert.Equal(t, "test_tool", stats.Name)
	assert.Equal(t, 100*time.Millisecond, stats.MinLatency)
	assert.Equal(t, 100*time.Millisecond, stats.MaxLatency)

	// Record a failure with 200ms
	m.RecordToolInvocation(ctx, "test_tool", 200*time.Millisecond, false)

	stats = m.GetToolStats("test_tool")
	require.NotNil(t, stats)
	assert.Equal(t, int64(2), stats.Invocations)
	assert.Equal(t, int64(1), stats.Successes)
	assert.Equal(t, int64(1), stats.Failures)
	assert.Equal(t, 100*time.Millisecond, stats.MinLatency)
	assert.Equal(t, 200*time.Millisecond, stats.MaxLatency)
	assert.Equal(t, 150*time.Millisecond, stats.AvgLatency)
}

func TestGetToolStats(t *testing.T) {
	m := newTestMetrics()
	ctx := context.Background()

	m.RecordToolInvocation(ctx, "tool_a", 50*time.Millisecond, true)
	m.RecordToolInvocation(ctx, "tool_a", 150*time.Millisecond, true)
	m.RecordToolInvocation(ctx, "tool_b", 300*time.Millisecond, false)

	// Verify tool_a stats
	statsA := m.GetToolStats("tool_a")
	require.NotNil(t, statsA)
	assert.Equal(t, int64(2), statsA.Invocations)
	assert.Equal(t, int64(2), statsA.Successes)
	assert.Equal(t, int64(0), statsA.Failures)

	// Nonexistent tool returns nil
	statsNone := m.GetToolStats("nonexistent")
	assert.Nil(t, statsNone)

	// Verify returned stats is a copy -- modifying it should not affect original
	statsA.Invocations = 999
	originalA := m.GetToolStats("tool_a")
	require.NotNil(t, originalA)
	assert.Equal(t, int64(2), originalA.Invocations)
}

func TestGetAllStats(t *testing.T) {
	m := newTestMetrics()
	ctx := context.Background()

	m.RecordToolInvocation(ctx, "tool_x", 10*time.Millisecond, true)
	m.RecordToolInvocation(ctx, "tool_y", 20*time.Millisecond, true)
	m.RecordToolInvocation(ctx, "tool_z", 30*time.Millisecond, false)

	all := m.GetAllStats()
	assert.Len(t, all, 3)
	assert.Contains(t, all, "tool_x")
	assert.Contains(t, all, "tool_y")
	assert.Contains(t, all, "tool_z")

	// Verify copies -- mutating returned map should not affect original
	all["tool_x"].Invocations = 999
	fresh := m.GetToolStats("tool_x")
	assert.Equal(t, int64(1), fresh.Invocations)
}

func TestGetSummary(t *testing.T) {
	t.Run("empty metrics", func(t *testing.T) {
		m := newTestMetrics()
		summary := m.GetSummary()

		assert.Equal(t, int64(0), summary["total_invocations"])
		assert.Equal(t, float64(0), summary["success_rate_pct"])
		assert.Equal(t, int64(0), summary["avg_latency_ms"])
		assert.Equal(t, 0, summary["tools_count"])
	})

	t.Run("with invocations", func(t *testing.T) {
		m := newTestMetrics()
		ctx := context.Background()

		// 3 successes, 1 failure = 75% success rate
		m.RecordToolInvocation(ctx, "tool1", 100*time.Millisecond, true)
		m.RecordToolInvocation(ctx, "tool1", 200*time.Millisecond, true)
		m.RecordToolInvocation(ctx, "tool1", 300*time.Millisecond, true)
		m.RecordToolInvocation(ctx, "tool1", 400*time.Millisecond, false)

		summary := m.GetSummary()
		assert.Equal(t, int64(4), summary["total_invocations"])
		assert.Equal(t, int64(3), summary["total_successes"])
		assert.Equal(t, int64(1), summary["total_failures"])
		assert.Equal(t, float64(75), summary["success_rate_pct"])

		// avg_latency_ms = (100+200+300+400)/4 = 250
		assert.Equal(t, int64(250), summary["avg_latency_ms"])
		assert.Equal(t, 1, summary["tools_count"])
	})
}

func TestReset(t *testing.T) {
	m := newTestMetrics()
	ctx := context.Background()

	m.RecordToolInvocation(ctx, "tool1", 100*time.Millisecond, true)
	m.RecordToolInvocation(ctx, "tool2", 200*time.Millisecond, false)

	// Verify data exists before reset
	assert.Equal(t, int64(2), m.totalInvocations)

	m.Reset()

	assert.Equal(t, int64(0), m.totalInvocations)
	assert.Equal(t, int64(0), m.totalSuccesses)
	assert.Equal(t, int64(0), m.totalFailures)
	assert.Equal(t, int64(0), m.totalLatencyMs)
	assert.Empty(t, m.toolInvocations)

	all := m.GetAllStats()
	assert.Empty(t, all)

	summary := m.GetSummary()
	assert.Equal(t, int64(0), summary["total_invocations"])
}

func TestConcurrentAccess(t *testing.T) {
	m := newTestMetrics()
	ctx := context.Background()

	const goroutines = 10
	const invocationsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			toolName := "concurrent_tool"
			for j := 0; j < invocationsPerGoroutine; j++ {
				success := j%2 == 0
				m.RecordToolInvocation(ctx, toolName, time.Duration(j)*time.Millisecond, success)
			}
		}(i)
	}

	wg.Wait()

	expectedTotal := int64(goroutines * invocationsPerGoroutine)
	assert.Equal(t, expectedTotal, m.totalInvocations)

	stats := m.GetToolStats("concurrent_tool")
	require.NotNil(t, stats)
	assert.Equal(t, expectedTotal, stats.Invocations)

	// Each goroutine: 50 successes, 50 failures (even/odd)
	expectedSuccesses := int64(goroutines * invocationsPerGoroutine / 2)
	assert.Equal(t, expectedSuccesses, m.totalSuccesses)
	assert.Equal(t, expectedSuccesses, m.totalFailures)
}
