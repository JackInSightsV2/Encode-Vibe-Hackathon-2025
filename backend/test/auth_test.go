package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  any    `json:"user"`
}

func TestAuth(t *testing.T) {
	// Test authentication
	loginReq := LoginRequest{
		Username: "admin",
		Password: "admin123",
	}

	reqBody, _ := json.Marshal(loginReq)
	
	// Login request
	resp, err := http.Post("http://localhost:8080/api/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("Login failed with status: %d\n", resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Response: %s\n", string(body))
		return
	}

	var loginResp LoginResponse
	json.NewDecoder(resp.Body).Decode(&loginResp)
	
	fmt.Printf("Login successful! Token: %s\n", loginResp.Token[:50]+"...")
	
	// Test token validation
	req, _ := http.NewRequest("GET", "http://localhost:8080/api/validate-token", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp2, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error validating token: %v\n", err)
		return
	}
	defer resp2.Body.Close()

	if resp2.StatusCode == 200 {
		fmt.Println("Token validation successful!")
	} else {
		fmt.Printf("Token validation failed with status: %d\n", resp2.StatusCode)
	}
}