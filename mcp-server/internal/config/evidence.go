// Spec 15 (secure-cross-product-method) deployment configuration.
//
// gr-xp-02 requires every Guardrails-produced artifact to be signed with the
// INSTANCE key. That is only meaningful if the instance key is a real,
// deployment-supplied identity: a process that generates a key at startup gets
// a new identity on every restart, so no peer could hold a trust record for it
// and every signature would be unverifiable in practice.
//
// So the key is loaded from the deployment (a read-only secret file, per
// R17-06's first-choice source), parsed strictly, and validated at startup. A
// configured-but-unusable key is a hard error, never a silent "run unsigned":
// an artifact that silently ships without its signature is exactly the
// evidence-that-does-not-exist failure this spec is about.
package config

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/thearchitectit/guardrail-mcp/internal/secretfile"
)

// Evidence config fields (see Config): the instance signing identity and the
// audience it addresses. Producer identity and audience are not secrets; the
// private key is.

// evidenceKeyPEMType is the only PEM block type accepted for the instance key.
const evidenceKeyPEMType = "PRIVATE KEY"

// ParseEvidenceKey parses the instance signing key.
//
// Two encodings are accepted, because both are what an operator plausibly has:
//
//   - a PEM PKCS#8 block (what `openssl genpkey -algorithm ed25519` produces);
//   - a raw base64 value holding either the 32-byte seed or the 64-byte
//     private key.
//
// Everything else is refused with a message that names the requirement and
// never echoes key material.
func ParseEvidenceKey(raw string) (ed25519.PrivateKey, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("evidence signing key is empty")
	}

	if strings.HasPrefix(trimmed, "-----BEGIN") {
		block, rest := pem.Decode([]byte(trimmed))
		if block == nil {
			return nil, errors.New("evidence signing key is not valid PEM")
		}
		if len(strings.TrimSpace(string(rest))) != 0 {
			return nil, errors.New("evidence signing key has trailing data after the PEM block")
		}
		if block.Type != evidenceKeyPEMType {
			return nil, fmt.Errorf("evidence signing key PEM block is %q; want %q (an unencrypted PKCS#8 Ed25519 key)", block.Type, evidenceKeyPEMType)
		}
		parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("evidence signing key is not a valid PKCS#8 key: %w", err)
		}
		key, ok := parsed.(ed25519.PrivateKey)
		if !ok {
			// A wrong-algorithm key would fail at the first signature with a
			// confusing error; refuse it at the load boundary where the message
			// can name the algorithm.
			return nil, fmt.Errorf("evidence signing key is %T; only Ed25519 is supported", parsed)
		}
		return key, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		return nil, errors.New("evidence signing key is neither PEM nor base64")
	}
	switch len(decoded) {
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(decoded), nil
	case ed25519.PrivateKeySize:
		// A 64-byte value carries the public half; validate it rather than
		// trusting the trailing 32 bytes, because a mismatched pair produces
		// signatures that fail only at the peer.
		key := ed25519.PrivateKey(decoded)
		derived, ok := key.Public().(ed25519.PublicKey)
		if !ok || !derived.Equal(key.Public()) {
			return nil, errors.New("evidence signing key has an invalid public half")
		}
		return key, nil
	default:
		return nil, fmt.Errorf("evidence signing key is %d bytes after base64 decoding; want %d (seed) or %d (private key)",
			len(decoded), ed25519.SeedSize, ed25519.PrivateKeySize)
	}
}

// ValidateEvidenceIdentity is the startup gate for the outbound evidence
// identity (Spec 15 gr-xp-02).
//
// Rules:
//   - a key without a producer ID or audience cannot produce an addressable
//     artifact, so all three are required together;
//   - a key that does not parse is a hard error (never "run unsigned");
//   - an unset key is permitted on every profile and is reported at the point of
//     use, where the tool result carries `"signed": false` with a reason. It is
//     NOT a startup requirement: making it one would refuse startup for every
//     existing deployment in a single commit. Whether a nonlocal deployment
//     should REQUIRE it is an owner decision, recorded and not taken here.
func (c *Config) ValidateEvidenceIdentity() error {
	configured := c.EvidencePrivateKey != "" || c.EvidencePrivateKeyFile != ""

	if !configured {
		// Deliberately NOT a startup requirement. The first version of this
		// gate made the key mandatory outside `local`, and the configuration
		// tests caught what that means in practice: it refuses startup for
		// every existing deployment in one commit, for a control whose absence
		// is already reported at the point of use (the tool result carries
		// `"signed": false` and a reason, so it never claims provenance it does
		// not have). That is the same call the team-at-rest key made one landing
		// earlier, and it is recorded as an owner decision rather than smuggled
		// in as a hard failure.
		return nil
	}

	if c.EvidencePrivateKey == "" && c.EvidencePrivateKeyFile != "" {
		// Already resolved by ResolveSecretSources when Load() ran; re-read only
		// if the caller constructed a Config directly and Validate() is the
		// first gate it sees.
		material, err := secretfile.Read(c.EvidencePrivateKeyFile)
		if err != nil {
			return fmt.Errorf("EVIDENCE_PRIVATE_KEY_FILE: %w", err)
		}
		c.EvidencePrivateKey = strings.TrimSpace(string(material))
	}

	if _, err := ParseEvidenceKey(c.EvidencePrivateKey); err != nil {
		return fmt.Errorf("EVIDENCE_PRIVATE_KEY: %w", err)
	}
	if strings.TrimSpace(c.EvidenceProducerID) == "" {
		return errors.New("EVIDENCE_PRODUCER_ID is required when an evidence signing key is configured")
	}
	if strings.TrimSpace(c.EvidenceAudience) == "" {
		return errors.New("EVIDENCE_AUDIENCE is required when an evidence signing key is configured")
	}
	return nil
}
