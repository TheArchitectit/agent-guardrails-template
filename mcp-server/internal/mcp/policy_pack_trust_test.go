package mcp

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/thearchitectit/guardrail-mcp/internal/evidence"
	"github.com/thearchitectit/guardrail-mcp/internal/guardrails"
)

// Spec 07 R07-4.3.4 / R07-4.3.5 — the request path for provenance.
//
// The loader proof lives in the guardrails package. What matters here is that a
// RUNNING server reports the trust state of the set it is actually enforcing,
// and that a failed trust-set load is reported as a failure with a reason
// rather than as "nothing configured". A loader that records provenance into a
// struct nobody reads is the same absence this spec keeps finding.

// signPackTree copies the checked-in pack tree into a temp dir, signs every
// pack, and writes a trust set. It returns the dir and the trust set.
//
// Copying rather than signing in place is deliberate: the checked-in tree must
// stay unsigned, because a repository that ships signatures over its own packs
// signed by a test key would look verified in production.
func signPackTree(t *testing.T) (string, *guardrails.TrustedKeySet) {
	t.Helper()
	src := policyPackDir(t)
	dir := t.TempDir()

	paths, err := filepath.Glob(filepath.Join(src, "core", "*.pack.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no packs under %s: %v", src, err)
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	for _, p := range paths {
		raw, rerr := os.ReadFile(p)
		if rerr != nil {
			t.Fatalf("read %s: %v", p, rerr)
		}
		rel, rerr := filepath.Rel(src, p)
		if rerr != nil {
			t.Fatalf("rel: %v", rerr)
		}
		dst := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(dst, raw, 0o644); err != nil {
			t.Fatalf("write %s: %v", dst, err)
		}
		sig, serr := guardrails.SignPolicyPack(priv, raw)
		if serr != nil {
			t.Fatalf("sign %s: %v", rel, serr)
		}
		if err := os.WriteFile(dst+".sig", sig, 0o644); err != nil {
			t.Fatalf("write sig: %v", err)
		}
	}
	lock, err := os.ReadFile(filepath.Join(src, "LOCK.json"))
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "LOCK.json"), lock, 0o644); err != nil {
		t.Fatalf("write lock: %v", err)
	}

	doc, _ := json.Marshal(map[string]any{
		"keys": []map[string]string{{
			"key_id":     evidence.KeyIDFor(pub),
			"public_key": base64.StdEncoding.EncodeToString(pub),
		}},
	})
	trustPath := filepath.Join(dir, "trusted-keys.json")
	if err := os.WriteFile(trustPath, doc, 0o600); err != nil {
		t.Fatalf("write trust: %v", err)
	}
	if err := os.Chmod(trustPath, 0o600); err != nil {
		t.Fatalf("chmod trust: %v", err)
	}
	set, err := guardrails.LoadTrustedKeySet(trustPath)
	if err != nil {
		t.Fatalf("load trust: %v", err)
	}
	return dir, set
}

// TestPolicyPackStatusReportsTrustStates is the R07-4.3.5 request path: the
// operator reading guardrail://policy-packs sees a provenance row per pack and
// sees whether verification was enabled.
func TestPolicyPackStatusReportsTrustStates(t *testing.T) {
	dir, trusted := signPackTree(t)
	s := &MCPServer{}
	m := guardrails.NewPolicyPackManager()
	if err := m.ActivateFromVerified(dir, s.RegisteredToolNames(), trusted); err != nil {
		t.Fatalf("activate verified: %v", err)
	}
	s.SetPolicyPacks(m)

	status := s.policyPackStatusSnapshot(nil)
	if status.Status != "active" {
		t.Fatalf("status = %q, want active", status.Status)
	}
	if !status.SignatureVerification {
		t.Fatal("signature_verification must report true for a set activated with a trust set")
	}
	if len(status.Trust) != len(status.Packs) {
		t.Fatalf("want one trust row per pack (%d), got %d", len(status.Packs), len(status.Trust))
	}
	for _, row := range status.Trust {
		if row.State != guardrails.TrustVerified {
			t.Fatalf("pack %q reported %q, want verified", row.Path, row.State)
		}
		if row.KeyID == "" {
			t.Fatalf("verified pack %q reports no key id", row.Path)
		}
	}

	// The same handler, read through the registered resource, must carry the
	// field — a Go accessor nobody surfaces is not a diagnostic.
	result, err := s.readPolicyPacksResource(t.Context(), "guardrail://policy-packs")
	if err != nil {
		t.Fatalf("read resource: %v", err)
	}
	if len(result.Contents) != 1 {
		t.Fatalf("want 1 resource content, got %d", len(result.Contents))
	}
	content, ok := result.Contents[0].(mcp.TextResourceContents)
	if !ok {
		t.Fatalf("resource content is %T, want TextResourceContents", result.Contents[0])
	}
	body := content.Text
	if !strings.Contains(body, `"trust"`) || !strings.Contains(body, `"verified"`) {
		t.Fatalf("resource body does not report trust; got %s", body)
	}
}

// TestPolicyPackStatusPinnedIsNotVerified is the conflation guard on the wire:
// the shipped, unsigned tree reports pinned and signature_verification=false,
// so a reader cannot mistake lock-only provenance for a verified set.
func TestPolicyPackStatusPinnedIsNotVerified(t *testing.T) {
	s := &MCPServer{}
	m := guardrails.NewPolicyPackManager()
	if err := m.ActivateFrom(policyPackDir(t), s.RegisteredToolNames()); err != nil {
		t.Fatalf("activate: %v", err)
	}
	s.SetPolicyPacks(m)

	status := s.policyPackStatusSnapshot(nil)
	if status.SignatureVerification {
		t.Fatal("the unsigned tree must not report signature verification as enabled")
	}
	if len(status.Trust) == 0 {
		t.Fatal("a pinned set must still report trust rows; an empty list reads as 'nothing to report'")
	}
	for _, row := range status.Trust {
		if row.State != guardrails.TrustPinned {
			t.Fatalf("pack %q reported %q, want pinned", row.Path, row.State)
		}
		if row.KeyID != "" {
			t.Fatalf("pinned pack %q names key %q; an unverified key id is an identity claim", row.Path, row.KeyID)
		}
	}
}

// TestPolicyPackStatusReportsTrustSetFailure: an unusable trusted key set is
// reported as "failed" WITH the reason. Before this, startup had no way to say
// why a configured set was not active, so an operator saw "no policy set has
// been activated" for a set that was rejected for a specific reason.
func TestPolicyPackStatusReportsTrustSetFailure(t *testing.T) {
	s := &MCPServer{}
	s.SetPolicyPacksError(fmt.Errorf("policy pack trusted key set: empty trust set"))

	status := s.policyPackStatusSnapshot(nil)
	if status.Status != "failed" {
		t.Fatalf("status = %q, want failed", status.Status)
	}
	if !strings.Contains(status.Error, "trusted key set") {
		t.Fatalf("error does not name the trust set; got %q", status.Error)
	}
}
