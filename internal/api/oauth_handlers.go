package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/fuomag9/uptime-kabomba/internal/config"
	"github.com/fuomag9/uptime-kabomba/internal/models"
	"github.com/fuomag9/uptime-kabomba/internal/oauth"
)

// HandleOAuthAuthorize initiates the OAuth authorization flow
func HandleOAuthAuthorize(db *gorm.DB, cfg *config.Config, oauthClient *oauth.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.OAuth == nil || !cfg.OAuth.Enabled {
			http.Error(w, "OAuth is not enabled", http.StatusNotFound)
			return
		}

		// Generate state and code verifier for PKCE
		state, err := oauth.GenerateState()
		if err != nil {
			log.Println("OAuth: Failed to generate state:", err)
			http.Error(w, "Failed to initiate OAuth", http.StatusInternalServerError)
			return
		}

		codeVerifier, err := oauth.GenerateCodeVerifier()
		if err != nil {
			log.Println("OAuth: Failed to generate code verifier:", err)
			http.Error(w, "Failed to initiate OAuth", http.StatusInternalServerError)
			return
		}

		// Determine redirect URL (use configured or construct from request)
		redirectURL := cfg.OAuth.RedirectURL
		if redirectURL == "" {
			redirectURL = getRedirectURL(r)
		}

		// Store session in database (10 minute expiry)
		session := models.OAuthSession{
			State:        state,
			CodeVerifier: codeVerifier,
			RedirectURI:  &redirectURL,
			ExpiresAt:    time.Now().Add(10 * time.Minute),
			CreatedAt:    time.Now(),
		}

		if err := db.Create(&session).Error; err != nil {
			log.Println("OAuth: Failed to create session:", err)
			http.Error(w, "Failed to initiate OAuth", http.StatusInternalServerError)
			return
		}

		// Generate code challenge for PKCE
		codeChallenge := oauth.GenerateCodeChallenge(codeVerifier)

		// Redirect to authorization endpoint
		authURL := oauthClient.GetAuthorizationURL(state, codeChallenge, redirectURL)
		log.Println("OAuth: Redirecting to authorization URL")
		http.Redirect(w, r, authURL, http.StatusFound)
	}
}

// OAuthCallbackResponse represents the OAuth callback response
type OAuthCallbackResponse struct {
	Action       string       `json:"action"` // "login", "link_required", "register", "error"
	Token        *string      `json:"token,omitempty"`
	User         *models.User `json:"user,omitempty"`
	LinkingToken *string      `json:"linking_token,omitempty"`
	Email        *string      `json:"email,omitempty"`
	Message      string       `json:"message,omitempty"`
}

// HandleOAuthCallback processes the OAuth callback
func HandleOAuthCallback(db *gorm.DB, cfg *config.Config, oauthClient *oauth.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		if code == "" || state == "" {
			log.Println("OAuth: Invalid callback - missing code or state")
			http.Error(w, "Invalid OAuth callback", http.StatusBadRequest)
			return
		}

		// Verify state and get code verifier
		var session models.OAuthSession
		err := db.Where("state = ? AND expires_at > ?", state, time.Now()).First(&session).Error
		if err != nil {
			log.Println("OAuth: Invalid or expired state:", err)
			http.Error(w, "Invalid or expired OAuth session", http.StatusBadRequest)
			return
		}

		// Get redirect URL from session
		redirectURL := cfg.OAuth.RedirectURL
		if session.RedirectURI != nil && *session.RedirectURI != "" {
			redirectURL = *session.RedirectURI
		}

		// Delete session (one-time use)
		db.Delete(&session)

		// Exchange code for access token
		ctx := r.Context()
		accessToken, err := oauthClient.ExchangeCode(ctx, code, session.CodeVerifier, redirectURL)
		if err != nil {
			log.Println("OAuth: Token exchange failed:", err)
			http.Error(w, "Failed to exchange authorization code", http.StatusInternalServerError)
			return
		}

		// Get user info from provider
		userInfo, err := oauthClient.GetUserInfo(ctx, accessToken)
		if err != nil {
			log.Println("OAuth: Failed to get user info:", err)
			http.Error(w, "Failed to get user information", http.StatusInternalServerError)
			return
		}

		// Handle user authentication/registration/linking
		response := handleOAuthUser(db, cfg, userInfo)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// handleOAuthUser processes OAuth user login/registration/linking
func handleOAuthUser(db *gorm.DB, cfg *config.Config, userInfo *oauth.UserInfo) OAuthCallbackResponse {
	provider := "oidc"

	// Scenario 1: Check if user already exists with this provider+subject
	var existingOAuthUser models.User
	err := db.Where("provider = ? AND subject = ?", provider, userInfo.Subject).First(&existingOAuthUser).Error
	if err == nil {
		// User exists with OAuth, log them in
		log.Printf("OAuth: Existing OAuth user logging in: %s", userInfo.Email)
		token, err := generateJWT(existingOAuthUser.ID, cfg.JWTSecret)
		if err != nil {
			log.Println("OAuth: Failed to generate JWT:", err)
			return OAuthCallbackResponse{
				Action:  "error",
				Message: "Failed to generate authentication token",
			}
		}

		return OAuthCallbackResponse{
			Action: "login",
			Token:  &token,
			User:   &existingOAuthUser,
		}
	}

	// Scenario 2: Check if email exists in system (potential account linking)
	var existingEmailUser models.User
	err = db.Where("email = ?", userInfo.Email).First(&existingEmailUser).Error
	if err == nil {
		// Email exists - require manual linking with password verification
		log.Printf("OAuth: Email collision detected for %s - requiring account linking", userInfo.Email)

		linkingToken := generateLinkingToken()

		// Store linking token in database
		oauthDataJSON, _ := json.Marshal(userInfo)
		oauthDataStr := string(oauthDataJSON)

		linking := models.OAuthLinkingToken{
			Token:     linkingToken,
			UserID:    existingEmailUser.ID,
			Provider:  provider,
			Subject:   userInfo.Subject,
			Email:     userInfo.Email,
			OAuthData: &oauthDataStr,
			ExpiresAt: time.Now().Add(5 * time.Minute),
			CreatedAt: time.Now(),
		}

		if err := db.Create(&linking).Error; err != nil {
			log.Println("OAuth: Failed to create linking token:", err)
			return OAuthCallbackResponse{
				Action:  "error",
				Message: "Failed to create account linking token",
			}
		}

		return OAuthCallbackResponse{
			Action:       "link_required",
			LinkingToken: &linkingToken,
			Email:        &userInfo.Email,
			Message:      "An account with this email already exists. Please verify your password to link accounts.",
		}
	}

	// Scenario 3: New user - create account
	log.Printf("OAuth: Creating new user from OAuth: %s", userInfo.Email)

	username := generateUsername(userInfo.Email, db)
	oauthDataJSON, _ := json.Marshal(userInfo)
	oauthDataStr := string(oauthDataJSON)

	newUser := models.User{
		Username:  username,
		Email:     &userInfo.Email,
		Password:  "oauth-no-password", // Placeholder for OAuth-only users
		Provider:  &provider,
		Subject:   &userInfo.Subject,
		OAuthData: &oauthDataStr,
		Active:    true,
		CreatedAt: time.Now(),
	}

	if err := db.Create(&newUser).Error; err != nil {
		log.Println("OAuth: Failed to create user:", err)
		return OAuthCallbackResponse{
			Action:  "error",
			Message: "Failed to create user account",
		}
	}

	// Generate JWT for new user
	token, err := generateJWT(newUser.ID, cfg.JWTSecret)
	if err != nil {
		log.Println("OAuth: Failed to generate JWT:", err)
		return OAuthCallbackResponse{
			Action:  "error",
			Message: "Failed to generate authentication token",
		}
	}

	return OAuthCallbackResponse{
		Action: "register",
		Token:  &token,
		User:   &newUser,
	}
}

// LinkAccountRequest represents account linking request
type LinkAccountRequest struct {
	LinkingToken string `json:"linking_token"`
	Password     string `json:"password"`
}

// HandleOAuthLinkAccount links an OAuth account to existing user
func HandleOAuthLinkAccount(db *gorm.DB, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LinkAccountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Find linking token
		var linking models.OAuthLinkingToken
		err := db.Where("token = ? AND expires_at > ?", req.LinkingToken, time.Now()).First(&linking).Error
		if err != nil {
			log.Println("OAuth: Invalid or expired linking token:", err)
			http.Error(w, "Invalid or expired linking token", http.StatusBadRequest)
			return
		}

		// Get user
		var user models.User
		if err := db.First(&user, linking.UserID).Error; err != nil {
			log.Println("OAuth: User not found for linking:", err)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Verify password
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			log.Printf("OAuth: Password verification failed for user %d", user.ID)
			// Delete linking token after failed attempt (security measure)
			db.Delete(&linking)
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}

		log.Printf("OAuth: Linking OAuth account to user %d (%s)", user.ID, user.Username)

		// Link accounts - update user with OAuth provider and subject
		user.Provider = &linking.Provider
		user.Subject = &linking.Subject
		if user.Email == nil {
			user.Email = &linking.Email
		}
		user.OAuthData = linking.OAuthData

		if err := db.Save(&user).Error; err != nil {
			log.Println("OAuth: Failed to link account:", err)
			http.Error(w, "Failed to link account", http.StatusInternalServerError)
			return
		}

		// Delete linking token (successful linking)
		db.Delete(&linking)

		// Generate JWT
		token, err := generateJWT(user.ID, cfg.JWTSecret)
		if err != nil {
			log.Println("OAuth: Failed to generate JWT after linking:", err)
			http.Error(w, "Failed to generate authentication token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LoginResponse{
			Token: token,
			User:  &user,
		})
	}
}

// HandleGetOAuthConfig returns OAuth configuration for frontend
func HandleGetOAuthConfig(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := struct {
			Enabled bool   `json:"enabled"`
			Issuer  string `json:"issuer,omitempty"`
		}{
			Enabled: cfg.OAuth != nil && cfg.OAuth.Enabled,
		}

		if response.Enabled {
			response.Issuer = cfg.OAuth.Issuer
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// Helper functions

// generateLinkingToken generates a random token for account linking
func generateLinkingToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// generateUsername generates a unique username from email
func generateUsername(email string, db *gorm.DB) string {
	// Extract base username from email
	parts := strings.Split(email, "@")
	baseUsername := parts[0]
	if baseUsername == "" {
		baseUsername = "user"
	}

	// Check if username exists
	username := baseUsername
	counter := 1

	for {
		var existingUser models.User
		err := db.Where("username = ?", username).First(&existingUser).Error
		if err == gorm.ErrRecordNotFound {
			// Username is available
			break
		}

		// Username exists, try with counter
		counter++
		username = fmt.Sprintf("%s%d", baseUsername, counter)
	}

	return username
}

// getRedirectURL constructs the OAuth redirect URL from the request
// Returns the FRONTEND URL, not the backend URL, since OIDC provider
// should redirect users to the frontend, not directly to backend
func getRedirectURL(r *http.Request) string {
	// Try to get frontend URL from Origin or Referer header
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = r.Header.Get("Referer")
		if origin != "" {
			// Extract just the origin from referer URL
			if idx := strings.Index(origin[8:], "/"); idx != -1 {
				origin = origin[:8+idx]
			}
		}
	}

	// Fallback to localhost:3000 if no origin detected
	if origin == "" {
		origin = "http://localhost:3000"
	}

	// OAuth callback goes to frontend, not backend
	return origin + "/oauth/callback"
}
