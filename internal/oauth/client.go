package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fuomag9/uptime-kabomba/internal/config"
)

// Client handles OIDC operations
type Client struct {
	config     *config.OAuthConfig
	discovery  *OIDCDiscovery
	httpClient *http.Client
}

// OIDCDiscovery holds OIDC provider metadata from .well-known/openid-configuration
type OIDCDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	JwksURI               string `json:"jwks_uri"`
}

// UserInfo holds user information from OIDC provider
type UserInfo struct {
	Subject string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// NewClient creates a new OAuth client with OIDC discovery
func NewClient(cfg *config.OAuthConfig) (*Client, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, fmt.Errorf("OAuth is not enabled")
	}

	client := &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Perform OIDC discovery
	if err := client.discover(); err != nil {
		return nil, fmt.Errorf("OIDC discovery failed: %w", err)
	}

	return client, nil
}

// discover performs OIDC discovery to find endpoints
func (c *Client) discover() error {
	discoveryURL := strings.TrimSuffix(c.config.Issuer, "/") + "/.well-known/openid-configuration"

	resp, err := c.httpClient.Get(discoveryURL)
	if err != nil {
		return fmt.Errorf("failed to fetch discovery document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discovery endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var discovery OIDCDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return fmt.Errorf("failed to decode discovery document: %w", err)
	}

	// Validate required fields
	if discovery.AuthorizationEndpoint == "" || discovery.TokenEndpoint == "" || discovery.UserinfoEndpoint == "" {
		return fmt.Errorf("discovery document missing required endpoints")
	}

	c.discovery = &discovery
	return nil
}

// GenerateState generates a random state parameter for CSRF protection
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GenerateCodeVerifier generates a PKCE code verifier
func GenerateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate code verifier: %w", err)
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b), nil
}

// GenerateCodeChallenge generates a PKCE code challenge from verifier using S256 method
func GenerateCodeChallenge(verifier string) string {
	h := sha256.New()
	h.Write([]byte(verifier))
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(h.Sum(nil))
}

// GetAuthorizationURL returns the OAuth authorization URL with PKCE
func (c *Client) GetAuthorizationURL(state, codeChallenge, redirectURL string) string {
	params := url.Values{}
	params.Set("client_id", c.config.ClientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", redirectURL)
	params.Set("scope", strings.Join(c.config.Scopes, " "))
	params.Set("state", state)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")

	return c.discovery.AuthorizationEndpoint + "?" + params.Encode()
}

// ExchangeCode exchanges authorization code for access token
func (c *Client) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURL string) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURL)
	data.Set("client_id", c.config.ClientID)
	data.Set("client_secret", c.config.ClientSecret)
	data.Set("code_verifier", codeVerifier)

	req, err := http.NewRequestWithContext(ctx, "POST", c.discovery.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		IDToken     string `json:"id_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if result.AccessToken == "" {
		return "", fmt.Errorf("token response missing access_token")
	}

	return result.AccessToken, nil
}

// GetUserInfo fetches user information from the userinfo endpoint
func (c *Client) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.discovery.UserinfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("userinfo request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode userinfo response: %w", err)
	}

	// Validate required fields
	if userInfo.Subject == "" {
		return nil, fmt.Errorf("userinfo response missing sub (subject) claim")
	}
	if userInfo.Email == "" {
		return nil, fmt.Errorf("userinfo response missing email")
	}

	return &userInfo, nil
}
