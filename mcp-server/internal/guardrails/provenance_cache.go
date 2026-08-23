package guardrails

import (
	"context"
	"sync"
	"time"
)

// InMemoryProvenanceCache is an in-memory implementation of ProvenanceCache.
// Used for testing; a Redis-backed version would implement the same interface.
type InMemoryProvenanceCache struct {
	mu       sync.RWMutex
	entries  map[string]provCacheEntry
	defaultTTL time.Duration
}

type provCacheEntry struct {
	prov      *Provenance
	expiresAt time.Time
}

// NewInMemoryProvenanceCache creates a new in-memory provenance cache.
func NewInMemoryProvenanceCache(defaultTTL time.Duration) *InMemoryProvenanceCache {
	return &InMemoryProvenanceCache{
		entries:    make(map[string]provCacheEntry),
		defaultTTL: defaultTTL,
	}
}

// Get retrieves a provenance entry from cache.
func (c *InMemoryProvenanceCache) Get(ctx context.Context, hash string) (*Provenance, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[hash]
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry.prov, true
}

// Set stores a provenance entry in cache.
func (c *InMemoryProvenanceCache) Set(ctx context.Context, hash string, prov *Provenance, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = c.defaultTTL
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[hash] = provCacheEntry{
		prov:      prov,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

// Clear removes all entries from cache.
func (c *InMemoryProvenanceCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]provCacheEntry)
}

// Prune removes expired entries.
func (c *InMemoryProvenanceCache) Prune() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	initialLen := len(c.entries)
	for hash, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, hash)
		}
	}

	return initialLen - len(c.entries)
}

// Count returns the number of entries in the cache.
func (c *InMemoryProvenanceCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// RedisProvenanceCache is a Redis-backed implementation of ProvenanceCache.
// This is a placeholder for the actual Redis client integration.
type RedisProvenanceCache struct {
	client  *redisClient
	prefix  string
	ttl     time.Duration
}

// redisClient is a minimal wrapper around the Redis client.
type redisClient struct {
	// In a real implementation, this would be *redis.Client
	addr string
}

// NewRedisProvenanceCache creates a new Redis-backed provenance cache.
func NewRedisProvenanceCache(addr string, prefix string, ttl time.Duration) *RedisProvenanceCache {
	return &RedisProvenanceCache{
		client: &redisClient{addr: addr},
		prefix: prefix,
		ttl:    ttl,
	}
}

// Get retrieves a provenance entry from Redis.
func (c *RedisProvenanceCache) Get(ctx context.Context, hash string) (*Provenance, bool) {
	// In a real implementation, this would query Redis
	// For now, return not found (cache miss)
	return nil, false
}

// Set stores a provenance entry in Redis.
func (c *RedisProvenanceCache) Set(ctx context.Context, hash string, prov *Provenance, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = c.ttl
	}

	// In a real implementation, this would write to Redis with TTL
	// For now, return nil (simulated success)
	return nil
}

// Ensure RedisProvenanceCache implements ProvenanceCache
var _ ProvenanceCache = (*RedisProvenanceCache)(nil)

// Ensure InMemoryProvenanceCache implements ProvenanceCache
var _ ProvenanceCache = (*InMemoryProvenanceCache)(nil)
