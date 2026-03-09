package models

import "time"

// Certificate stores a client TLS certificate for mTLS connections.
// key_pem is never serialised back to API callers — use the json:"-" tag.
type Certificate struct {
	ID        int       `json:"id"         gorm:"primaryKey;autoIncrement"`
	UserID    int       `json:"user_id"    gorm:"not null;index"`
	Name      string    `json:"name"       gorm:"not null"`
	CertPEM   string    `json:"cert_pem"   gorm:"not null"`
	KeyPEM    string    `json:"-"          gorm:"not null"`  // never returned by API
	CAPEM     string    `json:"ca_pem"`
	User      User      `json:"-"          gorm:"foreignKey:UserID"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Certificate) TableName() string { return "certificates" }
