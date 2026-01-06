package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/go-ping/ping"
)

// PingMonitor performs ICMP ping checks
type PingMonitor struct{}

func init() {
	RegisterMonitorType(&PingMonitor{})
}

func (p *PingMonitor) Name() string {
	return "ping"
}

func (p *PingMonitor) Check(ctx context.Context, monitor *Monitor) (*Heartbeat, error) {
	heartbeat := &Heartbeat{
		MonitorID: monitor.ID,
		Time:      time.Now(),
	}

	host := monitor.URL
	if host == "" {
		heartbeat.Status = StatusDown
		heartbeat.Message = "No host specified"
		return heartbeat, nil
	}

	// Get packet count from config (default 4)
	count := 4
	if c, ok := monitor.Config["packet_count"].(float64); ok {
		count = int(c)
	}

	// Get packet size from config (default 56 bytes)
	size := 56
	if s, ok := monitor.Config["packet_size"].(float64); ok {
		size = int(s)
	}

	// Create pinger
	pinger, err := ping.NewPinger(host)
	if err != nil {
		heartbeat.Status = StatusDown
		heartbeat.Message = fmt.Sprintf("Failed to create pinger: %v", err)
		return heartbeat, nil
	}

	// Configure pinger
	pinger.Count = count
	pinger.Size = size
	pinger.Timeout = time.Duration(monitor.Timeout) * time.Second
	pinger.SetPrivileged(false) // Use unprivileged mode (UDP) by default

	// Check if privileged mode is requested
	if priv, ok := monitor.Config["privileged"].(bool); ok && priv {
		pinger.SetPrivileged(true)
	}

	// Run ping with context support
	done := make(chan error, 1)
	go func() {
		done <- pinger.Run()
	}()

	select {
	case <-ctx.Done():
		pinger.Stop()
		heartbeat.Status = StatusDown
		heartbeat.Message = "Ping cancelled"
		return heartbeat, nil
	case err := <-done:
		if err != nil {
			heartbeat.Status = StatusDown
			heartbeat.Message = fmt.Sprintf("Ping failed: %v", err)
			return heartbeat, nil
		}
	}

	stats := pinger.Statistics()

	// Check if any packets were received
	if stats.PacketsRecv == 0 {
		heartbeat.Status = StatusDown
		heartbeat.Ping = int(stats.MaxRtt.Milliseconds())
		heartbeat.Message = fmt.Sprintf("No packets received (100%% packet loss)")
		return heartbeat, nil
	}

	// Calculate packet loss percentage
	packetLoss := stats.PacketLoss

	// Consider it down if packet loss > 50%
	if packetLoss > 50 {
		heartbeat.Status = StatusDown
		heartbeat.Ping = int(stats.AvgRtt.Milliseconds())
		heartbeat.Message = fmt.Sprintf("High packet loss: %.1f%% - %dms avg", packetLoss, stats.AvgRtt.Milliseconds())
		return heartbeat, nil
	}

	heartbeat.Status = StatusUp
	heartbeat.Ping = int(stats.AvgRtt.Milliseconds())
	heartbeat.Message = fmt.Sprintf("Ping OK - %dms avg (loss: %.1f%%)", stats.AvgRtt.Milliseconds(), packetLoss)

	return heartbeat, nil
}

func (p *PingMonitor) Validate(monitor *Monitor) error {
	if monitor.URL == "" {
		return fmt.Errorf("host is required")
	}

	// Validate packet count
	if count, ok := monitor.Config["packet_count"]; ok {
		if c, ok := count.(float64); ok {
			if c < 1 || c > 100 {
				return fmt.Errorf("packet count must be between 1 and 100")
			}
		} else {
			return fmt.Errorf("packet_count must be a number")
		}
	}

	// Validate packet size
	if size, ok := monitor.Config["packet_size"]; ok {
		if s, ok := size.(float64); ok {
			if s < 1 || s > 65500 {
				return fmt.Errorf("packet size must be between 1 and 65500")
			}
		} else {
			return fmt.Errorf("packet_size must be a number")
		}
	}

	return nil
}
