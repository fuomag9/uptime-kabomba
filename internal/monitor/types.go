package monitor

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// MonitorType interface that all monitor types must implement
type MonitorType interface {
	// Name returns the monitor type name (e.g., "http", "tcp", "ping")
	Name() string

	// Check performs the monitor check and returns a heartbeat
	Check(ctx context.Context, monitor *Monitor) (*Heartbeat, error)

	// Validate validates the monitor configuration
	Validate(monitor *Monitor) error
}

// Monitor represents a monitor configuration
type Monitor struct {
	ID             int                    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID         int                    `json:"user_id" gorm:"not null;index"`
	Name           string                 `json:"name" gorm:"not null"`
	Type           string                 `json:"type" gorm:"not null;index"`
	URL            string                 `json:"url"`
	Interval       int                    `json:"interval" gorm:"default:60"`        // seconds
	Timeout        int                    `json:"timeout" gorm:"default:30"`         // seconds
	ResendInterval int                    `json:"resend_interval" gorm:"default:0"`  // 0=once per downtime period, N=resend every N failures
	IPVersion      string                 `json:"ip_version" gorm:"default:'auto'"`  // auto, ipv4, ipv6
	Active         bool                   `json:"active" gorm:"default:true;index"`
	Config         map[string]interface{} `json:"config" gorm:"-"`                      // Type-specific config (not from DB)
	ConfigRaw      string                 `json:"-" gorm:"column:config;type:text"`     // JSON storage
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// TableName specifies the table name for Monitor
func (Monitor) TableName() string {
	return "monitors"
}

// AfterFind hook to unmarshal ConfigRaw into Config
func (m *Monitor) AfterFind(tx *gorm.DB) error {
	if m.ConfigRaw != "" {
		return json.Unmarshal([]byte(m.ConfigRaw), &m.Config)
	}
	return nil
}

// BeforeSave hook to marshal Config into ConfigRaw
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

// Heartbeat represents a monitor check result
type Heartbeat struct {
	ID        int       `json:"id" gorm:"primaryKey;autoIncrement"`
	MonitorID int       `json:"monitor_id" gorm:"not null;index"`
	Status    int       `json:"status" gorm:"not null;index"` // 0=down, 1=up, 2=pending, 3=maintenance
	Ping      int       `json:"ping"`                         // milliseconds
	Important bool      `json:"important" gorm:"default:false;index"`
	Message   string    `json:"message" gorm:"type:text"`
	Time      time.Time `json:"time" gorm:"not null;index"`
}

// TableName specifies the table name for Heartbeat
func (Heartbeat) TableName() string {
	return "heartbeats"
}

// Status constants
const (
	StatusDown        = 0
	StatusUp          = 1
	StatusPending     = 2
	StatusMaintenance = 3
)

// MonitorRegistry holds all registered monitor types
var monitorTypes = make(map[string]MonitorType)

// RegisterMonitorType registers a monitor type
func RegisterMonitorType(mt MonitorType) {
	monitorTypes[mt.Name()] = mt
}

// GetMonitorType returns a monitor type by name
func GetMonitorType(name string) (MonitorType, bool) {
	mt, ok := monitorTypes[name]
	return mt, ok
}

// GetAllMonitorTypes returns all registered monitor types
func GetAllMonitorTypes() map[string]MonitorType {
	return monitorTypes
}

// GetNetworkForIPVersion returns the appropriate network string for dial/lookup operations
// based on the monitor's IP version preference
func GetNetworkForIPVersion(baseNetwork string, ipVersion string) string {
	// If auto or empty, return the base network (default behavior)
	if ipVersion == "" || ipVersion == "auto" {
		return baseNetwork
	}

	// Force IPv4
	if ipVersion == "ipv4" {
		switch baseNetwork {
		case "tcp":
			return "tcp4"
		case "udp":
			return "udp4"
		case "ip":
			return "ip4"
		default:
			return baseNetwork
		}
	}

	// Force IPv6
	if ipVersion == "ipv6" {
		switch baseNetwork {
		case "tcp":
			return "tcp6"
		case "udp":
			return "udp6"
		case "ip":
			return "ip6"
		default:
			return baseNetwork
		}
	}

	return baseNetwork
}
