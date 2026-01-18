package websocket

import (
	"encoding/json"
	"flowday/internal/auth"
	"flowday/internal/logger"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	hub      *Hub
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins - adjust for production
		},
	}
)

type Hub struct {
	clients    map[primitive.ObjectID]*Client
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	userID   primitive.ObjectID
	send     chan Message
	mu       sync.Mutex
	closed   bool
	closedMu sync.Mutex
}

type Message struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

func InitHub() {
	hub = &Hub{
		clients:    make(map[primitive.ObjectID]*Client),
		broadcast:  make(chan Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
	go hub.run()
	logger.Log.Info("WebSocket hub initialized")
}

// safeCloseSend safely closes a client's send channel
func (c *Client) safeCloseSend() {
	c.closedMu.Lock()
	defer c.closedMu.Unlock()
	if !c.closed {
		close(c.send)
		c.closed = true
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// Remove old connection if exists
			if oldClient, exists := h.clients[client.userID]; exists {
				oldClient.safeCloseSend()
				delete(h.clients, client.userID)
			}
			h.clients[client.userID] = client
			h.mu.Unlock()
			logger.Log.WithFields(map[string]interface{}{
				"user_id": client.userID.Hex(),
			}).Info("Client registered")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.userID]; ok {
				delete(h.clients, client.userID)
				client.safeCloseSend()
			}
			h.mu.Unlock()
			logger.Log.WithFields(map[string]interface{}{
				"user_id": client.userID.Hex(),
			}).Info("Client unregistered")

		case message := <-h.broadcast:
			h.mu.RLock()
			clientsToRemove := make([]*Client, 0)
			for _, client := range h.clients {
				select {
				case client.send <- message:
				default:
					clientsToRemove = append(clientsToRemove, client)
				}
			}
			h.mu.RUnlock()
			
			// Remove clients with closed channels outside the read lock
			if len(clientsToRemove) > 0 {
				h.mu.Lock()
				for _, client := range clientsToRemove {
					if _, exists := h.clients[client.userID]; exists {
						client.safeCloseSend()
						delete(h.clients, client.userID)
					}
				}
				h.mu.Unlock()
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Log.WithError(err).Error("WebSocket error")
			}
			break
		}
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()

	for {
		select {
		case message, ok := <-c.send:
			c.mu.Lock()
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				c.mu.Unlock()
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				c.mu.Unlock()
				return
			}

			jsonData, err := json.Marshal(message)
			if err != nil {
				c.mu.Unlock()
				return
			}

			w.Write(jsonData)
			w.Write([]byte("\n"))

			if err := w.Close(); err != nil {
				c.mu.Unlock()
				return
			}
			c.mu.Unlock()
		}
	}
}

func HandleWebSocket(c *gin.Context) {
	// Get token from query parameter or Authorization header
	token := c.Query("token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
	}

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Validate token and get user ID
	userIDStr, err := auth.ValidateAccessToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to upgrade connection")
		return
	}

	client := &Client{
		hub:    hub,
		conn:   conn,
		userID: userID,
		send:   make(chan Message, 256),
	}

	hub.register <- client

	go client.writePump()
	go client.readPump()
}

func BroadcastMessage(userID primitive.ObjectID, payload map[string]interface{}) {
	if hub == nil {
		return
	}

	message := Message{
		Type:    "message",
		Payload: payload,
	}

	hub.mu.RLock()
	client, exists := hub.clients[userID]
	hub.mu.RUnlock()

	if exists {
		select {
		case client.send <- message:
		default:
			client.safeCloseSend()
			hub.mu.Lock()
			delete(hub.clients, userID)
			hub.mu.Unlock()
		}
	}
}

func BroadcastTaskCreate(userIDs []primitive.ObjectID, taskData map[string]interface{}) {
	if hub == nil {
		return
	}

	message := Message{
		Type:    "task_create",
		Payload: taskData,
	}

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	for _, userID := range userIDs {
		if client, exists := hub.clients[userID]; exists {
			select {
			case client.send <- message:
			default:
				client.safeCloseSend()
				hub.mu.Lock()
				delete(hub.clients, userID)
				hub.mu.Unlock()
			}
		}
	}
}

func BroadcastTaskUpdate(userIDs []primitive.ObjectID, taskData map[string]interface{}) {
	if hub == nil {
		return
	}

	message := Message{
		Type:    "task_update",
		Payload: taskData,
	}

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	for _, userID := range userIDs {
		if client, exists := hub.clients[userID]; exists {
			select {
			case client.send <- message:
			default:
				client.safeCloseSend()
				hub.mu.Lock()
				delete(hub.clients, userID)
				hub.mu.Unlock()
			}
		}
	}
}

func BroadcastTaskDelete(userIDs []primitive.ObjectID, taskID string) {
	if hub == nil {
		return
	}

	message := Message{
		Type: "task_delete",
		Payload: map[string]interface{}{
			"task_id": taskID,
		},
	}

	hub.mu.RLock()
	defer hub.mu.RUnlock()

	for _, userID := range userIDs {
		if client, exists := hub.clients[userID]; exists {
			select {
			case client.send <- message:
			default:
				client.safeCloseSend()
				hub.mu.Lock()
				delete(hub.clients, userID)
				hub.mu.Unlock()
			}
		}
	}
}
