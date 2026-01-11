package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Monitor represents a monitor configuration
type Monitor struct {
	ID             int                    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID         int                    `json:"user_id" gorm:"not null;index"`
	Name           string                 `json:"name" gorm:"not null"`
	Type           string                 `json:"type" gorm:"not null;index"`
	URL            string                 `json:"url"`
	Interval       int                    `json:"interval" gorm:"default:60"`        // seconds
	Timeout        int                    `json:"timeout" gorm:"default:30"`         // seconds
	ResendInterval int                    `json:"resend_interval" gorm:"default:1"`  // send notification after X consecutive failures
	Active         bool                   `json:"active" gorm:"default:true;index"`
	Config         map[string]interface{} `json:"config" gorm:"-"`
	ConfigRaw      string                 `json:"-" gorm:"column:config;type:text"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`

	// Relationships (optional, for eager loading)
	User          User           `json:"-" gorm:"foreignKey:UserID"`
	Heartbeats    []Heartbeat    `json:"-" gorm:"foreignKey:MonitorID"`
	Notifications []Notification `json:"-" gorm:"many2many:monitor_notifications"`
}

// TableName specifies the table name for Monitor
func (Monitor) TableName() string {
	return "monitors"
}

// BeforeSave marshals the Config map to JSON before saving (GORM hook)
func (m *Monitor) BeforeSave(tx *gorm.DB) error {
	if m.Config != nil {
		configJSON, err := json.Marshal(m.Config)
		if err != nil {
			return err
		}
		m.ConfigRaw = string(configJSON)
	}
	return nil
}

// AfterFind unmarshals the Config JSON after loading (GORM hook)
func (m *Monitor) AfterFind(tx *gorm.DB) error {
	if m.ConfigRaw != "" {
		return json.Unmarshal([]byte(m.ConfigRaw), &m.Config)
	}
	return nil
}
