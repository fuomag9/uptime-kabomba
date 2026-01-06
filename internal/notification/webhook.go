package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WebhookProvider sends webhook notifications
type WebhookProvider struct{}

func init() {
	RegisterProvider(&WebhookProvider{})
}

func (w *WebhookProvider) Name() string {
	return "webhook"
}

func (w *WebhookProvider) Send(ctx context.Context, notification *Notification, message *Message) error {
	// Get webhook configuration
	url, _ := notification.Config["webhook_url"].(string)
	method, _ := notification.Config["method"].(string)
	contentType, _ := notification.Config["content_type"].(string)
	customHeaders, _ := notification.Config["headers"].(map[string]interface{})

	if url == "" {
		return fmt.Errorf("webhook_url is required")
	}

	// Default method to POST
	if method == "" {
		method = "POST"
	}

	// Default content type to JSON
	if contentType == "" {
		contentType = "application/json"
	}

	// Build payload
	payload := map[string]interface{}{
		"title":        message.Title,
		"body":         message.Body,
		"monitor_name": message.MonitorName,
		"monitor_url":  message.MonitorURL,
		"status":       message.Status,
		"ping":         message.Ping,
		"time":         message.Time,
		"important":    message.Important,
	}

	// Marshal payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", "Uptime-Kuma-Go/1.0")

	// Add custom headers
	if customHeaders != nil {
		for key, value := range customHeaders {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}

	// Send request
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (w *WebhookProvider) Validate(config map[string]interface{}) error {
	url, ok := config["webhook_url"].(string)
	if !ok || url == "" {
		return fmt.Errorf("webhook_url is required")
	}

	return nil
}
