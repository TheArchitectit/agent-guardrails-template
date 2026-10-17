package audit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/thearchitectit/guardrail-mcp/internal/auth"
	"github.com/thearchitectit/guardrail-mcp/internal/database"
)

// captureStore records every persisted audit event. The rotation event goes
// through the async worker while the flush decision is written synchronously,
// so the test must inspect the whole set rather than "the last" write.
type captureStore struct {
	mu     sync.Mutex
	events []*database.AuditEvent
}

func (s *captureStore) Insert(ctx context.Context, event *database.AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *event
	s.events = append(s.events, &cp)
	return nil
}

func (s *captureStore) byType(t string) *database.AuditEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.events) - 1; i >= 0; i-- {
		if s.events[i].EventType == t {
			return s.events[i]
		}
	}
	return nil
}

// waitFor polls the store until an event of the given type is persisted or the
// deadline elapses. The rotation event travels the async worker channel, so a
// bare assertion after Log() would race the worker goroutine.
func (s *captureStore) waitFor(t *testing.T, eventType string) *database.AuditEvent {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if ev := s.byType(eventType); ev != nil {
			return ev
		}
		time.Sleep(5 * time.Millisecond)
	}
	return nil
}

// TestLogRotationRecordsOpaqueEvent covers Spec 16 R16-09 (operator audit) /
// Spec 17 R17-10 (log opaque rotation events) at the audit-logger boundary: an
// applied registry replacement is persisted as a critical event whose details
// identify the operation and the verifier key by fingerprint.
func TestLogRotationRecordsOpaqueEvent(t *testing.T) {
	store := &captureStore{}
	l := NewLoggerWithStore(8, store)
	defer l.Stop()

	l.LogRotation(context.Background(), auth.RotationEvent{
		Operation:      "rotate",
		KeyFingerprint: "abc123def456",
		RecordCount:    4,
		RevokedCount:   1,
		OverlapSeconds: 3600,
		OccurredAt:     time.Now().UTC(),
	})

	ev := store.waitFor(t, string(EventCredentialRotation))
	if ev == nil {
		t.Fatal("expected the rotation event to be persisted")
	}
	if ev.Actor != "system" {
		t.Fatalf("actor = %q, want system", ev.Actor)
	}
	if ev.Severity != string(SevCritical) {
		t.Fatalf("severity = %q, want critical", ev.Severity)
	}
	if ev.Details["operation"] != "rotate" {
		t.Fatalf("operation = %v", ev.Details["operation"])
	}
	if ev.Details["key_fingerprint"] != "abc123def456" {
		t.Fatalf("key_fingerprint = %v", ev.Details["key_fingerprint"])
	}
	if ev.Details["overlap_seconds"] != 3600 {
		t.Fatalf("overlap_seconds = %v", ev.Details["overlap_seconds"])
	}
	if ev.Details["record_count"] != 4 || ev.Details["revoked_count"] != 1 {
		t.Fatalf("counts = (%v, %v), want (4, 1)", ev.Details["record_count"], ev.Details["revoked_count"])
	}
}

// TestLogRotationDetailsCarryNoKeyMaterial is the opacity guard at the logger
// boundary: the persisted details map must hold only the opaque fields.
func TestLogRotationDetailsCarryNoKeyMaterial(t *testing.T) {
	store := &captureStore{}
	l := NewLoggerWithStore(8, store)
	defer l.Stop()

	l.LogRotation(context.Background(), auth.RotationEvent{
		Operation:      "reload",
		KeyFingerprint: "deadbeefcafe",
		RecordCount:    2,
	})

	ev := store.waitFor(t, string(EventCredentialRotation))
	if ev == nil {
		t.Fatal("expected the rotation event to be persisted")
	}
	allowed := map[string]bool{
		"operation": true, "key_fingerprint": true, "record_count": true,
		"revoked_count": true, "overlap_seconds": true,
	}
	for k := range ev.Details {
		if !allowed[k] {
			t.Fatalf("rotation details carry an unexpected field %q (a rotation event must stay opaque)", k)
		}
	}
}
