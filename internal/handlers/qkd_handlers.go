package handlers

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jaskrrish/Go-OKD/internal/models/qkd"
	qkdcore "github.com/jaskrrish/Go-OKD/internal/qkd"
	"github.com/jaskrrish/Go-OKD/internal/qkd/quantum"
)

// QKDHandler manages QKD-related HTTP requests
type QKDHandler struct {
	sessionManager *qkdcore.SessionManager
}

// NewQKDHandler creates a new QKD handler with a quantum backend
func NewQKDHandler(backend quantum.QuantumBackend) *QKDHandler {
	return &QKDHandler{
		sessionManager: qkdcore.NewSessionManager(backend),
	}
}

// InitiateSessionHandler handles POST /api/v1/qkd/session/initiate
// Alice initiates a new QKD session
func (h *QKDHandler) InitiateSessionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req qkd.SessionCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	session, err := h.sessionManager.CreateSession(&req)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, qkd.SessionResponse{
		Session: session,
	})
}

// JoinSessionHandler handles POST /api/v1/qkd/session/join
// Bob joins an existing QKD session
func (h *QKDHandler) JoinSessionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req qkd.SessionJoinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid session ID")
		return
	}

	session, err := h.sessionManager.JoinSession(sessionID, req.BobID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, qkd.SessionResponse{
		Session: session,
	})
}

// ExecuteKeyExchangeHandler handles POST /api/v1/qkd/session/{id}/execute
// Executes the BB84 key exchange for an active session
func (h *QKDHandler) ExecuteKeyExchangeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract session ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 6 {
		respondWithError(w, http.StatusBadRequest, "Invalid URL format")
		return
	}

	sessionID, err := uuid.Parse(pathParts[5])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid session ID")
		return
	}

	// Execute key exchange with full post-processing
	key, err := h.sessionManager.ExecuteKeyExchangeWithPostProcessing(sessionID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Key exchange failed: %v", err))
		return
	}

	// Get updated session info
	session, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve session")
		return
	}

	response := map[string]interface{}{
		"session": session,
		"key_id":  key.KeyID.String(),
		"message": "Quantum key generated successfully!",
	}

	respondWithJSON(w, http.StatusOK, response)
}

// GetSessionHandler handles GET /api/v1/qkd/session/{id}
// Retrieves information about a specific session
func (h *QKDHandler) GetSessionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract session ID from URL
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 6 {
		respondWithError(w, http.StatusBadRequest, "Invalid URL format")
		return
	}

	sessionID, err := uuid.Parse(pathParts[5])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid session ID")
		return
	}

	session, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, qkd.SessionResponse{
		Session: session,
	})
}

// GetKeyHandler handles GET /api/v1/qkd/key/{id}
// Retrieves a generated quantum key (requires authentication)
func (h *QKDHandler) GetKeyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract key ID from URL
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 6 {
		respondWithError(w, http.StatusBadRequest, "Invalid URL format")
		return
	}

	keyID, err := uuid.Parse(pathParts[5])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid key ID")
		return
	}

	// Get user ID from header (in production, this would come from JWT token)
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "User authentication required")
		return
	}

	key, err := h.sessionManager.GetKey(keyID, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err == qkd.ErrKeyNotFound {
			statusCode = http.StatusNotFound
		} else if err == qkd.ErrUnauthorized {
			statusCode = http.StatusForbidden
		} else if err == qkd.ErrKeyExpired {
			statusCode = http.StatusGone
		}
		respondWithError(w, statusCode, err.Error())
		return
	}

	response := qkd.KeyResponse{
		KeyID:     key.KeyID.String(),
		SessionID: key.SessionID.String(),
		KeyHex:    hex.EncodeToString(key.KeyMaterial),
		KeyLength: key.KeyLength,
		ExpiresAt: key.ExpiresAt,
	}

	respondWithJSON(w, http.StatusOK, response)
}

// RevokeKeyHandler handles DELETE /api/v1/qkd/key/{id}
// Revokes a quantum key
func (h *QKDHandler) RevokeKeyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 6 {
		respondWithError(w, http.StatusBadRequest, "Invalid URL format")
		return
	}

	keyID, err := uuid.Parse(pathParts[5])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid key ID")
		return
	}

	if err := h.sessionManager.RevokeKey(keyID); err != nil {
		respondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Key revoked successfully",
	})
}

// HealthCheckHandler handles GET /api/v1/qkd/health
// Returns health status of the QKD service
func (h *QKDHandler) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := map[string]interface{}{
		"status":  "healthy",
		"service": "Quantum Key Distribution",
		"version": "1.0.0",
	}

	respondWithJSON(w, http.StatusOK, health)
}

// respondWithJSON sends a JSON response
func respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// respondWithError sends an error response
func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	respondWithJSON(w, statusCode, map[string]string{
		"error": message,
	})
}
