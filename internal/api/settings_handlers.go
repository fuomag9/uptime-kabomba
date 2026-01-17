package api

import (
	"encoding/json"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/fuomag9/uptime-kabomba/internal/models"
)

// HandleGetUserSettings returns the current user's settings
func HandleGetUserSettings(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)

		var settings models.UserSettings
		result := db.Where("user_id = ?", user.ID).First(&settings)

		// Create default settings if not exist
		if result.Error == gorm.ErrRecordNotFound {
			settings = models.DefaultUserSettings(user.ID)
			settings.CreatedAt = time.Now()
			settings.UpdatedAt = time.Now()
			if err := db.Create(&settings).Error; err != nil {
				http.Error(w, "Failed to create default settings", http.StatusInternalServerError)
				return
			}
		} else if result.Error != nil {
			http.Error(w, "Failed to fetch settings", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(settings)
	}
}

// UpdateUserSettingsRequest represents the request body for updating settings
type UpdateUserSettingsRequest struct {
	HeartbeatRetentionDays  int `json:"heartbeat_retention_days"`
	HourlyStatRetentionDays int `json:"hourly_stat_retention_days"`
	DailyStatRetentionDays  int `json:"daily_stat_retention_days"`
}

// HandleUpdateUserSettings updates the user's settings
func HandleUpdateUserSettings(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)

		var req UpdateUserSettingsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Create settings object for validation
		newSettings := models.UserSettings{
			HeartbeatRetentionDays:  req.HeartbeatRetentionDays,
			HourlyStatRetentionDays: req.HourlyStatRetentionDays,
			DailyStatRetentionDays:  req.DailyStatRetentionDays,
		}

		// Validate retention values
		if err := newSettings.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Get or create existing settings
		var settings models.UserSettings
		result := db.Where("user_id = ?", user.ID).First(&settings)

		if result.Error == gorm.ErrRecordNotFound {
			// Create new settings
			settings = models.UserSettings{
				UserID:                  user.ID,
				HeartbeatRetentionDays:  req.HeartbeatRetentionDays,
				HourlyStatRetentionDays: req.HourlyStatRetentionDays,
				DailyStatRetentionDays:  req.DailyStatRetentionDays,
				CreatedAt:               time.Now(),
				UpdatedAt:               time.Now(),
			}
			if err := db.Create(&settings).Error; err != nil {
				http.Error(w, "Failed to create settings", http.StatusInternalServerError)
				return
			}
		} else if result.Error != nil {
			http.Error(w, "Failed to fetch settings", http.StatusInternalServerError)
			return
		} else {
			// Update existing settings
			settings.HeartbeatRetentionDays = req.HeartbeatRetentionDays
			settings.HourlyStatRetentionDays = req.HourlyStatRetentionDays
			settings.DailyStatRetentionDays = req.DailyStatRetentionDays
			settings.UpdatedAt = time.Now()

			if err := db.Save(&settings).Error; err != nil {
				http.Error(w, "Failed to update settings", http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(settings)
	}
}
