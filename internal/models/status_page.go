package models

import "time"

// StatusPage represents a public status page
type StatusPage struct {
	ID            int       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID        int       `json:"user_id" gorm:"not null;index"`
	Slug          string    `json:"slug" gorm:"uniqueIndex;not null"`
	Title         string    `json:"title" gorm:"not null"`
	Description   string    `json:"description"`
	Published     bool      `json:"published" gorm:"default:false;index"`
	ShowPoweredBy bool      `json:"show_powered_by" gorm:"default:true"`
	Theme         string    `json:"theme" gorm:"default:light"`
	CustomCSS     string    `json:"custom_css" gorm:"type:text"`
	Password      string    `json:"-"` // Never send to client
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Relationships (optional, for eager loading)
	User      User        `json:"-" gorm:"foreignKey:UserID"`
	Monitors  []Monitor   `json:"-" gorm:"many2many:status_page_monitors"`
	Incidents []Incident  `json:"-" gorm:"foreignKey:StatusPageID"`
}

// TableName specifies the table name for StatusPage
func (StatusPage) TableName() string {
	return "status_pages"
}

// StatusPageMonitor represents a monitor displayed on a status page
type StatusPageMonitor struct {
	StatusPageID int `json:"status_page_id" gorm:"primaryKey"`
	MonitorID    int `json:"monitor_id" gorm:"primaryKey"`
	DisplayOrder int `json:"display_order" gorm:"default:0"`

	// Relationships (optional, for eager loading)
	StatusPage StatusPage `json:"-" gorm:"foreignKey:StatusPageID"`
	Monitor    Monitor    `json:"-" gorm:"foreignKey:MonitorID"`
}

// TableName specifies the table name for StatusPageMonitor
func (StatusPageMonitor) TableName() string {
	return "status_page_monitors"
}

// Incident represents an incident posted on a status page
type Incident struct {
	ID           int       `json:"id" gorm:"primaryKey;autoIncrement"`
	StatusPageID int       `json:"status_page_id" gorm:"not null;index"`
	Title        string    `json:"title" gorm:"not null"`
	Content      string    `json:"content" gorm:"type:text"`
	Style        string    `json:"style" gorm:"default:info"` // info, warning, danger, success
	Pin          bool      `json:"pin" gorm:"default:false;index"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Relationship (optional, for eager loading)
	StatusPage StatusPage `json:"-" gorm:"foreignKey:StatusPageID"`
}

// TableName specifies the table name for Incident
func (Incident) TableName() string {
	return "incidents"
}

// StatusPageWithMonitors is a status page with its monitors
type StatusPageWithMonitors struct {
	StatusPage
	Monitors []Monitor `json:"monitors"`
}
