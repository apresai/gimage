package progress

import (
	"context"
	"testing"
)

func TestNoOpReporter(t *testing.T) {
	reporter := NewNoOpReporter()
	ctx := context.Background()

	// Should not panic with any operation
	reporter.Start(ctx, "test operation")
	reporter.Update(50, 100, "halfway")
	reporter.Complete("done")
	reporter.Error(nil)
}

func TestFromContext_DefaultNoOp(t *testing.T) {
	ctx := context.Background()
	reporter := FromContext(ctx)

	if reporter == nil {
		t.Fatal("FromContext should return a NoOpReporter, not nil")
	}

	// Should be usable without panic
	reporter.Start(ctx, "op")
	reporter.Update(1, 1, "done")
	reporter.Complete(nil)
}

func TestFromContext_WithAttachedReporter(t *testing.T) {
	noop := NewNoOpReporter()
	ctx := context.WithValue(context.Background(), reporterKey, noop)

	reporter := FromContext(ctx)
	if reporter != noop {
		t.Error("FromContext should return the attached reporter")
	}
}
