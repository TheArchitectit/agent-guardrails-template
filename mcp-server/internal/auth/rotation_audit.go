package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"time"
)

// Rotation audit (Spec 16 R16-09 "operator audit", Spec 17 R17-10 "log opaque
// rotation events").
//
// R16-09 requires the credential lifecycle to define "atomic reload or
// immutable deployment replacement, verifier-key rotation and operator audit".
// R17-10 requires rotation to "log opaque rotation events". Before this file
// the registry swapped its whole snapshot on every Reload and every
// RotateVerifierKey with no record at all: a rotation or a revocation reload
// left no attributable trace, so an operator could neither confirm that a
// rotation was applied nor reconstruct when it happened. The reload loop
// logged only *failures*.
//
// The event below is deliberately opaque: the verifier key, any credential
// secret and any verifier digest are never present. A key is identified only
// by a non-reversible fingerprint, which is what lets an operator correlate
// rotation events (R17-10: "identify the verifying key") without the log
// becoming a credential store.

// RotationEvent describes one applied credential-registry snapshot
// replacement. It carries no key material and no secret.
type RotationEvent struct {
	// Operation is "reload" (a source re-read) or "rotate" (a verifier-key
	// rotation).
	Operation string
	// KeyFingerprint is a non-reversible identifier of the verifier key in
	// force after the replacement. It is a fingerprint, not a credential: it
	// cannot be used to verify or mint anything.
	KeyFingerprint string
	// RecordCount is the number of credential records in the new snapshot.
	RecordCount int
	// RevokedCount is how many of those records are marked revoked.
	RevokedCount int
	// OverlapSeconds is the outgoing-key overlap window (0 for a reload).
	OverlapSeconds int
	// OccurredAt is when the replacement was applied (UTC).
	OccurredAt time.Time
}

// SetRotationObserver installs the callback invoked after every *applied*
// snapshot replacement that actually changed registry state. It is called from
// the rotation path without holding the registry lock, so the callback must not
// call back into the registry. A nil observer disables auditing.
//
// A replacement that leaves the state byte-for-byte identical (for example the
// reload loop re-reading an unchanged source) does NOT fire: a rotation event
// records a change, and emitting a heartbeat would make the log useless for
// reconstructing rotations.
func (r *Registry) SetRotationObserver(observer func(RotationEvent)) {
	if r == nil {
		return
	}
	r.obsMu.Lock()
	r.observer = observer
	r.obsMu.Unlock()
}

// keyFingerprint returns a short, non-reversible identifier for a verifier
// key. Domain separation keeps it from colliding with any other digest in the
// system; the result is truncated because it is an identifier, not an
// authentication artifact.
func keyFingerprint(key string) string {
	sum := sha256.Sum256([]byte("guardrail-verifier-key-fingerprint:" + key))
	return hex.EncodeToString(sum[:])[:12]
}

// rotationEvent builds the opaque event for an applied replacement.
func rotationEvent(operation, verifierKey string, snap *snapshot, overlap time.Duration, now time.Time) RotationEvent {
	ev := RotationEvent{
		Operation:      operation,
		KeyFingerprint: keyFingerprint(verifierKey),
		OccurredAt:     now.UTC(),
	}
	if overlap > 0 {
		ev.OverlapSeconds = int(overlap / time.Second)
	}
	if snap != nil {
		ev.RecordCount = len(snap.records)
		for i := range snap.records {
			if snap.records[i].Revoked {
				ev.RevokedCount++
			}
		}
	}
	return ev
}

// notifyRotation delivers an applied-replacement event to the observer, if one
// is installed. It is a no-op when no observer is set.
func (r *Registry) notifyRotation(ev RotationEvent) {
	if r == nil {
		return
	}
	r.obsMu.RLock()
	observer := r.observer
	r.obsMu.RUnlock()
	if observer != nil {
		observer(ev)
	}
}

// snapshotChanged reports whether a replacement materially changes registry
// state. Two snapshots are equivalent when they hold the same records (same
// fields, including revoked/expiry) and the same key set with the same
// validity windows.
func snapshotChanged(prev, next *snapshot) bool {
	if prev == nil || next == nil {
		return prev != next
	}
	if len(prev.keys) != len(next.keys) || len(prev.records) != len(next.records) {
		return true
	}
	for i := range prev.keys {
		if prev.keys[i].validUntil != next.keys[i].validUntil {
			return true
		}
		if string(prev.keys[i].key) != string(next.keys[i].key) {
			return true
		}
	}
	return !reflect.DeepEqual(prev.records, next.records)
}
