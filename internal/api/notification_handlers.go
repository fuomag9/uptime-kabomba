package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/fuomag9/uptime-kabomba/internal/models"
	"github.com/fuomag9/uptime-kabomba/internal/notification"
)

// HandleGetNotifications returns all notifications for the current user
func HandleGetNotificationsV2(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)

		var notifications []models.Notification
		err := db.Where("user_id = ?", user.ID).
			Order("created_at DESC").
			Find(&notifications).Error

		if err != nil {
			http.Error(w, "Failed to fetch notifications", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(notifications)
	}
}

// HandleGetNotification returns a single notification by ID
func HandleGetNotification(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		notificationID := chi.URLParam(r, "id")

		var notif models.Notification
		err := db.Where("id = ? AND user_id = ?", notificationID, user.ID).
			First(&notif).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "Notification not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to fetch notification", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(notif)
	}
}

// HandleCreateNotification creates a new notification
func HandleCreateNotification(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)

		var req struct {
			Name      string                 `json:"name"`
			Type      string                 `json:"type"`
			Config    map[string]interface{} `json:"config"`
			IsDefault bool                   `json:"is_default"`
			Active    bool                   `json:"active"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate provider type
		provider, ok := notification.GetProvider(req.Type)
		if !ok {
			http.Error(w, "Invalid notification type", http.StatusBadRequest)
			return
		}

		// Validate configuration
		if err := provider.Validate(req.Config); err != nil {
			http.Error(w, "Validation failed: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Marshal config to JSON
		configJSON, err := json.Marshal(req.Config)
		if err != nil {
			http.Error(w, "Failed to marshal config", http.StatusInternalServerError)
			return
		}

		// Create notification
		notif := models.Notification{
			UserID:    user.ID,
			Name:      req.Name,
			Type:      req.Type,
			Config:    string(configJSON),
			IsDefault: req.IsDefault,
			Active:    true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err = db.Create(&notif).Error
		if err != nil {
			http.Error(w, "Failed to create notification", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(notif)
	}
}

// HandleUpdateNotification updates an existing notification
func HandleUpdateNotification(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		notificationID := chi.URLParam(r, "id")

		var req struct {
			Name      string                 `json:"name"`
			Type      string                 `json:"type"`
			Config    map[string]interface{} `json:"config"`
			IsDefault bool                   `json:"is_default"`
			Active    bool                   `json:"active"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Parse notification ID
		id, err := strconv.Atoi(notificationID)
		if err != nil {
			http.Error(w, "Invalid notification ID", http.StatusBadRequest)
			return
		}

		// Verify ownership
		var count int64
		db.Model(&models.Notification{}).
			Where("id = ? AND user_id = ?", id, user.ID).
			Count(&count)
		if count == 0 {
			http.Error(w, "Notification not found", http.StatusNotFound)
			return
		}

		// Validate provider type
		provider, ok := notification.GetProvider(req.Type)
		if !ok {
			http.Error(w, "Invalid notification type", http.StatusBadRequest)
			return
		}

		// Validate configuration
		if err := provider.Validate(req.Config); err != nil {
			http.Error(w, "Validation failed: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Marshal config to JSON
		configJSON, err := json.Marshal(req.Config)
		if err != nil {
			http.Error(w, "Failed to marshal config", http.StatusInternalServerError)
			return
		}

		// Update database
		err = db.Model(&models.Notification{}).
			Where("id = ? AND user_id = ?", id, user.ID).
			Updates(map[string]interface{}{
				"name":       req.Name,
				"type":       req.Type,
				"config":     string(configJSON),
				"is_default": req.IsDefault,
				"active":     req.Active,
				"updated_at": time.Now(),
			}).Error

		if err != nil {
			http.Error(w, "Failed to update notification", http.StatusInternalServerError)
			return
		}

		// Return updated notification
		var notif models.Notification
		db.Where("id = ?", id).First(&notif)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(notif)
	}
}

// HandleDeleteNotification deletes a notification
func HandleDeleteNotification(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		notificationID := chi.URLParam(r, "id")

		// Parse notification ID
		id, err := strconv.Atoi(notificationID)
		if err != nil {
			http.Error(w, "Invalid notification ID", http.StatusBadRequest)
			return
		}

		// Delete from database
		result := db.Where("id = ? AND user_id = ?", id, user.ID).
			Delete(&models.Notification{})

		if result.Error != nil {
			http.Error(w, "Failed to delete notification", http.StatusInternalServerError)
			return
		}

		if result.RowsAffected == 0 {
			http.Error(w, "Notification not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// HandleTestNotification sends a test notification
func HandleTestNotification(db *gorm.DB, dispatcher *notification.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		notificationID := chi.URLParam(r, "id")

		// Get notification
		var notif notification.Notification
		err := db.Where("id = ? AND user_id = ?", notificationID, user.ID).
			First(&notif).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "Notification not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to fetch notification", http.StatusInternalServerError)
			}
			return
		}

		// Parse config JSON
		if notif.ConfigRaw != "" {
			var config map[string]interface{}
			if err := json.Unmarshal([]byte(notif.ConfigRaw), &config); err != nil {
				http.Error(w, "Invalid notification configuration", http.StatusInternalServerError)
				return
			}
			notif.Config = config
		}

		// Send test notification
		err = dispatcher.TestNotification(r.Context(), &notif)
		if err != nil {
			http.Error(w, "Failed to send test notification: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Test notification sent successfully"})
	}
}

// HandleGetAvailableProviders returns all available notification providers
func HandleGetAvailableProviders() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providers := notification.GetAllProviders()

		result := make([]map[string]string, 0, len(providers))
		for name := range providers {
			result = append(result, map[string]string{
				"name": name,
				"label": getProviderLabel(name),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// getProviderLabel returns a user-friendly label for a provider
func getProviderLabel(name string) string {
	labels := map[string]string{
		"smtp":       "Email (SMTP)",
		"webhook":    "Webhook",
		"discord":    "Discord",
		"slack":      "Slack",
		"telegram":   "Telegram",
		"teams":      "Microsoft Teams",
		"pagerduty":  "PagerDuty",
		"pushover":   "Pushover",
		"gotify":     "Gotify",
		"ntfy":       "Ntfy",
	}

	if label, ok := labels[name]; ok {
		return label
	}

	return name
}
