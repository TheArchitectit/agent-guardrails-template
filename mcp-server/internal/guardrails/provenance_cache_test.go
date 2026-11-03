package guardrails

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
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

// startMiniRESP2Server starts a hermetic in-process RESP2 server supporting
// GET, SET (with PX), DEL, and KEYS for testing RedisProvenanceCache.
func startMiniRESP2Server(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	type entry struct {
		val     string
		expires time.Time
	}
	var mu sync.Mutex
	store := make(map[string]entry)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				r := bufio.NewReader(c)
				for {
					_, _, args, _, err := readRESPValue(r)
					if err != nil || len(args) == 0 {
						return
					}
					cmd := strings.ToUpper(args[0])
					mu.Lock()
					switch cmd {
					case "SET":
						if len(args) >= 3 {
							exp := time.Now().Add(time.Hour)
							if len(args) >= 5 && strings.ToUpper(args[3]) == "PX" {
								if ms, err := strconv.ParseInt(args[4], 10, 64); err == nil {
									exp = time.Now().Add(time.Duration(ms) * time.Millisecond)
								}
							}
							store[args[1]] = entry{val: args[2], expires: exp}
							_, _ = c.Write([]byte("+OK\r\n"))
						} else {
							_, _ = c.Write([]byte("-ERR wrong args\r\n"))
						}
					case "GET":
						if len(args) >= 2 {
							e, ok := store[args[1]]
							if !ok || time.Now().After(e.expires) {
								delete(store, args[1])
								_, _ = c.Write([]byte("$-1\r\n"))
							} else {
								_, _ = c.Write([]byte(fmt.Sprintf("$%d\r\n%s\r\n", len(e.val), e.val)))
							}
						}
					case "DEL":
						deleted := 0
						for _, k := range args[1:] {
							if _, ok := store[k]; ok {
								delete(store, k)
								deleted++
							}
						}
						_, _ = c.Write([]byte(fmt.Sprintf(":%d\r\n", deleted)))
					case "KEYS":
						prefix := ""
						if len(args) >= 2 {
							prefix = strings.TrimSuffix(args[1], "*")
						}
						var keys []string
						now := time.Now()
						for k, e := range store {
							if now.After(e.expires) {
								delete(store, k)
								continue
							}
							if strings.HasPrefix(k, prefix) {
								keys = append(keys, k)
							}
						}
						var b strings.Builder
						b.WriteString(fmt.Sprintf("*%d\r\n", len(keys)))
						for _, k := range keys {
							b.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(k), k))
						}
						_, _ = c.Write([]byte(b.String()))
					default:
						_, _ = c.Write([]byte("-ERR unknown cmd\r\n"))
					}
					mu.Unlock()
				}
			}(conn)
		}
	}()

	return ln.Addr().String()
}

func TestRedisProvenanceCache_SetGetDeleteAndExpiration(t *testing.T) {
	addr := startMiniRESP2Server(t)
	cache := NewRedisProvenanceCache(addr, "guardrail:prov:", time.Hour)
	ctx := context.Background()

	prov := &Provenance{
		Source:     "api",
		SourcePath: "https://external.example/data",
		TrustLevel: TrustLevelUntrusted,
	}
	if err := cache.Set(ctx, "hash-redis-1", prov, time.Hour); err != nil {
		t.Fatalf("Redis Set: %v", err)
	}
	got, ok := cache.Get(ctx, "hash-redis-1")
	if !ok || got == nil {
		t.Fatal("expected Redis cache hit")
	}
	if got.SourcePath != prov.SourcePath || got.TrustLevel != TrustLevelUntrusted {
		t.Fatalf("unexpected Redis provenance: %+v", got)
	}
	if cache.Count() != 1 {
		t.Fatalf("expected Count()=1, got %d", cache.Count())
	}

	if err := cache.Delete(ctx, "hash-redis-1"); err != nil {
		t.Fatalf("Redis Delete: %v", err)
	}
	if _, ok := cache.Get(ctx, "hash-redis-1"); ok {
		t.Fatal("expected Redis cache miss after Delete")
	}

	// Fail-closed when Redis address is empty or unreachable.
	unconfigured := NewRedisProvenanceCache("", "guardrail:prov:", time.Hour)
	if err := unconfigured.Set(ctx, "h", prov, time.Minute); err == nil {
		t.Fatal("expected error on Set with empty Redis address")
	}
	if _, ok := unconfigured.Get(ctx, "h"); ok {
		t.Fatal("expected cache miss on Get with empty Redis address")
	}
}

