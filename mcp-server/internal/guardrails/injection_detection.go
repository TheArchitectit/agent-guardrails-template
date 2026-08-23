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
	"fmt"
	"strings"
	"sync"
	"time"
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
	Name() string
	Classify(ctx context.Context, text string) (safe bool, confidence float64, categories []InjectionCategory, err error)
	Available(ctx context.Context) bool
}

// ClassifierBackend is a factory for creating classifier instances.
type ClassifierBackend interface {
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
	Enabled        bool                  `yaml:"enabled" json:"enabled"`
	Layers         LayersConfig          `yaml:"layers" json:"layers"`
	FailPolicy     FailPolicy            `yaml:"fail_policy" json:"fail_policy"`
	SourcePolicies map[Source]FailPolicy `yaml:"source_policies" json:"source_policies"`
}

// DefaultPipelineConfig returns a safe default for the injection pipeline.
func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		Enabled:    true,
		FailPolicy: FailPolicyBlock,
		Layers: LayersConfig{
			PatternMatching: PatternMatchingConfig{
				Enabled: true,
			},
			Perplexity: PerplexityConfig{
				Enabled:   false,
				Threshold: 0.85,
			},
			Classifier: ClassifierConfig{},
			LLMSelfCheck: LLMSelfCheckConfig{
				Enabled: false,
			},
		},
		SourcePolicies: map[Source]FailPolicy{
			SourceUser:        FailPolicyLogOnly,
			SourceToolOutput:  FailPolicyBlock,
			SourceFileContent: FailPolicyBlock,
			SourceAPIResponse: FailPolicyBlock,
		},
	}
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

	if config.Layers.PatternMatching.Enabled {
		patterns, err := NewPatternMatcher(config.Layers.PatternMatching)
		if err != nil {
			return nil, fmt.Errorf("failed to create pattern matcher: %w", err)
		}
		p.patterns = patterns
	}

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
			result.Safe = false
			result.Confidence = 1.0
			result.Layer = string(LayerClassifier)
			result.Reason = fmt.Sprintf("Classifier error (fail-closed): %v", err)
			result.Decision = string(FailPolicyBlock)
			p.logAudit(ctx, result, source)
			return result
		}
		if !safe {
			threshold := p.config.Layers.Classifier.Threshold
			if threshold <= 0 {
				threshold = 0.7
			}
			if confidence < threshold {
				result.Layer = string(LayerClassifier)
				result.Categories = categoriesToStrings(categories)
				result.Reason = fmt.Sprintf("Classifier flagged injection (confidence %.2f < threshold %.2f): %s",
					confidence, threshold, strings.Join(result.Categories, ", "))
				result.Decision = string(FailPolicyLogOnly)
				p.logAudit(ctx, result, source)
				return result
			}
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

const maxConcurrentDetections = 8

// DetectBatch runs injection detection on multiple texts.
func (p *Pipeline) DetectBatch(ctx context.Context, items []BatchItem) BatchResult {
	if len(items) == 0 {
		return BatchResult{}
	}

	results := make([]BatchItemResult, len(items))
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentDetections)

	for i, item := range items {
		wg.Add(1)
		go func(idx int, item BatchItem) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result := p.Detect(ctx, item.Text, item.Source)
			results[idx] = BatchItemResult{
				ID:         item.ID,
				Safe:       result.Safe,
				Confidence: result.Confidence,
				Layer:      result.Layer,
				Reason:     result.Reason,
			}
		}(i, item)
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
