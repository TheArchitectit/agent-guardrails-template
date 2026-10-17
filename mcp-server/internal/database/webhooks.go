package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/thearchitectit/guardrail-mcp/internal/notifications"
)

// WebhookStore handles webhook config database operations.
type WebhookStore struct {
	db *DB
}

// NewWebhookStore creates a new webhook store.
func NewWebhookStore(db *DB) *WebhookStore {
	return &WebhookStore{db: db}
}

// Create inserts a new webhook configuration.
func (s *WebhookStore) Create(ctx context.Context, config *notifications.WebhookConfig) error {
	config.ID = generateUUID()
	now := time.Now().UTC()
	config.CreatedAt = now
	config.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO webhook_configs (id, team_id, url, events, secret_hmac, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, config.ID, config.TeamID, config.URL, pq.Array(config.Events),
		config.SecretHMAC, config.Enabled, config.CreatedAt, config.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}
	return nil
}

// rowScanner is the subset of *sql.Row / *sql.Rows the webhook row reader
// needs, so both a single-row and a multi-row query share one column list and
// one NULL-handling rule.
type rowScanner interface {
	Scan(dest ...any) error
}

// webhookColumns is the single column list for every webhook_configs read. The
// previous-key columns are rotation state (Spec 17 R17-10); keeping one list
// means a new column cannot be added to one query and forgotten in another.
const webhookColumns = `id, team_id, url, events, secret_hmac, secret_hmac_previous, secret_hmac_rotated_at, enabled, created_at, updated_at`

// scanWebhookConfig reads one webhook_configs row in webhookColumns order. The
// previous signing key is nullable; it is loaded into the model (it must be
// available to sign inside the overlap) but is never serialized outward.
func scanWebhookConfig(sc rowScanner) (*notifications.WebhookConfig, error) {
	var config notifications.WebhookConfig
	var events []string
	var prev sql.NullString
	var rotatedAt sql.NullTime
	if err := sc.Scan(
		&config.ID, &config.TeamID, &config.URL, pq.Array(&events),
		&config.SecretHMAC, &prev, &rotatedAt, &config.Enabled,
		&config.CreatedAt, &config.UpdatedAt,
	); err != nil {
		return nil, err
	}
	config.Events = events
	config.SecretHMACPrevious = prev.String
	if rotatedAt.Valid {
		t := rotatedAt.Time
		config.SecretHMACRotatedAt = &t
	}
	return &config, nil
}

// GetByID retrieves a webhook config by ID.
func (s *WebhookStore) GetByID(ctx context.Context, id string) (*notifications.WebhookConfig, error) {
	config, err := scanWebhookConfig(s.db.QueryRowContext(ctx, `
		SELECT `+webhookColumns+`
		FROM webhook_configs WHERE id = $1
	`, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("webhook not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get webhook: %w", err)
	}
	return config, nil
}

// ListByTeam retrieves all webhook configs for a team.
func (s *WebhookStore) ListByTeam(ctx context.Context, teamID string) ([]*notifications.WebhookConfig, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+webhookColumns+`
		FROM webhook_configs WHERE team_id = $1 ORDER BY created_at DESC
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}
	defer rows.Close()

	var configs []*notifications.WebhookConfig
	for rows.Next() {
		config, err := scanWebhookConfig(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook: %w", err)
		}
		configs = append(configs, config)
	}
	return configs, rows.Err()
}

// GetByEventType retrieves all enabled webhooks that subscribe to the given event type.
func (s *WebhookStore) GetByEventType(ctx context.Context, eventType string) ([]*notifications.WebhookConfig, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+webhookColumns+`
		FROM webhook_configs
		WHERE enabled = true AND $1 = ANY(events)
	`, eventType)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhooks by event: %w", err)
	}
	defer rows.Close()

	var configs []*notifications.WebhookConfig
	for rows.Next() {
		config, err := scanWebhookConfig(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook: %w", err)
		}
		configs = append(configs, config)
	}
	return configs, rows.Err()
}

// Update modifies an existing webhook configuration. It carries the rotation
// state through so an ordinary edit (URL/events/enabled) never silently
// discards an in-flight overlap or cuts over a rotation early.
func (s *WebhookStore) Update(ctx context.Context, config *notifications.WebhookConfig) error {
	config.UpdatedAt = time.Now().UTC()
	result, err := s.db.ExecContext(ctx, `
		UPDATE webhook_configs
		SET url = $2, events = $3, secret_hmac = $4, enabled = $5, updated_at = $6,
		    secret_hmac_previous = $7, secret_hmac_rotated_at = $8
		WHERE id = $1
	`, config.ID, config.URL, pq.Array(config.Events),
		config.SecretHMAC, config.Enabled, config.UpdatedAt,
		nullableSecret(config.SecretHMACPrevious), nullableTime(config.SecretHMACRotatedAt))
	if err != nil {
		return fmt.Errorf("failed to update webhook: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("webhook not found: %s", config.ID)
	}
	return nil
}

// RotateSigningKey stages a signing-key replacement: the incoming key becomes
// active and the outgoing key is retained as the overlap key, valid only until
// the bounded overlap elapses (Spec 17 R17-10). It returns the applied rotation
// event (opaque fingerprints only) for the caller to audit. The rotation is a
// single statement, so a reader never observes a half-applied key change.
func (s *WebhookStore) RotateSigningKey(ctx context.Context, id, newSecret string, now time.Time) (*notifications.WebhookRotationEvent, error) {
	if newSecret == "" {
		return nil, fmt.Errorf("refusing to rotate %s to an empty signing key", id)
	}
	now = now.UTC()
	// RETURNING reads the outgoing key (now stored as the previous key) in the
	// same statement that applies the replacement, so the event describes
	// exactly the state that was written and no separate read can race it.
	var previous sql.NullString
	err := s.db.QueryRowContext(ctx, `
		UPDATE webhook_configs
		SET secret_hmac_previous = secret_hmac,
		    secret_hmac = $2,
		    secret_hmac_rotated_at = $3,
		    updated_at = $3
		WHERE id = $1
		RETURNING secret_hmac_previous
	`, id, newSecret, now).Scan(&previous)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("webhook not found: %s", id)
		}
		return nil, fmt.Errorf("failed to rotate webhook signing key: %w", err)
	}
	return &notifications.WebhookRotationEvent{
		WebhookID:      id,
		Operation:      "rotate",
		KeyID:          notifications.WebhookKeyFingerprint(newSecret),
		PreviousKeyID:  notifications.WebhookKeyFingerprint(previous.String),
		OverlapSeconds: int(notifications.WebhookRotationOverlap / time.Second),
		OccurredAt:     now,
	}, nil
}

// CutoverSigningKey invalidates the outgoing signing key immediately: the
// active key is unchanged and the overlap key is cleared, so the next delivery
// signs with the active key only. It returns the applied event for audit.
func (s *WebhookStore) CutoverSigningKey(ctx context.Context, id string, now time.Time) (*notifications.WebhookRotationEvent, error) {
	now = now.UTC()
	// Read the fingerprints under the same statement that clears them, so the
	// event describes exactly the state that was removed.
	var active, previous sql.NullString
	err := s.db.QueryRowContext(ctx, `
		UPDATE webhook_configs
		SET secret_hmac_previous = NULL,
		    secret_hmac_rotated_at = NULL,
		    updated_at = $2
		WHERE id = $1
		RETURNING secret_hmac, secret_hmac_previous
	`, id, now).Scan(&active, &previous)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("webhook not found: %s", id)
		}
		return nil, fmt.Errorf("failed to cut over webhook signing key: %w", err)
	}
	return &notifications.WebhookRotationEvent{
		WebhookID:     id,
		Operation:     "cutover",
		KeyID:         notifications.WebhookKeyFingerprint(active.String),
		PreviousKeyID: notifications.WebhookKeyFingerprint(previous.String),
		OccurredAt:    now,
	}, nil
}

// nullableSecret maps an empty secret to SQL NULL so an absent overlap key is
// distinguishable from an empty one.
func nullableSecret(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nullableTime maps a nil instant to SQL NULL.
func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC()
}

// Delete removes a webhook configuration.
func (s *WebhookStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM webhook_configs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("webhook not found: %s", id)
	}
	return nil
}

// RecordDelivery inserts a webhook delivery record.
func (s *WebhookStore) RecordDelivery(ctx context.Context, delivery *notifications.WebhookDelivery) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO webhook_deliveries (id, webhook_id, event_type, status_code, response_body, success, error_message, delivered_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, delivery.ID, delivery.WebhookID, delivery.EventType,
		delivery.StatusCode, delivery.ResponseBody, delivery.Success,
		delivery.ErrorMessage, delivery.DeliveredAt)
	if err != nil {
		return fmt.Errorf("failed to record delivery: %w", err)
	}
	return nil
}

// ListDeliveries retrieves recent deliveries for a webhook.
func (s *WebhookStore) ListDeliveries(ctx context.Context, webhookID string, limit int) ([]*notifications.WebhookDelivery, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, webhook_id, event_type, status_code, response_body, success, error_message, delivered_at
		FROM webhook_deliveries
		WHERE webhook_id = $1
		ORDER BY delivered_at DESC
		LIMIT $2
	`, webhookID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []*notifications.WebhookDelivery
	for rows.Next() {
		var d notifications.WebhookDelivery
		if err := rows.Scan(
			&d.ID, &d.WebhookID, &d.EventType, &d.StatusCode,
			&d.ResponseBody, &d.Success, &d.ErrorMessage, &d.DeliveredAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan delivery: %w", err)
		}
		deliveries = append(deliveries, &d)
	}
	return deliveries, rows.Err()
}

// generateUUID generates a new UUID string.
func generateUUID() string {
	return uuid.New().String()
}
