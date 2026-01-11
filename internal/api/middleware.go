package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/fuomag9/uptime-kabomba/internal/config"
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

			// Content Security Policy
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self' ws: wss:;")

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

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use IP address as identifier
			identifier := r.RemoteAddr

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

// StrictRateLimitMiddleware creates a stricter rate limiting middleware for auth endpoints
func StrictRateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use IP address as identifier
			identifier := r.RemoteAddr

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
