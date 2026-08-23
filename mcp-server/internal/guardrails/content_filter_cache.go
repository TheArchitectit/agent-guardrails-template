package guardrails

import (
	"container/list"
	"sync"
	"time"
)

// DefaultCacheMaxSize is the default upper bound on cached entries.
const DefaultCacheMaxSize = 10000

// ResultCache provides TTL-based caching for classification results.
//
// It is bounded by a max-size cap: when Set pushes the entry count above the
// cap, the oldest entries (by insertion order) are evicted. Entries also expire
// via TTL and are pruned by a background goroutine. Stop() tears down that
// goroutine so it does not leak.
type ResultCache struct {
	mu       sync.RWMutex
	entries  map[string]*list.Element
	order    *list.List // tracks insertion order; front = oldest
	ttl      time.Duration
	maxSize  int
	stopCh   chan struct{}
	stopOnce sync.Once
}

type cacheEntry struct {
	key       string
	result    *ClassificationResult
	timestamp time.Time
}

// NewResultCache creates a cache with the specified TTL and starts a background
// goroutine that periodically prunes expired entries. The returned cache must be
// released with Stop() to avoid leaking the goroutine.
func NewResultCache(ttl time.Duration) *ResultCache {
	return newResultCache(ttl, DefaultCacheMaxSize)
}

// newResultCache is the internal constructor that allows overriding maxSize
// (used by tests and WithCacheMaxSize).
func newResultCache(ttl time.Duration, maxSize int) *ResultCache {
	if maxSize <= 0 {
		maxSize = DefaultCacheMaxSize
	}
	rc := &ResultCache{
		entries: make(map[string]*list.Element),
		order:   list.New(),
		ttl:     ttl,
		maxSize: maxSize,
		stopCh:  make(chan struct{}),
	}
	go rc.pruneLoop()
	return rc
}

func (rc *ResultCache) pruneLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-rc.stopCh:
			return
		case <-ticker.C:
			rc.prune()
		}
	}
}

// Stop tears down the background prune goroutine. Safe to call multiple times.
func (rc *ResultCache) Stop() {
	rc.stopOnce.Do(func() {
		close(rc.stopCh)
	})
}

// Get retrieves a cached result if it exists and hasn't expired.
// Returns a deep copy so concurrent callers do not share mutable state.
func (rc *ResultCache) Get(key string) *ClassificationResult {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	elem, ok := rc.entries[key]
	if !ok {
		return nil
	}

	entry := elem.Value.(*cacheEntry)
	if time.Since(entry.timestamp) > rc.ttl {
		return nil
	}

	return copyResult(entry.result)
}

// Set stores a result in the cache, evicting the oldest entries if the
// configured max-size cap is exceeded.
func (rc *ResultCache) Set(key string, result *ClassificationResult) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if elem, ok := rc.entries[key]; ok {
		// Replace value and move to most-recent position.
		entry := elem.Value.(*cacheEntry)
		entry.result = result
		entry.timestamp = time.Now()
		rc.order.MoveToBack(elem)
		return
	}

	entry := &cacheEntry{
		key:       key,
		result:    result,
		timestamp: time.Now(),
	}
	elem := rc.order.PushBack(entry)
	rc.entries[key] = elem

	// Evict oldest entries until within the cap.
	for rc.order.Len() > rc.maxSize {
		oldest := rc.order.Front()
		if oldest == nil {
			break
		}
		rc.removeElem(oldest)
	}
}

// Clear removes all cached entries.
func (rc *ResultCache) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.entries = make(map[string]*list.Element)
	rc.order.Init()
}

// Len returns the current number of cached entries.
func (rc *ResultCache) Len() int {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.order.Len()
}

func (rc *ResultCache) removeElem(elem *list.Element) {
	entry := elem.Value.(*cacheEntry)
	delete(rc.entries, entry.key)
	rc.order.Remove(elem)
}

func (rc *ResultCache) prune() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	now := time.Now()
	// Walk from oldest (front) and stop at the first non-expired entry.
	for elem := rc.order.Front(); elem != nil; {
		entry := elem.Value.(*cacheEntry)
		if now.Sub(entry.timestamp) > rc.ttl {
			next := elem.Next()
			rc.removeElem(elem)
			elem = next
		} else {
			break
		}
	}
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
