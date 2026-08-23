// Package guardrails provides prompt injection detection for the MCP server.
//
// The injection defense system implements a layered detection strategy:
//   - L1: Pattern matching (regex + keyword blocklists) — <1ms
//   - L2: Perplexity analysis (statistical anomaly detection) — <5ms
//   - L3: Classifier model (Llama Guard / NeMo / custom) — <50ms
//   - L4: LLM self-check (optional) — <200ms
//
// L1-L2 run on every input. L3 runs on inputs that pass L1-L2. L4 is optional.
package guardrails

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dlclark/regexp2"
)

// Source represents the origin of text being analyzed.
type Source string

const (
	SourceUser        Source = "user"
	SourceToolOutput  Source = "tool_output"
	SourceFileContent Source = "file_content"
	SourceAPIResponse Source = "api_response"
)

// InjectionCategory represents the type of injection detected.
type InjectionCategory string

const (
	CategoryDirectiveOverride   InjectionCategory = "directive_override"
	CategoryRolePlay            InjectionCategory = "role_play"
	CategoryEncodingBypass      InjectionCategory = "encoding_bypass"
	CategoryContextManipulation InjectionCategory = "context_manipulation"
	CategoryPrivilegeEscalation InjectionCategory = "privilege_escalation"
	CategoryDataExfiltration    InjectionCategory = "data_exfiltration"
)

// DetectionLayer indicates which layer detected the injection.
type DetectionLayer string

const (
	LayerPatternMatching DetectionLayer = "L1_pattern"
	LayerPerplexity      DetectionLayer = "L2_perplexity"
	LayerClassifier      DetectionLayer = "L3_classifier"
	LayerLLMSelfCheck    DetectionLayer = "L4_llm_self_check"
)

// FailPolicy determines what action to take when injection is detected.
type FailPolicy string

const (
	FailPolicyBlock   FailPolicy = "block"
	FailPolicyWarn    FailPolicy = "warn"
	FailPolicyLogOnly FailPolicy = "log_only"
)

// InjectionResult is the output of the injection detection pipeline.
type InjectionResult struct {
	Safe        bool      `json:"safe"`
	Confidence  float64   `json:"confidence"`
	Layer       string    `json:"layer"`
	Reason      string    `json:"reason,omitempty"`
	Categories  []string  `json:"categories,omitempty"`
	TextHash    string    `json:"text_hash"`
	Decision    string    `json:"decision"`
	ProcessedAt time.Time `json:"processed_at"`
}

// BatchResult is the output of batch scanning.
type BatchResult struct {
	Results []BatchItemResult `json:"results"`
}

// BatchItemResult is a single result in a batch scan.
type BatchItemResult struct {
	ID         string  `json:"id"`
	Safe       bool    `json:"safe"`
	Confidence float64 `json:"confidence"`
	Layer      string  `json:"layer,omitempty"`
	Reason     string  `json:"reason,omitempty"`
}

// AuditEvent represents a structured audit log entry for injection detection.
type AuditEvent struct {
	Event      string   `json:"event"`
	Timestamp  string   `json:"timestamp"`
	Source     string   `json:"source"`
	SourceTool string   `json:"source_tool,omitempty"`
	Safe       bool     `json:"safe"`
	Confidence float64  `json:"confidence"`
	Layer      string   `json:"layer"`
	Categories []string `json:"categories,omitempty"`
	TextHash   string   `json:"text_hash"`
	Decision   string   `json:"decision"`
	ToolCallID string   `json:"tool_call_id,omitempty"`
}

// InjectionClassifier defines the contract for L3 classifier backends.
type InjectionClassifier interface {
	// Classify analyzes text and returns a safety assessment.
	// Returns safe=true if no injection detected, safe=false with confidence and categories if detected.
	Classify(ctx context.Context, text string) (safe bool, confidence float64, categories []InjectionCategory, err error)

	// HealthCheck returns the current health of the classifier backend.
	HealthCheck(ctx context.Context) error
}

// ClassifierBackend is a factory for creating classifier instances.
type ClassifierBackend interface {
	// Create returns a configured InjectionClassifier.
	Create(config ClassifierConfig) (InjectionClassifier, error)
}

// ClassifierConfig holds configuration for classifier backends.
type ClassifierConfig struct {
	Backend   string        `yaml:"backend" json:"backend"`
	ModelPath string        `yaml:"model_path" json:"model_path"`
	Threshold float64       `yaml:"threshold" json:"threshold"`
	Timeout   time.Duration `yaml:"timeout" json:"timeout"`
	Endpoint  string        `yaml:"endpoint" json:"endpoint"`
}

// Pipeline orchestrates the layered injection detection.
type Pipeline struct {
	config      PipelineConfig
	patterns    *PatternMatcher
	perplexity  *PerplexityAnalyzer
	classifier  InjectionClassifier
	auditLogger AuditLogger
	mu          sync.RWMutex
}

// PipelineConfig configures the detection pipeline.
type PipelineConfig struct {
	Enabled        bool                    `yaml:"enabled" json:"enabled"`
	Layers         LayersConfig            `yaml:"layers" json:"layers"`
	FailPolicy     FailPolicy              `yaml:"fail_policy" json:"fail_policy"`
	SourcePolicies map[Source]FailPolicy   `yaml:"source_policies" json:"source_policies"`
}

// LayersConfig configures individual detection layers.
type LayersConfig struct {
	PatternMatching PatternMatchingConfig `yaml:"pattern_matching" json:"pattern_matching"`
	Perplexity      PerplexityConfig      `yaml:"perplexity" json:"perplexity"`
	Classifier      ClassifierConfig      `yaml:"classifier" json:"classifier"`
	LLMSelfCheck    LLMSelfCheckConfig    `yaml:"llm_self_check" json:"llm_self_check"`
}

// PatternMatchingConfig configures L1 pattern matching.
type PatternMatchingConfig struct {
	Enabled        bool     `yaml:"enabled" json:"enabled"`
	BlocklistPaths []string `yaml:"blocklists" json:"blocklists"`
	CustomPatterns []string `yaml:"custom_patterns" json:"custom_patterns"`
}

// PerplexityConfig configures L2 perplexity analysis.
type PerplexityConfig struct {
	Enabled   bool    `yaml:"enabled" json:"enabled"`
	Threshold float64 `yaml:"threshold" json:"threshold"`
}

// LLMSelfCheckConfig configures L4 LLM self-check.
type LLMSelfCheckConfig struct {
	Enabled   bool    `yaml:"enabled" json:"enabled"`
	Model     string  `yaml:"model" json:"model"`
	Threshold float64 `yaml:"threshold" json:"threshold"`
}

// AuditLogger defines the interface for audit logging.
type AuditLogger interface {
	LogInjection(ctx context.Context, event AuditEvent)
}

// NewPipeline creates a new injection detection pipeline.
func NewPipeline(config PipelineConfig, classifier InjectionClassifier, auditLogger AuditLogger) (*Pipeline, error) {
	p := &Pipeline{
		config:      config,
		classifier:  classifier,
		auditLogger: auditLogger,
	}

	// Initialize pattern matcher
	if config.Layers.PatternMatching.Enabled {
		patterns, err := NewPatternMatcher(config.Layers.PatternMatching)
		if err != nil {
			return nil, fmt.Errorf("failed to create pattern matcher: %w", err)
		}
		p.patterns = patterns
	}

	// Initialize perplexity analyzer
	if config.Layers.Perplexity.Enabled {
		p.perplexity = NewPerplexityAnalyzer(config.Layers.Perplexity)
	}

	return p, nil
}

// Detect runs the full injection detection pipeline on the given text.
func (p *Pipeline) Detect(ctx context.Context, text string, source Source) InjectionResult {
	result := InjectionResult{
		Safe:        true,
		Confidence:  0.0,
		TextHash:    hashText(text),
		ProcessedAt: time.Now().UTC(),
	}

	if !p.config.Enabled {
		result.Decision = string(FailPolicyLogOnly)
		return result
	}

	// Determine effective policy for this source
	policy := p.config.FailPolicy
	if sp, ok := p.config.SourcePolicies[source]; ok {
		policy = sp
	}

	// L1: Pattern Matching
	if p.patterns != nil {
		match, categories, err := p.patterns.Match(ctx, text)
		if err == nil && match {
			result.Safe = false
			result.Confidence = 0.95
			result.Layer = string(LayerPatternMatching)
			result.Categories = categoriesToStrings(categories)
			result.Reason = fmt.Sprintf("Pattern match: %s", strings.Join(result.Categories, ", "))
			result.Decision = string(policy)
			p.logAudit(ctx, result, source)
			return result
		}
	}

	// L2: Perplexity Analysis
	if p.perplexity != nil {
		score, anomaly := p.perplexity.Analyze(ctx, text)
		if anomaly {
			result.Safe = false
			result.Confidence = score
			result.Layer = string(LayerPerplexity)
			result.Reason = fmt.Sprintf("High perplexity anomaly detected (score: %.3f)", score)
			result.Decision = string(policy)
			p.logAudit(ctx, result, source)
			return result
		}
	}

	// L3: Classifier Model
	if p.classifier != nil {
		safe, confidence, categories, err := p.classifier.Classify(ctx, text)
		if err != nil {
			// Fail closed on classifier error
			result.Safe = false
			result.Confidence = 1.0
			result.Layer = string(LayerClassifier)
			result.Reason = fmt.Sprintf("Classifier error (fail-closed): %v", err)
			result.Decision = string(FailPolicyBlock)
			p.logAudit(ctx, result, source)
			return result
		}
		if !safe {
			result.Safe = false
			result.Confidence = confidence
			result.Layer = string(LayerClassifier)
			result.Categories = categoriesToStrings(categories)
			result.Reason = fmt.Sprintf("Classifier detected injection: %s", strings.Join(result.Categories, ", "))
			result.Decision = string(policy)
			p.logAudit(ctx, result, source)
			return result
		}
	}

	result.Decision = string(FailPolicyLogOnly)
	p.logAudit(ctx, result, source)
	return result
}

// DetectBatch runs injection detection on multiple texts.
func (p *Pipeline) DetectBatch(ctx context.Context, items []BatchItem) BatchResult {
	results := make([]BatchItemResult, 0, len(items))
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Process concurrently for throughput
	for _, item := range items {
		wg.Add(1)
		go func(item BatchItem) {
			defer wg.Done()
			result := p.Detect(ctx, item.Text, item.Source)
			mu.Lock()
			results = append(results, BatchItemResult{
				ID:         item.ID,
				Safe:       result.Safe,
				Confidence: result.Confidence,
				Layer:      result.Layer,
				Reason:     result.Reason,
			})
			mu.Unlock()
		}(item)
	}

	wg.Wait()
	return BatchResult{Results: results}
}

// BatchItem represents a single item in a batch scan.
type BatchItem struct {
	ID     string
	Text   string
	Source Source
}

// ReloadPatterns reloads pattern blocklists (hot-reload support).
func (p *Pipeline) ReloadPatterns() error {
	if p.patterns == nil {
		return nil
	}
	return p.patterns.Reload()
}

// logAudit emits an audit event for the detection result.
func (p *Pipeline) logAudit(ctx context.Context, result InjectionResult, source Source) {
	if p.auditLogger == nil {
		return
	}
	p.auditLogger.LogInjection(ctx, AuditEvent{
		Event:      "injection_detected",
		Timestamp:  result.ProcessedAt.Format(time.RFC3339),
		Source:     string(source),
		Safe:       result.Safe,
		Confidence: result.Confidence,
		Layer:      result.Layer,
		Categories: result.Categories,
		TextHash:   result.TextHash,
		Decision:   result.Decision,
	})
}

// PatternMatcher implements L1 pattern matching detection.
type PatternMatcher struct {
	config   PatternMatchingConfig
	patterns []*regexp2.Regexp
	mu       sync.RWMutex
}

// NewPatternMatcher creates a new pattern matcher with compiled regexes.
func NewPatternMatcher(config PatternMatchingConfig) (*PatternMatcher, error) {
	pm := &PatternMatcher{config: config}
	if err := pm.loadPatterns(); err != nil {
		return nil, err
	}
	return pm, nil
}

// loadPatterns compiles all patterns from blocklists and custom patterns.
func (pm *PatternMatcher) loadPatterns() error {
	var patterns []*regexp2.Regexp

	// Load from blocklist files
	for _, path := range pm.config.BlocklistPaths {
		lines, err := ReadBlocklistFile(path)
		if err != nil {
			slog.Warn("Failed to read blocklist", "path", path, "error", err)
			continue
		}
		for _, line := range lines {
			re, err := regexp2.Compile(line, regexp2.None)
			if err != nil {
				slog.Warn("Failed to compile pattern", "pattern", line, "error", err)
				continue
			}
			patterns = append(patterns, re)
		}
	}

	// Load custom patterns
	for _, pattern := range pm.config.CustomPatterns {
		re, err := regexp2.Compile(pattern, regexp2.None)
		if err != nil {
			slog.Warn("Failed to compile custom pattern", "pattern", pattern, "error", err)
			continue
		}
		patterns = append(patterns, re)
	}

	pm.mu.Lock()
	pm.patterns = patterns
	pm.mu.Unlock()

	return nil
}

// Match checks text against all compiled patterns.
func (pm *PatternMatcher) Match(ctx context.Context, text string) (bool, []InjectionCategory, error) {
	pm.mu.RLock()
	patterns := pm.patterns
	pm.mu.RUnlock()

	var categories []InjectionCategory
	matched := false

	for _, re := range patterns {
		isMatch, err := re.MatchString(text)
		if err != nil {
			continue
		}
		if isMatch {
			matched = true
			categories = append(categories, categorizePattern(re.String()))
		}
	}

	return matched, categories, nil
}

// Reload recompiles patterns from source files.
func (pm *PatternMatcher) Reload() error {
	return pm.loadPatterns()
}

// PerplexityAnalyzer implements L2 statistical anomaly detection.
type PerplexityAnalyzer struct {
	config PerplexityConfig
}

// NewPerplexityAnalyzer creates a new perplexity analyzer.
func NewPerplexityAnalyzer(config PerplexityConfig) *PerplexityAnalyzer {
	return &PerplexityAnalyzer{config: config}
}

// Analyze computes a perplexity-like score for the text.
// Returns the score and whether it exceeds the anomaly threshold.
func (pa *PerplexityAnalyzer) Analyze(ctx context.Context, text string) (float64, bool) {
	// Simplified perplexity proxy: measure character distribution anomaly
	// Real implementation would use a language model
	if len(text) == 0 {
		return 0, false
	}

	runes := []rune(text)
	n := float64(len(runes))

	// Calculate character entropy as a proxy for perplexity
	freq := make(map[rune]int)
	for _, r := range runes {
		freq[r]++
	}

	entropy := 0.0
	for _, count := range freq {
		p := float64(count) / n
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	// Normalize: high entropy (random-looking) or very low entropy (repetitive) = suspicious
	// Normal text typically has entropy 3.5-4.5 bits per character
	var score float64
	if entropy > 5.0 {
		score = (entropy - 5.0) / 3.0 // High entropy anomaly
	} else if entropy < 2.0 {
		score = (2.0 - entropy) / 2.0 // Low entropy anomaly
	}

	// Check for unusual character ratios (e.g., lots of special chars)
	specialCount := 0
	alphaCount := 0
	for _, r := range runes {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			alphaCount++
		} else if (r >= 33 && r <= 47) || (r >= 58 && r <= 64) {
			specialCount++
		}
	}

	if alphaCount > 0 {
		specialRatio := float64(specialCount) / float64(alphaCount)
		if specialRatio > 0.5 {
			if specialRatio > score {
				score = specialRatio
			}
		}
	}

	return score, score > pa.config.Threshold
}

// NoOpClassifier is a classifier backend that always returns safe.
// Used when no L3 backend is configured.
type NoOpClassifier struct{}

// Classify implements InjectionClassifier.
func (n *NoOpClassifier) Classify(ctx context.Context, text string) (bool, float64, []InjectionCategory, error) {
	return true, 0.0, nil, nil
}

// HealthCheck implements InjectionClassifier.
func (n *NoOpClassifier) HealthCheck(ctx context.Context) error {
	return nil
}

// hashText returns a SHA-256 hash of the text for privacy-preserving audit logs.
func hashText(text string) string {
	h := sha256.Sum256([]byte(text))
	return "sha256:" + hex.EncodeToString(h[:])
}

// categoriesToStrings converts InjectionCategory slice to string slice.
func categoriesToStrings(cats []InjectionCategory) []string {
	result := make([]string, len(cats))
	for i, c := range cats {
		result[i] = string(c)
	}
	return result
}

// categorizePattern attempts to categorize a matched pattern.
func categorizePattern(pattern string) InjectionCategory {
	p := strings.ToLower(pattern)
	switch {
	case strings.Contains(p, "ignore") || strings.Contains(p, "disregard") || strings.Contains(p, "override"):
		return CategoryDirectiveOverride
	case strings.Contains(p, "you are now") || strings.Contains(p, "act as") || strings.Contains(p, "pretend"):
		return CategoryRolePlay
	case strings.Contains(p, "base64") || strings.Contains(p, "rot13") || strings.Contains(p, "decode"):
		return CategoryEncodingBypass
	case strings.Contains(p, "system prompt") || strings.Contains(p, "system message"):
		return CategoryContextManipulation
	case strings.Contains(p, "root") || strings.Contains(p, "sudo") || strings.Contains(p, "admin"):
		return CategoryPrivilegeEscalation
	case strings.Contains(p, "print") && strings.Contains(p, "prompt"):
		return CategoryDataExfiltration
	default:
		return CategoryDirectiveOverride
	}
}

// DefaultAuditLogger implements AuditLogger using slog.
type DefaultAuditLogger struct{}

// LogInjection logs an injection detection event.
func (d *DefaultAuditLogger) LogInjection(ctx context.Context, event AuditEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		slog.Error("Failed to marshal audit event", "error", err)
		return
	}
	slog.Info("INJECTION_AUDIT", "event", string(data))
}

// Compile time interface checks
var _ InjectionClassifier = (*NoOpClassifier)(nil)
var _ AuditLogger = (*DefaultAuditLogger)(nil)

// ReadBlocklistFile reads a blocklist file and returns non-comment lines.
func ReadBlocklistFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	return lines, nil
}
