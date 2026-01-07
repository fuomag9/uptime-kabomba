package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"

	"github.com/fuomag9/uptime-kuma-go/internal/models"
)

// HandleGetAPIKeys returns all API keys for the current user
func HandleGetAPIKeys(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)

		var apiKeys []models.APIKey
		query := `SELECT id, user_id, name, key_hash, prefix, scopes, expires_at, last_used_at, created_at
		          FROM api_keys WHERE user_id = ? ORDER BY created_at DESC`

		err := db.Select(&apiKeys, query, user.ID)
		if err != nil {
			http.Error(w, "Failed to fetch API keys", http.StatusInternalServerError)
			return
		}

		// Parse scopes for each key
		for i := range apiKeys {
			apiKeys[i].AfterLoad()
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apiKeys)
	}
}

// HandleCreateAPIKey creates a new API key
func HandleCreateAPIKey(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)

		var req struct {
			Name      string    `json:"name"`
			Scopes    []string  `json:"scopes"`
			ExpiresAt *string   `json:"expires_at,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			http.Error(w, "Name is required", http.StatusBadRequest)
			return
		}

		if len(req.Scopes) == 0 {
			http.Error(w, "At least one scope is required", http.StatusBadRequest)
			return
		}

		// Validate scopes
		validScopes := map[string]bool{"read": true, "write": true, "admin": true}
		for _, scope := range req.Scopes {
			if !validScopes[scope] {
				http.Error(w, "Invalid scope: "+scope, http.StatusBadRequest)
				return
			}
		}

		// Generate API key (32 random bytes = 43 base64 chars)
		keyBytes := make([]byte, 32)
		if _, err := rand.Read(keyBytes); err != nil {
			http.Error(w, "Failed to generate API key", http.StatusInternalServerError)
			return
		}
		apiKey := base64.URLEncoding.EncodeToString(keyBytes)

		// Hash the key
		keyHash, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Failed to hash API key", http.StatusInternalServerError)
			return
		}

		// Get prefix (first 8 chars for display)
		prefix := apiKey[:8]

		// Parse expiration
		var expiresAt *time.Time
		if req.ExpiresAt != nil && *req.ExpiresAt != "" {
			t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
			if err != nil {
				http.Error(w, "Invalid expires_at format", http.StatusBadRequest)
				return
			}
			expiresAt = &t
		}

		// Marshal scopes
		scopesJSON, err := json.Marshal(req.Scopes)
		if err != nil {
			http.Error(w, "Failed to marshal scopes", http.StatusInternalServerError)
			return
		}

		// Insert into database
		query := `
			INSERT INTO api_keys (user_id, name, key_hash, prefix, scopes, expires_at, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			RETURNING id
		`

		var keyID int
		err = db.QueryRow(query, user.ID, req.Name, string(keyHash), prefix, string(scopesJSON), expiresAt, time.Now()).Scan(&keyID)
		if err != nil {
			http.Error(w, "Failed to create API key", http.StatusInternalServerError)
			return
		}

		// Return the key ONLY ONCE (never stored in plain text)
		response := map[string]interface{}{
			"id":         keyID,
			"name":       req.Name,
			"prefix":     prefix,
			"key":        apiKey, // ONLY sent once
			"scopes":     req.Scopes,
			"expires_at": expiresAt,
			"created_at": time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}
}

// HandleDeleteAPIKey deletes an API key
func HandleDeleteAPIKey(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*models.User)
		keyID := chi.URLParam(r, "id")

		// Delete from database
		result, err := db.Exec("DELETE FROM api_keys WHERE id = ? AND user_id = ?", keyID, user.ID)
		if err != nil {
			http.Error(w, "Failed to delete API key", http.StatusInternalServerError)
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			http.Error(w, "API key not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// APIKeyAuthMiddleware authenticates requests using API keys
func APIKeyAuthMiddleware(db *sqlx.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get API key from header
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				// Try Authorization header (Bearer token)
				authHeader := r.Header.Get("Authorization")
				if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
					apiKey = authHeader[7:]
				}
			}

			if apiKey == "" {
				http.Error(w, "API key required", http.StatusUnauthorized)
				return
			}

			// Get all API keys (we need to hash check each one)
			var apiKeys []models.APIKey
			query := `SELECT id, user_id, name, key_hash, prefix, scopes, expires_at, last_used_at, created_at FROM api_keys`
			err := db.Select(&apiKeys, query)
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Find matching key
			var matchedKey *models.APIKey
			for i := range apiKeys {
				if err := bcrypt.CompareHashAndPassword([]byte(apiKeys[i].KeyHash), []byte(apiKey)); err == nil {
					matchedKey = &apiKeys[i]
					break
				}
			}

			if matchedKey == nil {
				http.Error(w, "Invalid API key", http.StatusUnauthorized)
				return
			}

			// Check expiration
			if matchedKey.IsExpired() {
				http.Error(w, "API key expired", http.StatusUnauthorized)
				return
			}

			// Parse scopes
			matchedKey.AfterLoad()

			// Update last_used_at
			db.Exec("UPDATE api_keys SET last_used_at = ? WHERE id = ?", time.Now(), matchedKey.ID)

			// Get user
			var user models.User
			userQuery := `SELECT id, username, active, created_at FROM users WHERE id = ?`
			err = db.Get(&user, userQuery, matchedKey.UserID)
			if err != nil {
				http.Error(w, "User not found", http.StatusUnauthorized)
				return
			}

			if !user.Active {
				http.Error(w, "User inactive", http.StatusUnauthorized)
				return
			}

			// Store user and scopes in context
			ctx := r.Context()
			ctx = setUserContext(ctx, &user)
			ctx = setAPIKeyContext(ctx, matchedKey)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// apiKeyContextKey is the context key for API key
const apiKeyContextKey contextKey = "api_key"

func setUserContext(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func setAPIKeyContext(ctx context.Context, key *models.APIKey) context.Context {
	return context.WithValue(ctx, apiKeyContextKey, key)
}

func getAPIKeyFromContext(ctx context.Context) *models.APIKey {
	key, _ := ctx.Value(apiKeyContextKey).(*models.APIKey)
	return key
}
