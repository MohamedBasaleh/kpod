package kube

import (
	"sync"
	"time"
)

// CacheTTL controls short-lived in-memory discovery cache lifetimes.
type CacheTTL struct {
	Contexts   time.Duration
	Namespaces time.Duration
	Pods       time.Duration
}

func DefaultCacheTTL() CacheTTL {
	return CacheTTL{
		Contexts:   30 * time.Second,
		Namespaces: 20 * time.Second,
		Pods:       4 * time.Second,
	}
}

type cacheEntry struct {
	value any
	until time.Time
}

type memoryCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

func newMemoryCache() *memoryCache {
	return &memoryCache{entries: map[string]cacheEntry{}}
}

func (c *memoryCache) get(now time.Time, key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[key]
	if !ok || !now.Before(e.until) || e.until.IsZero() {
		return nil, false
	}
	return e.value, true
}

func (c *memoryCache) delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

func (c *memoryCache) set(now time.Time, key string, value any, ttl time.Duration) {
	if ttl <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = cacheEntry{value: value, until: now.Add(ttl)}
}
