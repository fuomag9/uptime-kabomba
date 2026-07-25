package models

import "time"

// RefreshToken represents a long-lived, rotating credential used to mint new
// access tokens without re-entering a password. The raw token is never
// stored - only a fast hash of it (see internal/api/auth_tokens.go).
type RefreshToken struct {
	ID        int        `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    int        `json:"user_id" gorm:"not null;index"`
	TokenHash string     `json:"-" gorm:"column:token_hash;uniqueIndex;not null"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// TableName specifies the table name for RefreshToken
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// RevokedAccessToken records a JWT access token (by jti) that must be
// rejected even though it hasn't expired yet.
type RevokedAccessToken struct {
	JTI       string    `json:"jti" gorm:"column:jti;primaryKey"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	RevokedAt time.Time `json:"revoked_at"`
}

// TableName specifies the table name for RevokedAccessToken
func (RevokedAccessToken) TableName() string {
	return "revoked_access_tokens"
}
