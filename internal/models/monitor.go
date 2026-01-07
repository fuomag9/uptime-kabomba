package models

import (
	"encoding/json"
	"time"
)

// Monitor represents a monitor configuration
type Monitor struct {
	ID        int                    `json:"id" db:"id"`
	UserID    int                    `json:"user_id" db:"user_id"`
	Name      string                 `json:"name" db:"name"`
	Type      string                 `json:"type" db:"type"`
	URL       string                 `json:"url" db:"url"`
	Interval  int                    `json:"interval" db:"interval"` // seconds
	Timeout   int                    `json:"timeout" db:"timeout"`   // seconds
	Active    bool                   `json:"active" db:"active"`
	Config    map[string]interface{} `json:"config" db:"-"` // Not from DB
	ConfigRaw string                 `json:"-" db:"config"`  // JSON storage
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" db:"updated_at"`
}

// BeforeSave marshals the Config map to JSON before saving
func (m *Monitor) BeforeSave() error {
	if m.Config != nil {
		configJSON, err := json.Marshal(m.Config)
		if err != nil {
			return err
		}
		m.ConfigRaw = string(configJSON)
	}
	return nil
}

// AfterLoad unmarshals the Config JSON after loading
func (m *Monitor) AfterLoad() error {
	if m.ConfigRaw != "" {
		return json.Unmarshal([]byte(m.ConfigRaw), &m.Config)
	}
	return nil
}
