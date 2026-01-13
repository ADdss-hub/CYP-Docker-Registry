// Package handler provides HTTP handlers for the container registry.
package handler

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// WSHandler handles WebSocket connections.
type WSHandler struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan *WSMessage
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.RWMutex
	logger     *zap.Logger
}

// WSMessage represents a WebSocket message.
type WSMessage struct {
	Type      string                 `json:"type"`
	Event     string                 `json:"event"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewWSHandler creates a new WSHandler instance.
func NewWSHandler(logger *zap.Logger) *WSHandler {
	h := &WSHandler{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan *WSMessage, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		logger:     logger,
	}

	go h.run()
	return h
}

// RegisterRoutes registers WebSocket routes.
func (h *WSHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/ws", h.HandleWebSocket)
}

// HandleWebSocket handles WebSocket upgrade requests.
func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("WebSocket upgrade failed", zap.Error(err))
		}
		return
	}

	h.register <- conn

	// Handle incoming messages
	go h.readPump(conn)
}

func (h *WSHandler) run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()

			// Send welcome message
			h.sendToClient(conn, &WSMessage{
				Type:      "system",
				Event:     "connected",
				Timestamp: time.Now(),
			})

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			for conn := range h.clients {
				h.sendToClient(conn, msg)
			}
			h.mu.RUnlock()
		}
	}
}

func (h *WSHandler) readPump(conn *websocket.Conn) {
	defer func() {
		h.unregister <- conn
	}()

	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				if h.logger != nil {
					h.logger.Error("WebSocket read error", zap.Error(err))
				}
			}
			break
		}

		// Handle incoming message
		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		// Handle ping
		if msg.Type == "ping" {
			h.sendToClient(conn, &WSMessage{
				Type:      "pong",
				Timestamp: time.Now(),
			})
		}
	}
}

func (h *WSHandler) sendToClient(conn *websocket.Conn, msg *WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		h.unregister <- conn
	}
}

// Broadcast sends a message to all connected clients.
func (h *WSHandler) Broadcast(msgType, event string, data map[string]interface{}) {
	h.broadcast <- &WSMessage{
		Type:      msgType,
		Event:     event,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// BroadcastNotification sends a notification to all clients.
func (h *WSHandler) BroadcastNotification(level, title, message string) {
	h.Broadcast("notification", level, map[string]interface{}{
		"title":   title,
		"message": message,
	})
}

// BroadcastSystemEvent sends a system event to all clients.
func (h *WSHandler) BroadcastSystemEvent(event string, data map[string]interface{}) {
	h.Broadcast("system", event, data)
}

// GetClientCount returns the number of connected clients.
func (h *WSHandler) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
