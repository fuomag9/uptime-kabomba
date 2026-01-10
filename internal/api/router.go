package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"gorm.io/gorm"

	"github.com/fuomag9/uptime-kabomba/internal/config"
	"github.com/fuomag9/uptime-kabomba/internal/websocket"
	"github.com/fuomag9/uptime-kabomba/internal/monitor"
	"github.com/fuomag9/uptime-kabomba/internal/notification"
)

// NewRouter creates a new HTTP router
func NewRouter(cfg *config.Config, db *gorm.DB, hub *websocket.Hub, executor *monitor.Executor, dispatcher *notification.Dispatcher) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// CORS - use configured origins
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Security headers
	r.Use(SecurityHeadersMiddleware)

	// Global rate limiter - 100 requests per minute per IP
	globalLimiter := NewRateLimiter(100.0/60.0, 20)
	globalLimiter.CleanupOldLimiters()
	r.Use(RateLimitMiddleware(globalLimiter))

	// Strict rate limiter for auth endpoints - 5 requests per 15 minutes
	authLimiter := NewRateLimiter(5.0/900.0, 5)
	authLimiter.CleanupOldLimiters()

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Auth routes with strict rate limiting
		r.With(StrictRateLimitMiddleware(authLimiter)).Post("/auth/login", HandleLogin(db, cfg))
		r.Post("/auth/logout", HandleLogout())
		r.With(StrictRateLimitMiddleware(authLimiter)).Post("/auth/setup", HandleSetup(db, cfg))
		r.Get("/auth/status", HandleGetSetupStatus(db))

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware(cfg.JWTSecret, db))

			// User routes
			r.Get("/user/me", HandleGetCurrentUser(db))

			// Monitor routes
			r.Get("/monitors", HandleGetMonitors(db))
			r.Post("/monitors", HandleCreateMonitor(db, executor))
			r.Get("/monitors/{id}", HandleGetMonitor(db))
			r.Put("/monitors/{id}", HandleUpdateMonitor(db, executor))
			r.Delete("/monitors/{id}", HandleDeleteMonitor(db, executor))
			r.Get("/monitors/{id}/heartbeats", HandleGetHeartbeats(db))
			r.Get("/monitors/{id}/uptime", HandleGetMonitorUptime(db))
			r.Get("/monitors/{id}/uptime/history", HandleGetMonitorUptimeHistory(db))
			r.Get("/monitors/{id}/uptime/hourly", HandleGetMonitorHourlyUptime(db))
			r.Get("/monitors/uptime/all", HandleGetAllMonitorsUptime(db))

			// Notification routes
			r.Get("/notifications", HandleGetNotificationsV2(db))
			r.Post("/notifications", HandleCreateNotification(db))
			r.Get("/notifications/providers", HandleGetAvailableProviders())
			r.Get("/notifications/{id}", HandleGetNotification(db))
			r.Put("/notifications/{id}", HandleUpdateNotification(db))
			r.Delete("/notifications/{id}", HandleDeleteNotification(db))
			r.Post("/notifications/{id}/test", HandleTestNotification(db, dispatcher))

			// Status Page routes (management)
			r.Get("/status-pages", HandleGetStatusPages(db))
			r.Post("/status-pages", HandleCreateStatusPage(db))
			r.Get("/status-pages/{id}", HandleGetStatusPage(db))
			r.Put("/status-pages/{id}", HandleUpdateStatusPage(db))
			r.Delete("/status-pages/{id}", HandleDeleteStatusPage(db))
			r.Get("/status-pages/{id}/incidents", HandleGetIncidents(db))
			r.Post("/status-pages/{id}/incidents", HandleCreateIncident(db))
			r.Delete("/status-pages/{id}/incidents/{incidentId}", HandleDeleteIncident(db))

			// API Key routes
			r.Get("/api-keys", HandleGetAPIKeys(db))
			r.Post("/api-keys", HandleCreateAPIKey(db))
			r.Delete("/api-keys/{id}", HandleDeleteAPIKey(db))
		})
	})

	// Public status page endpoint (no auth required)
	r.Get("/status/{slug}", HandleGetPublicStatusPage(db))

	// Prometheus metrics endpoint (no auth required)
	r.Get("/metrics", HandlePrometheusMetrics(db))

	// Badge endpoints (no auth required)
	r.Get("/api/badge/{id}/status", HandleStatusBadge(db))
	r.Get("/api/badge/{id}/uptime", HandleUptimeBadge(db))
	r.Get("/api/badge/{id}/ping", HandlePingBadge(db))

	// WebSocket endpoint
	r.Get("/ws", hub.HandleWebSocket)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return r
}
