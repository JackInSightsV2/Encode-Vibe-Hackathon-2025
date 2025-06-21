package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"qt1-middleware/auth"
	"qt1-middleware/models"
	"qt1-middleware/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// APIKeyHandlers handles API key management endpoints
type APIKeyHandlers struct {
	apiKeyManager *auth.APIKeyManager
	logger        utils.Logger
}

// NewAPIKeyHandlers creates new API key handlers
func NewAPIKeyHandlers(apiKeyManager *auth.APIKeyManager, logger utils.Logger) *APIKeyHandlers {
	return &APIKeyHandlers{
		apiKeyManager: apiKeyManager,
		logger:        logger,
	}
}

// CreateAPIKey creates a new API key for the authenticated user
func (h *APIKeyHandlers) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user from context
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "Authentication required", "AUTH_REQUIRED")
		return
	}

	// Parse request body
	var req models.APIKeyCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", "INVALID_JSON")
		return
	}

	// Validate request
	if err := h.validateCreateRequest(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, err.Error(), "VALIDATION_ERROR")
		return
	}

	// Create the API key
	apiKey, plainKey, err := h.apiKeyManager.GenerateAPIKey(r.Context(), user.ID, &req)
	if err != nil {
		h.logger.Error("Failed to create API key", map[string]interface{}{
			"error":   err.Error(),
			"user_id": user.ID,
			"name":    req.Name,
		})
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create API key", "CREATION_FAILED")
		return
	}

	// Format the final API key
	finalKey := h.apiKeyManager.FormatAPIKey(apiKey.ID, plainKey)

	// Log successful creation
	h.logger.Info("API key created", map[string]interface{}{
		"api_key_id": apiKey.ID,
		"user_id":    user.ID,
		"name":       apiKey.Name,
	})

	// Return response with the plain key (only shown once)
	response := &models.APIKeyCreateResponse{
		APIKey:   apiKey.ToResponse(),
		PlainKey: finalKey,
		Warning:  "This is the only time the full API key will be shown. Please save it securely.",
	}

	h.writeSuccessResponse(w, response)
}

// ListAPIKeys lists API keys for the authenticated user
func (h *APIKeyHandlers) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user from context
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "Authentication required", "AUTH_REQUIRED")
		return
	}

	// Parse query parameters
	activeOnly := r.URL.Query().Get("active") == "true"
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Get API keys
	apiKeys, err := h.apiKeyManager.GetUserAPIKeys(r.Context(), user.ID, activeOnly)
	if err != nil {
		h.logger.Error("Failed to list API keys", map[string]interface{}{
			"error":   err.Error(),
			"user_id": user.ID,
		})
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve API keys", "RETRIEVAL_FAILED")
		return
	}

	// Convert to response format
	var keyResponses []*models.APIKeyResponse
	for _, key := range apiKeys {
		keyResponses = append(keyResponses, key.ToResponse())
	}

	// Apply pagination
	total := len(keyResponses)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	paginatedKeys := keyResponses[start:end]
	totalPages := (total + pageSize - 1) / pageSize

	response := &models.APIKeyListResponse{
		Keys:       paginatedKeys,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	h.writeSuccessResponse(w, response)
}

// GetAPIKey retrieves a specific API key
func (h *APIKeyHandlers) GetAPIKey(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user from context
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "Authentication required", "AUTH_REQUIRED")
		return
	}

	// Get key ID from URL
	vars := mux.Vars(r)
	keyID := vars["keyId"]
	if keyID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "API key ID required", "MISSING_KEY_ID")
		return
	}

	// Get the API key and verify ownership through the manager
	apiKeys, err := h.apiKeyManager.GetUserAPIKeys(r.Context(), user.ID, false)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve API key", "RETRIEVAL_FAILED")
		return
	}

	// Find the specific key
	var targetKey *models.APIKey
	for _, key := range apiKeys {
		if key.ID == keyID {
			targetKey = key
			break
		}
	}

	if targetKey == nil {
		h.writeErrorResponse(w, http.StatusNotFound, "API key not found", "KEY_NOT_FOUND")
		return
	}

	h.writeSuccessResponse(w, targetKey.ToResponse())
}

// UpdateAPIKey updates an existing API key
func (h *APIKeyHandlers) UpdateAPIKey(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user from context
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "Authentication required", "AUTH_REQUIRED")
		return
	}

	// Get key ID from URL
	vars := mux.Vars(r)
	keyID := vars["keyId"]
	if keyID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "API key ID required", "MISSING_KEY_ID")
		return
	}

	// Parse request body
	var req models.APIKeyUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", "INVALID_JSON")
		return
	}

	// Update the API key
	updatedKey, err := h.apiKeyManager.UpdateAPIKey(r.Context(), keyID, user.ID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not owned") {
			h.writeErrorResponse(w, http.StatusNotFound, "API key not found", "KEY_NOT_FOUND")
		} else {
			h.logger.Error("Failed to update API key", map[string]interface{}{
				"error":      err.Error(),
				"user_id":    user.ID,
				"api_key_id": keyID,
			})
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update API key", "UPDATE_FAILED")
		}
		return
	}

	h.logger.Info("API key updated", map[string]interface{}{
		"api_key_id": keyID,
		"user_id":    user.ID,
	})

	h.writeSuccessResponse(w, updatedKey.ToResponse())
}

// DeleteAPIKey deletes an API key
func (h *APIKeyHandlers) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user from context
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "Authentication required", "AUTH_REQUIRED")
		return
	}

	// Get key ID from URL
	vars := mux.Vars(r)
	keyID := vars["keyId"]
	if keyID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "API key ID required", "MISSING_KEY_ID")
		return
	}

	// Delete the API key
	err := h.apiKeyManager.DeleteAPIKey(r.Context(), keyID, user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not owned") {
			h.writeErrorResponse(w, http.StatusNotFound, "API key not found", "KEY_NOT_FOUND")
		} else {
			h.logger.Error("Failed to delete API key", map[string]interface{}{
				"error":      err.Error(),
				"user_id":    user.ID,
				"api_key_id": keyID,
			})
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete API key", "DELETE_FAILED")
		}
		return
	}

	h.logger.Info("API key deleted", map[string]interface{}{
		"api_key_id": keyID,
		"user_id":    user.ID,
	})

	h.writeSuccessResponse(w, map[string]interface{}{
		"message": "API key deleted successfully",
	})
}

// RotateAPIKey rotates an existing API key (creates new, deactivates old)
func (h *APIKeyHandlers) RotateAPIKey(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user from context
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "Authentication required", "AUTH_REQUIRED")
		return
	}

	// Get key ID from URL
	vars := mux.Vars(r)
	keyID := vars["keyId"]
	if keyID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "API key ID required", "MISSING_KEY_ID")
		return
	}

	// Rotate the API key
	newKey, plainKey, err := h.apiKeyManager.RotateAPIKey(r.Context(), keyID, user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not owned") {
			h.writeErrorResponse(w, http.StatusNotFound, "API key not found", "KEY_NOT_FOUND")
		} else {
			h.logger.Error("Failed to rotate API key", map[string]interface{}{
				"error":      err.Error(),
				"user_id":    user.ID,
				"api_key_id": keyID,
			})
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to rotate API key", "ROTATION_FAILED")
		}
		return
	}

	// Format the final API key
	finalKey := h.apiKeyManager.FormatAPIKey(newKey.ID, plainKey)

	h.logger.Info("API key rotated", map[string]interface{}{
		"old_key_id": keyID,
		"new_key_id": newKey.ID,
		"user_id":    user.ID,
	})

	// Return response with the new key
	response := &models.APIKeyCreateResponse{
		APIKey:   newKey.ToResponse(),
		PlainKey: finalKey,
		Warning:  "This is the only time the new API key will be shown. The old key has been deactivated.",
	}

	h.writeSuccessResponse(w, response)
}

// GetAPIKeyUsage retrieves usage statistics for an API key
func (h *APIKeyHandlers) GetAPIKeyUsage(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user from context
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "Authentication required", "AUTH_REQUIRED")
		return
	}

	// Get key ID from URL
	vars := mux.Vars(r)
	keyID := vars["keyId"]
	if keyID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "API key ID required", "MISSING_KEY_ID")
		return
	}

	// Parse query parameters
	days := 30 // Default to 30 days
	if daysParam := r.URL.Query().Get("days"); daysParam != "" {
		if parsedDays, err := strconv.Atoi(daysParam); err == nil && parsedDays > 0 && parsedDays <= 365 {
			days = parsedDays
		}
	}

	// Get usage statistics
	usage, err := h.apiKeyManager.GetAPIKeyUsage(r.Context(), keyID, user.ID, days)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not owned") {
			h.writeErrorResponse(w, http.StatusNotFound, "API key not found", "KEY_NOT_FOUND")
		} else {
			h.logger.Error("Failed to get API key usage", map[string]interface{}{
				"error":      err.Error(),
				"user_id":    user.ID,
				"api_key_id": keyID,
			})
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve usage statistics", "USAGE_RETRIEVAL_FAILED")
		}
		return
	}

	h.writeSuccessResponse(w, map[string]interface{}{
		"api_key_id": keyID,
		"days":       days,
		"usage":      usage,
	})
}

// GetAPIKeyStats retrieves overall API key statistics (admin only)
func (h *APIKeyHandlers) GetAPIKeyStats(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user from context
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "Authentication required", "AUTH_REQUIRED")
		return
	}

	// Check if user is admin
	if user.Role != "admin" && user.Role != "super_admin" {
		h.writeErrorResponse(w, http.StatusForbidden, "Admin access required", "ADMIN_REQUIRED")
		return
	}

	// Get statistics
	stats, err := h.apiKeyManager.GetAPIKeyStats(r.Context())
	if err != nil {
		h.logger.Error("Failed to get API key stats", map[string]interface{}{
			"error":   err.Error(),
			"user_id": user.ID,
		})
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve statistics", "STATS_RETRIEVAL_FAILED")
		return
	}

	h.writeSuccessResponse(w, stats)
}

// Helper methods

func (h *APIKeyHandlers) validateCreateRequest(req *models.APIKeyCreateRequest) error {
	if req.Name == "" {
		return fmt.Errorf("API key name is required")
	}
	if len(req.Name) > 100 {
		return fmt.Errorf("API key name must be 100 characters or less")
	}
	if len(req.Description) > 500 {
		return fmt.Errorf("API key description must be 500 characters or less")
	}
	if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now()) {
		return fmt.Errorf("expiry date must be in the future")
	}
	return nil
}

func (h *APIKeyHandlers) writeSuccessResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	response := map[string]interface{}{
		"success": true,
		"data":    data,
	}
	
	json.NewEncoder(w).Encode(response)
}

func (h *APIKeyHandlers) writeErrorResponse(w http.ResponseWriter, statusCode int, message, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
	
	json.NewEncoder(w).Encode(response)
}