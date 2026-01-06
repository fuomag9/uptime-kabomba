package database

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"           // PostgreSQL driver
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/fuomag9/uptime-kuma-go/internal/config"
)

// Connect establishes a database connection
func Connect(cfg config.DatabaseConfig) (*sqlx.DB, error) {
	var db *sqlx.DB
	var err error

	switch cfg.Type {
	case "sqlite":
		db, err = sqlx.Connect("sqlite3", cfg.DSN+"?_foreign_keys=on&_journal_mode=WAL")
	case "postgres":
		db, err = sqlx.Connect("postgres", cfg.DSN)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)

	// Ping to verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
