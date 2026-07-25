package api

import (
	"net"
	"strings"
)

// ParseTrustedProxies converts configured CIDR (or bare IP) strings into
// *net.IPNet for use with ClientIP. Invalid entries are silently skipped.
func ParseTrustedProxies(cidrs []string) []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if !strings.Contains(c, "/") {
			if ip := net.ParseIP(c); ip != nil {
				if ip.To4() != nil {
					c += "/32"
				} else {
					c += "/128"
				}
			}
		}
		if _, network, err := net.ParseCIDR(c); err == nil {
			nets = append(nets, network)
		}
	}
	return nets
}

func isTrustedProxy(ip net.IP, trusted []*net.IPNet) bool {
	for _, n := range trusted {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// ClientIP returns the real client IP for r's RemoteAddr/X-Forwarded-For,
// honoring X-Forwarded-For only when the immediate TCP peer is a configured
// trusted proxy.
//
// This exists because chi's middleware.RealIP - previously installed
// unconditionally - trusts X-Forwarded-For (and X-Real-IP/True-Client-IP)
// from ANY client with no validation, so any request could set its own
// X-Forwarded-For and get a fresh rate-limit bucket every time, completely
// defeating both the global and the auth-endpoint rate limiters. Without a
// configured trusted proxy, X-Forwarded-For is ignored entirely and the raw
// TCP peer address is used - secure by default, at the cost of every request
// behind an *unconfigured* reverse proxy sharing one rate-limit bucket.
// Operators running behind a reverse proxy must set TRUSTED_PROXIES to that
// proxy's address/CIDR to get accurate per-client limiting.
func ClientIP(remoteAddr, xForwardedFor string, trustedProxies []*net.IPNet) string {
	peerIP, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		peerIP = remoteAddr
	}

	peer := net.ParseIP(peerIP)
	if peer == nil || len(trustedProxies) == 0 || !isTrustedProxy(peer, trustedProxies) {
		return peerIP
	}

	if xForwardedFor == "" {
		return peerIP
	}

	// Walk right-to-left; the rightmost entry that isn't itself a trusted
	// proxy is the real client, per the standard reverse-proxy-chain model.
	// This also means a client-supplied X-Forwarded-For prefix (the part an
	// untrusted caller controls) is never trusted - only entries appended by
	// a proxy we recognize are.
	parts := strings.Split(xForwardedFor, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		candidate := strings.TrimSpace(parts[i])
		ip := net.ParseIP(candidate)
		if ip == nil {
			continue
		}
		if !isTrustedProxy(ip, trustedProxies) {
			return candidate
		}
	}

	return peerIP
}
