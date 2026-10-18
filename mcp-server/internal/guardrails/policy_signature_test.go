package guardrails

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
	"time"

	"github.com/thearchitectit/guardrail-mcp/internal/evidence"
)

// Spec 07 R07-4.3.4 / R07-4.3.5 — signature verification and the trust state.
//
// Every case here drives the REAL loader over REAL pack files on disk and a
// REAL Ed25519 signature. A test that constructed PackTrust directly would
// prove the struct has fields, which is not the claim: the claim is that a
// running resolution refuses an unverified set and reports which of the three
// provenance states it is in.

// packUnderTest is the pack body every fixture signs. It names a tool from
// knownToolsFor so the only variable in each case is provenance.
const packUnderTest = `{
  "pack": "%s",
  "version": "0.1.0",
  "tier": "core",
  "rules": [
    {
      "id": "core.test.rule",
      "law": "law1",
      "enforcement": "blocking",
      "tool": "guardrail_verify_file_read",
      "on_unknown": "stop",
      "evidence": "test fixture: unit-test evidence tag"
    }
  ]
}`

// signatureFixture builds a resolvable pack tree (pack + matching lock) and
// returns the directory and the raw pack bytes.
func signatureFixture(t *testing.T) (dir string, packBytes []byte, packRel string) {
	t.Helper()
	dir = t.TempDir()
	name := "core/test"
	body := fmt.Sprintf(packUnderTest, name)
	writePack(t, dir, name, body)
	writeLockFor(t, dir, "")

	path := filepath.Join(dir, name+".pack.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture pack: %v", err)
	}
	return dir, raw, name + ".pack.json"
}

// newSigningKey returns a deterministic-per-run Ed25519 key and its public
// half. Generated at run time on purpose: a key literal in source is
// indistinguishable from a leaked key to a secret scanner, and a fixture gets
// no exemption.
func newSigningKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return pub, priv
}

// writeTrustFile writes an out-of-band trust set at the restricted mode the
// loader requires (0600). A 0644 file must be refused by secretfile.Read, and
// one of the cases asserts exactly that.
func writeTrustFile(t *testing.T, dir string, pub ed25519.PublicKey, mode os.FileMode, extra string) string {
	t.Helper()
	doc := fmt.Sprintf(`{
  "keys": [
    {
      "key_id": %q,
      "public_key": %q%s
    }
  ]
}`, evidence.KeyIDFor(pub), base64.StdEncoding.EncodeToString(pub), extra)
	path := filepath.Join(dir, "trusted-keys.json")
	if err := os.WriteFile(path, []byte(doc), mode); err != nil {
		t.Fatalf("write trust file: %v", err)
	}
	// WriteFile honours umask; force the mode the test intends.
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("chmod trust file: %v", err)
	}
	return path
}

func writeSignature(t *testing.T, dir, packRel string, priv ed25519.PrivateKey, raw []byte) {
	t.Helper()
	body, err := SignPolicyPack(priv, raw)
	if err != nil {
		t.Fatalf("sign pack: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, packRel+".sig"), body, 0o644); err != nil {
		t.Fatalf("write signature: %v", err)
	}
}

// loadTrusted reads the trust set the way startup does, so the test exercises
// the same file read and the same validation.
func loadTrusted(t *testing.T, path string) *TrustedKeySet {
	t.Helper()
	set, err := LoadTrustedKeySet(path)
	if err != nil {
		t.Fatalf("load trusted key set: %v", err)
	}
	return set
}

// TestPinnedIsNotVerified is the central R07-4.3.5 case: an unsigned pack in a
// deployment without a trust set is `pinned`, and a pack carrying a signature
// nobody checked is STILL `pinned`. The second half is the conflation trap —
// a signature present is not a signature verified.
func TestPinnedIsNotVerified(t *testing.T) {
	dir, raw, rel := signatureFixture(t)

	set, err := LoadPolicySet(dir, knownToolsFor(t))
	if err != nil {
		t.Fatalf("unsigned pack without a trust set must resolve as pinned: %v", err)
	}
	rows := set.TrustStates()
	if len(rows) != 1 {
		t.Fatalf("want one trust row per pack, got %d", len(rows))
	}
	if rows[0].State != TrustPinned {
		t.Fatalf("state = %q, want %q", rows[0].State, TrustPinned)
	}
	if rows[0].Path != rel {
		t.Fatalf("row path = %q, want %q", rows[0].Path, rel)
	}

	// Now add a signature but no trust set: still pinned, never verified.
	_, priv := newSigningKey(t)
	writeSignature(t, dir, rel, priv, raw)

	set, err = LoadPolicySet(dir, knownToolsFor(t))
	if err != nil {
		t.Fatalf("a signature without a trust set must not block resolution: %v", err)
	}
	rows = set.TrustStates()
	if rows[0].State != TrustPinned {
		t.Fatalf("a signature nobody verified reported as %q; want %q", rows[0].State, TrustPinned)
	}
	if rows[0].KeyID != "" {
		t.Fatalf("a pinned row must not name a verified key id, got %q", rows[0].KeyID)
	}
}

// TestVerifiedRequiresAConfiguredKey proves the happy path end to end: a real
// signature over the real bytes verifies against a key read out of band.
func TestVerifiedRequiresAConfiguredKey(t *testing.T) {
	dir, raw, rel := signatureFixture(t)
	pub, priv := newSigningKey(t)
	writeSignature(t, dir, rel, priv, raw)
	trusted := loadTrusted(t, writeTrustFile(t, dir, pub, 0o600, ""))

	set, err := LoadPolicySetVerified(dir, knownToolsFor(t), trusted)
	if err != nil {
		t.Fatalf("a correctly signed pack must resolve: %v", err)
	}
	rows := set.TrustStates()
	if rows[0].State != TrustVerified {
		t.Fatalf("state = %q, want %q", rows[0].State, TrustVerified)
	}
	if rows[0].KeyID != evidence.KeyIDFor(pub) {
		t.Fatalf("key id = %q, want %q", rows[0].KeyID, evidence.KeyIDFor(pub))
	}
	if rows[0].Reason != "" {
		t.Fatalf("a verified row should carry no reason, got %q", rows[0].Reason)
	}
}

// TestVerificationEnabledRefusesUnsignedPack is the "MUST block resolution"
// clause. With a trust set configured, an unsigned pack is an error — reported
// as locally_modified, because the bytes carry no attestation at all.
func TestVerificationEnabledRefusesUnsignedPack(t *testing.T) {
	dir, _, _ := signatureFixture(t)
	pub, _ := newSigningKey(t)
	trusted := loadTrusted(t, writeTrustFile(t, dir, pub, 0o600, ""))

	_, err := LoadPolicySetVerified(dir, knownToolsFor(t), trusted)
	if err == nil {
		t.Fatal("verification enabled with an unsigned pack must fail resolution")
	}
	if !strings.Contains(err.Error(), string(TrustLocallyModified)) {
		t.Fatalf("error must name the state %s; got %v", TrustLocallyModified, err)
	}
	if !strings.Contains(err.Error(), "R07-4.3.4") {
		t.Fatalf("error must cite the requirement; got %v", err)
	}
}

// TestSignatureFromUnknownKeyIsRefused: a mathematically valid signature from a
// key nobody provisioned is not evidence of anything.
func TestSignatureFromUnknownKeyIsRefused(t *testing.T) {
	dir, raw, rel := signatureFixture(t)
	_, stranger := newSigningKey(t)
	writeSignature(t, dir, rel, stranger, raw)

	trustedPub, _ := newSigningKey(t)
	trusted := loadTrusted(t, writeTrustFile(t, dir, trustedPub, 0o600, ""))

	_, err := LoadPolicySetVerified(dir, knownToolsFor(t), trusted)
	if err == nil {
		t.Fatal("a signature from an untrusted key must fail resolution")
	}
	if !strings.Contains(err.Error(), "not in the out-of-band trusted key set") {
		t.Fatalf("error must say the key is not trusted; got %v", err)
	}
}

// TestTamperedBytesFailDespiteAValidLookingSignature re-signs the ORIGINAL
// bytes, then edits the pack and re-locks it. The lock therefore matches the
// tampered file, so the only thing that can catch the edit is the signature —
// which is exactly the failure a lock alone cannot detect.
func TestTamperedBytesFailDespiteAValidLookingSignature(t *testing.T) {
	dir, raw, rel := signatureFixture(t)
	pub, priv := newSigningKey(t)
	writeSignature(t, dir, rel, priv, raw)
	trusted := loadTrusted(t, writeTrustFile(t, dir, pub, 0o600, ""))

	// Edit the pack to weaken nothing visible in the lock: a re-lock makes the
	// digest agree, so the signature is the only remaining evidence.
	path := filepath.Join(dir, rel)
	edited := strings.Replace(string(raw), `"enforcement": "blocking"`, `"enforcement": "advisory"`, 1)
	if edited == string(raw) {
		t.Fatal("fixture mutation did not change the pack")
	}
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatalf("rewrite pack: %v", err)
	}
	writeLockFor(t, dir, "")

	_, err := LoadPolicySetVerified(dir, knownToolsFor(t), trusted)
	if err == nil {
		t.Fatal("edited bytes under a stale signature must fail resolution")
	}
	if !strings.Contains(err.Error(), "signature does not verify") {
		t.Fatalf("error must blame the signature; got %v", err)
	}
}

// TestEmptyTrustSetIsRefused: an empty trust set would reject every pack for
// the wrong reason, so it is refused at load rather than at use.
func TestEmptyTrustSetIsRefused(t *testing.T) {
	if _, err := ParseTrustedKeySet([]byte(`{"keys": []}`)); err == nil {
		t.Fatal("an empty trust set must be refused")
	}
	if _, err := ParseTrustedKeySet([]byte(`{}`)); err == nil {
		t.Fatal("a trust document with no keys must be refused")
	}
}

// TestTrustSetRejectsDeclaredKeyIDMismatch: a declared id that disagrees with
// its own public key would let one record claim two identities.
func TestTrustSetRejectsDeclaredKeyIDMismatch(t *testing.T) {
	pub, _ := newSigningKey(t)
	doc := fmt.Sprintf(`{"keys":[{"key_id":"ed25519:deadbeef","public_key":%q}]}`,
		base64.StdEncoding.EncodeToString(pub))
	if _, err := ParseTrustedKeySet([]byte(doc)); err == nil {
		t.Fatal("a declared key id that disagrees with its key must be refused")
	}
}

// TestTrustSetRejectsExpiredKey: an expired trust record is an unbounded grant
// that reads as diligence.
func TestTrustSetRejectsExpiredKey(t *testing.T) {
	pub, _ := newSigningKey(t)
	doc := fmt.Sprintf(`{"keys":[{"public_key":%q,"expires":%q}]}`,
		base64.StdEncoding.EncodeToString(pub),
		time.Now().Add(-time.Hour).UTC().Format(time.RFC3339))
	if _, err := ParseTrustedKeySet([]byte(doc)); err == nil {
		t.Fatal("an expired trust record must be refused")
	}
}

// TestTrustSetRejectsGroupReadableFile: the key set is public material but a
// writable-by-others file decides policy authorship, so the same restricted
// file rule applies.
func TestTrustSetRejectsGroupReadableFile(t *testing.T) {
	pub, _ := newSigningKey(t)
	path := writeTrustFile(t, t.TempDir(), pub, 0o644, "")
	if _, err := LoadTrustedKeySet(path); err == nil {
		t.Fatal("a group/world-readable trust set must be refused")
	}
}

// TestSignatureFileMustBeWellFormed: a truncated or renamed signature file is
// an error, not an implicit "unsigned".
func TestSignatureFileMustBeWellFormed(t *testing.T) {
	dir, _, rel := signatureFixture(t)
	pub, _ := newSigningKey(t)
	trusted := loadTrusted(t, writeTrustFile(t, dir, pub, 0o600, ""))
	if err := os.WriteFile(filepath.Join(dir, rel+".sig"), []byte(`{"key_id":"x"}`), 0o644); err != nil {
		t.Fatalf("write sig: %v", err)
	}
	_, err := LoadPolicySetVerified(dir, knownToolsFor(t), trusted)
	if err == nil {
		t.Fatal("a signature file with no signature must fail resolution")
	}
}

// TestVerifiedActivationLeavesPriorSetInForce proves the failure is atomic at
// the manager too: a rejected candidate must not blank the active policy.
func TestVerifiedActivationLeavesPriorSetInForce(t *testing.T) {
	dir, raw, rel := signatureFixture(t)
	pub, priv := newSigningKey(t)
	trusted := loadTrusted(t, writeTrustFile(t, dir, pub, 0o600, ""))

	m := NewPolicyPackManager()
	if err := m.ActivateFromVerified(dir, knownToolsFor(t), trusted); err == nil {
		t.Fatal("activation without a signature must fail")
	}
	if m.Active().Active() {
		t.Fatal("a failed first activation must not produce an active set")
	}

	writeSignature(t, dir, rel, priv, raw)
	if err := m.ActivateFromVerified(dir, knownToolsFor(t), trusted); err != nil {
		t.Fatalf("activation of a verified set: %v", err)
	}
	first := m.Active().EffectiveDigest()

	// Break the signature and try again; the prior digest must survive.
	if err := os.WriteFile(filepath.Join(dir, rel+".sig"), []byte(`{"key_id":"ed25519:none","signature":"AAAA"}`), 0o644); err != nil {
		t.Fatalf("break sig: %v", err)
	}
	if err := m.ActivateFromVerified(dir, knownToolsFor(t), trusted); err == nil {
		t.Fatal("a broken signature must fail activation")
	}
	if got := m.Active().EffectiveDigest(); got != first {
		t.Fatalf("failed activation changed the active digest: %q -> %q", first, got)
	}
}

// TestCheckedInTreeReportsPinned proves the shipped tree's real state: it has
// no signatures and no trust set, so every pack is pinned. This is the honest
// answer and it is what the diagnostic will show a deployment today.
func TestCheckedInTreeReportsPinned(t *testing.T) {
	set, err := LoadPolicySet(repoPackDir(t), knownToolsFor(t))
	if err != nil {
		t.Fatalf("checked-in pack set: %v", err)
	}
	rows := set.TrustStates()
	if len(rows) != len(set.Packs()) {
		t.Fatalf("want one trust row per pack (%d), got %d", len(set.Packs()), len(rows))
	}
	for _, row := range rows {
		if row.State != TrustPinned {
			t.Fatalf("checked-in tree pack %q reported %q; the shipped tree carries no signatures", row.Path, row.State)
		}
	}
}

// TestSignatureDoesNotGrantApproval is the "MUST NOT be treated as semantic
// approval" clause, made checkable: a verified pack that WEAKENS a core rule
// still needs an exception. If verification had been wired into the approval
// path this test would pass resolution.
func TestSignatureDoesNotGrantApproval(t *testing.T) {
	dir := t.TempDir()
	writePack(t, dir, "core/base", `{
  "pack": "core/base", "version": "0.1.0", "tier": "core",
  "rules": [{"id": "core.shared", "law": "law1", "enforcement": "blocking",
             "tool": "guardrail_verify_file_read", "on_unknown": "stop",
             "evidence": "test fixture: unit-test evidence tag"}]
}`)
	writePack(t, dir, "domain/weaker", `{
  "pack": "domain/weaker", "version": "0.1.0", "tier": "domain",
  "rules": [{"id": "core.shared", "law": "law1", "enforcement": "advisory",
             "tool": "guardrail_verify_file_read", "on_unknown": "stop",
             "evidence": "test fixture: unit-test evidence tag"}]
}`)
	writeLockFor(t, dir, "")

	pub, priv := newSigningKey(t)
	trusted := loadTrusted(t, writeTrustFile(t, dir, pub, 0o600, ""))
	for _, rel := range []string{"core/base.pack.json", "domain/weaker.pack.json"} {
		raw, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		writeSignature(t, dir, rel, priv, raw)
	}

	_, err := LoadPolicySetVerified(dir, knownToolsFor(t), trusted)
	if err == nil {
		t.Fatal("a verified signature must not authorise a weakening; R07-4.3.4 says it is not approval")
	}
	if !strings.Contains(err.Error(), "exception") {
		t.Fatalf("the refusal must be the exception requirement; got %v", err)
	}
}

// TestSignatureCoversExactBytes pins the signature domain: the same key over
// different bytes must not verify, and base64 of a valid signature over other
// content must be rejected.
func TestSignatureCoversExactBytes(t *testing.T) {
	_, priv := newSigningKey(t)
	a := []byte(`{"pack":"core/a"}`)
	b := []byte(`{"pack":"core/b"}`)
	body, err := SignPolicyPack(priv, a)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	var sig packSignatureFile
	if err := json.Unmarshal(body, &sig); err != nil {
		t.Fatalf("decode sig: %v", err)
	}
	pub := priv.Public().(ed25519.PublicKey)
	if err := verifyDetachedPackSignature(pub, a, sig.Signature); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
	if err := verifyDetachedPackSignature(pub, b, sig.Signature); err == nil {
		t.Fatal("a signature over different bytes must not verify")
	}
}
