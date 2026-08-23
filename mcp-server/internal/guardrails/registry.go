package guardrails

/*
Guardrail Registry — wires all vertical slices together

Usage in cmd/server/main.go:

	import "github.com/thearchitectit/guardrail-mcp/internal/guardrails"

	// Create registry with all slices
	registry := guardrails.NewRegistry(
		guardrails.WithBash(validation.MatchPattern),
		guardrails.WithGit(validation.MatchPattern),
		guardrails.WithFileEdit(validation.MatchPattern),
	)

	// MCPServer now depends on the registry interface
*/

import (
	"context"

	"github.com/thearchitectit/guardrail-mcp/internal/domain"
)

// Registry is the unified entry point for all guardrail types
type Registry struct {
	bash      BashEvaluator
	git       GitEvaluator
	fileEdit  FileEditEvaluator
	transport Transport
}

// NewRegistry creates a new guardrail registry
func NewRegistry(opts ...Option) *Registry {
	r := &Registry{}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Option configures the registry
type Option func(*Registry)

// WithBash adds bash guardrail slice
func WithBash(patternFn func(string, string) (bool, error)) Option {
	return func(r *Registry) {
		r.bash = newBashEvaluator(patternFn)
	}
}

// WithGit adds git guardrail slice
func WithGit(patternFn func(string, string) (bool, error)) Option {
	return func(r *Registry) {
		r.git = newGitEvaluator(patternFn)
	}
}

// WithFileEdit adds file edit guardrail slice
func WithFileEdit(patternFn func(string, string) (bool, error)) Option {
	return func(r *Registry) {
		r.fileEdit = newFileEditEvaluator(patternFn)
	}
}

// Evaluator interfaces — each slice implements one

type BashEvaluator interface {
	Evaluate(ctx context.Context, command string) ([]domain.Violation, error)
}

type GitEvaluator interface {
	Evaluate(ctx context.Context, command string) ([]domain.Violation, error)
}

type FileEditEvaluator interface {
	Evaluate(ctx context.Context, filePath, content, sessionID string) ([]domain.Violation, error)
}

// Transport interface for cross-slice operations (file read verification)
type Transport interface {
	CheckFileRead(ctx context.Context, sessionID, filePath string) (*domain.FileReadVerification, error)
}

// GuardrailService implementation via registry

func (r *Registry) EvaluateCommand(ctx context.Context, command string) ([]domain.Violation, error) {
	if r.bash != nil {
		return r.bash.Evaluate(ctx, command)
	}
	return nil, nil
}

func (r *Registry) EvaluateGit(ctx context.Context, command string) ([]domain.Violation, error) {
	if r.git != nil {
		return r.git.Evaluate(ctx, command)
	}
	return nil, nil
}

func (r *Registry) EvaluateFileEdit(ctx context.Context, filePath, content, sessionID string) ([]domain.Violation, error) {
	if r.fileEdit != nil {
		return r.fileEdit.Evaluate(ctx, filePath, content, sessionID)
	}
	return nil, nil
}

func (r *Registry) EvaluateInput(ctx context.Context, input string, categories []string) ([]domain.Violation, error) {
	// Collect violations from ALL matching categories (not just the first)
	var allViolations []domain.Violation
	for _, cat := range categories {
		switch cat {
		case "bash", "command":
			violations, err := r.EvaluateCommand(ctx, input)
			if err != nil {
				return nil, err
			}
			allViolations = append(allViolations, violations...)
		case "git":
			violations, err := r.EvaluateGit(ctx, input)
			if err != nil {
				return nil, err
			}
			allViolations = append(allViolations, violations...)
		case "file_edit":
			// file_edit requires file path, skip generic evaluation
		}
	}
	if len(allViolations) > 0 {
		return allViolations, nil
	}
	// Only fall back to bash when no explicit categories were requested.
	// When callers ask for specific categories, evaluate exactly those and
	// never silently expand scope (which would also double-evaluate bash that
	// was already in the requested list).
	if len(categories) == 0 {
		return r.EvaluateCommand(ctx, input)
	}
	return nil, nil
}

func (r *Registry) CheckFileRead(ctx context.Context, sessionID, filePath string) (*domain.FileReadVerification, error) {
	if r.transport != nil {
		return r.transport.CheckFileRead(ctx, sessionID, filePath)
	}
	// No transport configured — cannot verify file reads.
	// Return false to fail-closed rather than silently passing.
	return &domain.FileReadVerification{WasRead: false}, nil
}

// Ensure Registry implements domain.GuardrailService
var _ domain.GuardrailService = (*Registry)(nil)

// --- Concrete evaluator implementations (would live in separate files per slice) ---

type bashEvaluator struct {
	patternFn func(string, string) (bool, error)
}

func newBashEvaluator(fn func(string, string) (bool, error)) *bashEvaluator {
	return &bashEvaluator{patternFn: fn}
}

func (e *bashEvaluator) Evaluate(ctx context.Context, command string) ([]domain.Violation, error) {
	// Placeholder — actual impl delegates to slice
	return nil, nil
}

type gitEvaluator struct {
	patternFn func(string, string) (bool, error)
}

func newGitEvaluator(fn func(string, string) (bool, error)) *gitEvaluator {
	return &gitEvaluator{patternFn: fn}
}

func (e *gitEvaluator) Evaluate(ctx context.Context, command string) ([]domain.Violation, error) {
	return nil, nil
}

type fileEditEvaluator struct {
	patternFn func(string, string) (bool, error)
}

func newFileEditEvaluator(fn func(string, string) (bool, error)) *fileEditEvaluator {
	return &fileEditEvaluator{patternFn: fn}
}

func (e *fileEditEvaluator) Evaluate(ctx context.Context, filePath, content, sessionID string) ([]domain.Violation, error) {
	return nil, nil
}
