package jobs

import (
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

// StatsAggregator aggregates heartbeat data into hourly and daily statistics
type StatsAggregator struct {
	db *sqlx.DB
}

// NewStatsAggregator creates a new statistics aggregator
func NewStatsAggregator(db *sqlx.DB) *StatsAggregator {
	return &StatsAggregator{db: db}
}

// AggregateHourly aggregates heartbeat data into hourly statistics
func (a *StatsAggregator) AggregateHourly() error {
	log.Println("Starting hourly statistics aggregation...")

	// Get all monitors
	var monitorIDs []int
	err := a.db.Select(&monitorIDs, "SELECT id FROM monitors")
	if err != nil {
		return err
	}

	now := time.Now()
	// Aggregate the previous hour
	hourStart := now.Add(-1 * time.Hour).Truncate(time.Hour)
	hourEnd := hourStart.Add(time.Hour)

	for _, monitorID := range monitorIDs {
		err := a.aggregateMonitorHourly(monitorID, hourStart, hourEnd)
		if err != nil {
			log.Printf("Failed to aggregate hourly stats for monitor %d: %v", monitorID, err)
			// Continue with other monitors
		}
	}

	log.Println("Hourly statistics aggregation completed")
	return nil
}

// aggregateMonitorHourly aggregates statistics for a single monitor and hour
func (a *StatsAggregator) aggregateMonitorHourly(monitorID int, hourStart, hourEnd time.Time) error {
	query := `
		SELECT
			MIN(ping) as ping_min,
			MAX(ping) as ping_max,
			AVG(ping) as ping_avg,
			SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) as up_count,
			SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END) as down_count,
			COUNT(*) as total_count
		FROM heartbeats
		WHERE monitor_id = ? AND time >= ? AND time < ?
	`

	var stats struct {
		PingMin    *int    `db:"ping_min"`
		PingMax    *int    `db:"ping_max"`
		PingAvg    *float64 `db:"ping_avg"`
		UpCount    int     `db:"up_count"`
		DownCount  int     `db:"down_count"`
		TotalCount int     `db:"total_count"`
	}

	err := a.db.Get(&stats, query, monitorID, hourStart, hourEnd)
	if err != nil {
		return err
	}

	// Skip if no data
	if stats.TotalCount == 0 {
		return nil
	}

	// Calculate uptime percentage
	uptimePercentage := 0.0
	if stats.TotalCount > 0 {
		uptimePercentage = (float64(stats.UpCount) / float64(stats.TotalCount)) * 100
	}

	// Insert or update hourly stat
	upsertQuery := `
		INSERT INTO stat_hourly (monitor_id, hour, ping_min, ping_max, ping_avg, up_count, down_count, total_count, uptime_percentage, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(monitor_id, hour) DO UPDATE SET
			ping_min = EXCLUDED.ping_min,
			ping_max = EXCLUDED.ping_max,
			ping_avg = EXCLUDED.ping_avg,
			up_count = EXCLUDED.up_count,
			down_count = EXCLUDED.down_count,
			total_count = EXCLUDED.total_count,
			uptime_percentage = EXCLUDED.uptime_percentage
	`

	pingMin := 0
	if stats.PingMin != nil {
		pingMin = *stats.PingMin
	}
	pingMax := 0
	if stats.PingMax != nil {
		pingMax = *stats.PingMax
	}
	pingAvg := 0.0
	if stats.PingAvg != nil {
		pingAvg = *stats.PingAvg
	}

	_, err = a.db.Exec(upsertQuery,
		monitorID, hourStart, pingMin, pingMax, pingAvg,
		stats.UpCount, stats.DownCount, stats.TotalCount, uptimePercentage, time.Now(),
	)

	return err
}

// AggregateDaily aggregates heartbeat data into daily statistics
func (a *StatsAggregator) AggregateDaily() error {
	log.Println("Starting daily statistics aggregation...")

	// Get all monitors
	var monitorIDs []int
	err := a.db.Select(&monitorIDs, "SELECT id FROM monitors")
	if err != nil {
		return err
	}

	// Aggregate the previous day
	yesterday := time.Now().AddDate(0, 0, -1)
	dayStart := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
	dayEnd := dayStart.AddDate(0, 0, 1)

	for _, monitorID := range monitorIDs {
		err := a.aggregateMonitorDaily(monitorID, dayStart, dayEnd)
		if err != nil {
			log.Printf("Failed to aggregate daily stats for monitor %d: %v", monitorID, err)
			// Continue with other monitors
		}
	}

	log.Println("Daily statistics aggregation completed")
	return nil
}

// aggregateMonitorDaily aggregates statistics for a single monitor and day
func (a *StatsAggregator) aggregateMonitorDaily(monitorID int, dayStart, dayEnd time.Time) error {
	query := `
		SELECT
			MIN(ping) as ping_min,
			MAX(ping) as ping_max,
			AVG(ping) as ping_avg,
			SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) as up_count,
			SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END) as down_count,
			COUNT(*) as total_count
		FROM heartbeats
		WHERE monitor_id = ? AND time >= ? AND time < ?
	`

	var stats struct {
		PingMin    *int    `db:"ping_min"`
		PingMax    *int    `db:"ping_max"`
		PingAvg    *float64 `db:"ping_avg"`
		UpCount    int     `db:"up_count"`
		DownCount  int     `db:"down_count"`
		TotalCount int     `db:"total_count"`
	}

	err := a.db.Get(&stats, query, monitorID, dayStart, dayEnd)
	if err != nil {
		return err
	}

	// Skip if no data
	if stats.TotalCount == 0 {
		return nil
	}

	// Calculate uptime percentage
	uptimePercentage := 0.0
	if stats.TotalCount > 0 {
		uptimePercentage = (float64(stats.UpCount) / float64(stats.TotalCount)) * 100
	}

	// Insert or update daily stat
	upsertQuery := `
		INSERT INTO stat_daily (monitor_id, date, ping_min, ping_max, ping_avg, up_count, down_count, total_count, uptime_percentage, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(monitor_id, date) DO UPDATE SET
			ping_min = EXCLUDED.ping_min,
			ping_max = EXCLUDED.ping_max,
			ping_avg = EXCLUDED.ping_avg,
			up_count = EXCLUDED.up_count,
			down_count = EXCLUDED.down_count,
			total_count = EXCLUDED.total_count,
			uptime_percentage = EXCLUDED.uptime_percentage
	`

	pingMin := 0
	if stats.PingMin != nil {
		pingMin = *stats.PingMin
	}
	pingMax := 0
	if stats.PingMax != nil {
		pingMax = *stats.PingMax
	}
	pingAvg := 0.0
	if stats.PingAvg != nil {
		pingAvg = *stats.PingAvg
	}

	_, err = a.db.Exec(upsertQuery,
		monitorID, dayStart.Format("2006-01-02"), pingMin, pingMax, pingAvg,
		stats.UpCount, stats.DownCount, stats.TotalCount, uptimePercentage, time.Now(),
	)

	return err
}
