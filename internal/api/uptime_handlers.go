package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/fuomag9/uptime-kuma-go/internal/models"
	"github.com/fuomag9/uptime-kuma-go/internal/uptime"
)

// HandleGetMonitorUptime returns uptime statistics for a monitor
func HandleGetMonitorUptime(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		monitorID := chi.URLParam(r, "id")

		// Verify ownership
		var count int
		db.Get(&count, "SELECT COUNT(*) FROM monitors WHERE id = $1 AND user_id = $2", monitorID, user.ID)
		if count == 0 {
			http.Error(w, "Monitor not found", http.StatusNotFound)
			return
		}

		id, _ := strconv.Atoi(monitorID)
		calculator := uptime.NewCalculator(db)

		// Get period from query param (default to 24h)
		period := r.URL.Query().Get("period")

		var stats *uptime.UptimeStats
		var err error

		switch period {
		case "7d":
			stats, err = calculator.Calculate7DayUptime(id)
		case "30d":
			stats, err = calculator.Calculate30DayUptime(id)
		case "90d":
			stats, err = calculator.Calculate90DayUptime(id)
		default:
			stats, err = calculator.Calculate24HourUptime(id)
		}

		if err != nil {
			http.Error(w, "Failed to calculate uptime", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}

// HandleGetMonitorUptimeHistory returns daily uptime history for a monitor
func HandleGetMonitorUptimeHistory(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		monitorID := chi.URLParam(r, "id")

		// Verify ownership
		var count int
		db.Get(&count, "SELECT COUNT(*) FROM monitors WHERE id = $1 AND user_id = $2", monitorID, user.ID)
		if count == 0 {
			http.Error(w, "Monitor not found", http.StatusNotFound)
			return
		}

		id, _ := strconv.Atoi(monitorID)
		calculator := uptime.NewCalculator(db)

		// Get days from query param (default to 30)
		daysStr := r.URL.Query().Get("days")
		days := 30
		if daysStr != "" {
			if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 365 {
				days = d
			}
		}

		history, err := calculator.GetDailyUptimeHistory(id, days)
		if err != nil {
			http.Error(w, "Failed to get uptime history", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(history)
	}
}

// HandleGetMonitorHourlyUptime returns hourly uptime for the last 24 hours
func HandleGetMonitorHourlyUptime(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		monitorID := chi.URLParam(r, "id")

		// Verify ownership
		var count int
		db.Get(&count, "SELECT COUNT(*) FROM monitors WHERE id = $1 AND user_id = $2", monitorID, user.ID)
		if count == 0 {
			http.Error(w, "Monitor not found", http.StatusNotFound)
			return
		}

		id, _ := strconv.Atoi(monitorID)
		calculator := uptime.NewCalculator(db)

		history, err := calculator.GetHourlyUptimeHistory(id)
		if err != nil {
			http.Error(w, "Failed to get hourly uptime", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(history)
	}
}

// HandleGetAllMonitorsUptime returns uptime for all monitors
func HandleGetAllMonitorsUptime(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)

		calculator := uptime.NewCalculator(db)

		// Get period from query param (default to 24h)
		period := r.URL.Query().Get("period")
		var duration time.Duration

		switch period {
		case "7d":
			duration = 7 * 24 * time.Hour
		case "30d":
			duration = 30 * 24 * time.Hour
		case "90d":
			duration = 90 * 24 * time.Hour
		default:
			duration = 24 * time.Hour
		}

		allStats, err := calculator.GetUptimeForAllMonitors(duration)
		if err != nil {
			http.Error(w, "Failed to calculate uptime", http.StatusInternalServerError)
			return
		}

		// Filter by user's monitors
		var monitorIDs []int
		query := `SELECT id FROM monitors WHERE user_id = $1 AND active = 1`
		db.Select(&monitorIDs, query, user.ID)

		userStats := make(map[int]*uptime.UptimeStats)
		for _, id := range monitorIDs {
			if stats, ok := allStats[id]; ok {
				userStats[id] = stats
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(userStats)
	}
}
