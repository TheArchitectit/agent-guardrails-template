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
	"sort"
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
	Safe       bool     `json:"safe"`
	Confidence float64  `json:"confidence"`
	Layer      string   `json:"layer"`
	Reason     string   `json:"reason,omitempty"`
	Categories []string `json:"categories,omitempty"`
	// LayersEvaluated names the layers that actually examined the text. It
	// exists because "safe" alone cannot tell a caller whether the semantic
	// layer ran or was simply absent: a deployment with L1 only and one with
	// L1+L3 both returned Safe=true, and nothing in the result distinguished
	// them (Spec 01).
	LayersEvaluated []string  `json:"layers_evaluated,omitempty"`
	TextHash        string    `json:"text_hash"`
	Decision        string    `json:"decision"`
	ProcessedAt     time.Time `json:"processed_at"`
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

// auditAttribution carries the calling tool and request identity into an audit
// event. Spec 01 declares SourceTool and ToolCallID on the event, but nothing
// populated them, so an audit record could not be tied to the tool call that
// produced it. Attribution travels on the context rather than the pipeline so
// the pipeline stays safe for concurrent calls from different tools.
type auditAttribution struct {
	sourceTool string
	toolCallID string
}

type auditAttributionKey struct{}

// WithAuditAttribution returns a context carrying the tool identity that the
// injection audit logger records. Empty values are ignored, so a caller that
// has no tool context (library use) is unaffected.
func WithAuditAttribution(ctx context.Context, sourceTool, toolCallID string) context.Context {
	if sourceTool == "" && toolCallID == "" {
		return ctx
	}
	return context.WithValue(ctx, auditAttributionKey{}, auditAttribution{
		sourceTool: sourceTool,
		toolCallID: toolCallID,
	})
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
//
// Enabled is the L3 gate and it is explicit. The layer used to run
// unconditionally against a NoOpClassifier, so every result looked like a
// three-layer scan while the semantic layer answered "safe" for text it never
// evaluated. An explicit switch makes "L3 did not run" representable, which is
// what the result now reports.
type ClassifierConfig struct {
	Enabled   bool          `yaml:"enabled" json:"enabled"`
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
//
// L1 pattern matching is enabled AND carries the built-in pattern set. It is
// not enough for L1 to be enabled: an enabled matcher with no patterns
// compiles to an empty list, Match never fires, and Detect returns Safe=true
// for text that was never evaluated. The default therefore seeds the patterns
// declared by DefaultPatterns (Spec 01 Sec. 3.3 categories) so the shipped
// configuration actually evaluates input.
func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		Enabled:    true,
		FailPolicy: FailPolicyBlock,
		Layers: LayersConfig{
			PatternMatching: PatternMatchingConfig{
				Enabled:        true,
				CustomPatterns: DefaultPatternList(),
			},
			Perplexity: PerplexityConfig{
				// Spec 01 Sec. 2.2: "L1-L2 run on every input." The shipped
				// default must match the spec rather than undercut it. Measured
				// before flipping: ordinary prose, source code and SQL all
				// score 0.000 (no anomaly); only degenerate repetition trips
				// the entropy floor.
				Enabled:   true,
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

// DefaultPatternList returns the built-in injection patterns as a stable,
// deterministically ordered slice. The order is sorted by category name so a
// configuration diff is reproducible rather than map-iteration dependent.
func DefaultPatternList() []string {
	patterns := DefaultPatterns()
	categories := make([]string, 0, len(patterns))
	for category := range patterns {
		categories = append(categories, category)
	}
	sort.Strings(categories)

	out := make([]string, 0, len(patterns))
	for _, category := range categories {
		out = append(out, patterns[category])
	}
	return out
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

	// L4 has no implementation: LayerLLMSelfCheck is declared and the config
	// struct carries llm_self_check, but no code path ever assigns the layer,
	// so enabling it changed nothing and the pipeline still reported a
	// complete scan. Refuse at construction rather than accept a switch that
	// does nothing (Spec 01 L4; P5/D-18 fail-closed).
	if config.Layers.LLMSelfCheck.Enabled {
		return nil, fmt.Errorf("L4 LLM self-check is enabled in configuration but has no implementation; refusing to construct a pipeline that would silently skip it (Spec 01)")
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
		if err == nil {
			result.LayersEvaluated = append(result.LayersEvaluated, string(LayerPatternMatching))
		}
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
		result.LayersEvaluated = append(result.LayersEvaluated, string(LayerPerplexity))
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

	// L3: Classifier Model. The classifier is nil when the layer is not
	// enabled, so "no semantic coverage" is visible in LayersEvaluated rather
	// than masked by a backend that always answers safe. Availability is NOT
	// probed here: that would be a network round trip on every scan. A backend
	// that is unreachable errors on Classify and fails closed below.
	if p.classifier != nil {
		safe, confidence, categories, err := p.classifier.Classify(ctx, text)
		if err == nil {
			result.LayersEvaluated = append(result.LayersEvaluated, string(LayerClassifier))
		}
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
	// Read the sink under the lock: SetInjectionAuditLogger replaces it at
	// runtime, and an unsynchronised read races that write.
	p.mu.RLock()
	logger := p.auditLogger
	p.mu.RUnlock()
	if logger == nil {
		return
	}
	event := AuditEvent{
		Event:      "injection_detected",
		Timestamp:  result.ProcessedAt.Format(time.RFC3339),
		Source:     string(source),
		Safe:       result.Safe,
		Confidence: result.Confidence,
		Layer:      result.Layer,
		Categories: result.Categories,
		TextHash:   result.TextHash,
		Decision:   result.Decision,
	}
	if attr, ok := ctx.Value(auditAttributionKey{}).(auditAttribution); ok {
		event.SourceTool = attr.sourceTool
		event.ToolCallID = attr.toolCallID
	}
	logger.LogInjection(ctx, event)
}
