package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"qt1-middleware/models"
	"strconv"
	"strings"
	"time"
)

// SessionHandlers provides HTTP handlers for session management
type SessionHandlers struct {
	sessionManager *SessionManager
	authService    AuthService
}

// NewSessionHandlers creates new session handlers
func NewSessionHandlers(sessionManager *SessionManager, authService AuthService) *SessionHandlers {
	return &SessionHandlers{
		sessionManager: sessionManager,
		authService:    authService,
	}
}

// GetUserSessions returns sessions for the authenticated user
func (h *SessionHandlers) GetUserSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get authenticated user
	user, ok := GetUserFromRequest(r)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "User not authenticated")
		return
	}

	// Get query parameters
	activeOnly := r.URL.Query().Get("active") == "true"

	// Get user sessions
	sessions, err := h.sessionManager.GetUserSessions(r.Context(), user.ID, activeOnly)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "SESSION_FETCH_FAILED", "Failed to fetch sessions")
		return
	}

	// Convert to response format
	sessionResponses := make([]*models.SessionResponse, len(sessions))
	for i, session := range sessions {
		sessionResponses[i] = session.ToResponse()
	}

	response := map[string]interface{}{
		"success":  true,
		"sessions": sessionResponses,
		"count":    len(sessionResponses),
		"filters": map[string]interface{}{
			"active_only": activeOnly,
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// InvalidateSession invalidates a specific session
func (h *SessionHandlers) InvalidateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get authenticated user
	user, ok := GetUserFromRequest(r)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "User not authenticated")
		return
	}

	// Parse request body
	var req struct {
		SessionID int `json:"session_id" validate:"required,gt=0"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Get session to verify ownership
	sessions, err := h.sessionManager.GetUserSessions(r.Context(), user.ID, false)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "SESSION_FETCH_FAILED", "Failed to fetch sessions")
		return
	}

	// Check if user owns the session
	sessionFound := false
	for _, session := range sessions {
		if session.ID == req.SessionID {
			sessionFound = true
			break
		}
	}

	if !sessionFound {
		h.writeErrorResponse(w, http.StatusForbidden, "SESSION_NOT_OWNED", "Session not owned by user")
		return
	}

	// Invalidate session
	if err := h.sessionManager.InvalidateSession(r.Context(), req.SessionID); err != nil {
		if sessionErr, ok := err.(*SessionError); ok {
			h.writeErrorResponse(w, http.StatusBadRequest, sessionErr.Code, sessionErr.Message)
		} else {
			h.writeErrorResponse(w, http.StatusInternalServerError, "INVALIDATION_FAILED", "Failed to invalidate session")
		}
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Session invalidated successfully",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// InvalidateAllSessions invalidates all sessions for the authenticated user
func (h *SessionHandlers) InvalidateAllSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get authenticated user
	user, ok := GetUserFromRequest(r)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "User not authenticated")
		return
	}

	// Parse request body (optional - can exclude current session)
	var req struct {
		ExcludeCurrent bool `json:"exclude_current"`
	}

	// Try to parse body, but don't fail if empty
	json.NewDecoder(r.Body).Decode(&req)

	// Get current session if we need to exclude it
	if req.ExcludeCurrent {
		// This would require passing the current session ID through context
		// For now, we'll invalidate all sessions
		// TODO: Implement current session exclusion
	}

	// Invalidate all user sessions
	if err := h.sessionManager.InvalidateAllUserSessions(r.Context(), user.ID); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "INVALIDATION_FAILED", "Failed to invalidate sessions")
		return
	}

	// TODO: If excluding current session, recreate current session

	response := map[string]interface{}{
		"success": true,
		"message": "All sessions invalidated successfully",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// GetSessionAnalytics returns session analytics (admin only)
func (h *SessionHandlers) GetSessionAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters for date range
	startTimeStr := r.URL.Query().Get("start_time")
	endTimeStr := r.URL.Query().Get("end_time")

	// Default to last 24 hours if not specified
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	if startTimeStr != "" {
		if parsed, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = parsed
		}
	}

	if endTimeStr != "" {
		if parsed, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = parsed
		}
	}

	// Get analytics
	analytics, err := h.sessionManager.GetSessionAnalytics(r.Context(), startTime, endTime)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "ANALYTICS_FAILED", "Failed to fetch analytics")
		return
	}

	response := map[string]interface{}{
		"success":    true,
		"analytics":  analytics,
		"date_range": map[string]interface{}{
			"start_time": startTime.Format(time.RFC3339),
			"end_time":   endTime.Format(time.RFC3339),
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// GetSuspiciousActivity returns suspicious activity for the authenticated user
func (h *SessionHandlers) GetSuspiciousActivity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get authenticated user
	user, ok := GetUserFromRequest(r)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "User not authenticated")
		return
	}

	// Get suspicious activities
	activities, err := h.sessionManager.DetectSuspiciousActivity(r.Context(), user.ID)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "DETECTION_FAILED", "Failed to detect suspicious activity")
		return
	}

	response := map[string]interface{}{
		"success":              true,
		"suspicious_activities": activities,
		"count":                len(activities),
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// GetUserDevices returns device information for the authenticated user
func (h *SessionHandlers) GetUserDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get authenticated user
	user, ok := GetUserFromRequest(r)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "User not authenticated")
		return
	}

	// Get user devices
	devices, err := h.sessionManager.GetDeviceInfo(user.ID)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "DEVICE_FETCH_FAILED", "Failed to fetch device information")
		return
	}

	response := map[string]interface{}{
		"success": true,
		"devices": devices,
		"count":   len(devices),
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// ManageDevice handles device management operations (trust, block, etc.)
func (h *SessionHandlers) ManageDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get authenticated user
	user, ok := GetUserFromRequest(r)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "User not authenticated")
		return
	}

	// Parse request body
	var req struct {
		Fingerprint string `json:"fingerprint" validate:"required"`
		Action      string `json:"action" validate:"required,oneof=trust untrust verify block unblock remove"`
		Reason      string `json:"reason,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Get device tracker (assuming it's available through session manager)
	deviceTracker := h.sessionManager.deviceTracker

	var err error
	var message string

	switch req.Action {
	case "trust":
		err = deviceTracker.TrustDevice(user.ID, req.Fingerprint)
		message = "Device trusted successfully"
	case "untrust":
		err = deviceTracker.UntrustDevice(user.ID, req.Fingerprint)
		message = "Device untrusted successfully"
	case "verify":
		err = deviceTracker.VerifyDevice(user.ID, req.Fingerprint)
		message = "Device verified successfully"
	case "block":
		if req.Reason == "" {
			req.Reason = "Blocked by user"
		}
		err = deviceTracker.BlockDevice(user.ID, req.Fingerprint, req.Reason)
		message = "Device blocked successfully"
	case "unblock":
		err = deviceTracker.UnblockDevice(user.ID, req.Fingerprint)
		message = "Device unblocked successfully"
	case "remove":
		err = deviceTracker.RemoveDevice(user.ID, req.Fingerprint)
		message = "Device removed successfully"
	default:
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_ACTION", "Invalid device action")
		return
	}

	if err != nil {
		if deviceErr, ok := err.(*DeviceError); ok {
			h.writeErrorResponse(w, http.StatusBadRequest, deviceErr.Code, deviceErr.Message)
		} else {
			h.writeErrorResponse(w, http.StatusInternalServerError, "DEVICE_ACTION_FAILED", "Failed to perform device action")
		}
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": message,
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// AdminForceLogout allows admins to force logout from a specific session
func (h *SessionHandlers) AdminForceLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req struct {
		SessionID int    `json:"session_id" validate:"required,gt=0"`
		Reason    string `json:"reason" validate:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Force logout
	if err := h.sessionManager.ForceLogout(r.Context(), req.SessionID, req.Reason); err != nil {
		if sessionErr, ok := err.(*SessionError); ok {
			h.writeErrorResponse(w, http.StatusBadRequest, sessionErr.Code, sessionErr.Message)
		} else {
			h.writeErrorResponse(w, http.StatusInternalServerError, "FORCE_LOGOUT_FAILED", "Failed to force logout")
		}
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Session forcefully logged out",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// GetRateLimitStatus returns the current rate limit status for the client
func (h *SessionHandlers) GetRateLimitStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get client IP
	clientIP := h.getClientIP(r)

	// Get rate limit status
	status := h.sessionManager.rateLimiter.GetStatus(clientIP)

	response := map[string]interface{}{
		"success":         true,
		"rate_limit_status": status,
		"client_ip":       clientIP,
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// GetSecurityStats returns security statistics (admin only)
func (h *SessionHandlers) GetSecurityStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get device stats
	deviceStats := h.sessionManager.deviceTracker.GetDeviceStats()

	// Get rate limiter stats
	rateLimitStats := h.sessionManager.rateLimiter.GetStats()

	// Get suspicious devices
	suspiciousDevices := h.sessionManager.deviceTracker.GetSuspiciousDevices(3) // Threshold of 3

	response := map[string]interface{}{
		"success": true,
		"stats": map[string]interface{}{
			"devices":           deviceStats,
			"rate_limiting":     rateLimitStats,
			"suspicious_devices": suspiciousDevices,
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// ExtendSession extends the current session
func (h *SessionHandlers) ExtendSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req struct {
		Duration string `json:"duration" validate:"required"` // e.g., "1h", "30m"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Parse duration
	duration, err := time.ParseDuration(req.Duration)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_DURATION", "Invalid duration format")
		return
	}

	// Limit extension to reasonable bounds (e.g., max 24 hours)
	maxExtension := 24 * time.Hour
	if duration > maxExtension {
		duration = maxExtension
	}

	// Get current session ID from context
	// This would need to be set during authentication
	sessionIDStr := r.Header.Get("X-Session-ID")
	if sessionIDStr == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "SESSION_ID_REQUIRED", "Session ID required for extension")
		return
	}

	sessionID, err := strconv.Atoi(sessionIDStr)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_SESSION_ID", "Invalid session ID")
		return
	}

	// Extend session
	if err := h.sessionManager.ExtendSession(r.Context(), sessionID, duration); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "EXTENSION_FAILED", "Failed to extend session")
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Session extended by %v", duration),
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// Helper methods

func (h *SessionHandlers) writeErrorResponse(w http.ResponseWriter, statusCode int, errorCode, message string) {
	response := map[string]interface{}{
		"success": false,
		"error": map[string]string{
			"code":    errorCode,
			"message": message,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

func (h *SessionHandlers) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (h *SessionHandlers) getClientIP(r *http.Request) string {
	// Check for forwarded IP addresses
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check for real IP
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fallback to remote address
	ip := r.RemoteAddr
	if colonIndex := strings.LastIndex(ip, ":"); colonIndex != -1 {
		ip = ip[:colonIndex]
	}
	return ip
}