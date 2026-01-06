package monitor

import (
	"context"
	"fmt"
	"net"
	"time"
)

// TCPMonitor checks if a TCP port is open
type TCPMonitor struct{}

func init() {
	RegisterMonitorType(&TCPMonitor{})
}

func (t *TCPMonitor) Name() string {
	return "tcp"
}

func (t *TCPMonitor) Check(ctx context.Context, monitor *Monitor) (*Heartbeat, error) {
	heartbeat := &Heartbeat{
		MonitorID: monitor.ID,
		Time:      time.Now(),
	}

	// Parse host:port from URL
	host := monitor.URL
	if host == "" {
		heartbeat.Status = StatusDown
		heartbeat.Message = "No host specified"
		return heartbeat, nil
	}

	// Get port from config or default to 80
	port := 80
	if p, ok := monitor.Config["port"].(float64); ok {
		port = int(p)
	}

	// Construct address
	address := fmt.Sprintf("%s:%d", host, port)

	// Create dialer with timeout
	dialer := &net.Dialer{
		Timeout: time.Duration(monitor.Timeout) * time.Second,
	}

	// Measure connection time
	start := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", address)
	ping := time.Since(start).Milliseconds()

	if err != nil {
		heartbeat.Status = StatusDown
		heartbeat.Ping = int(ping)
		heartbeat.Message = fmt.Sprintf("Connection failed: %v", err)
		return heartbeat, nil
	}
	defer conn.Close()

	heartbeat.Status = StatusUp
	heartbeat.Ping = int(ping)
	heartbeat.Message = fmt.Sprintf("Port %d is open - %dms", port, ping)

	return heartbeat, nil
}

func (t *TCPMonitor) Validate(monitor *Monitor) error {
	if monitor.URL == "" {
		return fmt.Errorf("host is required")
	}

	// Validate port if provided
	if port, ok := monitor.Config["port"]; ok {
		if p, ok := port.(float64); ok {
			if p < 1 || p > 65535 {
				return fmt.Errorf("port must be between 1 and 65535")
			}
		} else {
			return fmt.Errorf("port must be a number")
		}
	}

	return nil
}
