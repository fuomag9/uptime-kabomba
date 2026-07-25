// Package netsec centralizes SSRF (Server-Side Request Forgery) protection so
// every outbound network call the server makes on a user's behalf - HTTP/page
// change monitor checks, ping/DNS/TCP monitor checks, and notification
// provider deliveries - is validated against the same policy.
package netsec

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"slices"
	"strings"
)

// SSRFProtection validates hostnames/URLs to prevent Server-Side Request
// Forgery attacks against private networks and cloud metadata endpoints.
type SSRFProtection struct {
	allowPrivateIPs        bool
	allowMetadataEndpoints bool
}

// NewSSRFProtection creates a new SSRF protection validator.
func NewSSRFProtection(allowPrivateIPs bool, allowMetadataEndpoints bool) *SSRFProtection {
	return &SSRFProtection{
		allowPrivateIPs:        allowPrivateIPs,
		allowMetadataEndpoints: allowMetadataEndpoints,
	}
}

// ValidateURL validates a URL (scheme must be http/https) against SSRF
// attacks. It resolves the hostname and rejects the URL if any resolved IP is
// disallowed. Intended for one-off validation (e.g. at monitor/notification
// create time) - it does not pin an IP, so it does not by itself protect
// against DNS-rebinding between validation and the later connection. Callers
// that make the actual outbound connection should prefer ResolveHost/
// SafeDialContext immediately before dialing.
func (s *SSRFProtection) ValidateURL(rawURL string) error {
	hostname, err := s.hostnameFromURL(rawURL)
	if err != nil {
		return err
	}
	_, err = s.ResolveHost(hostname)
	return err
}

// ValidateHost validates a bare hostname or IP literal (no URL scheme),
// resolving and checking every IP it maps to. Used by monitor types that take
// a plain host (ping, tcp, dns server) instead of a full URL.
func (s *SSRFProtection) ValidateHost(host string) error {
	_, err := s.ResolveHost(host)
	return err
}

// hostnameFromURL parses rawURL, requires an http/https scheme, and returns
// its hostname.
func (s *SSRFProtection) hostnameFromURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("only http and https schemes are allowed")
	}

	hostname := parsedURL.Hostname()
	if hostname == "" {
		return "", fmt.Errorf("URL must have a hostname")
	}

	return hostname, nil
}

// ResolveHost resolves hostname (or parses it as an IP literal directly),
// validates every resolved/parsed IP against the SSRF policy, and returns one
// validated IP to connect to. Resolution and validation happen atomically in
// this single call so callers that dial the returned IP immediately (rather
// than letting a dialer re-resolve the hostname later) are not vulnerable to
// DNS-rebinding TOCTOU races.
func (s *SSRFProtection) ResolveHost(hostname string) (net.IP, error) {
	if hostname == "" {
		return nil, fmt.Errorf("hostname is required")
	}

	if s.isBlockedHostname(hostname) {
		return nil, fmt.Errorf("access to this hostname is not allowed")
	}

	// IP literal (including bracketed IPv6 like "[::1]") - no DNS involved.
	if ip := net.ParseIP(strings.Trim(hostname, "[]")); ip != nil {
		if err := s.validateIP(ip); err != nil {
			return nil, fmt.Errorf("IP address %s is not allowed: %w", ip.String(), err)
		}
		return ip, nil
	}

	ips, err := net.LookupIP(hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve hostname: %w", err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("hostname does not resolve to any IP address")
	}

	// Reject the hostname outright if any resolved IP is disallowed, rather
	// than silently picking only the "safe" ones - a mixed public/private
	// answer is itself a signal something is wrong (e.g. rebinding already in
	// progress at resolution time).
	for _, ip := range ips {
		if err := s.validateIP(ip); err != nil {
			return nil, fmt.Errorf("IP address %s is not allowed: %w", ip.String(), err)
		}
	}

	return ips[0], nil
}

// SafeDialContext returns a DialContext-compatible function that resolves,
// validates, and pins the destination IP atomically for every connection
// attempt, dialing the validated IP literal directly instead of a hostname.
// Because net/http's Transport invokes DialContext again for every redirect
// hop, using this as a client's Transport.DialContext also closes the
// SSRF-via-redirect gap: a redirect to a disallowed host is re-validated (and
// blocked) at the moment of connection, not just against the original URL.
func (s *SSRFProtection) SafeDialContext(baseDialer *net.Dialer) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid address %q: %w", addr, err)
		}

		ip, err := s.ResolveHost(host)
		if err != nil {
			return nil, err
		}

		return baseDialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
	}
}

// isBlockedHostname checks if a hostname is explicitly blocked.
func (s *SSRFProtection) isBlockedHostname(hostname string) bool {
	hostname = strings.ToLower(hostname)

	// Block common localhost variations
	localhostVariations := []string{
		"localhost",
		"localhost.localdomain",
		"127.0.0.1",
		"[::1]",
		"::1",
		"0.0.0.0",
	}

	if slices.Contains(localhostVariations, hostname) {
		return !s.allowPrivateIPs
	}

	// Block cloud metadata endpoints unless explicitly allowed
	if !s.allowMetadataEndpoints {
		metadataEndpoints := []string{
			"169.254.169.254", // AWS, Azure, GCP metadata
			"metadata.google.internal",
			"169.254.170.2", // AWS ECS metadata
			"fd00:ec2::254", // AWS IMDSv2 IPv6
		}

		for _, blocked := range metadataEndpoints {
			if hostname == blocked || strings.HasSuffix(hostname, "."+blocked) {
				return true
			}
		}
	}

	return false
}

// validateIP checks if an IP address is allowed.
func (s *SSRFProtection) validateIP(ip net.IP) error {
	// If private IPs are allowed, skip checks
	if s.allowPrivateIPs {
		return nil
	}

	// Check for private IP ranges
	if s.isPrivateIP(ip) {
		return fmt.Errorf("access to private IP addresses is not allowed")
	}

	// Check for loopback
	if ip.IsLoopback() {
		return fmt.Errorf("access to loopback addresses is not allowed")
	}

	// Check for link-local
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("access to link-local addresses is not allowed")
	}

	// Check for multicast
	if ip.IsMulticast() {
		return fmt.Errorf("access to multicast addresses is not allowed")
	}

	// Check for unspecified (0.0.0.0 or ::)
	if ip.IsUnspecified() {
		return fmt.Errorf("access to unspecified addresses is not allowed")
	}

	return nil
}

// isPrivateIP checks if an IP is in a private range.
func (s *SSRFProtection) isPrivateIP(ip net.IP) bool {
	// Private IPv4 ranges
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16", // Link-local / AWS metadata
		"127.0.0.0/8",    // Loopback
	}

	for _, cidr := range privateRanges {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(ip) {
			return true
		}
	}

	// Private IPv6 ranges
	if ip.To4() == nil {
		// IPv6
		privateV6Ranges := []string{
			"fc00::/7",  // Unique local address
			"fe80::/10", // Link-local
			"::1/128",   // Loopback
			"fd00::/8",  // Unique local address (more specific)
		}

		for _, cidr := range privateV6Ranges {
			_, network, _ := net.ParseCIDR(cidr)
			if network.Contains(ip) {
				return true
			}
		}
	}

	return false
}
