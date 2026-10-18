// Package team provides team management functionality with optional encryption.
package team

import (
	"errors"
	"fmt"
	"strings"

	"github.com/thearchitectit/guardrail-mcp/internal/secretbox"
)

// Team data at rest (SEC-007 / Spec 17 R17-06).
//
// Team assignments persist as JSON under `.teams/`, and the assignee names in
// them are personal data. R17-06 requires that such material either live in a
// secret store or be encrypted at rest, and the secret inventory has documented
// `TEAM_ENCRYPTION_KEY` as the key that does it since the inventory was written.
//
// It did not do it. This file previously held a standalone AES-GCM manager that
// *nothing constructed*: `NewEncryptionManager` had zero non-test callers, so
// `Manager.save` wrote plaintext JSON at mode 0644 and the documented control
// protected nothing. Worse, the manager it would have used returned the input
// unchanged on every error path — a decryption failure silently yielded the
// ciphertext as if it were the plaintext, which is a fail-open on exactly the
// path that must not fail open.
//
// The manager is now a thin, fail-closed wrapper over internal/secretbox, which
// is the repository's single decision for sealing credential material at rest
// (versioned `v1:` form, AES-256-GCM, authenticated). There is one sealing
// implementation in this tree, not two.
//
// An unconfigured manager is a legitimate state: `local` may run without a key,
// and data written then is legacy plaintext that this type reports as such
// rather than mislabelling. What is not legitimate — and is now an error — is
// reading sealed data without a usable key.

// ErrTeamEncryptionKeyRequired is returned when sealed team data is read with no
// configured key. Returning the stored bytes instead would present ciphertext to
// callers as if it were an assignee name.
var ErrTeamEncryptionKeyRequired = errors.New("team data is encrypted at rest but no TEAM_ENCRYPTION_KEY is configured")

// EncryptionManager seals and opens team data at rest.
type EncryptionManager struct {
	sealer *secretbox.Sealer
}

// NewEncryptionManager builds a manager from an operator-supplied key. An empty
// key yields a disabled manager and no error: team data is an optional feature
// and `local` is allowed to run unsealed. A non-empty key that cannot be used is
// an error — a manager that silently does nothing while a key is configured is
// the defect this function exists to remove.
func NewEncryptionManager(key string) (*EncryptionManager, error) {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return &EncryptionManager{}, nil
	}
	sealer, err := secretbox.New(trimmed)
	if err != nil {
		return nil, fmt.Errorf("team encryption key is unusable: %w", err)
	}
	return &EncryptionManager{sealer: sealer}, nil
}

// Enabled reports whether team data is sealed at rest. False means the manager
// has no key: writes stay plaintext and reads accept plaintext only.
func (em *EncryptionManager) Enabled() bool {
	return em != nil && em.sealer != nil
}

// Seal protects one field value. An empty value stays empty (there is nothing to
// protect, and `assigned_to` distinguishes absent from empty).
func (em *EncryptionManager) Seal(plaintext string) (string, error) {
	if !em.Enabled() {
		return plaintext, nil
	}
	if plaintext == "" {
		return "", nil
	}
	return em.sealer.Seal(plaintext)
}

// Open reverses Seal. A value that is not in the sealed form is returned
// unchanged with sealed=false, so data written before encryption was wired keeps
// loading while a migration is in progress — the caller can report that honestly
// instead of claiming the data is protected.
//
// A sealed value with no configured key, a wrong key, or a tampered ciphertext
// is an ERROR. It is never returned as if it were the plaintext.
func (em *EncryptionManager) Open(stored string) (plaintext string, sealed bool, err error) {
	if stored == "" {
		return "", false, nil
	}
	if !secretbox.IsSealed(stored) {
		return stored, false, nil
	}
	if !em.Enabled() {
		return "", true, ErrTeamEncryptionKeyRequired
	}
	return em.sealer.Open(stored)
}

// sealTeam returns a copy of the team with its assignee names sealed. The copy
// matters: sealing must not mutate the in-memory state callers read from.
func (em *EncryptionManager) sealTeam(team Team) (Team, error) {
	if !em.Enabled() {
		return team, nil
	}
	sealed := copyTeam(team)
	for i := range sealed.Roles {
		if sealed.Roles[i].AssignedTo == nil || *sealed.Roles[i].AssignedTo == "" {
			continue
		}
		value, err := em.Seal(*sealed.Roles[i].AssignedTo)
		if err != nil {
			return Team{}, fmt.Errorf("failed to seal assignment for team %d role %q: %w", team.ID, sealed.Roles[i].Name, err)
		}
		sealed.Roles[i].AssignedTo = &value
	}
	return sealed, nil
}

// openTeam returns a copy of the team with its assignee names opened.
func (em *EncryptionManager) openTeam(team Team) (Team, error) {
	opened := copyTeam(team)
	for i := range opened.Roles {
		if opened.Roles[i].AssignedTo == nil || *opened.Roles[i].AssignedTo == "" {
			continue
		}
		value, _, err := em.Open(*opened.Roles[i].AssignedTo)
		if err != nil {
			return Team{}, fmt.Errorf("failed to open assignment for team %d role %q: %w", team.ID, opened.Roles[i].Name, err)
		}
		opened.Roles[i].AssignedTo = &value
	}
	return opened, nil
}
