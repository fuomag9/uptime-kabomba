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
			log.Println("Login: Failed to decode request:", err.Error())
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		log.Println("Login attempt for username:", req.Username)

		// Find user
		var user models.User
		err := db.Where("username = ?", req.Username).First(&user).Error
		if err != nil {
			log.Println("Login: User not found:", req.Username, "Error:", err.Error())
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		log.Println("Login: User found, checking password. Hash:", user.Password[:20]+"...")

		// Verify password
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			log.Println("Login: Password check failed:", err.Error())
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		log.Println("Login: Success for user:", req.Username)

		// Check 2FA if enabled
		if user.TotpSecret != nil && *user.TotpSecret != "" {
			// TODO: Implement 2FA verification
			// For now, skip 2FA check
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

		// Check if setup is already done
		var count int64
		err := db.Model(&models.User{}).Count(&count).Error
		if err != nil {
			log.Println("Error checking user count:", err.Error())
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		if count > 0 {
			http.Error(w, "Setup already completed", http.StatusBadRequest)
			return
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Println("Error hashing password:", err.Error())
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}

		// Create user
		newUser := models.User{
			Username:  req.Username,
			Password:  string(hashedPassword),
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
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	return token.SignedString([]byte(secret))
}
