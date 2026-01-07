package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"github.com/fuomag9/uptime-kabomba/internal/models"
	"github.com/fuomag9/uptime-kabomba/internal/uptime"
)

// HandleStatusBadge generates a status badge SVG
func HandleStatusBadge(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		monitorID := chi.URLParam(r, "id")
		id, err := strconv.Atoi(monitorID)
		if err != nil {
			http.Error(w, "Invalid monitor ID", http.StatusBadRequest)
			return
		}

		// Check if monitor exists
		var count int64
		db.Model(&models.Monitor{}).Where("id = ?", id).Count(&count)
		if count == 0 {
			http.Error(w, "Monitor not found", http.StatusNotFound)
			return
		}

		// Get latest heartbeat
		var heartbeat models.Heartbeat
		err = db.Where("monitor_id = ?", id).
			Order("time DESC").
			Limit(1).
			First(&heartbeat).Error

		var statusText, color string
		if err != nil {
			// No heartbeats yet
			statusText = "unknown"
			color = "gray"
		} else {
			switch heartbeat.Status {
			case 1:
				statusText = "up"
				color = "brightgreen"
			case 0:
				statusText = "down"
				color = "red"
			case 3:
				statusText = "maintenance"
				color = "yellow"
			default:
				statusText = "unknown"
				color = "gray"
			}
		}

		svg := generateBadgeSVG("status", statusText, color)

		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Write([]byte(svg))
	}
}

// HandleUptimeBadge generates an uptime percentage badge
func HandleUptimeBadge(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		monitorID := chi.URLParam(r, "id")
		id, err := strconv.Atoi(monitorID)
		if err != nil {
			http.Error(w, "Invalid monitor ID", http.StatusBadRequest)
			return
		}

		// Check if monitor exists
		var count int64
		db.Model(&models.Monitor{}).Where("id = ?", id).Count(&count)
		if count == 0 {
			http.Error(w, "Monitor not found", http.StatusNotFound)
			return
		}

		calculator := uptime.NewCalculator(db)

		// Get period from query (default 30d)
		period := r.URL.Query().Get("period")
		var stats *uptime.UptimeStats

		switch period {
		case "24h":
			stats, err = calculator.Calculate24HourUptime(id)
		case "7d":
			stats, err = calculator.Calculate7DayUptime(id)
		case "90d":
			stats, err = calculator.Calculate90DayUptime(id)
		default:
			stats, err = calculator.Calculate30DayUptime(id)
		}

		var uptimeText, color string
		if err != nil || stats.TotalChecks == 0 {
			uptimeText = "N/A"
			color = "gray"
		} else {
			uptimeText = fmt.Sprintf("%.2f%%", stats.UptimePercentage)

			// Color based on uptime
			if stats.UptimePercentage >= 99.9 {
				color = "brightgreen"
			} else if stats.UptimePercentage >= 99.0 {
				color = "green"
			} else if stats.UptimePercentage >= 95.0 {
				color = "yellowgreen"
			} else if stats.UptimePercentage >= 90.0 {
				color = "yellow"
			} else {
				color = "red"
			}
		}

		label := "uptime"
		if period != "" {
			label = fmt.Sprintf("uptime (%s)", period)
		}

		svg := generateBadgeSVG(label, uptimeText, color)

		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Write([]byte(svg))
	}
}

// HandlePingBadge generates a ping time badge
func HandlePingBadge(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		monitorID := chi.URLParam(r, "id")
		id, err := strconv.Atoi(monitorID)
		if err != nil {
			http.Error(w, "Invalid monitor ID", http.StatusBadRequest)
			return
		}

		// Check if monitor exists
		var count int64
		db.Model(&models.Monitor{}).Where("id = ?", id).Count(&count)
		if count == 0 {
			http.Error(w, "Monitor not found", http.StatusNotFound)
			return
		}

		// Get average ping from last 10 heartbeats
		var result struct {
			AvgPing float64 `gorm:"column:avg_ping"`
		}
		err = db.Raw(`SELECT AVG(ping) as avg_ping FROM (SELECT ping FROM heartbeats WHERE monitor_id = ? AND status = 1 ORDER BY time DESC LIMIT 10)`, id).
			Scan(&result).Error
		avgPing := result.AvgPing

		var pingText, color string
		if err != nil || avgPing == 0 {
			pingText = "N/A"
			color = "gray"
		} else {
			pingText = fmt.Sprintf("%.0fms", avgPing)

			// Color based on ping
			if avgPing < 100 {
				color = "brightgreen"
			} else if avgPing < 300 {
				color = "green"
			} else if avgPing < 500 {
				color = "yellow"
			} else if avgPing < 1000 {
				color = "orange"
			} else {
				color = "red"
			}
		}

		svg := generateBadgeSVG("response time", pingText, color)

		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Write([]byte(svg))
	}
}

// generateBadgeSVG generates a shields.io style badge
func generateBadgeSVG(label, message, color string) string {
	// Color mapping
	colorMap := map[string]string{
		"brightgreen": "#4c1",
		"green":       "#97ca00",
		"yellowgreen": "#a4a61d",
		"yellow":      "#dfb317",
		"orange":      "#fe7d37",
		"red":         "#e05d44",
		"blue":        "#007ec6",
		"gray":        "#555",
		"lightgray":   "#9f9f9f",
	}

	hexColor, ok := colorMap[color]
	if !ok {
		hexColor = colorMap["gray"]
	}

	labelWidth := len(label) * 6 + 10
	messageWidth := len(message) * 6 + 10
	totalWidth := labelWidth + messageWidth

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20">
  <linearGradient id="b" x2="0" y2="100%%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <mask id="a">
    <rect width="%d" height="20" rx="3" fill="#fff"/>
  </mask>
  <g mask="url(#a)">
    <path fill="#555" d="M0 0h%dv20H0z"/>
    <path fill="%s" d="M%d 0h%dv20H%dz"/>
    <path fill="url(#b)" d="M0 0h%dv20H0z"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
    <text x="%d" y="15" fill="#010101" fill-opacity=".3">%s</text>
    <text x="%d" y="14">%s</text>
    <text x="%d" y="15" fill="#010101" fill-opacity=".3">%s</text>
    <text x="%d" y="14">%s</text>
  </g>
</svg>`,
		totalWidth,
		totalWidth,
		labelWidth, hexColor, labelWidth, messageWidth, labelWidth,
		totalWidth,
		labelWidth/2, label,
		labelWidth/2, label,
		labelWidth+messageWidth/2, message,
		labelWidth+messageWidth/2, message,
	)

	return svg
}
