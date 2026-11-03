package guardrails

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
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

// NewInMemoryProvenanceCache creates a new in-memory provenance cache and starts
// a background goroutine that periodically prunes expired entries.
func NewInMemoryProvenanceCache(defaultTTL time.Duration) *InMemoryProvenanceCache {
	c := &InMemoryProvenanceCache{
		entries:    make(map[string]provCacheEntry),
		defaultTTL: defaultTTL,
	}
	go c.pruneLoop()
	return c
}

// pruneLoop runs Prune() on a 30s ticker for the lifetime of the cache.
func (c *InMemoryProvenanceCache) pruneLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		c.Prune()
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

// RedisProvenanceCache is a Redis-backed implementation of ProvenanceCache
// speaking the Redis Serialization Protocol (RESP2) over TCP. It fails closed
// when unconfigured or unreachable rather than fabricating cache hits.
type RedisProvenanceCache struct {
	client  *redisClient
	prefix  string
	ttl     time.Duration
	timeout time.Duration
}

// redisClient is a minimal wrapper around the Redis TCP address.
type redisClient struct {
	addr string
}

// NewRedisProvenanceCache creates a new Redis-backed provenance cache.
func NewRedisProvenanceCache(addr string, prefix string, ttl time.Duration) *RedisProvenanceCache {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	if prefix == "" {
		prefix = "guardrail:prov:"
	}
	return &RedisProvenanceCache{
		client:  &redisClient{addr: strings.TrimSpace(addr)},
		prefix:  prefix,
		ttl:     ttl,
		timeout: 2 * time.Second,
	}
}

func (c *RedisProvenanceCache) key(hash string) string {
	return fmt.Sprintf("%s%s", c.prefix, hash)
}

func (c *RedisProvenanceCache) dial(ctx context.Context) (net.Conn, error) {
	if c == nil || c.client == nil || c.client.addr == "" {
		return nil, fmt.Errorf("redis provenance cache address is empty")
	}
	d := net.Dialer{Timeout: c.timeout}
	conn, err := d.DialContext(ctx, "tcp", c.client.addr)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(c.timeout)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}
	_ = conn.SetDeadline(deadline)
	return conn, nil
}

func writeRESPCommand(w io.Writer, args ...string) error {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("*%d\r\n", len(args)))
	for _, a := range args {
		b.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(a), a))
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func readRESPValue(r *bufio.Reader) (kind byte, str string, arr []string, isNil bool, err error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return 0, "", nil, false, err
	}
	line = strings.TrimRight(line, "\r\n")
	if len(line) == 0 {
		return 0, "", nil, false, fmt.Errorf("empty RESP response line")
	}
	kind = line[0]
	payload := line[1:]
	switch kind {
	case '+':
		return kind, payload, nil, false, nil
	case '-':
		return kind, "", nil, false, fmt.Errorf("redis error: %s", payload)
	case ':':
		return kind, payload, nil, false, nil
	case '$':
		n, err := strconv.Atoi(payload)
		if err != nil {
			return kind, "", nil, false, fmt.Errorf("invalid bulk length %q: %w", payload, err)
		}
		if n < 0 {
			return kind, "", nil, true, nil
		}
		buf := make([]byte, n+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return kind, "", nil, false, err
		}
		return kind, string(buf[:n]), nil, false, nil
	case '*':
		n, err := strconv.Atoi(payload)
		if err != nil {
			return kind, "", nil, false, fmt.Errorf("invalid array length %q: %w", payload, err)
		}
		if n < 0 {
			return kind, "", nil, true, nil
		}
		items := make([]string, 0, n)
		for i := 0; i < n; i++ {
			_, s, _, nilElem, err := readRESPValue(r)
			if err != nil {
				return kind, "", nil, false, err
			}
			if !nilElem {
				items = append(items, s)
			}
		}
		return kind, "", items, false, nil
	default:
		return 0, "", nil, false, fmt.Errorf("unexpected RESP prefix %q", kind)
	}
}

// Get retrieves a provenance entry from Redis.
func (c *RedisProvenanceCache) Get(ctx context.Context, hash string) (*Provenance, bool) {
	if hash == "" {
		return nil, false
	}
	conn, err := c.dial(ctx)
	if err != nil {
		return nil, false
	}
	defer conn.Close()

	if err := writeRESPCommand(conn, "GET", c.key(hash)); err != nil {
		return nil, false
	}
	_, raw, _, isNil, err := readRESPValue(bufio.NewReader(conn))
	if err != nil || isNil || raw == "" {
		return nil, false
	}
	var prov Provenance
	if err := json.Unmarshal([]byte(raw), &prov); err != nil {
		return nil, false
	}
	return &prov, true
}

// Set stores a provenance entry in Redis with a millisecond PX TTL.
func (c *RedisProvenanceCache) Set(ctx context.Context, hash string, prov *Provenance, ttl time.Duration) error {
	if hash == "" {
		return fmt.Errorf("hash cannot be empty")
	}
	if prov == nil {
		return fmt.Errorf("provenance cannot be nil")
	}
	if ttl <= 0 {
		ttl = c.ttl
	}
	ms := ttl.Milliseconds()
	if ms <= 0 {
		ms = 1
	}
	encoded, err := json.Marshal(prov)
	if err != nil {
		return fmt.Errorf("marshal provenance: %w", err)
	}
	conn, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := writeRESPCommand(conn, "SET", c.key(hash), string(encoded), "PX", strconv.FormatInt(ms, 10)); err != nil {
		return err
	}
	_, reply, _, _, err := readRESPValue(bufio.NewReader(conn))
	if err != nil {
		return err
	}
	if reply != "OK" {
		return fmt.Errorf("unexpected redis SET reply %q", reply)
	}
	return nil
}

// Delete removes a provenance entry from Redis.
func (c *RedisProvenanceCache) Delete(ctx context.Context, hash string) error {
	if hash == "" {
		return nil
	}
	conn, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := writeRESPCommand(conn, "DEL", c.key(hash)); err != nil {
		return err
	}
	_, _, _, _, err = readRESPValue(bufio.NewReader(conn))
	return err
}

// Count returns the number of provenance keys matching the cache prefix in Redis.
func (c *RedisProvenanceCache) Count() int {
	conn, err := c.dial(context.Background())
	if err != nil {
		return 0
	}
	defer conn.Close()

	if err := writeRESPCommand(conn, "KEYS", c.prefix+"*"); err != nil {
		return 0
	}
	_, _, items, isNil, err := readRESPValue(bufio.NewReader(conn))
	if err != nil || isNil {
		return 0
	}
	return len(items)
}

// Ensure RedisProvenanceCache implements ProvenanceCache
var _ ProvenanceCache = (*RedisProvenanceCache)(nil)

// Ensure InMemoryProvenanceCache implements ProvenanceCache
var _ ProvenanceCache = (*InMemoryProvenanceCache)(nil)
