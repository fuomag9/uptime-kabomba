package models

import "time"

// User represents a user in the system
type User struct {
	ID         int       `json:"id" db:"id"`
	Username   string    `json:"username" db:"username"`
	Password   string    `json:"-" db:"password"` // Never expose password in JSON
	Active     bool      `json:"active" db:"active"`
	TotpSecret string    `json:"-" db:"totp_secret"` // 2FA secret
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}
