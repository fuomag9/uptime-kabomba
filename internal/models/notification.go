package models

import "time"

// Notification represents a notification configuration
type Notification struct {
	ID        int       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    int       `json:"user_id" gorm:"not null;index"`
	Name      string    `json:"name" gorm:"not null"`
	Type      string    `json:"type" gorm:"not null"`
	Config    string    `json:"-" gorm:"type:text"` // JSON storage
	IsDefault bool      `json:"is_default" gorm:"default:false"`
	Active    bool      `json:"active" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships (optional, for eager loading)
	User     User      `json:"-" gorm:"foreignKey:UserID"`
	Monitors []Monitor `json:"-" gorm:"many2many:monitor_notifications"`
}

// TableName specifies the table name for Notification
func (Notification) TableName() string {
	return "notifications"
}

// MonitorNotification links monitors to notifications
type MonitorNotification struct {
	MonitorID      int `gorm:"primaryKey"`
	NotificationID int `gorm:"primaryKey"`

	// Relationships (optional, for eager loading)
	Monitor      Monitor      `json:"-" gorm:"foreignKey:MonitorID"`
	Notification Notification `json:"-" gorm:"foreignKey:NotificationID"`
}

// TableName specifies the table name for MonitorNotification
func (MonitorNotification) TableName() string {
	return "monitor_notifications"
}
