package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

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
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
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
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // For development
	})
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		ID:   r.RemoteAddr, // Use a better ID in production (e.g., user ID)
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
			log.Printf("WebSocket read error: %v", err)
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
			log.Printf("WebSocket write error: %v", err)
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
