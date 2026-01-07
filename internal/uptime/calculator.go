package uptime

import (
	"time"

	"gorm.io/gorm"

	"github.com/fuomag9/uptime-kabomba/internal/models"
)

// Calculator calculates uptime statistics for monitors
type Calculator struct {
	db *gorm.DB
}

// NewCalculator creates a new uptime calculator
func NewCalculator(db *gorm.DB) *Calculator {
	return &Calculator{db: db}
}

// UptimeStats represents uptime statistics for a monitor
type UptimeStats struct {
	MonitorID         int     `json:"monitor_id"`
	UptimePercentage  float64 `json:"uptime_percentage"`
	TotalChecks       int     `json:"total_checks"`
	UpChecks          int     `json:"up_checks"`
	DownChecks        int     `json:"down_checks"`
	AveragePing       float64 `json:"average_ping"`
	StartTime         string  `json:"start_time"`
	EndTime           string  `json:"end_time"`
}

// Calculate24HourUptime calculates uptime for the last 24 hours
func (c *Calculator) Calculate24HourUptime(monitorID int) (*UptimeStats, error) {
	return c.CalculateUptimeForPeriod(monitorID, 24*time.Hour)
}

// Calculate7DayUptime calculates uptime for the last 7 days
func (c *Calculator) Calculate7DayUptime(monitorID int) (*UptimeStats, error) {
	return c.CalculateUptimeForPeriod(monitorID, 7*24*time.Hour)
}

// Calculate30DayUptime calculates uptime for the last 30 days
func (c *Calculator) Calculate30DayUptime(monitorID int) (*UptimeStats, error) {
	return c.CalculateUptimeForPeriod(monitorID, 30*24*time.Hour)
}

// Calculate90DayUptime calculates uptime for the last 90 days
func (c *Calculator) Calculate90DayUptime(monitorID int) (*UptimeStats, error) {
	return c.CalculateUptimeForPeriod(monitorID, 90*24*time.Hour)
}

// CalculateUptimeForPeriod calculates uptime for a specific time period
func (c *Calculator) CalculateUptimeForPeriod(monitorID int, duration time.Duration) (*UptimeStats, error) {
	endTime := time.Now()
	startTime := endTime.Add(-duration)

	query := `
		SELECT
			COUNT(*) as total_checks,
			SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) as up_checks,
			SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END) as down_checks,
			AVG(CASE WHEN status = 1 THEN ping ELSE NULL END) as average_ping
		FROM heartbeats
		WHERE monitor_id = ? AND time >= ? AND time <= ?
	`

	var stats struct {
		TotalChecks int     `gorm:"column:total_checks"`
		UpChecks    int     `gorm:"column:up_checks"`
		DownChecks  int     `gorm:"column:down_checks"`
		AveragePing float64 `gorm:"column:average_ping"`
	}

	err := c.db.Raw(query, monitorID, startTime, endTime).Scan(&stats).Error
	if err != nil {
		return nil, err
	}

	// Calculate uptime percentage
	uptimePercentage := 0.0
	if stats.TotalChecks > 0 {
		uptimePercentage = (float64(stats.UpChecks) / float64(stats.TotalChecks)) * 100
	}

	return &UptimeStats{
		MonitorID:        monitorID,
		UptimePercentage: uptimePercentage,
		TotalChecks:      stats.TotalChecks,
		UpChecks:         stats.UpChecks,
		DownChecks:       stats.DownChecks,
		AveragePing:      stats.AveragePing,
		StartTime:        startTime.Format(time.RFC3339),
		EndTime:          endTime.Format(time.RFC3339),
	}, nil
}

// CalculateUptimeForTimeRange calculates uptime between two specific times
func (c *Calculator) CalculateUptimeForTimeRange(monitorID int, startTime, endTime time.Time) (*UptimeStats, error) {
	query := `
		SELECT
			COUNT(*) as total_checks,
			SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) as up_checks,
			SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END) as down_checks,
			AVG(CASE WHEN status = 1 THEN ping ELSE NULL END) as average_ping
		FROM heartbeats
		WHERE monitor_id = ? AND time >= ? AND time <= ?
	`

	var stats struct {
		TotalChecks int     `gorm:"column:total_checks"`
		UpChecks    int     `gorm:"column:up_checks"`
		DownChecks  int     `gorm:"column:down_checks"`
		AveragePing float64 `gorm:"column:average_ping"`
	}

	err := c.db.Raw(query, monitorID, startTime, endTime).Scan(&stats).Error
	if err != nil {
		return nil, err
	}

	uptimePercentage := 0.0
	if stats.TotalChecks > 0 {
		uptimePercentage = (float64(stats.UpChecks) / float64(stats.TotalChecks)) * 100
	}

	return &UptimeStats{
		MonitorID:        monitorID,
		UptimePercentage: uptimePercentage,
		TotalChecks:      stats.TotalChecks,
		UpChecks:         stats.UpChecks,
		DownChecks:       stats.DownChecks,
		AveragePing:      stats.AveragePing,
		StartTime:        startTime.Format(time.RFC3339),
		EndTime:          endTime.Format(time.RFC3339),
	}, nil
}

// GetUptimeForAllMonitors calculates uptime for all active monitors
func (c *Calculator) GetUptimeForAllMonitors(duration time.Duration) (map[int]*UptimeStats, error) {
	// Get all active monitor IDs
	var monitors []models.Monitor
	err := c.db.Where("active = ?", true).
		Select("id").
		Find(&monitors).Error
	if err != nil {
		return nil, err
	}

	var monitorIDs []int
	for _, m := range monitors {
		monitorIDs = append(monitorIDs, m.ID)
	}

	results := make(map[int]*UptimeStats)
	for _, monitorID := range monitorIDs {
		stats, err := c.CalculateUptimeForPeriod(monitorID, duration)
		if err != nil {
			// Log error but continue
			continue
		}
		results[monitorID] = stats
	}

	return results, nil
}

// DailyUptimePoint represents uptime for a single day
type DailyUptimePoint struct {
	Date             string  `json:"date"`
	UptimePercentage float64 `json:"uptime_percentage"`
	TotalChecks      int     `json:"total_checks"`
	UpChecks         int     `json:"up_checks"`
}

// GetDailyUptimeHistory returns uptime stats for each day in the given period
func (c *Calculator) GetDailyUptimeHistory(monitorID int, days int) ([]DailyUptimePoint, error) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	query := `
		SELECT
			DATE(time) as date,
			COUNT(*) as total_checks,
			SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) as up_checks
		FROM heartbeats
		WHERE monitor_id = ? AND time >= ? AND time <= ?
		GROUP BY DATE(time)
		ORDER BY date ASC
	`

	rows, err := c.db.Raw(query, monitorID, startTime, endTime).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DailyUptimePoint
	for rows.Next() {
		var point DailyUptimePoint
		var totalChecks, upChecks int
		var date string

		if err := rows.Scan(&date, &totalChecks, &upChecks); err != nil {
			continue
		}

		uptimePercentage := 0.0
		if totalChecks > 0 {
			uptimePercentage = (float64(upChecks) / float64(totalChecks)) * 100
		}

		point.Date = date
		point.UptimePercentage = uptimePercentage
		point.TotalChecks = totalChecks
		point.UpChecks = upChecks

		results = append(results, point)
	}

	return results, nil
}

// HourlyUptimePoint represents uptime for a single hour
type HourlyUptimePoint struct {
	Hour             string  `json:"hour"`
	UptimePercentage float64 `json:"uptime_percentage"`
	TotalChecks      int     `json:"total_checks"`
	UpChecks         int     `json:"up_checks"`
}

// GetHourlyUptimeHistory returns uptime stats for each hour in the last 24 hours
func (c *Calculator) GetHourlyUptimeHistory(monitorID int) ([]HourlyUptimePoint, error) {
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	query := `
		SELECT
			strftime('%Y-%m-%d %H:00:00', time) as hour,
			COUNT(*) as total_checks,
			SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) as up_checks
		FROM heartbeats
		WHERE monitor_id = ? AND time >= ? AND time <= ?
		GROUP BY hour
		ORDER BY hour ASC
	`

	rows, err := c.db.Raw(query, monitorID, startTime, endTime).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []HourlyUptimePoint
	for rows.Next() {
		var point HourlyUptimePoint
		var totalChecks, upChecks int
		var hour string

		if err := rows.Scan(&hour, &totalChecks, &upChecks); err != nil {
			continue
		}

		uptimePercentage := 0.0
		if totalChecks > 0 {
			uptimePercentage = (float64(upChecks) / float64(totalChecks)) * 100
		}

		point.Hour = hour
		point.UptimePercentage = uptimePercentage
		point.TotalChecks = totalChecks
		point.UpChecks = upChecks

		results = append(results, point)
	}

	return results, nil
}
