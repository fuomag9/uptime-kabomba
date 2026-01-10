package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fuomag9/uptime-kabomba/internal/api"
	"github.com/fuomag9/uptime-kabomba/internal/config"
	"github.com/fuomag9/uptime-kabomba/internal/database"
	"github.com/fuomag9/uptime-kabomba/internal/jobs"
	"github.com/fuomag9/uptime-kabomba/internal/monitor"
	"github.com/fuomag9/uptime-kabomba/internal/notification"
	"github.com/fuomag9/uptime-kabomba/internal/oauth"
	"github.com/fuomag9/uptime-kabomba/internal/websocket"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Get underlying SQL database for cleanup
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
	}
	defer sqlDB.Close()

	// Run migrations
	if err := database.RunMigrations(cfg.Database); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize WebSocket hub
	hub := websocket.NewHub(cfg.JWTSecret)
	go hub.Run()

	// Initialize notification dispatcher
	dispatcher := notification.NewDispatcher(db)

	// Initialize monitor executor
	executor := monitor.NewExecutor(db, hub, dispatcher)
	if err := executor.Start(); err != nil {
		log.Fatalf("Failed to start monitor executor: %v", err)
	}
	defer executor.Stop()

	// Initialize job scheduler
	scheduler := jobs.NewScheduler(db)
	scheduler.Start()

	// Start OAuth cleanup job if OAuth is enabled
	if cfg.OAuth != nil && cfg.OAuth.Enabled {
		oauth.StartCleanupJob(db)
	}
	defer scheduler.Stop()

	// Setup API router
	router := api.NewRouter(cfg, db, hub, executor, dispatcher)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on port %d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
