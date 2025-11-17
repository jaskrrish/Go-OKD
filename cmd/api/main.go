package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jaskrrish/Go-OKD/internal/handlers"
	"github.com/jaskrrish/Go-OKD/internal/qkd/quantum"
)

func main() {
	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create a new HTTP multiplexer
	mux := http.NewServeMux()

	// Initialize quantum backend (simulator for development)
	quantumBackend := quantum.NewSimulatorBackend(true, 0.05) // 5% noise
	qkdHandler := handlers.NewQKDHandler(quantumBackend)

	// Register existing routes
	mux.HandleFunc("/", handlers.HomeHandler)
	mux.HandleFunc("/health", handlers.HealthHandler)
	mux.HandleFunc("/api/v1/users", handlers.UsersHandler)

	// Register QKD routes
	mux.HandleFunc("/api/v1/qkd/health", qkdHandler.HealthCheckHandler)
	mux.HandleFunc("/api/v1/qkd/session/initiate", qkdHandler.InitiateSessionHandler)
	mux.HandleFunc("/api/v1/qkd/session/join", qkdHandler.JoinSessionHandler)
	mux.HandleFunc("/api/v1/qkd/session/", handleQKDSession(qkdHandler))
	mux.HandleFunc("/api/v1/qkd/key/", handleQKDKey(qkdHandler))

	// Create server with timeouts
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Server starting on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// loggingMiddleware logs all incoming requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("%s %s %s", r.Method, r.RequestURI, r.RemoteAddr)
		next.ServeHTTP(w, r)
		log.Printf("Request completed in %v", time.Since(start))
	})
}

// handleQKDSession routes QKD session-related requests
func handleQKDSession(qkdHandler *handlers.QKDHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/execute") {
			qkdHandler.ExecuteKeyExchangeHandler(w, r)
		} else {
			qkdHandler.GetSessionHandler(w, r)
		}
	}
}

// handleQKDKey routes QKD key-related requests
func handleQKDKey(qkdHandler *handlers.QKDHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			qkdHandler.RevokeKeyHandler(w, r)
		} else {
			qkdHandler.GetKeyHandler(w, r)
		}
	}
}
