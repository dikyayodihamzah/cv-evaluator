package resilience

import (
	"errors"
	"sync"
	"time"
)

type CircuitState int

const (
	Closed CircuitState = iota
	Open
	HalfOpen
)

type CircuitBreaker struct {
	maxFailures  int
	timeout      time.Duration
	state        CircuitState
	failures     int
	lastFailTime time.Time
	mutex        sync.RWMutex
}

type CircuitBreakerOption func(*CircuitBreaker)

func NewCircuitBreaker(maxFailures int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures: maxFailures,
		timeout:     timeout,
		state:       Closed,
	}
}

var ErrCircuitBreakerOpen = errors.New("circuit breaker is open")

func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if cb.state == Open {
		if time.Since(cb.lastFailTime) > cb.timeout {
			cb.state = HalfOpen
		} else {
			return ErrCircuitBreakerOpen
		}
	}

	err := fn()

	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()

		if cb.failures >= cb.maxFailures {
			cb.state = Open
		}
		return err
	}

	// Success
	if cb.state == HalfOpen {
		cb.state = Closed
	}
	cb.failures = 0

	return nil
}

func (cb *CircuitBreaker) State() CircuitState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.state = Closed
	cb.failures = 0
}
