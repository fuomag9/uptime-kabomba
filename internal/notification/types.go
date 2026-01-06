package notification

import (
	"context"
	"fmt"
	"sync"
)

// Provider defines the interface for all notification providers
type Provider interface {
	// Name returns the unique identifier for this provider
	Name() string

	// Send sends a notification with the given message
	Send(ctx context.Context, notification *Notification, message *Message) error

	// Validate validates the provider configuration
	Validate(config map[string]interface{}) error
}

// Notification represents a notification configuration
type Notification struct {
	ID        int                    `json:"id" db:"id"`
	UserID    int                    `json:"user_id" db:"user_id"`
	Name      string                 `json:"name" db:"name"`
	Type      string                 `json:"type" db:"type"` // smtp, webhook, discord, etc.
	Config    map[string]interface{} `json:"config"`
	ConfigRaw string                 `json:"-" db:"config"` // JSON storage
	IsDefault bool                   `json:"is_default" db:"is_default"`
	Active    bool                   `json:"active" db:"active"`
	CreatedAt string                 `json:"created_at" db:"created_at"`
	UpdatedAt string                 `json:"updated_at" db:"updated_at"`
}

// Message represents a notification message to be sent
type Message struct {
	Title       string
	Body        string
	MonitorName string
	MonitorURL  string
	Status      string // "up", "down", "maintenance"
	Ping        int    // milliseconds
	Time        string
	Important   bool
}

// Registry holds all registered notification providers
var (
	providers = make(map[string]Provider)
	mu        sync.RWMutex
)

// RegisterProvider registers a new notification provider
func RegisterProvider(provider Provider) {
	mu.Lock()
	defer mu.Unlock()
	providers[provider.Name()] = provider
}

// GetProvider returns a provider by name
func GetProvider(name string) (Provider, bool) {
	mu.RLock()
	defer mu.RUnlock()
	provider, ok := providers[name]
	return provider, ok
}

// GetAllProviders returns all registered providers
func GetAllProviders() map[string]Provider {
	mu.RLock()
	defer mu.RUnlock()
	result := make(map[string]Provider)
	for k, v := range providers {
		result[k] = v
	}
	return result
}

// FormatMessage formats a notification message with common details
func FormatMessage(msg *Message) string {
	var statusEmoji string
	switch msg.Status {
	case "up":
		statusEmoji = "✅"
	case "down":
		statusEmoji = "❌"
	case "maintenance":
		statusEmoji = "🔧"
	default:
		statusEmoji = "ℹ️"
	}

	body := fmt.Sprintf("%s %s\n\n", statusEmoji, msg.Title)
	body += msg.Body + "\n\n"
	body += fmt.Sprintf("Monitor: %s\n", msg.MonitorName)

	if msg.MonitorURL != "" {
		body += fmt.Sprintf("URL: %s\n", msg.MonitorURL)
	}

	if msg.Ping > 0 {
		body += fmt.Sprintf("Response Time: %dms\n", msg.Ping)
	}

	body += fmt.Sprintf("Time: %s\n", msg.Time)

	return body
}
