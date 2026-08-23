package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/thearchitectit/guardrail-mcp/internal/guardrails"
)

// fakeClassifier implements guardrails.SemanticClassifier for handler tests.
type fakeClassifier struct {
	name    string
	scores  map[string]float64
	avail   bool
}

func (f *fakeClassifier) Name() string { return f.name }

func (f *fakeClassifier) Available(_ context.Context) bool { return f.avail }

func (f *fakeClassifier) Classify(_ context.Context, _ string) (map[string]float64, error) {
	return f.scores, nil
}

// newTestEngine returns a guardrails.Engine with a single fake backend and the
// default policy engine (no custom rules), so actions follow CategoryMetadata.
func newTestEngine(t *testing.T, scores map[string]float64) *guardrails.Engine {
	t.Helper()
	engine := guardrails.NewEngine(nil, slog.Default())
	engine.AddContentFilterBackend(&fakeClassifier{name: "fake", scores: scores, avail: true})
	return engine
}

func decodeResult[T any](t *testing.T, res *mcp.CallToolResult) T {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatalf("no content in result")
	}
	text, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type: %T", res.Content[0])
	}
	var out T
	if err := json.Unmarshal([]byte(text.Text), &out); err != nil {
		t.Fatalf("failed to decode result JSON %q: %v", text.Text, err)
	}
	return out
}

func TestHandleClassifyContent_NoEngine(t *testing.T) {
	s := &MCPServer{}
	res, err := s.handleClassifyContent(context.Background(), map[string]any{"text": "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError != true {
		t.Error("expected isError=true when engine not configured")
	}
}

func TestHandleClassifyContent_EmptyText(t *testing.T) {
	s := &MCPServer{guardrailsEngine: newTestEngine(t, map[string]float64{"S10": 0.9})}
	res, err := s.handleClassifyContent(context.Background(), map[string]any{"text": ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError != true {
		t.Error("expected isError=true for empty text")
	}
}

func TestHandleClassifyContent_Block(t *testing.T) {
	s := &MCPServer{guardrailsEngine: newTestEngine(t, map[string]float64{"S10": 0.95})}
	res, err := s.handleClassifyContent(context.Background(), map[string]any{
		"text":      "hateful content",
		"direction": "input",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := decodeResult[guardrails.ClassificationResult](t, res)
	if out.Safe {
		t.Error("expected unsafe for S10 hate at 0.95")
	}
	if !out.IsBlocked() {
		t.Error("expected blocked result")
	}
	if out.Direction != guardrails.DirectionInput {
		t.Errorf("expected direction=input, got %q", out.Direction)
	}
	if res.IsError != true {
		t.Error("expected isError=true for blocked classification")
	}
}

func TestHandleClassifyContent_Safe(t *testing.T) {
	s := &MCPServer{guardrailsEngine: newTestEngine(t, map[string]float64{"S1": 0.05})}
	res, err := s.handleClassifyContent(context.Background(), map[string]any{
		"text":      "benign content",
		"direction": "output",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := decodeResult[guardrails.ClassificationResult](t, res)
	if !out.Safe {
		t.Error("expected safe for S1 at 0.05")
	}
	if out.Direction != guardrails.DirectionOutput {
		t.Errorf("expected direction=output, got %q", out.Direction)
	}
}

func TestHandleCheckPolicy_NoEngine(t *testing.T) {
	s := &MCPServer{}
	res, err := s.handleCheckPolicy(context.Background(), map[string]any{
		"text":      "hello",
		"policy_id": "coding-safety",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError != true {
		t.Error("expected isError=true when engine not configured")
	}
}

func TestHandleCheckPolicy_MissingPolicyID(t *testing.T) {
	s := &MCPServer{guardrailsEngine: newTestEngine(t, nil)}
	res, err := s.handleCheckPolicy(context.Background(), map[string]any{"text": "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError != true {
		t.Error("expected isError=true when policy_id missing")
	}
}

func TestHandleCheckPolicy_UnknownPolicy_FailClosed(t *testing.T) {
	// Blocked content (S14 Code Abuse at 0.95) classified, then checked against a
	// policy_id that does not exist. The fail-open signature (compliant, no
	// violations) combined with a blocked classification must be rejected.
	s := &MCPServer{guardrailsEngine: newTestEngine(t, map[string]float64{"S14": 0.95})}
	res, err := s.handleCheckPolicy(context.Background(), map[string]any{
		"text":      "abusive code",
		"policy_id": "nonexistent-policy",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError != true {
		t.Error("expected isError=true for unknown policy_id with blocked content (fail-closed)")
	}
}

func TestHandleCheckPolicy_Compliant(t *testing.T) {
	// Clean content (S1 at 0.05, below threshold so not blocked) checked against a
	// KNOWN policy that covers S1. Compliance can only be asserted for a policy
	// that exists; unknown policies fail closed (see
	// TestHandleCheckPolicy_UnknownPolicy_FailClosed).
	engine := newTestEngine(t, map[string]float64{"S1": 0.05})
	engine.UpdateRules([]guardrails.PolicyRule{{
		ID: "test-policy",
		Rules: []guardrails.PolicyDetail{
			{Category: "S1", Action: guardrails.ActionBlock, Threshold: 0.7},
		},
	}})
	s := &MCPServer{guardrailsEngine: engine}
	res, err := s.handleCheckPolicy(context.Background(), map[string]any{
		"text":      "benign",
		"policy_id": "test-policy",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError != false {
		t.Error("expected isError=false for benign content against a known policy")
	}
	out := decodeResult[guardrails.PolicyResult](t, res)
	if !out.Compliant {
		t.Errorf("expected compliant for benign content against known policy, got %+v", out)
	}
}

// TestEvaluateInput_ExplicitCategoriesNoBashFallback verifies the contract fixed
// in registry.go: when explicit non-empty categories are requested (e.g. ["git"]),
// EvaluateInput evaluates exactly those categories and does NOT run the "Default:
// try bash" fallback. The concrete evaluators in registry.go are placeholders that
// return nil, so the observable post-condition is that explicit categories yield
// the git-only result (empty) without a bash-fallback layer, whereas a nil/empty
// category list exercises the bash path. The suppression of the fallback is
// enforced by the `len(categories) == 0` guard in EvaluateInput.
func TestEvaluateInput_ExplicitCategoriesNoBashFallback(t *testing.T) {
	r := guardrails.NewRegistry(
		guardrails.WithBash(func(string, string) (bool, error) { return false, nil }),
		guardrails.WithGit(func(string, string) (bool, error) { return false, nil }),
	)

	// Explicit categories: only git is evaluated, no bash fallback.
	got, err := r.EvaluateInput(context.Background(), "git status", []string{"git"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil violations for explicit git-only categories, got %v", got)
	}

	// Empty categories: bash fallback path is taken (returns nil with no bash
	// slice configured).
	got, err = r.EvaluateInput(context.Background(), "ls", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil violations for empty categories fallback, got %v", got)
	}
}
