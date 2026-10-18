package evidence

// Detached signing over stored artifacts (Spec 08 R08-4.3.6).
//
// The cross-product envelope contract (gr-xp-01..06) describes artifacts that
// travel between products. A control-plane evidence bundle is a different
// thing: it is stored locally by the process that produced it, and the
// requirement is narrower — the retained record must be verifiable as the one
// that evaluation produced, so a later reader can tell an edited record from an
// original.
//
// Reusing the instance key is deliberate: one process presents one identity,
// and a second key for local storage would be a second thing to provision and
// rotate. The signature is domain separated, so a signature over a stored
// record can never be replayed as a signature over a cross-product envelope,
// and the reverse.
//
// This is not an authorization. A verified record says "this producer wrote
// this bundle"; it says nothing about whether the bundle's verdict permits any
// effect (gr-xp-03).

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
)

// storedRecordDomain separates local stored-evidence signatures from the
// envelope signature domain. Without it the same key would produce signatures
// that are valid over two different artifact contracts.
const storedRecordDomain = "guardrail-stored-evidence-v1\x00"

// SignDetached signs an arbitrary payload with this instance's key, binding the
// payload to a caller-supplied purpose string.
//
// The purpose is required and must not be empty: a signature that does not say
// what it is over can be moved to another artifact with the same bytes, which
// is the failure domain separation exists to prevent.
func (i *Instance) SignDetached(purpose string, payload []byte) (string, error) {
	if i == nil || len(i.private) != ed25519.PrivateKeySize {
		return "", errors.New("signing identity is not usable")
	}
	if purpose == "" {
		return "", errors.New("a detached signature requires a purpose")
	}
	if len(payload) == 0 {
		return "", errors.New("refusing to sign an empty payload")
	}
	msg := detachedMessage(purpose, payload)
	return base64.StdEncoding.EncodeToString(ed25519.Sign(i.private, msg)), nil
}

// VerifyDetached checks a detached signature against a public key.
//
// A malformed signature and a wrong signature return the same error on purpose:
// the caller is deciding whether to trust stored bytes, and the distinction is
// not actionable for them.
func VerifyDetached(pub ed25519.PublicKey, purpose string, payload []byte, signatureB64 string) error {
	if len(pub) != ed25519.PublicKeySize {
		return errors.New("verification key is not an Ed25519 public key")
	}
	if purpose == "" {
		return errors.New("a detached signature requires a purpose")
	}
	if len(payload) == 0 {
		return errors.New("refusing to verify an empty payload")
	}
	sig, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return errors.New("signature is not a valid Ed25519 detached signature")
	}
	if !ed25519.Verify(pub, detachedMessage(purpose, payload), sig) {
		return fmt.Errorf("detached signature does not verify for purpose %q", purpose)
	}
	return nil
}

// detachedMessage is the exact byte sequence signed: purpose, separator, payload.
// The NUL after the purpose keeps "a" + "bc" from colliding with "ab" + "c".
func detachedMessage(purpose string, payload []byte) []byte {
	out := make([]byte, 0, len(storedRecordDomain)+len(purpose)+1+len(payload))
	out = append(out, storedRecordDomain...)
	out = append(out, purpose...)
	out = append(out, 0)
	out = append(out, payload...)
	return out
}
