package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SlackProvider sends Slack webhook notifications
type SlackProvider struct{}

func init() {
	RegisterProvider(&SlackProvider{})
}

func (s *SlackProvider) Name() string {
	return "slack"
}

func (s *SlackProvider) Send(ctx context.Context, notification *Notification, message *Message) error {
	// Get Slack webhook URL
	webhookURL, _ := notification.Config["webhook_url"].(string)
	channel, _ := notification.Config["channel"].(string)
	username, _ := notification.Config["username"].(string)
	iconEmoji, _ := notification.Config["icon_emoji"].(string)

	if webhookURL == "" {
		return fmt.Errorf("webhook_url is required")
	}

	// Default username
	if username == "" {
		username = "Uptime Kabomba"
	}

	// Default icon based on status
	if iconEmoji == "" {
		switch message.Status {
		case "up":
			iconEmoji = ":white_check_mark:"
		case "down":
			iconEmoji = ":x:"
		case "maintenance":
			iconEmoji = ":wrench:"
		default:
			iconEmoji = ":information_source:"
		}
	}

	// Determine color based on status
	var color string
	switch message.Status {
	case "up":
		color = "good" // Green
	case "down":
		color = "danger" // Red
	case "maintenance":
		color = "#0000FF" // Blue
	default:
		color = "#808080" // Gray
	}

	// Build Slack attachment
	attachment := map[string]interface{}{
		"color":      color,
		"title":      message.Title,
		"text":       message.Body,
		"ts":         time.Now().Unix(),
		"footer":     "Uptime Kabomba",
		"footer_icon": "https://uptime.kuma.pet/img/icon.svg",
		"fields":     []map[string]interface{}{},
	}

	// Add fields
	fields := attachment["fields"].([]map[string]interface{})
	fields = append(fields, map[string]interface{}{
		"title": "Monitor",
		"value": message.MonitorName,
		"short": true,
	})

	fields = append(fields, map[string]interface{}{
		"title": "Status",
		"value": message.Status,
		"short": true,
	})

	if message.Ping > 0 {
		fields = append(fields, map[string]interface{}{
			"title": "Response Time",
			"value": fmt.Sprintf("%dms", message.Ping),
			"short": true,
		})
	}

	if message.MonitorURL != "" {
		fields = append(fields, map[string]interface{}{
			"title": "URL",
			"value": message.MonitorURL,
			"short": false,
		})
	}

	attachment["fields"] = fields

	// Build payload
	payload := map[string]interface{}{
		"username":    username,
		"icon_emoji":  iconEmoji,
		"attachments": []interface{}{attachment},
	}

	// Add channel if specified
	if channel != "" {
		payload["channel"] = channel
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
		return fmt.Errorf("failed to send Slack webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (s *SlackProvider) Validate(config map[string]interface{}) error {
	webhookURL, ok := config["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return fmt.Errorf("webhook_url is required")
	}

	return nil
}
