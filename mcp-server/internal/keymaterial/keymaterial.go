// Package keymaterial is the single decision for the strength of a long-lived
// cryptographic key held by the deployment (Spec 17 R17-07).
//
// R17-07: "Generate the verifier key out of band with >=256 random bits; a
// length check is not an entropy proof."
//
// Before this existed the only check on the credential-registry verifier key
// was a 16-character length floor (config.go validateCredentialRegistry and
// auth buildSnapshot). A length check accepts strings.Repeat("x", 64) — a key
// with a single distinct byte and therefore zero real entropy — and any
// 16-character key drawn from a tiny alphabet. The verifier key is the HMAC key
// behind every stored credential verifier, so a guessable key makes the whole
// credential registry forgeable: anyone who can predict the key can mint a
// verifier digest for a secret of their choosing and resolve as any principal.
//
// Two loaders enforce the key-strength rule — the credential-registry loader
// (internal/auth) and the configuration gate (internal/config) — and both must
// apply the same rule or the stricter one is decorative. Keeping one
// implementation here is what stops the two call sites from drifting apart.
//
// The estimator is a deliberate *upper* bound (alphabet size × effective
// length), so it can only reject a key that is provably weak; it never blesses
// a key as provably strong.
package keymaterial

import (
	"fmt"
	"math"
	"strings"
)

const (
	// MinVerifierKeyBits is the published floor enforced at every registry load
	// boundary: a key that cannot carry this many bits is refused, so an
	// unusable key can never silently produce a registry that fails open.
	MinVerifierKeyBits = 128.0
	// ProductionVerifierKeyBits is the R17-07 floor for a nonlocal or
	// production deployment (R17-07 names ">=256 random bits"). The `local`
	// profile keeps MinVerifierKeyBits so the documented development posture
	// still starts; that exemption is named in the spec, not silent.
	ProductionVerifierKeyBits = 256.0
)

// effectiveLen returns the length of the shortest repeating block that
// generates the whole string, or len(s) when the string is not periodic. A
// repeated block ("abcabcabc") carries no more entropy than a single copy, so a
// key built from a short pattern must not be credited its full length.
func effectiveLen(s string) int {
	n := len(s)
	for p := 1; p <= n/2; p++ {
		if n%p != 0 {
			continue
		}
		periodic := true
		for i := p; i < n; i++ {
			if s[i] != s[i%p] {
				periodic = false
				break
			}
		}
		if periodic {
			return p
		}
	}
	return n
}

// VerifierKeyEntropyBits returns an upper bound on the number of random bits a
// key could carry, from the size of its byte alphabet and its effective length.
// An empty key, or a key whose bytes are all identical, returns 0: such a key
// carries no randomness at all.
func VerifierKeyEntropyBits(key string) float64 {
	s := strings.TrimSpace(key)
	if s == "" {
		return 0
	}
	var seen [256]bool
	distinct := 0
	for i := 0; i < len(s); i++ {
		if !seen[s[i]] {
			seen[s[i]] = true
			distinct++
		}
	}
	if distinct < 2 {
		return 0
	}
	return float64(effectiveLen(s)) * math.Log2(float64(distinct))
}

// ValidateVerifierKeyEntropy refuses a verifier key that cannot carry minBits
// of entropy. The error names R17-07 and the observed bound, so an operator
// learns the key is unusable rather than merely short.
func ValidateVerifierKeyEntropy(key string, minBits float64) error {
	bits := VerifierKeyEntropyBits(key)
	if bits < minBits {
		return fmt.Errorf(
			"credential registry verifier key carries at most %.0f bits of entropy, below the required %.0f (Spec 17 R17-07: generate it out of band with a random key; a length check is not an entropy proof)",
			bits, minBits)
	}
	return nil
}
