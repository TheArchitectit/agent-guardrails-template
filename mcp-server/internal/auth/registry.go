// Package auth implements the explicit credential-to-principal registry
// defined by Spec 15 / gr-xp-01.
//
// Identity is never derived from the request secret or from a truncated log
// hash: a caller principal is resolved only from a stored credential record.
// Each record carries a keyed one-way verifier (HMAC-SHA256) of the secret;
// the secret itself is never stored, logged, or returned.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// verifierLen is the hex length of a SHA-256 HMAC digest.
const verifierLen = sha256.Size * 2

// Record is a stored credential record. It identifies the credential and the
// principal it maps to. It never contains the presented secret: only a keyed
// one-way verifier digest of it.
type Record struct {
	CredentialID string    `json:"credential_id"`
	PrincipalID  string    `json:"principal_id"`
	Scopes       []string  `json:"scopes,omitempty"`
	IssuedAt     time.Time `json:"issued_at,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	Revoked      bool      `json:"revoked,omitempty"`
	// Verifier is the hex-encoded keyed one-way digest of the credential
	// secret, produced with Digest and the registry verifier key.
	Verifier string `json:"verifier"`
}

// Principal is the resolved identity of a caller.
type Principal struct {
	ID           string
	CredentialID string
	Scopes       []string
}

// Registry resolves a presented secret to a principal.
type Registry struct {
	records     []Record
	verifierKey []byte
}

// Digest returns the hex-encoded keyed one-way verifier for a secret. It is
// the only supported way to derive a stored Verifier; the raw secret is never
// persisted.
func Digest(verifierKey, secret string) string {
	mac := hmac.New(sha256.New, []byte(verifierKey))
	mac.Write([]byte(secret))
	return hex.EncodeToString(mac.Sum(nil))
}

// New builds a registry from the given verifier key and records. It rejects
// records that lack a credential ID, principal ID, or a well-formed verifier.
func New(verifierKey string, records []Record) (*Registry, error) {
	if len(records) == 0 {
		return nil, nil
	}
	if verifierKey == "" {
		return nil, fmt.Errorf("credential registry requires a non-empty verifier key")
	}
	seen := make(map[string]bool, len(records))
	for i, r := range records {
		if r.CredentialID == "" {
			return nil, fmt.Errorf("credential registry record %d: missing credential_id", i)
		}
		if r.PrincipalID == "" {
			return nil, fmt.Errorf("credential registry record %d: missing principal_id", i)
		}
		if len(r.Verifier) != verifierLen {
			return nil, fmt.Errorf("credential registry record %d: verifier must be %d hex chars", i, verifierLen)
		}
		if _, err := hex.DecodeString(r.Verifier); err != nil {
			return nil, fmt.Errorf("credential registry record %d: verifier is not valid hex", i)
		}
		if r.Verifier != strings.ToLower(r.Verifier) {
			return nil, fmt.Errorf("credential registry record %d: verifier must be lowercase hex", i)
		}
		if seen[r.Verifier] {
			return nil, fmt.Errorf("credential registry record %d: duplicate verifier", i)
		}
		seen[r.Verifier] = true
	}
	return &Registry{records: records, verifierKey: []byte(verifierKey)}, nil
}

// Enabled reports whether the registry holds usable records. An absent or
// unusable registry is not an error; callers then fall back to the legacy
// two-key surface.
func (r *Registry) Enabled() bool {
	return r != nil && len(r.records) > 0 && len(r.verifierKey) > 0
}

// Resolve maps a presented secret to a principal. It compares the keyed
// verifier of the presented secret against every record with a constant-time
// comparison, so the time taken does not reveal which record (if any) matched.
// Revoked and expired records never resolve.
func (r *Registry) Resolve(secret string) (Principal, bool) {
	if !r.Enabled() || secret == "" {
		return Principal{}, false
	}

	presented := Digest(string(r.verifierKey), secret)

	var matched *Record
	for i := range r.records {
		// No early exit: every record is compared on every call.
		if subtle.ConstantTimeCompare([]byte(presented), []byte(r.records[i].Verifier)) == 1 {
			matched = &r.records[i]
		}
	}
	if matched == nil {
		return Principal{}, false
	}
	if matched.Revoked {
		return Principal{}, false
	}
	if !matched.ExpiresAt.IsZero() && time.Now().After(matched.ExpiresAt) {
		return Principal{}, false
	}

	return Principal{
		ID:           matched.PrincipalID,
		CredentialID: matched.CredentialID,
		Scopes:       matched.Scopes,
	}, true
}

// LoadFromSources builds a registry from the configured out-of-band sources: an
// inline JSON record list and/or a JSON file path. When neither source yields
// records it returns (nil, nil), meaning "no registry configured".
func LoadFromSources(inlineJSON, filePath, verifierKey string) (*Registry, error) {
	var records []Record

	if strings.TrimSpace(inlineJSON) != "" {
		var inline []Record
		if err := json.Unmarshal([]byte(inlineJSON), &inline); err != nil {
			return nil, fmt.Errorf("credential registry (inline JSON) is invalid: %w", err)
		}
		records = append(records, inline...)
	}

	if strings.TrimSpace(filePath) != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("credential registry file could not be read: %w", err)
		}
		var fromFile []Record
		if err := json.Unmarshal(data, &fromFile); err != nil {
			return nil, fmt.Errorf("credential registry file is invalid JSON: %w", err)
		}
		records = append(records, fromFile...)
	}

	return New(verifierKey, records)
}
