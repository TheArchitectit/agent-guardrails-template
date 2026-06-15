package notifications

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sony/gobreaker"
	"github.com/thearchitectit/guardrail-mcp/internal/domain"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

// WebhookStore defines the persistence interface for webhooks.
type WebhookStore interface {
	Create(ctx context.Context, config *WebhookConfig) error
	GetByID(ctx context.Context, id string) (*WebhookConfig, error)
	ListByTeam(ctx context.Context, teamID string) ([]*WebhookConfig, error)
	GetByEventType(ctx context.Context, eventType string) ([]*WebhookConfig, error)
	Update(ctx context.Context, config *WebhookConfig) error
	Delete(ctx context.Context, id string) error
	RecordDelivery(ctx context.Context, delivery *WebhookDelivery) error
}

// Dispatcher subscribes to domain events and delivers webhook notifications.
type Dispatcher struct {
	store     WebhookStore
	client    *http.Client
	breakers  map[string]*gobreaker.CircuitBreaker
	breakersMu sync.RWMutex
}

// NewDispatcher creates a new webhook dispatcher.
func NewDispatcher(store WebhookStore) *Dispatcher {
	return &Dispatcher{
		store: store,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		breakers: make(map[string]*gobreaker.CircuitBreaker),
	}
}

// Subscribe registers the dispatcher's event handlers on the event bus.
func (d *Dispatcher) Subscribe(bus domain.EventBus) {
	bus.Subscribe(domain.EventViolationDetected, d.handleViolation)
	bus.Subscribe(domain.EventHaltTriggered, d.handleHalt)
}

func (d *Dispatcher) handleViolation(ctx context.Context, event domain.Event) {
	d.dispatchEvent(ctx, string(event.Type), event.Payload)
}

func (d *Dispatcher) handleHalt(ctx context.Context, event domain.Event) {
	d.dispatchEvent(ctx, string(event.Type), event.Payload)
}

// dispatchEvent finds matching webhooks and delivers asynchronously.
func (d *Dispatcher) dispatchEvent(ctx context.Context, eventType string, payload interface{}) {
	webhooks, err := d.store.GetByEventType(ctx, eventType)
	if err != nil {
		slog.Error("failed to load webhooks for event", "event_type", eventType, "error", err)
		return
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal event payload", "event_type", eventType, "error", err)
		return
	}

	for _, wh := range webhooks {
		if !wh.Enabled {
			continue
		}
		go d.deliver(wh, eventType, payloadJSON)
	}
}

// deliver sends a single webhook notification with HMAC signing and retry.
func (d *Dispatcher) deliver(wh *WebhookConfig, eventType string, payloadJSON []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	webhookID := uuid.New().String()
	signature := signPayload(payloadJSON, wh.SecretHMAC)

	wp := WebhookPayload{
		ID:        webhookID,
		EventType: eventType,
		Timestamp: time.Now().UTC(),
		Payload:   payloadJSON,
		Signature: signature,
	}

	body, err := json.Marshal(wp)
	if err != nil {
		slog.Error("failed to marshal webhook payload", "webhook_id", wh.ID, "error", err)
		return
	}

	breaker := d.getBreaker(wh.ID)

	var lastErr error
	var lastStatusCode int
	var lastResponseBody string

	for attempt := 0; attempt < 3; attempt++ {
		result, err := breaker.Execute(func() (interface{}, error) {
			return d.doRequest(ctx, wh.URL, body, signature)
		})

		if err != nil {
			lastErr = err
			if attempt < 2 {
				time.Sleep(time.Duration(1<<uint(attempt)) * time.Second)
				continue
			}
			break
		}

		resp := result.(*deliveryResult)
		lastStatusCode = resp.StatusCode
		lastResponseBody = resp.Body

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			d.recordDelivery(ctx, wh.ID, eventType, lastStatusCode, lastResponseBody, true, "")
			return
		}

		lastErr = fmt.Errorf("webhook returned status %d", resp.StatusCode)
		if attempt < 2 {
			time.Sleep(time.Duration(1<<uint(attempt)) * time.Second)
		}
	}

	errMsg := ""
	if lastErr != nil {
		errMsg = lastErr.Error()
	}
	d.recordDelivery(ctx, wh.ID, eventType, lastStatusCode, lastResponseBody, false, errMsg)
	slog.Warn("webhook delivery failed after retries",
		"webhook_id", wh.ID,
		"url", wh.URL,
		"event_type", eventType,
		"error", errMsg,
	)
}

type deliveryResult struct {
	StatusCode int
	Body       string
}

func (d *Dispatcher) doRequest(ctx context.Context, url string, body []byte, signature string) (*deliveryResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Guardrail-Signature", signature)
	req.Header.Set("X-Guardrail-Event", "webhook")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	respBody := string(buf[:n])

	return &deliveryResult{
		StatusCode: resp.StatusCode,
		Body:       respBody,
	}, nil
}

func (d *Dispatcher) recordDelivery(ctx context.Context, webhookID, eventType string, statusCode int, responseBody string, success bool, errMsg string) {
	delivery := &WebhookDelivery{
		ID:           uuid.New().String(),
		WebhookID:    webhookID,
		EventType:    eventType,
		StatusCode:   statusCode,
		ResponseBody: responseBody,
		Success:      success,
		ErrorMessage: errMsg,
		DeliveredAt:  time.Now().UTC(),
	}
	if err := d.store.RecordDelivery(ctx, delivery); err != nil {
		slog.Error("failed to record webhook delivery", "error", err)
	}
}

func (d *Dispatcher) getBreaker(webhookID string) *gobreaker.CircuitBreaker {
	d.breakersMu.RLock()
	if cb, ok := d.breakers[webhookID]; ok {
		d.breakersMu.RUnlock()
		return cb
	}
	d.breakersMu.RUnlock()

	d.breakersMu.Lock()
	defer d.breakersMu.Unlock()

	// Double-check after acquiring write lock
	if cb, ok := d.breakers[webhookID]; ok {
		return cb
	}

	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        fmt.Sprintf("webhook-%s", webhookID),
		MaxRequests: 3,
		Interval:    5 * time.Minute,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			slog.Warn("webhook circuit breaker state change",
				"name", name,
				"from", from.String(),
				"to", to.String(),
			)
		},
	})
	d.breakers[webhookID] = cb
	return cb
}

// signPayload computes HMAC-SHA256 of the payload.
func signPayload(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature verifies an incoming webhook signature.
func VerifySignature(payload []byte, secret, signature string) bool {
	expected := signPayload(payload, secret)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// SendTestEvent sends a test event to a webhook URL.
func (d *Dispatcher) SendTestEvent(ctx context.Context, webhookID string) (*WebhookDelivery, error) {
	wh, err := d.store.GetByID(ctx, webhookID)
	if err != nil {
		return nil, fmt.Errorf("webhook not found: %w", err)
	}

	testPayload := models.PreventionRule{
		RuleID:  "test-rule",
		Name:    "Test Webhook",
		Message: "This is a test webhook delivery",
	}

	testEvent := map[string]interface{}{
		"rule":   testPayload,
		"result": "test",
	}

	payloadJSON, _ := json.Marshal(testEvent)
	signature := signPayload(payloadJSON, wh.SecretHMAC)

	wp := WebhookPayload{
		ID:        uuid.New().String(),
		EventType: "test",
		Timestamp: time.Now().UTC(),
		Payload:   payloadJSON,
		Signature: signature,
	}

	body, _ := json.Marshal(wp)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Guardrail-Signature", signature)
	req.Header.Set("X-Guardrail-Event", "test")

	resp, err := d.client.Do(req)
	if err != nil {
		delivery := &WebhookDelivery{
			ID:           uuid.New().String(),
			WebhookID:    webhookID,
			EventType:    "test",
			Success:      false,
			ErrorMessage: err.Error(),
			DeliveredAt:  time.Now().UTC(),
		}
		_ = d.store.RecordDelivery(ctx, delivery)
		return delivery, nil
	}
	defer resp.Body.Close()

	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)

	delivery := &WebhookDelivery{
		ID:           uuid.New().String(),
		WebhookID:    webhookID,
		EventType:    "test",
		StatusCode:   resp.StatusCode,
		ResponseBody: string(buf[:n]),
		Success:      resp.StatusCode >= 200 && resp.StatusCode < 300,
		DeliveredAt:  time.Now().UTC(),
	}
	_ = d.store.RecordDelivery(ctx, delivery)
	return delivery, nil
}
