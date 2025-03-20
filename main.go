package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/raj-vidya-kender/echohttp/ui"
)

type requestData struct {
	ID        int64       `json:"id"`
	Timestamp time.Time   `json:"timestamp"`
	Data      string      `json:"data"`
	Headers   http.Header `json:"headers"`
}

// echoServer holds the application state
type echoServer struct {
	db *sql.DB
	mu sync.RWMutex
}

func (s *echoServer) initDB() error {
	// Create the requests table if it doesn't exist
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME NOT NULL,
			data TEXT NOT NULL,
			headers TEXT NOT NULL
		)
	`)
	return err
}

func (s *echoServer) handleRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGet(w, r)
	case http.MethodPost:
		s.handlePost(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *echoServer) handleGet(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, timestamp, data, headers
		FROM requests
		ORDER BY timestamp DESC
	`)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Printf("Database error: %v", err)
		return
	}
	defer rows.Close()

	var requests []requestData
	for rows.Next() {
		var req requestData
		var headersJSON string
		err := rows.Scan(&req.ID, &req.Timestamp, &req.Data, &headersJSON)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		// Parse headers from JSON
		if err := json.Unmarshal([]byte(headersJSON), &req.Headers); err != nil {
			log.Printf("Error parsing headers: %v", err)
			continue
		}

		requests = append(requests, req)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(requests)
}

func (s *echoServer) handlePost(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "body couldn't be read", http.StatusBadRequest)
		return
	}

	// Convert headers to JSON for storage
	headersJSON, err := json.Marshal(r.Header)
	if err != nil {
		http.Error(w, "Error processing headers", http.StatusInternalServerError)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Insert the request into the database
	_, err = s.db.Exec(`
		INSERT INTO requests (timestamp, data, headers)
		VALUES (?, ?, ?)
	`, time.Now(), string(body), string(headersJSON))

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Printf("Database error: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func main() {
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8025"
	}

	// Initialize SQLite database
	db, err := sql.Open("sqlite3", "echo.db")
	if err != nil {
		log.Fatalf("error opening database: %v", err)
	}
	defer db.Close()

	server := &echoServer{db: db}

	// Initialize the database schema
	if err := server.initDB(); err != nil {
		log.Fatalf("error initializing database: %v", err)
	}

	// Create a new HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: nil, // Will be set below
	}

	// Create a channel to listen for errors coming from the server
	serverErrors := make(chan error, 1)

	// Create a channel to listen for OS signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Set up routes
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.FS(ui.Assets()))))
	http.HandleFunc("/echo", server.handleRequests)

	// Start the server in a goroutine
	go func() {
		log.Printf("Server starting on %s", srv.Addr)
		serverErrors <- srv.ListenAndServe()
	}()

	// Blocking select waiting for either a signal or server error
	select {
	case err := <-serverErrors:
		log.Printf("Server error: %v", err)

	case sig := <-shutdown:
		log.Printf("Received signal %v, initiating graceful shutdown...", sig)

		// Create a context with timeout for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Could not stop server gracefully: %v", err)
			panic(err)
		}

		log.Println("Server stopped gracefully")
	}
}
