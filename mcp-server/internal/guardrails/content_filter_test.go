package guardrails

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
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

// === ResultCache ===

func TestResultCache_SetGet(t *testing.T) {
	cache := NewResultCache(time.Minute)
	key := "k"
	res := &ClassificationResult{Safe: false, Backend: "x"}
	cache.Set(key, res)

	got := cache.Get(key)
	if got == nil {
		t.Fatal("expected cached result, got nil")
	}
	if got.Backend != "x" {
		t.Errorf("expected backend 'x', got %q", got.Backend)
	}
}

func TestResultCache_Expiry(t *testing.T) {
	cache := NewResultCache(10 * time.Millisecond)
	cache.Set("k", &ClassificationResult{Safe: true})
	time.Sleep(20 * time.Millisecond)

	if got := cache.Get("k"); got != nil {
		t.Error("expired entry should return nil")
	}
}

func TestResultCache_DeepCopyOnGet(t *testing.T) {
	cache := NewResultCache(time.Minute)
	cache.Set("k", &ClassificationResult{
		Safe:       false,
		Categories: []CategoryResult{{ID: "S1", Action: ActionBlock}},
	})

	got := cache.Get("k")
	got.Categories[0].Action = ActionAllow
	got.Safe = true

	again := cache.Get("k")
	if again.Safe {
		t.Error("mutating returned copy must not corrupt cache")
	}
	if again.Categories[0].Action != ActionBlock {
		t.Error("cache entry should retain original block action")
	}
}

func TestResultCache_Clear(t *testing.T) {
	cache := NewResultCache(time.Minute)
	cache.Set("k", &ClassificationResult{Safe: true})
	cache.Clear()
	if got := cache.Get("k"); got != nil {
		t.Error("clear should remove all entries")
	}
}

// === PolicyEngine ===

func TestPolicyEngine_EvaluateDefaultActions(t *testing.T) {
	pe := NewPolicyEngine(nil)
	res := pe.Evaluate(map[string]float64{
		"S10": 0.9, // Hate -> block
		"S13": 0.8, // Elections -> warn
	}, "backend")
	if res.Safe {
		t.Error("S10 block should make result unsafe")
	}
	if res.OverallRisk != 0.9 {
		t.Errorf("expected overall risk 0.9, got %v", res.OverallRisk)
	}
}

func TestPolicyEngine_ThresholdDowngrades(t *testing.T) {
	pe := NewPolicyEngineWithThresholds(nil, ThresholdConfig{Default: 0.9})
	res := pe.Evaluate(map[string]float64{"S10": 0.5}, "backend")
	// 0.5 < 0.9 default threshold -> downgraded to allow
	if !res.Safe {
		t.Error("score below threshold should downgrade to allow")
	}
	if len(res.Categories) == 0 || res.Categories[0].Action != ActionAllow {
		t.Error("expected action allow for below-threshold score")
	}
}

func TestPolicyEngine_PerCategoryThreshold(t *testing.T) {
	pe := NewPolicyEngineWithThresholds(nil, ThresholdConfig{
		Default:     0.9,
		PerCategory: map[string]float64{"S10": 0.3},
	})
	// 0.5 >= 0.3 per-category threshold -> block retained
	res := pe.Evaluate(map[string]float64{"S10": 0.5}, "backend")
	if res.Safe {
		t.Error("score above per-category threshold should remain block")
	}
}

func TestPolicyEngine_RuleOverrides(t *testing.T) {
	pe := NewPolicyEngineWithThresholds([]PolicyRule{{
		ID: "strict",
		Rules: []PolicyDetail{{
			Category:  "S13",
			Action:    ActionBlock,
			Threshold: 0.0,
		}},
		Overrides: []PolicyOverride{{
			Category:    "S13",
			Action:      ActionAllow,
			Description: "elections exempt",
		}},
	}}, ThresholdConfig{Default: 0.7})

	// S13 default is Warn; override forces Allow even at high score
	res := pe.Evaluate(map[string]float64{"S13": 0.95}, "backend")
	if !res.Safe {
		t.Error("S13 override to Allow should make result safe")
	}
	if len(res.Categories) != 1 || res.Categories[0].Action != ActionAllow {
		t.Errorf("expected S13 override action allow, got %+v", res.Categories)
	}
}

func TestPolicyEngine_CheckPolicy(t *testing.T) {
	pe := NewPolicyEngine([]PolicyRule{{
		ID: "p1",
		Rules: []PolicyDetail{{
			Category:    "S10",
			Action:      ActionBlock,
			Description: "no hate",
		}},
	}})

	res := pe.Evaluate(map[string]float64{"S10": 0.9}, "backend")
	pr := pe.CheckPolicy(res, "p1")
	if pr.Compliant {
		t.Error("blocking result should be non-compliant with policy")
	}
	if len(pr.Violations) == 0 {
		t.Error("expected violation for blocked S10")
	}

	// Unknown policy is compliant (no rules to violate)
	pr2 := pe.CheckPolicy(res, "unknown")
	if !pr2.Compliant {
		t.Error("unknown policy should default compliant")
	}
}

func TestPolicyEngine_UpdateRules(t *testing.T) {
	pe := NewPolicyEngine(nil)
	pe.UpdateRules([]PolicyRule{{ID: "new"}})
	res := pe.Evaluate(map[string]float64{"S10": 0.9}, "backend")
	if res.Safe {
		t.Error("S10 default block should still apply after update")
	}
}

// === CategoryMetadata taxonomy ===

func TestCategoryMetadata_CanonicalTaxonomy(t *testing.T) {
	cases := map[string]struct {
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

	for id, want := range cases {
		name, action := CategoryMetadata(id)
		if name != want.name {
			t.Errorf("%s: expected name %q, got %q", id, want.name, name)
		}
		if action != want.action {
			t.Errorf("%s: expected action %q, got %q", id, want.action, action)
		}
	}

	// Unknown category
	name, action := CategoryMetadata("S99")
	if name != "Unknown" || action != ActionWarn {
		t.Errorf("unknown category: expected (Unknown, warn), got (%q, %q)", name, action)
	}
}

// === Taxonomy alignment: backends must use canonical S1-S15 ===

func TestLlamaGuardPrompt_UsesCanonicalTaxonomy(t *testing.T) {
	prompt := contentFilterLlamaGuardPrompt("test")

	// The prompt must describe the canonical S14/S15 (Code Abuse / Data Exfiltration),
	// not the old mislabeled S8/S9.
	canonical := []string{
		"S5: Defamation",
		"S8: Intellectual Property",
		"S10: Hate",
		"S13: Elections",
		"S14: Code Abuse",
		"S15: Data Exfiltration",
	}
	for _, label := range canonical {
		if !strings.Contains(prompt, label) {
			t.Errorf("LlamaGuard prompt missing canonical label %q", label)
		}
	}

	// Ensure old incorrect labels are gone
	stale := []string{
		"S5: Suicide and Self-Harm",
		"S8: Code Abuse",
		"S9: Data Exfiltration",
		"S10: Harassment and Bullying",
		"S14: Deception and Misinformation",
		"S15: System Instruction Override",
	}
	for _, s := range stale {
		if strings.Contains(prompt, s) {
			t.Errorf("LlamaGuard prompt still contains stale label %q", s)
		}
	}
}

func TestOpenAIModeration_CanonicalMapping(t *testing.T) {
	// The OpenAI backend maps self-harm -> S11 (Self-Harm), hate/harassment -> S10 (Hate),
	// sexual -> S12 (Sexual Content), sexual minors -> S4 (Child Exploitation).
	// Verify the emptyScores baseline contains all canonical keys.
	b := NewOpenAIModerationBackend("")
	scores := b.emptyScores()
	for _, id := range AllCategoryIDs() {
		if _, ok := scores[id]; !ok {
			t.Errorf("emptyScores missing canonical category %s", id)
		}
	}
}
