package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID         int        `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID     int        `json:"user_id" gorm:"not null;index"`
	Name       string     `json:"name" gorm:"not null"`
	KeyHash    string     `json:"-" gorm:"not null;uniqueIndex"` // Never send to client
	Prefix     string     `json:"prefix" gorm:"not null;index"`
	ScopesRaw  string     `json:"-" gorm:"column:scopes;type:text"` // JSON storage
	Scopes     []string   `json:"scopes" gorm:"-"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty" gorm:"index"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`

	// Relationship (optional, for eager loading)
	User User `json:"-" gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for APIKey
func (APIKey) TableName() string {
	return "api_keys"
}

// BeforeSave marshals Scopes to JSON (GORM hook)
func (k *APIKey) BeforeSave(tx *gorm.DB) error {
	if k.Scopes != nil {
		scopesJSON, err := json.Marshal(k.Scopes)
		if err != nil {
			return err
		}
		k.ScopesRaw = string(scopesJSON)
	}
	return nil
}

// AfterFind unmarshals Scopes from JSON (GORM hook)
func (k *APIKey) AfterFind(tx *gorm.DB) error {
	if k.ScopesRaw != "" {
		return json.Unmarshal([]byte(k.ScopesRaw), &k.Scopes)
	}
	return nil
}

// HasScope checks if the API key has a specific scope
func (k *APIKey) HasScope(scope string) bool {
	for _, s := range k.Scopes {
		if s == scope || s == "admin" {
			return true
		}
	}
	return false
}

// IsExpired checks if the API key has expired
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return k.ExpiresAt.Before(time.Now())
}
