package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"github.com/fuomag9/uptime-kabomba/internal/config"
	"github.com/fuomag9/uptime-kabomba/internal/models"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour

	accessCookieName  = "access_token"
	refreshCookieName = "refresh_token"
)

// generateOpaqueToken returns a URL-safe, base64-encoded random token with
// 256 bits of entropy - used for both JWT jti values and refresh tokens.
func generateOpaqueToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashRefreshToken deterministically hashes a refresh token for storage/
// lookup. SHA-256 (not bcrypt) is appropriate here because the input is
// already 256 bits of random entropy, not a low-entropy secret like a
// password - a slow KDF adds cost with no security benefit and would
// prevent the indexed `WHERE token_hash = ?` lookup the refresh flow needs.
func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// generateAccessToken issues a short-lived, signed JWT carrying a unique jti
// so it can be individually revoked (see revokeAccessToken) before it
// naturally expires.
func generateAccessToken(userID int, secret string) (tokenString string, jti string, expiresAt time.Time, err error) {
	jti, err = generateOpaqueToken()
	if err != nil {
		return "", "", time.Time{}, err
	}

	expiresAt = time.Now().Add(accessTokenTTL)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"jti":     jti,
		"exp":     expiresAt.Unix(),
	})

	tokenString, err = token.SignedString([]byte(secret))
	return tokenString, jti, expiresAt, err
}

// issueRefreshToken creates and persists a new refresh token for userID,
// returning the raw value to send to the client (only the hash is stored).
func issueRefreshToken(db *gorm.DB, userID int) (string, time.Time, error) {
	raw, err := generateOpaqueToken()
	if err != nil {
		return "", time.Time{}, err
	}

	expiresAt := time.Now().Add(refreshTokenTTL)
	rt := models.RefreshToken{
		UserID:    userID,
		TokenHash: hashRefreshToken(raw),
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	if err := db.Create(&rt).Error; err != nil {
		return "", time.Time{}, err
	}

	return raw, expiresAt, nil
}

// setAuthCookies sets the access and refresh token cookies on the response.
// Both are HttpOnly (unreadable by JS - the point of moving off
// localStorage) and Secure outside development. The refresh cookie is
// scoped to /api/auth so it's never sent on the far larger set of ordinary
// API requests that only ever need the access token.
func setAuthCookies(w http.ResponseWriter, cfg *config.Config, accessToken string, accessExpiresAt time.Time, refreshToken string, refreshExpiresAt time.Time) {
	secure := cfg.Environment != "development"

	http.SetCookie(w, &http.Cookie{
		Name:     accessCookieName,
		Value:    accessToken,
		Path:     "/",
		Expires:  accessExpiresAt,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    refreshToken,
		Path:     "/api/auth",
		Expires:  refreshExpiresAt,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearAuthCookies removes both auth cookies from the browser (logout).
func clearAuthCookies(w http.ResponseWriter, cfg *config.Config) {
	secure := cfg.Environment != "development"

	http.SetCookie(w, &http.Cookie{
		Name:     accessCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/api/auth",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// issueSession generates a fresh access+refresh token pair, persists the
// refresh token, and sets both cookies on the response. This is the common
// path after any successful login/setup/registration/account-link/refresh.
func issueSession(w http.ResponseWriter, db *gorm.DB, cfg *config.Config, userID int) error {
	accessToken, _, accessExp, err := generateAccessToken(userID, cfg.JWTSecret)
	if err != nil {
		return fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, refreshExp, err := issueRefreshToken(db, userID)
	if err != nil {
		return fmt.Errorf("failed to issue refresh token: %w", err)
	}

	setAuthCookies(w, cfg, accessToken, accessExp, refreshToken, refreshExp)
	return nil
}

// revokeAccessToken adds jti to the revocation list so AuthMiddleware
// rejects it even before its natural expiry (used on logout).
func revokeAccessToken(db *gorm.DB, jti string, expiresAt time.Time) {
	if jti == "" {
		return
	}
	db.Create(&models.RevokedAccessToken{
		JTI:       jti,
		ExpiresAt: expiresAt,
		RevokedAt: time.Now(),
	})
}

// revokeRefreshToken marks the refresh token matching raw as revoked (rather
// than deleting the row), so a later replay of an already-used/revoked
// token can still be recognized - e.g. as a signal of token theft - instead
// of simply looking like "not found".
func revokeRefreshToken(db *gorm.DB, raw string) {
	if raw == "" {
		return
	}
	db.Model(&models.RefreshToken{}).
		Where("token_hash = ? AND revoked_at IS NULL", hashRefreshToken(raw)).
		Update("revoked_at", time.Now())
}

// revokeAllRefreshTokensForUser revokes every outstanding refresh token for
// userID - used on password change so a leaked refresh token stops working
// the moment the user rotates their password.
func revokeAllRefreshTokensForUser(db *gorm.DB, userID int) {
	db.Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now())
}

// isAccessTokenRevoked reports whether jti is on the revocation list.
func isAccessTokenRevoked(db *gorm.DB, jti string) bool {
	if jti == "" {
		return false
	}
	var count int64
	db.Model(&models.RevokedAccessToken{}).Where("jti = ?", jti).Count(&count)
	return count > 0
}

// parseAccessTokenClaims parses and validates tokenString as an HS256 JWT
// signed with secret (algorithm pinned to prevent an alg-confusion/"none"
// attack) and checks its expiry. It does not check the revocation list or
// load the user - callers do that separately since not all of them need it.
func parseAccessTokenClaims(tokenString, secret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		if token.Method.Alg() != "HS256" {
			return nil, fmt.Errorf("unexpected signing algorithm: %v", token.Method.Alg())
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, fmt.Errorf("token has no expiry")
	}
	if time.Now().Unix() > int64(exp) {
		return nil, fmt.Errorf("token expired")
	}

	return claims, nil
}

// accessTokenFromRequest extracts the raw JWT from either the Authorization
// header (Bearer scheme - kept for non-browser/API clients) or the
// access_token cookie (browser clients, set by setAuthCookies).
func accessTokenFromRequest(r *http.Request) string {
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		if token, ok := strings.CutPrefix(authHeader, "Bearer "); ok {
			return token
		}
	}
	if cookie, err := r.Cookie(accessCookieName); err == nil {
		return cookie.Value
	}
	return ""
}
