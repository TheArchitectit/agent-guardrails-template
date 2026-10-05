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
	"strings"
	"sync"
	"time"
)

// verifierLen is the hex length of a SHA-256 HMAC digest.
const verifierLen = sha256.Size * 2

// minVerifierKeyLen is the minimum usable length of the registry verifier key.
// Shorter material is rejected at load so an unusable key can never silently
// produce a registry that fails open.
const minVerifierKeyLen = 16

// RevocationPropagationBound is the published maximum delay between a
// revocation becoming visible in the registry source and new web/MCP requests
// being denied (Spec 16 §4 R16-09). The bound is met by loading the complete
// replacement snapshot and swapping it in atomically: a revoked record denies
// the very next Resolve after Reload, so propagation is bounded by the reload
// interval, not by any lazy re-verification. Operators MUST reload (or
// restart/replace the process) often enough to stay within this bound.
const RevocationPropagationBound = 60 * time.Second

// MaxRotationOverlap is the maximum supported overlap window during which a
// rotated-out verifier key still resolves alongside its replacement
// (Spec 16 §4: planned rotation overlap is at most 24 hours). A longer window
// is rejected rather than silently accepted, and a rotated-out key is never
// accepted indefinitely.
const MaxRotationOverlap = 24 * time.Hour

// Record is a stored credential record. It identifies the credential and the
// principal it maps to. It never contains the presented secret: only a keyed
// one-way verifier digest of it.
type Record struct {
	CredentialID string   `json:"credential_id"`
	PrincipalID  string   `json:"principal_id"`
	Scopes       []string `json:"scopes,omitempty"`
	// Role is the named role grant for the principal (Spec 16 §4 catalog).
	// Missing/unknown roles deny at the authorization decision boundary.
	Role string `json:"role,omitempty"`
	// Resources lists project/resource IDs the principal may act on.
	// "*" grants all resources. An empty list grants none.
	Resources []string  `json:"resources,omitempty"`
	IssuedAt  time.Time `json:"issued_at,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	Revoked   bool      `json:"revoked,omitempty"`
	// Verifier is the hex-encoded keyed one-way digest of the credential
	// secret, produced with Digest and the registry verifier key.
	Verifier string `json:"verifier"`
}

// Principal is the resolved identity of a caller. Identity comes only from
// the credential record — never from tool arguments or request content.
type Principal struct {
	ID           string
	CredentialID string
	Scopes       []string
	Role         string
	Resources    []string
}

// Caller converts a resolved principal into an authorization Caller.
func (p Principal) Caller() Caller {
	return Caller{
		PrincipalID:  p.ID,
		CredentialID: p.CredentialID,
		Scopes:       p.Scopes,
		Role:         p.Role,
		Resources:    p.Resources,
	}
}

// verifierKeyEntry is one active verifier key. validUntil is the end of a
// rotation overlap window; the zero value means the key is valid until it is
// explicitly replaced.
type verifierKeyEntry struct {
	key        []byte
	validUntil time.Time
}

func (k verifierKeyEntry) active(now time.Time) bool {
	return k.validUntil.IsZero() || now.Before(k.validUntil)
}

// snapshot is an immutable, fully-validated registry state. It is replaced
// whole, never mutated in place, so concurrent Resolve calls always observe a
// consistent snapshot.
type snapshot struct {
	records []Record
	keys    []verifierKeyEntry
}

// Registry resolves a presented secret to a principal. The whole state lives
// behind an atomically-swappable snapshot pointer guarded by mu so that a
// reload never exposes a partially-built or partially-validated registry.
type Registry struct {
	mu   sync.RWMutex
	snap *snapshot
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
// records that lack a credential ID, principal ID, or a well-formed verifier,
// records carrying a scope outside the approved catalog, duplicate credential
// IDs or verifiers, and unusable verifier-key material.
func New(verifierKey string, records []Record) (*Registry, error) {
	if len(records) == 0 {
		return nil, nil
	}
	snap, err := buildSnapshot([]verifierKeyEntry{{key: []byte(verifierKey)}}, records, time.Now())
	if err != nil {
		return nil, err
	}
	return &Registry{snap: snap}, nil
}

// buildSnapshot validates a complete candidate state. Nothing is returned
// unless every key and every record passes, so a caller can never install a
// partially-validated replacement.
func buildSnapshot(keys []verifierKeyEntry, records []Record, now time.Time) (*snapshot, error) {
	if len(records) == 0 {
		return nil, nil
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("credential registry requires a non-empty verifier key")
	}

	usableKeys := 0
	for _, k := range keys {
		if strings.TrimSpace(string(k.key)) == "" {
			return nil, fmt.Errorf("credential registry requires a non-empty verifier key")
		}
		if len(k.key) < minVerifierKeyLen {
			return nil, fmt.Errorf("credential registry verifier key must be at least %d characters", minVerifierKeyLen)
		}
		if k.active(now) {
			usableKeys++
		}
	}
	if usableKeys == 0 {
		return nil, fmt.Errorf("credential registry has no usable verifier key")
	}

	seenVerifier := make(map[string]bool, len(records))
	seenCredential := make(map[string]bool, len(records))
	for i, r := range records {
		if r.CredentialID == "" {
			return nil, fmt.Errorf("credential registry record %d: missing credential_id", i)
		}
		if r.PrincipalID == "" {
			return nil, fmt.Errorf("credential registry record %d: missing principal_id", i)
		}
		if seenCredential[r.CredentialID] {
			return nil, fmt.Errorf("credential registry record %d: duplicate credential_id", i)
		}
		seenCredential[r.CredentialID] = true
		// Every declared scope must be in the approved catalog
		// (mcp:read, mcp:validate, mcp:mutate, rest:read, rest:write,
		// ide:validate, admin:manage). An unknown or empty scope is rejected
		// here so it can never reach the decision boundary as a silent grant.
		for _, s := range r.Scopes {
			if s == "" || !KnownScopes[s] {
				return nil, fmt.Errorf("credential registry record %d: scope outside approved catalog", i)
			}
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
		if seenVerifier[r.Verifier] {
			return nil, fmt.Errorf("credential registry record %d: duplicate verifier", i)
		}
		seenVerifier[r.Verifier] = true
	}
	return &snapshot{records: records, keys: keys}, nil
}

// Enabled reports whether the registry holds usable records. An absent or
// unusable registry is not an error; callers then fall back to the legacy
// two-key surface.
func (r *Registry) Enabled() bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.snap != nil && len(r.snap.records) > 0 && len(r.snap.keys) > 0
}

// Resolve maps a presented secret to a principal. It compares the keyed
// verifier of the presented secret against every record with a constant-time
// comparison, so the time taken does not reveal which record (if any) matched.
// Revoked and expired records never resolve.
func (r *Registry) Resolve(secret string) (Principal, bool) {
	if secret == "" {
		return Principal{}, false
	}
	if r == nil {
		return Principal{}, false
	}

	r.mu.RLock()
	snap := r.snap
	r.mu.RUnlock()
	if snap == nil || len(snap.records) == 0 || len(snap.keys) == 0 {
		return Principal{}, false
	}

	now := time.Now()
	var matched *Record
	for ki := range snap.keys {
		if !snap.keys[ki].active(now) {
			continue
		}
		presented := Digest(string(snap.keys[ki].key), secret)
		for i := range snap.records {
			// No early exit: every record is compared under every active key.
			if subtle.ConstantTimeCompare([]byte(presented), []byte(snap.records[i].Verifier)) == 1 {
				matched = &snap.records[i]
			}
		}
	}
	if matched == nil {
		return Principal{}, false
	}
	if matched.Revoked {
		return Principal{}, false
	}
	if !matched.ExpiresAt.IsZero() && now.After(matched.ExpiresAt) {
		return Principal{}, false
	}

	return Principal{
		ID:           matched.PrincipalID,
		CredentialID: matched.CredentialID,
		Scopes:       matched.Scopes,
		Role:         matched.Role,
		Resources:    matched.Resources,
	}, true
}

// Reload validates a complete replacement record set under the given verifier
// key and swaps it in atomically. On any validation error the previous
// snapshot is left untouched, so a failed reload never enlarges access and
// never restores a broader prior permission. A reload to an empty record set
// is rejected: silently disabling the registry would fall back to the legacy
// unlimited surface.
func (r *Registry) Reload(verifierKey string, records []Record) error {
	if r == nil {
		return fmt.Errorf("credential registry is not initialized")
	}
	if len(records) == 0 {
		return fmt.Errorf("credential registry reload rejected an empty record set")
	}
	snap, err := buildSnapshot([]verifierKeyEntry{{key: []byte(verifierKey)}}, records, time.Now())
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.snap = snap
	r.mu.Unlock()
	return nil
}

// ReloadFromSources re-reads the configured registry sources and swaps the
// result in atomically, using the given verifier key. It is the revocation
// path used by an operator or a periodic reloader: a record marked Revoked in
// the new source denies the next Resolve, well within
// RevocationPropagationBound.
func (r *Registry) ReloadFromSources(inlineJSON, filePath, verifierKey string) error {
	records, err := readRecords(inlineJSON, filePath)
	if err != nil {
		return err
	}
	return r.Reload(verifierKey, records)
}

// RotateVerifierKey installs a new verifier key while keeping the current
// keys usable for an overlap window of at most MaxRotationOverlap. Records
// verifier digests may be computed under either key during the window, so a
// receiver can migrate without dropping valid callers; after the window only
// the new key resolves. Rotation is a full validated replacement — a bad
// replacement leaves the current state intact.
func (r *Registry) RotateVerifierKey(newVerifierKey string, records []Record, overlap time.Duration) error {
	if r == nil {
		return fmt.Errorf("credential registry is not initialized")
	}
	if len(records) == 0 {
		return fmt.Errorf("credential registry rotation rejected an empty record set")
	}
	if overlap <= 0 {
		return fmt.Errorf("credential registry rotation requires a positive overlap window")
	}
	if overlap > MaxRotationOverlap {
		return fmt.Errorf("credential registry rotation overlap %v exceeds the %v maximum", overlap, MaxRotationOverlap)
	}

	now := time.Now()
	r.mu.RLock()
	current := r.snap
	r.mu.RUnlock()

	keys := make([]verifierKeyEntry, 0, 2)
	if current != nil {
		for _, k := range current.keys {
			// Bound every outgoing key to the overlap window; it is never
			// accepted indefinitely.
			keys = append(keys, verifierKeyEntry{key: k.key, validUntil: now.Add(overlap)})
		}
	}
	keys = append(keys, verifierKeyEntry{key: []byte(newVerifierKey)})

	snap, err := buildSnapshot(keys, records, now)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.snap = snap
	r.mu.Unlock()
	return nil
}

// SourcesConfigured reports whether any registry source is set. When true the
// registry is mandatory: a load failure must fail closed, never fall back to
// the legacy unrestricted surface.
func SourcesConfigured(inlineJSON, filePath string) bool {
	return strings.TrimSpace(inlineJSON) != "" || strings.TrimSpace(filePath) != ""
}

// readRecords parses the configured out-of-band registry sources: an inline
// JSON record list and/or a read-only JSON secret file. A configured source
// that is empty, malformed, or unreadable is an error.
func readRecords(inlineJSON, filePath string) ([]Record, error) {
	configured := SourcesConfigured(inlineJSON, filePath)
	var records []Record

	if strings.TrimSpace(inlineJSON) != "" {
		var inline []Record
		if err := json.Unmarshal([]byte(inlineJSON), &inline); err != nil {
			return nil, fmt.Errorf("credential registry (inline JSON) is invalid: %w", err)
		}
		records = append(records, inline...)
	}

	if strings.TrimSpace(filePath) != "" {
		data, err := readSecretFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("credential registry file could not be read: %w", err)
		}
		var fromFile []Record
		if err := json.Unmarshal(data, &fromFile); err != nil {
			return nil, fmt.Errorf("credential registry file is invalid JSON: %w", err)
		}
		records = append(records, fromFile...)
	}

	if configured && len(records) == 0 {
		return nil, fmt.Errorf("credential registry is configured but contains no records")
	}
	return records, nil
}

// LoadFromSources builds a registry from the configured out-of-band sources: an
// inline JSON record list and/or a JSON file path. When no source is configured
// it returns (nil, nil), meaning "no registry configured" (legacy migration
// mode). When a source IS configured, malformed/unreadable/empty input or an
// unusable verifier key is an error so callers can fail closed.
func LoadFromSources(inlineJSON, filePath, verifierKey string) (*Registry, error) {
	records, err := readRecords(inlineJSON, filePath)
	if err != nil {
		return nil, err
	}
	return New(verifierKey, records)
}

// LoadFromSourcesEx is LoadFromSources with support for a verifier key supplied
// as a read-only secret file (CREDENTIAL_VERIFIER_KEY_FILE) instead of inline
// environment material. Inline material wins when both are present. When no
// registry source is configured it returns (nil, nil).
func LoadFromSourcesEx(inlineJSON, filePath, inlineVerifierKey, verifierKeyFile string) (*Registry, error) {
	if !SourcesConfigured(inlineJSON, filePath) {
		return nil, nil
	}
	records, err := readRecords(inlineJSON, filePath)
	if err != nil {
		return nil, err
	}
	key, err := resolveVerifierKey(inlineVerifierKey, verifierKeyFile)
	if err != nil {
		return nil, err
	}
	return New(key, records)
}

// resolveVerifierKey returns the configured verifier key, reading it from a
// restricted secret file when no inline material is present.
func resolveVerifierKey(inline, filePath string) (string, error) {
	if strings.TrimSpace(inline) != "" {
		return inline, nil
	}
	if strings.TrimSpace(filePath) == "" {
		return "", nil
	}
	data, err := readSecretFile(filePath)
	if err != nil {
		return "", fmt.Errorf("credential verifier key file could not be read: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}
