package guardrails

import (
	"context"
	"testing"
)

// TestEngine_TrustedSourceStillRunsContentFilter verifies that a Trusted source
// tag no longer unconditionally allows content. Trusted sources skip the
// injection deep-scan, but content-filter classification (and fail-closed
// behavior) still runs, so blocked content is still blocked.
func TestEngine_TrustedSourceStillRunsContentFilter(t *testing.T) {
	cfg := DefaultEngineConfig()
	cfg.Injection.Enabled = false
	cfg.ContentFilter.Enabled = true

	// A trusted source path (matches the default trust list).
	trustedPolicy := []TrustPolicy{
		{SourcePattern: "docs/safe.md", TrustLevel: TrustLevelTrusted, Action: ActionAllow},
	}
	cfg.Provenance = &ProvenanceConfig{
		Enabled:             true,
		SourceTrustPolicies: trustedPolicy,
	}

	// Backend that classifies any non-empty text as S10 (Hate -> block).
	blockBackend := newAvailableClassifier("fake", map[string]float64{"S10": 0.9})

	e := NewEngine(cfg, nil)
	e.AddContentFilterBackend(blockBackend)
	defer e.Stop()

	ctx := context.Background()
	res := e.Evaluate(ctx, EvalInput{
		Text:       "hateful content",
		Source:     SourceFileContent,
		SourceTool: "docs/safe.md",
		Direction:  DirectionInput,
	})

	if res.Safe {
		t.Error("trusted source must not unconditionally allow blocked content")
	}
	if res.Decision != "block" {
		t.Errorf("expected decision 'block' for trusted blocked content, got %q", res.Decision)
	}
	if res.Content == nil || !res.Content.IsBlocked() {
		t.Error("expected content classification to run for trusted source")
	}
	if res.Provenance == nil || res.Provenance.TrustLevel != TrustLevelTrusted {
		t.Error("expected trusted provenance to be recorded")
	}
}

// TestEngine_TrustedSourceAllowWhenSafe verifies the trusted fast-path still
// allows benign content (no regression on the common case).
func TestEngine_TrustedSourceAllowWhenSafe(t *testing.T) {
	cfg := DefaultEngineConfig()
	cfg.Injection.Enabled = false
	cfg.ContentFilter.Enabled = true

	trustedPolicy := []TrustPolicy{
		{SourcePattern: "docs/safe.md", TrustLevel: TrustLevelTrusted, Action: ActionAllow},
	}
	cfg.Provenance = &ProvenanceConfig{
		Enabled:             true,
		SourceTrustPolicies: trustedPolicy,
	}

	// Backend that classifies nothing as blocked.
	safeBackend := newAvailableClassifier("fake", map[string]float64{})

	e := NewEngine(cfg, nil)
	e.AddContentFilterBackend(safeBackend)
	defer e.Stop()

	ctx := context.Background()
	res := e.Evaluate(ctx, EvalInput{
		Text:       "benign content",
		Source:     SourceFileContent,
		SourceTool: "docs/safe.md",
		Direction:  DirectionInput,
	})

	if !res.Safe || res.Decision != "allow" {
		t.Errorf("trusted benign content should still be allowed, got safe=%v decision=%q",
			res.Safe, res.Decision)
	}
}