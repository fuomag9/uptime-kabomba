package monitor

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/fuomag9/uptime-kuma-go/internal/websocket"
	"github.com/fuomag9/uptime-kuma-go/internal/notification"
)

// Executor manages monitor execution
type Executor struct {
	db         *sqlx.DB
	hub        *websocket.Hub
	dispatcher *notification.Dispatcher
	monitors   map[int]*monitorJob
	mu         sync.RWMutex
}

// monitorJob represents a running monitor job
type monitorJob struct {
	monitor    *Monitor
	ticker     *time.Ticker
	stop       chan bool
	executor   *Executor
	lastStatus int // Track last status for change detection
}

// NewExecutor creates a new monitor executor
func NewExecutor(db *sqlx.DB, hub *websocket.Hub, dispatcher *notification.Dispatcher) *Executor {
	return &Executor{
		db:         db,
		hub:        hub,
		dispatcher: dispatcher,
		monitors:   make(map[int]*monitorJob),
	}
}

// Start loads all active monitors and starts monitoring
func (e *Executor) Start() error {
	// Load all active monitors
	var monitors []*Monitor
	query := `SELECT id, user_id, name, type, url, interval, timeout, active, config, created_at, updated_at FROM monitors WHERE active = true`
	err := e.db.Select(&monitors, query)
	if err != nil {
		return err
	}

	log.Printf("Starting %d active monitors", len(monitors))

	for _, monitor := range monitors {
		// Parse config JSON
		if monitor.ConfigJSON != "" {
			if err := json.Unmarshal([]byte(monitor.ConfigJSON), &monitor.Config); err != nil {
				log.Printf("Failed to parse config for monitor %d: %v", monitor.ID, err)
				continue
			}
		}

		e.StartMonitor(monitor)
	}

	return nil
}

// StartMonitor starts monitoring for a specific monitor
func (e *Executor) StartMonitor(monitor *Monitor) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Stop existing job if running
	if job, exists := e.monitors[monitor.ID]; exists {
		job.stop <- true
		delete(e.monitors, monitor.ID)
	}

	// Get last heartbeat status from database
	lastStatus := StatusPending
	var lastHeartbeat struct {
		Status int `db:"status"`
	}
	query := `SELECT status FROM heartbeats WHERE monitor_id = ? ORDER BY time DESC LIMIT 1`
	if err := e.db.Get(&lastHeartbeat, query, monitor.ID); err == nil {
		lastStatus = lastHeartbeat.Status
	}

	// Create new job
	job := &monitorJob{
		monitor:    monitor,
		ticker:     time.NewTicker(time.Duration(monitor.Interval) * time.Second),
		stop:       make(chan bool),
		executor:   e,
		lastStatus: lastStatus,
	}

	e.monitors[monitor.ID] = job

	// Run first check immediately
	go job.runCheck()

	// Start ticker
	go func() {
		for {
			select {
			case <-job.ticker.C:
				go job.runCheck()
			case <-job.stop:
				job.ticker.Stop()
				return
			}
		}
	}()

	log.Printf("Started monitor: %s (ID: %d, Interval: %ds)", monitor.Name, monitor.ID, monitor.Interval)
}

// StopMonitor stops monitoring for a specific monitor
func (e *Executor) StopMonitor(monitorID int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if job, exists := e.monitors[monitorID]; exists {
		job.stop <- true
		delete(e.monitors, monitorID)
		log.Printf("Stopped monitor ID: %d", monitorID)
	}
}

// Stop stops all monitors
func (e *Executor) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	for id, job := range e.monitors {
		job.stop <- true
		delete(e.monitors, id)
	}

	log.Println("All monitors stopped")
}

// runCheck performs a single monitor check
func (job *monitorJob) runCheck() {
	monitor := job.monitor

	// Get monitor type
	monitorType, ok := GetMonitorType(monitor.Type)
	if !ok {
		log.Printf("Unknown monitor type: %s for monitor ID %d", monitor.Type, monitor.ID)
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(monitor.Timeout+5)*time.Second)
	defer cancel()

	// Perform check
	heartbeat, err := monitorType.Check(ctx, monitor)
	if err != nil {
		log.Printf("Monitor check failed for %s (ID: %d): %v", monitor.Name, monitor.ID, err)
		return
	}

	// Save heartbeat to database
	if err := job.saveHeartbeat(heartbeat); err != nil {
		log.Printf("Failed to save heartbeat for monitor %d: %v", monitor.ID, err)
		return
	}

	// Broadcast heartbeat via WebSocket
	if job.executor.hub != nil {
		job.executor.hub.Broadcast("heartbeat", heartbeat)
	}

	// Detect status changes and send notifications
	if job.executor.dispatcher != nil {
		ctx := context.Background()

		// Monitor went down (from up to down)
		if job.lastStatus == StatusUp && heartbeat.Status == StatusDown {
			monitorURL := "" // TODO: Generate monitor URL when status pages are implemented
			err := job.executor.dispatcher.NotifyMonitorDown(ctx, monitor.ID, monitor.Name, monitorURL, heartbeat.Ping, heartbeat.Message)
			if err != nil {
				log.Printf("Failed to send down notification for monitor %d: %v", monitor.ID, err)
			} else {
				log.Printf("Sent DOWN notification for monitor %s (ID: %d)", monitor.Name, monitor.ID)
			}
		}

		// Monitor came back up (from down to up)
		if job.lastStatus == StatusDown && heartbeat.Status == StatusUp {
			monitorURL := ""
			err := job.executor.dispatcher.NotifyMonitorUp(ctx, monitor.ID, monitor.Name, monitorURL, heartbeat.Ping, heartbeat.Message)
			if err != nil {
				log.Printf("Failed to send up notification for monitor %d: %v", monitor.ID, err)
			} else {
				log.Printf("Sent UP notification for monitor %s (ID: %d)", monitor.Name, monitor.ID)
			}
		}
	}

	// Update last status
	job.lastStatus = heartbeat.Status

	// Log status
	statusText := "DOWN"
	if heartbeat.Status == StatusUp {
		statusText = "UP"
	}
	log.Printf("Monitor %s (ID: %d): %s - %dms - %s",
		monitor.Name, monitor.ID, statusText, heartbeat.Ping, heartbeat.Message)
}

// saveHeartbeat saves a heartbeat to the database
func (job *monitorJob) saveHeartbeat(heartbeat *Heartbeat) error {
	query := `
		INSERT INTO heartbeats (monitor_id, status, ping, important, message, time)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := job.executor.db.Exec(query,
		heartbeat.MonitorID,
		heartbeat.Status,
		heartbeat.Ping,
		heartbeat.Important,
		heartbeat.Message,
		heartbeat.Time,
	)

	return err
}
