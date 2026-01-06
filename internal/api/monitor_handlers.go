package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/fuomag9/uptime-kuma-go/internal/models"
	"github.com/fuomag9/uptime-kuma-go/internal/monitor"
)

// MonitorExecutor interface for monitor execution
type MonitorExecutor interface {
	StartMonitor(m *monitor.Monitor)
	StopMonitor(monitorID int)
}

// HandleGetMonitors returns all monitors for the current user
func HandleGetMonitors(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)

		var monitors []models.Monitor
		query := `SELECT id, user_id, name, type, url, interval, timeout, active, config, created_at, updated_at
		          FROM monitors WHERE user_id = $1 ORDER BY created_at DESC`

		err := db.Select(&monitors, query, user.ID)
		if err != nil {
			http.Error(w, "Failed to fetch monitors", http.StatusInternalServerError)
			return
		}

		// Parse config JSON for each monitor
		for i := range monitors {
			if monitors[i].ConfigRaw != "" {
				json.Unmarshal([]byte(monitors[i].ConfigRaw), &monitors[i].Config)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(monitors)
	}
}

// HandleGetMonitor returns a single monitor by ID
func HandleGetMonitor(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		monitorID := chi.URLParam(r, "id")

		var mon models.Monitor
		query := `SELECT id, user_id, name, type, url, interval, timeout, active, config, created_at, updated_at
		          FROM monitors WHERE id = $1 AND user_id = $2`

		err := db.Get(&mon, query, monitorID, user.ID)
		if err != nil {
			http.Error(w, "Monitor not found", http.StatusNotFound)
			return
		}

		// Parse config JSON
		if mon.ConfigRaw != "" {
			json.Unmarshal([]byte(mon.ConfigRaw), &mon.Config)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mon)
	}
}

// HandleCreateMonitor creates a new monitor
func HandleCreateMonitor(db *sqlx.DB, executor MonitorExecutor) http.HandlerFunc {
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

		// Marshal config to JSON
		configJSON, err := json.Marshal(mon.Config)
		if err != nil {
			http.Error(w, "Failed to marshal config", http.StatusInternalServerError)
			return
		}
		mon.ConfigRaw = string(configJSON)

		// Insert into database
		query := `
			INSERT INTO monitors (user_id, name, type, url, interval, timeout, active, config, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id
		`

		err = db.QueryRow(query,
			mon.UserID, mon.Name, mon.Type, mon.URL, mon.Interval, mon.Timeout,
			mon.Active, mon.ConfigRaw, mon.CreatedAt, mon.UpdatedAt,
		).Scan(&mon.ID)

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
func HandleUpdateMonitor(db *sqlx.DB, executor MonitorExecutor) http.HandlerFunc {
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
		var count int
		db.Get(&count, "SELECT COUNT(*) FROM monitors WHERE id = $1 AND user_id = $2", mon.ID, user.ID)
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

		// Marshal config to JSON
		configJSON, err := json.Marshal(mon.Config)
		if err != nil {
			http.Error(w, "Failed to marshal config", http.StatusInternalServerError)
			return
		}
		mon.ConfigRaw = string(configJSON)
		mon.UpdatedAt = time.Now()

		// Update database
		query := `
			UPDATE monitors
			SET name = $1, type = $2, url = $3, interval = $4, timeout = $5,
			    active = $6, config = $7, updated_at = $8
			WHERE id = $9 AND user_id = $10
		`

		_, err = db.Exec(query,
			mon.Name, mon.Type, mon.URL, mon.Interval, mon.Timeout,
			mon.Active, mon.ConfigRaw, mon.UpdatedAt, mon.ID, user.ID,
		)

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
func HandleDeleteMonitor(db *sqlx.DB, executor MonitorExecutor) http.HandlerFunc {
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
		query := `DELETE FROM monitors WHERE id = $1 AND user_id = $2`
		result, err := db.Exec(query, id, user.ID)
		if err != nil {
			http.Error(w, "Failed to delete monitor", http.StatusInternalServerError)
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			http.Error(w, "Monitor not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// HandleGetHeartbeats returns heartbeats for a monitor
func HandleGetHeartbeats(db *sqlx.DB) http.HandlerFunc {
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
		query := `
			SELECT id, monitor_id, status, ping, important, message, time
			FROM heartbeats
			WHERE monitor_id = $1
			ORDER BY time DESC
			LIMIT $2
		`

		err := db.Select(&heartbeats, query, monitorID, limit)
		if err != nil {
			http.Error(w, "Failed to fetch heartbeats", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(heartbeats)
	}
}
