package hcl

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCircuitBreaker(t *testing.T) {
	options := CircuitBreakerOption{
		MaxFailures:   3,
		HalfOpenLimit: 2,
		ResetTimeout:  10 * time.Second,
	}

	cb := NewCircuitBreaker(options)

	assert.Equal(t, 3, cb.maxFailures)
	assert.Equal(t, 2, cb.halfOpenLimit)
	assert.Equal(t, 10*time.Second, cb.resetTimeout)
	assert.Equal(t, "", cb.state) // Default state should be empty (which is treated as CLOSED)
	assert.Equal(t, 0, cb.failureCount)
	assert.Equal(t, 0, cb.successCount)
}

func TestCircuitBreakerAllow(t *testing.T) {
	t.Run("Closed state", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerOption{
			MaxFailures:   3,
			HalfOpenLimit: 2,
			ResetTimeout:  10 * time.Second,
		})
		// Default state is treated as CLOSED
		assert.True(t, cb.allow())
	})

	t.Run("Open state - timeout not reached", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerOption{
			MaxFailures:   3,
			HalfOpenLimit: 2,
			ResetTimeout:  10 * time.Second,
		})
		cb.state = OPEN
		cb.lastFailTime = time.Now()
		assert.False(t, cb.allow())
	})

	t.Run("Open state - timeout reached", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerOption{
			MaxFailures:   3,
			HalfOpenLimit: 2,
			ResetTimeout:  10 * time.Millisecond,
		})
		cb.state = OPEN
		cb.lastFailTime = time.Now().Add(-20 * time.Millisecond)

		assert.True(t, cb.allow())
		assert.Equal(t, HALF_OPEN, cb.state)
		assert.Equal(t, 0, cb.successCount)
		assert.Equal(t, 0, cb.failureCount)
	})

	t.Run("Half-open state - under limit", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerOption{
			MaxFailures:   3,
			HalfOpenLimit: 2,
			ResetTimeout:  10 * time.Second,
		})
		cb.state = HALF_OPEN
		cb.successCount = 1
		assert.True(t, cb.allow())
	})

	t.Run("Half-open state - at limit", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerOption{
			MaxFailures:   3,
			HalfOpenLimit: 2,
			ResetTimeout:  10 * time.Second,
		})
		cb.state = HALF_OPEN
		cb.successCount = 2
		assert.False(t, cb.allow())
	})
}

func TestCircuitBreakerReportResult(t *testing.T) {
	t.Run("Report success in closed state", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerOption{
			MaxFailures:   3,
			HalfOpenLimit: 2,
			ResetTimeout:  10 * time.Second,
		})
		// Default state is treated as CLOSED

		cb.reportResult(true)
		assert.Equal(t, 1, cb.successCount)
		assert.Equal(t, 0, cb.failureCount)
		assert.Equal(t, "", cb.state) // State should remain closed
	})

	t.Run("Report failure in closed state - below limit", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerOption{
			MaxFailures:   3,
			HalfOpenLimit: 2,
			ResetTimeout:  10 * time.Second,
		})

		cb.reportResult(false)
		assert.Equal(t, 0, cb.successCount)
		assert.Equal(t, 1, cb.failureCount)
		assert.Equal(t, "", cb.state)             // State should remain closed
		assert.False(t, cb.lastFailTime.IsZero()) // lastFailTime should be set
	})

	t.Run("Report failure in closed state - at limit", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerOption{
			MaxFailures:   3,
			HalfOpenLimit: 2,
			ResetTimeout:  10 * time.Second,
		})
		cb.failureCount = 2

		cb.reportResult(false)
		assert.Equal(t, 0, cb.successCount)
		assert.Equal(t, 3, cb.failureCount)
		assert.Equal(t, OPEN, cb.state) // State should change to OPEN
	})

	t.Run("Report failure in half-open state", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerOption{
			MaxFailures:   3,
			HalfOpenLimit: 2,
			ResetTimeout:  10 * time.Second,
		})
		cb.state = HALF_OPEN

		cb.reportResult(false)
		assert.Equal(t, 0, cb.successCount)
		assert.Equal(t, 1, cb.failureCount)
		assert.Equal(t, OPEN, cb.state) // State should change to OPEN
	})

	t.Run("Report success in half-open state - below limit", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerOption{
			MaxFailures:   3,
			HalfOpenLimit: 2,
			ResetTimeout:  10 * time.Second,
		})
		cb.state = HALF_OPEN

		cb.reportResult(true)
		assert.Equal(t, 1, cb.successCount)
		assert.Equal(t, 0, cb.failureCount)
		assert.Equal(t, HALF_OPEN, cb.state) // State should remain HALF_OPEN
	})

	t.Run("Report success in half-open state - at limit", func(t *testing.T) {
		cb := NewCircuitBreaker(CircuitBreakerOption{
			MaxFailures:   3,
			HalfOpenLimit: 2,
			ResetTimeout:  10 * time.Second,
		})
		cb.state = HALF_OPEN
		cb.successCount = 1

		cb.reportResult(true)
		assert.Equal(t, 2, cb.successCount)
		assert.Equal(t, 0, cb.failureCount)
		assert.Equal(t, CLOSED, cb.state) // State should change to CLOSED
	})
}
