package controlplane

// Evidence persistence, retention, access control and auditable deletion
// (Spec 08 R08-4.3.6).
//
// R08-4.3.6 says: "Evidence retention, signing, access control, and deletion
// policy MUST be configurable and auditable."
//
// Before this file the bundle was produced, returned to the caller and then
// gone. That satisfies "an evaluation happened" and fails the requirement that
// an operator can answer, months later, what the composition proved — and it
// leaves no artifact against which a replay claim could be checked.
//
// Four rules shape the implementation, and each exists because the obvious
// weaker version is a claim without a mechanism:
//
//  1. Configurable means explicit. An unset retention is an error, not
//     "keep forever": "forever" is a policy nobody chose, and a store that
//     silently keeps every bundle grows without bound on a box whose disk is a
//     production dependency.
//  2. Signed means verifiable. Each record is written with a detached
//     signature over the exact stored bytes, under a stated purpose. A record
//     whose signature does not verify is reported as `unverified`, never
//     silently returned as evidence — an edited record that reads like an
//     original is worse than a missing one.
//  3. Access control reuses the same principal model as tools and resources
//     (Spec 16 R16-04). Evidence reads are ADMIN-class: a bundle names every
//     component, every check and the reasons a control blocked. Read-class
//     authority is for the guardrail documents, not for the record of what a
//     deployment actually did.
//  4. Deletion is auditable or it does not happen. Every removal — retention
//     sweep or explicit erase — emits a record naming the evaluation id, the
//     reason and the age. A silent unlink is indistinguishable from a record
//     that was never written.

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/thearchitectit/guardrail-mcp/internal/auth"
	"github.com/thearchitectit/guardrail-mcp/internal/evidence"
)

// evidenceRecordPurpose binds a stored record's signature to stored evidence.
// A signature that does not name what it covers can be moved to another
// artifact carrying the same bytes.
const evidenceRecordPurpose = "controlplane.evidence.record.v1"

// StoreConfig is the configured retention and identity policy for retained
// evidence. Every field is required unless stated; there is no implicit
// default, because a default here is a policy the operator did not choose.
type StoreConfig struct {
	// Dir is where records are written. It is created at 0700 when missing.
	Dir string
	// Retention is how long a record is kept. It must be positive.
	Retention time.Duration
	// MaxRecords bounds the store independently of time. Zero means "no count
	// bound" and is permitted only when Retention is set: time alone still
	// bounds the store, and refusing both would make a small deployment
	// unable to retain anything.
	MaxRecords int
	// Signer supplies the instance identity. A store configured without a
	// signer is refused: an unsigned record cannot be distinguished from an
	// edited one, which defeats the requirement.
	Signer *evidence.Instance
	// Now is injectable for deterministic tests.
	Now func() time.Time
	// Audit receives deletion and refusal events. A nil Audit is refused: a
	// deletion policy that cannot be audited is the thing R08-4.3.6 forbids.
	Audit func(Event)
}

// Event is one auditable store action. It carries no bundle content — only the
// identity of the record and why something happened.
type Event struct {
	Action       string    `json:"action"` // "stored" | "deleted" | "erased" | "refused"
	EvaluationID string    `json:"evaluation_id,omitempty"`
	Reason       string    `json:"reason"`
	Age          string    `json:"age,omitempty"`
	At           time.Time `json:"at"`
	Count        int       `json:"count,omitempty"`
}

// StoredRecord is the on-disk shape. The signature covers RecordBytes, which
// is the canonical JSON of every other field, so a tampered bundle, a swapped
// evaluation id or an edited stored_at all fail verification.
type StoredRecord struct {
	Bundle    *Bundle   `json:"bundle"`
	KeyID     string    `json:"key_id"`
	Purpose   string    `json:"purpose"`
	StoredAt  time.Time `json:"stored_at"`
	Signature string    `json:"signature"`
}

// recordSigned is the exact structure whose canonical bytes carry the
// signature. It deliberately excludes the signature itself.
type recordSigned struct {
	Bundle   *Bundle   `json:"bundle"`
	KeyID    string    `json:"key_id"`
	Purpose  string    `json:"purpose"`
	StoredAt time.Time `json:"stored_at"`
}

// ErrNoSigner is returned when a store is built without an identity.
var ErrNoSigner = errors.New("evidence store requires a signing identity")

// ErrNoAudit is returned when a store is built without an audit sink.
var ErrNoAudit = errors.New("evidence store requires an audit sink")

// errStoreNotConfigured and errStorePermissionDenied are the two stable
// failures List and Read share. They are deliberately non-leaking: a caller
// that is refused learns that it was refused, not whether a record exists.
var (
	errStoreNotConfigured    = errors.New("evidence store is not configured")
	errStorePermissionDenied = errors.New("permission denied")
)

// EvidenceStore persists evaluation bundles under a configured policy.
type EvidenceStore struct {
	cfg StoreConfig
	now func() time.Time
}

// NewEvidenceStore validates the policy and returns a store. Every refusal
// names the field, because a store that silently runs with a weaker policy than
// the operator configured is the failure mode this requirement is about.
func NewEvidenceStore(cfg StoreConfig) (*EvidenceStore, error) {
	if strings.TrimSpace(cfg.Dir) == "" {
		return nil, errors.New("evidence store directory is required")
	}
	if cfg.Retention <= 0 {
		return nil, fmt.Errorf("evidence retention must be positive, got %s", cfg.Retention)
	}
	if cfg.MaxRecords < 0 {
		return nil, fmt.Errorf("evidence max_records must be non-negative, got %d", cfg.MaxRecords)
	}
	if cfg.Signer == nil {
		return nil, ErrNoSigner
	}
	if cfg.Audit == nil {
		return nil, ErrNoAudit
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	if err := os.MkdirAll(cfg.Dir, 0o700); err != nil {
		return nil, fmt.Errorf("create evidence store directory: %w", err)
	}
	return &EvidenceStore{cfg: cfg, now: now}, nil
}

// SignerKeyID is the public identifier of the signing identity, exposed so the
// process can report which key its stored evidence will verify under.
func (s *EvidenceStore) SignerKeyID() string {
	if s == nil || s.cfg.Signer == nil {
		return ""
	}
	return s.cfg.Signer.KeyID()
}

// Append seals and writes one bundle.
//
// A bundle whose Digest does not match its content is refused. The digest is
// recomputed here rather than trusted: a caller that hands over a bundle with a
// hand-set digest would otherwise have its claim signed by this process, which
// is the "evidence that proves the wrong thing" defect the spec names.
func (s *EvidenceStore) Append(b *Bundle) (string, error) {
	if s == nil {
		return "", errors.New("evidence store is not configured")
	}
	if b == nil {
		return "", errors.New("refusing to store a nil bundle")
	}
	if b.EvaluationID == "" {
		return "", errors.New("refusing to store a bundle with no evaluation id")
	}
	if got := b.computeDigest(); got != b.Digest {
		return "", fmt.Errorf("refusing to store bundle %s: recorded digest %s does not match its content (%s)",
			b.EvaluationID, b.Digest, got)
	}

	rec := recordSigned{
		Bundle:   b,
		KeyID:    s.cfg.Signer.KeyID(),
		Purpose:  evidenceRecordPurpose,
		StoredAt: s.now().UTC(),
	}
	payload := canonicalJSON(rec)
	sig, err := s.cfg.Signer.SignDetached(evidenceRecordPurpose, payload)
	if err != nil {
		return "", fmt.Errorf("sign evidence record: %w", err)
	}
	stored := StoredRecord{
		Bundle:    b,
		KeyID:     rec.KeyID,
		Purpose:   rec.Purpose,
		StoredAt:  rec.StoredAt,
		Signature: sig,
	}
	body, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode evidence record: %w", err)
	}

	// A record's storage identity is its CONTENT digest, not its evaluation id.
	// Two evaluations of the same subject under the same manifest share an
	// evaluation id (that is what makes a replay comparison meaningful) and can
	// still differ: a check that failed on one run and passed on the next is
	// exactly the history retention exists to preserve. Keying by evaluation id
	// would let the second run silently overwrite the first — evidence that
	// reports the present as if it were the past.
	path := s.recordPath(b.Digest)
	if existing, err := os.ReadFile(path); err == nil {
		if string(existing) == string(body) {
			// Byte-identical re-append: the record is already retained. It is not
			// an error and it is not a second write; reporting it as stored keeps
			// the audit trail honest about what happened.
			s.cfg.Audit(Event{
				Action:       "stored",
				EvaluationID: b.EvaluationID,
				Reason:       "identical record already retained",
				At:           rec.StoredAt,
			})
			return path, nil
		}
		return "", fmt.Errorf("refusing to overwrite retained evidence for evaluation %s: a record with the same digest exists and differs in content", b.EvaluationID)
	}
	if err := writeFile0600(path, body); err != nil {
		return "", err
	}
	s.cfg.Audit(Event{
		Action:       "stored",
		EvaluationID: b.EvaluationID,
		Reason:       "evaluation bundle retained",
		At:           rec.StoredAt,
	})
	return path, nil
}

// Read returns one stored record by its CONTENT digest, after verifying the
// signature and after the caller is authorized for an ADMIN-class read.
//
// A record that fails verification returns an error, not the record: returning
// the bytes with a flag would make every caller responsible for checking the
// flag, and the one that forgets is the one whose report is wrong.
func (s *EvidenceStore) Read(recordDigest string, caller auth.Caller) (*Bundle, error) {
	if s == nil {
		return nil, errStoreNotConfigured
	}
	if !s.authorize(caller) {
		return nil, errStorePermissionDenied
	}
	rec, err := s.loadVerified(recordDigest)
	if err != nil {
		return nil, err
	}
	return rec.Bundle, nil
}

// ReadLatestFor returns the most recent retained record for an evaluation id.
//
// Two evaluations of the same subject share an evaluation id and can differ (a
// check that failed on one run and passed on the next). "The latest" is a
// deliberate, named choice; the alternative — an implicit pick — makes a
// reader believe they are looking at one run while seeing another.
func (s *EvidenceStore) ReadLatestFor(evaluationID string, caller auth.Caller) (*Bundle, error) {
	if s == nil {
		return nil, errStoreNotConfigured
	}
	if !s.authorize(caller) {
		return nil, errStorePermissionDenied
	}
	var best *StoredRecord
	for _, name := range s.recordFiles() {
		raw, err := os.ReadFile(filepath.Join(s.cfg.Dir, name))
		if err != nil {
			continue
		}
		var rec StoredRecord
		if err := json.Unmarshal(raw, &rec); err != nil || rec.Bundle == nil {
			continue
		}
		if rec.Bundle.EvaluationID != evaluationID {
			continue
		}
		if best == nil || rec.StoredAt.After(best.StoredAt) {
			candidate := rec
			best = &candidate
		}
	}
	if best == nil {
		return nil, fmt.Errorf("no retained record for evaluation %q", evaluationID)
	}
	verified, err := s.loadVerified(best.Bundle.Digest)
	if err != nil {
		return nil, err
	}
	return verified.Bundle, nil
}

// loadVerified reads and fully verifies one record: signature over the stored
// bytes, purpose binding, evaluation identity and content digest. Every check
// is repeated here rather than delegated, because this is the single path both
// Read and ReadLatestFor use.
func (s *EvidenceStore) loadVerified(recordDigest string) (*StoredRecord, error) {
	path := s.recordPath(recordDigest)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read evidence record %q: %w", recordDigest, err)
	}
	var rec StoredRecord
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&rec); err != nil {
		return nil, fmt.Errorf("decode evidence record %q: %w", recordDigest, err)
	}
	if rec.Bundle == nil {
		return nil, fmt.Errorf("evidence record %q carries no bundle", recordDigest)
	}
	if rec.Purpose != evidenceRecordPurpose {
		return nil, fmt.Errorf("evidence record %q carries purpose %q", recordDigest, rec.Purpose)
	}
	payload := canonicalJSON(recordSigned{
		Bundle:   rec.Bundle,
		KeyID:    rec.KeyID,
		Purpose:  rec.Purpose,
		StoredAt: rec.StoredAt,
	})
	if err := evidence.VerifyDetached(s.publicKey(), rec.Purpose, payload, rec.Signature); err != nil {
		return nil, fmt.Errorf("evidence record %q is unverified: %w", recordDigest, err)
	}
	if rec.Bundle.Digest != recordDigest {
		return nil, fmt.Errorf("evidence record %q contains bundle %s", recordDigest, rec.Bundle.Digest)
	}
	if got := rec.Bundle.computeDigest(); got != rec.Bundle.Digest {
		return nil, fmt.Errorf("evidence record %q is unverified: stored digest does not match its content", recordDigest)
	}
	return &rec, nil
}

// recordFiles lists the record files, sorted for determinism.
//
// Only files this store writes are considered: a directory is not a place a
// caller can put a file and have it treated as evidence, so anything not named
// by recordPath is ignored rather than parsed.
func (s *EvidenceStore) recordFiles() []string {
	entries, err := os.ReadDir(s.cfg.Dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if !strings.HasPrefix(n, "eval-") || !strings.HasSuffix(n, ".json") {
			continue
		}
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// authorize applies the same principal decision model as tools and resources
// (Spec 16 R16-04), at ADMIN class. An evidence bundle describes what a
// deployment blocked and why; read-class authority is not sufficient.
func (s *EvidenceStore) authorize(caller auth.Caller) bool {
	d := auth.Decide(caller, auth.ActionRequest{
		Name:     "controlplane.evidence.read",
		Kind:     auth.ActionAdmin,
		Resource: "guardrail://control-plane/evidence",
		Scopes:   []string{auth.ScopeAdminManage},
	})
	return d.Allow
}

// Sweep applies the retention and count policy and reports what it removed.
//
// Deletion order is oldest-first and deterministic, so a repeated sweep over an
// unchanged store deletes nothing and a test can assert exactly which records
// were removed. Every removal emits an audit event; a sweep that removed
// something without an event is a bug in this function, not a policy choice.
func (s *EvidenceStore) Sweep() ([]Event, error) {
	if s == nil {
		return nil, errors.New("evidence store is not configured")
	}
	if _, err := os.ReadDir(s.cfg.Dir); err != nil {
		return nil, fmt.Errorf("read evidence store: %w", err)
	}
	type candidate struct {
		id     string
		digest string
		path   string
		stored time.Time
	}
	var live []candidate
	for _, name := range s.recordFiles() {
		path := filepath.Join(s.cfg.Dir, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var rec StoredRecord
		if err := json.Unmarshal(raw, &rec); err != nil || rec.Bundle == nil {
			// An unreadable record is not evidence and is not deleted
			// silently: it is reported as refused so an operator sees a
			// corrupt store rather than a store that quietly shrank.
			s.cfg.Audit(Event{Action: "refused", Reason: "record is not decodable: " + name, At: s.now().UTC()})
			continue
		}
		live = append(live, candidate{id: rec.Bundle.EvaluationID, digest: rec.Bundle.Digest, path: path, stored: rec.StoredAt})
	}
	sort.Slice(live, func(i, j int) bool {
		if live[i].stored.Equal(live[j].stored) {
			return live[i].digest < live[j].digest
		}
		return live[i].stored.Before(live[j].stored)
	})

	now := s.now().UTC()
	var removed []Event
	for _, c := range live {
		age := now.Sub(c.stored)
		if age <= s.cfg.Retention {
			break
		}
		ev := Event{
			Action:       "deleted",
			EvaluationID: c.id,
			Reason:       fmt.Sprintf("retention %s exceeded", s.cfg.Retention),
			Age:          age.Truncate(time.Second).String(),
			At:           now,
		}
		if err := os.Remove(c.path); err != nil {
			return removed, fmt.Errorf("delete evidence record %q: %w", c.id, err)
		}
		s.cfg.Audit(ev)
		removed = append(removed, ev)
	}

	// The count bound applies after time retention, to what survives it.
	remaining := live[len(removed):]
	if s.cfg.MaxRecords > 0 && len(remaining) > s.cfg.MaxRecords {
		over := remaining[:len(remaining)-s.cfg.MaxRecords]
		for _, c := range over {
			ev := Event{
				Action:       "deleted",
				EvaluationID: c.id,
				Reason:       fmt.Sprintf("max_records %d exceeded", s.cfg.MaxRecords),
				Age:          now.Sub(c.stored).Truncate(time.Second).String(),
				At:           now,
			}
			if err := os.Remove(c.path); err != nil {
				return removed, fmt.Errorf("delete evidence record %q: %w", c.id, err)
			}
			s.cfg.Audit(ev)
			removed = append(removed, ev)
		}
	}
	return removed, nil
}

// Erase removes one record on request. It is the explicit half of the deletion
// policy: a subject-access or incident erasure needs to name a record, and it
// must be distinguishable from retention in the audit trail.
func (s *EvidenceStore) Erase(recordDigest, reason string) error {
	if s == nil {
		return errStoreNotConfigured
	}
	if strings.TrimSpace(reason) == "" {
		return errors.New("erasing a retained record requires a reason")
	}
	path := s.recordPath(recordDigest)
	rec, err := s.loadVerified(recordDigest)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("no retained record %q", recordDigest)
		}
		return fmt.Errorf("erase evidence record %q: %w", recordDigest, err)
	}
	s.cfg.Audit(Event{
		Action:       "erased",
		EvaluationID: rec.Bundle.EvaluationID,
		Reason:       reason,
		At:           s.now().UTC(),
	})
	return nil
}

// EraseEvaluation removes EVERY retained record for one evaluation id.
//
// This is the subject-access / incident case: the caller knows which evaluation
// they need gone, not which content digest each run produced. It is separate
// from Erase on purpose — "remove this record" and "remove this evaluation's
// history" are different requests, and collapsing them means an operator who
// meant the first silently destroys the second.
func (s *EvidenceStore) EraseEvaluation(evaluationID, reason string) (int, error) {
	if s == nil {
		return 0, errStoreNotConfigured
	}
	if strings.TrimSpace(reason) == "" {
		return 0, errors.New("erasing a retained evaluation requires a reason")
	}
	var removed int
	for _, name := range s.recordFiles() {
		path := filepath.Join(s.cfg.Dir, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var rec StoredRecord
		if err := json.Unmarshal(raw, &rec); err != nil || rec.Bundle == nil {
			continue
		}
		if rec.Bundle.EvaluationID != evaluationID {
			continue
		}
		if err := os.Remove(path); err != nil {
			return removed, fmt.Errorf("erase evidence record %q: %w", rec.Bundle.Digest, err)
		}
		s.cfg.Audit(Event{
			Action:       "erased",
			EvaluationID: evaluationID,
			Reason:       reason,
			At:           s.now().UTC(),
		})
		removed++
	}
	if removed == 0 {
		return 0, fmt.Errorf("no retained records for evaluation %q", evaluationID)
	}
	return removed, nil
}

// recordPath maps a record's content digest to a file name. The digest is a
// derived value, but it is still sanitised: a caller-supplied identity must not
// be able to address a path outside the store.
func (s *EvidenceStore) recordPath(recordDigest string) string {
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			return r
		case r == ':':
			return '_'
		default:
			return '_'
		}
	}, recordDigest)
	return filepath.Join(s.cfg.Dir, "eval-"+safe+".json")
}

func (s *EvidenceStore) publicKey() ed25519.PublicKey {
	return s.cfg.Signer.PublicKey()
}

// writeFile0600 writes atomically at owner-only mode. A record readable by
// another local account is an access-control failure that no policy field can
// repair afterwards.
func writeFile0600(path string, body []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".evidence-*")
	if err != nil {
		return fmt.Errorf("create evidence temp file: %w", err)
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("set evidence record mode: %w", err)
	}
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		return fmt.Errorf("write evidence record: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync evidence record: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close evidence record: %w", err)
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("replace evidence record: %w", err)
	}
	return nil
}

// Retention reports the configured window as a string. It is exposed so the
// control-plane diagnostic can report the policy from the artifact rather than
// from a runbook.
func (s *EvidenceStore) Retention() string {
	if s == nil {
		return ""
	}
	return s.cfg.Retention.String()
}

// MaxRecords reports the configured count bound (0 = not configured).
func (s *EvidenceStore) MaxRecords() int {
	if s == nil {
		return 0
	}
	return s.cfg.MaxRecords
}

// StoreSummary is the non-content state of the store.
type StoreSummary struct {
	// Records is the number of stored records.
	Records int `json:"records"`
	// Unverified counts records whose signature or digest did not check out.
	// A non-zero value is a store-integrity finding.
	Unverified int `json:"unverified"`
	// Errors counts files that could not be decoded at all.
	Errors int `json:"errors"`
}

// Summary reads the store without content and without authorization. It is
// deliberately coarse: a status endpoint that answers "how many records" should
// not be a way to enumerate which evaluations ran, which is what the
// ADMIN-class List is for.
func (s *EvidenceStore) Summary() StoreSummary {
	var out StoreSummary
	if s == nil {
		return out
	}
	for _, name := range s.recordFiles() {
		raw, err := os.ReadFile(filepath.Join(s.cfg.Dir, name))
		if err != nil {
			out.Errors++
			continue
		}
		var rec StoredRecord
		if err := json.Unmarshal(raw, &rec); err != nil || rec.Bundle == nil {
			out.Errors++
			continue
		}
		out.Records++
		payload := canonicalJSON(recordSigned{
			Bundle:   rec.Bundle,
			KeyID:    rec.KeyID,
			Purpose:  rec.Purpose,
			StoredAt: rec.StoredAt,
		})
		if err := evidence.VerifyDetached(s.publicKey(), rec.Purpose, payload, rec.Signature); err != nil {
			out.Unverified++
			continue
		}
		if got := rec.Bundle.computeDigest(); got != rec.Bundle.Digest {
			out.Unverified++
		}
	}
	return out
}

// RetentionSweepInterval is how often the retention policy is applied. It is
// published so a change is visible in the build rather than hidden in a caller.
//
// A daily sweep would be wrong: the window is a time bound and a store that is
// only trimmed once a day can exceed it by a day. A sweep every interval keeps
// the store within one interval of its configured window, which is the
// guarantee an operator reads the setting to mean.
const RetentionSweepInterval = time.Hour

// StartRetentionLoop applies the retention policy immediately and then every
// interval until ctx is cancelled.
//
// This exists because Sweep with no caller is a built-but-unreachable control:
// the policy would be configurable, auditable and never enforced — the exact
// defect class this spec is about. The first sweep runs before the function
// returns so a store that was already over its window is trimmed at startup
// rather than an interval later.
func (s *EvidenceStore) StartRetentionLoop(ctx context.Context, interval time.Duration) {
	if s == nil {
		return
	}
	if interval <= 0 {
		interval = RetentionSweepInterval
	}
	sweep := func() {
		removed, err := s.Sweep()
		if err != nil {
			// A sweep failure is reported, never fatal: the process is serving
			// and a store that could not be trimmed must not take the API down.
			// The audit sink already carries whatever was removed before the
			// error, so a partial sweep is reconstructable.
			s.cfg.Audit(Event{Action: "refused", Reason: "retention sweep failed: " + err.Error(), At: s.now().UTC()})
			return
		}
		if len(removed) > 0 {
			s.cfg.Audit(Event{Action: "deleted", Reason: "retention sweep complete", Count: len(removed), At: s.now().UTC()})
		}
	}
	sweep()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sweep()
			}
		}
	}()
}
