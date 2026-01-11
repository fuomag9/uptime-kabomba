package monitor

import (
	"context"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"
	"github.com/fuomag9/uptime-kabomba/internal/websocket"
	"github.com/fuomag9/uptime-kabomba/internal/notification"
)

// Executor manages monitor execution
type Executor struct {
	db         *gorm.DB
	hub        *websocket.Hub
	dispatcher *notification.Dispatcher
	monitors   map[int]*monitorJob
	mu         sync.RWMutex
}

// monitorJob represents a running monitor job
type monitorJob struct {
	monitor            *Monitor
	ticker             *time.Ticker
	stop               chan bool
	executor           *Executor
	lastStatus         int // Track last status for change detection
	consecutiveFailures int // Track consecutive down statuses
}

// NewExecutor creates a new monitor executor
func NewExecutor(db *gorm.DB, hub *websocket.Hub, dispatcher *notification.Dispatcher) *Executor {
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
	err := e.db.Where("active = ?", true).Find(&monitors).Error
	if err != nil {
		return err
	}

	log.Printf("Starting %d active monitors", len(monitors))

	for _, monitor := range monitors {
		// Config is already parsed by AfterFind hook
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
		Status int `gorm:"column:status"`
	}
	query := `SELECT status FROM heartbeats WHERE monitor_id = ? ORDER BY time DESC LIMIT 1`
	if err := e.db.Raw(query, monitor.ID).Scan(&lastHeartbeat).Error; err == nil {
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
		monitorURL := "" // TODO: Generate monitor URL when status pages are implemented

		// Track consecutive failures for resend logic
		if heartbeat.Status == StatusDown {
			job.consecutiveFailures++

			// Determine if we should send notification based on resend_interval
			resendInterval := monitor.ResendInterval
			if resendInterval < 1 {
				resendInterval = 1 // Ensure minimum of 1
			}

			shouldNotify := false
			if job.consecutiveFailures == resendInterval {
				// First notification after reaching threshold
				shouldNotify = true
			} else if job.consecutiveFailures > resendInterval {
				// Resend notification: check if it's time to resend
				// After threshold is reached, resend every resend_interval failures
				if (job.consecutiveFailures-resendInterval)%resendInterval == 0 {
					shouldNotify = true
				}
			}

			if shouldNotify {
				err := job.executor.dispatcher.NotifyMonitorDown(ctx, monitor.ID, monitor.Name, monitorURL, heartbeat.Ping, heartbeat.Message)
				if err != nil {
					log.Printf("Failed to send down notification for monitor %d: %v", monitor.ID, err)
				} else {
					log.Printf("Sent DOWN notification for monitor %s (ID: %d) after %d consecutive failures",
						monitor.Name, monitor.ID, job.consecutiveFailures)
				}
			} else {
				log.Printf("Monitor %s (ID: %d) is down (%d consecutive failures), waiting for threshold %d",
					monitor.Name, monitor.ID, job.consecutiveFailures, resendInterval)
			}
		} else if heartbeat.Status == StatusUp {
			// Monitor came back up - reset consecutive failures and send notification if was down
			if job.consecutiveFailures > 0 {
				err := job.executor.dispatcher.NotifyMonitorUp(ctx, monitor.ID, monitor.Name, monitorURL, heartbeat.Ping, heartbeat.Message)
				if err != nil {
					log.Printf("Failed to send up notification for monitor %d: %v", monitor.ID, err)
				} else {
					log.Printf("Sent UP notification for monitor %s (ID: %d) after %d failures",
						monitor.Name, monitor.ID, job.consecutiveFailures)
				}
				job.consecutiveFailures = 0 // Reset counter
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

	err := job.executor.db.Exec(query,
		heartbeat.MonitorID,
		heartbeat.Status,
		heartbeat.Ping,
		heartbeat.Important,
		heartbeat.Message,
		heartbeat.Time,
	).Error

	return err
}
