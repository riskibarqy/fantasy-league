package resilience

import (
	"errors"
	"sync"
	"time"
)

var ErrCircuitOpen = errors.New("circuit breaker is open")

type CircuitState string

const (
	CircuitStateClosed   CircuitState = "closed"
	CircuitStateOpen     CircuitState = "open"
	CircuitStateHalfOpen CircuitState = "half_open"
)

// CircuitBreaker is a small stateful breaker for dependency protection.
type CircuitBreaker struct {
	mu sync.Mutex

	failureThreshold int
	openTimeout      time.Duration
	halfOpenMaxReq   int

	state               CircuitState
	consecutiveFailures int
	openedAt            time.Time
	halfOpenInFlight    int
	halfOpenSuccesses   int
	now                 func() time.Time
}

func NewCircuitBreaker(failureThreshold int, openTimeout time.Duration, halfOpenMaxReq int) *CircuitBreaker {
	if failureThreshold < 1 {
		failureThreshold = 1
	}
	if openTimeout <= 0 {
		openTimeout = 15 * time.Second
	}
	if halfOpenMaxReq < 1 {
		halfOpenMaxReq = 1
	}

	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		openTimeout:      openTimeout,
		halfOpenMaxReq:   halfOpenMaxReq,
		state:            CircuitStateClosed,
		now:              time.Now,
	}
}

func (b *CircuitBreaker) Allow() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := b.now()
	if b.state == CircuitStateOpen {
		if now.Sub(b.openedAt) < b.openTimeout {
			return ErrCircuitOpen
		}
		b.toHalfOpen()
	}

	if b.state == CircuitStateHalfOpen {
		if b.halfOpenInFlight >= b.halfOpenMaxReq {
			return ErrCircuitOpen
		}
		b.halfOpenInFlight++
	}

	return nil
}

func (b *CircuitBreaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case CircuitStateClosed:
		b.consecutiveFailures = 0
	case CircuitStateHalfOpen:
		if b.halfOpenInFlight > 0 {
			b.halfOpenInFlight--
		}
		b.halfOpenSuccesses++
		if b.halfOpenSuccesses >= b.halfOpenMaxReq && b.halfOpenInFlight == 0 {
			b.toClosed()
		}
	}
}

func (b *CircuitBreaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case CircuitStateClosed:
		b.consecutiveFailures++
		if b.consecutiveFailures >= b.failureThreshold {
			b.toOpen()
		}
	case CircuitStateHalfOpen:
		if b.halfOpenInFlight > 0 {
			b.halfOpenInFlight--
		}
		b.toOpen()
	case CircuitStateOpen:
		b.openedAt = b.now()
	}
}

func (b *CircuitBreaker) State() CircuitState {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state == CircuitStateOpen {
		if b.now().Sub(b.openedAt) >= b.openTimeout {
			return CircuitStateHalfOpen
		}
	}

	return b.state
}

func (b *CircuitBreaker) toClosed() {
	b.state = CircuitStateClosed
	b.consecutiveFailures = 0
	b.halfOpenInFlight = 0
	b.halfOpenSuccesses = 0
	b.openedAt = time.Time{}
}

func (b *CircuitBreaker) toOpen() {
	b.state = CircuitStateOpen
	b.openedAt = b.now()
	b.halfOpenInFlight = 0
	b.halfOpenSuccesses = 0
}

func (b *CircuitBreaker) toHalfOpen() {
	b.state = CircuitStateHalfOpen
	b.halfOpenInFlight = 0
	b.halfOpenSuccesses = 0
}
