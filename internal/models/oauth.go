package models

import "time"

// OAuthLinkingToken represents a temporary token for linking OAuth accounts to existing users
type OAuthLinkingToken struct {
	ID        int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Token     string    `json:"token" gorm:"uniqueIndex;not null"`
	UserID    int       `json:"user_id" gorm:"not null"`
	Provider  string    `json:"provider" gorm:"not null"`
	Subject   string    `json:"subject" gorm:"not null"`
	Email     string    `json:"email" gorm:"not null"`
	OAuthData *string   `json:"oauth_data"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName specifies the table name for OAuthLinkingToken
func (OAuthLinkingToken) TableName() string {
	return "oauth_linking_tokens"
}

// OAuthSession represents a PKCE session for OAuth authorization flow
type OAuthSession struct {
	ID           int       `json:"id" gorm:"primaryKey;autoIncrement"`
	State        string    `json:"state" gorm:"uniqueIndex;not null"`
	CodeVerifier string    `json:"code_verifier" gorm:"not null"`
	RedirectURI  *string   `json:"redirect_uri"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at" gorm:"not null"`
}

// TableName specifies the table name for OAuthSession
func (OAuthSession) TableName() string {
	return "oauth_sessions"
}
