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
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
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
		// Fail open: allow content but log warning.
		// Do NOT cache this result — a cached fail-open would create a
		// TTL-length allow-all window if the backend recovers (Bug C8).
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

// NewResultCache creates a cache with the specified TTL and starts a background
// goroutine that periodically prunes expired entries (Bug H4).
func NewResultCache(ttl time.Duration) *ResultCache {
	rc := &ResultCache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
	}
	go rc.pruneLoop()
	return rc
}

// pruneLoop runs prune() on a 30s ticker for the lifetime of the cache.
func (rc *ResultCache) pruneLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		rc.prune()
	}
}

// Get retrieves a cached result if it exists and hasn't expired.
//
// It returns a deep copy so concurrent callers do not share the same mutable
// *ClassificationResult (Bug C8).
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

// copyResult returns a deep copy of a ClassificationResult so callers cannot
// mutate the cached object.
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

// Classify sends text to Llama Guard via Ollama for classification.
// Returns normalized scores (0-1) for each S1-S15 category.
func (b *LlamaGuardBackend) Classify(ctx context.Context, text string) (map[string]float64, error) {
	if text == "" {
		return b.emptyScores(), nil
	}

	prompt := contentFilterLlamaGuardPrompt(text)
	reqBody := map[string]interface{}{
		"model":  b.model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature":  0.0,
			"num_predict":  100,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal ollama request: %w", err)
	}

	url := b.ollamaURL + "/api/generate"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("create ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var ollamaResp struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("decode ollama response: %w", err)
	}

	return b.parseLlamaGuardResponse(ollamaResp.Response), nil
}

// contentFilterLlamaGuardPrompt formats text for Llama Guard content safety classification.
func contentFilterLlamaGuardPrompt(text string) string {
	return fmt.Sprintf(`[INST] Task: Classify the following text for safety violations.

Text:
"""
%s
"""

Classify into one or more of these categories with a score from 0.0 to 1.0:
S1: Violence and Hate
S2: Sexual Content
S3: Guns and Illegal Weapons
S4: Regulated Substances
S5: Suicide and Self-Harm
S6: Sexual Violence
S7: Criminal Planning
S8: Code Abuse (malicious code generation)
S9: Data Exfiltration (secrets/credential theft)
S10: Harassment and Bullying
S11: Self-Harm Promotion
S12: Sexual Content (explicit)
S13: Hate Speech
S14: Deception and Misinformation
S15: System Instruction Override

Respond in JSON format only:
{"categories": {"S1": 0.0, "S2": 0.0, ...}}
[/INST]`, text)
}

// parseLlamaGuardResponse parses the Llama Guard output into category scores.
func (b *LlamaGuardBackend) parseLlamaGuardResponse(response string) map[string]float64 {
	response = strings.TrimSpace(response)

	// Try to extract JSON from response (may be wrapped in markdown code blocks)
	jsonStr := response
	if idx := strings.Index(response, "{"); idx >= 0 {
		jsonStr = response[idx:]
	}
	if idx := strings.LastIndex(jsonStr, "}"); idx >= 0 {
		jsonStr = jsonStr[:idx+1]
	}

	var parsed struct {
		Categories map[string]float64 `json:"categories"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err == nil && len(parsed.Categories) > 0 {
		scores := b.emptyScores()
		for key, val := range parsed.Categories {
			scores[key] = val
		}
		return scores
	}

	// If JSON parsing fails, check for simple safe/unsafe
	lower := strings.ToLower(response)
	if strings.Contains(lower, "safe") && !strings.Contains(lower, "unsafe") {
		return b.emptyScores()
	}

	// Unsafe without structured data — assign default high scores
	if strings.Contains(lower, "unsafe") {
		scores := b.emptyScores()
		scores["S1"] = 0.8 // Default to violence/hate as a catch-all
		return scores
	}

	return b.emptyScores()
}

// emptyScores returns zero scores for all categories.
func (b *LlamaGuardBackend) emptyScores() map[string]float64 {
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

// emptyScores returns zero scores for all categories.
func (b *OpenAIModerationBackend) emptyScores() map[string]float64 {
	scores := make(map[string]float64)
	for _, cat := range AllCategoryIDs() {
		scores[cat] = 0.0
	}
	return scores
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
// Maps OpenAI's category names to the S1-S15 taxonomy.
func (b *OpenAIModerationBackend) Classify(ctx context.Context, text string) (map[string]float64, error) {
	if text == "" {
		return b.emptyScores(), nil
	}

	reqBody := map[string]interface{}{
		"input": text,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal openai request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/moderations", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("create openai request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+b.apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai returned status %d: %s", resp.StatusCode, string(body))
	}

	var modResp struct {
		Results []struct {
			Categories struct {
				Hate            bool    `json:"hate"`
				HateThreatening bool    `json:"hate/threatening"`
				Harassment      bool    `json:"harassment"`
				SelfHarm        bool    `json:"self-harm"`
				Sexual          bool    `json:"sexual"`
				SexualMinors    bool    `json:"sexual/minors"`
				Violence        bool    `json:"violence"`
				ViolenceGraphic bool    `json:"violence/graphic"`
			} `json:"categories"`
			CategoryScores map[string]float64 `json:"category_scores"`
			Flagged        bool               `json:"flagged"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&modResp); err != nil {
		return nil, fmt.Errorf("decode openai response: %w", err)
	}

	if len(modResp.Results) == 0 {
		return b.emptyScores(), nil
	}

	r := modResp.Results[0]
	scores := b.emptyScores()

	// Map OpenAI categories to S1-S15 taxonomy
	if val, ok := r.CategoryScores["violence"]; ok {
		scores["S1"] = val
	} else if r.Categories.Violence {
		scores["S1"] = 0.8
	}

	if val, ok := r.CategoryScores["sexual"]; ok {
		scores["S2"] = val
	} else if r.Categories.Sexual {
		scores["S2"] = 0.8
	}

	if r.Categories.Violence && r.Categories.ViolenceGraphic {
		scores["S3"] = 0.6
	}

	if val, ok := r.CategoryScores["self-harm"]; ok {
		scores["S5"] = val
	} else if r.Categories.SelfHarm {
		scores["S5"] = 0.8
	}

	if r.Categories.Harassment && r.Categories.Sexual {
		scores["S6"] = 0.7
	}

	if val, ok := r.CategoryScores["harassment"]; ok {
		scores["S10"] = val
	} else if r.Categories.Harassment {
		scores["S10"] = 0.8
	}

	if val, ok := r.CategoryScores["self-harm/intent"]; ok {
		scores["S11"] = val
	}

	if r.Categories.Sexual {
		if val, ok := r.CategoryScores["sexual"]; ok {
			scores["S12"] = val
		} else {
			scores["S12"] = 0.8
		}
	}

	if val, ok := r.CategoryScores["hate"]; ok {
		scores["S13"] = val
	} else if r.Categories.Hate {
		scores["S13"] = 0.8
	}

	return scores, nil
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
	if url == "" {
		return false
	}
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func getEnv(key, defaultVal string) string {
	if v, ok := lookupEnv(key); ok {
		return v
	}
	return defaultVal
}

func lookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
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
