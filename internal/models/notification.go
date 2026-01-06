package models

import "time"

// Notification represents a notification configuration
type Notification struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Name      string    `json:"name" db:"name"`
	Type      string    `json:"type" db:"type"`
	Config    string    `json:"-" db:"config"` // JSON storage
	IsDefault bool      `json:"is_default" db:"is_default"`
	Active    bool      `json:"active" db:"active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// MonitorNotification links monitors to notifications
type MonitorNotification struct {
	MonitorID      int `db:"monitor_id"`
	NotificationID int `db:"notification_id"`
}
