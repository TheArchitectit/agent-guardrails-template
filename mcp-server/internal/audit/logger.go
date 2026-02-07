package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// EventType represents categories of audit events
type EventType string

const (
	EventAuthSuccess    EventType = "auth_success"
	EventAuthFailure    EventType = "auth_failure"
	EventValidation     EventType = "validation"
	EventRuleChange     EventType = "rule_change"
	EventDocChange      EventType = "document_change"
	EventConfigChange   EventType = "config_change"
	EventAccessDenied   EventType = "access_denied"
	EventSessionCreated EventType = "session_created"
	EventSessionExpired EventType = "session_expired"
)

// Severity represents event severity
type Severity string

const (
	SevInfo     Severity = "info"
	SevWarning  Severity = "warning"
	SevCritical Severity = "critical"
)

// Event represents a single audit event
type Event struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Type      EventType              `json:"type"`
	Severity  Severity               `json:"severity"`
	Actor     string                 `json:"actor"`    // Hashed API key or user ID
	Action    string                 `json:"action"`   // What was done
	Resource  string                 `json:"resource"` // What was affected
	Status    string                 `json:"status"`   // success, failure
	Details   map[string]interface{} `json:"details"`  // Additional context
	ClientIP  string                 `json:"client_ip"`
	UserAgent string                 `json:"user_agent"`
	RequestID string                 `json:"request_id"`
}

// Logger handles audit event recording
type Logger struct {
	backend chan Event
}

// NewLogger creates an audit logger
func NewLogger(bufferSize int) *Logger {
	l := &Logger{
		backend: make(chan Event, bufferSize),
	}
	go l.process()
	return l
}

// Log records an audit event
func (l *Logger) Log(ctx context.Context, event Event) {
	event.ID = uuid.New().String()
	event.Timestamp = time.Now().UTC()

	// Extract request context
	if reqID := ctx.Value("request_id"); reqID != nil {
		event.RequestID = reqID.(string)
	}

	select {
	case l.backend <- event:
	default:
		// Buffer full - log to stderr and continue
		slog.Error("audit buffer full, dropping event", "type", event.Type)
	}
}

// process writes events to persistent storage
func (l *Logger) process() {
	for event := range l.backend {
		// Write to structured log (forward to SIEM if configured)
		data, _ := json.Marshal(event)
		slog.Info("AUDIT", "event", string(data))

		// TODO: Write to database for long-term storage
		// This enables querying audit history via Web UI
	}
}

// LogAuth logs authentication events
func (l *Logger) LogAuth(ctx context.Context, success bool, actor, reason string) {
	eventType := EventAuthSuccess
	severity := SevInfo
	if !success {
		eventType = EventAuthFailure
		severity = SevWarning
	}

	status := "success"
	if !success {
		status = "failure"
	}

	l.Log(ctx, Event{
		Type:     eventType,
		Severity: severity,
		Actor:    actor,
		Action:   "authenticate",
		Status:   status,
		Details:  map[string]interface{}{"reason": reason},
	})
}

// LogValidation logs validation events
func (l *Logger) LogValidation(ctx context.Context, actor, tool string, allowed bool, violations int) {
	status := "allowed"
	if !allowed {
		status = "denied"
	}

	l.Log(ctx, Event{
		Type:     EventValidation,
		Severity: SevInfo,
		Actor:    actor,
		Action:   "validate",
		Resource: tool,
		Status:   status,
		Details: map[string]interface{}{
			"violations": violations,
		},
	})
}

// LogRuleChange logs rule modification events
func (l *Logger) LogRuleChange(ctx context.Context, actor, ruleID, action string) {
	l.Log(ctx, Event{
		Type:     EventRuleChange,
		Severity: SevCritical, // Rule changes are security-critical
		Actor:    actor,
		Action:   action, // create, update, delete, toggle
		Resource: ruleID,
		Status:   "success",
	})
}

// LogDocChange logs document modification events
func (l *Logger) LogDocChange(ctx context.Context, actor, docSlug, action string) {
	l.Log(ctx, Event{
		Type:     EventDocChange,
		Severity: SevInfo,
		Actor:    actor,
		Action:   action,
		Resource: docSlug,
		Status:   "success",
	})
}

// LogSession logs session lifecycle events
func (l *Logger) LogSession(ctx context.Context, eventType EventType, token, projectSlug string) {
	l.Log(ctx, Event{
		Type:     eventType,
		Severity: SevInfo,
		Actor:    "system",
		Action:   string(eventType),
		Resource: projectSlug,
		Details: map[string]interface{}{
			"session_hash": hashToken(token),
		},
	})
}

// hashToken creates a short hash for logging
func hashToken(token string) string {
	if len(token) < 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
