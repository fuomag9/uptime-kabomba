package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// Dispatcher handles sending notifications
type Dispatcher struct {
	db *gorm.DB
}

// NewDispatcher creates a new notification dispatcher
func NewDispatcher(db *gorm.DB) *Dispatcher {
	return &Dispatcher{db: db}
}

// NotifyMonitorDown sends notifications when a monitor goes down
func (d *Dispatcher) NotifyMonitorDown(ctx context.Context, monitorID int, monitorName, monitorURL string, ping int, message string) error {
	return d.sendMonitorNotifications(ctx, monitorID, &Message{
		Title:       "Monitor is DOWN",
		Body:        message,
		MonitorName: monitorName,
		MonitorURL:  monitorURL,
		Status:      "down",
		Ping:        ping,
		Time:        time.Now().Format(time.RFC3339),
		Important:   true,
	})
}

// NotifyMonitorUp sends notifications when a monitor comes back up
func (d *Dispatcher) NotifyMonitorUp(ctx context.Context, monitorID int, monitorName, monitorURL string, ping int, message string) error {
	return d.sendMonitorNotifications(ctx, monitorID, &Message{
		Title:       "Monitor is UP",
		Body:        message,
		MonitorName: monitorName,
		MonitorURL:  monitorURL,
		Status:      "up",
		Ping:        ping,
		Time:        time.Now().Format(time.RFC3339),
		Important:   false,
	})
}

// sendMonitorNotifications sends notifications to all configured providers for a monitor
func (d *Dispatcher) sendMonitorNotifications(ctx context.Context, monitorID int, msg *Message) error {
	// Get all notifications linked to this monitor
	notifications, err := d.getMonitorNotifications(monitorID)
	if err != nil {
		return fmt.Errorf("failed to get monitor notifications: %w", err)
	}

	// If no specific notifications, get default notifications
	if len(notifications) == 0 {
		notifications, err = d.getDefaultNotifications()
		if err != nil {
			return fmt.Errorf("failed to get default notifications: %w", err)
		}
	}

	// Send to all notifications concurrently
	errCh := make(chan error, len(notifications))
	for _, notif := range notifications {
		go func(n *Notification) {
			if err := d.sendNotification(ctx, n, msg); err != nil {
				log.Printf("Failed to send notification via %s (%s): %v", n.Type, n.Name, err)
				errCh <- err
			} else {
				errCh <- nil
			}
		}(notif)
	}

	// Collect results
	var errors []error
	for i := 0; i < len(notifications); i++ {
		if err := <-errCh; err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to send %d/%d notifications", len(errors), len(notifications))
	}

	return nil
}

// sendNotification sends a notification using the appropriate provider
func (d *Dispatcher) sendNotification(ctx context.Context, notif *Notification, msg *Message) error {
	if !notif.Active {
		return nil // Skip inactive notifications
	}

	provider, ok := GetProvider(notif.Type)
	if !ok {
		return fmt.Errorf("unknown notification provider: %s", notif.Type)
	}

	return provider.Send(ctx, notif, msg)
}

// getMonitorNotifications gets all notifications linked to a monitor
func (d *Dispatcher) getMonitorNotifications(monitorID int) ([]*Notification, error) {
	query := `
		SELECT n.id, n.user_id, n.name, n.type, n.config, n.is_default, n.active, n.created_at, n.updated_at
		FROM notifications n
		INNER JOIN monitor_notifications mn ON n.id = mn.notification_id
		WHERE mn.monitor_id = ? AND n.active = true
	`

	var notifications []*Notification
	err := d.db.Raw(query, monitorID).Scan(&notifications).Error
	if err != nil {
		return nil, err
	}

	// Parse config JSON for each notification
	for _, notif := range notifications {
		if notif.ConfigRaw != "" {
			var config map[string]interface{}
			if err := json.Unmarshal([]byte(notif.ConfigRaw), &config); err != nil {
				log.Printf("Failed to parse notification config for %s: %v", notif.Name, err)
				continue
			}
			notif.Config = config
		}
	}

	return notifications, nil
}

// getDefaultNotifications gets all default notifications for a user
func (d *Dispatcher) getDefaultNotifications() ([]*Notification, error) {
	query := `
		SELECT id, user_id, name, type, config, is_default, active, created_at, updated_at
		FROM notifications
		WHERE is_default = true AND active = true
	`

	var notifications []*Notification
	err := d.db.Raw(query).Scan(&notifications).Error
	if err != nil {
		return nil, err
	}

	// Parse config JSON for each notification
	for _, notif := range notifications {
		if notif.ConfigRaw != "" {
			var config map[string]interface{}
			if err := json.Unmarshal([]byte(notif.ConfigRaw), &config); err != nil {
				log.Printf("Failed to parse notification config for %s: %v", notif.Name, err)
				continue
			}
			notif.Config = config
		}
	}

	return notifications, nil
}

// TestNotification sends a test notification
func (d *Dispatcher) TestNotification(ctx context.Context, notif *Notification) error {
	msg := &Message{
		Title:       "Test Notification",
		Body:        "This is a test notification from Uptime Kabomba.",
		MonitorName: "Test Monitor",
		Status:      "up",
		Time:        time.Now().Format(time.RFC3339),
		Important:   false,
	}

	return d.sendNotification(ctx, notif, msg)
}
