package jobs

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/robfig/cron/v3"
)

// Scheduler manages background jobs
type Scheduler struct {
	cron *cron.Cron
	db   *sqlx.DB
}

// NewScheduler creates a new job scheduler
func NewScheduler(db *sqlx.DB) *Scheduler {
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

// cleanupOldHeartbeats removes old heartbeat data
func (s *Scheduler) cleanupOldHeartbeats() {
	// Keep heartbeats for last 90 days
	query := `
		DELETE FROM heartbeats
		WHERE important = false
		AND time < datetime('now', '-90 days')
	`

	result, err := s.db.Exec(query)
	if err != nil {
		log.Printf("Failed to cleanup old heartbeats: %v", err)
		return
	}

	rows, _ := result.RowsAffected()
	log.Printf("Cleaned up %d old heartbeats", rows)
}

// cleanupOldStats removes aggregated stats older than 1 year
func (s *Scheduler) cleanupOldStats() {
	// Delete hourly stats older than 1 year
	hourlyQuery := `DELETE FROM stat_hourly WHERE hour < datetime('now', '-365 days')`
	result, err := s.db.Exec(hourlyQuery)
	if err != nil {
		log.Printf("Failed to cleanup old hourly stats: %v", err)
	} else {
		rows, _ := result.RowsAffected()
		log.Printf("Cleaned up %d old hourly stats", rows)
	}

	// Delete daily stats older than 2 years
	dailyQuery := `DELETE FROM stat_daily WHERE date < date('now', '-730 days')`
	result, err = s.db.Exec(dailyQuery)
	if err != nil {
		log.Printf("Failed to cleanup old daily stats: %v", err)
	} else {
		rows, _ := result.RowsAffected()
		log.Printf("Cleaned up %d old daily stats", rows)
	}
}

// vacuumDatabase runs VACUUM on SQLite database
func (s *Scheduler) vacuumDatabase() {
	_, err := s.db.Exec("VACUUM")
	if err != nil {
		log.Printf("Failed to vacuum database: %v", err)
		return
	}

	log.Println("Database vacuum completed")
}
