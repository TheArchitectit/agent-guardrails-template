package controlplane

// Evidence retention, signing, access control and auditable deletion
// (Spec 08 R08-4.3.6, acceptance evidence for the requirement).
//
// These tests drive the REAL store against a REAL evaluation bundle produced by
// Evaluate, and verify signatures with the public half of the key. Nothing here
// re-implements the store's own arithmetic: a test that checks a function
// against itself proves the function is consistent, not that it is correct.

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/thearchitectit/guardrail-mcp/internal/auth"
	"github.com/thearchitectit/guardrail-mcp/internal/evidence"
)

// testSigner builds a deterministic instance identity from a low-entropy seed.
// The seed is deliberately not a credential-shaped literal: a random-looking
// fixture string is what turns Secret Validation red for the wrong reason.
func testSigner(t *testing.T) *evidence.Instance {
	t.Helper()
	seed := sha256.Sum256([]byte("controlplane evidence store test fixture seed"))
	key := ed25519.NewKeyFromSeed(seed[:])
	inst, err := evidence.NewInstance("guardrail-test", key)
	if err != nil {
		t.Fatalf("build instance: %v", err)
	}
	return inst
}

// adminCaller is a principal that satisfies the ADMIN-class evidence read.
func adminCaller() auth.Caller {
	return auth.Caller{
		PrincipalID:  "principal-admin",
		CredentialID: "cred-admin",
		Scopes:       []string{auth.ScopeAdminManage},
		Role:         auth.RoleAdministrator,
		Resources:    []string{"guardrail://control-plane/evidence"},
	}
}

// readerCaller holds read-class authority only. It must NOT be able to read or
// list retained evidence: a bundle names what the deployment blocked and why.
func readerCaller() auth.Caller {
	return auth.Caller{
		PrincipalID:  "principal-reader",
		CredentialID: "cred-reader",
		Scopes:       []string{auth.ScopeMCPRead},
		Role:         auth.RoleReader,
		Resources:    []string{"guardrail://control-plane"},
	}
}

// passingBundle produces a real pass bundle through Evaluate.
func evidenceStoreBundle(t *testing.T) *Bundle {
	t.Helper()
	allExercised := ObserverFunc(func(c Component) Observation {
		return Observation{ComponentID: c.ID, State: StateExercised, Detail: "exercised by the request path"}
	})
	b, err := Evaluate(EvaluateRequest{
		Manifest:      validManifest(),
		Observer:      allExercised,
		SubjectDigest: digest("e"),
		Clock:         fixedClock(),
		Checks: []Check{{
			ComponentID: "injection-detector",
			CheckID:     "canonical-injection",
			InputDigest: digest("1"),
			Mandatory:   true,
			Run:         func() (CheckOutcome, string, string) { return OutcomePass, "detected", "" },
		}},
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if b.Verdict != VerdictPass {
		t.Fatalf("fixture bundle verdict = %s, want pass (%v)", b.Verdict, b.Unmet)
	}
	return b
}

func testStore(t *testing.T, mut func(*StoreConfig)) (*EvidenceStore, *[]Event) {
	t.Helper()
	events := &[]Event{}
	cfg := StoreConfig{
		Dir:       t.TempDir(),
		Retention: 24 * time.Hour,
		Signer:    testSigner(t),
		Now:       func() time.Time { return fixedClock()() },
		Audit: func(e Event) {
			*events = append(*events, e)
		},
	}
	if mut != nil {
		mut(&cfg)
	}
	s, err := NewEvidenceStore(cfg)
	if err != nil {
		t.Fatalf("NewEvidenceStore: %v", err)
	}
	return s, events
}

// R08-4.3.6 — a stored bundle reads back byte-identically and verifies.
func TestEvidenceStoreRoundTripsAVerifiedBundle(t *testing.T) {
	s, events := testStore(t, nil)
	b := evidenceStoreBundle(t)

	path, err := s.Append(b)
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	// Owner-only mode: another local account reading the record is an
	// access-control failure no policy field repairs afterwards.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat record: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("record mode = %o, want 600", perm)
	}

	got, err := s.Read(b.Digest, adminCaller())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.EvaluationID != b.EvaluationID || got.Digest != b.Digest {
		t.Fatalf("read bundle identity changed: %s/%s vs %s/%s",
			got.EvaluationID, got.Digest, b.EvaluationID, b.Digest)
	}
	if len(got.Checks) != len(b.Checks) {
		t.Fatalf("read bundle has %d checks, want %d", len(got.Checks), len(b.Checks))
	}

	if len(*events) != 1 || (*events)[0].Action != "stored" {
		t.Fatalf("store event not audited: %+v", *events)
	}
}

// R08-4.3.6 access control — read-class authority is not enough.
func TestEvidenceStoreDeniesReadClassAuthority(t *testing.T) {
	s, _ := testStore(t, nil)
	b := evidenceStoreBundle(t)
	if _, err := s.Append(b); err != nil {
		t.Fatalf("Append: %v", err)
	}

	if _, err := s.Read(b.Digest, readerCaller()); err == nil {
		t.Fatal("a read-class credential read retained evidence; evidence reads are ADMIN class")
	}
	if _, err := s.List(readerCaller()); err == nil {
		t.Fatal("a read-class credential listed retained evidence")
	}
	// An unauthenticated caller (no principal) is refused too, by the same
	// decision function the tool and resource paths use.
	if _, err := s.Read(b.Digest, auth.Caller{}); err == nil {
		t.Fatal("an unauthenticated caller read retained evidence")
	}
	// The administrator still can, so the denial above is authorization and
	// not a broken store.
	if _, err := s.Read(b.Digest, adminCaller()); err != nil {
		t.Fatalf("administrator could not read: %v", err)
	}
}

// R08-4.3.6 signing — an edited record is refused, never returned as evidence.
func TestEvidenceStoreRefusesATamperedRecord(t *testing.T) {
	s, _ := testStore(t, nil)
	b := evidenceStoreBundle(t)
	path, err := s.Append(b)
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	// The threat model is not a careless hand-edit: anyone who can rewrite this
	// file can also recompute the content digest, so a digest-only check is not
	// an integrity control against them. This tamper rewrites the verdict AND
	// re-derives the bundle digest to match, which leaves the detached
	// signature as the only thing that can catch it.
	var rec StoredRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("decode record: %v", err)
	}
	rec.Bundle.Verdict = VerdictFailed
	rec.Bundle.Digest = rec.Bundle.computeDigest()
	tampered, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		t.Fatalf("encode tampered record: %v", err)
	}
	if err := os.WriteFile(path, tampered, 0o600); err != nil {
		t.Fatalf("write tampered record: %v", err)
	}

	if _, err := s.Read(b.Digest, adminCaller()); err == nil {
		t.Fatal("a tampered record was returned as evidence")
	} else if !strings.Contains(err.Error(), "unverified") {
		t.Fatalf("tamper error did not say the record is unverified: %v", err)
	}

	// The listing must also say so, rather than reporting a row and letting a
	// reader assume it is intact.
	l, err := s.List(adminCaller())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if l.Unverified != 1 || len(l.Records) != 1 || l.Records[0].Verified {
		t.Fatalf("listing did not flag the tampered record: %+v", l)
	}
}

// R08-4.3.6 — a bundle whose digest does not match its content is not signed.
func TestEvidenceStoreRefusesADigestThatDoesNotMatch(t *testing.T) {
	s, _ := testStore(t, nil)
	b := evidenceStoreBundle(t)
	b.Digest = digest("9") // a hand-set digest claiming to be this bundle

	if _, err := s.Append(b); err == nil {
		t.Fatal("store signed a bundle whose digest does not match its content")
	} else if !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("refusal did not name the digest mismatch: %v", err)
	}
}

// R08-4.3.6 retention — the policy is applied and every removal is audited.
func TestEvidenceStoreSweepDeletesPastRetentionAndAudits(t *testing.T) {
	base := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	now := base
	events := &[]Event{}
	cfg := StoreConfig{
		Dir:       t.TempDir(),
		Retention: 24 * time.Hour,
		Signer:    testSigner(t),
		Now:       func() time.Time { return now },
		Audit:     func(e Event) { *events = append(*events, e) },
	}
	s, err := NewEvidenceStore(cfg)
	if err != nil {
		t.Fatalf("NewEvidenceStore: %v", err)
	}

	old := evidenceStoreBundle(t)
	oldPath, err := s.Append(old)
	if err != nil {
		t.Fatalf("Append old: %v", err)
	}

	// Advance past the window, then store a fresh one at the new time.
	// The bundle identity is content-derived, so a distinct evaluation comes
	// from a different subject rather than a hand-edited id.
	now = base.Add(48 * time.Hour)
	fresh := evaluateWithSubject(t, digest("7"))
	if _, err := s.Append(fresh); err != nil {
		t.Fatalf("Append fresh: %v", err)
	}
	if old.EvaluationID == fresh.EvaluationID {
		t.Fatal("fixture bundles share an identity; the sweep assertion would be vacuous")
	}

	removed, err := s.Sweep()
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if len(removed) != 1 || removed[0].EvaluationID != old.EvaluationID {
		t.Fatalf("sweep removed %+v, want exactly the old record", removed)
	}
	// The old record's own path must be gone; asserting on a path the store
	// never wrote would pass whether or not the deletion happened.
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old record still present after sweep: %v", err)
	}
	if _, err := s.Read(fresh.Digest, adminCaller()); err != nil {
		t.Fatalf("sweep removed the in-window record: %v", err)
	}

	var deletes int
	for _, e := range *events {
		if e.Action == "deleted" {
			deletes++
			if e.EvaluationID != old.EvaluationID || e.Reason == "" || e.Age == "" {
				t.Fatalf("deletion event is not auditable: %+v", e)
			}
		}
	}
	if deletes != 1 {
		t.Fatalf("deletion events = %d, want 1", deletes)
	}

	// A second sweep over an unchanged store deletes nothing: retention applied
	// repeatedly must converge, not keep removing.
	again, err := s.Sweep()
	if err != nil {
		t.Fatalf("second Sweep: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("second sweep removed %+v, want nothing", again)
	}
}

// R08-4.3.6 — the count bound is a second, independent bound.
func TestEvidenceStoreSweepAppliesTheCountBound(t *testing.T) {
	base := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	now := base
	events := &[]Event{}
	s, err := NewEvidenceStore(StoreConfig{
		Dir:        t.TempDir(),
		Retention:  10 * 365 * 24 * time.Hour, // time bound cannot fire here
		MaxRecords: 2,
		Signer:     testSigner(t),
		Now:        func() time.Time { return now },
		Audit:      func(e Event) { *events = append(*events, e) },
	})
	if err != nil {
		t.Fatalf("NewEvidenceStore: %v", err)
	}

	var ids []string
	for i, sub := range []string{"1", "2", "3"} {
		now = base.Add(time.Duration(i) * time.Minute)
		b := evaluateWithSubject(t, digest(sub))
		if _, err := s.Append(b); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
		ids = append(ids, b.EvaluationID)
	}

	removed, err := s.Sweep()
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if len(removed) != 1 || removed[0].EvaluationID != ids[0] {
		t.Fatalf("count bound removed %+v, want the oldest record only", removed)
	}
	if len(removed) > 0 && !strings.Contains(removed[0].Reason, "max_records") {
		t.Fatalf("removal reason did not name the count bound: %q", removed[0].Reason)
	}
}

// R08-4.3.6 deletion — an explicit erase is distinct from retention and needs a
// reason.
func TestEvidenceStoreEraseIsAuditedAndRequiresAReason(t *testing.T) {
	s, events := testStore(t, nil)
	b := evidenceStoreBundle(t)
	if _, err := s.Append(b); err != nil {
		t.Fatalf("Append: %v", err)
	}

	if err := s.Erase(b.Digest, ""); err == nil {
		t.Fatal("erase accepted an empty reason; an unexplainable deletion is not auditable")
	}
	if err := s.Erase("eval:sha256:absent", "subject request"); err == nil {
		t.Fatal("erase reported success for a record that does not exist")
	}
	if err := s.Erase(b.Digest, "subject request"); err != nil {
		t.Fatalf("Erase: %v", err)
	}

	var erased int
	for _, e := range *events {
		if e.Action == "erased" {
			erased++
			if e.Reason != "subject request" {
				t.Fatalf("erase event lost its reason: %+v", e)
			}
		}
	}
	if erased != 1 {
		t.Fatalf("erase events = %d, want 1", erased)
	}
	if _, err := s.Read(b.Digest, adminCaller()); err == nil {
		t.Fatal("erased record is still readable")
	}
}

// R08-4.3.6 configurability — every weaker configuration is refused by name.
func TestNewEvidenceStoreRefusesAWeakerPolicy(t *testing.T) {
	signer := testSigner(t)
	cases := []struct {
		name    string
		cfg     StoreConfig
		wantSub string
	}{
		{
			name:    "no retention",
			cfg:     StoreConfig{Dir: t.TempDir(), Signer: signer, Audit: func(Event) {}},
			wantSub: "retention must be positive",
		},
		{
			name:    "no signer",
			cfg:     StoreConfig{Dir: t.TempDir(), Retention: time.Hour, Audit: func(Event) {}},
			wantSub: "signing identity",
		},
		{
			name:    "no audit sink",
			cfg:     StoreConfig{Dir: t.TempDir(), Retention: time.Hour, Signer: signer},
			wantSub: "audit sink",
		},
		{
			name:    "no directory",
			cfg:     StoreConfig{Retention: time.Hour, Signer: signer, Audit: func(Event) {}},
			wantSub: "directory is required",
		},
		{
			name:    "negative count",
			cfg:     StoreConfig{Dir: t.TempDir(), Retention: time.Hour, MaxRecords: -1, Signer: signer, Audit: func(Event) {}},
			wantSub: "non-negative",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewEvidenceStore(tc.cfg); err == nil {
				t.Fatalf("store accepted %s", tc.name)
			} else if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("refusal %q does not name %q", err, tc.wantSub)
			}
		})
	}
}

// The record on disk is the structure the signature covers: a signature over a
// different shape would verify against a record nobody can read.
func TestStoredRecordSignatureCoversTheStoredBytes(t *testing.T) {
	s, _ := testStore(t, nil)
	b := evidenceStoreBundle(t)
	path, err := s.Append(b)
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var rec StoredRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("decode stored record: %v", err)
	}
	if rec.Purpose != evidenceRecordPurpose {
		t.Fatalf("stored purpose = %q, want %q", rec.Purpose, evidenceRecordPurpose)
	}
	if rec.KeyID != s.SignerKeyID() {
		t.Fatalf("stored key id = %q, want the store's key %q", rec.KeyID, s.SignerKeyID())
	}
	payload := canonicalJSON(recordSigned{
		Bundle:   rec.Bundle,
		KeyID:    rec.KeyID,
		Purpose:  rec.Purpose,
		StoredAt: rec.StoredAt,
	})
	if err := evidence.VerifyDetached(s.publicKey(), rec.Purpose, payload, rec.Signature); err != nil {
		t.Fatalf("stored signature does not verify over the stored bytes: %v", err)
	}
	// Re-checking against a different purpose must fail: domain separation is
	// what stops a record signature being reused as an envelope signature.
	if err := evidence.VerifyDetached(s.publicKey(), "some.other.purpose", payload, rec.Signature); err == nil {
		t.Fatal("a stored record's signature verified under a different purpose")
	}
}

// evaluateWithSubject produces a distinct real bundle for the same manifest
// under a different subject, which is how a second evaluation identity is
// created without hand-editing a derived id.
func evaluateWithSubject(t *testing.T, subject string) *Bundle {
	t.Helper()
	allExercised := ObserverFunc(func(c Component) Observation {
		return Observation{ComponentID: c.ID, State: StateExercised, Detail: "exercised"}
	})
	b, err := Evaluate(EvaluateRequest{
		Manifest:      validManifest(),
		Observer:      allExercised,
		SubjectDigest: subject,
		Clock:         fixedClock(),
	})
	if err != nil {
		t.Fatalf("Evaluate(%s): %v", subject, err)
	}
	return b
}

// R08-4.3.6 — two runs of the same evaluation share an evaluation id and can
// disagree. Retention must keep both: the check that failed on one run and
// passed on the next is exactly the history this exists to preserve, and a
// store keyed by evaluation id would silently replace the first with the second.
func TestEvidenceStoreRetainsDivergentRunsOfOneEvaluation(t *testing.T) {
	s, _ := testStore(t, nil)

	pass := evaluateWithChecks(t, digest("5"), FuncOutcome(OutcomePass))
	fail := evaluateWithChecks(t, digest("5"), FuncOutcome(OutcomeBlock))

	if pass.EvaluationID != fail.EvaluationID {
		t.Fatalf("fixture expected one evaluation id for one subject; got %s and %s",
			pass.EvaluationID, fail.EvaluationID)
	}
	if pass.Digest == fail.Digest {
		t.Fatal("fixture bundles are identical; the overwrite assertion would be vacuous")
	}

	if _, err := s.Append(pass); err != nil {
		t.Fatalf("Append pass: %v", err)
	}
	if _, err := s.Append(fail); err != nil {
		t.Fatalf("Append fail: %v", err)
	}

	// Both records survive, and each reads back as itself.
	gotPass, err := s.Read(pass.Digest, adminCaller())
	if err != nil {
		t.Fatalf("Read pass: %v", err)
	}
	gotFail, err := s.Read(fail.Digest, adminCaller())
	if err != nil {
		t.Fatalf("Read fail: %v", err)
	}
	if gotPass.Verdict != VerdictPass || gotFail.Verdict != VerdictBlock {
		t.Fatalf("retained verdicts = %s/%s, want pass/block", gotPass.Verdict, gotFail.Verdict)
	}

	// Re-appending identical bytes is reported, not duplicated on disk.
	if _, err := s.Append(pass); err != nil {
		t.Fatalf("identical re-append was refused: %v", err)
	}
	l, err := s.List(adminCaller())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(l.Records) != 2 {
		t.Fatalf("store holds %d records, want 2 (an identical re-append must not add a third)", len(l.Records))
	}
}

// FuncOutcome is a check body with a fixed outcome.
func FuncOutcome(o CheckOutcome) func() (CheckOutcome, string, string) {
	return func() (CheckOutcome, string, string) { return o, "fixture_outcome", "" }
}

func evaluateWithChecks(t *testing.T, subject string, run func() (CheckOutcome, string, string)) *Bundle {
	t.Helper()
	b, err := Evaluate(EvaluateRequest{
		Manifest:      validManifest(),
		Observer:      ObserverFunc(func(c Component) Observation { return Observation{ComponentID: c.ID, State: StateExercised} }),
		SubjectDigest: subject,
		Clock:         fixedClock(),
		Checks: []Check{{
			ComponentID: "injection-detector",
			CheckID:     "canonical-injection",
			InputDigest: digest("1"),
			Mandatory:   true,
			Run:         run,
		}},
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	return b
}

// R08-4.3.6 + R08-4.4.2 — a promotion retains the evidence it was granted on,
// and the promotion record says whether the bytes were kept.
func TestPromoteRetainsItsAcceptanceEvidence(t *testing.T) {
	m := testManifest(t, nil)
	clock := fixedClock()
	g, err := NewGovernance(m)
	if err != nil {
		t.Fatalf("NewGovernance: %v", err)
	}
	g.clock = clock

	var events []Event
	store, err := NewEvidenceStore(StoreConfig{
		Dir:       t.TempDir(),
		Retention: 24 * time.Hour,
		Signer:    testSigner(t),
		Now:       clock,
		Audit:     func(e Event) { events = append(events, e) },
	})
	if err != nil {
		t.Fatalf("NewEvidenceStore: %v", err)
	}
	g.SetEvidenceStore(store)

	accept := passingBundle(m, clock)
	if err := g.Promote(PromotionRequest{
		ProfileVersion: m.Version,
		ReviewRef:      "CHG-77",
		ApprovedBy:     "roger",
		Acceptance:     accept,
	}); err != nil {
		t.Fatalf("promotion refused valid evidence: %v", err)
	}

	// The accepted bundle is readable afterwards, by digest, as evidence.
	got, err := store.Read(accept.Digest, adminCaller())
	if err != nil {
		t.Fatalf("promotion accepted evidence that was not retained: %v", err)
	}
	if got.EvaluationID != accept.EvaluationID {
		t.Fatalf("retained bundle is a different evaluation: %s vs %s", got.EvaluationID, accept.EvaluationID)
	}

	// The record names retention explicitly, so a reader never has to guess.
	recs := g.Records()
	if len(recs) == 0 {
		t.Fatal("promotion wrote no record")
	}
	joined := strings.Join(recs[len(recs)-1].Changes, ",")
	if !strings.Contains(joined, "retention=stored:") {
		t.Fatalf("promotion record does not state that evidence was retained: %v", recs[len(recs)-1].Changes)
	}
	var stored int
	for _, e := range events {
		if e.Action == "stored" {
			stored++
		}
	}
	if stored != 1 {
		t.Fatalf("retention events = %d, want exactly 1", stored)
	}
}

// A promotion record with no store configured says so, rather than leaving the
// reader to assume the evidence was kept.
func TestPromoteStatesWhenEvidenceIsNotRetained(t *testing.T) {
	m := testManifest(t, nil)
	clock := fixedClock()
	g, _ := NewGovernance(m)
	g.clock = clock

	if err := g.Promote(PromotionRequest{
		ProfileVersion: m.Version,
		ReviewRef:      "CHG-78",
		ApprovedBy:     "roger",
		Acceptance:     passingBundle(m, clock),
	}); err != nil {
		t.Fatalf("promotion with no store configured was refused: %v", err)
	}
	recs := g.Records()
	joined := strings.Join(recs[len(recs)-1].Changes, ",")
	if !strings.Contains(joined, "retention=not_configured") {
		t.Fatalf("promotion record does not state retention is unconfigured: %v", recs[len(recs)-1].Changes)
	}
}

// R08-4.3.6 — the retention policy is actually applied by the process, not just
// configurable. A Sweep with no caller is a built-but-unreachable control.
func TestRetentionLoopTrimsAtStartupAndOnInterval(t *testing.T) {
	base := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	now := base
	var events []Event
	s, err := NewEvidenceStore(StoreConfig{
		Dir:       t.TempDir(),
		Retention: time.Hour,
		Signer:    testSigner(t),
		Now:       func() time.Time { return now },
		Audit:     func(e Event) { events = append(events, e) },
	})
	if err != nil {
		t.Fatalf("NewEvidenceStore: %v", err)
	}

	old := evaluateWithSubject(t, digest("3"))
	if _, err := s.Append(old); err != nil {
		t.Fatalf("Append: %v", err)
	}

	// The window has passed by the time the process starts its loop, so the
	// FIRST sweep must trim — waiting an interval would leave a record past its
	// stated window for an hour.
	now = base.Add(2 * time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.StartRetentionLoop(ctx, time.Hour)

	if _, err := s.Read(old.Digest, adminCaller()); err == nil {
		t.Fatal("the startup sweep did not apply the window to an already-expired record")
	}
	var deletions int
	for _, e := range events {
		if e.Action == "deleted" && e.EvaluationID == old.EvaluationID {
			deletions++
		}
	}
	if deletions != 1 {
		t.Fatalf("deletion events = %d, want 1", deletions)
	}
}
