package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Credential issuance and registry authoring (Spec 16 Sec. 4 R16-09).
//
// R16-09 requires the credential lifecycle to define "approved key generation,
// issuance, distinct principal assignment, expiry, overlap/rotation,
// revocation..." and warns: "Do not claim a lifecycle API exists until one is
// implemented and protected."
//
// Before this file the registry could verify, reload, rotate and revoke, and
// nothing in the tree could ISSUE a credential: Digest() had exactly one
// non-test caller (Resolve), so an operator had to compute the HMAC by hand,
// hand-write registry JSON, and hope it satisfied the loader's rules (known
// scopes, <=90-day window, unique verifier, valid hex). A single mistyped
// field produced a registry the server refused at startup — fail-closed, but
// only after the operator had already deployed it.
//
// The functions here reuse buildSnapshot (via New) as the ONE validation
// implementation, so a registry written by this path cannot differ from one
// the server accepts. Nothing here weakens a check: issuance refuses anything
// the loader would refuse, and adds the operator-facing rules (a role must
// exist in the published catalog, a scope list must be non-empty) that would
// otherwise mint a credential that denies every call it is used for.

// MinCredentialSecretBytes is the entropy of a generated credential secret.
// 32 bytes (256 bits) matches the verifier-key floor and is far beyond what an
// offline guess of a bearer token requires.
const MinCredentialSecretBytes = 32

// IssueOptions describes one credential to mint. Secret is optional: when empty
// a fresh 256-bit secret is generated. It exists so a caller (and a test) can
// supply the value a real operator already holds, without the issuance path
// ever reading it from argv.
type IssueOptions struct {
	CredentialID string
	PrincipalID  string
	Role         string
	Scopes       []string
	Resources    []string
	// Validity is the requested window. It must be positive and at most
	// MaxCredentialValidity (90 days); issuance never truncates a longer
	// request silently, because a silently shortened grant is as surprising as
	// a silently extended one.
	Validity time.Duration
	// Secret, when set, is used as-is. When empty, one is generated.
	Secret string
	// Now defaults to time.Now(). Tests pin it.
	Now time.Time
}

// GenerateCredentialSecret returns a fresh URL-safe 256-bit secret. It is the
// only generator: a credential minted by hand or by a shell script would not
// carry this entropy, and R17-07 already refuses to treat "long enough" as a
// strength proof for the verifier key.
func GenerateCredentialSecret() (string, error) {
	buf := make([]byte, MinCredentialSecretBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("credential secret generation failed: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// IssueRecord builds a validated credential record and returns the secret that
// must be handed to its holder. The secret is returned, never stored: only the
// keyed verifier is written to the record.
//
// Validation is delegated to New(), i.e. to buildSnapshot, so issuance and
// loading cannot disagree. The extra rules below are about intent, not
// strength: an unknown role or an empty scope list produces a credential that
// resolves and then denies every call, which reads to its holder as a broken
// deployment.
func IssueRecord(verifierKey string, opts IssueOptions) (Record, string, error) {
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	if strings.TrimSpace(opts.CredentialID) == "" {
		return Record{}, "", fmt.Errorf("credential_id is required")
	}
	if strings.TrimSpace(opts.PrincipalID) == "" {
		return Record{}, "", fmt.Errorf("principal_id is required")
	}
	if !KnownRoles[opts.Role] {
		return Record{}, "", fmt.Errorf("role %q is not in the published role catalog (Spec 16 Sec. 4)", opts.Role)
	}
	if len(opts.Scopes) == 0 {
		return Record{}, "", fmt.Errorf("at least one scope is required; an empty scope list grants nothing")
	}
	scopes := make([]string, 0, len(opts.Scopes))
	seen := make(map[string]bool, len(opts.Scopes))
	for _, s := range opts.Scopes {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if seen[s] {
			continue
		}
		seen[s] = true
		scopes = append(scopes, s)
	}
	if len(scopes) == 0 {
		return Record{}, "", fmt.Errorf("at least one non-empty scope is required")
	}
	if opts.Validity <= 0 {
		return Record{}, "", fmt.Errorf("validity must be positive (Spec 16 Sec. 4 / R16-09: credentials expire)")
	}
	if opts.Validity > MaxCredentialValidity {
		return Record{}, "", fmt.Errorf("validity %v exceeds the %d-day maximum (Spec 16 Sec. 4 / R16-09)", opts.Validity, int(MaxCredentialValidity.Hours()/24))
	}

	secret := opts.Secret
	if secret == "" {
		var err error
		secret, err = GenerateCredentialSecret()
		if err != nil {
			return Record{}, "", err
		}
	}

	rec := Record{
		CredentialID: opts.CredentialID,
		PrincipalID:  opts.PrincipalID,
		Scopes:       scopes,
		Role:         opts.Role,
		Resources:    append([]string(nil), opts.Resources...),
		IssuedAt:     now.UTC(),
		ExpiresAt:    now.Add(opts.Validity).UTC(),
		Verifier:     Digest(verifierKey, secret),
	}

	// The single validation path: a record that New() rejects is not issued.
	if _, err := New(verifierKey, []Record{rec}); err != nil {
		return Record{}, "", err
	}
	return rec, secret, nil
}

// LoadRegistryFile reads a registry file through the same restricted reader the
// server uses (secretfile.Read: regular file, no group/world access). A file
// the server would refuse is refused here too, so an operator learns about a
// bad mode before writing, not at the next restart.
func LoadRegistryFile(path string) ([]Record, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("registry file path is required")
	}
	data, err := readSecretFile(path)
	if err != nil {
		return nil, err
	}
	var records []Record
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("registry file is not a JSON array of credential records: %w", err)
	}
	return records, nil
}

// WriteRegistryFile validates the COMPLETE record set under verifierKey and
// then writes it atomically (temp file in the same directory, 0600, rename).
//
// The whole-set validation is the point: appending one record cannot produce a
// file that the server rejects, because the same buildSnapshot runs first. A
// partial write is impossible — the rename is atomic — and a failed write
// leaves the previous file untouched.
func WriteRegistryFile(path, verifierKey string, records []Record) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("registry file path is required")
	}
	if len(records) == 0 {
		// An empty registry is "no registry configured" to the server, which
		// falls back to the legacy two-key surface. Refuse to create that
		// state by accident.
		return fmt.Errorf("refusing to write an empty credential registry: an empty source is rejected by the loader and would disable the registry")
	}
	if _, err := New(verifierKey, records); err != nil {
		return err
	}

	body, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("registry serialization failed: %w", err)
	}
	body = append(body, '\n')

	dir := filepath.Dir(path)
	// Refuse to write through a symlink: the file must be the file the server
	// reads, not a link an attacker can repoint between validation and rename.
	if fi, err := os.Lstat(path); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to write through a symlink at %s", path)
	}

	tmp, err := os.CreateTemp(dir, ".credential-registry-*.tmp")
	if err != nil {
		return fmt.Errorf("registry temp file could not be created: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		// Best effort: if the rename succeeded this is a no-op on a missing
		// path; if it failed the temp file must not linger at 0600 with
		// verifier material in it.
		_ = os.Remove(tmpName)
	}()

	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("registry temp file mode could not be restricted: %w", err)
	}
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		return fmt.Errorf("registry temp file write failed: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("registry temp file sync failed: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("registry temp file close failed: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("registry file could not be replaced atomically: %w", err)
	}
	return nil
}

// RevokeInFile marks one credential revoked and rewrites the registry. It is
// the operator half of R16-09 revocation: the running server denies the record
// on its next reload (inside RevocationPropagationBound), and the file no
// longer presents it as usable.
//
// A missing credential id is an error rather than a silent no-op: an operator
// revoking by typo must not be told the revocation succeeding.
func RevokeInFile(path, verifierKey, credentialID string) (Record, error) {
	records, err := LoadRegistryFile(path)
	if err != nil {
		return Record{}, err
	}
	found := false
	for i := range records {
		if records[i].CredentialID != credentialID {
			continue
		}
		records[i].Revoked = true
		found = true
		break
	}
	if !found {
		return Record{}, fmt.Errorf("credential %q is not present in %s", credentialID, path)
	}
	if err := WriteRegistryFile(path, verifierKey, records); err != nil {
		return Record{}, err
	}
	for _, r := range records {
		if r.CredentialID == credentialID {
			return r, nil
		}
	}
	return Record{}, fmt.Errorf("credential %q disappeared during rewrite", credentialID)
}
