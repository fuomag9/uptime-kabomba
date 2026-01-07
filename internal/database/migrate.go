package database

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/fuomag9/uptime-kabomba/internal/config"
)

// RunMigrations runs database migrations
func RunMigrations(cfg config.DatabaseConfig) error {
	// Connect to database for migrations
	gormDB, err := Connect(cfg)
	if err != nil {
		return err
	}

	// Get underlying SQL database
	sqlDB, err := gormDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}
	defer sqlDB.Close()

	var driver database.Driver
	switch cfg.Type {
	case "sqlite":
		driver, err = sqlite3.WithInstance(sqlDB, &sqlite3.Config{})
	case "postgres":
		driver, err = postgres.WithInstance(sqlDB, &postgres.Config{})
	default:
		return fmt.Errorf("unsupported database type for migrations: %s", cfg.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://./migrations",
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
