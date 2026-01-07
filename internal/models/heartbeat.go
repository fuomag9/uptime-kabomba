package models

import "time"

// Heartbeat represents a monitor check result
type Heartbeat struct {
	ID        int       `json:"id" gorm:"primaryKey;autoIncrement"`
	MonitorID int       `json:"monitor_id" gorm:"not null;index:idx_monitor_time"`
	Status    int       `json:"status" gorm:"not null"` // 0=down, 1=up, 2=pending, 3=maintenance
	Ping      int       `json:"ping"`                   // milliseconds
	Important bool      `json:"important" gorm:"default:false"`
	Message   string    `json:"message"`
	Time      time.Time `json:"time" gorm:"not null;index:idx_monitor_time,sort:desc;index:idx_time"`

	// Relationship (optional, for eager loading)
	Monitor Monitor `json:"-" gorm:"foreignKey:MonitorID"`
}

// TableName specifies the table name for Heartbeat
func (Heartbeat) TableName() string {
	return "heartbeats"
}
