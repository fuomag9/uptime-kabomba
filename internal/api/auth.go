package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/fuomag9/uptime-kabomba/internal/config"
	"github.com/fuomag9/uptime-kabomba/internal/models"
)

type contextKey string

const userContextKey contextKey = "user"

// LoginRequest represents login credentials
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents login response. The access/refresh tokens
// themselves are never included here - they're set as HttpOnly cookies
// (see issueSession) so client-side JS can never read them, which is the
// whole point: a token sitting in a JSON response is one JSON.parse (or one
// XSS bug) away from being just as readable as if it were in localStorage.
type LoginResponse struct {
	User *models.User `json:"user"`
}

// HandleLogin handles user login
func HandleLogin(db *gorm.DB, cfg *config.Config, loginAttempts *LoginAttemptTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Println("Login: Failed to decode request")
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		log.Println("Login: Authentication attempt")

		// Per-account lockout, independent of source IP: the IP-based
		// StrictRateLimitMiddleware in front of this handler is the first
		// line of defense, but it's still just one shared bucket per
		// (proxy-reported) IP. This ensures a single account can't be
		// brute-forced no matter how many source IPs/proxies are involved.
		attemptKey := strings.ToLower(strings.TrimSpace(req.Username))
		if attemptKey != "" && loginAttempts.IsLocked(attemptKey) {
			log.Println("Login: Account temporarily locked due to repeated failures")
			http.Error(w, "Too many failed login attempts. Please try again later.", http.StatusTooManyRequests)
			return
		}

		// Find user
		var user models.User
		// Find user by username or email
		err := db.Where("username = ? OR email = ?", req.Username, req.Username).First(&user).Error
		if err != nil {
			log.Println("Login: Authentication failed - user not found")
			if attemptKey != "" {
				loginAttempts.RecordFailure(attemptKey)
			}
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
			if attemptKey != "" {
				loginAttempts.RecordFailure(attemptKey)
			}
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		log.Println("Login: Successful authentication")
		if attemptKey != "" {
			loginAttempts.RecordSuccess(attemptKey)
		}

		if err := issueSession(w, db, cfg, user.ID); err != nil {
			log.Println("Login: Failed to issue session:", err)
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LoginResponse{
			User: &user,
		})
	}
}

// HandleLogout handles user logout
func HandleLogout(db *gorm.DB, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Actually revoke the access token (by jti) and the refresh token,
		// rather than relying on the client to just forget them - a copied
		// cookie value would otherwise stay valid until its natural expiry.
		if raw := accessTokenFromRequest(r); raw != "" {
			if claims, err := parseAccessTokenClaims(raw, cfg.JWTSecret); err == nil {
				if jti, ok := claims["jti"].(string); ok {
					if exp, ok := claims["exp"].(float64); ok {
						revokeAccessToken(db, jti, time.Unix(int64(exp), 0))
					}
				}
			}
		}
		if cookie, err := r.Cookie(refreshCookieName); err == nil {
			revokeRefreshToken(db, cookie.Value)
		}

		clearAuthCookies(w, cfg)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Logged out successfully"}`))
	}
}

// HandleRefreshToken exchanges a valid refresh token cookie for a new
// access+refresh token pair. Rotation: the presented refresh token is
// revoked as part of issuing its replacement, so if it's ever replayed
// again (e.g. because it was stolen and the legitimate user already
// refreshed) the replay fails closed rather than silently succeeding.
func HandleRefreshToken(db *gorm.DB, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(refreshCookieName)
		if err != nil || cookie.Value == "" {
			http.Error(w, "Missing refresh token", http.StatusUnauthorized)
			return
		}

		var stored models.RefreshToken
		if err := db.Where("token_hash = ?", hashRefreshToken(cookie.Value)).First(&stored).Error; err != nil {
			http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
			return
		}

		if stored.RevokedAt != nil || time.Now().After(stored.ExpiresAt) {
			http.Error(w, "Refresh token expired or revoked", http.StatusUnauthorized)
			return
		}

		var user models.User
		if err := db.Where("id = ? AND active = ?", stored.UserID, true).First(&user).Error; err != nil {
			http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
			return
		}

		// Rotate: revoke the token just used before issuing its replacement.
		revokeRefreshToken(db, cookie.Value)

		if err := issueSession(w, db, cfg, user.ID); err != nil {
			log.Println("Refresh: Failed to issue session:", err)
			http.Error(w, "Failed to refresh session", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Session refreshed"}`))
	}
}

// HandleSetup handles initial setup
func HandleSetup(db *gorm.DB, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var count int64
		if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
			log.Println("Error checking user count:", err.Error())
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		if count > 0 {
			http.Error(w, "Setup already completed", http.StatusForbidden)
			return
		}

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
		newUser := models.User{
			Username:  req.Username,
			Password:  string(hashedPassword),
			Provider:  new("local"),
			Active:    true,
			CreatedAt: time.Now(),
		}

		err = db.Create(&newUser).Error
		if err != nil {
			log.Println("Error creating user:", err.Error())
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}

		if err := issueSession(w, db, cfg, newUser.ID); err != nil {
			log.Println("Setup: Failed to issue session:", err)
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LoginResponse{
			User: &newUser,
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

// AuthMiddleware validates JWT tokens, read from either the Authorization
// header (API/programmatic clients) or the access_token cookie (the web
// app - see setAuthCookies).
func AuthMiddleware(jwtSecret string, db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := accessTokenFromRequest(r)
			if tokenString == "" {
				http.Error(w, "Missing authentication", http.StatusUnauthorized)
				return
			}

			claims, err := parseAccessTokenClaims(tokenString, jwtSecret)
			if err != nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Reject tokens that were explicitly revoked (e.g. by logout)
			// before their natural expiry - see revokeAccessToken.
			if jti, ok := claims["jti"].(string); ok && isAccessTokenRevoked(db, jti) {
				http.Error(w, "Token revoked", http.StatusUnauthorized)
				return
			}

			userID := int(claims["user_id"].(float64))

			// Load user from database
			var user models.User
			err = db.Where("id = ?", userID).First(&user).Error
			if err != nil {
				log.Println("AuthMiddleware: Failed to load user:", err.Error())
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Check if user is active
			if !user.Active {
				http.Error(w, "Account disabled", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, &user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

