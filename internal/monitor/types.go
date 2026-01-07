package monitor

import (
	"context"
	"time"
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
	ID       int                    `json:"id" db:"id"`
	UserID   int                    `json:"user_id" db:"user_id"`
	Name     string                 `json:"name" db:"name"`
	Type     string                 `json:"type" db:"type"`
	URL      string                 `json:"url" db:"url"`
	Interval int                    `json:"interval" db:"interval"` // seconds
	Timeout  int                    `json:"timeout" db:"timeout"`   // seconds
	Active   bool                   `json:"active" db:"active"`
	Config   map[string]interface{} `json:"config" db:"-"` // Type-specific config (not from DB)
	ConfigJSON string                `json:"-" db:"config"` // JSON storage
	CreatedAt time.Time             `json:"created_at" db:"created_at"`
	UpdatedAt time.Time             `json:"updated_at" db:"updated_at"`
}

// Heartbeat represents a monitor check result
type Heartbeat struct {
	ID        int       `json:"id" db:"id"`
	MonitorID int       `json:"monitor_id" db:"monitor_id"`
	Status    int       `json:"status" db:"status"` // 0=down, 1=up, 2=pending, 3=maintenance
	Ping      int       `json:"ping" db:"ping"`     // milliseconds
	Important bool      `json:"important" db:"important"`
	Message   string    `json:"message" db:"message"`
	Time      time.Time `json:"time" db:"time"`
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
