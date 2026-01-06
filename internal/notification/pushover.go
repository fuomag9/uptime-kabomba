package notification

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PushoverProvider sends Pushover notifications
type PushoverProvider struct{}

func init() {
	RegisterProvider(&PushoverProvider{})
}

func (p *PushoverProvider) Name() string {
	return "pushover"
}

func (p *PushoverProvider) Send(ctx context.Context, notification *Notification, message *Message) error {
	// Get Pushover configuration
	userKey, _ := notification.Config["user_key"].(string)
	apiToken, _ := notification.Config["api_token"].(string)
	priority, _ := notification.Config["priority"].(float64)
	sound, _ := notification.Config["sound"].(string)
	device, _ := notification.Config["device"].(string)

	if userKey == "" {
		return fmt.Errorf("user_key is required")
	}

	if apiToken == "" {
		return fmt.Errorf("api_token is required")
	}

	// Default priority based on status
	if priority == 0 {
		if message.Status == "down" {
			priority = 1 // High priority
		} else {
			priority = 0 // Normal priority
		}
	}

	// Build message text
	messageText := FormatMessage(message)

	// Build form data
	data := url.Values{}
	data.Set("token", apiToken)
	data.Set("user", userKey)
	data.Set("title", message.Title)
	data.Set("message", messageText)
	data.Set("priority", fmt.Sprintf("%d", int(priority)))

	if sound != "" {
		data.Set("sound", sound)
	}

	if device != "" {
		data.Set("device", device)
	}

	if message.MonitorURL != "" {
		data.Set("url", message.MonitorURL)
		data.Set("url_title", "View Monitor")
	}

	// Send to Pushover API
	apiURL := "https://api.pushover.net/1/messages.json"

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Pushover notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Pushover API returned status %d", resp.StatusCode)
	}

	return nil
}

func (p *PushoverProvider) Validate(config map[string]interface{}) error {
	userKey, ok := config["user_key"].(string)
	if !ok || userKey == "" {
		return fmt.Errorf("user_key is required")
	}

	apiToken, ok := config["api_token"].(string)
	if !ok || apiToken == "" {
		return fmt.Errorf("api_token is required")
	}

	return nil
}
