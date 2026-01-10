package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strconv"
)

// Config holds application configuration
type Config struct {
	Port         int
	Database     DatabaseConfig
	JWTSecret    string
	Environment  string
	CORSOrigins  []string
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
	env := getEnv("ENVIRONMENT", "production")
	jwtSecret := loadJWTSecret(env)

	cfg := &Config{
		Port: getEnvInt("PORT", 8080),
		Database: DatabaseConfig{
			Type:         getEnv("DATABASE_TYPE", "sqlite"),
			DSN:          getEnv("DATABASE_DSN", "/data/uptime.db"),
			MaxOpenConns: getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 5),
		},
		JWTSecret:   jwtSecret,
		Environment: env,
		CORSOrigins: loadCORSOrigins(env),
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	return cfg
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Environment == "production" {
		if len(c.JWTSecret) < 32 {
			return fmt.Errorf("JWT_SECRET must be at least 32 characters in production")
		}

		// Check for insecure default secrets
		insecureSecrets := []string{
			"change-this-secret-in-production",
			"change-me-in-production",
			"secret",
			"password",
			"changeme",
		}
		for _, insecure := range insecureSecrets {
			if c.JWTSecret == insecure {
				return fmt.Errorf("JWT_SECRET is set to an insecure default value. Please set a strong random secret")
			}
		}
	}

	if len(c.CORSOrigins) == 0 {
		return fmt.Errorf("at least one CORS origin must be configured")
	}

	return nil
}

func loadJWTSecret(env string) string {
	secret := os.Getenv("JWT_SECRET")

	// If JWT_SECRET is not set, generate a random one for development
	if secret == "" {
		if env == "production" {
			log.Fatal("FATAL: JWT_SECRET environment variable is required in production")
		}

		// Generate random secret for development
		log.Println("WARNING: JWT_SECRET not set. Generating random secret for development.")
		log.Println("WARNING: This secret will change on restart. Set JWT_SECRET in production!")
		return generateRandomSecret()
	}

	// Validate secret length
	if len(secret) < 16 {
		log.Fatal("FATAL: JWT_SECRET must be at least 16 characters long")
	}

	return secret
}

func loadCORSOrigins(env string) []string {
	originsEnv := os.Getenv("CORS_ORIGINS")

	if originsEnv != "" {
		// Parse comma-separated origins
		origins := []string{}
		for _, origin := range splitAndTrim(originsEnv, ",") {
			if origin != "" {
				origins = append(origins, origin)
			}
		}
		return origins
	}

	// Default origins based on environment
	if env == "development" {
		return []string{"http://localhost:3000", "http://localhost:8080"}
	}

	// In production, require explicit CORS configuration
	log.Println("WARNING: CORS_ORIGINS not set. Using default localhost origins.")
	log.Println("WARNING: Set CORS_ORIGINS environment variable for production deployments.")
	return []string{"http://localhost:3000", "http://localhost:8080"}
}

func splitAndTrim(s, sep string) []string {
	parts := []string{}
	for i := 0; i < len(s); {
		j := i
		for j < len(s) && string(s[j]) != sep {
			j++
		}
		part := s[i:j]
		// Trim spaces
		start, end := 0, len(part)
		for start < end && part[start] == ' ' {
			start++
		}
		for end > start && part[end-1] == ' ' {
			end--
		}
		if start < end {
			parts = append(parts, part[start:end])
		}
		i = j + 1
	}
	return parts
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

func generateRandomSecret() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatal("Failed to generate random secret:", err)
	}
	return base64.URLEncoding.EncodeToString(bytes)
}
