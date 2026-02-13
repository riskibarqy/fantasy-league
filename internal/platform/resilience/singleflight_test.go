package resilience

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSingleFlight_Do(t *testing.T) {
	var g SingleFlight
	var counter int32

	const workers = 20
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			<-start
			_, err, _ := g.Do("token-key", func() (any, error) {
				atomic.AddInt32(&counter, 1)
				time.Sleep(20 * time.Millisecond)
				return "ok", nil
			})
			if err != nil {
				t.Errorf("singleflight call failed: %v", err)
			}
		}()
	}

	close(start)
	wg.Wait()

	if got := atomic.LoadInt32(&counter); got != 1 {
		t.Fatalf("expected function to run once, got %d", got)
	}
}
