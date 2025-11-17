package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jaskrrish/Go-OKD/internal/models"
)

// HomeHandler handles requests to the root path
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	response := map[string]string{
		"message": "Welcome to Go-OKD API",
		"version": "1.0.0",
		"status":  "running",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HealthHandler handles health check requests
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "go-okd-api",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(health)
}

// UsersHandler handles user-related requests
func UsersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getUsersHandler(w, r)
	case http.MethodPost:
		createUserHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getUsersHandler returns a list of users
func getUsersHandler(w http.ResponseWriter, r *http.Request) {
	// Mock data - in production, this would come from a database
	users := []models.User{
		{
			ID:        1,
			Username:  "john_doe",
			Email:     "john@example.com",
			CreatedAt: time.Now().Add(-24 * time.Hour),
		},
		{
			ID:        2,
			Username:  "jane_smith",
			Email:     "jane@example.com",
			CreatedAt: time.Now().Add(-48 * time.Hour),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// createUserHandler creates a new user
func createUserHandler(w http.ResponseWriter, r *http.Request) {
	var user models.User

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// In production, you would save this to a database
	user.ID = 3 // Mock ID
	user.CreatedAt = time.Now()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}
