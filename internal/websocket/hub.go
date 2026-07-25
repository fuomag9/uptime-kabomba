package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"nhooyr.io/websocket"

	"github.com/fuomag9/uptime-kabomba/internal/models"
)

// Message represents a WebSocket message
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Client represents a WebSocket client
type Client struct {
	ID     string
	UserID int
	Conn   *websocket.Conn
	Hub    *Hub
	Send   chan []byte
}

// broadcastMessage pairs an encoded message with the user it's scoped to.
type broadcastMessage struct {
	recipientUserID int
	data            []byte
}

// Hub maintains active clients and broadcasts messages
type Hub struct {
	clients        map[*Client]bool
	broadcast      chan broadcastMessage
	register       chan *Client
	unregister     chan *Client
	mu             sync.RWMutex
	jwtSecret      string
	allowedOrigins []string
	db             *gorm.DB
}

// NewHub creates a new Hub
func NewHub(jwtSecret string, allowedOrigins []string, db *gorm.DB) *Hub {
	return &Hub{
		clients:        make(map[*Client]bool),
		broadcast:      make(chan broadcastMessage, 256),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		jwtSecret:      jwtSecret,
		allowedOrigins: allowedOrigins,
		db:             db,
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client connected: %s", client.ID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				log.Printf("WebSocket client disconnected: %s", client.ID)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				// A monitor's events only ever go to the user who owns it -
				// every other connected client (any authenticated user, per
				// HandleWebSocket) must not see it.
				if client.UserID != message.recipientUserID {
					continue
				}
				select {
				case client.Send <- message.data:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastToUser sends a message to every connected client belonging to
// recipientUserID only. There is deliberately no "broadcast to everyone"
// path: every event this hub carries (heartbeats today) is owned by exactly
// one user, and previously being fanned out to all connected clients let any
// authenticated user passively observe every other tenant's monitor status,
// ping, and error messages in real time.
func (h *Hub) BroadcastToUser(recipientUserID int, msgType string, payload interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := Message{
		Type:    msgType,
		Payload: payloadJSON,
	}

	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	h.broadcast <- broadcastMessage{recipientUserID: recipientUserID, data: msgJSON}
	return nil
}

// HandleWebSocket handles WebSocket connections
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Authenticate WebSocket connection. Precedence: Authorization header
	// (API/non-browser clients) -> access_token cookie (the web app - sent
	// automatically by the browser on the WS handshake, unlike headers,
	// which JS can't set on a WebSocket connection) -> Sec-WebSocket-Protocol
	// subprotocol (legacy fallback from before cookie-based auth).
	var token string
	usedSubprotocol := false
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		var ok bool
		token, ok = strings.CutPrefix(authHeader, "Bearer ")
		if !ok {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}
	} else if cookie, err := r.Cookie("access_token"); err == nil && cookie.Value != "" {
		token = cookie.Value
	} else if subprotocolHeader := r.Header.Get("Sec-WebSocket-Protocol"); subprotocolHeader != "" {
		parts := strings.Split(subprotocolHeader, ",")
		if len(parts) > 0 {
			token = strings.TrimSpace(parts[0])
			usedSubprotocol = true
		}
	}

	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Validate JWT token with algorithm check
	userID := ""
	var jti string
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Validate the algorithm is HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// Only accept HS256
		if token.Method.Alg() != "HS256" {
			return nil, fmt.Errorf("unexpected signing algorithm: %v", token.Method.Alg())
		}
		return []byte(h.jwtSecret), nil
	})

	if err == nil && parsedToken.Valid {
		claims := parsedToken.Claims.(jwt.MapClaims)

		// Explicitly validate expiry
		if exp, ok := claims["exp"].(float64); ok {
			if time.Now().Unix() > int64(exp) {
				log.Printf("WebSocket JWT token expired from %s", r.RemoteAddr)
				http.Error(w, "Token expired", http.StatusUnauthorized)
				return
			}
		} else {
			http.Error(w, "Token has no expiry", http.StatusUnauthorized)
			return
		}

		if j, ok := claims["jti"].(string); ok {
			jti = j
		}

		if uid, ok := claims["user_id"].(float64); ok {
			userID = strconv.Itoa(int(uid))
		}
	}

	// Require authentication for WebSocket connections
	if userID == "" {
		log.Printf("WebSocket connection rejected: no valid authentication from %s", r.RemoteAddr)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Reject tokens revoked before their natural expiry (see HandleLogout).
	if jti != "" {
		var revokedCount int64
		h.db.Table("revoked_access_tokens").Where("jti = ?", jti).Count(&revokedCount)
		if revokedCount > 0 {
			http.Error(w, "Token revoked", http.StatusUnauthorized)
			return
		}
	}

	// Ensure user exists and is active
	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var user models.User
	if err := h.db.Where("id = ?", userIDInt).First(&user).Error; err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if !user.Active {
		http.Error(w, "Account disabled", http.StatusUnauthorized)
		return
	}

	// Use configured allowed origins (same as CORS)
	// nhooyr.io/websocket matches OriginPatterns against the host only,
	// so we need to extract hosts from full URLs.
	allowedOrigins := h.allowedOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"http://localhost:3000"}
	}
	originHosts := make([]string, 0, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if u, err := url.Parse(origin); err == nil && u.Host != "" {
			originHosts = append(originHosts, u.Host)
		} else {
			originHosts = append(originHosts, origin)
		}
	}

	acceptOptions := &websocket.AcceptOptions{
		OriginPatterns: originHosts,
	}
	if usedSubprotocol {
		acceptOptions.Subprotocols = []string{token}
	}

	conn, err := websocket.Accept(w, r, acceptOptions)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	clientID := "user:" + userID

	client := &Client{
		ID:     clientID,
		UserID: userIDInt,
		Conn:   conn,
		Hub:    h,
		Send:   make(chan []byte, 256),
	}

	h.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// readPump reads messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close(websocket.StatusNormalClosure, "")
	}()

	ctx := context.Background()
	for {
		_, message, err := c.Conn.Read(ctx)
		if err != nil {
			// Only log unexpected errors, not normal closures
			status := websocket.CloseStatus(err)
			if status == websocket.StatusNormalClosure ||
			   status == websocket.StatusGoingAway ||
			   status == websocket.StatusNoStatusRcvd {
				// Normal disconnect - don't log as error
				break
			}
			log.Printf("WebSocket unexpected error: %v", err)
			break
		}

		// Parse message
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Failed to parse WebSocket message: %v", err)
			continue
		}

		// Handle message based on type
		c.handleMessage(msg)
	}
}

// writePump writes messages to the WebSocket connection
func (c *Client) writePump() {
	ctx := context.Background()
	for message := range c.Send {
		err := c.Conn.Write(ctx, websocket.MessageText, message)
		if err != nil {
			// Only log unexpected write errors
			status := websocket.CloseStatus(err)
			if status != websocket.StatusNormalClosure &&
			   status != websocket.StatusGoingAway &&
			   status != websocket.StatusNoStatusRcvd {
				log.Printf("WebSocket unexpected write error: %v", err)
			}
			return
		}
	}
}

// handleMessage handles incoming WebSocket messages
func (c *Client) handleMessage(msg Message) {
	switch msg.Type {
	case "subscribe":
		// TODO: Implement monitor subscription
		log.Printf("Client %s subscribed", c.ID)
	case "unsubscribe":
		// TODO: Implement monitor unsubscription
		log.Printf("Client %s unsubscribed", c.ID)
	case "ping":
		// Respond to ping
		response, _ := json.Marshal(Message{
			Type:    "pong",
			Payload: json.RawMessage(`{}`),
		})
		c.Send <- response
	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}
