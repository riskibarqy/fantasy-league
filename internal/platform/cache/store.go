package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/platform/resilience"
)

type entry struct {
	value     any
	expiresAt time.Time
}

type Store struct {
	mu      sync.RWMutex
	entries map[string]entry
	ttl     time.Duration
	flight  resilience.SingleFlight
}

func NewStore(ttl time.Duration) *Store {
	return &Store{
		entries: make(map[string]entry),
		ttl:     ttl,
	}
}

func (s *Store) Get(_ context.Context, key string) (any, bool) {
	if key == "" {
		return nil, false
	}

	now := time.Now()
	s.mu.RLock()
	e, ok := s.entries[key]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if s.ttl > 0 && !e.expiresAt.After(now) {
		s.mu.Lock()
		delete(s.entries, key)
		s.mu.Unlock()
		return nil, false
	}

	return e.value, true
}

func (s *Store) Set(_ context.Context, key string, value any) {
	if key == "" {
		return
	}

	expiresAt := time.Time{}
	if s.ttl > 0 {
		expiresAt = time.Now().Add(s.ttl)
	}

	s.mu.Lock()
	s.entries[key] = entry{
		value:     value,
		expiresAt: expiresAt,
	}
	s.mu.Unlock()
}

func (s *Store) Delete(_ context.Context, key string) {
	if key == "" {
		return
	}

	s.mu.Lock()
	delete(s.entries, key)
	s.mu.Unlock()
}

func (s *Store) DeletePrefix(_ context.Context, prefix string) {
	if prefix == "" {
		return
	}

	s.mu.Lock()
	for key := range s.entries {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(s.entries, key)
		}
	}
	s.mu.Unlock()
}

func (s *Store) GetOrLoad(ctx context.Context, key string, loader func(context.Context) (any, error)) (any, error) {
	if loader == nil {
		return nil, fmt.Errorf("loader is required")
	}
	if key == "" {
		return loader(ctx)
	}

	if value, ok := s.Get(ctx, key); ok {
		return value, nil
	}

	value, err, _ := s.flight.Do(key, func() (any, error) {
		if cached, ok := s.Get(ctx, key); ok {
			return cached, nil
		}

		loaded, loadErr := loader(ctx)
		if loadErr != nil {
			return nil, loadErr
		}
		s.Set(ctx, key, loaded)
		return loaded, nil
	})
	if err != nil {
		return nil, err
	}

	return value, nil
}
