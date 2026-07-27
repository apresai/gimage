// Package progress provides interfaces and implementations for tracking
// long-running operations with progress reporting capabilities.
package progress

import (
	"context"
)

// ProgressReporter is an interface for reporting progress of long-running operations.
// Implementations can provide different output mechanisms (TUI, CLI, logging, etc.).
type ProgressReporter interface {
	// Start initiates a progress tracking session for an operation.
	// The operation string describes what is being tracked.
	Start(ctx context.Context, operation string)

	// Update reports progress during an operation.
	// current and total represent the progress (e.g., bytes processed, items completed).
	// message provides additional context about the current state.
	Update(current, total int64, message string)

	// Complete marks the operation as successfully finished.
	// result can contain information about the final outcome.
	Complete(result interface{})

	// Error reports that the operation failed with an error.
	Error(err error)
}

// NoOpReporter is a silent reporter that does nothing.
// This is the default reporter for CLI operations that don't need progress output.
type NoOpReporter struct{}

// NewNoOpReporter creates a new no-op reporter.
func NewNoOpReporter() *NoOpReporter {
	return &NoOpReporter{}
}

func (r *NoOpReporter) Start(ctx context.Context, operation string) {}
func (r *NoOpReporter) Update(current, total int64, message string) {}
func (r *NoOpReporter) Complete(result interface{})                 {}
func (r *NoOpReporter) Error(err error)                             {}

// ContextKey type for context values
type contextKey string

const reporterKey contextKey = "progress_reporter"

// FromContext retrieves a reporter from the context.
// If no reporter is found, it returns a NoOpReporter.
func FromContext(ctx context.Context) ProgressReporter {
	if reporter, ok := ctx.Value(reporterKey).(ProgressReporter); ok {
		return reporter
	}
	return NewNoOpReporter()
}
