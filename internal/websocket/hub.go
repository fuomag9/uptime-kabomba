package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"nhooyr.io/websocket"
)

// Message represents a WebSocket message
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Client represents a WebSocket client
type Client struct {
	ID   string
	Conn *websocket.Conn
	Hub  *Hub
	Send chan []byte
}

// Hub maintains active clients and broadcasts messages
type Hub struct {
	clients       map[*Client]bool
	broadcast     chan []byte
	register      chan *Client
	unregister    chan *Client
	mu            sync.RWMutex
	jwtSecret     string
	allowedOrigins []string
}

// NewHub creates a new Hub
func NewHub(jwtSecret string, allowedOrigins []string) *Hub {
	return &Hub{
		clients:       make(map[*Client]bool),
		broadcast:     make(chan []byte, 256),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		jwtSecret:     jwtSecret,
		allowedOrigins: allowedOrigins,
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
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(msgType string, payload interface{}) error {
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

	h.broadcast <- msgJSON
	return nil
}

// HandleWebSocket handles WebSocket connections
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Authenticate WebSocket connection
	token := r.URL.Query().Get("token")
	if token == "" {
		// Try getting from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	// Validate JWT token with algorithm check
	userID := ""
	if token != "" {
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
			}

			if uid, ok := claims["user_id"].(float64); ok {
				userID = string(rune(int(uid)))
			}
		}
	}

	// Require authentication for WebSocket connections
	if userID == "" {
		log.Printf("WebSocket connection rejected: no valid authentication from %s", r.RemoteAddr)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Use configured allowed origins (same as CORS)
	allowedOrigins := h.allowedOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"http://localhost:3000"}
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: allowedOrigins,
	})
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	clientID := r.RemoteAddr
	if userID != "" {
		clientID = "user:" + userID
	}

	client := &Client{
		ID:   clientID,
		Conn: conn,
		Hub:  h,
		Send: make(chan []byte, 256),
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
