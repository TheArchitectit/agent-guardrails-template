package guardrails

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestWrapWithProvenance_TrustedSource(t *testing.T) {
	prov := &Provenance{
		Source:     "file",
		SourcePath: "CLAUDE.md",
		TrustLevel: TrustLevelTrusted,
	}

	content := "This is trusted content."
	result := WrapWithProvenance(content, prov)

	if result != content {
		t.Errorf("expected unchanged content, got %q", result)
	}
}

func TestWrapWithProvenance_UntrustedSource(t *testing.T) {
	prov := &Provenance{
		Source:     "file",
		SourcePath: "config/data.json",
		TrustLevel: TrustLevelUntrusted,
	}

	content := "This is untrusted content."
	result := WrapWithProvenance(content, prov)

	if !strings.Contains(result, "[UNTRUSTED]") {
		t.Errorf("expected [UNTRUSTED] marker, got %q", result)
	}
	if !strings.Contains(result, "config/data.json") {
		t.Errorf("expected source path, got %q", result)
	}
}

func TestWrapWithProvenance_UnknownSource(t *testing.T) {
	prov := &Provenance{
		Source:     "api",
		SourcePath: "https://example.com",
		TrustLevel: TrustLevelUnknown,
	}

	content := "This is unknown content."
	result := WrapWithProvenance(content, prov)

	if !strings.Contains(result, "unknown source") {
		t.Errorf("expected source marker, got %q", result)
	}
}

func TestInMemoryProvenanceCache_SetAndGet(t *testing.T) {
	cache := NewInMemoryProvenanceCache(time.Hour)
	ctx := context.Background()

	prov := &Provenance{
		Source:     "file",
		SourcePath: "test.json",
		TrustLevel: TrustLevelUntrusted,
	}

	err := cache.Set(ctx, "hash1", prov, time.Hour)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, ok := cache.Get(ctx, "hash1")
	if !ok {
		t.Error("expected cache hit")
	}
	if got.Source != "file" {
		t.Errorf("expected source=file, got %q", got.Source)
	}
}

func TestInMemoryProvenanceCache_Miss(t *testing.T) {
	cache := NewInMemoryProvenanceCache(time.Hour)
	ctx := context.Background()

	_, ok := cache.Get(ctx, "nonexistent")
	if ok {
		t.Error("expected cache miss")
	}
}

func TestInMemoryProvenanceCache_Expiration(t *testing.T) {
	cache := NewInMemoryProvenanceCache(time.Hour)
	ctx := context.Background()

	prov := &Provenance{
		Source:     "file",
		SourcePath: "test.json",
		TrustLevel: TrustLevelUntrusted,
	}

	err := cache.Set(ctx, "hash1", prov, time.Millisecond)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	_, ok := cache.Get(ctx, "hash1")
	if ok {
		t.Error("expected cache miss after expiration")
	}
}

func TestInMemoryProvenanceCache_Prune(t *testing.T) {
	cache := NewInMemoryProvenanceCache(time.Hour)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		prov := &Provenance{
			Source:     "file",
			SourcePath: fmt.Sprintf("test%d.json", i),
			TrustLevel: TrustLevelUntrusted,
		}
		_ = cache.Set(ctx, fmt.Sprintf("hash%d", i), prov, time.Millisecond)
	}

	for i := 5; i < 10; i++ {
		prov := &Provenance{
			Source:     "file",
			SourcePath: fmt.Sprintf("test%d.json", i),
			TrustLevel: TrustLevelUntrusted,
		}
		_ = cache.Set(ctx, fmt.Sprintf("hash%d", i), prov, time.Hour)
	}

	time.Sleep(10 * time.Millisecond)

	pruned := cache.Prune()
	if pruned != 5 {
		t.Errorf("expected 5 pruned, got %d", pruned)
	}
	if cache.Count() != 5 {
		t.Errorf("expected 5 remaining, got %d", cache.Count())
	}
}
