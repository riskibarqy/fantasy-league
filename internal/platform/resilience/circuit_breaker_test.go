package resilience

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_BasicTransitions(t *testing.T) {
	b := NewCircuitBreaker(2, 5*time.Second, 1)

	now := time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC)
	b.now = func() time.Time { return now }

	if err := b.Allow(); err != nil {
		t.Fatalf("expected allow in closed state: %v", err)
	}

	b.RecordFailure()
	if state := b.State(); state != CircuitStateClosed {
		t.Fatalf("expected closed after first failure, got %s", state)
	}

	b.RecordFailure()
	if state := b.State(); state != CircuitStateOpen {
		t.Fatalf("expected open after threshold failures, got %s", state)
	}

	if err := b.Allow(); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected circuit open error, got %v", err)
	}

	now = now.Add(6 * time.Second)
	if err := b.Allow(); err != nil {
		t.Fatalf("expected half-open probe to pass, got %v", err)
	}
	if state := b.State(); state != CircuitStateHalfOpen {
		t.Fatalf("expected half-open state, got %s", state)
	}

	b.RecordSuccess()
	if state := b.State(); state != CircuitStateClosed {
		t.Fatalf("expected closed after successful half-open probe, got %s", state)
	}
}
