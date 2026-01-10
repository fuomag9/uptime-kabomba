package models

import "time"

// User represents a user in the system
type User struct {
	ID         int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Username   string    `json:"username" gorm:"uniqueIndex;not null"`
	Email      *string   `json:"email,omitempty" gorm:"uniqueIndex"` // Email (nullable, unique)
	Password   string    `json:"-" gorm:"not null"`                   // Never expose password in JSON
	Provider   *string   `json:"provider,omitempty"`                  // Auth provider: 'local' or 'oidc'
	Subject    *string   `json:"subject,omitempty"`                   // OAuth subject (sub claim)
	OAuthData  *string   `json:"-" gorm:"column:oauth_data"`          // JSON blob for OAuth data
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

// IsOAuthUser returns true if user authenticated via OAuth
func (u *User) IsOAuthUser() bool {
	return u.Provider != nil && *u.Provider != "local"
}

// HasPassword returns true if user has a password set
func (u *User) HasPassword() bool {
	return u.Password != "" && u.Password != "oauth-no-password"
}
