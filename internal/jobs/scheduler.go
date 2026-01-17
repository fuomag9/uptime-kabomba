package jobs

import (
	"fmt"
	"log"

	"gorm.io/gorm"
	"github.com/robfig/cron/v3"

	"github.com/fuomag9/uptime-kabomba/internal/models"
)

// Scheduler manages background jobs
type Scheduler struct {
	cron *cron.Cron
	db   *gorm.DB
}

// NewScheduler creates a new job scheduler
func NewScheduler(db *gorm.DB) *Scheduler {
	return &Scheduler{
		cron: cron.New(),
		db:   db,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	// Create stats aggregator
	aggregator := NewStatsAggregator(s.db)

	// Aggregate hourly statistics every hour at minute 5
	s.cron.AddFunc("5 * * * *", func() {
		aggregator.AggregateHourly()
	})

	// Aggregate daily statistics daily at 2:00 AM
	s.cron.AddFunc("0 2 * * *", func() {
		aggregator.AggregateDaily()
	})

	// Cleanup old heartbeats daily at 3:14 AM
	s.cron.AddFunc("14 3 * * *", func() {
		log.Println("Running cleanup job...")
		s.cleanupOldHeartbeats()
	})

	// Cleanup old aggregated stats (keep 1 year)
	s.cron.AddFunc("30 3 * * *", func() {
		log.Println("Running stats cleanup job...")
		s.cleanupOldStats()
	})

	// Vacuum database weekly at 2:30 AM on Sunday
	s.cron.AddFunc("30 2 * * 0", func() {
		log.Println("Running vacuum job...")
		s.vacuumDatabase()
	})

	s.cron.Start()
	log.Println("Job scheduler started")
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.cron.Stop()
	log.Println("Job scheduler stopped")
}

// cleanupOldHeartbeats removes old heartbeat data based on user settings
func (s *Scheduler) cleanupOldHeartbeats() {
	// Get all user settings
	var settings []models.UserSettings
	s.db.Find(&settings)

	// Create a map of user ID to retention days
	userRetention := make(map[int]int)
	for _, setting := range settings {
		userRetention[setting.UserID] = setting.HeartbeatRetentionDays
	}

	// Get all users with monitors
	var users []struct {
		UserID int
	}
	s.db.Model(&models.Monitor{}).Select("DISTINCT user_id").Scan(&users)

	totalCleaned := int64(0)

	for _, user := range users {
		// Get retention days for this user (default 90 if not configured)
		retentionDays := 90
		if days, ok := userRetention[user.UserID]; ok {
			retentionDays = days
		}

		// Get monitor IDs for this user
		var monitorIDs []int
		s.db.Model(&models.Monitor{}).
			Where("user_id = ?", user.UserID).
			Pluck("id", &monitorIDs)

		if len(monitorIDs) == 0 {
			continue
		}

		// Delete old heartbeats for these monitors
		query := fmt.Sprintf(`
			DELETE FROM heartbeats
			WHERE important = false
			AND monitor_id IN (?)
			AND time < datetime('now', '-%d days')
		`, retentionDays)

		result := s.db.Exec(query, monitorIDs)
		if result.Error != nil {
			log.Printf("Failed to cleanup heartbeats for user %d: %v", user.UserID, result.Error)
			continue
		}

		if result.RowsAffected > 0 {
			log.Printf("User %d: Cleaned up %d heartbeats (retention: %d days)", user.UserID, result.RowsAffected, retentionDays)
			totalCleaned += result.RowsAffected
		}
	}

	log.Printf("Total heartbeats cleaned up: %d", totalCleaned)
}

// cleanupOldStats removes aggregated stats based on user settings
func (s *Scheduler) cleanupOldStats() {
	// Get all user settings
	var settings []models.UserSettings
	s.db.Find(&settings)

	// Create maps for user retention settings
	userHourlyRetention := make(map[int]int)
	userDailyRetention := make(map[int]int)
	for _, setting := range settings {
		userHourlyRetention[setting.UserID] = setting.HourlyStatRetentionDays
		userDailyRetention[setting.UserID] = setting.DailyStatRetentionDays
	}

	// Get all users with monitors
	var users []struct {
		UserID int
	}
	s.db.Model(&models.Monitor{}).Select("DISTINCT user_id").Scan(&users)

	totalHourlyCleaned := int64(0)
	totalDailyCleaned := int64(0)

	for _, user := range users {
		// Get retention days for this user (defaults: 365 hourly, 730 daily)
		hourlyRetention := 365
		if days, ok := userHourlyRetention[user.UserID]; ok {
			hourlyRetention = days
		}
		dailyRetention := 730
		if days, ok := userDailyRetention[user.UserID]; ok {
			dailyRetention = days
		}

		// Get monitor IDs for this user
		var monitorIDs []int
		s.db.Model(&models.Monitor{}).
			Where("user_id = ?", user.UserID).
			Pluck("id", &monitorIDs)

		if len(monitorIDs) == 0 {
			continue
		}

		// Delete old hourly stats
		hourlyQuery := fmt.Sprintf(`
			DELETE FROM stat_hourly
			WHERE monitor_id IN (?)
			AND hour < datetime('now', '-%d days')
		`, hourlyRetention)

		result := s.db.Exec(hourlyQuery, monitorIDs)
		if result.Error != nil {
			log.Printf("Failed to cleanup hourly stats for user %d: %v", user.UserID, result.Error)
		} else if result.RowsAffected > 0 {
			log.Printf("User %d: Cleaned up %d hourly stats (retention: %d days)", user.UserID, result.RowsAffected, hourlyRetention)
			totalHourlyCleaned += result.RowsAffected
		}

		// Delete old daily stats
		dailyQuery := fmt.Sprintf(`
			DELETE FROM stat_daily
			WHERE monitor_id IN (?)
			AND date < date('now', '-%d days')
		`, dailyRetention)

		result = s.db.Exec(dailyQuery, monitorIDs)
		if result.Error != nil {
			log.Printf("Failed to cleanup daily stats for user %d: %v", user.UserID, result.Error)
		} else if result.RowsAffected > 0 {
			log.Printf("User %d: Cleaned up %d daily stats (retention: %d days)", user.UserID, result.RowsAffected, dailyRetention)
			totalDailyCleaned += result.RowsAffected
		}
	}

	log.Printf("Total stats cleaned up: %d hourly, %d daily", totalHourlyCleaned, totalDailyCleaned)
}

// vacuumDatabase runs VACUUM on SQLite database
func (s *Scheduler) vacuumDatabase() {
	result := s.db.Exec("VACUUM")
	if result.Error != nil {
		log.Printf("Failed to vacuum database: %v", result.Error)
		return
	}

	log.Println("Database vacuum completed")
}
