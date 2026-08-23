package guardrails

import (
	"sync"
	"time"
)

// ResultCache provides TTL-based caching for classification results.
type ResultCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

type cacheEntry struct {
	result    *ClassificationResult
	timestamp time.Time
}

// NewResultCache creates a cache with the specified TTL and starts a background
// goroutine that periodically prunes expired entries.
func NewResultCache(ttl time.Duration) *ResultCache {
	rc := &ResultCache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
	}
	go rc.pruneLoop()
	return rc
}

func (rc *ResultCache) pruneLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		rc.prune()
	}
}

// Get retrieves a cached result if it exists and hasn't expired.
// Returns a deep copy so concurrent callers do not share mutable state.
func (rc *ResultCache) Get(key string) *ClassificationResult {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	entry, ok := rc.entries[key]
	if !ok {
		return nil
	}

	if time.Since(entry.timestamp) > rc.ttl {
		return nil
	}

	return copyResult(entry.result)
}

// Set stores a result in the cache.
func (rc *ResultCache) Set(key string, result *ClassificationResult) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.entries[key] = cacheEntry{result: result, timestamp: time.Now()}
}

func copyResult(r *ClassificationResult) *ClassificationResult {
	if r == nil {
		return nil
	}
	categories := make([]CategoryResult, len(r.Categories))
	copy(categories, r.Categories)
	return &ClassificationResult{
		Safe:        r.Safe,
		OverallRisk: r.OverallRisk,
		Categories:  categories,
		Backend:     r.Backend,
		LatencyMs:   r.LatencyMs,
		Direction:   r.Direction,
	}
}

// Clear removes all cached entries.
func (rc *ResultCache) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.entries = make(map[string]cacheEntry)
}

func (rc *ResultCache) prune() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	now := time.Now()
	for key, entry := range rc.entries {
		if now.Sub(entry.timestamp) > rc.ttl {
			delete(rc.entries, key)
		}
	}
}
