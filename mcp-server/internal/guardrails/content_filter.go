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

// RiskLevel maps overall risk scores to human-readable levels.
type RiskLevel string

const (
	RiskLevelNone     RiskLevel = "none"
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
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
	PolicyID   string              `json:"policy_id"`
	Compliant  bool                `json:"compliant"`
	Violations []ContentViolation  `json:"violations"`
}

// SemanticClassifier defines the interface for semantic content classification backends.
type SemanticClassifier interface {
	// Name returns the backend identifier (e.g., "llama-guard-3", "openai-moderation").
	Name() string

	// Classify evaluates text against the safety taxonomy.
	// Returns category scores keyed by category ID (S1-S15).
	Classify(ctx context.Context, text string) (map[string]float64, error)

	// Available returns true if the backend is reachable and ready.
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
//
// Backends are tried in order; if the first fails, the next is used (failover).
func NewContentFilter(backends []SemanticClassifier, policies []PolicyRule, opts ...FilterOption) *ContentFilter {
	cf := &ContentFilter{
		backends:     backends,
		policyEngine: NewPolicyEngine(policies),
		cache:        NewResultCache(60 * time.Second),
		config:       DefaultFilterConfig(),
	}
	for _, opt := range opts {
		opt(cf)
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

// WithConfig sets the content filter configuration.
func WithConfig(cfg *ContentFilterConfig) FilterOption {
	return func(cf *ContentFilter) {
		cf.config = cfg
	}
}

// Classify performs end-to-end content classification.
//
// 1. Check cache for existing result
// 2. Try each backend in order until one succeeds
// 3. Apply policy engine to map categories → actions
// 4. Cache and return the result
func (cf *ContentFilter) Classify(ctx context.Context, text string, direction ContentDirection) (*ClassificationResult, error) {
	if text == "" {
		return &ClassificationResult{
			Safe:      true,
			Backend:   "none",
			LatencyMs: 0,
			Direction: direction,
		}, nil
	}

	// Check cache first
	cacheKey := cf.cacheKey(text, direction)
	if cached := cf.cache.Get(cacheKey); cached != nil {
		return cached, nil
	}

	// Try backends in order
	startTime := time.Now()
	var lastErr error
	var categoryScores map[string]float64
	var backendUsed string

	for _, backend := range cf.backends {
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
		// All backends failed
		if cf.config.FailPolicy == ContentFailPolicyBlock {
			return &ClassificationResult{
				Safe:        false,
				OverallRisk: 1.0,
				Categories: []CategoryResult{{
					ID:     "SYSTEM",
					Name:   "System Error",
					Score:  1.0,
					Action: ActionBlock,
					Reason: "All classification backends unavailable",
				}},
				Backend:   "fail-open",
				LatencyMs: time.Since(startTime).Milliseconds(),
				Direction: direction,
			}, fmt.Errorf("all backends failed: %w", lastErr)
		}
		// Fail open: allow content but log warning
		return &ClassificationResult{
			Safe:      true,
			Backend:   "fail-open",
			LatencyMs: time.Since(startTime).Milliseconds(),
			Direction: direction,
		}, nil
	}

	// Apply policy engine to map categories to actions
	result := cf.policyEngine.Evaluate(categoryScores, backendUsed)
	result.LatencyMs = time.Since(startTime).Milliseconds()
	result.Direction = direction

	// Cache the result
	cf.cache.Set(cacheKey, result)

	return result, nil
}

// CheckPolicy checks if text complies with a specific named policy.
func (cf *ContentFilter) CheckPolicy(ctx context.Context, text string, policyID string) (*PolicyResult, error) {
	result, err := cf.Classify(ctx, text, DirectionOutput)
	if err != nil {
		return nil, err
	}

	return cf.policyEngine.CheckPolicy(result, policyID), nil
}

// StreamingHandler processes text chunks for real-time filtering.
type StreamingHandler struct {
	filter    *ContentFilter
	onBlock   func(result *ClassificationResult)
	onWarn    func(result *ClassificationResult)
	chunkSize int
	mu        sync.Mutex
}

// NewStreamingHandler creates a streaming content filter.
func NewStreamingHandler(filter *ContentFilter, onBlock, onWarn func(*ClassificationResult)) *StreamingHandler {
	return &StreamingHandler{
		filter:    filter,
		onBlock:   onBlock,
		onWarn:    onWarn,
		chunkSize: 512, // characters per chunk
	}
}

// ProcessChunk classifies a text chunk. Returns true if generation should continue.
func (sh *StreamingHandler) ProcessChunk(ctx context.Context, chunk string, direction ContentDirection) bool {
	result, err := sh.filter.Classify(ctx, chunk, direction)
	if err != nil {
		slog.Warn("Streaming chunk classification failed", "error", err)
		return true // continue on error (fail-open for streaming)
	}

	if result.IsBlocked() {
		if sh.onBlock != nil {
			sh.onBlock(result)
		}
		return false // stop generation
	}

	for _, cat := range result.Categories {
		if cat.Action == ActionWarn && sh.onWarn != nil {
			sh.onWarn(result)
			break
		}
	}

	return true
}

// SetChunkSize overrides the default chunk size for streaming.
func (sh *StreamingHandler) SetChunkSize(size int) {
	sh.chunkSize = size
}

// cacheKey generates a deterministic cache key for text + direction.
// cacheKey generates a deterministic cache key for text + direction.
func (cf *ContentFilter) cacheKey(text string, direction ContentDirection) string {
	h := sha256.New()
	h.Write([]byte(text))
	h.Write([]byte(direction))
	return hex.EncodeToString(h.Sum(nil))
}

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

// NewResultCache creates a cache with the specified TTL.
func NewResultCache(ttl time.Duration) *ResultCache {
	return &ResultCache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
	}
}

// Get retrieves a cached result if it exists and hasn't expired.
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

	return entry.result
}

// Set stores a result in the cache.
func (rc *ResultCache) Set(key string, result *ClassificationResult) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.entries[key] = cacheEntry{result: result, timestamp: time.Now()}
}

// Clear removes all cached entries.
func (rc *ResultCache) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.entries = make(map[string]cacheEntry)
}

// prune removes expired entries (called periodically or on access).
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

// === Llama Guard Backend ===

// LlamaGuardBackend implements SemanticClassifier for local Llama Guard models via Ollama.
type LlamaGuardBackend struct {
	model     string
	ollamaURL string
	client    *httpClient
	enabled   bool
}

// NewLlamaGuardBackend creates a Llama Guard backend instance.
func NewLlamaGuardBackend(model, ollamaURL string) *LlamaGuardBackend {
	return &LlamaGuardBackend{
		model:     model,
		ollamaURL: ollamaURL,
		client:    newHTTPClient(30 * time.Second),
		enabled:   true,
	}
}

// Name returns the backend identifier.
func (b *LlamaGuardBackend) Name() string {
	return fmt.Sprintf("llama-guard:%s", b.model)
}

// Available checks if the Ollama service is reachable.
func (b *LlamaGuardBackend) Available(ctx context.Context) bool {
	if !b.enabled {
		return false
	}
	// Health check: GET /api/tags
	return b.client.getHealth(ctx, b.ollamaURL+"/api/tags")
}

// Classify sends text to Llama Guard for classification.
//
// In production, this calls Ollama's /api/generate with the Llama Guard prompt template.
// Returns normalized scores (0-1) for each S1-S15 category.
func (b *LlamaGuardBackend) Classify(ctx context.Context, text string) (map[string]float64, error) {
	// This is the production stub — actual implementation would:
	// 1. Format the Llama Guard prompt with the input text
	// 2. POST to /api/generate
	// 3. Parse the response to extract category scores
	//
	// For now, return empty scores (no violations) as the baseline.
	// Real implementation depends on Ollama's response format.
	return b.parseResponse(text), nil
}

// parseResponse extracts category scores from Llama Guard output.
func (b *LlamaGuardBackend) parseResponse(text string) map[string]float64 {
	// Placeholder: returns zero scores for all categories.
	// Real implementation parses the "unsafe\nS10\n..." format.
	scores := make(map[string]float64)
	for _, cat := range AllCategoryIDs() {
		scores[cat] = 0.0
	}
	return scores
}

// SetEnabled enables or disables the backend.
func (b *LlamaGuardBackend) SetEnabled(enabled bool) {
	b.enabled = enabled
}

// === OpenAI Moderation Backend ===

// OpenAIModerationBackend implements SemanticClassifier for the OpenAI Moderation API.
type OpenAIModerationBackend struct {
	apiKey string
	client *httpClient
}

// NewOpenAIModerationBackend creates an OpenAI Moderation backend.
// The apiKey is read from environment (OPENAI_API_KEY) if empty.
func NewOpenAIModerationBackend(apiKey string) *OpenAIModerationBackend {
	if apiKey == "" {
		apiKey = getEnv("OPENAI_API_KEY", "")
	}
	return &OpenAIModerationBackend{
		apiKey: apiKey,
		client: newHTTPClient(30 * time.Second),
	}
}

// Name returns the backend identifier.
func (b *OpenAIModerationBackend) Name() string {
	return "openai-moderation"
}

// Available checks if the backend is configured with an API key.
func (b *OpenAIModerationBackend) Available(ctx context.Context) bool {
	return b.apiKey != ""
}

// Classify sends text to OpenAI Moderation API for classification.
//
// Maps OpenAI's category names to the S1-S15 taxonomy.
func (b *OpenAIModerationBackend) Classify(ctx context.Context, text string) (map[string]float64, error) {
	// Production implementation:
	// 1. POST to https://api.openai.com/v1/moderations
	// 2. Map response categories to S1-S15
	// 3. Return normalized scores
	//
	// Stub implementation returns zero scores.
	return b.mapFromOpenAI(nil), nil
}

// mapFromOpenAI converts OpenAI moderation categories to S1-S15 scores.
func (b *OpenAIModerationBackend) mapFromOpenAI(openAIResult interface{}) map[string]float64 {
	// Mapping:
	// OpenAI "hate" → S10
	// OpenAI "harassment" → S10
	// OpenAI "self-harm" → S11
	// OpenAI "sexual" → S12
	// OpenAI "violence" → S1
	// ... etc
	scores := make(map[string]float64)
	for _, cat := range AllCategoryIDs() {
		scores[cat] = 0.0
	}
	return scores
}

// === Taxonomy Helpers ===

// CategoryMetadata returns the name and default action for a category.
func CategoryMetadata(id string) (name string, defaultAction Action) {
	meta := map[string]struct {
		name   string
		action Action
	}{
		"S1":  {"Violent Crimes", ActionBlock},
		"S2":  {"Non-Violent Crimes", ActionBlock},
		"S3":  {"Sex Crimes", ActionBlock},
		"S4":  {"Child Exploitation", ActionBlock},
		"S5":  {"Defamation", ActionWarn},
		"S6":  {"Specialized Advice", ActionWarn},
		"S7":  {"Privacy", ActionBlock},
		"S8":  {"Intellectual Property", ActionWarn},
		"S9":  {"Indiscriminate Weapons", ActionBlock},
		"S10": {"Hate", ActionBlock},
		"S11": {"Self-Harm", ActionBlock},
		"S12": {"Sexual Content", ActionBlock},
		"S13": {"Elections", ActionWarn},
		"S14": {"Code Abuse", ActionBlock},
		"S15": {"Data Exfiltration", ActionBlock},
	}

	m, ok := meta[id]
	if !ok {
		return "Unknown", ActionWarn
	}
	return m.name, m.action
}

// RiskFromScore converts a normalized score to a RiskLevel.
func RiskFromScore(score float64) RiskLevel {
	switch {
	case score >= 0.9:
		return RiskLevelCritical
	case score >= 0.7:
		return RiskLevelHigh
	case score >= 0.4:
		return RiskLevelMedium
	case score > 0:
		return RiskLevelLow
	default:
		return RiskLevelNone
	}
}

// === HTTP Client ===

// httpClient is a simple wrapper around http.Client for backend calls.
type httpClient struct {
	timeout time.Duration
}

func newHTTPClient(timeout time.Duration) *httpClient {
	return &httpClient{timeout: timeout}
}

func (c *httpClient) getHealth(ctx context.Context, url string) bool {
	// Production: make actual HTTP request
	// Stub: return false if URL is empty
	if url == "" || url == "http://localhost:11434/api/tags" {
		// Default URL without service running = unavailable
		return false
	}
	return true
}

func getEnv(key, defaultVal string) string {
	if v, ok := lookupEnv(key); ok {
		return v
	}
	return defaultVal
}

func lookupEnv(key string) (string, bool) {
	// Stub — replace with os.LookupEnv in production
	return "", false
}

// === Policy Engine ===

// PolicyEngine evaluates category scores against configured policies.
type PolicyEngine struct {
	rules []PolicyRule
	mu    sync.RWMutex
}

// NewPolicyEngine creates a policy engine with the given rules.
func NewPolicyEngine(rules []PolicyRule) *PolicyEngine {
	return &PolicyEngine{rules: rules}
}

// Evaluate maps category scores to actions based on policy rules.
func (pe *PolicyEngine) Evaluate(scores map[string]float64, backendName string) *ClassificationResult {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	var categories []CategoryResult
	var maxRisk float64

	for catID, score := range scores {
		if score <= 0 {
			continue
		}

		name, defaultAction := CategoryMetadata(catID)
		action := defaultAction
		reason := ""

		// Apply policy overrides
		for _, rule := range pe.rules {
			for _, detail := range rule.Rules {
				if detail.Category == catID {
					action = detail.Action
					reason = detail.Description
					if detail.Threshold > 0 && score < detail.Threshold {
						action = ActionAllow
					}
				}
			}
		}

		categories = append(categories, CategoryResult{
			ID:     catID,
			Name:   name,
			Score:  score,
			Action: action,
			Reason: reason,
		})

		if score > maxRisk {
			maxRisk = score
		}
	}

	return &ClassificationResult{
		Safe:        !hasBlock(categories),
		OverallRisk: maxRisk,
		Categories:  categories,
		Backend:     backendName,
	}
}

// CheckPolicy checks if the classification result complies with a specific policy.
func (pe *PolicyEngine) CheckPolicy(result *ClassificationResult, policyID string) *PolicyResult {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	var targetRule *PolicyRule
	for i := range pe.rules {
		if pe.rules[i].ID == policyID {
			targetRule = &pe.rules[i]
			break
		}
	}

	if targetRule == nil {
		return &PolicyResult{
			PolicyID:  policyID,
			Compliant: true,
		}
	}

	var violations []ContentViolation
	for _, cat := range result.Categories {
		for _, detail := range targetRule.Rules {
			if detail.Category == cat.ID && cat.Action == ActionBlock {
				violations = append(violations, ContentViolation{
					CategoryID:   cat.ID,
					CategoryName: cat.Name,
					Score:        cat.Score,
					Action:       cat.Action,
					Reason:       detail.Description,
				})
			}
		}
	}

	return &PolicyResult{
		PolicyID:   policyID,
		Compliant:  len(violations) == 0,
		Violations: violations,
	}
}

// UpdateRules replaces the policy rules (for hot-reload).
func (pe *PolicyEngine) UpdateRules(rules []PolicyRule) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.rules = rules
}

func hasBlock(categories []CategoryResult) bool {
	for _, c := range categories {
		if c.Action == ActionBlock {
			return true
		}
	}
	return false
}
