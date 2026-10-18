package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Spec 16 Sec. 4 R16-09 — credential issuance and registry authoring.
//
// R16-09 requires "approved key generation, issuance, distinct principal
// assignment, expiry, overlap/rotation, revocation" and states "Do not claim a
// lifecycle API exists until one is implemented and protected."
//
// Before this file the tree could resolve, reload, rotate and revoke, and could
// not issue: Digest() had one non-test caller (Resolve), so an operator minted
// a credential by computing an HMAC by hand and hand-writing registry JSON.
// These tests drive the REAL issuance path against the REAL registry loader —
// not a mock of it — because the defect class this program keeps finding is a
// helper that passes its own unit test while the production path rejects (or
// silently accepts) something else.

func issueOpts() IssueOptions {
	return IssueOptions{
		CredentialID: "cred-issued-1",
		PrincipalID:  "principal-issued-1",
		Role:         RoleDeveloper,
		Scopes:       []string{ScopeMCPRead, ScopeMCPMutate},
		Resources:    []string{"project-a"},
		Validity:     30 * 24 * time.Hour,
		Secret:       "issued-secret-value-for-tests",
	}
}

// TestIssuedCredentialResolvesThroughTheRealRegistry proves the record the
// issuer builds is accepted by the loader AND that the credential it hands out
// actually authenticates to the intended principal. An issuer that produced a
// record the server refuses, or one that resolves to nobody, would pass a test
// that only inspected fields.
func TestIssuedCredentialResolvesThroughTheRealRegistry(t *testing.T) {
	opts := issueOpts()
	rec, secret, err := IssueRecord(testVerifierKey, opts)
	if err != nil {
		t.Fatalf("IssueRecord: %v", err)
	}
	if secret != opts.Secret {
		t.Fatalf("issue must return the supplied secret, got a different value")
	}

	reg, err := New(testVerifierKey, []Record{rec})
	if err != nil {
		t.Fatalf("the issued record must be accepted by the loader, got %v", err)
	}
	p, ok := reg.Resolve(secret)
	if !ok {
		t.Fatal("the issued credential must resolve")
	}
	if p.ID != opts.PrincipalID || p.CredentialID != opts.CredentialID {
		t.Fatalf("resolved identity = %q/%q, want %q/%q", p.ID, p.CredentialID, opts.PrincipalID, opts.CredentialID)
	}
	if p.Role != RoleDeveloper {
		t.Fatalf("resolved role = %q, want %q", p.Role, RoleDeveloper)
	}
	if strings.Join(p.Scopes, ",") != "mcp:read,mcp:mutate" {
		t.Fatalf("resolved scopes = %v", p.Scopes)
	}
}

// TestIssuedRecordCarriesNoSecret asserts the persisted record holds only the
// keyed verifier: R16-09 / R17-10 require the raw credential never to be
// stored, and an issuer is the most likely place for it to leak into the file.
func TestIssuedRecordCarriesNoSecret(t *testing.T) {
	opts := issueOpts()
	rec, secret, err := IssueRecord(testVerifierKey, opts)
	if err != nil {
		t.Fatalf("IssueRecord: %v", err)
	}
	body, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(body), secret) {
		t.Fatal("the serialized record contains the raw secret")
	}
	if strings.Contains(string(body), opts.Secret) {
		t.Fatal("the serialized record contains the supplied secret")
	}
	if rec.Verifier != Digest(testVerifierKey, secret) {
		t.Fatal("the record must carry the keyed verifier of the secret it was issued with")
	}
}

// TestIssueRefusesUnknownRoleAndEmptyScopes covers the two ways issuance can
// mint a credential that resolves and then denies every call — which reads to
// its holder as a broken deployment rather than a policy decision.
func TestIssueRefusesUnknownRoleAndEmptyScopes(t *testing.T) {
	opts := issueOpts()
	opts.Role = "superuser"
	if _, _, err := IssueRecord(testVerifierKey, opts); err == nil {
		t.Fatal("an unknown role must be refused")
	} else if !strings.Contains(err.Error(), "role catalog") {
		t.Fatalf("error must name the role catalog, got %v", err)
	}

	opts = issueOpts()
	opts.Scopes = nil
	if _, _, err := IssueRecord(testVerifierKey, opts); err == nil {
		t.Fatal("an empty scope list must be refused")
	}

	// A scope outside the published catalog is refused by the loader; issuance
	// must surface that rather than write a file the server rejects.
	opts = issueOpts()
	opts.Scopes = []string{"root:everything"}
	if _, _, err := IssueRecord(testVerifierKey, opts); err == nil {
		t.Fatal("a scope outside the approved catalog must be refused")
	}
}

// TestIssueRefusesInvalidValidity covers the expiry bound at issuance: R16-09
// requires credentials to expire, and a request longer than the published
// maximum is refused rather than silently truncated.
func TestIssueRefusesInvalidValidity(t *testing.T) {
	opts := issueOpts()
	opts.Validity = 0
	if _, _, err := IssueRecord(testVerifierKey, opts); err == nil {
		t.Fatal("zero validity must be refused: a credential must expire")
	}

	opts = issueOpts()
	opts.Validity = MaxCredentialValidity + 24*time.Hour
	if _, _, err := IssueRecord(testVerifierKey, opts); err == nil {
		t.Fatal("validity beyond the published maximum must be refused")
	} else if !strings.Contains(err.Error(), "maximum") {
		t.Fatalf("error must name the maximum, got %v", err)
	}

	// Exactly at the bound is accepted.
	opts = issueOpts()
	opts.Validity = MaxCredentialValidity
	if _, _, err := IssueRecord(testVerifierKey, opts); err != nil {
		t.Fatalf("validity at the bound must be accepted, got %v", err)
	}
}

// TestGenerateCredentialSecretIsRandomAndSized proves the generated secret is
// not a constant or a short value: two issuances must differ, and the material
// must carry the published entropy.
func TestGenerateCredentialSecretIsRandomAndSized(t *testing.T) {
	a, err := GenerateCredentialSecret()
	if err != nil {
		t.Fatalf("GenerateCredentialSecret: %v", err)
	}
	b, err := GenerateCredentialSecret()
	if err != nil {
		t.Fatalf("GenerateCredentialSecret: %v", err)
	}
	if a == b {
		t.Fatal("two generated secrets are identical")
	}
	// base64.RawURLEncoding: 4 chars per 3 bytes, no padding.
	if len(a) < 42 {
		t.Fatalf("generated secret is %d chars, shorter than 32 bytes of base64", len(a))
	}

	// And the generated value actually issues a working credential.
	opts := issueOpts()
	opts.Secret = ""
	_, secret, err := IssueRecord(testVerifierKey, opts)
	if err != nil {
		t.Fatalf("IssueRecord with a generated secret: %v", err)
	}
	if secret == "" {
		t.Fatal("issuance must return the generated secret exactly once")
	}
}

// TestRegistryFileRoundTripIsServerLoadable writes a registry with the issuer
// and loads it through LoadFromSources — the same entry point the server uses —
// so a file this tool writes cannot differ from a file the server accepts.
func TestRegistryFileRoundTripIsServerLoadable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")

	rec, secret, err := IssueRecord(testVerifierKey, issueOpts())
	if err != nil {
		t.Fatalf("IssueRecord: %v", err)
	}
	if err := WriteRegistryFile(path, testVerifierKey, []Record{rec}); err != nil {
		t.Fatalf("WriteRegistryFile: %v", err)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("registry file mode = %#o, want 0600", info.Mode().Perm())
		}
	}

	reg, err := LoadFromSources("", path, testVerifierKey)
	if err != nil {
		t.Fatalf("the written registry must load through the server path: %v", err)
	}
	if _, ok := reg.Resolve(secret); !ok {
		t.Fatal("the credential written to the file must resolve from the loaded registry")
	}

	// No temp file may survive the write.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".credential-registry-") {
			t.Fatalf("temp file survived the write: %s", e.Name())
		}
	}
}

// TestWriteRegistryFileRefusesInvalidSet proves the whole-set validation is the
// same one the loader runs: a set containing any invalid record is refused
// BEFORE the file is touched, so a failed append cannot replace a live
// registry with something the server will reject at the next restart.
func TestWriteRegistryFileRefusesInvalidSet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")

	good, _, err := IssueRecord(testVerifierKey, issueOpts())
	if err != nil {
		t.Fatalf("IssueRecord: %v", err)
	}
	if err := WriteRegistryFile(path, testVerifierKey, []Record{good}); err != nil {
		t.Fatalf("initial write: %v", err)
	}

	bad := good
	bad.CredentialID = "cred-bad"
	bad.Verifier = "not-hex"
	if err := WriteRegistryFile(path, testVerifierKey, []Record{good, bad}); err == nil {
		t.Fatal("a set containing an invalid record must be refused")
	}

	// The previous file is intact and still loads.
	reg, err := LoadFromSources("", path, testVerifierKey)
	if err != nil {
		t.Fatalf("the previous registry must survive a refused write: %v", err)
	}
	if !reg.Enabled() {
		t.Fatal("the previous registry must still be enabled")
	}

	// An empty set is refused: the loader treats an empty source as
	// "no registry configured" and falls back to the legacy surface.
	if err := WriteRegistryFile(path, testVerifierKey, nil); err == nil {
		t.Fatal("writing an empty registry must be refused")
	}
}

// TestWriteRegistryFileRefusesSymlink proves the write cannot be redirected
// between validation and rename.
func TestWriteRegistryFileRefusesSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "real.json")
	link := filepath.Join(dir, "link.json")
	if err := os.WriteFile(target, []byte("[]\n"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	rec, _, err := IssueRecord(testVerifierKey, issueOpts())
	if err != nil {
		t.Fatalf("IssueRecord: %v", err)
	}
	if err := WriteRegistryFile(link, testVerifierKey, []Record{rec}); err == nil {
		t.Fatal("writing through a symlink must be refused")
	}
}

// TestRevokeInFileDeniesOnReload is the operator revocation path end to end:
// the file is rewritten, the registry reloads the new source, and the credential
// denies well inside the published bound. It also proves an unknown credential
// id is an error rather than a silent success.
func TestRevokeInFileDeniesOnReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")

	keep, keepSecret, err := IssueRecord(testVerifierKey, issueOpts())
	if err != nil {
		t.Fatalf("IssueRecord: %v", err)
	}
	opts := issueOpts()
	opts.CredentialID = "cred-issued-2"
	opts.PrincipalID = "principal-issued-2"
	opts.Secret = "second-secret-value-for-tests"
	doomed, doomedSecret, err := IssueRecord(testVerifierKey, opts)
	if err != nil {
		t.Fatalf("IssueRecord: %v", err)
	}
	if err := WriteRegistryFile(path, testVerifierKey, []Record{keep, doomed}); err != nil {
		t.Fatalf("WriteRegistryFile: %v", err)
	}

	reg, err := LoadFromSources("", path, testVerifierKey)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if _, ok := reg.Resolve(doomedSecret); !ok {
		t.Fatal("credential must resolve before revocation")
	}

	start := time.Now()
	rec, err := RevokeInFile(path, testVerifierKey, doomed.CredentialID)
	if err != nil {
		t.Fatalf("RevokeInFile: %v", err)
	}
	if !rec.Revoked {
		t.Fatal("RevokeInFile must report the record as revoked")
	}
	if err := reg.ReloadFromSources("", path, testVerifierKey); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if _, ok := reg.Resolve(doomedSecret); ok {
		t.Fatal("revoked credential still resolves after reload")
	}
	if _, ok := reg.Resolve(keepSecret); !ok {
		t.Fatal("an unrelated credential must survive the revocation")
	}
	if elapsed := time.Since(start); elapsed > RevocationPropagationBound {
		t.Fatalf("revocation took %v, exceeding the published bound %v", elapsed, RevocationPropagationBound)
	}

	if _, err := RevokeInFile(path, testVerifierKey, "cred-does-not-exist"); err == nil {
		t.Fatal("revoking an unknown credential id must be an error, not a silent no-op")
	}
}

// TestLoadRegistryFileRefusesLooseMode proves the issuer reads through the same
// restricted-file rule as the server: an operator learns about a group/world
// readable registry here, not at the next restart.
func TestLoadRegistryFileRefusesLooseMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX mode check")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")
	rec, _, err := IssueRecord(testVerifierKey, issueOpts())
	if err != nil {
		t.Fatalf("IssueRecord: %v", err)
	}
	body, err := json.Marshal([]Record{rec})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	if _, err := LoadRegistryFile(path); err == nil {
		t.Fatal("a group/world readable registry must be refused")
	}
}
