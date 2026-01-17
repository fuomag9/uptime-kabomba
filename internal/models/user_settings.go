package models

import (
	"fmt"
	"time"
)

// UserSettings holds user-configurable settings like data retention periods
type UserSettings struct {
	ID                      int       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID                  int       `json:"user_id" gorm:"uniqueIndex;not null"`
	HeartbeatRetentionDays  int       `json:"heartbeat_retention_days" gorm:"default:90;not null"`
	HourlyStatRetentionDays int       `json:"hourly_stat_retention_days" gorm:"default:365;not null"`
	DailyStatRetentionDays  int       `json:"daily_stat_retention_days" gorm:"default:730;not null"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`

	// Relationships
	User User `json:"-" gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for UserSettings
func (UserSettings) TableName() string {
	return "user_settings"
}

// Validate checks if retention values are within acceptable ranges
func (s *UserSettings) Validate() error {
	if s.HeartbeatRetentionDays < 7 || s.HeartbeatRetentionDays > 365 {
		return fmt.Errorf("heartbeat retention must be between 7 and 365 days")
	}
	if s.HourlyStatRetentionDays < 30 || s.HourlyStatRetentionDays > 730 {
		return fmt.Errorf("hourly stat retention must be between 30 and 730 days")
	}
	if s.DailyStatRetentionDays < 90 || s.DailyStatRetentionDays > 1825 {
		return fmt.Errorf("daily stat retention must be between 90 and 1825 days")
	}
	return nil
}

// DefaultUserSettings returns settings with default values
func DefaultUserSettings(userID int) UserSettings {
	return UserSettings{
		UserID:                  userID,
		HeartbeatRetentionDays:  90,
		HourlyStatRetentionDays: 365,
		DailyStatRetentionDays:  730,
	}
}
