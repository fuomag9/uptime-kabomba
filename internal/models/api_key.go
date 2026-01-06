package models

import (
	"encoding/json"
	"time"
)

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID         int       `json:"id" db:"id"`
	UserID     int       `json:"user_id" db:"user_id"`
	Name       string    `json:"name" db:"name"`
	KeyHash    string    `json:"-" db:"key_hash"` // Never send to client
	Prefix     string    `json:"prefix" db:"prefix"`
	ScopesRaw  string    `json:"-" db:"scopes"` // JSON storage
	Scopes     []string  `json:"scopes"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// BeforeSave marshals Scopes to JSON
func (k *APIKey) BeforeSave() error {
	if k.Scopes != nil {
		scopesJSON, err := json.Marshal(k.Scopes)
		if err != nil {
			return err
		}
		k.ScopesRaw = string(scopesJSON)
	}
	return nil
}

// AfterLoad unmarshals Scopes from JSON
func (k *APIKey) AfterLoad() error {
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
