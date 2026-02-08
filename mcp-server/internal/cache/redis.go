package cache

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
)

// Client wraps Redis client with guardrail-specific operations
type Client struct {
	client *redis.Client
	ttl    time.Duration
}

// New creates a new Redis client
func New(cfg *config.Config) (*Client, error) {
	opts := &redis.Options{
		Addr:         cfg.RedisAddr(),
		Password:     cfg.RedisPassword,
		DB:           0,
		PoolSize:     20,
		MinIdleConns: 5,
		MaxRetries:   3,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	// TLS for production
	if cfg.RedisUseTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: cfg.RedisHost,
		}
	}

	client := redis.NewClient(opts)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	slog.Info("Redis connected", "addr", cfg.RedisAddr())

	return &Client{
		client: client,
		ttl:    5 * time.Minute,
	}, nil
}

// HealthCheck verifies Redis connectivity
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return c.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (c *Client) Close() error {
	slog.Info("Closing Redis connection")
	return c.client.Close()
}

// Get retrieves a value from cache
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	return c.client.Get(ctx, key).Bytes()
}

// Set stores a value in cache
func (c *Client) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl == 0 {
		ttl = c.ttl
	}
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Delete removes a key from cache
func (c *Client) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Cache keys
const (
	KeyActiveRules    = "guardrail:rules:active"
	KeyDocument       = "guardrail:doc:%s"     // Format with slug
	KeyRule           = "guardrail:rule:%s"    // Format with rule_id
	KeyProjectContext = "guardrail:project:%s" // Format with slug
	KeySearchResults  = "guardrail:search:%s"  // Format with query hash
	KeySession        = "guardrail:session:%s" // Format with token
)

// GetActiveRules retrieves cached active rules
func (c *Client) GetActiveRules(ctx context.Context) ([]byte, error) {
	return c.Get(ctx, KeyActiveRules)
}

// SetActiveRules caches active rules
func (c *Client) SetActiveRules(ctx context.Context, data []byte, ttl time.Duration) error {
	return c.Set(ctx, KeyActiveRules, data, ttl)
}

// InvalidateOnRuleChange clears rule-related caches
func (c *Client) InvalidateOnRuleChange(ctx context.Context, ruleID string) error {
	// Use a timeout context to prevent long-running operations
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pipe := c.client.Pipeline()

	// Delete specific rule cache
	pipe.Del(ctx, fmt.Sprintf(KeyRule, ruleID))

	// Delete active rules list
	pipe.Del(ctx, KeyActiveRules)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to invalidate rule cache: %w", err)
	}

	// Delete search result caches using SCAN instead of KEYS for production safety
	// KEYS is blocking and should not be used in production
	return c.deleteKeysByPattern(ctx, "guardrail:search:*")
}

// InvalidateOnDocumentChange clears doc-related caches
func (c *Client) InvalidateOnDocumentChange(ctx context.Context, slug string) error {
	// Use a timeout context to prevent long-running operations
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pipe := c.client.Pipeline()

	// Delete specific document cache
	pipe.Del(ctx, fmt.Sprintf(KeyDocument, slug))

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to invalidate document cache: %w", err)
	}

	// Delete search result caches using SCAN instead of KEYS for production safety
	return c.deleteKeysByPattern(ctx, "guardrail:search:*")
}

// InvalidateOnProjectChange clears project caches
func (c *Client) InvalidateOnProjectChange(ctx context.Context, slug string) error {
	return c.Delete(ctx, fmt.Sprintf(KeyProjectContext, slug))
}

// DistributedRateLimiter implements distributed rate limiting
type DistributedRateLimiter struct {
	redis  *redis.Client
	window time.Duration
}

// NewDistributedLimiter creates a new distributed rate limiter
func (c *Client) NewDistributedLimiter() *DistributedRateLimiter {
	return &DistributedRateLimiter{
		redis:  c.client,
		window: time.Minute,
	}
}

// Allow checks if a request is allowed under the rate limit
func (dl *DistributedRateLimiter) Allow(ctx context.Context, key string, limit int) bool {
	// Sliding window counter in Redis
	now := time.Now().Unix()
	windowKey := fmt.Sprintf("ratelimit:%s:%d", key, now/60)

	pipe := dl.redis.Pipeline()
	incr := pipe.Incr(ctx, windowKey)
	pipe.Expire(ctx, windowKey, dl.window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		// Fail closed on Redis error - security first
		slog.Error("Rate limiting Redis error, failing closed", "error", err)
		return false
	}

	return incr.Val() <= int64(limit)
}

// PubSub provides access to Redis Pub/Sub for cache coordination
func (c *Client) PubSub(ctx context.Context) *redis.PubSub {
	return c.client.Subscribe(ctx, "cache:invalidations")
}

// Publish sends a message to a channel
func (c *Client) Publish(ctx context.Context, channel string, message interface{}) error {
	return c.client.Publish(ctx, channel, message).Err()
}

// InvalidationMessage represents a cache invalidation event
type InvalidationMessage struct {
	Type   string `json:"type"`
	ID     string `json:"id"`
	Source string `json:"source"`
}

// BroadcastInvalidation sends an invalidation message to all instances
func (c *Client) BroadcastInvalidation(ctx context.Context, msg InvalidationMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.Publish(ctx, "cache:invalidations", data)
}

// deleteKeysByPattern safely deletes keys matching a pattern using SCAN
// This is a non-blocking alternative to KEYS for production use
func (c *Client) deleteKeysByPattern(ctx context.Context, pattern string) error {
	var cursor uint64
	var keys []string

	// Use SCAN to iterate through keys in a non-blocking way
	for {
		var err error
		keys, cursor, err = c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}

		// Delete keys in batch if any found
		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				slog.Warn("Failed to delete some keys during cache invalidation", "error", err)
				// Continue even if some deletions fail
			}
		}

		// Exit when cursor returns to 0
		if cursor == 0 {
			break
		}
	}

	return nil
}
