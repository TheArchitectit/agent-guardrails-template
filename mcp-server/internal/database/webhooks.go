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

// GetByID retrieves a webhook config by ID.
func (s *WebhookStore) GetByID(ctx context.Context, id string) (*notifications.WebhookConfig, error) {
	var config notifications.WebhookConfig
	var events []string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, team_id, url, events, secret_hmac, enabled, created_at, updated_at
		FROM webhook_configs WHERE id = $1
	`, id).Scan(
		&config.ID, &config.TeamID, &config.URL, pq.Array(&events),
		&config.SecretHMAC, &config.Enabled, &config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("webhook not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get webhook: %w", err)
	}
	config.Events = events
	return &config, nil
}

// ListByTeam retrieves all webhook configs for a team.
func (s *WebhookStore) ListByTeam(ctx context.Context, teamID string) ([]*notifications.WebhookConfig, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, team_id, url, events, secret_hmac, enabled, created_at, updated_at
		FROM webhook_configs WHERE team_id = $1 ORDER BY created_at DESC
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}
	defer rows.Close()

	var configs []*notifications.WebhookConfig
	for rows.Next() {
		var config notifications.WebhookConfig
		var events []string
		if err := rows.Scan(
			&config.ID, &config.TeamID, &config.URL, pq.Array(&events),
			&config.SecretHMAC, &config.Enabled, &config.CreatedAt, &config.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan webhook: %w", err)
		}
		config.Events = events
		configs = append(configs, &config)
	}
	return configs, rows.Err()
}

// GetByEventType retrieves all enabled webhooks that subscribe to the given event type.
func (s *WebhookStore) GetByEventType(ctx context.Context, eventType string) ([]*notifications.WebhookConfig, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, team_id, url, events, secret_hmac, enabled, created_at, updated_at
		FROM webhook_configs
		WHERE enabled = true AND $1 = ANY(events)
	`, eventType)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhooks by event: %w", err)
	}
	defer rows.Close()

	var configs []*notifications.WebhookConfig
	for rows.Next() {
		var config notifications.WebhookConfig
		var events []string
		if err := rows.Scan(
			&config.ID, &config.TeamID, &config.URL, pq.Array(&events),
			&config.SecretHMAC, &config.Enabled, &config.CreatedAt, &config.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan webhook: %w", err)
		}
		config.Events = events
		configs = append(configs, &config)
	}
	return configs, rows.Err()
}

// Update modifies an existing webhook configuration.
func (s *WebhookStore) Update(ctx context.Context, config *notifications.WebhookConfig) error {
	config.UpdatedAt = time.Now().UTC()
	result, err := s.db.ExecContext(ctx, `
		UPDATE webhook_configs
		SET url = $2, events = $3, secret_hmac = $4, enabled = $5, updated_at = $6
		WHERE id = $1
	`, config.ID, config.URL, pq.Array(config.Events),
		config.SecretHMAC, config.Enabled, config.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update webhook: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("webhook not found: %s", config.ID)
	}
	return nil
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
