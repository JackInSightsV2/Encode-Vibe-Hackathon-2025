package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type WSLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type WSLoginResponse struct {
	Token string `json:"token"`
	User  any    `json:"user"`
}

func TestWebSocket(t *testing.T) {
	// First, get a token
	loginReq := WSLoginRequest{
		Username: "admin",
		Password: "admin123",
	}

	reqBody, _ := json.Marshal(loginReq)
	resp, err := http.Post("http://localhost:8080/api/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Printf("Error logging in: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var loginResp WSLoginResponse
	json.NewDecoder(resp.Body).Decode(&loginResp)
	
	fmt.Printf("Got token: %s...\n", loginResp.Token[:30])

	// Test secure WebSocket connection with token
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws-secure"}
	q := u.Query()
	q.Set("token", loginResp.Token)
	u.RawQuery = q.Encode()

	fmt.Printf("Connecting to secure WebSocket: %s\n", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Printf("Error connecting to secure WebSocket: %v\n", err)
		return
	}
	defer c.Close()

	fmt.Println("Connected to secure WebSocket successfully!")

	// Send a test message
	testMessage := map[string]interface{}{
		"type": "client_message",
		"data": map[string]interface{}{
			"text": "Hello from authenticated client!",
		},
		"id": "test-123",
	}

	if err := c.WriteJSON(testMessage); err != nil {
		fmt.Printf("Error sending message: %v\n", err)
		return
	}

	fmt.Println("Sent test message")

	// Read response
	c.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, message, err := c.ReadMessage()
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	fmt.Printf("Received response: %s\n", string(message))

	// Test unauthorized access
	fmt.Println("\nTesting unauthorized access...")
	u2 := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws-secure"}
	
	_, resp2, err := websocket.DefaultDialer.Dial(u2.String(), nil)
	if err != nil {
		fmt.Printf("Expected error for unauthorized access: %v\n", err)
		if resp2 != nil {
			fmt.Printf("Response status: %s\n", resp2.Status)
		}
	} else {
		fmt.Println("ERROR: Unauthorized access should have failed!")
	}

	fmt.Println("Security test completed!")
}