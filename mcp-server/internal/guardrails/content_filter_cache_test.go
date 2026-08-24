package guardrails

import (
	"testing"
	"time"
)

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

func TestResultCache_MaxSizeEviction(t *testing.T) {
	cache := newResultCache(time.Minute, 3)
	cache.Set("a", &ClassificationResult{Safe: true})
	cache.Set("b", &ClassificationResult{Safe: true})
	cache.Set("c", &ClassificationResult{Safe: true})
	if cache.Len() != 3 {
		t.Fatalf("expected 3 entries, got %d", cache.Len())
	}

	// Adding a 4th entry should evict the oldest ("a").
	cache.Set("d", &ClassificationResult{Safe: true})
	if cache.Len() != 3 {
		t.Fatalf("expected 3 entries after eviction, got %d", cache.Len())
	}
	if got := cache.Get("a"); got != nil {
		t.Error("oldest entry 'a' should have been evicted")
	}
	if got := cache.Get("d"); got == nil {
		t.Error("newest entry 'd' should still be present")
	}

	// Existing keys are updated in place and do not grow the map.
	cache.Set("b", &ClassificationResult{Safe: false})
	if cache.Len() != 3 {
		t.Errorf("expected 3 entries after in-place update, got %d", cache.Len())
	}
	if got := cache.Get("b"); got == nil || got.Safe {
		t.Error("updated entry 'b' should reflect new value")
	}
}

func TestResultCache_StopReleasesGoroutine(t *testing.T) {
	cache := newResultCache(10*time.Millisecond, 100)
	cache.Set("k", &ClassificationResult{Safe: true})
	cache.Stop()
	cache.Stop() // idempotent
	cache.Set("k2", &ClassificationResult{Safe: true})
	if got := cache.Get("k2"); got == nil {
		t.Error("cache should still be usable after Stop")
	}
}
