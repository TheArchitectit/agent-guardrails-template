package notifications

import (
	"encoding/json"
	"time"
)

// WebhookConfig represents a configured webhook endpoint.
type WebhookConfig struct {
	ID         string    `json:"id"`
	TeamID     string    `json:"team_id"`
	URL        string    `json:"url"`
	Events     []string  `json:"events"`
	SecretHMAC string    `json:"secret_hmac,omitempty"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// WebhookPayload is the body sent to webhook endpoints.
type WebhookPayload struct {
	ID        string          `json:"id"`
	EventType string          `json:"event_type"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
	Signature string          `json:"signature"`
}

// WebhookDelivery records the result of a webhook delivery attempt.
type WebhookDelivery struct {
	ID           string    `json:"id"`
	WebhookID    string    `json:"webhook_id"`
	EventType    string    `json:"event_type"`
	StatusCode   int       `json:"status_code,omitempty"`
	ResponseBody string    `json:"response_body,omitempty"`
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"error_message,omitempty"`
	DeliveredAt  time.Time `json:"delivered_at"`
}
