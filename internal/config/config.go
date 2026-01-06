package config

import (
	"os"
	"strconv"
)

// Config holds application configuration
type Config struct {
	Port         int
	Database     DatabaseConfig
	JWTSecret    string
	Environment  string
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Type         string // sqlite, postgres
	DSN          string
	MaxOpenConns int
	MaxIdleConns int
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		Port: getEnvInt("PORT", 8080),
		Database: DatabaseConfig{
			Type:         getEnv("DATABASE_TYPE", "sqlite"),
			DSN:          getEnv("DATABASE_DSN", "/data/uptime.db"),
			MaxOpenConns: getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 5),
		},
		JWTSecret:   getEnv("JWT_SECRET", generateDefaultSecret()),
		Environment: getEnv("ENVIRONMENT", "production"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}

func generateDefaultSecret() string {
	// In production, this should be set via environment variable
	return "change-this-secret-in-production"
}
