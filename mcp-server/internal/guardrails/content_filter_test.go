package guardrails

import (
	"context"
	"errors"
	"testing"
)

// fakeClassifier is a test double for SemanticClassifier.
type fakeClassifier struct {
	name      string
	scores    map[string]float64
	err       error
	available bool
	// classifyCalls counts invocations for cache-hit assertions.
	classifyCalls int
}

func (f *fakeClassifier) Name() string { return f.name }

func (f *fakeClassifier) Available(_ context.Context) bool { return f.available }

func (f *fakeClassifier) Classify(_ context.Context, _ string) (map[string]float64, error) {
	f.classifyCalls++
	if f.err != nil {
		return nil, f.err
	}
	return f.scores, nil
}

func newAvailableClassifier(name string, scores map[string]float64) *fakeClassifier {
	return &fakeClassifier{name: name, scores: scores, available: true}
}

// === ContentFilter.Classify ===

func TestClassify_EmptyText(t *testing.T) {
	cf := NewContentFilter(nil, nil)
	res, err := cf.Classify(context.Background(), "", DirectionInput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Safe {
		t.Error("empty text should be safe")
	}
	if res.Backend != "none" {
		t.Errorf("expected backend 'none', got %q", res.Backend)
	}
}

func TestClassify_FailOpen(t *testing.T) {
	cfg := DefaultFilterConfig()
	cfg.FailPolicy = ContentFailPolicyAllow
	cf := NewContentFilter(nil, nil, WithConfig(cfg))

	res, err := cf.Classify(context.Background(), "hello", DirectionInput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Safe {
		t.Error("fail-open should be safe when no backends available")
	}
	if res.Backend != "fail-open" {
		t.Errorf("expected backend 'fail-open', got %q", res.Backend)
	}
}

func TestClassify_FailClosed(t *testing.T) {
	cfg := DefaultFilterConfig()
	cfg.FailPolicy = ContentFailPolicyBlock
	cf := NewContentFilter(nil, nil, WithConfig(cfg))

	res, err := cf.Classify(context.Background(), "hello", DirectionInput)
	if err != nil {
		t.Fatalf("fail-closed should return block result without error, got: %v", err)
	}
	if res.Safe {
		t.Error("fail-closed should be unsafe when no backends available")
	}
	if !res.IsBlocked() {
		t.Error("fail-closed result should be blocked")
	}
	if res.Backend != "fail-closed" {
		t.Errorf("expected backend 'fail-closed', got %q", res.Backend)
	}
}

func TestClassify_BackendUnavailable_FailClosed(t *testing.T) {
	cfg := DefaultFilterConfig()
	cfg.FailPolicy = ContentFailPolicyBlock
	backend := &fakeClassifier{name: "unavailable", available: false}
	cf := NewContentFilter([]SemanticClassifier{backend}, nil, WithConfig(cfg))

	res, err := cf.Classify(context.Background(), "hello", DirectionInput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Safe {
		t.Error("unavailable backend with fail-closed should block")
	}
}

func TestClassify_BackendSuccess(t *testing.T) {
	backend := newAvailableClassifier("fake", map[string]float64{"S10": 0.9})
	cf := NewContentFilter([]SemanticClassifier{backend}, nil)

	res, err := cf.Classify(context.Background(), "hateful text", DirectionInput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Safe {
		t.Error("S10 at 0.9 should be blocked (Hate is ActionBlock)")
	}
	if res.Backend != "fake" {
		t.Errorf("expected backend 'fake', got %q", res.Backend)
	}
	if res.OverallRisk != 0.9 {
		t.Errorf("expected overall risk 0.9, got %v", res.OverallRisk)
	}
}

func TestClassify_CacheHit(t *testing.T) {
	backend := newAvailableClassifier("fake", map[string]float64{"S10": 0.9})
	cf := NewContentFilter([]SemanticClassifier{backend}, nil)

	ctx := context.Background()
	if _, err := cf.Classify(ctx, "cached text", DirectionInput); err != nil {
		t.Fatalf("first classify: %v", err)
	}
	if _, err := cf.Classify(ctx, "cached text", DirectionInput); err != nil {
		t.Fatalf("second classify: %v", err)
	}

	if backend.classifyCalls != 1 {
		t.Errorf("expected backend called once (cached second time), got %d calls", backend.classifyCalls)
	}
}

func TestClassify_CacheIsolation(t *testing.T) {
	backend := newAvailableClassifier("fake", map[string]float64{"S10": 0.9})
	cf := NewContentFilter([]SemanticClassifier{backend}, nil)

	ctx := context.Background()
	first, err := cf.Classify(ctx, "mutate me", DirectionInput)
	if err != nil {
		t.Fatalf("first classify: %v", err)
	}

	// Mutate the returned result — must not affect cached entry
	first.Categories[0].Action = ActionAllow
	first.Safe = true

	second, err := cf.Classify(ctx, "mutate me", DirectionInput)
	if err != nil {
		t.Fatalf("second classify: %v", err)
	}
	if second.Safe {
		t.Error("mutating returned result must not corrupt cached entry")
	}
	if len(second.Categories) == 0 || second.Categories[0].Action != ActionBlock {
		t.Error("cached entry should still hold block action")
	}
}

func TestClassify_BackendErrorThenFailClosed(t *testing.T) {
	cfg := DefaultFilterConfig()
	cfg.FailPolicy = ContentFailPolicyBlock
	backend := &fakeClassifier{
		name:      "broken",
		available: true,
		err:       errors.New("boom"),
	}
	cf := NewContentFilter([]SemanticClassifier{backend}, nil, WithConfig(cfg))

	res, err := cf.Classify(context.Background(), "hello", DirectionInput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Safe {
		t.Error("erroring backend with fail-closed should block")
	}
	// Reason should contain the underlying error, not a nil wrap
	if len(res.Categories) == 0 || res.Categories[0].Reason == "" {
		t.Error("fail-closed reason should contain the backend error")
	}
}

func TestContentFilter_UpdateRulesInvalidatesCache(t *testing.T) {
	// Blocking backend (S10 at 0.9) cached, then a rule override flips S10 to
	// allow. Without cache invalidation the second call would still block.
	backend := newAvailableClassifier("fake", map[string]float64{"S10": 0.9})
	cf := NewContentFilter([]SemanticClassifier{backend}, nil)
	defer cf.Stop()

	ctx := context.Background()
	first, err := cf.Classify(ctx, "text", DirectionInput)
	if err != nil {
		t.Fatalf("first classify: %v", err)
	}
	if first.Safe {
		t.Fatal("expected first result blocked (S10 default block)")
	}

	// Hot-reload rules: override S10 to allow.
	cf.UpdateRules([]PolicyRule{{
		ID:        "lenient",
		Overrides: []PolicyOverride{{Category: "S10", Action: ActionAllow, Description: "ok"}},
	}})

	second, err := cf.Classify(ctx, "text", DirectionInput)
	if err != nil {
		t.Fatalf("second classify: %v", err)
	}
	if !second.Safe {
		t.Error("after UpdateRules override to allow, cached stale block must be gone")
	}
}

func TestContentFilter_ClassifyStreamShortCircuitsOnBlock(t *testing.T) {
	// Dynamic classifier that flags S10 once "HATE_TRIGGER" appears in text.
	backend := &fakeClassifier{
		name:      "stream-fake",
		available: true,
		scores:    map[string]float64{"S10": 0.05},
	}
	cf := NewContentFilter([]SemanticClassifier{backend}, nil)
	defer cf.Stop()

	// Replace backend with one that inspects the accumulated buffer.
	streamBackend := &dynamicStreamClassifier{}
	cf = NewContentFilter([]SemanticClassifier{streamBackend}, nil)
	defer cf.Stop()

	ch := make(chan string, 4)
	ch <- "Hello safe prefix "
	ch <- "HATE_TRIGGER mid-stream "
	ch <- "trailing chunk that should not be read"
	close(ch)

	res, err := cf.ClassifyStream(context.Background(), ch, DirectionOutput)
	if err != nil {
		t.Fatalf("ClassifyStream: %v", err)
	}
	if res.Safe || !res.IsBlocked() {
		t.Fatalf("expected stream to block on trigger chunk, got Safe=%v", res.Safe)
	}
	if streamBackend.calls != 2 {
		t.Fatalf("expected short-circuit after 2nd chunk, got %d calls", streamBackend.calls)
	}
}

type dynamicStreamClassifier struct {
	calls int
}

func (d *dynamicStreamClassifier) Name() string                      { return "dynamic-stream" }
func (d *dynamicStreamClassifier) Available(_ context.Context) bool { return true }
func (d *dynamicStreamClassifier) Classify(_ context.Context, text string) (map[string]float64, error) {
	d.calls++
	if len(text) > 20 {
		return map[string]float64{"S10": 0.95}, nil
	}
	return map[string]float64{"S10": 0.01}, nil
}

func TestContentFilter_MultiBackendFailoverAndReloadConfig(t *testing.T) {
	primaryDown := &fakeClassifier{name: "llama-guard-down", available: false}
	secondaryErr := &fakeClassifier{name: "openai-err", available: true, err: errors.New("503 upstream")}
	tertiaryOK := newAvailableClassifier("perspective-ok", map[string]float64{"S11": 0.1})

	cf := NewContentFilter([]SemanticClassifier{primaryDown, secondaryErr, tertiaryOK}, nil)
	defer cf.Stop()

	res, err := cf.Classify(context.Background(), "benign text", DirectionInput)
	if err != nil {
		t.Fatalf("Classify failover: %v", err)
	}
	if !res.Safe {
		t.Fatalf("expected tertiary backend to classify safe, got Safe=false")
	}
	if res.Backend != "perspective-ok" {
		t.Fatalf("expected backend perspective-ok, got %q", res.Backend)
	}

	// ReloadConfig with updated policy rules clears cache and applies new config.
	newCfg := DefaultFilterConfig()
	newCfg.Policies = []PolicyRule{{
		ID:        "strict-s11",
		Overrides: []PolicyOverride{{Category: "S11", Action: ActionBlock, Description: "strict"}},
	}}
	if err := cf.ReloadConfig(newCfg); err != nil {
		t.Fatalf("ReloadConfig: %v", err)
	}
	if err := cf.ReloadConfig(nil); err == nil {
		t.Fatal("expected nil config reload to fail")
	}
}

