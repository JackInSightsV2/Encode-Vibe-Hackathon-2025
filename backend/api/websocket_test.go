package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebSocketUpgrade(t *testing.T) {
	// Create a new WebSocket manager for testing
	wsManager := NewWebSocketManager()
	defer wsManager.Close()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(wsManager.HandleWebSocket))
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := strings.Replace(server.URL, "http://", "ws://", 1)

	// Create WebSocket connection
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Test that connection is established successfully with JSON message
	clientMsg := WSMessage{
		Type: MessageTypeClientMessage,
		Data: map[string]interface{}{
			"text": "ping test",
		},
	}

	msgData, err := json.Marshal(clientMsg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, msgData)
	if err != nil {
		t.Fatalf("Failed to send test message: %v", err)
	}

	// Set read deadline for test
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	// Read ack response to verify connection works
	messageType, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if messageType != websocket.TextMessage {
		t.Errorf("Expected text message, got %v", messageType)
	}

	// Parse response as WSMessage
	var response WSMessage
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Type != MessageTypeAck {
		t.Errorf("Expected ack message, got %s", response.Type)
	}
}

func TestWebSocketEcho(t *testing.T) {
	// Create a new WebSocket manager for testing
	wsManager := NewWebSocketManager()
	defer wsManager.Close()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(wsManager.HandleWebSocket))
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := strings.Replace(server.URL, "http://", "ws://", 1)

	// Create WebSocket connection
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Send test message as JSON
	clientMsg := WSMessage{
		Type: MessageTypeClientMessage,
		ID:   "test-456",
		Data: map[string]interface{}{
			"text": "Hello WebSocket",
		},
	}

	msgData, err := json.Marshal(clientMsg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, msgData)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Set read deadline for test
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Read ack response
	messageType, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read ack: %v", err)
	}

	if messageType != websocket.TextMessage {
		t.Errorf("Expected text message, got %v", messageType)
	}

	// Parse response
	var ackMsg WSMessage
	if err := json.Unmarshal(message, &ackMsg); err != nil {
		t.Fatalf("Failed to parse ack message: %v", err)
	}

	if ackMsg.Type != MessageTypeAck {
		t.Errorf("Expected ack message, got %s", ackMsg.Type)
	}

	if ackMsg.ID != clientMsg.ID {
		t.Errorf("Expected ID %s, got %s", clientMsg.ID, ackMsg.ID)
	}
}

func TestWebSocketConnectionCount(t *testing.T) {
	// Create a new WebSocket manager for testing
	wsManager := NewWebSocketManager()
	defer wsManager.Close()

	// Initial connection count should be 0
	if count := wsManager.GetConnectionCount(); count != 0 {
		t.Errorf("Expected 0 connections, got %d", count)
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(wsManager.HandleWebSocket))
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := strings.Replace(server.URL, "http://", "ws://", 1)

	// Create first WebSocket connection
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn1.Close()

	// Give some time for connection to be registered
	time.Sleep(100 * time.Millisecond)

	// Check connection count
	if count := wsManager.GetConnectionCount(); count != 1 {
		t.Errorf("Expected 1 connection, got %d", count)
	}

	// Create second WebSocket connection
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to second WebSocket: %v", err)
	}
	defer conn2.Close()

	// Give some time for connection to be registered
	time.Sleep(100 * time.Millisecond)

	// Check connection count
	if count := wsManager.GetConnectionCount(); count != 2 {
		t.Errorf("Expected 2 connections, got %d", count)
	}
}

func TestMessageParsing(t *testing.T) {
	wsManager := NewWebSocketManager()
	defer wsManager.Close()

	tests := []struct {
		name        string
		input       string
		expectError bool
		expectType  string
	}{
		{
			name:        "Valid client message",
			input:       `{"type":"client_message","data":{"text":"hello"}}`,
			expectError: false,
			expectType:  "client_message",
		},
		{
			name:        "Valid message with ID",
			input:       `{"type":"client_message","id":"123","data":{"text":"hello"}}`,
			expectError: false,
			expectType:  "client_message",
		},
		{
			name:        "Invalid JSON",
			input:       `{"type":"client_message","data":{"text":"hello"}`,
			expectError: true,
		},
		{
			name:        "Missing type",
			input:       `{"data":{"text":"hello"}}`,
			expectError: true,
		},
		{
			name:        "Empty message",
			input:       `{}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := wsManager.parseMessage([]byte(tt.input))
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if msg.Type != tt.expectType {
				t.Errorf("Expected type %s, got %s", tt.expectType, msg.Type)
			}
			
			// Check timestamp was added
			if msg.Timestamp == "" {
				t.Errorf("Expected timestamp to be set")
			}
		})
	}
}

func TestMessageRouting(t *testing.T) {
	wsManager := NewWebSocketManager()
	defer wsManager.Close()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(wsManager.HandleWebSocket))
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := strings.Replace(server.URL, "http://", "ws://", 1)

	// Create WebSocket connection
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Test valid client message
	clientMsg := WSMessage{
		Type: MessageTypeClientMessage,
		ID:   "test-123",
		Data: map[string]interface{}{
			"text": "Hello from test",
		},
	}

	msgData, err := json.Marshal(clientMsg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	// Send message
	err = conn.WriteMessage(websocket.TextMessage, msgData)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Read acknowledgment
	_, response, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	// Parse response
	var ackMsg WSMessage
	if err := json.Unmarshal(response, &ackMsg); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify acknowledgment
	if ackMsg.Type != MessageTypeAck {
		t.Errorf("Expected ack message, got %s", ackMsg.Type)
	}

	if ackMsg.ID != clientMsg.ID {
		t.Errorf("Expected ID %s, got %s", clientMsg.ID, ackMsg.ID)
	}
}

func TestInvalidMessageHandling(t *testing.T) {
	wsManager := NewWebSocketManager()
	defer wsManager.Close()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(wsManager.HandleWebSocket))
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := strings.Replace(server.URL, "http://", "ws://", 1)

	// Create WebSocket connection
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	tests := []struct {
		name          string
		message       string
		expectError   bool
		expectedError string
	}{
		{
			name:          "Invalid JSON",
			message:       `{"invalid": json}`,
			expectError:   true,
			expectedError: "invalid JSON format",
		},
		{
			name:          "Unknown message type",
			message:       `{"type":"unknown_type","data":{"test":"value"}}`,
			expectError:   true,
			expectedError: "no handler found for message type: unknown_type",
		},
		{
			name:          "Missing type",
			message:       `{"data":{"test":"value"}}`,
			expectError:   true,
			expectedError: "message type is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Send invalid message
			err = conn.WriteMessage(websocket.TextMessage, []byte(tt.message))
			if err != nil {
				t.Fatalf("Failed to send message: %v", err)
			}

			if tt.expectError {
				// Set read deadline
				conn.SetReadDeadline(time.Now().Add(2 * time.Second))

				// Read error response
				_, response, err := conn.ReadMessage()
				if err != nil {
					t.Fatalf("Failed to read error response: %v", err)
				}

				// Parse error response
				var errorMsg WSMessage
				if err := json.Unmarshal(response, &errorMsg); err != nil {
					t.Fatalf("Failed to parse error response: %v", err)
				}

				// Verify error message
				if errorMsg.Type != MessageTypeError {
					t.Errorf("Expected error message, got %s", errorMsg.Type)
				}

				if !strings.Contains(errorMsg.Error, tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, errorMsg.Error)
				}
			}
		})
	}
}