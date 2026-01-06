package monitor

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// DNSMonitor performs DNS query checks
type DNSMonitor struct{}

func init() {
	RegisterMonitorType(&DNSMonitor{})
}

func (d *DNSMonitor) Name() string {
	return "dns"
}

func (d *DNSMonitor) Check(ctx context.Context, monitor *Monitor) (*Heartbeat, error) {
	heartbeat := &Heartbeat{
		MonitorID: monitor.ID,
		Time:      time.Now(),
	}

	hostname := monitor.URL
	if hostname == "" {
		heartbeat.Status = StatusDown
		heartbeat.Message = "No hostname specified"
		return heartbeat, nil
	}

	// Get DNS server from config (default to system resolver)
	dnsServer := ""
	if server, ok := monitor.Config["dns_server"].(string); ok {
		dnsServer = server
	}

	// Get query type from config (default A record)
	queryType := "A"
	if qt, ok := monitor.Config["query_type"].(string); ok {
		queryType = strings.ToUpper(qt)
	}

	// Get expected result from config (optional)
	expectedResult := ""
	if expected, ok := monitor.Config["expected_result"].(string); ok {
		expectedResult = expected
	}

	// Create resolver
	resolver := &net.Resolver{}
	if dnsServer != "" {
		// Use custom DNS server
		if !strings.Contains(dnsServer, ":") {
			dnsServer = dnsServer + ":53"
		}
		resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Duration(monitor.Timeout) * time.Second,
				}
				return d.DialContext(ctx, network, dnsServer)
			},
		}
	}

	// Create context with timeout
	queryCtx, cancel := context.WithTimeout(ctx, time.Duration(monitor.Timeout)*time.Second)
	defer cancel()

	// Measure query time
	start := time.Now()
	var results []string
	var err error

	switch queryType {
	case "A":
		var ips []net.IP
		ips, err = resolver.LookupIP(queryCtx, "ip4", hostname)
		for _, ip := range ips {
			results = append(results, ip.String())
		}
	case "AAAA":
		var ips []net.IP
		ips, err = resolver.LookupIP(queryCtx, "ip6", hostname)
		for _, ip := range ips {
			results = append(results, ip.String())
		}
	case "CNAME":
		var cname string
		cname, err = resolver.LookupCNAME(queryCtx, hostname)
		if cname != "" {
			results = append(results, cname)
		}
	case "MX":
		var mxs []*net.MX
		mxs, err = resolver.LookupMX(queryCtx, hostname)
		for _, mx := range mxs {
			results = append(results, fmt.Sprintf("%s (priority: %d)", mx.Host, mx.Pref))
		}
	case "NS":
		var nss []*net.NS
		nss, err = resolver.LookupNS(queryCtx, hostname)
		for _, ns := range nss {
			results = append(results, ns.Host)
		}
	case "TXT":
		var txts []string
		txts, err = resolver.LookupTXT(queryCtx, hostname)
		results = txts
	default:
		heartbeat.Status = StatusDown
		heartbeat.Message = fmt.Sprintf("Unsupported query type: %s", queryType)
		return heartbeat, nil
	}

	ping := time.Since(start).Milliseconds()

	if err != nil {
		heartbeat.Status = StatusDown
		heartbeat.Ping = int(ping)
		heartbeat.Message = fmt.Sprintf("DNS query failed: %v", err)
		return heartbeat, nil
	}

	if len(results) == 0 {
		heartbeat.Status = StatusDown
		heartbeat.Ping = int(ping)
		heartbeat.Message = fmt.Sprintf("No %s records found", queryType)
		return heartbeat, nil
	}

	// Check if result matches expected (if specified)
	if expectedResult != "" {
		found := false
		for _, result := range results {
			if strings.Contains(result, expectedResult) {
				found = true
				break
			}
		}
		if !found {
			heartbeat.Status = StatusDown
			heartbeat.Ping = int(ping)
			heartbeat.Message = fmt.Sprintf("Expected result '%s' not found in: %s", expectedResult, strings.Join(results, ", "))
			return heartbeat, nil
		}
	}

	heartbeat.Status = StatusUp
	heartbeat.Ping = int(ping)
	heartbeat.Message = fmt.Sprintf("%s query OK - %s - %dms", queryType, strings.Join(results, ", "), ping)

	return heartbeat, nil
}

func (d *DNSMonitor) Validate(monitor *Monitor) error {
	if monitor.URL == "" {
		return fmt.Errorf("hostname is required")
	}

	// Validate query type
	if qt, ok := monitor.Config["query_type"]; ok {
		if queryType, ok := qt.(string); ok {
			validTypes := []string{"A", "AAAA", "CNAME", "MX", "NS", "TXT"}
			valid := false
			queryType = strings.ToUpper(queryType)
			for _, vt := range validTypes {
				if queryType == vt {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("query_type must be one of: A, AAAA, CNAME, MX, NS, TXT")
			}
		} else {
			return fmt.Errorf("query_type must be a string")
		}
	}

	return nil
}
