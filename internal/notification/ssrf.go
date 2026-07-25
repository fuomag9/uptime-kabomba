package notification

import "fmt"

// OutboundGuard validates destinations before a provider is allowed to
// contact them, so a user-supplied webhook/server/SMTP destination can't be
// used to reach cloud metadata endpoints or internal-only services (SSRF).
// Satisfied by *netsec.SSRFProtection; kept as an interface here so this
// package doesn't need to import internal/monitor or internal/netsec's
// concrete type directly at every call site.
type OutboundGuard interface {
	ValidateURL(rawURL string) error
	ValidateHost(host string) error
}

var outboundGuard OutboundGuard

// SetOutboundGuard installs the SSRF guard used by every provider that sends
// to a user-supplied destination (webhook, discord, slack, teams, gotify,
// ntfy, smtp). Must be called once at startup before any notification is
// sent; if never called, outbound validation is skipped (fail-open avoids
// breaking existing deployments that don't wire it up, but main.go always
// wires it up in practice).
func SetOutboundGuard(g OutboundGuard) {
	outboundGuard = g
}

// validateOutboundURL rejects rawURL if it resolves to a disallowed
// destination (private/loopback/link-local IPs or cloud metadata endpoints,
// per the configured SSRF policy).
func validateOutboundURL(rawURL string) error {
	if outboundGuard == nil {
		return nil
	}
	if err := outboundGuard.ValidateURL(rawURL); err != nil {
		return fmt.Errorf("destination not allowed: %w", err)
	}
	return nil
}

// validateOutboundHost is the bare-host equivalent of validateOutboundURL,
// for providers (SMTP) that take a host:port rather than a full URL.
func validateOutboundHost(host string) error {
	if outboundGuard == nil {
		return nil
	}
	if err := outboundGuard.ValidateHost(host); err != nil {
		return fmt.Errorf("destination not allowed: %w", err)
	}
	return nil
}
