package observability

import (
	"context"
	"sync"
	"time"
)

// ToolMetrics tracks invocation metrics for MCP tools
type ToolMetrics struct {
	mu               sync.RWMutex
	toolInvocations  map[string]*ToolStats
	totalInvocations int64
	totalSuccesses   int64
	totalFailures    int64
	totalLatencyMs   int64
}

// ToolStats holds statistics for a single tool
type ToolStats struct {
	Name         string
	Invocations  int64
	Successes    int64
	Failures     int64
	TotalLatency time.Duration
	AvgLatency   time.Duration
	MinLatency   time.Duration
	MaxLatency   time.Duration
	LastInvoked  time.Time
}

var (
	// globalMetrics is the singleton metrics instance
	globalMetrics *ToolMetrics
	metricsOnce   sync.Once
)

// GetMetrics returns the global metrics instance
func GetMetrics() *ToolMetrics {
	metricsOnce.Do(func() {
		globalMetrics = &ToolMetrics{
			toolInvocations: make(map[string]*ToolStats),
		}
	})
	return globalMetrics
}

// RecordToolInvocation records a tool invocation with timing and success/failure
func (m *ToolMetrics) RecordToolInvocation(ctx context.Context, toolName string, duration time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get or create tool stats
	stats, exists := m.toolInvocations[toolName]
	if !exists {
		stats = &ToolStats{
			Name:       toolName,
			MinLatency: duration,
			MaxLatency: duration,
		}
		m.toolInvocations[toolName] = stats
	}

	// Update stats
	stats.Invocations++
	stats.TotalLatency += duration
	stats.AvgLatency = time.Duration(int64(stats.TotalLatency) / stats.Invocations)
	stats.LastInvoked = time.Now()

	if duration < stats.MinLatency {
		stats.MinLatency = duration
	}
	if duration > stats.MaxLatency {
		stats.MaxLatency = duration
	}

	if success {
		stats.Successes++
		m.totalSuccesses++
	} else {
		stats.Failures++
		m.totalFailures++
	}

	m.totalInvocations++
	m.totalLatencyMs += duration.Milliseconds()

	// Log metrics event
	logger := Logger(ctx).With().
		Str("component", "metrics").
		Str("tool", toolName).
		Int64("duration_ms", duration.Milliseconds()).
		Bool("success", success).
		Logger()

	if success {
		logger.Debug().Msg("Tool invocation completed")
	} else {
		logger.Warn().Msg("Tool invocation failed")
	}
}

// LogSummary writes aggregate and per-tool metrics to the log. The MCP server
// calls this on shutdown so a session's tool usage is diagnosable from logs
// alone, without reproducing the session.
//
// Logged at Info rather than Debug: metrics nobody sees by default are the same
// as no metrics. Volume is bounded by the number of distinct tools invoked.
func (m *ToolMetrics) LogSummary(ctx context.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.totalInvocations == 0 {
		return
	}

	logger := Logger(ctx).With().Str("component", "metrics").Logger()

	logger.Info().
		Int64("total_invocations", m.totalInvocations).
		Int64("total_successes", m.totalSuccesses).
		Int64("total_failures", m.totalFailures).
		Float64("success_rate_pct", float64(m.totalSuccesses)/float64(m.totalInvocations)*100).
		Int64("avg_latency_ms", m.totalLatencyMs/m.totalInvocations).
		Int("tools_count", len(m.toolInvocations)).
		Msg("Metrics summary")

	for _, stats := range m.toolInvocations {
		logger.Info().
			Str("tool", stats.Name).
			Int64("invocations", stats.Invocations).
			Int64("successes", stats.Successes).
			Int64("failures", stats.Failures).
			Int64("total_latency_ms", stats.TotalLatency.Milliseconds()).
			Int64("avg_latency_ms", stats.AvgLatency.Milliseconds()).
			Int64("min_latency_ms", stats.MinLatency.Milliseconds()).
			Int64("max_latency_ms", stats.MaxLatency.Milliseconds()).
			Time("last_invoked", stats.LastInvoked).
			Msg("Tool metrics")
	}
}
