package generate

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/sony/gobreaker/v2"
)

// Circuit breaker configuration constants
const (
	// maxConsecutiveFailures is the number of consecutive failures before opening the circuit
	maxConsecutiveFailures = 5

	// circuitBreakerInterval is the period of the cyclic state transition from StateOpen to StateHalfOpen
	circuitBreakerInterval = 60 * time.Second

	// circuitBreakerTimeout is the timeout for the StateHalfOpen state
	circuitBreakerTimeout = 30 * time.Second
)

// newCircuitBreaker creates a new generic circuit breaker with standard settings
func newCircuitBreaker[T any](name string) *gobreaker.CircuitBreaker[T] {
	settings := gobreaker.Settings{
		Name:        name,
		MaxRequests: 3, // Allow 3 requests in half-open state to test recovery
		Interval:    circuitBreakerInterval,
		Timeout:     circuitBreakerTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Open circuit after consecutive failures threshold
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.ConsecutiveFailures >= maxConsecutiveFailures ||
				(counts.Requests >= 10 && failureRatio > 0.6)
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			// Log state transitions for debugging
			fmt.Fprintf(os.Stderr, "[CIRCUIT-BREAKER] %s: state changed from %s to %s\n", name, from, to)
		},
	}

	return gobreaker.NewCircuitBreaker[T](settings)
}

// isCircuitBreakerError checks if an error is a circuit breaker error
func isCircuitBreakerError(err error) bool {
	return errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests)
}
