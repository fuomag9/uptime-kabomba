package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/fuomag9/uptime-kuma-go/internal/models"
	"github.com/fuomag9/uptime-kuma-go/internal/notification"
)

// HandleGetNotifications returns all notifications for the current user
func HandleGetNotificationsV2(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)

		var notifications []models.Notification
		query := `SELECT id, user_id, name, type, config, is_default, active, created_at, updated_at
		          FROM notifications WHERE user_id = ? ORDER BY created_at DESC`

		err := db.Select(&notifications, query, user.ID)
		if err != nil {
			http.Error(w, "Failed to fetch notifications", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(notifications)
	}
}

// HandleGetNotification returns a single notification by ID
func HandleGetNotification(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		notificationID := chi.URLParam(r, "id")

		var notif models.Notification
		query := `SELECT id, user_id, name, type, config, is_default, active, created_at, updated_at
		          FROM notifications WHERE id = ? AND user_id = ?`

		err := db.Get(&notif, query, notificationID, user.ID)
		if err != nil {
			http.Error(w, "Notification not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(notif)
	}
}

// HandleCreateNotification creates a new notification
func HandleCreateNotification(db *sqlx.DB) http.HandlerFunc {
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

		// Insert into database
		query := `
			INSERT INTO notifications (user_id, name, type, config, is_default, active, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			RETURNING id
		`

		var notif models.Notification
		notif.UserID = user.ID
		notif.Name = req.Name
		notif.Type = req.Type
		notif.Config = string(configJSON)
		notif.IsDefault = req.IsDefault
		notif.Active = true
		notif.CreatedAt = time.Now()
		notif.UpdatedAt = time.Now()

		err = db.QueryRow(query,
			notif.UserID, notif.Name, notif.Type, notif.Config,
			notif.IsDefault, notif.Active, notif.CreatedAt, notif.UpdatedAt,
		).Scan(&notif.ID)

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
func HandleUpdateNotification(db *sqlx.DB) http.HandlerFunc {
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
		var count int
		db.Get(&count, "SELECT COUNT(*) FROM notifications WHERE id = ? AND user_id = ?", id, user.ID)
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
		query := `
			UPDATE notifications
			SET name = ?, type = ?, config = ?, is_default = ?, active = ?, updated_at = ?
			WHERE id = ? AND user_id = ?
		`

		_, err = db.Exec(query,
			req.Name, req.Type, string(configJSON), req.IsDefault, req.Active, time.Now(), id, user.ID,
		)

		if err != nil {
			http.Error(w, "Failed to update notification", http.StatusInternalServerError)
			return
		}

		// Return updated notification
		var notif models.Notification
		db.Get(&notif, "SELECT id, user_id, name, type, config, is_default, active, created_at, updated_at FROM notifications WHERE id = ?", id)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(notif)
	}
}

// HandleDeleteNotification deletes a notification
func HandleDeleteNotification(db *sqlx.DB) http.HandlerFunc {
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
		query := `DELETE FROM notifications WHERE id = ? AND user_id = ?`
		result, err := db.Exec(query, id, user.ID)
		if err != nil {
			http.Error(w, "Failed to delete notification", http.StatusInternalServerError)
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			http.Error(w, "Notification not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// HandleTestNotification sends a test notification
func HandleTestNotification(db *sqlx.DB, dispatcher *notification.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		notificationID := chi.URLParam(r, "id")

		// Get notification
		var notif notification.Notification
		query := `SELECT id, user_id, name, type, config, is_default, active, created_at, updated_at
		          FROM notifications WHERE id = ? AND user_id = ?`

		err := db.Get(&notif, query, notificationID, user.ID)
		if err != nil {
			http.Error(w, "Notification not found", http.StatusNotFound)
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
