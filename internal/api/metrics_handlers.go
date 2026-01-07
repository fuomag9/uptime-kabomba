package api

import (
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/fuomag9/uptime-kabomba/internal/models"
	"github.com/fuomag9/uptime-kabomba/internal/uptime"
)

// HandlePrometheusMetrics exports metrics in Prometheus format
func HandlePrometheusMetrics(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set content type for Prometheus
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")

		// Get all monitors
		var monitors []struct {
			ID       int    `gorm:"column:id"`
			Name     string `gorm:"column:name"`
			Type     string `gorm:"column:type"`
			URL      string `gorm:"column:url"`
			Active   bool   `gorm:"column:active"`
			UserID   int    `gorm:"column:user_id"`
		}

		if err := db.Model(&models.Monitor{}).
			Select("id, name, type, url, active, user_id").
			Find(&monitors).Error; err != nil {
			http.Error(w, "Failed to fetch monitors", http.StatusInternalServerError)
			return
		}

		calculator := uptime.NewCalculator(db)

		// Write metrics header
		fmt.Fprintln(w, "# HELP uptime_monitor_up Monitor status (1 = up, 0 = down)")
		fmt.Fprintln(w, "# TYPE uptime_monitor_up gauge")

		fmt.Fprintln(w, "# HELP uptime_monitor_ping_ms Monitor response time in milliseconds")
		fmt.Fprintln(w, "# TYPE uptime_monitor_ping_ms gauge")

		fmt.Fprintln(w, "# HELP uptime_monitor_uptime_percentage Monitor uptime percentage (24h)")
		fmt.Fprintln(w, "# TYPE uptime_monitor_uptime_percentage gauge")

		fmt.Fprintln(w, "# HELP uptime_monitor_total_checks Total number of checks (24h)")
		fmt.Fprintln(w, "# TYPE uptime_monitor_total_checks counter")

		fmt.Fprintln(w, "# HELP uptime_monitor_active Monitor active status")
		fmt.Fprintln(w, "# TYPE uptime_monitor_active gauge")

		// Write metrics for each monitor
		for _, monitor := range monitors {
			labels := fmt.Sprintf(`monitor_id="%d",monitor_name="%s",monitor_type="%s",user_id="%d"`,
				monitor.ID, monitor.Name, monitor.Type, monitor.UserID)

			// Get latest heartbeat
			var heartbeat models.Heartbeat
			err := db.Where("monitor_id = ?", monitor.ID).
				Order("time DESC").
				Limit(1).
				First(&heartbeat).Error
			if err == nil {
				// Monitor status
				status := 0
				if heartbeat.Status == 1 {
					status = 1
				}
				fmt.Fprintf(w, "uptime_monitor_up{%s} %d\n", labels, status)

				// Monitor ping
				fmt.Fprintf(w, "uptime_monitor_ping_ms{%s} %d\n", labels, heartbeat.Ping)
			} else {
				// No heartbeat data
				fmt.Fprintf(w, "uptime_monitor_up{%s} 0\n", labels)
				fmt.Fprintf(w, "uptime_monitor_ping_ms{%s} 0\n", labels)
			}

			// Get 24h uptime stats
			stats, err := calculator.Calculate24HourUptime(monitor.ID)
			if err == nil {
				fmt.Fprintf(w, "uptime_monitor_uptime_percentage{%s} %.2f\n", labels, stats.UptimePercentage)
				fmt.Fprintf(w, "uptime_monitor_total_checks{%s} %d\n", labels, stats.TotalChecks)
			}

			// Monitor active status
			activeValue := 0
			if monitor.Active {
				activeValue = 1
			}
			fmt.Fprintf(w, "uptime_monitor_active{%s} %d\n", labels, activeValue)
		}

		// System metrics
		fmt.Fprintln(w, "# HELP uptime_system_total_monitors Total number of monitors")
		fmt.Fprintln(w, "# TYPE uptime_system_total_monitors gauge")
		fmt.Fprintf(w, "uptime_system_total_monitors %d\n", len(monitors))

		// Count active monitors
		activeCount := 0
		for _, m := range monitors {
			if m.Active {
				activeCount++
			}
		}
		fmt.Fprintln(w, "# HELP uptime_system_active_monitors Number of active monitors")
		fmt.Fprintln(w, "# TYPE uptime_system_active_monitors gauge")
		fmt.Fprintf(w, "uptime_system_active_monitors %d\n", activeCount)

		// Heartbeat count (total in database)
		var totalHeartbeats int64
		db.Model(&models.Heartbeat{}).Count(&totalHeartbeats)
		fmt.Fprintln(w, "# HELP uptime_system_total_heartbeats Total heartbeats recorded")
		fmt.Fprintln(w, "# TYPE uptime_system_total_heartbeats counter")
		fmt.Fprintf(w, "uptime_system_total_heartbeats %d\n", totalHeartbeats)

		// Database size (SQLite specific)
		var pageCount, pageSize int
		db.Raw("PRAGMA page_count").Scan(&pageCount)
		db.Raw("PRAGMA page_size").Scan(&pageSize)
		dbSize := pageCount * pageSize
		fmt.Fprintln(w, "# HELP uptime_system_database_size_bytes Database size in bytes")
		fmt.Fprintln(w, "# TYPE uptime_system_database_size_bytes gauge")
		fmt.Fprintf(w, "uptime_system_database_size_bytes %d\n", dbSize)

		// Timestamp
		fmt.Fprintln(w, "# HELP uptime_system_scrape_timestamp_seconds Unix timestamp of this scrape")
		fmt.Fprintln(w, "# TYPE uptime_system_scrape_timestamp_seconds gauge")
		fmt.Fprintf(w, "uptime_system_scrape_timestamp_seconds %d\n", time.Now().Unix())
	}
}
