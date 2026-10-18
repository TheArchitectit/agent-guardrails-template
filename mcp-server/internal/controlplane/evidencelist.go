package controlplane

// Evidence store listing (Spec 08 R08-4.3.6, access-control half).
//
// A store an operator can only read by knowing an evaluation id is a store
// nobody finds. The listing reports what is retained, which key it verifies
// under, and — for each record — whether the signature actually verifies. That
// last field is the point: a listing that reports a record as present but does
// not say whether it is intact invites a reader to assume the bytes are
// evidence.
//
// The listing carries no bundle content. Content stays behind a single-record
// read so an access-control mistake on one record cannot become a bulk export.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/thearchitectit/guardrail-mcp/internal/auth"
	"github.com/thearchitectit/guardrail-mcp/internal/evidence"
)

// RetainedRecord is one row of the listing.
type RetainedRecord struct {
	EvaluationID string    `json:"evaluation_id"`
	Verdict      string    `json:"verdict"`
	StoredAt     time.Time `json:"stored_at"`
	// Verified is true only when the detached signature checked out over the
	// stored bytes. `false` means the record is present and is NOT evidence.
	Verified bool `json:"verified"`
	// Reason explains an unverified row and is empty when verified.
	Reason string `json:"reason,omitempty"`
	// Age is the retention clock for this record, so an operator can see which
	// rows the next sweep will remove.
	Age string `json:"age"`
}

// Listing is the whole store view.
type Listing struct {
	// Directory is reported as a path only; it holds no secret.
	Directory string `json:"directory"`
	// Retention and MaxRecords are the configured policy, echoed so a reader
	// learns the policy from the artifact rather than from a runbook.
	Retention  string `json:"retention"`
	MaxRecords int    `json:"max_records,omitempty"`
	// KeyID is the identity retained records verify under.
	KeyID string `json:"key_id,omitempty"`
	// RetentionEnforced is false when the store was opened without a checker
	// (a read-only inspection). It is reported rather than assumed so a
	// consumer cannot read "retention" and infer enforcement.
	RetentionEnforced bool             `json:"retention_enforced"`
	Records           []RetainedRecord `json:"records"`
	// Unverified counts rows whose signature did not verify. A non-zero value
	// is a store-integrity finding, not a cosmetic detail.
	Unverified int `json:"unverified"`
}

// List returns the store view. Like Read it is an ADMIN-class operation: the
// set of evaluations a deployment ran is operational detail, not guardrail
// documentation.
func (s *EvidenceStore) List(caller auth.Caller) (*Listing, error) {
	if s == nil {
		return nil, errStoreNotConfigured
	}
	if !s.authorize(caller) {
		return nil, errStorePermissionDenied
	}
	out := &Listing{
		Directory:         s.cfg.Dir,
		Retention:         s.cfg.Retention.String(),
		MaxRecords:        s.cfg.MaxRecords,
		KeyID:             s.SignerKeyID(),
		RetentionEnforced: true,
		Records:           []RetainedRecord{},
	}
	entries, err := os.ReadDir(s.cfg.Dir)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(s.cfg.Dir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var rec StoredRecord
		if err := json.Unmarshal(raw, &rec); err != nil || rec.Bundle == nil {
			out.Records = append(out.Records, RetainedRecord{
				EvaluationID: strings.TrimSuffix(strings.TrimPrefix(e.Name(), "eval-"), ".json"),
				Verified:     false,
				Reason:       "record is not decodable",
				StoredAt:     time.Time{},
			})
			out.Unverified++
			continue
		}
		row := RetainedRecord{
			EvaluationID: rec.Bundle.EvaluationID,
			Verdict:      string(rec.Bundle.Verdict),
			StoredAt:     rec.StoredAt,
			Age:          now.Sub(rec.StoredAt).Truncate(time.Second).String(),
		}
		payload := canonicalJSON(recordSigned{
			Bundle:   rec.Bundle,
			KeyID:    rec.KeyID,
			Purpose:  rec.Purpose,
			StoredAt: rec.StoredAt,
		})
		if err := evidence.VerifyDetached(s.publicKey(), rec.Purpose, payload, rec.Signature); err != nil {
			row.Reason = err.Error()
		} else if got := rec.Bundle.computeDigest(); got != rec.Bundle.Digest {
			row.Reason = "stored digest does not match its content"
		} else {
			row.Verified = true
		}
		if !row.Verified {
			out.Unverified++
		}
		out.Records = append(out.Records, row)
	}
	sort.Slice(out.Records, func(i, j int) bool {
		if out.Records[i].StoredAt.Equal(out.Records[j].StoredAt) {
			return out.Records[i].EvaluationID < out.Records[j].EvaluationID
		}
		return out.Records[i].StoredAt.Before(out.Records[j].StoredAt)
	})
	return out, nil
}
