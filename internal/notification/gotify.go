package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GotifyProvider sends Gotify notifications (self-hosted)
type GotifyProvider struct{}

func init() {
	RegisterProvider(&GotifyProvider{})
}

func (g *GotifyProvider) Name() string {
	return "gotify"
}

func (g *GotifyProvider) Send(ctx context.Context, notification *Notification, message *Message) error {
	// Get Gotify configuration
	serverURL, _ := notification.Config["server_url"].(string)
	appToken, _ := notification.Config["app_token"].(string)
	priority, _ := notification.Config["priority"].(float64)

	if serverURL == "" {
		return fmt.Errorf("server_url is required")
	}

	if appToken == "" {
		return fmt.Errorf("app_token is required")
	}

	// Default priority based on status
	if priority == 0 {
		if message.Status == "down" {
			priority = 8 // High priority
		} else {
			priority = 5 // Normal priority
		}
	}

	// Build message text
	messageText := FormatMessage(message)

	// Build Gotify payload
	payload := map[string]interface{}{
		"title":    message.Title,
		"message":  messageText,
		"priority": int(priority),
		"extras": map[string]interface{}{
			"monitor": message.MonitorName,
			"status":  message.Status,
			"url":     message.MonitorURL,
		},
	}

	// Marshal payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Send to Gotify server
	url := fmt.Sprintf("%s/message?token=%s", serverURL, appToken)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Gotify notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Gotify API returned status %d", resp.StatusCode)
	}

	return nil
}

func (g *GotifyProvider) Validate(config map[string]interface{}) error {
	serverURL, ok := config["server_url"].(string)
	if !ok || serverURL == "" {
		return fmt.Errorf("server_url is required")
	}

	appToken, ok := config["app_token"].(string)
	if !ok || appToken == "" {
		return fmt.Errorf("app_token is required")
	}

	return nil
}
