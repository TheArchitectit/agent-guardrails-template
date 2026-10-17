package audit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/thearchitectit/guardrail-mcp/internal/auth"
	"github.com/thearchitectit/guardrail-mcp/internal/database"
	"github.com/thearchitectit/guardrail-mcp/internal/notifications"
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

	// A deliberately low-entropy synthetic fingerprint: a real fingerprint is 12
	// hex characters, but a high-entropy fixture literal trips the generic-api-key
	// secret scanner (the cause is the fixture, not the scan). The contract under
	// test is that the field is persisted and echoed, not its entropy.
	l.LogRotation(context.Background(), auth.RotationEvent{
		Operation:      "rotate",
		KeyFingerprint: "aaaa0000bbbb",
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
	if ev.Details["key_fingerprint"] != "aaaa0000bbbb" {
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
		KeyFingerprint: "cccc1111dddd",
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

// TestLogWebhookRotationRecordsOpaqueEvent covers Spec 17 R17-10 at the
// audit-logger boundary for the webhook half: an applied webhook signing-key
// rotation is persisted as a critical event whose details identify the active
// key by fingerprint and record the bounded overlap — never a signing secret.
func TestLogWebhookRotationRecordsOpaqueEvent(t *testing.T) {
	store := &captureStore{}
	l := NewLoggerWithStore(8, store)
	defer l.Stop()

	l.LogWebhookRotation(context.Background(), notifications.WebhookRotationEvent{
		WebhookID:      "wh-1",
		Operation:      "rotate",
		KeyID:          "eeee2222ffff",
		PreviousKeyID:  "1111aaaa2222",
		OverlapSeconds: 86400,
		OccurredAt:     time.Now().UTC(),
	})

	ev := store.waitFor(t, string(EventCredentialRotation))
	if ev == nil {
		t.Fatal("expected the webhook rotation event to be persisted")
	}
	if ev.Actor != "system" {
		t.Fatalf("actor = %q, want system", ev.Actor)
	}
	if ev.Severity != string(SevCritical) {
		t.Fatalf("severity = %q, want critical", ev.Severity)
	}
	if ev.Resource != "webhook_signing_key" {
		t.Fatalf("resource = %q, want webhook_signing_key", ev.Resource)
	}
	if ev.Details["operation"] != "rotate" {
		t.Fatalf("operation = %v", ev.Details["operation"])
	}
	if ev.Details["webhook_id"] != "wh-1" {
		t.Fatalf("webhook_id = %v", ev.Details["webhook_id"])
	}
	if ev.Details["key_fingerprint"] != "eeee2222ffff" {
		t.Fatalf("key_fingerprint = %v", ev.Details["key_fingerprint"])
	}
	if ev.Details["previous_key_id"] != "1111aaaa2222" {
		t.Fatalf("previous_key_id = %v", ev.Details["previous_key_id"])
	}
	if ev.Details["overlap_seconds"] != 86400 {
		t.Fatalf("overlap_seconds = %v", ev.Details["overlap_seconds"])
	}
}

// TestLogWebhookRotationDetailsCarryNoKeyMaterial is the opacity guard for the
// webhook rotation event: the persisted details map must hold only the opaque
// fields, never a signing secret.
func TestLogWebhookRotationDetailsCarryNoKeyMaterial(t *testing.T) {
	store := &captureStore{}
	l := NewLoggerWithStore(8, store)
	defer l.Stop()

	l.LogWebhookRotation(context.Background(), notifications.WebhookRotationEvent{
		WebhookID: "wh-2",
		Operation: "cutover",
		KeyID:     "3333bbbb4444",
	})

	ev := store.waitFor(t, string(EventCredentialRotation))
	if ev == nil {
		t.Fatal("expected the webhook rotation event to be persisted")
	}
	allowed := map[string]bool{
		"operation": true, "webhook_id": true, "key_fingerprint": true,
		"previous_key_id": true, "overlap_seconds": true,
	}
	for k := range ev.Details {
		if !allowed[k] {
			t.Fatalf("webhook rotation details carry an unexpected field %q (the event must stay opaque)", k)
		}
	}
}
