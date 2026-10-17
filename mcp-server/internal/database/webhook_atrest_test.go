package database

import (
	"database/sql"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/thearchitectit/guardrail-mcp/internal/notifications"
	"github.com/thearchitectit/guardrail-mcp/internal/secretbox"
)

// fakeRowScanner fills the webhookColumns destinations with canned values, so
// the at-rest encryption decision can be exercised without a live PostgreSQL.
type fakeRowScanner struct {
	id        string
	teamID    string
	url       string
	events    []string
	secret    string
	previous  sql.NullString
	rotatedAt sql.NullTime
	enabled   bool
	createdAt time.Time
	updatedAt time.Time
}

func (f *fakeRowScanner) Scan(dest ...any) error {
	if len(dest) != 10 {
		return sql.ErrNoRows
	}
	*dest[0].(*string) = f.id
	*dest[1].(*string) = f.teamID
	*dest[2].(*string) = f.url
	*(dest[3].(*pq.StringArray)) = f.events
	*dest[4].(*string) = f.secret
	*dest[5].(*sql.NullString) = f.previous
	*dest[6].(*sql.NullTime) = f.rotatedAt
	*dest[7].(*bool) = f.enabled
	*dest[8].(*time.Time) = f.createdAt
	*dest[9].(*time.Time) = f.updatedAt
	return nil
}

const atRestCanary = "webhook-signing-secret-atrest-canary-01"

// TestWebhookScanDecryptsSealedSigningMaterial proves a sealed value in the
// `secret_hmac` column is decrypted on read, so the dispatcher signs with the
// original key while the stored form stays ciphertext (Spec 17 R17-06).
func TestWebhookScanDecryptsSealedSigningMaterial(t *testing.T) {
	sealer, err := secretbox.New("operator-webhook-key-01")
	if err != nil {
		t.Fatalf("secretbox.New: %v", err)
	}
	stored, err := sealer.Seal(atRestCanary)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if stored == atRestCanary {
		t.Fatal("precondition: the stored value must be ciphertext")
	}

	store := NewWebhookStore(nil).WithSealer(sealer)
	cfg, err := store.scanWebhookConfig(&fakeRowScanner{secret: stored, enabled: true})
	if err != nil {
		t.Fatalf("scanWebhookConfig: %v", err)
	}
	if cfg.SecretHMAC != atRestCanary {
		t.Fatalf("sealed signing key must decrypt on read: got %q", cfg.SecretHMAC)
	}
}

// TestWebhookScanReadsLegacyPlaintextSigningMaterial proves a pre-existing
// plaintext row keeps working during migration.
func TestWebhookScanReadsLegacyPlaintextSigningMaterial(t *testing.T) {
	sealer, _ := secretbox.New("operator-webhook-key-01")
	store := NewWebhookStore(nil).WithSealer(sealer)
	cfg, err := store.scanWebhookConfig(&fakeRowScanner{secret: atRestCanary, enabled: true})
	if err != nil {
		t.Fatalf("scanWebhookConfig: %v", err)
	}
	if cfg.SecretHMAC != atRestCanary {
		t.Fatalf("legacy plaintext signing key must pass through: got %q", cfg.SecretHMAC)
	}
}

// TestWebhookScanDecryptsSealedPreviousKey proves the rotation-overlap key is
// decrypted too, so the overlap signature uses the real outgoing key.
func TestWebhookScanDecryptsSealedPreviousKey(t *testing.T) {
	sealer, _ := secretbox.New("operator-webhook-key-01")
	const previous = "webhook-signing-secret-previous-01"
	storedPrev, _ := sealer.Seal(previous)
	storedActive, _ := sealer.Seal(atRestCanary)
	now := time.Now().UTC()

	store := NewWebhookStore(nil).WithSealer(sealer)
	cfg, err := store.scanWebhookConfig(&fakeRowScanner{
		secret:    storedActive,
		previous:  sql.NullString{String: storedPrev, Valid: true},
		rotatedAt: sql.NullTime{Time: now, Valid: true},
		enabled:   true,
	})
	if err != nil {
		t.Fatalf("scanWebhookConfig: %v", err)
	}
	if cfg.SecretHMACPrevious != previous {
		t.Fatalf("sealed previous key must decrypt on read: got %q", cfg.SecretHMACPrevious)
	}
}

// TestWebhookScanRefusesSealedValueWithoutKey is the fail-closed control: a
// sealed row read by a store with no encryption key must be refused, never
// silently treated as the signing key.
func TestWebhookScanRefusesSealedValueWithoutKey(t *testing.T) {
	sealer, _ := secretbox.New("operator-webhook-key-01")
	stored, _ := sealer.Seal(atRestCanary)

	store := NewWebhookStore(nil) // no sealer
	if _, err := store.scanWebhookConfig(&fakeRowScanner{secret: stored, enabled: true}); err == nil {
		t.Fatal("a sealed value with no configured key must be refused, not used as a signing key")
	}
}

// TestWebhookScanRefusesTamperedSealedValue proves a corrupt stored key never
// becomes a usable signing key.
func TestWebhookScanRefusesTamperedSealedValue(t *testing.T) {
	sealer, _ := secretbox.New("operator-webhook-key-01")
	stored, _ := sealer.Seal(atRestCanary)
	body := []byte(stored)
	body[len("v1:")+2] ^= 0x01

	store := NewWebhookStore(nil).WithSealer(sealer)
	if _, err := store.scanWebhookConfig(&fakeRowScanner{secret: string(body), enabled: true}); err == nil {
		t.Fatal("a tampered sealed value must fail authentication, not decrypt")
	}
}

// TestWebhookSealSecretIsCiphertext proves the write path seals: the value that
// reaches the database column is not the plaintext signing key.
func TestWebhookSealSecretIsCiphertext(t *testing.T) {
	sealer, _ := secretbox.New("operator-webhook-key-01")
	store := NewWebhookStore(nil).WithSealer(sealer)
	stored, err := store.sealSecret(atRestCanary)
	if err != nil {
		t.Fatalf("sealSecret: %v", err)
	}
	if stored == atRestCanary {
		t.Fatal("sealSecret must not return the plaintext")
	}
	if !secretbox.IsSealed(stored) {
		t.Fatalf("sealed value must carry the versioned prefix: %q", stored)
	}
	// Round-trip: what the write path stores, the read path recovers.
	plaintext, sealed, err := sealer.Open(stored)
	if err != nil || !sealed || plaintext != atRestCanary {
		t.Fatalf("write/read round trip failed: got %q sealed=%v err=%v", plaintext, sealed, err)
	}
}

// TestWebhookSealSecretPassthroughWithoutSealer proves the local posture (no
// key) stores the value unchanged rather than erroring.
func TestWebhookSealSecretPassthroughWithoutSealer(t *testing.T) {
	store := NewWebhookStore(nil)
	stored, err := store.sealSecret(atRestCanary)
	if err != nil {
		t.Fatalf("sealSecret without sealer: %v", err)
	}
	if stored != atRestCanary {
		t.Fatalf("without a sealer the value must pass through, got %q", stored)
	}
}

// TestWebhookRotationEventFingerprintUsesPlaintext pins that the rotation
// event identifies keys by the plaintext fingerprint (matching the receiver
// contract) even when the stored value is ciphertext.
func TestWebhookRotationEventFingerprintUsesPlaintext(t *testing.T) {
	sealer, _ := secretbox.New("operator-webhook-key-01")
	store := NewWebhookStore(nil).WithSealer(sealer)
	stored, _ := store.sealSecret(atRestCanary)
	plaintext, err := store.openSecret(stored)
	if err != nil {
		t.Fatalf("openSecret: %v", err)
	}
	if got, want := notifications.WebhookKeyFingerprint(plaintext), notifications.WebhookKeyFingerprint(atRestCanary); got != want {
		t.Fatalf("fingerprint must be over the plaintext key: got %s want %s", got, want)
	}
}
