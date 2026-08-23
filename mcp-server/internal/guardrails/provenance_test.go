package guardrails

import (
	"context"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Provenance tracking unit tests
// ---------------------------------------------------------------------------

func TestProvenanceTracker_TagContent_CreatesProvenance(t *testing.T) {
	cache := NewInMemoryProvenanceCache(time.Hour)
	policies := DefaultProvenanceConfig().SourceTrustPolicies
	tracker := NewProvenanceTracker(policies, cache)

	ctx := context.Background()
	content := "Hello World"
	prov, err := tracker.TagContent(ctx, content, "file", "test.json", "agent-001")
	if err != nil {
		t.Fatalf("TagContent: %v", err)
	}

	if prov.Source != "file" {
		t.Errorf("expected source=file, got %q", prov.Source)
	}
	if prov.SourcePath != "test.json" {
		t.Errorf("expected source_path=test.json, got %q", prov.SourcePath)
	}
	if prov.ReadBy != "agent-001" {
		t.Errorf("expected read_by=agent-001, got %q", prov.ReadBy)
	}
	if prov.Hash == "" {
		t.Error("expected non-empty hash")
	}
	if prov.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestProvenanceTracker_TagContent_CachesResult(t *testing.T) {
	cache := NewInMemoryProvenanceCache(time.Hour)
	policies := DefaultProvenanceConfig().SourceTrustPolicies
	tracker := NewProvenanceTracker(policies, cache)

	ctx := context.Background()
	content := "Hello World"

	prov1, err := tracker.TagContent(ctx, content, "file", "test.json", "agent-001")
	if err != nil {
		t.Fatalf("TagContent (1): %v", err)
	}

	prov2, err := tracker.TagContent(ctx, content, "file", "test.json", "agent-001")
	if err != nil {
		t.Fatalf("TagContent (2): %v", err)
	}

	if prov1.Hash != prov2.Hash {
		t.Errorf("expected same hash, got %q vs %q", prov1.Hash, prov2.Hash)
	}
}

func TestProvenanceTracker_TagContent_TrustedSource(t *testing.T) {
	cache := NewInMemoryProvenanceCache(time.Hour)
	policies := DefaultProvenanceConfig().SourceTrustPolicies
	tracker := NewProvenanceTracker(policies, cache)

	ctx := context.Background()
	prov, err := tracker.TagContent(ctx, "content", "file", "CLAUDE.md", "agent-001")
	if err != nil {
		t.Fatalf("TagContent: %v", err)
	}

	if prov.TrustLevel != TrustLevelTrusted {
		t.Errorf("expected trusted, got %q", prov.TrustLevel)
	}
}

func TestProvenanceTracker_TagContent_UntrustedSource(t *testing.T) {
	cache := NewInMemoryProvenanceCache(time.Hour)
	policies := DefaultProvenanceConfig().SourceTrustPolicies
	tracker := NewProvenanceTracker(policies, cache)

	ctx := context.Background()
	prov, err := tracker.TagContent(ctx, "content", "file", "config/data.json", "agent-001")
	if err != nil {
		t.Fatalf("TagContent: %v", err)
	}

	if prov.TrustLevel != TrustLevelUntrusted {
		t.Errorf("expected untrusted, got %q", prov.TrustLevel)
	}
}

func TestResolveTrust_TrustedSource(t *testing.T) {
	cache := NewInMemoryProvenanceCache(time.Hour)
	policies := DefaultProvenanceConfig().SourceTrustPolicies
	tracker := NewProvenanceTracker(policies, cache)

	level, action := tracker.resolveTrust("CLAUDE.md")
	if level != TrustLevelTrusted {
		t.Errorf("expected trusted, got %q", level)
	}
	if action != ActionAllow {
		t.Errorf("expected allow, got %q", action)
	}
}

func TestResolveTrust_UntrustedSource(t *testing.T) {
	cache := NewInMemoryProvenanceCache(time.Hour)
	policies := DefaultProvenanceConfig().SourceTrustPolicies
	tracker := NewProvenanceTracker(policies, cache)

	level, action := tracker.resolveTrust("config/data.json")
	if level != TrustLevelUntrusted {
		t.Errorf("expected untrusted, got %q", level)
	}
	if action != ActionScanAndWarn {
		t.Errorf("expected scan_and_warn, got %q", action)
	}
}

func TestResolveTrust_GithubSource(t *testing.T) {
	cache := NewInMemoryProvenanceCache(time.Hour)
	policies := DefaultProvenanceConfig().SourceTrustPolicies
	tracker := NewProvenanceTracker(policies, cache)

	level, action := tracker.resolveTrust("github.com")
	if level != TrustLevelUntrusted {
		t.Errorf("expected untrusted, got %q", level)
	}
	if action != ActionScanAndBlock {
		t.Errorf("expected scan_and_block, got %q", action)
	}
}

func TestResolveTrust_UnknownSource(t *testing.T) {
	cache := NewInMemoryProvenanceCache(time.Hour)
	policies := DefaultProvenanceConfig().SourceTrustPolicies
	tracker := NewProvenanceTracker(policies, cache)

	level, _ := tracker.resolveTrust("unknown-source.xyz")
	if level != TrustLevelUnknown {
		t.Errorf("expected unknown, got %q", level)
	}
}

func TestMatchPattern_Extension(t *testing.T) {
	tests := []struct {
		pattern, path string
		want          bool
	}{
		{"*.json", "config/data.json", true},
		{"*.json", "README.md", false},
		{"*.md", "docs/guide.md", true},
		{"*.yaml", "config.yaml", true},
	}

	for _, tc := range tests {
		got := matchPattern(tc.pattern, tc.path)
		if got != tc.want {
			t.Errorf("matchPattern(%q, %q) = %v, want %v", tc.pattern, tc.path, got, tc.want)
		}
	}
}

func TestMatchPattern_GlobPatterns(t *testing.T) {
	tests := []struct {
		pattern, path string
		want          bool
	}{
		{"CLAUDE.md", "CLAUDE.md", true},
		{"CLAUDE.md", "other.md", false},
		{"docs/**/*.md", "docs/guide.md", true},
		{"docs/**/*.md", "docs/sub/deep.md", true},
		{"docs/**/*.md", "other/guide.md", false},
		{"api.internal.*", "api.internal.dev", true},
		{"api.internal.*", "api.internal.prod", true},
		{"api.internal.*", "api.external.dev", false},
		{"localhost", "localhost", true},
		{"localhost", "remotehost", false},
		{"*", "anything", true},
		{"config/guardrails.yaml", "config/guardrails.yaml", true},
		{"config/guardrails.yaml", "other/guardrails.yaml", false},
		{"github.com", "github.com", true},
	}

	for _, tc := range tests {
		got := matchPattern(tc.pattern, tc.path)
		if got != tc.want {
			t.Errorf("matchPattern(%q, %q) = %v, want %v", tc.pattern, tc.path, got, tc.want)
		}
	}
}

func TestSanitizeContent_StripsZeroWidthChars(t *testing.T) {
	input := "Hello​World‌foo‍bar⁠baz"
	expected := "HelloWorldfoobarbaz"
	result := SanitizeContent(input)
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestSanitizeContent_StripsBidiOverrides(t *testing.T) {
	input := "Hello‮World‬foo"
	expected := "HelloWorldfoo"
	result := SanitizeContent(input)
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestSanitizeContent_StripsInvisibleFormatChars(t *testing.T) {
	input := "Hello⁡World⁢foo⁣bar⁤baz"
	expected := "HelloWorldfoobarbaz"
	result := SanitizeContent(input)
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestSanitizeContent_StripsBidiControlChars(t *testing.T) {
	input := "Hello⁦World⁧foo⁨bar⁩baz"
	expected := "HelloWorldfoobarbaz"
	result := SanitizeContent(input)
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestSanitizeContent_StripsLineSeparatorChars(t *testing.T) {
	input := "Hello World foo"
	expected := "HelloWorldfoo"
	result := SanitizeContent(input)
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestSanitizeContent_StripsInvisibleFormattingChars(t *testing.T) {
	input := "Hello؜World͏foo"
	expected := "HelloWorldfoo"
	result := SanitizeContent(input)
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestSanitizeContent_PreservesNormalText(t *testing.T) {
	input := "Hello World! This is normal text with numbers 123."
	result := SanitizeContent(input)
	if result != input {
		t.Errorf("expected unchanged, got %q", result)
	}
}

func TestSanitizeContent_EmptyInput(t *testing.T) {
	result := SanitizeContent("")
	if result != "" {
		t.Errorf("expected empty, got %q", result)
	}
}
