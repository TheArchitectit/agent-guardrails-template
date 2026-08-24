package mcp

import (
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
)

// getTeamManagerPath returns the absolute path to the team_manager.py script
// This ensures the script can be found regardless of the working directory
func getTeamManagerPath() string {
	// Get the directory of this Go source file
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	// Navigate from mcp-server/internal/mcp/ to repo root, then to scripts/
	// Path: mcp-server/internal/mcp/ -> ../../../scripts/
	return filepath.Join(dir, "..", "..", "..", "scripts", "team_manager.py")
}

// getRepoRoot returns the absolute path to the repo root directory
func getRepoRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	// Navigate from mcp-server/internal/mcp/ to repo root
	return filepath.Join(dir, "..", "..", "..")
}

// SEC-005: Rate limiting configuration
const (
	defaultRateLimitRequests = 100
	defaultRateLimitWindow   = 60 // seconds
)

// rateBucket represents a token bucket for rate limiting
type rateBucket struct {
	tokens    int
	lastReset time.Time
}

// rateLimiter implements token bucket rate limiting
type rateLimiter struct {
	mu            sync.RWMutex
	buckets       map[string]*rateBucket
	requestsLimit int
	windowSeconds int
}

// globalRateLimiter is the singleton rate limiter instance
var globalRateLimiter = &rateLimiter{
	buckets:       make(map[string]*rateBucket),
	requestsLimit: defaultRateLimitRequests,
	windowSeconds: defaultRateLimitWindow,
}

// checkRateLimit checks if a request is allowed for the given user
// Returns (allowed, rateLimitHeaders)
func (rl *rateLimiter) checkRateLimit(userID string) (bool, map[string]string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.buckets[userID]

	if !exists || now.Sub(bucket.lastReset) >= time.Duration(rl.windowSeconds)*time.Second {
		// Create new bucket or reset expired bucket
		rl.buckets[userID] = &rateBucket{
			tokens:    rl.requestsLimit - 1, // Consume one token
			lastReset: now,
		}
		remaining := rl.requestsLimit - 1
		resetTime := now.Add(time.Duration(rl.windowSeconds) * time.Second).Unix()
		return true, map[string]string{
			"X-RateLimit-Limit":     strconv.Itoa(rl.requestsLimit),
			"X-RateLimit-Remaining": strconv.Itoa(remaining),
			"X-RateLimit-Reset":     strconv.Itoa(int(resetTime)),
		}
	}

	// Check if tokens available
	if bucket.tokens <= 0 {
		resetTime := bucket.lastReset.Add(time.Duration(rl.windowSeconds) * time.Second).Unix()
		return false, map[string]string{
			"X-RateLimit-Limit":     strconv.Itoa(rl.requestsLimit),
			"X-RateLimit-Remaining": "0",
			"X-RateLimit-Reset":     strconv.Itoa(int(resetTime)),
		}
	}

	// Consume token
	bucket.tokens--
	remaining := bucket.tokens
	resetTime := bucket.lastReset.Add(time.Duration(rl.windowSeconds) * time.Second).Unix()
	return true, map[string]string{
		"X-RateLimit-Limit":     strconv.Itoa(rl.requestsLimit),
		"X-RateLimit-Remaining": strconv.Itoa(remaining),
		"X-RateLimit-Reset":     strconv.Itoa(int(resetTime)),
	}
}

// cleanupOldBuckets removes expired buckets (call periodically)
func (rl *rateLimiter) cleanupOldBuckets() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	window := time.Duration(rl.windowSeconds) * time.Second

	for userID, bucket := range rl.buckets {
		if now.Sub(bucket.lastReset) > window*2 {
			delete(rl.buckets, userID)
		}
	}
}

// validateProjectName validates project name to prevent command injection
