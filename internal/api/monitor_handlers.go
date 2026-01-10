package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/fuomag9/uptime-kabomba/internal/models"
	"github.com/fuomag9/uptime-kabomba/internal/monitor"
)

// MonitorExecutor interface for monitor execution
type MonitorExecutor interface {
	StartMonitor(m *monitor.Monitor)
	StopMonitor(monitorID int)
}

// MonitorWithStatus includes monitor data with its last heartbeat
type MonitorWithStatus struct {
	models.Monitor
	LastHeartbeat *models.Heartbeat `json:"last_heartbeat,omitempty"`
}

// HandleGetMonitors returns all monitors for the current user with their last heartbeat
func HandleGetMonitors(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)

		var monitors []models.Monitor
		err := db.Where("user_id = ?", user.ID).
			Order("created_at DESC").
			Find(&monitors).Error

		if err != nil {
			http.Error(w, "Failed to fetch monitors", http.StatusInternalServerError)
			return
		}

		// AfterFind hook automatically unmarshals Config JSON

		// Fetch last heartbeat for each monitor
		monitorsWithStatus := make([]MonitorWithStatus, len(monitors))
		for i, mon := range monitors {
			monitorsWithStatus[i] = MonitorWithStatus{
				Monitor: mon,
			}

			var lastHeartbeat models.Heartbeat
			err := db.Where("monitor_id = ?", mon.ID).
				Order("time DESC").
				Limit(1).
				First(&lastHeartbeat).Error

			if err == nil {
				monitorsWithStatus[i].LastHeartbeat = &lastHeartbeat
			}
			// Ignore error if no heartbeat exists yet
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(monitorsWithStatus)
	}
}

// HandleGetMonitor returns a single monitor by ID
func HandleGetMonitor(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		monitorID := chi.URLParam(r, "id")

		var mon models.Monitor
		err := db.Where("id = ? AND user_id = ?", monitorID, user.ID).
			First(&mon).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "Monitor not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to fetch monitor", http.StatusInternalServerError)
			}
			return
		}

		// AfterFind hook automatically unmarshals Config JSON

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mon)
	}
}

// HandleCreateMonitor creates a new monitor
func HandleCreateMonitor(db *gorm.DB, executor MonitorExecutor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)

		var mon models.Monitor
		if err := json.NewDecoder(r.Body).Decode(&mon); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Set user ID
		mon.UserID = user.ID

		// Set defaults
		if mon.Interval == 0 {
			mon.Interval = 60
		}
		if mon.Timeout == 0 {
			mon.Timeout = 30
		}
		mon.Active = true
		mon.CreatedAt = time.Now()
		mon.UpdatedAt = time.Now()

		// Validate monitor type
		monitorType, ok := monitor.GetMonitorType(mon.Type)
		if !ok {
			http.Error(w, "Invalid monitor type", http.StatusBadRequest)
			return
		}

		// Convert to internal monitor type for validation
		internalMon := &monitor.Monitor{
			Name:     mon.Name,
			Type:     mon.Type,
			URL:      mon.URL,
			Interval: mon.Interval,
			Timeout:  mon.Timeout,
			Config:   mon.Config,
		}

		// Validate configuration
		if err := monitorType.Validate(internalMon); err != nil {
			http.Error(w, "Validation failed: "+err.Error(), http.StatusBadRequest)
			return
		}

		// BeforeSave hook will automatically marshal Config to ConfigRaw

		// Insert into database
		err := db.Create(&mon).Error
		if err != nil {
			http.Error(w, "Failed to create monitor", http.StatusInternalServerError)
			return
		}

		// Start monitoring
		if executor != nil && mon.Active {
			internalMon.ID = mon.ID
			internalMon.UserID = mon.UserID
			internalMon.Active = mon.Active
			internalMon.CreatedAt = mon.CreatedAt
			internalMon.UpdatedAt = mon.UpdatedAt
			executor.StartMonitor(internalMon)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(mon)
	}
}

// HandleUpdateMonitor updates an existing monitor
func HandleUpdateMonitor(db *gorm.DB, executor MonitorExecutor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		monitorID := chi.URLParam(r, "id")

		var mon models.Monitor
		if err := json.NewDecoder(r.Body).Decode(&mon); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Parse monitor ID
		id, err := strconv.Atoi(monitorID)
		if err != nil {
			http.Error(w, "Invalid monitor ID", http.StatusBadRequest)
			return
		}
		mon.ID = id

		// Verify ownership
		var count int64
		db.Model(&models.Monitor{}).
			Where("id = ? AND user_id = ?", mon.ID, user.ID).
			Count(&count)
		if count == 0 {
			http.Error(w, "Monitor not found", http.StatusNotFound)
			return
		}

		// Validate monitor type
		monitorType, ok := monitor.GetMonitorType(mon.Type)
		if !ok {
			http.Error(w, "Invalid monitor type", http.StatusBadRequest)
			return
		}

		// Convert to internal monitor type for validation
		internalMon := &monitor.Monitor{
			ID:       mon.ID,
			UserID:   user.ID,
			Name:     mon.Name,
			Type:     mon.Type,
			URL:      mon.URL,
			Interval: mon.Interval,
			Timeout:  mon.Timeout,
			Active:   mon.Active,
			Config:   mon.Config,
		}

		// Validate configuration
		if err := monitorType.Validate(internalMon); err != nil {
			http.Error(w, "Validation failed: "+err.Error(), http.StatusBadRequest)
			return
		}

		// BeforeSave hook will automatically marshal Config to ConfigRaw
		mon.UpdatedAt = time.Now()

		// Update database
		err = db.Model(&models.Monitor{}).
			Where("id = ? AND user_id = ?", mon.ID, user.ID).
			Updates(map[string]interface{}{
				"name":       mon.Name,
				"type":       mon.Type,
				"url":        mon.URL,
				"interval":   mon.Interval,
				"timeout":    mon.Timeout,
				"active":     mon.Active,
				"config":     mon.ConfigRaw,
				"updated_at": mon.UpdatedAt,
			}).Error

		if err != nil {
			http.Error(w, "Failed to update monitor", http.StatusInternalServerError)
			return
		}

		// Restart monitoring
		if executor != nil {
			executor.StopMonitor(mon.ID)
			if mon.Active {
				executor.StartMonitor(internalMon)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mon)
	}
}

// HandleDeleteMonitor deletes a monitor
func HandleDeleteMonitor(db *gorm.DB, executor MonitorExecutor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		monitorID := chi.URLParam(r, "id")

		// Parse monitor ID
		id, err := strconv.Atoi(monitorID)
		if err != nil {
			http.Error(w, "Invalid monitor ID", http.StatusBadRequest)
			return
		}

		// Stop monitoring
		if executor != nil {
			executor.StopMonitor(id)
		}

		// Delete from database
		result := db.Where("id = ? AND user_id = ?", id, user.ID).
			Delete(&models.Monitor{})

		if result.Error != nil {
			http.Error(w, "Failed to delete monitor", http.StatusInternalServerError)
			return
		}

		if result.RowsAffected == 0 {
			http.Error(w, "Monitor not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// HandleGetHeartbeats returns heartbeats for a monitor
func HandleGetHeartbeats(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		monitorID := chi.URLParam(r, "id")

		// Get limit from query params (default 100)
		limitStr := r.URL.Query().Get("limit")
		limit := 100
		if limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
				limit = l
			}
		}

		var heartbeats []monitor.Heartbeat
		err := db.Where("monitor_id = ?", monitorID).
			Order("time DESC").
			Limit(limit).
			Find(&heartbeats).Error

		if err != nil {
			http.Error(w, "Failed to fetch heartbeats", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(heartbeats)
	}
}

// HandleGetMonitorNotifications returns all notifications linked to a specific monitor
func HandleGetMonitorNotifications(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		monitorIDStr := chi.URLParam(r, "id")

		// Parse monitor ID
		monitorID, err := strconv.Atoi(monitorIDStr)
		if err != nil {
			http.Error(w, "Invalid monitor ID", http.StatusBadRequest)
			return
		}

		// Verify user owns monitor
		var mon models.Monitor
		err = db.Where("id = ? AND user_id = ?", monitorID, user.ID).First(&mon).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "Monitor not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to fetch monitor", http.StatusInternalServerError)
			}
			return
		}

		// Get notifications linked to this monitor
		var notifications []models.Notification
		err = db.Table("notifications").
			Joins("INNER JOIN monitor_notifications ON monitor_notifications.notification_id = notifications.id").
			Where("monitor_notifications.monitor_id = ? AND notifications.user_id = ?", monitorID, user.ID).
			Find(&notifications).Error

		if err != nil {
			http.Error(w, "Failed to fetch notifications", http.StatusInternalServerError)
			return
		}

		// Parse config JSON for each notification
		for i := range notifications {
			if notifications[i].Config != "" {
				configMap := make(map[string]interface{})
				if err := json.Unmarshal([]byte(notifications[i].Config), &configMap); err == nil {
					// Just verify parsing works, frontend will use the string
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(notifications)
	}
}

// UpdateMonitorNotificationsRequest represents the request body for updating monitor notifications
type UpdateMonitorNotificationsRequest struct {
	NotificationIDs []int `json:"notification_ids"`
}

// HandleUpdateMonitorNotifications replaces all notification associations for a monitor
func HandleUpdateMonitorNotifications(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		monitorIDStr := chi.URLParam(r, "id")

		// Parse monitor ID
		monitorID, err := strconv.Atoi(monitorIDStr)
		if err != nil {
			http.Error(w, "Invalid monitor ID", http.StatusBadRequest)
			return
		}

		// Parse request body
		var req UpdateMonitorNotificationsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Verify user owns monitor
		var mon models.Monitor
		err = db.Where("id = ? AND user_id = ?", monitorID, user.ID).First(&mon).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "Monitor not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to fetch monitor", http.StatusInternalServerError)
			}
			return
		}

		// Verify all notification IDs belong to user
		if len(req.NotificationIDs) > 0 {
			var count int64
			err = db.Model(&models.Notification{}).
				Where("id IN ? AND user_id = ?", req.NotificationIDs, user.ID).
				Count(&count).Error

			if err != nil {
				http.Error(w, "Failed to verify notifications", http.StatusInternalServerError)
				return
			}

			if int(count) != len(req.NotificationIDs) {
				http.Error(w, "One or more notification IDs are invalid", http.StatusBadRequest)
				return
			}
		}

		// Use transaction to replace associations
		err = db.Transaction(func(tx *gorm.DB) error {
			// Delete existing associations
			if err := tx.Exec("DELETE FROM monitor_notifications WHERE monitor_id = ?", monitorID).Error; err != nil {
				return err
			}

			// Insert new associations
			for _, notificationID := range req.NotificationIDs {
				if err := tx.Exec("INSERT INTO monitor_notifications (monitor_id, notification_id) VALUES (?, ?)", monitorID, notificationID).Error; err != nil {
					return err
				}
			}

			return nil
		})

		if err != nil {
			http.Error(w, "Failed to update monitor notifications", http.StatusInternalServerError)
			return
		}

		// Return updated list of notifications
		var notifications []models.Notification
		err = db.Table("notifications").
			Joins("INNER JOIN monitor_notifications ON monitor_notifications.notification_id = notifications.id").
			Where("monitor_notifications.monitor_id = ? AND notifications.user_id = ?", monitorID, user.ID).
			Find(&notifications).Error

		if err != nil {
			http.Error(w, "Failed to fetch updated notifications", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(notifications)
	}
}
