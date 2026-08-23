// Package guardrails provides semantic content filtering for the MCP server.
//
// This module implements Spec 02: Semantic Content Filtering — a multi-classifier
// system that evaluates text against configurable safety policies before it
// reaches the agent or leaves the agent's outputs.
//
// The taxonomy extends Llama Guard S1-S13 with two coding-specific categories:
// S14 (Code Abuse) and S15 (Data Exfiltration).
package guardrails

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ContentDirection indicates whether the text is entering or leaving the agent.
type ContentDirection string

const (
	DirectionInput  ContentDirection = "input"
	DirectionOutput ContentDirection = "output"
)

// CategoryResult is the classification result for a single safety category.
type CategoryResult struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Score  float64 `json:"score"`
	Action Action  `json:"action"`
	Reason string  `json:"reason,omitempty"`
}

// ClassificationResult is the complete output of content classification.
type ClassificationResult struct {
	Safe        bool             `json:"safe"`
	OverallRisk float64          `json:"overall_risk"`
	Categories  []CategoryResult `json:"categories"`
	Backend     string           `json:"backend"`
	LatencyMs   int64            `json:"latency_ms"`
	Direction   ContentDirection `json:"direction,omitempty"`
}

// IsBlocked returns true if any category triggered a block action.
func (r *ClassificationResult) IsBlocked() bool {
	for _, c := range r.Categories {
		if c.Action == ActionBlock {
			return true
		}
	}
	return false
}

// ContentViolation represents a policy violation detected during classification.
type ContentViolation struct {
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Score        float64 `json:"score"`
	Action       Action  `json:"action"`
	Reason       string  `json:"reason"`
}

// PolicyResult is the output of checking text against a specific policy.
type PolicyResult struct {
	PolicyID   string             `json:"policy_id"`
	Compliant  bool               `json:"compliant"`
	Violations []ContentViolation `json:"violations"`
}

// SemanticClassifier defines the interface for semantic content classification backends.
type SemanticClassifier interface {
	Name() string
	Classify(ctx context.Context, text string) (map[string]float64, error)
	Available(ctx context.Context) bool
}

// ContentFilter orchestrates classification and policy enforcement.
type ContentFilter struct {
	backends     []SemanticClassifier
	policyEngine *PolicyEngine
	cache        *ResultCache
	config       *ContentFilterConfig
	mu           sync.RWMutex
}

// NewContentFilter creates a content filter with the specified backends and policies.
func NewContentFilter(backends []SemanticClassifier, policies []PolicyRule, opts ...FilterOption) *ContentFilter {
	cfg := DefaultFilterConfig()
	cf := &ContentFilter{
		backends:     backends,
		policyEngine: NewPolicyEngineWithThresholds(policies, cfg.Thresholds),
		cache:        NewResultCache(60 * time.Second),
		config:       cfg,
	}
	for _, opt := range opts {
		opt(cf)
	}
	// Rebuild policy engine if config was overridden
	if cf.config != nil {
		cf.policyEngine = NewPolicyEngineWithThresholds(policies, cf.config.Thresholds)
	}
	return cf
}

// FilterOption configures the content filter.
type FilterOption func(*ContentFilter)

// WithCacheTTL overrides the default cache TTL (60s).
func WithCacheTTL(ttl time.Duration) FilterOption {
	return func(cf *ContentFilter) {
		cf.cache = NewResultCache(ttl)
	}
}

// WithCacheMaxSize overrides the default cache entry cap (DefaultCacheMaxSize).
func WithCacheMaxSize(maxSize int) FilterOption {
	return func(cf *ContentFilter) {
		cf.cache = newResultCache(cf.cache.ttl, maxSize)
	}
}

// WithConfig sets the content filter configuration.
func WithConfig(cfg *ContentFilterConfig) FilterOption {
	return func(cf *ContentFilter) {
		cf.config = cfg
	}
}

// Classify performs end-to-end content classification.
func (cf *ContentFilter) Classify(ctx context.Context, text string, direction ContentDirection) (*ClassificationResult, error) {
	if text == "" {
		return &ClassificationResult{
			Safe:      true,
			Backend:   "none",
			LatencyMs: 0,
			Direction: direction,
		}, nil
	}

	cacheKey := cf.cacheKey(text, direction)
	if cached := cf.cache.Get(cacheKey); cached != nil {
		return cached, nil
	}

	startTime := time.Now()
	var lastErr error
	var categoryScores map[string]float64
	var backendUsed string

	// Take a snapshot of backends under read lock to avoid data race
	// with concurrent AddContentFilterBackend calls.
	cf.mu.RLock()
	backends := make([]SemanticClassifier, len(cf.backends))
	copy(backends, cf.backends)
	cf.mu.RUnlock()

	for _, backend := range backends {
		if !backend.Available(ctx) {
			slog.Debug("Backend unavailable, skipping",
				"backend", backend.Name(),
			)
			continue
		}

		scores, err := backend.Classify(ctx, text)
		if err != nil {
			slog.Warn("Backend classification failed",
				"backend", backend.Name(),
				"error", err,
			)
			lastErr = err
			continue
		}
		categoryScores = scores
		backendUsed = backend.Name()
		break
	}

	if categoryScores == nil {
		if cf.config.FailPolicy == ContentFailPolicyBlock {
			// Fail-closed: block content when backends unavailable.
			// Return the block result without an error — the result IS the decision.
			errMsg := "all backends unavailable"
			if lastErr != nil {
				errMsg = fmt.Sprintf("all backends failed: %v", lastErr)
			}
			return &ClassificationResult{
				Safe:        false,
				OverallRisk: 1.0,
				Categories: []CategoryResult{{
					ID:     "SYSTEM",
					Name:   "System Error",
					Score:  1.0,
					Action: ActionBlock,
					Reason: errMsg,
				}},
				Backend:   "fail-closed",
				LatencyMs: time.Since(startTime).Milliseconds(),
				Direction: direction,
			}, nil
		}
		slog.Warn("All classification backends failed, failing open",
			"text_len", len(text),
		)
		return &ClassificationResult{
			Safe:      true,
			Backend:   "fail-open",
			LatencyMs: time.Since(startTime).Milliseconds(),
			Direction: direction,
		}, nil
	}

	result := cf.policyEngine.Evaluate(categoryScores, backendUsed)
	result.LatencyMs = time.Since(startTime).Milliseconds()
	result.Direction = direction

	cf.cache.Set(cacheKey, result)

	// Return a copy so the caller cannot mutate the cached entry.
	return copyResult(result), nil
}

// Stop releases the background cache prune goroutine.
func (cf *ContentFilter) Stop() {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	if cf.cache != nil {
		cf.cache.Stop()
	}
}

// AddContentFilterBackend appends a semantic classification backend. Adding a
// backend changes which classification results are produced, so the cache is
// invalidated to avoid serving stale results.
func (cf *ContentFilter) AddContentFilterBackend(backend SemanticClassifier) {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	cf.backends = append(cf.backends, backend)
	if cf.cache != nil {
		cf.cache.Clear()
	}
}

// UpdateRules replaces the policy rules used for classification. Because rule
// changes can flip category actions, the cache is invalidated so stale results
// are not served until the TTL expires.
func (cf *ContentFilter) UpdateRules(rules []PolicyRule) {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	cf.policyEngine = NewPolicyEngineWithThresholds(rules, cf.config.Thresholds)
	if cf.cache != nil {
		cf.cache.Clear()
	}
}

// CheckPolicy checks if text complies with a specific named policy.
func (cf *ContentFilter) CheckPolicy(ctx context.Context, text string, policyID string) (*PolicyResult, error) {
	result, err := cf.Classify(ctx, text, DirectionOutput)
	if err != nil {
		return nil, err
	}

	return cf.policyEngine.CheckPolicy(result, policyID), nil
}

func (cf *ContentFilter) cacheKey(text string, direction ContentDirection) string {
	h := sha256.New()
	h.Write([]byte(text))
	h.Write([]byte(direction))
	return hex.EncodeToString(h.Sum(nil))
}
