package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

// Integration tests for Cycle 5B: Authentication Middleware Integration

func setupTestServer(t *testing.T) (*httptest.Server, *AuthManager) {
	// Create in-memory database
	db := setupTestDB(t)
	
	// Create auth manager
	authManager, err := NewAuthManager(db)
	if err != nil {
		t.Fatalf("Failed to create auth manager: %v", err)
	}
	
	// Create test server
	mux := http.NewServeMux()
	
	// Register auth routes
	authManager.RegisterRoutes(mux)
	
	// Add test protected endpoint
	mux.HandleFunc("/api/test/protected", authManager.ProtectEndpoint(func(w http.ResponseWriter, r *http.Request) {
		user, ok := authManager.GetUserFromRequest(r)
		if !ok {
			http.Error(w, "No user in context", http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Access granted",
			"user":    user.ToResponse(),
		})
	}))
	
	// Add test admin endpoint
	mux.HandleFunc("/api/test/admin", authManager.ProtectAdminEndpoint(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Admin access granted",
		})
	}))
	
	// Add test moderator endpoint
	mux.HandleFunc("/api/test/moderator", authManager.ProtectRoleEndpoint("moderator", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Moderator access granted",
		})
	}))
	
	// Add test optional auth endpoint
	mux.HandleFunc("/api/test/optional", authManager.OptionalAuth(func(w http.ResponseWriter, r *http.Request) {
		user, ok := authManager.GetUserFromRequest(r)
		
		response := map[string]interface{}{
			"message": "Public endpoint",
		}
		
		if ok {
			response["user"] = user.ToResponse()
		} else {
			response["user"] = nil
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	
	server := httptest.NewServer(mux)
	return server, authManager
}

func TestAuthenticationFlow(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()
	
	// Test user registration
	registerData := map[string]interface{}{
		"username": "testuser",
		"email":    "test@example.com",
		"password": "TestPassword123!",
		"role":     "user",
	}
	
	registerBody, _ := json.Marshal(registerData)
	resp, err := http.Post(server.URL+"/api/auth/register", "application/json", bytes.NewBuffer(registerBody))
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}
	
	var registerResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&registerResponse); err != nil {
		t.Fatalf("Failed to decode register response: %v", err)
	}
	
	// Test user login
	loginData := map[string]interface{}{
		"username": "testuser",
		"password": "TestPassword123!",
	}
	
	loginBody, _ := json.Marshal(loginData)
	resp, err = http.Post(server.URL+"/api/auth/login", "application/json", bytes.NewBuffer(loginBody))
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	var loginResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&loginResponse); err != nil {
		t.Fatalf("Failed to decode login response: %v", err)
	}
	
	token, ok := loginResponse["token"].(string)
	if !ok || token == "" {
		t.Fatal("No token in login response")
	}
	
	// Test token validation
	req, _ := http.NewRequest("GET", server.URL+"/api/auth/validate", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestProtectedEndpoints(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()
	
	// Create test users with different roles
	users := []struct {
		username string
		email    string
		password string
		role     string
	}{
		{"testadmin", "testadmin@test.com", "AdminPass123!", "admin"},
		{"testmoderator", "testmod@test.com", "ModPass123!", "moderator"},
		{"testuser", "testuser@test.com", "UserPass123!", "user"},
	}
	
	tokens := make(map[string]string)
	
	// Register and login users
	for _, userData := range users {
		// Register
		registerData := map[string]interface{}{
			"username": userData.username,
			"email":    userData.email,
			"password": userData.password,
			"role":     userData.role,
		}
		
		registerBody, _ := json.Marshal(registerData)
		resp, err := http.Post(server.URL+"/api/auth/register", "application/json", bytes.NewBuffer(registerBody))
		if err != nil {
			t.Fatalf("Failed to register %s: %v", userData.username, err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Failed to register %s with status %d", userData.username, resp.StatusCode)
		}
		resp.Body.Close()
		
		// Login
		loginData := map[string]interface{}{
			"username": userData.username,
			"password": userData.password,
		}
		
		loginBody, _ := json.Marshal(loginData)
		resp, err = http.Post(server.URL+"/api/auth/login", "application/json", bytes.NewBuffer(loginBody))
		if err != nil {
			t.Fatalf("Failed to login %s: %v", userData.username, err)
		}
		
		var loginResponse map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&loginResponse); err != nil {
			t.Fatalf("Failed to decode login response for %s: %v", userData.username, err)
		}
		resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Login failed for %s with status %d", userData.username, resp.StatusCode)
		}
		
		token, ok := loginResponse["token"].(string)
		if !ok || token == "" {
			t.Fatalf("No token in login response for %s", userData.username)
		}
		tokens[userData.role] = token
	}
	
	// Test protected endpoint access
	tests := []struct {
		name       string
		endpoint   string
		role       string
		expectCode int
	}{
		{"User accesses protected endpoint", "/api/test/protected", "user", 200},
		{"Moderator accesses protected endpoint", "/api/test/protected", "moderator", 200},
		{"Admin accesses protected endpoint", "/api/test/protected", "admin", 200},
		
		{"User cannot access admin endpoint", "/api/test/admin", "user", 403},
		{"Moderator cannot access admin endpoint", "/api/test/admin", "moderator", 403},
		{"Admin can access admin endpoint", "/api/test/admin", "admin", 200},
		
		{"User cannot access moderator endpoint", "/api/test/moderator", "user", 403},
		{"Moderator can access moderator endpoint", "/api/test/moderator", "moderator", 200},
		{"Admin can access moderator endpoint", "/api/test/moderator", "admin", 200},
	}
	
	client := &http.Client{}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", server.URL+test.endpoint, nil)
			req.Header.Set("Authorization", "Bearer "+tokens[test.role])
			
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != test.expectCode {
				t.Errorf("Expected status %d, got %d", test.expectCode, resp.StatusCode)
			}
		})
	}
}

func TestUnauthorizedAccess(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()
	
	client := &http.Client{}
	
	// Test accessing protected endpoint without token
	req, _ := http.NewRequest("GET", server.URL+"/api/test/protected", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
	
	// Test accessing protected endpoint with invalid token
	req, _ = http.NewRequest("GET", server.URL+"/api/test/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestOptionalAuthentication(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()
	
	client := &http.Client{}
	
	// Test optional auth endpoint without token
	req, _ := http.NewRequest("GET", server.URL+"/api/test/optional", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response["user"] != nil {
		t.Error("Expected null user for unauthenticated request")
	}
	
	// Register and login a user for authenticated test
	registerData := map[string]interface{}{
		"username": "optionaluser",
		"email":    "optional@test.com",
		"password": "OptionalPass123!",
		"role":     "user",
	}
	
	registerBody, _ := json.Marshal(registerData)
	http.Post(server.URL+"/api/auth/register", "application/json", bytes.NewBuffer(registerBody))
	
	loginData := map[string]interface{}{
		"username": "optionaluser",
		"password": "OptionalPass123!",
	}
	
	loginBody, _ := json.Marshal(loginData)
	resp, _ = http.Post(server.URL+"/api/auth/login", "application/json", bytes.NewBuffer(loginBody))
	
	var loginResponse map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&loginResponse)
	resp.Body.Close()
	
	token := loginResponse["token"].(string)
	
	// Test optional auth endpoint with token
	req, _ = http.NewRequest("GET", server.URL+"/api/test/optional", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response["user"] == nil {
		t.Error("Expected user data for authenticated request")
	}
}

func TestPasswordRequirements(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()
	
	weakPasswords := []string{
		"weak",           // too short
		"password",       // common password
		"12345678",       // no letters
		"abcdefgh",       // no numbers, no uppercase, no special chars
		"ABCDEFGH",       // no numbers, no lowercase, no special chars
		"Abcdefgh",       // no numbers, no special chars
		"Abcdefg1",       // no special chars
		"Password123",    // no special chars
	}
	
	for i, password := range weakPasswords {
		t.Run(fmt.Sprintf("WeakPassword_%d", i), func(t *testing.T) {
			registerData := map[string]interface{}{
				"username": fmt.Sprintf("weakuser%d", i),
				"email":    fmt.Sprintf("weak%d@test.com", i),
				"password": password,
				"role":     "user",
			}
			
			registerBody, _ := json.Marshal(registerData)
			resp, err := http.Post(server.URL+"/api/auth/register", "application/json", bytes.NewBuffer(registerBody))
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()
			
			if resp.StatusCode == http.StatusCreated {
				t.Errorf("Expected weak password '%s' to be rejected", password)
			}
		})
	}
	
	// Test strong password
	registerData := map[string]interface{}{
		"username": "stronguser",
		"email":    "strong@test.com",
		"password": "StrongPassword123!",
		"role":     "user",
	}
	
	registerBody, _ := json.Marshal(registerData)
	resp, err := http.Post(server.URL+"/api/auth/register", "application/json", bytes.NewBuffer(registerBody))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		t.Error("Expected strong password to be accepted")
	}
}

func TestTokenRefresh(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()
	
	// Register and login user
	registerData := map[string]interface{}{
		"username": "refreshuser",
		"email":    "refresh@test.com",
		"password": "RefreshPass123!",
		"role":     "user",
	}
	
	registerBody, _ := json.Marshal(registerData)
	http.Post(server.URL+"/api/auth/register", "application/json", bytes.NewBuffer(registerBody))
	
	loginData := map[string]interface{}{
		"username": "refreshuser",
		"password": "RefreshPass123!",
	}
	
	loginBody, _ := json.Marshal(loginData)
	resp, _ := http.Post(server.URL+"/api/auth/login", "application/json", bytes.NewBuffer(loginBody))
	
	var loginResponse map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&loginResponse)
	resp.Body.Close()
	
	refreshToken := loginResponse["refresh_token"].(string)
	
	// Test token refresh
	refreshData := map[string]interface{}{
		"refresh_token": refreshToken,
	}
	
	refreshBody, _ := json.Marshal(refreshData)
	resp, err := http.Post(server.URL+"/api/auth/refresh", "application/json", bytes.NewBuffer(refreshBody))
	if err != nil {
		t.Fatalf("Failed to refresh token: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	var refreshResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&refreshResponse); err != nil {
		t.Fatalf("Failed to decode refresh response: %v", err)
	}
	
	newToken, ok := refreshResponse["token"].(string)
	if !ok || newToken == "" {
		t.Error("Expected new token in refresh response")
	}
	
	// Test that new token works
	client := &http.Client{}
	req, _ := http.NewRequest("GET", server.URL+"/api/test/protected", nil)
	req.Header.Set("Authorization", "Bearer "+newToken)
	
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("New token should work, got status %d", resp.StatusCode)
	}
}

func TestWebSocketAuthentication(t *testing.T) {
	server, authManager := setupTestServer(t)
	defer server.Close()
	
	// Register and login user
	registerData := map[string]interface{}{
		"username": "wsuser",
		"email":    "ws@test.com",
		"password": "WSPass123!",
		"role":     "user",
	}
	
	registerBody, _ := json.Marshal(registerData)
	http.Post(server.URL+"/api/auth/register", "application/json", bytes.NewBuffer(registerBody))
	
	loginData := map[string]interface{}{
		"username": "wsuser",
		"password": "WSPass123!",
	}
	
	loginBody, _ := json.Marshal(loginData)
	resp, _ := http.Post(server.URL+"/api/auth/login", "application/json", bytes.NewBuffer(loginBody))
	
	var loginResponse map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&loginResponse)
	resp.Body.Close()
	
	token := loginResponse["token"].(string)
	
	// Test WebSocket authentication
	req, _ := http.NewRequest("GET", "ws://localhost/ws-secure?token="+token, nil)
	
	user, err := authManager.AuthenticateWebSocket(req)
	if err != nil {
		t.Fatalf("WebSocket authentication failed: %v", err)
	}
	
	if user.Username != "wsuser" {
		t.Errorf("Expected username 'wsuser', got '%s'", user.Username)
	}
}

func TestCORSHeaders(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()
	
	client := &http.Client{}
	
	// Test CORS preflight request
	req, _ := http.NewRequest("OPTIONS", server.URL+"/api/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")
	
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("CORS preflight request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204 for OPTIONS, got %d", resp.StatusCode)
	}
	
	// Check CORS headers
	expectedHeaders := map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
		"Access-Control-Allow-Headers": "Authorization, Content-Type, X-Requested-With",
	}
	
	for header, expectedValue := range expectedHeaders {
		if resp.Header.Get(header) != expectedValue {
			t.Errorf("Expected %s: %s, got %s", header, expectedValue, resp.Header.Get(header))
		}
	}
}