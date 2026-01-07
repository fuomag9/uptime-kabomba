package models

import "time"

// User represents a user in the system
type User struct {
	ID         int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Username   string    `json:"username" gorm:"uniqueIndex;not null"`
	Password   string    `json:"-" gorm:"not null"` // Never expose password in JSON
	Active     bool      `json:"active" gorm:"default:true"`
	TotpSecret *string   `json:"-" gorm:"column:totp_secret"` // 2FA secret (nullable)
	CreatedAt  time.Time `json:"created_at"`

	// Relationships (optional, for eager loading)
	Monitors      []Monitor      `json:"-" gorm:"foreignKey:UserID"`
	Notifications []Notification `json:"-" gorm:"foreignKey:UserID"`
	APIKeys       []APIKey       `json:"-" gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for User
func (User) TableName() string {
	return "users"
}
