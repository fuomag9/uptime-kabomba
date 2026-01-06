package notification

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// NtfyProvider sends Ntfy notifications (self-hosted or ntfy.sh)
type NtfyProvider struct{}

func init() {
	RegisterProvider(&NtfyProvider{})
}

func (n *NtfyProvider) Name() string {
	return "ntfy"
}

func (n *NtfyProvider) Send(ctx context.Context, notification *Notification, message *Message) error {
	// Get Ntfy configuration
	serverURL, _ := notification.Config["server_url"].(string)
	topic, _ := notification.Config["topic"].(string)
	priority, _ := notification.Config["priority"].(float64)
	username, _ := notification.Config["username"].(string)
	password, _ := notification.Config["password"].(string)

	// Default server to ntfy.sh
	if serverURL == "" {
		serverURL = "https://ntfy.sh"
	}

	if topic == "" {
		return fmt.Errorf("topic is required")
	}

	// Default priority based on status
	if priority == 0 {
		if message.Status == "down" {
			priority = 4 // High priority
		} else {
			priority = 3 // Default priority
		}
	}

	// Build message text
	messageText := FormatMessage(message)

	// Send to Ntfy server
	url := fmt.Sprintf("%s/%s", serverURL, topic)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(messageText))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Title", message.Title)
	req.Header.Set("Priority", fmt.Sprintf("%d", int(priority)))
	req.Header.Set("Tags", getTagsForStatus(message.Status))

	// Add authentication if provided
	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	// Add action if URL is available
	if message.MonitorURL != "" {
		req.Header.Set("Actions", fmt.Sprintf("view, View Monitor, %s", message.MonitorURL))
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Ntfy notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Ntfy server returned status %d", resp.StatusCode)
	}

	return nil
}

func (n *NtfyProvider) Validate(config map[string]interface{}) error {
	topic, ok := config["topic"].(string)
	if !ok || topic == "" {
		return fmt.Errorf("topic is required")
	}

	return nil
}

// getTagsForStatus returns emoji tags based on status
func getTagsForStatus(status string) string {
	switch status {
	case "up":
		return "white_check_mark,+1"
	case "down":
		return "x,warning"
	case "maintenance":
		return "wrench,tools"
	default:
		return "information_source"
	}
}
