package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"golang.org/x/crypto/bcrypt"

	"github.com/fuomag9/uptime-kabomba/internal/config"
	"github.com/fuomag9/uptime-kabomba/internal/models"
)

type contextKey string

const userContextKey contextKey = "user"

// LoginRequest represents login credentials
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token,omitempty"` // 2FA token
}

// LoginResponse represents login response
type LoginResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

// HandleLogin handles user login
func HandleLogin(db *gorm.DB, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Println("Login: Failed to decode request")
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		log.Println("Login: Authentication attempt")

		// Find user
		var user models.User
		// Find user by username or email
		err := db.Where("username = ? OR email = ?", req.Username, req.Username).First(&user).Error
		if err != nil {
			log.Println("Login: Authentication failed - user not found")
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// Check if user has a password (OAuth-only users don't)
		if !user.HasPassword() {
			log.Println("Login: User has no password (OAuth-only account)")
			http.Error(w, "This account uses OAuth authentication. Please sign in with OAuth.", http.StatusUnauthorized)
			return
		}

		// Verify password
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			log.Println("Login: Authentication failed - invalid password")
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		log.Println("Login: Successful authentication")

		// Check 2FA if enabled
		if user.TotpSecret != nil && *user.TotpSecret != "" {
		// WARNING: 2FA is not properly implemented - this is bypassed
		log.Println("WARNING: User has 2FA configured but it is not enforced")
		}

		// Generate JWT
		token, err := generateJWT(user.ID, cfg.JWTSecret)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LoginResponse{
			Token: token,
			User:  &user,
		})
	}
}

// HandleLogout handles user logout
func HandleLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// In a stateless JWT system, logout is handled client-side
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Logged out successfully"}`))
	}
}

// HandleSetup handles initial setup
func HandleSetup(db *gorm.DB, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Println("Error hashing password:", err.Error())
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}

		// Create user with local provider
		provider := "local"
		newUser := models.User{
			Username:  req.Username,
			Password:  string(hashedPassword),
			Provider:  &provider,
			Active:    true,
			CreatedAt: time.Now(),
		}

		err = db.Create(&newUser).Error
		if err != nil {
			log.Println("Error creating user:", err.Error())
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}

		// Generate JWT
		token, err := generateJWT(newUser.ID, cfg.JWTSecret)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LoginResponse{
			Token: token,
			User:  &newUser,
		})
	}
}

// HandleGetCurrentUser returns the current authenticated user
func HandleGetCurrentUser(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}
}

// StatusResponse represents setup status response
type StatusResponse struct {
	SetupComplete bool `json:"setupComplete"`
}

// HandleGetSetupStatus checks if setup has been completed
func HandleGetSetupStatus(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var count int64
		err := db.Model(&models.User{}).Count(&count).Error
		if err != nil {
			log.Println("Error checking user count:", err.Error())
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(StatusResponse{
			SetupComplete: count > 0,
		})
	}
}

// AuthMiddleware validates JWT tokens
func AuthMiddleware(jwtSecret string, db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing authorization header", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			// Parse token
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			claims := token.Claims.(jwt.MapClaims)
			userID := int(claims["user_id"].(float64))

			// Load user from database
			var user models.User
			err = db.Where("id = ?", userID).First(&user).Error
			if err != nil {
				log.Println("AuthMiddleware: Failed to load user:", err.Error())
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, &user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// generateJWT generates a JWT token for a user
func generateJWT(userID int, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(2 * time.Hour).Unix(), // Reduced from 24h to 2h for security
	})

	return token.SignedString([]byte(secret))
}
