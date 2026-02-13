package anubis

import (
	"sync"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/domain/user"
)

type cacheEntry struct {
	principal user.Principal
	expiresAt time.Time
}

type inMemoryPrincipalCache struct {
	mu         sync.RWMutex
	entries    map[string]cacheEntry
	ttl        time.Duration
	maxEntries int
}

func newInMemoryPrincipalCache(ttl time.Duration, maxEntries int) *inMemoryPrincipalCache {
	return &inMemoryPrincipalCache{
		entries:    make(map[string]cacheEntry),
		ttl:        ttl,
		maxEntries: maxEntries,
	}
}

func (c *inMemoryPrincipalCache) Get(key string) (user.Principal, bool) {
	now := time.Now()

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return user.Principal{}, false
	}
	if !entry.expiresAt.After(now) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return user.Principal{}, false
	}

	return entry.principal, true
}

func (c *inMemoryPrincipalCache) Set(key string, principal user.Principal) {
	if c.ttl <= 0 {
		return
	}

	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.maxEntries > 0 && len(c.entries) >= c.maxEntries {
		c.evictExpired(now)
		if len(c.entries) >= c.maxEntries {
			c.evictOne()
		}
	}

	c.entries[key] = cacheEntry{
		principal: principal,
		expiresAt: now.Add(c.ttl),
	}
}

func (c *inMemoryPrincipalCache) evictExpired(now time.Time) {
	for key, entry := range c.entries {
		if !entry.expiresAt.After(now) {
			delete(c.entries, key)
		}
	}
}

func (c *inMemoryPrincipalCache) evictOne() {
	for key := range c.entries {
		delete(c.entries, key)
		return
	}
}
