package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jaskrrish/Go-OKD/internal/handlers"
)

func main() {
	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create a new HTTP multiplexer
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/", handlers.HomeHandler)
	mux.HandleFunc("/health", handlers.HealthHandler)
	mux.HandleFunc("/api/v1/users", handlers.UsersHandler)

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
