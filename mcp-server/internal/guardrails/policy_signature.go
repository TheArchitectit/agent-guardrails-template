package guardrails

// Spec 07 R07-4.3.4 and R07-4.3.5 — pack signatures and the trust-state record.
//
// R07-4.3.4: "If signature verification is enabled, the trusted-key set MUST be
// supplied out of band. A signature failure MUST block resolution. A valid
// signature MUST NOT be treated as semantic approval."
//
// R07-4.3.5: "The consumer MUST record whether a resolved pack is verified,
// merely pinned, or locally modified; these states MUST not be conflated."
//
// Both were absent: the lock record proves the bytes on disk are the bytes that
// were reviewed, and nothing more. A lock is written by the same pipeline that
// reads it, so it protects against accidental drift, not against an author who
// re-locks. And "the digest matched" was the only provenance fact the server
// could state — which means an operator could not tell a reviewed, pinned set
// from a signed one, and a failure to verify would have looked like a plain
// digest mismatch.
//
// Three design points carry the requirements:
//
//  1. Trust arrives out of band. The key set is read from a restricted file the
//     operator provisions; it is NEVER taken from the pack, from the signature
//     file, or from the lock. A key that travels with the artifact it verifies
//     proves only that someone could write both.
//  2. Verification is all-or-nothing per set. With a trust set configured, a
//     pack with no signature, an unknown key id, a malformed signature or a
//     signature that does not match the bytes is a RESOLUTION ERROR. There is no
//     "warn and continue": a set that half-verifies is a set whose provenance
//     cannot be stated.
//  3. Verified is not approved. The signature says who wrote the bytes. It says
//     nothing about whether the rules are correct, permitted or reviewed — that
//     is reproducibility (the lock) and human review (the exception ledger).
//     The trust state is reported as a separate field precisely so it cannot be
//     read as "this pack is good".

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/thearchitectit/guardrail-mcp/internal/evidence"
	"github.com/thearchitectit/guardrail-mcp/internal/secretfile"
)

// TrustState is the R07-4.3.5 provenance class of one pack. The three values
// are mutually exclusive and are always reported explicitly: a missing field
// would be read as "nothing to see", which is the conflation the requirement
// forbids.
type TrustState string

const (
	// TrustVerified: a signature over this pack's exact bytes verified against
	// a key from the out-of-band trust set.
	TrustVerified TrustState = "verified"
	// TrustPinned: the bytes match the reviewed lock record, and no signature
	// was verified. This is the state of every pack in a deployment that has
	// not enabled signature verification — honest, and weaker than verified.
	TrustPinned TrustState = "pinned"
	// TrustLocallyModified: the bytes on disk are not the bytes the lock (and,
	// where present, the signature) attest to. Resolution refuses; the state is
	// named so an operator is not left reading a bare digest mismatch and
	// guessing whether the pack was edited or the lock was stale.
	TrustLocallyModified TrustState = "locally_modified"
)

// PackTrust is one pack's provenance record, reported on the diagnostic
// surface and carried in resolution errors.
type PackTrust struct {
	Path   string     `json:"path"`
	Pack   string     `json:"pack"`
	State  TrustState `json:"state"`
	KeyID  string     `json:"key_id,omitempty"`
	Reason string     `json:"reason,omitempty"`
}

// packSignatureDomain separates a policy-pack signature from every other
// signature this process produces. Without it a signature over a stored
// evidence record could be presented as a signature over a policy pack when the
// bytes happened to coincide.
const packSignatureDomain = "guardrail-policy-pack-v1\x00"

// TrustedKeySet is the out-of-band set of keys permitted to sign policy packs.
// It is immutable once built: a mutation in place would let a concurrent
// resolver read half a change.
type TrustedKeySet struct {
	byKeyID map[string]ed25519.PublicKey
	order   []string
}

// Len reports the number of trusted keys. Non-vacuity is asserted by the
// loader: a trust set with no keys would reject every pack for the wrong
// reason and is refused rather than treated as "no signatures required".
func (t *TrustedKeySet) Len() int {
	if t == nil {
		return 0
	}
	return len(t.byKeyID)
}

// KeyIDs returns the trusted key ids in a stable order (diagnostics and
// operator output must not depend on map iteration).
func (t *TrustedKeySet) KeyIDs() []string {
	if t == nil {
		return nil
	}
	out := make([]string, len(t.order))
	copy(out, t.order)
	return out
}

// Lookup returns the trusted public key for a key id.
func (t *TrustedKeySet) Lookup(keyID string) (ed25519.PublicKey, bool) {
	if t == nil {
		return nil, false
	}
	key, ok := t.byKeyID[keyID]
	return key, ok
}

// trustedKeyFile is the on-disk out-of-band key set.
//
// key_id is optional and, when present, must equal the id derived from the
// public key. Accepting a declared id that disagrees with its key would let one
// record claim two identities.
type trustedKeyFile struct {
	Keys []trustedKeyRecord `json:"keys"`
}

type trustedKeyRecord struct {
	KeyID     string `json:"key_id,omitempty"`
	PublicKey string `json:"public_key"`
	// Expires is optional RFC3339. A past expiry is refused at load: an expired
	// trust record is an unbounded grant that reads as diligence.
	Expires string `json:"expires,omitempty"`
}

// LoadTrustedKeySet reads the out-of-band trust set from a restricted file
// (Spec 17 R17-06: regular file, no group/world access).
//
// The path is operator-supplied configuration, never a value from a pack, a
// signature file or a lock record — that is what "out of band" means here.
func LoadTrustedKeySet(path string) (*TrustedKeySet, error) {
	raw, err := secretfile.Read(path)
	if err != nil {
		return nil, fmt.Errorf("policy pack trusted key set: %w", err)
	}
	return ParseTrustedKeySet(raw)
}

// ParseTrustedKeySet validates and indexes a trust-set document. It is separate
// from the file read so it can be tested without a filesystem, and so the same
// validation can be applied to a value an operator pastes for diagnosis.
func ParseTrustedKeySet(raw []byte) (*TrustedKeySet, error) {
	var doc trustedKeyFile
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("policy pack trusted key set is not a recognised document: %w", err)
	}
	if len(doc.Keys) == 0 {
		// Non-vacuity: an empty trust set would make every pack fail
		// verification, which reads as "the packs are unsigned" rather than
		// "the trust set was not provisioned".
		return nil, fmt.Errorf("policy pack trusted key set contains no keys; refusing an empty trust set")
	}

	set := &TrustedKeySet{byKeyID: make(map[string]ed25519.PublicKey, len(doc.Keys))}
	for i, rec := range doc.Keys {
		material := strings.TrimSpace(rec.PublicKey)
		if material == "" {
			return nil, fmt.Errorf("trusted key record %d has no public_key", i)
		}
		decoded, err := base64.StdEncoding.DecodeString(material)
		if err != nil {
			return nil, fmt.Errorf("trusted key record %d public_key is not base64", i)
		}
		if len(decoded) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("trusted key record %d key is %d bytes; want %d (Ed25519 public key)",
				i, len(decoded), ed25519.PublicKeySize)
		}
		pub := ed25519.PublicKey(decoded)
		// The id is DERIVED, never taken on trust. evidence.KeyIDFor is the same
		// construction the cross-product contract uses, so one key has one id
		// across the program rather than one per subsystem.
		id := evidence.KeyIDFor(pub)
		if declared := strings.TrimSpace(rec.KeyID); declared != "" && declared != id {
			return nil, fmt.Errorf("trusted key record %d declares key_id %q but its public key derives %q", i, declared, id)
		}
		if _, dup := set.byKeyID[id]; dup {
			return nil, fmt.Errorf("duplicate trusted key id %s (two records share one public key)", id)
		}
		if strings.TrimSpace(rec.Expires) != "" {
			expiry, perr := time.Parse(time.RFC3339, rec.Expires)
			if perr != nil {
				return nil, fmt.Errorf("trusted key record %d expires is not RFC3339: %w", i, perr)
			}
			if !expiry.After(time.Now().UTC()) {
				return nil, fmt.Errorf("trusted key record %d (%s) expired at %s; an expired trust record must be replaced, not used",
					i, id, rec.Expires)
			}
		}
		set.byKeyID[id] = pub
		set.order = append(set.order, id)
	}
	sort.Strings(set.order)
	return set, nil
}

// packSignatureFile is the on-disk detached signature for one pack.
type packSignatureFile struct {
	KeyID     string `json:"key_id"`
	Signature string `json:"signature"`
}

// verifyPackTrust decides the R07-4.3.5 state of one pack.
//
// When no trust set is configured, the pack is `pinned`: the lock proves its
// bytes were reviewed, and nothing claims who wrote them. When a trust set IS
// configured, every pack must verify — absent, unknown-key, malformed and
// mismatched signatures all return an error, because a set that verifies
// partially cannot state its own provenance.
func verifyPackTrust(packPath, rel string, raw []byte, set *TrustedKeySet) (PackTrust, error) {
	trust := PackTrust{Path: rel, Pack: strings.TrimSuffix(rel, ".pack.json")}

	sigPath := packPath + ".sig"
	sigRaw, sigErr := os.ReadFile(sigPath)
	if sigErr != nil {
		if !os.IsNotExist(sigErr) {
			return trust, fmt.Errorf("policy pack %q signature file: %w", rel, sigErr)
		}
		if set == nil {
			trust.State = TrustPinned
			trust.Reason = "no signature and no trusted key set configured; the lock record is the only provenance"
			return trust, nil
		}
		trust.State = TrustLocallyModified
		return trust, fmt.Errorf("policy pack %q has no signature but signature verification is enabled (R07-4.3.4); state=%s",
			rel, TrustLocallyModified)
	}

	var sig packSignatureFile
	dec := json.NewDecoder(strings.NewReader(string(sigRaw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&sig); err != nil {
		return trust, fmt.Errorf("policy pack %q signature file is not a recognised signature: %w", rel, err)
	}
	if strings.TrimSpace(sig.KeyID) == "" || strings.TrimSpace(sig.Signature) == "" {
		return trust, fmt.Errorf("policy pack %q signature file is missing key_id or signature", rel)
	}

	if set == nil {
		// A signature exists but nothing was provisioned to check it against.
		// Reporting `pinned` (not `verified`) is the whole point of R07-4.3.5: a
		// signature nobody verified is not evidence of anything.
		//
		// KeyID is deliberately NOT populated here. A key id beside state=pinned
		// reads as "signed by this key", which is precisely the conflation the
		// requirement forbids; the claimed id is named in the reason instead,
		// where it is explicitly unverified text and not an identity claim.
		trust.State = TrustPinned
		trust.Reason = fmt.Sprintf("unverified signature present, claimed key %q, but no trusted key set is configured", sig.KeyID)
		return trust, nil
	}

	trust.KeyID = sig.KeyID

	pub, ok := set.Lookup(sig.KeyID)
	if !ok {
		return trust, fmt.Errorf("policy pack %q is signed by key %q which is not in the out-of-band trusted key set (R07-4.3.4); state=%s",
			rel, sig.KeyID, TrustLocallyModified)
	}

	if err := verifyDetachedPackSignature(pub, raw, sig.Signature); err != nil {
		trust.State = TrustLocallyModified
		return trust, fmt.Errorf("policy pack %q signature does not verify (%v); state=%s", rel, err, TrustLocallyModified)
	}

	trust.State = TrustVerified
	return trust, nil
}

// verifyDetachedPackSignature checks one Ed25519 signature over the exact pack
// bytes. Domain separated so a signature from any other artifact contract is
// not valid here even if the bytes coincide.
func verifyDetachedPackSignature(pub ed25519.PublicKey, payload []byte, signatureB64 string) error {
	if len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("verification key is not an Ed25519 public key")
	}
	if len(payload) == 0 {
		return fmt.Errorf("refusing to verify an empty payload")
	}
	sig, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("signature is not a valid Ed25519 detached signature")
	}
	if !ed25519.Verify(pub, packSignatureMessage(payload), sig) {
		return fmt.Errorf("signature does not match the pack bytes")
	}
	return nil
}

func packSignatureMessage(payload []byte) []byte {
	h := sha256.Sum256(payload)
	out := make([]byte, 0, len(packSignatureDomain)+len(h))
	out = append(out, packSignatureDomain...)
	out = append(out, h[:]...)
	return out
}

// SignPolicyPack produces the detached signature file content for a pack. It
// exists for the same reason the credential issuer exists: an operator who must
// compute a signature by hand will compute it over the wrong bytes.
//
// It is deliberately NOT used by the server at startup — the running process
// only verifies. A process that can sign its own policy can re-author it.
func SignPolicyPack(private ed25519.PrivateKey, raw []byte) ([]byte, error) {
	if len(private) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("signing key must be an Ed25519 private key (%d bytes), got %d",
			ed25519.PrivateKeySize, len(private))
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("refusing to sign an empty pack")
	}
	pub, ok := private.Public().(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("signing key has no usable public half")
	}
	doc := packSignatureFile{
		KeyID:     evidence.KeyIDFor(pub),
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, packSignatureMessage(raw))),
	}
	return json.MarshalIndent(doc, "", "  ")
}

// packSignaturePath is the signature location for a pack: adjacent, with a
// suffix. Adjacency is deliberate — a signature stored elsewhere can be paired
// with the wrong pack by a path typo, and the pairing would then look valid.
func packSignaturePath(packPath string) string {
	return filepath.Join(filepath.Dir(packPath), filepath.Base(packPath)+".sig")
}
