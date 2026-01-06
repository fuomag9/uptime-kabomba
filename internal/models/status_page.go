package models

import "time"

// StatusPage represents a public status page
type StatusPage struct {
	ID            int       `json:"id" db:"id"`
	UserID        int       `json:"user_id" db:"user_id"`
	Slug          string    `json:"slug" db:"slug"`
	Title         string    `json:"title" db:"title"`
	Description   string    `json:"description" db:"description"`
	Published     bool      `json:"published" db:"published"`
	ShowPoweredBy bool      `json:"show_powered_by" db:"show_powered_by"`
	Theme         string    `json:"theme" db:"theme"`
	CustomCSS     string    `json:"custom_css" db:"custom_css"`
	Password      string    `json:"-" db:"password"` // Never send to client
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// StatusPageMonitor represents a monitor displayed on a status page
type StatusPageMonitor struct {
	StatusPageID int `json:"status_page_id" db:"status_page_id"`
	MonitorID    int `json:"monitor_id" db:"monitor_id"`
	DisplayOrder int `json:"display_order" db:"display_order"`
}

// Incident represents an incident posted on a status page
type Incident struct {
	ID           int       `json:"id" db:"id"`
	StatusPageID int       `json:"status_page_id" db:"status_page_id"`
	Title        string    `json:"title" db:"title"`
	Content      string    `json:"content" db:"content"`
	Style        string    `json:"style" db:"style"` // info, warning, danger, success
	Pin          bool      `json:"pin" db:"pin"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// StatusPageWithMonitors is a status page with its monitors
type StatusPageWithMonitors struct {
	StatusPage
	Monitors []Monitor `json:"monitors"`
}
