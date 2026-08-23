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
		// Finding #7: case-insensitive matching (CLAUDE.md matches claude.md)
		{"CLAUDE.md", "claude.md", true},
		{"docs/**/*.md", "docs/guide.md", true},
		{"docs/**/*.md", "docs/sub/deep.md", true},
		{"docs/**/*.md", "other/guide.md", false},
		// Finding #4: a/**/c.md must require >=1 segment (no zero-collapse)
		{"a/**/c.md", "a/c.md", false},
		{"a/**/c.md", "a/b/c.md", true},
		// Finding #3: bare ** matches any path including single-segment
		{"**", "foo.md", true},
		{"**", "a/b/c.md", true},
		// Finding #1: trailing * imposes single segment boundary
		{"api.internal.*", "api.internal.dev", true},
		{"api.internal.*", "api.internal.prod", true},
		{"api.internal.*", "api.internal.foo", true},
		{"api.internal.*", "api.internal.evil.attacker.com", false},
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

// Finding #2: cache key must include source and agentID so re-tagging the same
// path with a different source/agent does not return stale cached provenance.
func TestProvenanceTracker_TagContent_CacheKeyIncludesSourceAndAgent(t *testing.T) {
	cache := NewInMemoryProvenanceCache(time.Hour)
	policies := DefaultProvenanceConfig().SourceTrustPolicies
	tracker := NewProvenanceTracker(policies, cache)

	ctx := context.Background()
	content := "Hello World"

	prov1, err := tracker.TagContent(ctx, content, "file", "test.json", "agent-001")
	if err != nil {
		t.Fatalf("TagContent (1): %v", err)
	}
	prov2, err := tracker.TagContent(ctx, content, "api", "test.json", "agent-001")
	if err != nil {
		t.Fatalf("TagContent (2): %v", err)
	}
	if prov1.Hash == prov2.Hash {
		t.Error("expected different hashes for different sources, got same")
	}
	if prov1.Source == prov2.Source {
		t.Errorf("expected different sources, both got %q", prov1.Source)
	}

	prov3, err := tracker.TagContent(ctx, content, "file", "test.json", "agent-002")
	if err != nil {
		t.Fatalf("TagContent (3): %v", err)
	}
	if prov1.Hash == prov3.Hash {
		t.Error("expected different hashes for different agents, got same")
	}
	if prov1.ReadBy == prov3.ReadBy {
		t.Errorf("expected different ReadBy, both got %q", prov1.ReadBy)
	}
}

// Finding #5: base64 decoder must NOT trigger on bare alphabet runs — only on
// explicitly marked content (base64: label or data-URI scheme).
func TestDecodeObfuscation_Base64RequiresMarker(t *testing.T) {
	// Ordinary English text must NOT be decoded as base64.
	ordinary := "The quick brown fox jumps over the lazy dog"
	decoded, wasObfuscated := DecodeObfuscation(ordinary)
	if wasObfuscated {
		t.Errorf("ordinary text was wrongly decoded: %q -> %q", ordinary, decoded)
	}
	if decoded != ordinary {
		t.Errorf("ordinary text was altered: %q -> %q", ordinary, decoded)
	}

	// Explicitly marked base64 should still decode.
	encoded := "base64:SGVsbG8gV29ybGQ="
	decoded, wasObfuscated = DecodeObfuscation(encoded)
	if !wasObfuscated {
		t.Error("explicitly marked base64 was not decoded")
	}
}

// Finding #6: ROT13 detector must catch the classic phrase and resist false
// positives on normal English.
func TestDecodeObfuscation_ROT13Detection(t *testing.T) {
	// Classic ROT13 phrase must be detected and decoded.
	rotPhrase := "Gur dhvpx oebja sbk bzcf"
	decoded, wasObfuscated := DecodeObfuscation(rotPhrase)
	if !wasObfuscated {
		t.Errorf("ROT13 phrase not detected: %q", rotPhrase)
	}
	if decoded == rotPhrase {
		t.Error("ROT13 phrase was not decoded")
	}

	// Normal English must NOT be flagged as ROT13.
	normal := "This is a normal English sentence about the weather"
	_, wasObfuscated = DecodeObfuscation(normal)
	if wasObfuscated {
		t.Errorf("normal English wrongly flagged as ROT13: %q", normal)
	}
}
