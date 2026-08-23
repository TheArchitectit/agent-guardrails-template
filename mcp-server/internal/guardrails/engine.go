// Package guardrails — Engine is the top-level orchestrator that composes
// all guardrail subsystems into a single evaluation entry point.
package guardrails

import (
	"context"
	"fmt"
	"log/slog"
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

	// Initialize injection detection pipeline with NoOp classifier
	pipeline, err := NewPipeline(config.Injection, &NoOpClassifier{}, &defaultAuditLogger{logger: logger})
	if err != nil {
		logger.Warn("failed to init injection pipeline", "error", err)
	} else {
		e.pipeline = pipeline
	}

	// Initialize content filter (backends added via AddBackend in production)
	e.filter = NewContentFilter(nil, nil)

	// Initialize provenance tracker
	if config.Provenance != nil && config.Provenance.Enabled {
		cache := NewInMemoryProvenanceCache(config.Provenance.CacheDuration())
		e.tracker = NewProvenanceTracker(config.Provenance.SourceTrustPolicies, cache)
	}

	// Initialize sandbox manager
	e.sandbox = NewSandboxManager(&config.Sandbox, logger)

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

// AddContentFilterBackend adds a semantic classification backend.
func (e *Engine) AddContentFilterBackend(backend SemanticClassifier) {
	if e.filter != nil {
		e.filter.mu.Lock()
		e.filter.backends = append(e.filter.backends, backend)
		e.filter.mu.Unlock()
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
	if e.tracker != nil && input.Text != "" {
		prov, err := e.tracker.TagContent(ctx, input.Text,
			string(input.Source), input.SourceTool, "engine")
		if err != nil {
			e.logger.Warn("provenance tagging failed", "error", err)
			result.Warnings = append(result.Warnings, fmt.Sprintf("provenance: %v", err))
		} else {
			result.Provenance = prov
			if prov.TrustLevel == TrustLevelTrusted {
				e.logger.Debug("trusted source, skipping deep scan", "source", input.SourceTool)
				result.LatencyMs = time.Since(start).Milliseconds()
				return result
			}
		}
	}

	// 2. Injection detection
	if e.pipeline != nil && e.config.Injection.Enabled && input.Text != "" {
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
		limits := ResourceLimits{}
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
func (e *Engine) DetectInjection(ctx context.Context, text string, source Source) InjectionResult {
	if e.pipeline == nil {
		return InjectionResult{Safe: true, Decision: string(FailPolicyLogOnly)}
	}
	return e.pipeline.Detect(ctx, text, source)
}

// ClassifyContent is a convenience method for content-only classification.
func (e *Engine) ClassifyContent(ctx context.Context, text string, direction ContentDirection) (*ClassificationResult, error) {
	if e.filter == nil {
		return &ClassificationResult{Safe: true, Backend: "none"}, nil
	}
	return e.filter.Classify(ctx, text, direction)
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
