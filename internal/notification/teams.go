package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// TeamsProvider sends Microsoft Teams webhook notifications
type TeamsProvider struct{}

func init() {
	RegisterProvider(&TeamsProvider{})
}

func (t *TeamsProvider) Name() string {
	return "teams"
}

func (t *TeamsProvider) Send(ctx context.Context, notification *Notification, message *Message) error {
	// Get Teams webhook URL
	webhookURL, _ := notification.Config["webhook_url"].(string)

	if webhookURL == "" {
		return fmt.Errorf("webhook_url is required")
	}

	// Determine theme color based on status
	var themeColor string
	switch message.Status {
	case "up":
		themeColor = "00FF00" // Green
	case "down":
		themeColor = "FF0000" // Red
	case "maintenance":
		themeColor = "0000FF" // Blue
	default:
		themeColor = "808080" // Gray
	}

	// Build facts
	facts := []map[string]string{
		{
			"name":  "Monitor",
			"value": message.MonitorName,
		},
		{
			"name":  "Status",
			"value": message.Status,
		},
	}

	if message.Ping > 0 {
		facts = append(facts, map[string]string{
			"name":  "Response Time",
			"value": fmt.Sprintf("%dms", message.Ping),
		})
	}

	if message.MonitorURL != "" {
		facts = append(facts, map[string]string{
			"name":  "URL",
			"value": message.MonitorURL,
		})
	}

	facts = append(facts, map[string]string{
		"name":  "Time",
		"value": message.Time,
	})

	// Build MessageCard payload for Teams
	payload := map[string]interface{}{
		"@type":      "MessageCard",
		"@context":   "https://schema.org/extensions",
		"summary":    message.Title,
		"themeColor": themeColor,
		"title":      message.Title,
		"text":       message.Body,
		"sections": []map[string]interface{}{
			{
				"facts": facts,
			},
		},
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
		return fmt.Errorf("failed to send Teams webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Teams webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (t *TeamsProvider) Validate(config map[string]interface{}) error {
	webhookURL, ok := config["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return fmt.Errorf("webhook_url is required")
	}

	return nil
}
