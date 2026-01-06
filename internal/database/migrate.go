package database

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/fuomag9/uptime-kuma-go/internal/config"
)

// RunMigrations runs database migrations
func RunMigrations(cfg config.DatabaseConfig) error {
	// Connect to database for migrations
	db, err := Connect(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	var driver database.Driver
	switch cfg.Type {
	case "sqlite":
		driver, err = sqlite3.WithInstance(db.DB, &sqlite3.Config{})
	case "postgres":
		driver, err = postgres.WithInstance(db.DB, &postgres.Config{})
	default:
		return fmt.Errorf("unsupported database type for migrations: %s", cfg.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		cfg.Type,
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
