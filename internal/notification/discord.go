package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// DiscordProvider sends Discord webhook notifications
type DiscordProvider struct{}

func init() {
	RegisterProvider(&DiscordProvider{})
}

func (d *DiscordProvider) Name() string {
	return "discord"
}

func (d *DiscordProvider) Send(ctx context.Context, notification *Notification, message *Message) error {
	// Get Discord webhook URL
	webhookURL, _ := notification.Config["webhook_url"].(string)
	username, _ := notification.Config["username"].(string)

	if webhookURL == "" {
		return fmt.Errorf("webhook_url is required")
	}

	// Default username
	if username == "" {
		username = "Uptime Kabomba"
	}

	// Determine color based on status
	var color int
	switch message.Status {
	case "up":
		color = 0x00FF00 // Green
	case "down":
		color = 0xFF0000 // Red
	case "maintenance":
		color = 0x0000FF // Blue
	default:
		color = 0x808080 // Gray
	}

	// Build Discord embed
	embed := map[string]interface{}{
		"title":       message.Title,
		"description": message.Body,
		"color":       color,
		"timestamp":   message.Time,
		"fields": []map[string]interface{}{
			{
				"name":   "Monitor",
				"value":  message.MonitorName,
				"inline": true,
			},
			{
				"name":   "Status",
				"value":  message.Status,
				"inline": true,
			},
		},
	}

	// Add ping if available
	if message.Ping > 0 {
		embed["fields"] = append(embed["fields"].([]map[string]interface{}), map[string]interface{}{
			"name":   "Response Time",
			"value":  fmt.Sprintf("%dms", message.Ping),
			"inline": true,
		})
	}

	// Add URL if available
	if message.MonitorURL != "" {
		embed["fields"] = append(embed["fields"].([]map[string]interface{}), map[string]interface{}{
			"name":   "URL",
			"value":  message.MonitorURL,
			"inline": false,
		})
	}

	// Build payload
	payload := map[string]interface{}{
		"username": username,
		"embeds":   []interface{}{embed},
	}

	// Marshal payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Send webhook
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Discord webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Discord webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (d *DiscordProvider) Validate(config map[string]interface{}) error {
	webhookURL, ok := config["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return fmt.Errorf("webhook_url is required")
	}

	return nil
}
