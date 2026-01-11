package monitor

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// HTTPMonitor implements HTTP/HTTPS monitoring
type HTTPMonitor struct{}

func init() {
	RegisterMonitorType(&HTTPMonitor{})
}

// Name returns the monitor type name
func (h *HTTPMonitor) Name() string {
	return "http"
}

// Validate validates the HTTP monitor configuration
func (h *HTTPMonitor) Validate(monitor *Monitor) error {
	if monitor.URL == "" {
		return fmt.Errorf("URL is required for HTTP monitor")
	}

	if !strings.HasPrefix(monitor.URL, "http://") && !strings.HasPrefix(monitor.URL, "https://") {
		return fmt.Errorf("URL must start with http:// or https://")
	}

	// SSRF Protection - validate URL to prevent access to private IPs and metadata endpoints
	ssrfProtection := NewSSRFProtection(false) // Don't allow private IPs by default
	if err := ssrfProtection.ValidateURL(monitor.URL); err != nil {
		return fmt.Errorf("URL validation failed: %w", err)
	}

	if monitor.Timeout <= 0 {
		monitor.Timeout = 30
	}

	if monitor.Interval <= 0 {
		monitor.Interval = 60
	}

	return nil
}

// Check performs the HTTP check
func (h *HTTPMonitor) Check(ctx context.Context, monitor *Monitor) (*Heartbeat, error) {
	heartbeat := &Heartbeat{
		MonitorID: monitor.ID,
		Time:      time.Now(),
		Status:    StatusDown,
	}

	// Get config values
	method := h.getConfigString(monitor, "method", "GET")
	headers := h.getConfigMap(monitor, "headers")
	body := h.getConfigString(monitor, "body", "")
	acceptedStatusCodes := h.getConfigIntSlice(monitor, "accepted_status_codes", []int{200})
	keyword := h.getConfigString(monitor, "keyword", "")
	invertKeyword := h.getConfigBool(monitor, "invert_keyword", false)
	ignoreTLS := h.getConfigBool(monitor, "ignore_tls", false)
	followRedirects := h.getConfigBool(monitor, "follow_redirects", true)

	// Create HTTP client with IP version support
	client := &http.Client{
		Timeout: time.Duration(monitor.Timeout) * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// Determine network based on IP version preference
				network = GetNetworkForIPVersion(network, monitor.IPVersion)
				dialer := &net.Dialer{
					Timeout: time.Duration(monitor.Timeout) * time.Second,
				}
				return dialer.DialContext(ctx, network, addr)
			},
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: ignoreTLS,
			},
		},
	}

	// Disable redirects if needed
	if !followRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// Create request
	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, monitor.URL, reqBody)
	if err != nil {
		heartbeat.Message = fmt.Sprintf("Failed to create request: %v", err)
		return heartbeat, nil
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Perform request and measure time
	start := time.Now()
	resp, err := client.Do(req)
	ping := time.Since(start).Milliseconds()
	heartbeat.Ping = int(ping)

	if err != nil {
		heartbeat.Message = fmt.Sprintf("Request failed: %v", err)
		return heartbeat, nil
	}
	defer resp.Body.Close()

	// Check status code
	statusOK := false
	for _, code := range acceptedStatusCodes {
		if resp.StatusCode == code {
			statusOK = true
			break
		}
	}

	if !statusOK {
		heartbeat.Message = fmt.Sprintf("Unexpected status code: %d", resp.StatusCode)
		return heartbeat, nil
	}

	// Check keyword if specified
	if keyword != "" {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			heartbeat.Message = fmt.Sprintf("Failed to read response body: %v", err)
			return heartbeat, nil
		}

		bodyText := string(bodyBytes)
		containsKeyword := strings.Contains(bodyText, keyword)

		if invertKeyword {
			// Keyword should NOT be present
			if containsKeyword {
				heartbeat.Message = fmt.Sprintf("Keyword '%s' found (inverted check)", keyword)
				return heartbeat, nil
			}
		} else {
			// Keyword should be present
			if !containsKeyword {
				heartbeat.Message = fmt.Sprintf("Keyword '%s' not found", keyword)
				return heartbeat, nil
			}
		}
	}

	// All checks passed
	heartbeat.Status = StatusUp
	heartbeat.Message = fmt.Sprintf("HTTP %d - %dms", resp.StatusCode, ping)

	return heartbeat, nil
}

// Helper methods to get config values
func (h *HTTPMonitor) getConfigString(monitor *Monitor, key, defaultValue string) string {
	if monitor.Config == nil {
		return defaultValue
	}
	if val, ok := monitor.Config[key].(string); ok {
		return val
	}
	return defaultValue
}

func (h *HTTPMonitor) getConfigInt(monitor *Monitor, key string, defaultValue int) int {
	if monitor.Config == nil {
		return defaultValue
	}
	if val, ok := monitor.Config[key].(float64); ok {
		return int(val)
	}
	return defaultValue
}

func (h *HTTPMonitor) getConfigBool(monitor *Monitor, key string, defaultValue bool) bool {
	if monitor.Config == nil {
		return defaultValue
	}
	if val, ok := monitor.Config[key].(bool); ok {
		return val
	}
	return defaultValue
}

func (h *HTTPMonitor) getConfigMap(monitor *Monitor, key string) map[string]string {
	if monitor.Config == nil {
		return make(map[string]string)
	}
	if val, ok := monitor.Config[key].(map[string]interface{}); ok {
		result := make(map[string]string)
		for k, v := range val {
			if str, ok := v.(string); ok {
				result[k] = str
			}
		}
		return result
	}
	return make(map[string]string)
}

func (h *HTTPMonitor) getConfigIntSlice(monitor *Monitor, key string, defaultValue []int) []int {
	if monitor.Config == nil {
		return defaultValue
	}
	if val, ok := monitor.Config[key].([]interface{}); ok {
		result := make([]int, 0, len(val))
		for _, v := range val {
			if num, ok := v.(float64); ok {
				result = append(result, int(num))
			}
		}
		return result
	}
	return defaultValue
}
