package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// TelegramProvider sends Telegram bot notifications
type TelegramProvider struct{}

func init() {
	RegisterProvider(&TelegramProvider{})
}

func (t *TelegramProvider) Name() string {
	return "telegram"
}

func (t *TelegramProvider) Send(ctx context.Context, notification *Notification, message *Message) error {
	// Get Telegram configuration
	botToken, _ := notification.Config["bot_token"].(string)
	chatID, _ := notification.Config["chat_id"].(string)
	disableNotification, _ := notification.Config["disable_notification"].(bool)

	if botToken == "" {
		return fmt.Errorf("bot_token is required")
	}

	if chatID == "" {
		return fmt.Errorf("chat_id is required")
	}

	// Build message text with HTML formatting
	var statusEmoji string
	switch message.Status {
	case "up":
		statusEmoji = "✅"
	case "down":
		statusEmoji = "❌"
	case "maintenance":
		statusEmoji = "🔧"
	default:
		statusEmoji = "ℹ️"
	}

	text := fmt.Sprintf("<b>%s %s</b>\n\n", statusEmoji, message.Title)
	text += fmt.Sprintf("%s\n\n", message.Body)
	text += fmt.Sprintf("<b>Monitor:</b> %s\n", message.MonitorName)

	if message.MonitorURL != "" {
		text += fmt.Sprintf("<b>URL:</b> %s\n", message.MonitorURL)
	}

	if message.Ping > 0 {
		text += fmt.Sprintf("<b>Response Time:</b> %dms\n", message.Ping)
	}

	text += fmt.Sprintf("<b>Time:</b> %s", message.Time)

	// Build payload
	payload := map[string]interface{}{
		"chat_id":              chatID,
		"text":                 text,
		"parse_mode":           "HTML",
		"disable_notification": disableNotification,
	}

	// Marshal payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Send message via Telegram Bot API
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Telegram API returned status %d", resp.StatusCode)
	}

	// Parse response to check for errors
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode Telegram response: %w", err)
	}

	if ok, _ := result["ok"].(bool); !ok {
		description, _ := result["description"].(string)
		return fmt.Errorf("Telegram API error: %s", description)
	}

	return nil
}

func (t *TelegramProvider) Validate(config map[string]interface{}) error {
	botToken, ok := config["bot_token"].(string)
	if !ok || botToken == "" {
		return fmt.Errorf("bot_token is required")
	}

	chatID, ok := config["chat_id"].(string)
	if !ok || chatID == "" {
		return fmt.Errorf("chat_id is required")
	}

	return nil
}
