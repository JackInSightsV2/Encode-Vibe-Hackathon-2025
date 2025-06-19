package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message types constants
const (
	MessageTypeHealthUpdate    = "health_update"
	MessageTypeLogEntry        = "log_entry"
	MessageTypeConfigChange    = "config_change"
	MessageTypeKillSwitch      = "kill_switch_update"
	MessageTypeProviderStatus  = "provider_status"
	MessageTypeClientMessage   = "client_message"
	MessageTypeError           = "error"
	MessageTypeAck             = "ack"
)

// WSMessage represents the standard WebSocket message format
type WSMessage struct {
	Type      string                 `json:"type"`
	Timestamp string                 `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	ID        string                 `json:"id,omitempty"`
}

// MessageHandler interface for handling different message types
type MessageHandler interface {
	HandleMessage(conn *websocket.Conn, msg WSMessage) error
	GetMessageType() string
}

// WebSocket upgrader with proper settings
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Check allowed origins for security
		origin := r.Header.Get("Origin")
		
		// Allow same origin
		if origin == "" {
			return true // Same origin requests don't have Origin header
		}
		
		// For development, allow localhost origins
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:8080", 
			"http://127.0.0.1:3000",
			"http://127.0.0.1:8080",
		}
		
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				return true
			}
		}
		
		// In production, you would check against your domain
		// return origin == "https://yourdomain.com"
		
		return false
	},
}

// WebSocketManager manages WebSocket connections
type WebSocketManager struct {
	connections    map[*websocket.Conn]*UserConnection
	handlers       map[string]MessageHandler
	mutex          sync.RWMutex
	pingTicker     *time.Ticker
	ctx            context.Context
	cancel         context.CancelFunc
}

// UserConnection represents an authenticated WebSocket connection
type UserConnection struct {
	User       *User
	ConnectedAt time.Time
	LastSeen    time.Time
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager() *WebSocketManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	wsm := &WebSocketManager{
		connections: make(map[*websocket.Conn]*UserConnection),
		handlers:    make(map[string]MessageHandler),
		pingTicker:  time.NewTicker(30 * time.Second), // Ping every 30 seconds
		ctx:         ctx,
		cancel:      cancel,
	}
	
	// Register default handlers
	wsm.RegisterHandler(&ClientMessageHandler{})
	
	// Start ping goroutine
	go wsm.startPingLoop()
	
	return wsm
}

// RegisterHandler registers a message handler for a specific message type
func (wsm *WebSocketManager) RegisterHandler(handler MessageHandler) {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()
	wsm.handlers[handler.GetMessageType()] = handler
}

// parseMessage parses and validates incoming WebSocket messages
func (wsm *WebSocketManager) parseMessage(data []byte) (*WSMessage, error) {
	var msg WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("invalid JSON format: %v", err)
	}
	
	// Validate required fields
	if msg.Type == "" {
		return nil, fmt.Errorf("message type is required")
	}
	
	// Add timestamp if not provided
	if msg.Timestamp == "" {
		msg.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	
	return &msg, nil
}

// routeMessage routes a message to the appropriate handler with permission checking
func (wsm *WebSocketManager) routeMessage(conn *websocket.Conn, msg *WSMessage) error {
	wsm.mutex.RLock()
	userConn, exists := wsm.connections[conn]
	if !exists {
		wsm.mutex.RUnlock()
		return fmt.Errorf("connection not found")
	}
	
	handler, handlerExists := wsm.handlers[msg.Type]
	wsm.mutex.RUnlock()
	
	if !handlerExists {
		return fmt.Errorf("no handler found for message type: %s", msg.Type)
	}
	
	// Check permissions based on message type
	if !wsm.hasPermissionForMessage(userConn.User, msg.Type) {
		return fmt.Errorf("insufficient permissions for message type: %s", msg.Type)
	}
	
	// Update last seen time
	wsm.mutex.Lock()
	userConn.LastSeen = time.Now()
	wsm.mutex.Unlock()
	
	return handler.HandleMessage(conn, *msg)
}

// hasPermissionForMessage checks if a user has permission for a specific message type
func (wsm *WebSocketManager) hasPermissionForMessage(user *User, messageType string) bool {
	switch messageType {
	case MessageTypeHealthUpdate:
		return user.CanReceiveHealthUpdates()
	case MessageTypeLogEntry:
		return user.CanReceiveLogUpdates()
	case MessageTypeClientMessage:
		return true // All authenticated users can send client messages
	case MessageTypeAck:
		return true // All authenticated users can receive acks
	case MessageTypeError:
		return true // All authenticated users can receive errors
	default:
		return user.IsAdmin // Unknown message types require admin
	}
}

// sendErrorMessage sends an error message back to the client
func (wsm *WebSocketManager) sendErrorMessage(conn *websocket.Conn, errorMsg string, originalID string) {
	errorResponse := WSMessage{
		Type:      MessageTypeError,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Error:     errorMsg,
		ID:        originalID,
	}
	
	data, err := json.Marshal(errorResponse)
	if err != nil {
		log.Printf("Failed to marshal error message: %v", err)
		return
	}
	
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("Failed to send error message: %v", err)
	}
}

// HandleWebSocketSecure handles authenticated WebSocket connections
func (wsm *WebSocketManager) HandleWebSocketSecure(w http.ResponseWriter, r *http.Request) {
	// Check rate limiting
	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.Header.Get("X-Real-IP")
	}
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}
	
	if !WSRateLimiter.CheckRateLimit(clientIP) {
		log.Printf("Rate limit exceeded for IP: %s", clientIP)
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	
	// Authenticate user
	user, err := AuthSvc.AuthenticateRequest(r)
	if err != nil {
		log.Printf("WebSocket authentication failed: %v", err)
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	
	// Check WebSocket permissions
	if !user.CanAccessWebSocket() {
		log.Printf("WebSocket access denied for user: %s", user.Username)
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}
	
	wsm.handleAuthenticatedConnection(w, r, user)
}

// HandleWebSocket handles WebSocket connection upgrade and management (legacy/dev mode)
func (wsm *WebSocketManager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// For development/testing - create a default admin user
	defaultUser := &User{
		ID:       "dev-user",
		Username: "developer",
		Email:    "dev@localhost",
		Roles:    []string{"admin", "websocket", "health", "logs"},
		IsAdmin:  true,
	}
	
	wsm.handleAuthenticatedConnection(w, r, defaultUser)
}

// handleAuthenticatedConnection handles the actual WebSocket connection logic
func (wsm *WebSocketManager) handleAuthenticatedConnection(w http.ResponseWriter, r *http.Request, user *User) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	
	// Add connection to manager with user info
	wsm.addConnection(conn, user)
	defer wsm.removeConnection(conn)
	
	// Record connection in health manager
	if HealthMgr != nil {
		HealthMgr.RecordConnection()
	}
	
	log.Printf("WebSocket connection established from %s for user: %s", r.RemoteAddr, user.Username)
	
	// Set read deadline and pong handler
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(appData string) error {
		log.Printf("Received pong from %s", conn.RemoteAddr())
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	
	// Handle incoming messages
	for {
		select {
		case <-wsm.ctx.Done():
			return
		default:
			// Read message from client
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return
			}
			
			// Handle different message types
			switch messageType {
			case websocket.TextMessage:
				// Try to parse as JSON message
				wsMsg, parseErr := wsm.parseMessage(message)
				if parseErr != nil {
					log.Printf("Message parsing error: %v", parseErr)
					wsm.sendErrorMessage(conn, parseErr.Error(), "")
					continue
				}
				
				log.Printf("Received message type '%s' from %s", wsMsg.Type, conn.RemoteAddr())
				
				// Route message to appropriate handler
				if err := wsm.routeMessage(conn, wsMsg); err != nil {
					log.Printf("Message routing error: %v", err)
					wsm.sendErrorMessage(conn, err.Error(), wsMsg.ID)
				}
				
			case websocket.BinaryMessage:
				log.Printf("Received binary message of length: %d", len(message))
				wsm.sendErrorMessage(conn, "binary messages not supported", "")
				
			case websocket.CloseMessage:
				log.Printf("Received close message from %s", conn.RemoteAddr())
				return
			}
		}
	}
}

// addConnection safely adds a connection to the manager
func (wsm *WebSocketManager) addConnection(conn *websocket.Conn, user *User) {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()
	wsm.connections[conn] = &UserConnection{
		User:        user,
		ConnectedAt: time.Now(),
		LastSeen:    time.Now(),
	}
	log.Printf("Added WebSocket connection for user %s. Total connections: %d", user.Username, len(wsm.connections))
}

// removeConnection safely removes a connection from the manager
func (wsm *WebSocketManager) removeConnection(conn *websocket.Conn) {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()
	
	if userConn, exists := wsm.connections[conn]; exists {
		username := "unknown"
		if userConn.User != nil {
			username = userConn.User.Username
		}
		delete(wsm.connections, conn)
		conn.Close()
		log.Printf("Removed WebSocket connection for user %s. Total connections: %d", username, len(wsm.connections))
	}
}

// GetConnectionCount returns the current number of active connections
func (wsm *WebSocketManager) GetConnectionCount() int {
	wsm.mutex.RLock()
	defer wsm.mutex.RUnlock()
	return len(wsm.connections)
}

// startPingLoop sends periodic ping messages to all connections
func (wsm *WebSocketManager) startPingLoop() {
	defer wsm.pingTicker.Stop()
	
	for {
		select {
		case <-wsm.ctx.Done():
			return
		case <-wsm.pingTicker.C:
			wsm.sendPingToAll()
		}
	}
}

// sendPingToAll sends ping messages to all active connections
func (wsm *WebSocketManager) sendPingToAll() {
	wsm.mutex.RLock()
	connections := make([]*websocket.Conn, 0, len(wsm.connections))
	for conn := range wsm.connections {
		connections = append(connections, conn)
	}
	wsm.mutex.RUnlock()
	
	for _, conn := range connections {
		err := conn.WriteMessage(websocket.PingMessage, []byte{})
		if err != nil {
			log.Printf("Error sending ping to %s: %v", conn.RemoteAddr(), err)
			wsm.removeConnection(conn)
		}
	}
	
	if len(connections) > 0 {
		log.Printf("Sent ping to %d connections", len(connections))
	}
}

// Close gracefully shuts down the WebSocket manager
func (wsm *WebSocketManager) Close() {
	wsm.cancel()
	
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()
	
	// Close all connections
	for conn := range wsm.connections {
		conn.WriteMessage(websocket.CloseMessage, []byte{})
		conn.Close()
	}
	
	wsm.connections = make(map[*websocket.Conn]*UserConnection)
	log.Printf("WebSocket manager closed")
}

// BroadcastMessage broadcasts a message to all connected clients with appropriate permissions
func (wsm *WebSocketManager) BroadcastMessage(message WSMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling broadcast message: %v", err)
		return
	}
	
	wsm.mutex.RLock()
	authorizedConnections := make([]*websocket.Conn, 0, len(wsm.connections))
	for conn, userConn := range wsm.connections {
		if wsm.hasPermissionForMessage(userConn.User, message.Type) {
			authorizedConnections = append(authorizedConnections, conn)
		}
	}
	wsm.mutex.RUnlock()
	
	successCount := 0
	for _, conn := range authorizedConnections {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("Error broadcasting to %s: %v", conn.RemoteAddr(), err)
			wsm.removeConnection(conn)
		} else {
			successCount++
		}
	}
	
	log.Printf("Broadcasted message type '%s' to %d/%d authorized connections", 
		message.Type, successCount, len(wsm.connections))
}

// ClientMessageHandler handles basic client messages
type ClientMessageHandler struct{}

func (h *ClientMessageHandler) GetMessageType() string {
	return MessageTypeClientMessage
}

func (h *ClientMessageHandler) HandleMessage(conn *websocket.Conn, msg WSMessage) error {
	log.Printf("Handling client message: %v", msg.Data)
	
	// Send acknowledgment back to client
	ackMessage := WSMessage{
		Type:      MessageTypeAck,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data: map[string]interface{}{
			"original_message": msg.Data,
			"status":          "received",
		},
		ID: msg.ID,
	}
	
	data, err := json.Marshal(ackMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal ack message: %v", err)
	}
	
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("failed to send ack message: %v", err)
	}
	
	return nil
}

// Global WebSocket manager instance
var WSManager *WebSocketManager

// Initialize the WebSocket manager and health manager
func init() {
	WSManager = NewWebSocketManager()
	HealthMgr = NewHealthManager(WSManager)
	
	// Register health handler
	WSManager.RegisterHandler(HealthMgr)
}