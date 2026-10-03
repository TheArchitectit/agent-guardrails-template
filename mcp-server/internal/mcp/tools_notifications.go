package mcp

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/thearchitectit/guardrail-mcp/internal/notifications"
)

// validateWebhookURL rejects webhook targets that would turn the dispatcher
// into a server-side request forgery vector.
//
// The URL was previously stored verbatim: no scheme check, no host check, no
// address filtering. Because the dispatcher POSTs to whatever is stored, any
// caller could direct the server at loopback, link-local or private-range
// services, including cloud metadata endpoints.
//
// Resolution happens at configuration time, so this does not defend against
// DNS rebinding between configuration and delivery; it removes the trivial
// class of targets and the stored-internal-address case.
func validateWebhookURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid url: scheme must be http or https")
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("invalid url: host is required")
	}
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".localhost") {
		return fmt.Errorf("refusing webhook to loopback host %q", host)
	}

	// A literal IP needs no resolution.
	if ip := net.ParseIP(host); ip != nil {
		return checkWebhookIP(ip, host)
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("cannot resolve webhook host %q: %w", host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("cannot resolve webhook host %q", host)
	}
	for _, ip := range ips {
		if err := checkWebhookIP(ip, host); err != nil {
			return err
		}
	}
	return nil
}

func checkWebhookIP(ip net.IP, host string) error {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() {
		return fmt.Errorf("refusing webhook to non-public address %s for host %q", ip, host)
	}
	return nil
}

// notificationToolList returns the tool definitions for webhook notification tools.
func (s *MCPServer) notificationToolList() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "configure_webhook",
			Description: "Create or update a webhook endpoint for receiving violation and halt notifications",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"team_id": map[string]interface{}{
						"type":        "string",
						"description": "Team ID to associate the webhook with",
					},
					"url": map[string]interface{}{
						"type":        "string",
						"description": "Webhook endpoint URL (must be HTTPS in production)",
					},
					"events": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Event types to subscribe to: violation.detected, halt.triggered",
					},
					"secret_hmac": map[string]interface{}{
						"type":        "string",
						"description": "HMAC-SHA256 secret for signing payloads (generate with: openssl rand -hex 32)",
					},
					"enabled": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether the webhook is active (default: true)",
					},
					"webhook_id": map[string]interface{}{
						"type":        "string",
						"description": "Existing webhook ID to update (omit to create new)",
					},
				},
				Required: []string{"team_id", "url", "events", "secret_hmac"},
			},
		},
		{
			Name:        "test_webhook",
			Description: "Send a test event to a configured webhook endpoint",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"webhook_id": map[string]interface{}{
						"type":        "string",
						"description": "Webhook ID to test",
					},
				},
				Required: []string{"webhook_id"},
			},
		},
		{
			Name:        "list_webhooks",
			Description: "List all configured webhooks for a team",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"team_id": map[string]interface{}{
						"type":        "string",
						"description": "Team ID to list webhooks for",
					},
				},
				Required: []string{"team_id"},
			},
		},
		{
			Name:        "delete_webhook",
			Description: "Delete a configured webhook endpoint",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"webhook_id": map[string]interface{}{
						"type":        "string",
						"description": "Webhook ID to delete",
					},
				},
				Required: []string{"webhook_id"},
			},
		},
		{
			Name:        "get_webhook_deliveries",
			Description: "View recent webhook delivery history",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"webhook_id": map[string]interface{}{
						"type":        "string",
						"description": "Webhook ID to get deliveries for",
					},
					"limit": map[string]interface{}{
						"type":        "number",
						"description": "Max deliveries to return (default: 50)",
					},
				},
				Required: []string{"webhook_id"},
			},
		},
	}
}

func (s *MCPServer) handleConfigureWebhook(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	teamID, _ := args["team_id"].(string)
	url, _ := args["url"].(string)
	secretHMAC, _ := args["secret_hmac"].(string)
	// enabled defaults to true, matching the documented schema. A bare type
	// assertion made an omitted value false, so creating a webhook without
	// explicitly passing enabled produced a disabled one that silently
	// received nothing.
	enabled := true
	if v, ok := args["enabled"].(bool); ok {
		enabled = v
	}
	webhookID, _ := args["webhook_id"].(string)

	eventsRaw, ok := args["events"].([]interface{})
	if !ok {
		return buildToolResult(map[string]interface{}{
			"error": "events must be an array of strings",
		}, true)
	}

	events := make([]string, 0, len(eventsRaw))
	for _, e := range eventsRaw {
		if ev, ok := e.(string); ok {
			events = append(events, ev)
		}
	}

	if teamID == "" || url == "" || secretHMAC == "" || len(events) == 0 {
		return buildToolResult(map[string]interface{}{
			"error": "team_id, url, events, and secret_hmac are required",
		}, true)
	}

	if err := validateWebhookURL(url); err != nil {
		return buildToolResult(map[string]interface{}{
			"error": err.Error(),
		}, true)
	}

	if webhookID != "" {
		// Update existing
		config, err := s.webhookStore.GetByID(ctx, webhookID)
		if err != nil {
			return buildToolResult(map[string]interface{}{
				"error": fmt.Sprintf("webhook not found: %v", err),
			}, true)
		}
		config.URL = url
		config.Events = events
		config.SecretHMAC = secretHMAC
		config.Enabled = enabled
		if err := s.webhookStore.Update(ctx, config); err != nil {
			return buildToolResult(map[string]interface{}{
				"error": fmt.Sprintf("failed to update webhook: %v", err),
			}, true)
		}
		return buildToolResult(map[string]interface{}{
			"success":    true,
			"webhook_id": config.ID,
			"message":    "Webhook updated successfully",
		}, false)
	}

	// Create new
	config := &notifications.WebhookConfig{
		TeamID:     teamID,
		URL:        url,
		Events:     events,
		SecretHMAC: secretHMAC,
		Enabled:    enabled,
	}

	if err := s.webhookStore.Create(ctx, config); err != nil {
		return buildToolResult(map[string]interface{}{
			"error": fmt.Sprintf("failed to create webhook: %v", err),
		}, true)
	}

	return buildToolResult(map[string]interface{}{
		"success":    true,
		"webhook_id": config.ID,
		"message":    "Webhook created successfully",
	}, false)
}

func (s *MCPServer) handleTestWebhook(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	webhookID, _ := args["webhook_id"].(string)
	if webhookID == "" {
		return buildToolResult(map[string]interface{}{
			"error": "webhook_id is required",
		}, true)
	}

	if s.webhookDispatcher == nil {
		return buildToolResult(map[string]interface{}{
			"error": "webhook dispatcher not initialized",
		}, true)
	}

	delivery, err := s.webhookDispatcher.SendTestEvent(ctx, webhookID)
	if err != nil {
		return buildToolResult(map[string]interface{}{
			"error": fmt.Sprintf("test failed: %v", err),
		}, true)
	}

	return buildToolResult(map[string]interface{}{
		"success":       delivery.Success,
		"status_code":   delivery.StatusCode,
		"response_body": delivery.ResponseBody,
		"error_message": delivery.ErrorMessage,
		"delivery_id":   delivery.ID,
	}, false)
}

func (s *MCPServer) handleListWebhooks(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	teamID, _ := args["team_id"].(string)
	if teamID == "" {
		return buildToolResult(map[string]interface{}{
			"error": "team_id is required",
		}, true)
	}

	configs, err := s.webhookStore.ListByTeam(ctx, teamID)
	if err != nil {
		return buildToolResult(map[string]interface{}{
			"error": fmt.Sprintf("failed to list webhooks: %v", err),
		}, true)
	}

	// Mask secrets in response
	result := make([]map[string]interface{}, 0, len(configs))
	for _, c := range configs {
		result = append(result, map[string]interface{}{
			"id":         c.ID,
			"team_id":    c.TeamID,
			"url":        c.URL,
			"events":     c.Events,
			"enabled":    c.Enabled,
			"created_at": c.CreatedAt,
			"updated_at": c.UpdatedAt,
		})
	}

	return buildToolResult(map[string]interface{}{
		"webhooks": result,
		"count":    len(result),
	}, false)
}

func (s *MCPServer) handleDeleteWebhook(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	webhookID, _ := args["webhook_id"].(string)
	if webhookID == "" {
		return buildToolResult(map[string]interface{}{
			"error": "webhook_id is required",
		}, true)
	}

	if err := s.webhookStore.Delete(ctx, webhookID); err != nil {
		return buildToolResult(map[string]interface{}{
			"error": fmt.Sprintf("failed to delete webhook: %v", err),
		}, true)
	}

	return buildToolResult(map[string]interface{}{
		"success": true,
		"message": "Webhook deleted successfully",
	}, false)
}

func (s *MCPServer) handleGetWebhookDeliveries(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	webhookID, _ := args["webhook_id"].(string)
	if webhookID == "" {
		return buildToolResult(map[string]interface{}{
			"error": "webhook_id is required",
		}, true)
	}

	limit := 50
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	deliveries, err := s.webhookStore.ListDeliveries(ctx, webhookID, limit)
	if err != nil {
		return buildToolResult(map[string]interface{}{
			"error": fmt.Sprintf("failed to get deliveries: %v", err),
		}, true)
	}

	return buildToolResult(map[string]interface{}{
		"deliveries": deliveries,
		"count":      len(deliveries),
	}, false)
}
