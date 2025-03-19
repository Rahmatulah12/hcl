package hcl

import (
	"sync"
	"time"
)

const (
	CLOSED    = "CLOSED"
	HALF_OPEN = "HALF-OPEN"
	OPEN      = "OPEN"
)

type CircuitBreaker struct {
	mu            sync.Mutex
	failureCount  int
	successCount  int
	state         string
	lastFailTime  time.Time
	maxFailures   int
	resetTimeout  time.Duration
	halfOpenLimit int
}

type CircuitBreakerOption struct {
	MaxFailures   int
	HalfOpenLimit int
	ResetTimeout  time.Duration
}

func NewCircuitBreaker(options CircuitBreakerOption) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:   options.MaxFailures,
		halfOpenLimit: options.HalfOpenLimit,
		resetTimeout:  options.ResetTimeout,
	}
}

func (cb *CircuitBreaker) allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case OPEN:
		if time.Since(cb.lastFailTime) > cb.resetTimeout {

			cb.state = HALF_OPEN
			cb.successCount = 0
			cb.failureCount = 0

			return true
		}
		return false
	case HALF_OPEN:
		return cb.successCount < cb.halfOpenLimit
	default: // CLOSED state
		return true
	}
}

// ReportResult updates the circuit breaker state based on success or failure
func (cb *CircuitBreaker) reportResult(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch success {
	case false:
		cb.failureCount++
		cb.lastFailTime = time.Now()

		if cb.state == HALF_OPEN {
			cb.state = OPEN
			return
		}

		if cb.failureCount >= cb.maxFailures {
			cb.state = OPEN
			return
		}
	default:
		cb.successCount++
		if cb.state == HALF_OPEN && cb.successCount >= cb.halfOpenLimit {
			cb.state = CLOSED
		}
	}
}
