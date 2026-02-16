package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestStore_GetOrLoad_UsesSingleFlight(t *testing.T) {
	t.Parallel()

	store := NewStore(time.Minute)
	var calls atomic.Int32

	loader := func(context.Context) (any, error) {
		calls.Add(1)
		time.Sleep(20 * time.Millisecond)
		return "value", nil
	}

	const workers = 32
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(workers)
	errCh := make(chan error, workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			<-start
			v, err := store.GetOrLoad(context.Background(), "same-key", loader)
			if err != nil {
				errCh <- err
				return
			}
			if got, _ := v.(string); got != "value" {
				errCh <- errUnexpectedValue
			}
		}()
	}

	close(start)
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("loader called %d times, want 1", got)
	}
}

func TestStore_GetOrLoad_UsesCachedValueAfterFirstLoad(t *testing.T) {
	t.Parallel()

	store := NewStore(time.Minute)
	var calls atomic.Int32

	loader := func(context.Context) (any, error) {
		calls.Add(1)
		return "cached", nil
	}

	if _, err := store.GetOrLoad(context.Background(), "k", loader); err != nil {
		t.Fatalf("first GetOrLoad error: %v", err)
	}
	if _, err := store.GetOrLoad(context.Background(), "k", loader); err != nil {
		t.Fatalf("second GetOrLoad error: %v", err)
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("loader called %d times, want 1", got)
	}
}

var errUnexpectedValue = errors.New("unexpected loaded value")
