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
	res, err := s.handleClassifyContent(context.Background(), map[string]interface{}{"text": "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError != true {
		t.Error("expected isError=true when engine not configured")
	}
}

func TestHandleClassifyContent_EmptyText(t *testing.T) {
	s := &MCPServer{guardrailsEngine: newTestEngine(t, map[string]float64{"S10": 0.9})}
	res, err := s.handleClassifyContent(context.Background(), map[string]interface{}{"text": ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError != true {
		t.Error("expected isError=true for empty text")
	}
}

func TestHandleClassifyContent_Block(t *testing.T) {
	s := &MCPServer{guardrailsEngine: newTestEngine(t, map[string]float64{"S10": 0.95})}
	res, err := s.handleClassifyContent(context.Background(), map[string]interface{}{
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
	res, err := s.handleClassifyContent(context.Background(), map[string]interface{}{
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
	res, err := s.handleCheckPolicy(context.Background(), map[string]interface{}{
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
	res, err := s.handleCheckPolicy(context.Background(), map[string]interface{}{"text": "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError != true {
		t.Error("expected isError=true when policy_id missing")
	}
}

func TestHandleCheckPolicy_Compliant(t *testing.T) {
	s := &MCPServer{guardrailsEngine: newTestEngine(t, map[string]float64{"S1": 0.05})}
	res, err := s.handleCheckPolicy(context.Background(), map[string]interface{}{
		"text":      "benign",
		"policy_id": "nonexistent-policy",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := decodeResult[guardrails.PolicyResult](t, res)
	if !out.Compliant {
		t.Error("expected compliant for unknown policy")
	}
}
