package api

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/fuomag9/uptime-kabomba/internal/config"
	"github.com/fuomag9/uptime-kabomba/internal/models"
	"golang.org/x/time/rate"
)

// SecurityHeadersMiddleware adds security headers to all responses
func SecurityHeadersMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent clickjacking
			w.Header().Set("X-Frame-Options", "DENY")

			// Prevent MIME sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// XSS Protection
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			// Content Security Policy. Notably applies to the badge SVG
			// endpoints (image/svg+xml, served publicly with no auth): SVG
			// rendered as a top-level navigation executes any <script> it
			// contains unless script-src blocks it, so no 'unsafe-inline'/
			// 'unsafe-eval' here even though this middleware also covers
			// plain JSON API responses that don't need a CSP at all.
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self' ws: wss:;")

			// Referrer Policy
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Permissions Policy
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			// HSTS - enable in production
			if cfg.Environment == "production" {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimiter stores rate limiters per identifier (IP or user)
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    b,
	}
}

// GetLimiter returns a rate limiter for the given identifier
func (rl *RateLimiter) GetLimiter(identifier string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[identifier]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[identifier] = limiter
	}

	return limiter
}

// CleanupOldLimiters removes limiters that haven't been used recently
func (rl *RateLimiter) CleanupOldLimiters() {
	ticker := time.NewTicker(10 * time.Minute)
	go func() {
		for range ticker.C {
			rl.mu.Lock()
			// Simple cleanup - could be improved with last-used tracking
			if len(rl.limiters) > 10000 {
				rl.limiters = make(map[string]*rate.Limiter)
			}
			rl.mu.Unlock()
		}
	}()
}

// RateLimitMiddleware creates a rate limiting middleware keyed on the
// un-spoofable client IP (see ClientIP - X-Forwarded-For is only honored
// from a configured trusted proxy).
func RateLimitMiddleware(limiter *RateLimiter, trustedProxies []*net.IPNet) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identifier := ClientIP(r.RemoteAddr, r.Header.Get("X-Forwarded-For"), trustedProxies)

			// Get the rate limiter for this identifier
			lim := limiter.GetLimiter(identifier)

			if !lim.Allow() {
				http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// StrictRateLimitMiddleware creates a stricter rate limiting middleware for
// auth endpoints, keyed the same way as RateLimitMiddleware.
func StrictRateLimitMiddleware(limiter *RateLimiter, trustedProxies []*net.IPNet) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identifier := ClientIP(r.RemoteAddr, r.Header.Get("X-Forwarded-For"), trustedProxies)

			// Get the rate limiter for this identifier
			lim := limiter.GetLimiter(identifier)

			if !lim.Allow() {
				http.Error(w, "Too many attempts. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// UserRateLimitMiddleware creates a rate limiting middleware keyed on the
// authenticated user's ID rather than IP. Must run after AuthMiddleware.
// Used for endpoints where the abuse case is "one account doing this too
// often" regardless of source IP (e.g. the notification test endpoint).
func UserRateLimitMiddleware(limiter *RateLimiter, trustedProxies []*net.IPNet) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identifier := ClientIP(r.RemoteAddr, r.Header.Get("X-Forwarded-For"), trustedProxies)
			if user, ok := r.Context().Value(userContextKey).(*models.User); ok {
				identifier = "user:" + strconv.Itoa(user.ID)
			}

			lim := limiter.GetLimiter(identifier)
			if !lim.Allow() {
				http.Error(w, "Too many attempts. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// LoginAttemptTracker tracks failed login attempts per account identifier
// (independent of source IP), so an attacker rotating X-Forwarded-For/IP
// can't bypass the per-account brute-force lockout the way they could bypass
// pure IP-based rate limiting. This makes the account lockout a second,
// independent line of defense rather than making the IP limiter the *only*
// thing standing between an attacker and unlimited password guesses.
type LoginAttemptTracker struct {
	mu       sync.Mutex
	attempts map[string]*loginAttemptState
}

type loginAttemptState struct {
	count       int
	lockedUntil time.Time
	lastAttempt time.Time
}

const (
	maxLoginAttempts   = 5
	loginLockoutWindow = 15 * time.Minute
)

// NewLoginAttemptTracker creates a new LoginAttemptTracker.
func NewLoginAttemptTracker() *LoginAttemptTracker {
	return &LoginAttemptTracker{attempts: make(map[string]*loginAttemptState)}
}

// IsLocked reports whether key (e.g. a normalized username) is currently
// locked out.
func (t *LoginAttemptTracker) IsLocked(key string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	state, ok := t.attempts[key]
	if !ok {
		return false
	}
	return time.Now().Before(state.lockedUntil)
}

// RecordFailure records a failed attempt for key, locking it out once
// maxLoginAttempts is reached within loginLockoutWindow.
func (t *LoginAttemptTracker) RecordFailure(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	state, ok := t.attempts[key]
	if !ok {
		state = &loginAttemptState{}
		t.attempts[key] = state
	} else if !state.lastAttempt.IsZero() && now.Sub(state.lastAttempt) > loginLockoutWindow {
		// Previous failure aged out of the window - start counting again.
		state.count = 0
	}

	state.count++
	state.lastAttempt = now
	if state.count >= maxLoginAttempts {
		state.lockedUntil = now.Add(loginLockoutWindow)
	}
}

// RecordSuccess clears any tracked failures for key.
func (t *LoginAttemptTracker) RecordSuccess(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.attempts, key)
}

// Cleanup periodically purges stale entries so the map doesn't grow
// unbounded from usernames tried once and never retried.
func (t *LoginAttemptTracker) Cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	go func() {
		for range ticker.C {
			t.mu.Lock()
			now := time.Now()
			for k, state := range t.attempts {
				if now.After(state.lockedUntil) && now.Sub(state.lastAttempt) > loginLockoutWindow {
					delete(t.attempts, k)
				}
			}
			t.mu.Unlock()
		}
	}()
}
