// Package guardrails — Engine is the top-level orchestrator that composes
// all guardrail subsystems into a single evaluation entry point.
package guardrails

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// EngineConfig holds configuration for the orchestrating Engine.
type EngineConfig struct {
	Injection     PipelineConfig      `yaml:"injection" json:"injection"`
	ContentFilter ContentFilterConfig `yaml:"content_filter" json:"content_filter"`
	Sandbox       SandboxConfig       `yaml:"sandbox" json:"sandbox"`
	Provenance    *ProvenanceConfig   `yaml:"provenance" json:"provenance"`
	Enabled       bool                `yaml:"enabled" json:"enabled"`
}

// DefaultEngineConfig returns a safe default Engine configuration.
func DefaultEngineConfig() *EngineConfig {
	provCfg := DefaultProvenanceConfig()
	filterCfg := DefaultFilterConfig()
	sandboxCfg := DefaultSandboxConfig()
	return &EngineConfig{
		Enabled:       true,
		Injection:     DefaultPipelineConfig(),
		ContentFilter: *filterCfg,
		Sandbox:       *sandboxCfg,
		Provenance:    provCfg,
	}
}

// defaultAuditLogger adapts slog.Logger to the AuditLogger interface.
type defaultAuditLogger struct {
	logger *slog.Logger
}

func (l *defaultAuditLogger) LogInjection(ctx context.Context, event AuditEvent) {
	l.logger.Info("injection_detected",
		"safe", event.Safe,
		"confidence", event.Confidence,
		"layer", event.Layer,
		"categories", event.Categories,
		"decision", event.Decision,
	)
}

// Engine composes all guardrail subsystems into a single evaluation pipeline.
type Engine struct {
	config   *EngineConfig
	pipeline *Pipeline
	filter   *ContentFilter
	tracker  *ProvenanceTracker
	sandbox  *SandboxManager
	logger   *slog.Logger

	// provRecords records every external-content scan so a caller can look a
	// verdict back up by provenance id (Spec 05 Sec. 3.1
	// guardrail_check_provenance). Lazily created: an Engine built as a struct
	// literal must still record rather than panic on a nil registry.
	provMu      sync.Mutex
	provRecords *provenanceRegistry
}

// NewEngine creates an Engine from the given configuration.
func NewEngine(config *EngineConfig, logger *slog.Logger) *Engine {
	if config == nil {
		config = DefaultEngineConfig()
	}
	if logger == nil {
		logger = slog.Default()
	}

	e := &Engine{
		config: config,
		logger: logger,
	}

	// Initialize injection detection pipeline. L3 is nil unless a deployment
	// supplies a real backend: the pipeline used to be constructed with a
	// NoOpClassifier, so L3 ran on every input and answered "safe" for text it
	// never evaluated. An absent semantic backend must be an absent layer, not
	// a passing one (Spec 01; P5/D-18).
	pipeline, err := NewPipeline(config.Injection, nil, &defaultAuditLogger{logger: logger})
	if err != nil {
		logger.Error("failed to init injection pipeline", "error", err)
	} else {
		e.pipeline = pipeline
	}

	// Initialize content filter. Configured policies must be passed here:
	// without them the policy engine has no rules, and every
	// guardrail_check_policy call falls through to the fail-closed
	// "unknown policy_id" branch regardless of the policy requested.
	//
	// The whole config is handed over, not just the policies. Before this,
	// NewContentFilter built its own DefaultFilterConfig(), so the
	// deployment's `fail_policy`, `thresholds` and `taxonomy` were inert even
	// when supplied programmatically: a file that set `fail_policy: allow` or
	// narrowed the taxonomy changed nothing the filter did.
	filterCfg := config.ContentFilter
	e.filter = NewContentFilter(nil, config.ContentFilter.Policies, WithConfig(&filterCfg))

	// Initialize provenance tracker
	if config.Provenance != nil && config.Provenance.Enabled {
		cache := NewInMemoryProvenanceCache(config.Provenance.CacheDuration())
		e.tracker = NewProvenanceTracker(config.Provenance.SourceTrustPolicies, cache)
	}

	// Initialize sandbox manager
	e.sandbox = NewSandboxManager(&config.Sandbox, logger)

	// Initialize the provenance verdict registry. TTL follows the provenance
	// cache window so a lookup and a cache entry expire together.
	provTTL := time.Hour
	if config.Provenance != nil {
		provTTL = config.Provenance.CacheDuration()
	}
	e.provRecords = newProvenanceRegistry(provTTL)

	logger.Info("guardrails engine initialized",
		"injection_enabled", config.Injection.Enabled,
		"content_filter_enabled", config.ContentFilter.Enabled,
		"provenance_enabled", config.Provenance != nil && config.Provenance.Enabled,
	)

	return e
}

// SetClassifier replaces the injection detection classifier backend.
func (e *Engine) SetClassifier(classifier InjectionClassifier) {
	if e.pipeline != nil {
		e.pipeline.mu.Lock()
		e.pipeline.classifier = classifier
		e.pipeline.mu.Unlock()
	}
}

// EnableInjectionClassifier attaches the L3 semantic backend and marks the
// layer enabled in the pipeline configuration.
//
// It exists so a deployment can turn L3 on without reaching into pipeline
// internals, and so the switch and the backend cannot disagree: the previous
// shape had a config flag with no effect and a backend with no config.
func (e *Engine) EnableInjectionClassifier(classifier InjectionClassifier, cfg ClassifierConfig) error {
	if classifier == nil {
		return fmt.Errorf("cannot enable the L3 classifier without a backend")
	}
	if e.pipeline == nil {
		return fmt.Errorf("cannot enable the L3 classifier: injection pipeline is not initialized")
	}
	if cfg.Threshold <= 0 || cfg.Threshold > 1 {
		return fmt.Errorf("classifier threshold must be in (0,1], got %.2f", cfg.Threshold)
	}

	e.pipeline.mu.Lock()
	e.pipeline.classifier = classifier
	e.pipeline.config.Layers.Classifier = cfg
	e.pipeline.mu.Unlock()

	if e.config != nil {
		e.config.Injection.Layers.Classifier = cfg
	}
	return nil
}

// InjectionPipelineConfigured reports whether the engine holds a constructed
// injection pipeline. NewEngine logs and continues when construction fails (a
// configuration naming an unimplemented layer, for example), so a caller that
// must not serve without detection needs a way to ask. Nil means nothing
// scans any input.
func (e *Engine) InjectionPipelineConfigured() bool {
	return e != nil && e.pipeline != nil
}

// InjectionClassifierConfigured reports whether a semantic L3 backend is
// attached. Callers use it to state coverage rather than assume it.
func (e *Engine) InjectionClassifierConfigured() bool {
	if e.pipeline == nil {
		return false
	}
	e.pipeline.mu.RLock()
	defer e.pipeline.mu.RUnlock()
	return e.pipeline.classifier != nil
}

// SetInjectionAuditLogger replaces the sink that records injection decisions.
// It exists so a deployment (or a test) can route the audit trail somewhere
// other than the process logger without rebuilding the pipeline.
func (e *Engine) SetInjectionAuditLogger(logger AuditLogger) {
	if e.pipeline != nil {
		e.pipeline.mu.Lock()
		e.pipeline.auditLogger = logger
		e.pipeline.mu.Unlock()
	}
}

// ContentFilterBackendCount reports how many semantic backends the content
// filter has. Zero means the Spec 02 layer cannot evaluate anything: a direct
// classification fails closed, and a composed safety chain reports the layer
// as unavailable instead of attributing a block to content that was never read.
func (e *Engine) ContentFilterBackendCount() int {
	if e.filter == nil {
		return 0
	}
	return e.filter.BackendCount()
}

// AddContentFilterBackend adds a semantic classification backend.
func (e *Engine) AddContentFilterBackend(backend SemanticClassifier) {
	if e.filter != nil {
		e.filter.AddContentFilterBackend(backend)
	}
}

// ReloadInjectionPatterns recompiles the L1 blocklists from their source files
// (Spec 01 Sec. 4.2: "Blocklists are hot-reloaded (SIGHUP or file watcher)").
//
// This is the caller ReloadPatterns never had. A failed reload leaves the prior
// compiled set in force — a reload must not be able to disarm a running control
// — so the error is for the operator, not a reason to stop serving.
func (e *Engine) ReloadInjectionPatterns() error {
	if e == nil || e.pipeline == nil {
		return fmt.Errorf("injection pipeline is not configured; nothing to reload")
	}
	return e.pipeline.ReloadPatterns()
}

// Stop releases background resources owned by the engine (cache prune loop, etc.).
func (e *Engine) Stop() {
	if e.filter != nil {
		e.filter.Stop()
	}
}

// EvalInput is the input to Engine.Evaluate.
type EvalInput struct {
	Text       string           `json:"text"`
	Source     Source           `json:"source"`
	SourceTool string           `json:"source_tool,omitempty"`
	Direction  ContentDirection `json:"direction"`
	Command    string           `json:"command,omitempty"`
	ToolPolicy *ToolPolicy      `json:"tool_policy,omitempty"`
}

// ToolPolicy defines sandbox requirements for a specific tool type.
type ToolPolicy struct {
	SandboxLevel SandboxLevel   `json:"sandbox_level"`
	Limits       ResourceLimits `json:"limits"`
}

// EvalResult is the output of Engine.Evaluate.
type EvalResult struct {
	Safe       bool                  `json:"safe"`
	Decision   string                `json:"decision"`
	Reason     string                `json:"reason,omitempty"`
	Injection  *InjectionResult      `json:"injection,omitempty"`
	Content    *ClassificationResult `json:"content,omitempty"`
	Provenance *Provenance           `json:"provenance,omitempty"`
	Sandbox    *SandboxResult        `json:"sandbox,omitempty"`
	LatencyMs  int64                 `json:"latency_ms"`
	Warnings   []string              `json:"warnings,omitempty"`
}

// Evaluate runs the full guardrail pipeline on the given input.
func (e *Engine) Evaluate(ctx context.Context, input EvalInput) *EvalResult {
	start := time.Now()
	result := &EvalResult{
		Safe:     true,
		Decision: "allow",
	}

	if !e.config.Enabled {
		return result
	}

	direction := input.Direction
	if direction == "" {
		direction = DirectionInput
	}

	// 1. Provenance tracking
	trustedSource := false
	if e.tracker != nil && input.Text != "" {
		prov, err := e.tracker.TagContent(ctx, input.Text,
			string(input.Source), input.SourceTool, "engine")
		if err != nil {
			e.logger.Warn("provenance tagging failed", "error", err)
			result.Warnings = append(result.Warnings, fmt.Sprintf("provenance: %v", err))
		} else {
			result.Provenance = prov
			if prov.TrustLevel == TrustLevelTrusted {
				trustedSource = true
			}
		}
	}

	// 2. Injection detection.
	// Trusted sources skip the injection deep-scan only — they are still
	// subject to content-filter classification and sandbox checks below.
	// A trusted tag must never unconditionally allow content.
	if e.pipeline != nil && e.config.Injection.Enabled && input.Text != "" && !trustedSource {
		injResult := e.pipeline.Detect(ctx, input.Text, input.Source)
		result.Injection = &injResult
		if !injResult.Safe {
			result.Safe = false
			result.Decision = injResult.Decision
			result.Reason = fmt.Sprintf("injection detected at %s: %s",
				injResult.Layer, injResult.Reason)
			result.LatencyMs = time.Since(start).Milliseconds()
			return result
		}
	}

	// 3. Content filtering
	if e.filter != nil && e.config.ContentFilter.Enabled && input.Text != "" {
		contentResult, err := e.filter.Classify(ctx, input.Text, direction)
		if err != nil {
			e.logger.Warn("content classification failed", "error", err)
			result.Warnings = append(result.Warnings, fmt.Sprintf("content_filter: %v", err))
		} else {
			result.Content = contentResult
			if contentResult != nil && contentResult.IsBlocked() {
				result.Safe = false
				result.Decision = "block"
				// Build reason from blocked categories
				var blocked []string
				for _, cat := range contentResult.Categories {
					if cat.Action == ActionBlock {
						blocked = append(blocked, cat.Name)
					}
				}
				result.Reason = fmt.Sprintf("content policy violation: %v", blocked)
				result.LatencyMs = time.Since(start).Milliseconds()
				return result
			}
		}
	}

	// 4. Sandbox execution
	if e.sandbox != nil && input.Command != "" {
		level := LevelL0
		// A policy that names no limits must inherit the sandbox's own
		// defaults, never an empty ResourceLimits. Passing the zero value made
		// effectiveDeadline zero, so the command was killed before it started
		// and the caller received ErrSandboxViolation — a security verdict
		// about work that never ran, which also taught operators to ignore the
		// violation signal.
		limits := e.sandbox.DefaultLimits()
		if input.ToolPolicy != nil {
			level = input.ToolPolicy.SandboxLevel
			limits = input.ToolPolicy.Limits
		}
		sandboxResult, err := e.sandbox.Execute(ctx, input.Command, level, limits)
		if err != nil {
			e.logger.Warn("sandbox execution failed", "error", err)
			result.Warnings = append(result.Warnings, fmt.Sprintf("sandbox: %v", err))
		}
		if sandboxResult != nil {
			result.Sandbox = sandboxResult
			// Treat non-zero exit code as a sandbox violation
			if sandboxResult.ExitCode != 0 {
				sandboxResult.SandboxViolations = append(sandboxResult.SandboxViolations,
					fmt.Sprintf("non-zero exit code: %d", sandboxResult.ExitCode))
			}
			if len(sandboxResult.SandboxViolations) > 0 {
				result.Safe = false
				result.Decision = "block"
				result.Reason = fmt.Sprintf("sandbox violation: %v",
					sandboxResult.SandboxViolations)
				result.LatencyMs = time.Since(start).Milliseconds()
				return result
			}
		}
	}

	result.LatencyMs = time.Since(start).Milliseconds()
	return result
}

// DetectInjection is a convenience method for injection-only detection.
//
// An engine without a pipeline reports NOT EVALUATED (safe=false) rather than
// safe. NewEngine only leaves the pipeline nil when construction failed — a
// configuration that names an unimplemented layer, for instance — and the old
// branch turned that failure into an unconditional "safe" verdict for every
// input. The batch path already failed closed; this brings the single-text
// path in line. Absent coverage must never read as a passing scan (P5/D-18).
func (e *Engine) DetectInjection(ctx context.Context, text string, source Source) InjectionResult {
	if e.pipeline == nil {
		return InjectionResult{
			Safe:        false,
			Confidence:  1.0,
			TextHash:    hashText(text),
			ProcessedAt: time.Now().UTC(),
			Reason:      "injection detection pipeline is not initialized; text was not evaluated",
			Decision:    string(FailPolicyBlock),
		}
	}
	return e.pipeline.Detect(ctx, text, source)
}

// DetectInjectionBatch is a convenience method for injection-only batch
// detection. An unconfigured pipeline reports each item as NOT evaluated
// (safe=false, layer "none") rather than safe: an absent detector must never
// present input as vetted.
func (e *Engine) DetectInjectionBatch(ctx context.Context, items []BatchItem) BatchResult {
	if e.pipeline == nil {
		out := BatchResult{Results: make([]BatchItemResult, len(items))}
		for i, item := range items {
			out.Results[i] = BatchItemResult{
				ID:     item.ID,
				Safe:   false,
				Reason: "injection detection not configured",
			}
		}
		return out
	}
	return e.pipeline.DetectBatch(ctx, items)
}

// ClassifyContent is a convenience method for content-only classification.
func (e *Engine) ClassifyContent(ctx context.Context, text string, direction ContentDirection) (*ClassificationResult, error) {
	if e.filter == nil {
		return &ClassificationResult{Safe: true, Backend: "none"}, nil
	}
	return e.filter.Classify(ctx, text, direction)
}

// UpdateRules replaces the content filter's policy rules (hot-reload). It is a
// convenience wrapper around ContentFilter.UpdateRules and invalidates the
// filter's result cache.
func (e *Engine) UpdateRules(rules []PolicyRule) {
	if e.filter != nil {
		e.filter.UpdateRules(rules)
	}
}

// CheckPolicy is a convenience method for policy-only checks.
func (e *Engine) CheckPolicy(ctx context.Context, text string, policyID string) (*PolicyResult, error) {
	if e.filter == nil {
		return &PolicyResult{PolicyID: policyID, Compliant: true}, nil
	}
	return e.filter.CheckPolicy(ctx, text, policyID)
}

// ExecuteSandbox is a convenience method for sandbox-only execution.
func (e *Engine) ExecuteSandbox(ctx context.Context, command string, level SandboxLevel, limits ResourceLimits) (*SandboxResult, error) {
	if e.sandbox == nil {
		return &SandboxResult{ExitCode: 0}, nil
	}
	return e.sandbox.Execute(ctx, command, level, limits)
}

// ExecuteSandboxFor runs a command under the isolation level and resource
// limits declared for toolType (Spec 03 Sec. 3.3).
//
// The declared per-tool policies are the whole point of the table: before this
// method, the only exported execution path required the caller to supply the
// level and every limit by hand, so an operator's policy changed nothing.
// A requestedLevel may raise isolation but never lower it.
func (e *Engine) ExecuteSandboxFor(ctx context.Context, toolType, command string, requestedLevel SandboxLevel) (*SandboxResult, error) {
	if e.sandbox == nil {
		return nil, fmt.Errorf("sandbox is not configured")
	}
	return e.sandbox.ExecuteFor(ctx, toolType, command, requestedLevel)
}

// SetSandboxPolicy installs or replaces one tool type's sandbox policy.
// The config is validated as a whole before it is applied.
func (e *Engine) SetSandboxPolicy(toolType string, policy SandboxPolicy) error {
	if e.sandbox == nil {
		return fmt.Errorf("sandbox is not configured")
	}
	return e.sandbox.SetToolPolicy(toolType, policy)
}

// SandboxPolicies returns the active per-tool policy table.
func (e *Engine) SandboxPolicies() map[string]SandboxPolicy {
	if e.sandbox == nil {
		return nil
	}
	return e.sandbox.ToolPolicies()
}

// SandboxConfig returns a copy of the active sandbox configuration.
func (e *Engine) SandboxConfig() SandboxConfig {
	if e.sandbox == nil {
		return SandboxConfig{}
	}
	return e.sandbox.Config()
}

// ExecuteSandboxForLimits runs a command under the policy for toolType with
// per-request narrowing overrides applied.
func (e *Engine) ExecuteSandboxForLimits(ctx context.Context, toolType, command string, requestedLevel SandboxLevel, o LimitOverrides) (*SandboxResult, error) {
	if e.sandbox == nil {
		return nil, fmt.Errorf("sandbox is not configured")
	}
	return e.sandbox.ExecuteForLimits(ctx, toolType, command, requestedLevel, o)
}
