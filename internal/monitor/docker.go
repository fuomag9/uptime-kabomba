package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/client"
)

// DockerMonitor checks if a Docker container is running
type DockerMonitor struct{}

func init() {
	RegisterMonitorType(&DockerMonitor{})
}

func (d *DockerMonitor) Name() string {
	return "docker"
}

func (d *DockerMonitor) Check(ctx context.Context, monitor *Monitor) (*Heartbeat, error) {
	heartbeat := &Heartbeat{
		MonitorID: monitor.ID,
		Time:      time.Now(),
	}

	containerName := monitor.URL
	if containerName == "" {
		heartbeat.Status = StatusDown
		heartbeat.Message = "No container name specified"
		return heartbeat, nil
	}

	// Get Docker host from config (default to local socket)
	dockerHost := ""
	if host, ok := monitor.Config["docker_host"].(string); ok {
		dockerHost = host
	}

	// Create Docker client
	var cli *client.Client
	var err error

	if dockerHost != "" {
		cli, err = client.NewClientWithOpts(
			client.WithHost(dockerHost),
			client.WithAPIVersionNegotiation(),
		)
	} else {
		cli, err = client.NewClientWithOpts(
			client.FromEnv,
			client.WithAPIVersionNegotiation(),
		)
	}

	if err != nil {
		heartbeat.Status = StatusDown
		heartbeat.Message = fmt.Sprintf("Failed to create Docker client: %v", err)
		return heartbeat, nil
	}
	defer cli.Close()

	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, time.Duration(monitor.Timeout)*time.Second)
	defer cancel()

	// Measure response time
	start := time.Now()

	// Inspect container
	containerJSON, err := cli.ContainerInspect(checkCtx, containerName)
	ping := time.Since(start).Milliseconds()

	if err != nil {
		heartbeat.Status = StatusDown
		heartbeat.Ping = int(ping)
		heartbeat.Message = fmt.Sprintf("Container not found: %v", err)
		return heartbeat, nil
	}

	// Check if container is running
	if !containerJSON.State.Running {
		heartbeat.Status = StatusDown
		heartbeat.Ping = int(ping)
		heartbeat.Message = fmt.Sprintf("Container is %s", containerJSON.State.Status)
		return heartbeat, nil
	}

	// Check if container is healthy (if health check is configured)
	if containerJSON.State.Health != nil {
		health := containerJSON.State.Health.Status
		if health != "healthy" && health != "" {
			// If health check exists but is not healthy
			if health == "starting" {
				// Container is starting, consider it pending
				heartbeat.Status = StatusPending
				heartbeat.Ping = int(ping)
				heartbeat.Message = fmt.Sprintf("Container is starting (health: %s)", health)
				return heartbeat, nil
			}
			heartbeat.Status = StatusDown
			heartbeat.Ping = int(ping)
			heartbeat.Message = fmt.Sprintf("Container is unhealthy (health: %s)", health)
			return heartbeat, nil
		}
		heartbeat.Message = fmt.Sprintf("Container is running and healthy - %dms", ping)
	} else {
		heartbeat.Message = fmt.Sprintf("Container is running - %dms", ping)
	}

	heartbeat.Status = StatusUp
	heartbeat.Ping = int(ping)

	return heartbeat, nil
}

func (d *DockerMonitor) Validate(monitor *Monitor) error {
	if monitor.URL == "" {
		return fmt.Errorf("container name or ID is required")
	}

	// Validate Docker host format if provided
	if host, ok := monitor.Config["docker_host"]; ok {
		if _, ok := host.(string); !ok {
			return fmt.Errorf("docker_host must be a string")
		}
	}

	return nil
}
