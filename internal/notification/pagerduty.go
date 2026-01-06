package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// PagerDutyProvider sends PagerDuty Events API notifications
type PagerDutyProvider struct{}

func init() {
	RegisterProvider(&PagerDutyProvider{})
}

func (p *PagerDutyProvider) Name() string {
	return "pagerduty"
}

func (p *PagerDutyProvider) Send(ctx context.Context, notification *Notification, message *Message) error {
	// Get PagerDuty configuration
	integrationKey, _ := notification.Config["integration_key"].(string)
	severity, _ := notification.Config["severity"].(string)

	if integrationKey == "" {
		return fmt.Errorf("integration_key is required")
	}

	// Default severity
	if severity == "" {
		if message.Status == "down" {
			severity = "critical"
		} else {
			severity = "info"
		}
	}

	// Determine event action based on status
	eventAction := "trigger"
	if message.Status == "up" {
		eventAction = "resolve"
	}

	// Build custom details
	customDetails := map[string]interface{}{
		"monitor":       message.MonitorName,
		"status":        message.Status,
		"message":       message.Body,
		"response_time": fmt.Sprintf("%dms", message.Ping),
		"time":          message.Time,
	}

	if message.MonitorURL != "" {
		customDetails["url"] = message.MonitorURL
	}

	// Build PagerDuty Events API v2 payload
	payload := map[string]interface{}{
		"routing_key":  integrationKey,
		"event_action": eventAction,
		"dedup_key":    fmt.Sprintf("uptime-kuma-%s", message.MonitorName),
		"payload": map[string]interface{}{
			"summary":        message.Title,
			"source":         "Uptime Kuma",
			"severity":       severity,
			"timestamp":      message.Time,
			"custom_details": customDetails,
		},
	}

	// Marshal payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Send to PagerDuty Events API
	url := "https://events.pagerduty.com/v2/enqueue"

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send PagerDuty event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("PagerDuty API returned status %d", resp.StatusCode)
	}

	// Parse response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode PagerDuty response: %w", err)
	}

	if status, _ := result["status"].(string); status != "success" {
		message, _ := result["message"].(string)
		return fmt.Errorf("PagerDuty API error: %s", message)
	}

	return nil
}

func (p *PagerDutyProvider) Validate(config map[string]interface{}) error {
	integrationKey, ok := config["integration_key"].(string)
	if !ok || integrationKey == "" {
		return fmt.Errorf("integration_key is required")
	}

	return nil
}
