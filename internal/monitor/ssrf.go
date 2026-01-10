package monitor

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// SSRFProtection validates URLs to prevent Server-Side Request Forgery attacks
type SSRFProtection struct {
	allowPrivateIPs bool
}

// NewSSRFProtection creates a new SSRF protection validator
func NewSSRFProtection(allowPrivateIPs bool) *SSRFProtection {
	return &SSRFProtection{
		allowPrivateIPs: allowPrivateIPs,
	}
}

// ValidateURL validates a URL against SSRF attacks
func (s *SSRFProtection) ValidateURL(rawURL string) error {
	// Parse URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Check scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("only http and https schemes are allowed")
	}

	// Extract hostname
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL must have a hostname")
	}

	// Check for blocked hostnames
	if s.isBlockedHostname(hostname) {
		return fmt.Errorf("access to this hostname is not allowed")
	}

	// Resolve hostname to IP
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return fmt.Errorf("failed to resolve hostname: %w", err)
	}

	if len(ips) == 0 {
		return fmt.Errorf("hostname does not resolve to any IP address")
	}

	// Check each resolved IP
	for _, ip := range ips {
		if err := s.validateIP(ip); err != nil {
			return fmt.Errorf("IP address %s is not allowed: %w", ip.String(), err)
		}
	}

	return nil
}

// isBlockedHostname checks if a hostname is explicitly blocked
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

	for _, blocked := range localhostVariations {
		if hostname == blocked {
			return !s.allowPrivateIPs
		}
	}

	// Block cloud metadata endpoints
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

	return false
}

// validateIP checks if an IP address is allowed
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

// isPrivateIP checks if an IP is in a private range
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
			"fc00::/7",   // Unique local address
			"fe80::/10",  // Link-local
			"::1/128",    // Loopback
			"fd00::/8",   // Unique local address (more specific)
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
