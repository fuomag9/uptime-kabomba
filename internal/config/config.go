package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Config holds application configuration
type Config struct {
	Port                   int
	Database               DatabaseConfig
	JWTSecret              string
	Environment            string
	CORSOrigins            []string
	OAuth                  *OAuthConfig
	AllowPrivateIPs        bool
	AllowMetadataEndpoints bool
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Type         string // postgres
	DSN          string
	MaxOpenConns int
	MaxIdleConns int
}

// OAuthConfig holds OAuth2/OIDC configuration
type OAuthConfig struct {
	Enabled      bool
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// Load loads configuration from environment variables
func Load() *Config {
	env := getEnv("ENVIRONMENT", "production")
	jwtSecret := loadJWTSecret(env)
	oauthConfig := loadOAuthConfig()

	cfg := &Config{
		Port: getEnvInt("PORT", 8080),
		Database: DatabaseConfig{
			Type:         getEnv("DATABASE_TYPE", "postgres"),
			DSN:          getEnv("DATABASE_DSN", buildPostgresDSN()),
			MaxOpenConns: getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 5),
		},
		JWTSecret:              jwtSecret,
		Environment:            env,
		CORSOrigins:            loadCORSOrigins(env),
		OAuth:                  oauthConfig,
		AllowPrivateIPs:        getEnvBool("ALLOW_PRIVATE_IPS", false),
		AllowMetadataEndpoints: getEnvBool("ALLOW_METADATA_ENDPOINTS", false),
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	return cfg
}

func buildPostgresDSN() string {
	host := getEnv("POSTGRES_HOST", "localhost")
	port := getEnv("POSTGRES_PORT", "5432")
	user := getEnv("POSTGRES_USER", "uptime")
	password := getEnv("POSTGRES_PASSWORD", "secret")
	dbName := getEnv("POSTGRES_DB", "uptime")
	sslMode := getEnv("POSTGRES_SSLMODE", "disable")

	u := url.URL{
		Scheme: "postgresql",
		User:   url.UserPassword(user, password),
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   dbName,
	}

	query := u.Query()
	query.Set("sslmode", sslMode)
	u.RawQuery = query.Encode()

	return u.String()
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

	if c.Database.Type != "postgres" {
		return fmt.Errorf("unsupported database type: %s", c.Database.Type)
	}

	// Validate OAuth config if enabled
	if c.OAuth != nil && c.OAuth.Enabled {
		if c.OAuth.Issuer == "" {
			return fmt.Errorf("OAUTH_ISSUER is required when OAuth is enabled")
		}
		if c.OAuth.ClientID == "" || c.OAuth.ClientSecret == "" {
			return fmt.Errorf("OAUTH_CLIENT_ID and OAUTH_CLIENT_SECRET are required when OAuth is enabled")
		}
		// OAuth redirect URL is derived from APP_URL
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
	if appURL := getAppURL(); appURL != "" {
		return []string{appURL}
	}

	// Default origins based on environment
	if env == "development" {
		return []string{"http://localhost:3000", "http://localhost:8080"}
	}

	// In production, require explicit CORS configuration
	log.Println("WARNING: APP_URL not set. Using default localhost origins.")
	log.Println("WARNING: Set APP_URL environment variable for production deployments.")
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

func getEnvBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
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

func getAppURL() string {
	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		return ""
	}
	return strings.TrimRight(appURL, "/")
}

func loadOAuthConfig() *OAuthConfig {
	issuer := os.Getenv("OAUTH_ISSUER")
	if issuer == "" {
		return &OAuthConfig{Enabled: false}
	}

	clientID := os.Getenv("OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("OAUTH_CLIENT_SECRET")
	redirectURL := ""
	if appURL := getAppURL(); appURL != "" {
		redirectURL = appURL + "/oauth/callback"
	}

	if clientID == "" || clientSecret == "" {
		log.Println("WARNING: OAUTH_ISSUER is set but OAUTH_CLIENT_ID or OAUTH_CLIENT_SECRET is missing. OAuth will be disabled.")
		return &OAuthConfig{Enabled: false}
	}

	// Default scopes
	scopes := []string{"openid", "profile", "email"}
	if scopesEnv := os.Getenv("OAUTH_SCOPES"); scopesEnv != "" {
		scopes = splitAndTrim(scopesEnv, ",")
	}

	return &OAuthConfig{
		Enabled:      true,
		Issuer:       issuer,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
	}
}
