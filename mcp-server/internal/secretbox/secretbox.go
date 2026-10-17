// Package secretbox is the single decision for encrypting credential material
// at rest (Spec 17 R17-06).
//
// R17-06 requires that secret material which persists outside a secret store
// "MUST define encrypted-at-rest storage or a managed secret reference". The
// webhook signing key is exactly such material: it persists as the
// `webhook_configs.secret_hmac` column (and the rotation overlap column
// `secret_hmac_previous`), so a database dump, a backup, or a replica read
// exposed the signing secret that authorises every delivery to that receiver.
//
// The package is deliberately small and fail-closed:
//
//   - AES-256-GCM with a random per-value nonce; the GCM tag authenticates the
//     ciphertext, so a tampered or wrong-key value is an error, never a silent
//     pass-through.
//   - A versioned `v1:` prefix on the stored form, so a sealed value is
//     distinguishable from a legacy plaintext one and a future scheme can be
//     introduced without guessing.
//   - Open reports whether the stored value was actually sealed, so a caller
//     can tell "encrypted at rest" from "legacy plaintext still present" rather
//     than assuming the stronger of the two.
//
// This is NOT the same as the credential-registry verifier key (R17-07) or the
// team-secret manager (internal/team). The team manager's Encrypt/Decrypt
// return the input unchanged on any error, which is a fail-open pattern this
// package must not copy: here an authentication failure is an error the caller
// must handle.
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

// sealedPrefix versions the stored form. A value without it is legacy plaintext
// (readable for migration, never produced by this package).
const sealedPrefix = "v1:"

// aad is the additional authenticated data bound into every seal. Binding the
// prefix means a `v1:` value cannot be replayed as a value of another scheme or
// context.
var aad = []byte(sealedPrefix)

// Sealer encrypts and authenticates short credential values at rest.
type Sealer struct {
	aead cipher.AEAD
}

// New derives a Sealer from an operator-supplied key. The key is domain-
// separated so the same operator key cannot be confused with another use, and
// hashed to the 32 bytes AES-256 requires. An empty key is an error: a sealer
// that silently does nothing is a fail-open.
func New(key string) (*Sealer, error) {
	k := strings.TrimSpace(key)
	if k == "" {
		return nil, errors.New("secretbox: encryption key is empty")
	}
	sum := sha256.Sum256([]byte("guardrail-secretbox-v1:" + k))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, fmt.Errorf("secretbox: cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("secretbox: gcm: %w", err)
	}
	return &Sealer{aead: aead}, nil
}

// IsSealed reports whether a stored value carries the sealed form.
func IsSealed(stored string) bool {
	return strings.HasPrefix(stored, sealedPrefix)
}

// Seal encrypts a plaintext secret into the stored form. An empty plaintext
// seals to an empty string (there is nothing to protect, and the store's NULL
// handling depends on distinguishing absent from empty).
func (s *Sealer) Seal(plaintext string) (string, error) {
	if s == nil || s.aead == nil {
		return "", errors.New("secretbox: sealer is not initialized")
	}
	if plaintext == "" {
		return "", nil
	}
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("secretbox: nonce: %w", err)
	}
	// Seal appends the ciphertext to the nonce, so the stored bytes are
	// nonce || ciphertext || tag.
	sealed := s.aead.Seal(nonce, nonce, []byte(plaintext), aad)
	return sealedPrefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// Open decrypts a stored value. It returns the plaintext and whether the stored
// value was actually sealed: a legacy plaintext value (no prefix) is returned
// unchanged with sealed=false so a pre-existing row keeps working while a
// migration is in progress — the caller can report that honestly rather than
// claiming the value is encrypted at rest.
//
// A sealed value that fails authentication — a wrong key, a tampered
// ciphertext, or a truncated value — is an error. It is never returned as
// plaintext.
func (s *Sealer) Open(stored string) (plaintext string, sealed bool, err error) {
	if stored == "" {
		return "", false, nil
	}
	if !IsSealed(stored) {
		return stored, false, nil
	}
	if s == nil || s.aead == nil {
		return "", false, errors.New("secretbox: sealer is not initialized")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, sealedPrefix))
	if err != nil {
		return "", false, fmt.Errorf("secretbox: stored value is not valid base64")
	}
	ns := s.aead.NonceSize()
	if len(raw) < ns+s.aead.Overhead() {
		return "", false, errors.New("secretbox: stored value is too short to be sealed")
	}
	plain, err := s.aead.Open(nil, raw[:ns], raw[ns:], aad)
	if err != nil {
		return "", false, errors.New("secretbox: stored value failed authentication (wrong key or tampered ciphertext)")
	}
	return string(plain), true, nil
}
