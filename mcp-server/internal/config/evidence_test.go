package config

// Spec 15 gr-xp-02 configuration evidence: the instance signing key is parsed
// strictly and refused when unusable. These tests drive the real parser; a
// parser that only "accepts a string" is how a wrong-algorithm key reaches the
// first signature with a confusing error at a peer instead of at startup.

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseEvidenceKeyAcceptsPKCS8PEM(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	got, err := ParseEvidenceKey(string(pemBytes))
	if err != nil {
		t.Fatalf("ParseEvidenceKey: %v", err)
	}
	if !got.Public().(ed25519.PublicKey).Equal(pub) {
		t.Fatal("parsed key does not match the generated public half")
	}
}

func TestParseEvidenceKeyAcceptsRawSeedAndFullKey(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	seed := priv.Seed()
	got, err := ParseEvidenceKey(base64.StdEncoding.EncodeToString(seed))
	if err != nil {
		t.Fatalf("seed form: %v", err)
	}
	if !got.Equal(priv) {
		t.Fatal("32-byte seed form produced a different key")
	}

	got2, err := ParseEvidenceKey(base64.StdEncoding.EncodeToString(priv))
	if err != nil {
		t.Fatalf("64-byte form: %v", err)
	}
	if !got2.Equal(priv) {
		t.Fatal("64-byte form produced a different key")
	}
}

// A wrong-algorithm key must be refused at the load boundary, naming Ed25519.
func TestParseEvidenceKeyRefusesRSA(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(rsaKey)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	_, err = ParseEvidenceKey(string(pemBytes))
	if err == nil {
		t.Fatal("RSA key accepted as an Ed25519 evidence identity")
	}
	if !strings.Contains(err.Error(), "Ed25519") {
		t.Fatalf("refusal does not name the required algorithm: %v", err)
	}
}

func TestParseEvidenceKeyRefusesGarbage(t *testing.T) {
	cases := map[string]string{
		"empty":           "",
		"not base64":      "!!!not-base64!!!",
		"wrong length":    base64.StdEncoding.EncodeToString(make([]byte, 17)),
		"encrypted pem":   "-----BEGIN ENCRYPTED PRIVATE KEY-----\nAAAA\n-----END ENCRYPTED PRIVATE KEY-----",
		"trailing data":   "-----BEGIN PRIVATE KEY-----\nAAAA\n-----END PRIVATE KEY-----\nextra",
		"public key type": "-----BEGIN PUBLIC KEY-----\nAAAA\n-----END PUBLIC KEY-----",
	}
	for name, value := range cases {
		if _, err := ParseEvidenceKey(value); err == nil {
			t.Fatalf("%s: accepted", name)
		}
	}
}

// A configured key with no producer ID or audience cannot produce an
// addressable artifact, so all three are required together.
func TestValidateEvidenceIdentityRequiresAddressing(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	encoded := base64.StdEncoding.EncodeToString(priv.Seed())

	cfg := &Config{EvidencePrivateKey: encoded}
	if err := cfg.ValidateEvidenceIdentity(); err == nil {
		t.Fatal("key without producer_id/audience accepted")
	}

	cfg = &Config{EvidencePrivateKey: encoded, EvidenceProducerID: "guardrails"}
	if err := cfg.ValidateEvidenceIdentity(); err == nil {
		t.Fatal("key without audience accepted")
	}

	cfg = &Config{EvidencePrivateKey: encoded, EvidenceProducerID: "guardrails", EvidenceAudience: "devgate"}
	if err := cfg.ValidateEvidenceIdentity(); err != nil {
		t.Fatalf("complete identity refused: %v", err)
	}
}

// An unset key is permitted (reported at the point of use, not refused at
// startup). This is the deliberate narrowing recorded in the gate's docstring:
// making it mandatory would refuse startup for every existing deployment.
func TestValidateEvidenceIdentityAllowsUnset(t *testing.T) {
	for _, profile := range []string{"local", "tailnet", "public"} {
		cfg := &Config{DeploymentProfile: profile}
		if err := cfg.ValidateEvidenceIdentity(); err != nil {
			t.Fatalf("profile %s refused an unset key: %v", profile, err)
		}
	}
}

// The key accepts a read-only secret file, and a loose-mode file is refused by
// the shared secretfile rule (the same rule every other secret uses).
func TestEvidenceKeyFileSource(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "evidence.key")
	if err := os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(priv.Seed())), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg := &Config{EvidencePrivateKeyFile: path, EvidenceProducerID: "guardrails", EvidenceAudience: "devgate"}
	if err := cfg.ValidateEvidenceIdentity(); err != nil {
		t.Fatalf("0600 key file refused: %v", err)
	}

	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	loose := &Config{EvidencePrivateKeyFile: path, EvidenceProducerID: "guardrails", EvidenceAudience: "devgate"}
	if err := loose.ValidateEvidenceIdentity(); err == nil {
		t.Fatal("0644 key file accepted")
	}
}

// The private key is masked in any config dump; the addressing fields are not.
func TestMaskedHidesEvidenceKeyOnly(t *testing.T) {
	cfg := &Config{
		EvidencePrivateKey:     "SECRET-VALUE",
		EvidencePrivateKeyFile: "/run/secrets/evidence.key",
		EvidenceProducerID:     "guardrails",
		EvidenceAudience:       "devgate",
	}
	masked := cfg.Masked()
	if masked.EvidencePrivateKey != "***" {
		t.Fatal("evidence private key is not masked")
	}
	if masked.EvidencePrivateKeyFile != "***" {
		t.Fatal("evidence key file path is not masked")
	}
	if masked.EvidenceProducerID != "guardrails" || masked.EvidenceAudience != "devgate" {
		t.Fatal("public addressing data was masked")
	}
}
