package models

import "time"

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
