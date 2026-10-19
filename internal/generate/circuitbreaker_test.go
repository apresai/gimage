package generate

import (
	"errors"
	"testing"
	"time"

	"github.com/sony/gobreaker/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCircuitBreaker(t *testing.T) {
	cb := newCircuitBreaker[any]("TestAPI")

	require.NotNil(t, cb, "Circuit breaker should not be nil")
	assert.Equal(t, "TestAPI", cb.Name(), "Circuit breaker should have correct name")
	assert.Equal(t, gobreaker.StateClosed, cb.State(), "Circuit breaker should start in closed state")
}

func TestCircuitBreakerOpensAfterConsecutiveFailures(t *testing.T) {
	cb := newCircuitBreaker[any]("TestAPI")

	// Simulate consecutive failures
	failureCount := 0
	for i := 0; i < maxConsecutiveFailures+1; i++ {
		_, err := cb.Execute(func() (any, error) {
			return nil, errors.New("simulated failure")
		})
		if err != nil {
			failureCount++
		}
	}

	assert.Equal(t, maxConsecutiveFailures+1, failureCount, "Should have failed exactly maxConsecutiveFailures+1 times")
	assert.Equal(t, gobreaker.StateOpen, cb.State(), "Circuit breaker should be in open state after consecutive failures")
}

func TestCircuitBreakerFailsFastWhenOpen(t *testing.T) {
	cb := newCircuitBreaker[any]("TestAPI")

	// Force circuit breaker open by causing consecutive failures
	for i := 0; i < maxConsecutiveFailures+1; i++ {
		_, _ = cb.Execute(func() (any, error) {
			return nil, errors.New("simulated failure")
		})
	}

	// Circuit should be open now
	require.Equal(t, gobreaker.StateOpen, cb.State())

	// Try to execute - should fail fast without calling the function
	called := false
	_, err := cb.Execute(func() (any, error) {
		called = true
		return "success", nil
	})

	assert.False(t, called, "Function should not be called when circuit is open")
	assert.Error(t, err, "Should return error when circuit is open")
	assert.ErrorIs(t, err, gobreaker.ErrOpenState, "Should return ErrOpenState")
}

func TestCircuitBreakerRecoversAfterTimeout(t *testing.T) {
	// Create circuit breaker with short timeout for testing
	settings := gobreaker.Settings{
		Name:        "TestRecovery",
		MaxRequests: 1,
		Interval:    1 * time.Second,
		Timeout:     100 * time.Millisecond, // Short timeout for testing
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	}
	cb := gobreaker.NewCircuitBreaker[string](settings)

	// Force circuit breaker open
	for i := 0; i < 3; i++ {
		_, _ = cb.Execute(func() (string, error) {
			return "", errors.New("simulated failure")
		})
	}

	require.Equal(t, gobreaker.StateOpen, cb.State(), "Circuit should be open")

	// Wait for timeout to move to half-open state
	time.Sleep(150 * time.Millisecond)

	// Try to execute - circuit should be in half-open state
	result, err := cb.Execute(func() (string, error) {
		return "recovered", nil
	})

	assert.NoError(t, err, "Should succeed in half-open state")
	assert.Equal(t, "recovered", result, "Should return correct result")
	assert.Equal(t, gobreaker.StateClosed, cb.State(), "Circuit should be closed after successful request in half-open")
}

func TestCircuitBreakerAllowsSuccessfulRequests(t *testing.T) {
	cb := newCircuitBreaker[int]("TestAPI")

	// Execute successful requests
	for i := 0; i < 10; i++ {
		result, err := cb.Execute(func() (int, error) {
			return i, nil
		})

		assert.NoError(t, err, "Successful request should not error")
		assert.Equal(t, i, result, "Should return correct result")
	}

	assert.Equal(t, gobreaker.StateClosed, cb.State(), "Circuit should remain closed for successful requests")
}

func TestCircuitBreakerFailureRatio(t *testing.T) {
	cb := newCircuitBreaker[string]("TestAPI")

	// Execute mix of successful and failed requests
	// Need at least 10 requests for failure ratio to trigger (from ReadyToTrip logic)
	successCount := 0
	failureCount := 0

	for i := 0; i < 15; i++ {
		_, err := cb.Execute(func() (string, error) {
			// 80% failure rate
			if i%5 != 0 {
				return "", errors.New("simulated failure")
			}
			return "success", nil
		})

		if err != nil {
			failureCount++
		} else {
			successCount++
		}
	}

	// With 80% failure rate and 15 requests, circuit should open
	assert.Equal(t, gobreaker.StateOpen, cb.State(), "Circuit should open due to high failure ratio")
	assert.Greater(t, failureCount, successCount, "Should have more failures than successes")
}

func TestIsCircuitBreakerError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ErrOpenState",
			err:      gobreaker.ErrOpenState,
			expected: true,
		},
		{
			name:     "ErrTooManyRequests",
			err:      gobreaker.ErrTooManyRequests,
			expected: true,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCircuitBreakerError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
